package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/trungqwe/ai-supervisor/internal/domain"
)

// ClearanceBasis names the approved D6 proof class for exactly one lineage.
type ClearanceBasis string

const (
	ClearancePhysical              ClearanceBasis = "CLASS_A_PHYSICAL"
	ClearanceAbsenceAdministrative ClearanceBasis = "CLASS_B_ABSENCE_ADMINISTRATIVE"
	ClearanceRiskAdministrative    ClearanceBasis = "CLASS_C_RISK_ADMINISTRATIVE"
)

type LineageClearance struct {
	AttemptID          string // empty means current WorkerSession
	SessionID          string
	TerminalGeneration string
	Basis              ClearanceBasis
	StopOperationID    string
	Evidence           string
}

type QuarantineClearance struct {
	PairID             string
	RestoreOperationID string // empty for unlinked D6
	Principal          string // supplied only after trusted-boundary authentication
	At                 time.Time
	Session            LineageClearance
	Attempts           []LineageClearance
}

type clearanceLineage struct{ session, generation string }

// ClearQuarantineWithEvidence is D6. It never resolves a restore operation;
// linked restore Tx D must already be committed before this transaction.
func (s *Store) ClearQuarantineWithEvidence(ctx context.Context, request QuarantineClearance) error {
	if request.PairID == "" || request.Principal == "" || request.Session.AttemptID != "" || request.Session.Evidence == "" {
		return fmt.Errorf("%w: exact Pair, principal and session evidence required", ErrStateConflict)
	}
	if request.At.IsZero() {
		request.At = timeNow()
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// A historical resolved restore ID is not an admission token. Every current
	// unresolved restore on the Pair must be absent in this same D6 transaction.
	if err = rejectUnresolvedRestore(ctx, tx, request.PairID); err != nil {
		return err
	}
	if request.RestoreOperationID != "" {
		var state string
		if err = tx.QueryRowContext(ctx, `SELECT resolution_state FROM pair_restore_operations WHERE operation_id=? AND pair_id=?`, request.RestoreOperationID, request.PairID).Scan(&state); err != nil {
			return err
		}
		if state != string(domain.RestoreResolved) {
			return fmt.Errorf("%w: linked Tx D must precede D6", ErrStateConflict)
		}
	}
	var session, generation, quarantine string
	if err = tx.QueryRowContext(ctx, `SELECT session_id,terminal_generation,quarantine_state FROM worker_sessions WHERE pair_id=?`, request.PairID).Scan(&session, &generation, &quarantine); err != nil {
		return err
	}
	if quarantine != string(domain.QuarantineQuarantined) || request.Session.SessionID != session || request.Session.TerminalGeneration != generation {
		return ErrAttemptLineageMismatch
	}
	rows, err := tx.QueryContext(ctx, `SELECT a.attempt_id,a.session_id,a.terminal_generation FROM task_attempts a JOIN tasks t ON t.task_id=a.task_id WHERE t.pair_id=? AND a.quarantine_state='QUARANTINED'`, request.PairID)
	if err != nil {
		return err
	}
	expected := map[string]clearanceLineage{}
	for rows.Next() {
		var id string
		var s, g sql.NullString
		if err = rows.Scan(&id, &s, &g); err != nil {
			rows.Close()
			return err
		}
		if !s.Valid || !g.Valid {
			rows.Close()
			return ErrAttemptLineageMismatch
		}
		expected[id] = clearanceLineage{s.String, g.String}
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	if len(expected) != len(request.Attempts) {
		return fmt.Errorf("%w: D6 requires every quarantined attempt lineage", ErrAttemptLineageMismatch)
	}
	if err = guardPairStopsForClearance(ctx, tx, request, session, generation, expected); err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, item := range request.Attempts {
		if item.AttemptID == "" || seen[item.AttemptID] || item.Evidence == "" {
			return ErrAttemptLineageMismatch
		}
		seen[item.AttemptID] = true
		want, ok := expected[item.AttemptID]
		if !ok || want.session != item.SessionID || want.generation != item.TerminalGeneration {
			return ErrAttemptLineageMismatch
		}
		if err = verifyClearanceBasis(ctx, tx, request, item); err != nil {
			return err
		}
	}
	if err = verifyClearanceBasis(ctx, tx, request, request.Session); err != nil {
		return err
	}
	for _, item := range request.Attempts {
		res, writeErr := tx.ExecContext(ctx, `UPDATE task_attempts SET quarantine_state='CLEAN' WHERE attempt_id=? AND session_id=? AND terminal_generation=? AND quarantine_state='QUARANTINED' AND ended_at IS NOT NULL`, item.AttemptID, item.SessionID, item.TerminalGeneration)
		if writeErr = operationCASResult(res, writeErr, item.AttemptID, "attempt clearance"); writeErr != nil {
			return writeErr
		}
		if err = appendClearanceAudit(ctx, tx, request, item); err != nil {
			return err
		}
	}
	res, err := tx.ExecContext(ctx, `UPDATE worker_sessions SET quarantine_state='CLEAN',updated_at=? WHERE pair_id=? AND session_id=? AND terminal_generation=? AND quarantine_state='QUARANTINED'`, formatTime(request.At), request.PairID, session, generation)
	if err = operationCASResult(res, err, request.PairID, "session clearance"); err != nil {
		return err
	}
	if err = appendClearanceAudit(ctx, tx, request, request.Session); err != nil {
		return err
	}
	return tx.Commit()
}

// guardPairStopsForClearance inspects every durable stop on the Pair, not only
// the evidence selected by the caller. An unresolved newer stop keeps D6
// locked even if an older operation has an accepted audit or physical proof.
func guardPairStopsForClearance(ctx context.Context, tx *sql.Tx, request QuarantineClearance, currentSession, currentGeneration string, attempts map[string]clearanceLineage) error {
	rows, err := tx.QueryContext(ctx, `SELECT operation_id,session_id,terminal_generation,attempt_id,resolution_state,restore_operation_id FROM stop_operations WHERE pair_id=? ORDER BY rowid`, request.PairID)
	if err != nil {
		return err
	}
	type stopRisk struct {
		id, session, generation, resolution string
		attempt, restore                    sql.NullString
	}
	var risks []stopRisk
	for rows.Next() {
		var r stopRisk
		if err = rows.Scan(&r.id, &r.session, &r.generation, &r.attempt, &r.resolution, &r.restore); err != nil {
			rows.Close()
			return err
		}
		risks = append(risks, r)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	var latestSession string
	latestAttempt := map[string]string{}
	for _, r := range risks {
		if r.session == currentSession && r.generation == currentGeneration {
			latestSession = r.id
		}
		if r.attempt.Valid {
			if a, exists := attempts[r.attempt.String]; exists && a.session == r.session && a.generation == r.generation {
				latestAttempt[r.attempt.String] = r.id
			}
		}
		if r.resolution == string(domain.StopResolutionTerminationConfirmed) {
			continue
		}
		class := ""
		switch r.resolution {
		case string(domain.StopResolutionTargetAbsent):
			class = "CLASS_B"
		case string(domain.StopResolutionAdministrativeRiskAccepted):
			class = "CLASS_C"
		default:
			return fmt.Errorf("%w: Pair stop %s remains unresolved", ErrQuarantinedExecution, r.id)
		}
		if r.session == currentSession && r.generation == currentGeneration {
			cleared, e := priorLineageClearanceTx(ctx, tx, r.id, request.PairID, currentSession, currentGeneration, "")
			if e != nil {
				return e
			}
			if !cleared {
				ok, e := acceptedLineageTx(ctx, tx, r.id, request.PairID, request.Principal, currentSession, currentGeneration, "", class, r.restore.String)
				if e != nil {
					return e
				}
				if !ok {
					return fmt.Errorf("%w: WorkerSession risk from stop %s is not reconciled", ErrQuarantinedExecution, r.id)
				}
			}
		}
		if r.attempt.Valid {
			if a, exists := attempts[r.attempt.String]; exists && a.session == r.session && a.generation == r.generation {
				cleared, e := priorLineageClearanceTx(ctx, tx, r.id, request.PairID, r.session, r.generation, r.attempt.String)
				if e != nil {
					return e
				}
				if cleared {
					continue
				}
				ok, e := acceptedLineageTx(ctx, tx, r.id, request.PairID, request.Principal, r.session, r.generation, r.attempt.String, class, r.restore.String)
				if e != nil {
					return e
				}
				if !ok {
					return fmt.Errorf("%w: attempt risk from stop %s is not reconciled", ErrQuarantinedExecution, r.id)
				}
			}
		}
	}
	if latestSession != "" && request.Session.StopOperationID != latestSession {
		return fmt.Errorf("%w: D6 session evidence predates current stop", ErrStateConflict)
	}
	for _, item := range request.Attempts {
		if latest := latestAttempt[item.AttemptID]; latest != "" && item.StopOperationID != latest {
			return fmt.Errorf("%w: D6 attempt evidence predates current stop", ErrAttemptLineageMismatch)
		}
	}
	return nil
}

func priorLineageClearanceTx(ctx context.Context, tx *sql.Tx, stopID, pairID, session, generation, attemptID string) (bool, error) {
	var count int
	err := tx.QueryRowContext(ctx, `SELECT count(*) FROM audit_events WHERE event_type IN ('QUARANTINE_RESOLVED_PHYSICAL','QUARANTINE_RESOLVED_ADMINISTRATIVE') AND pair_id=? AND COALESCE(attempt_id,'')=? AND json_extract(details_json,'$.stop_operation_id')=? AND json_extract(details_json,'$.session_id')=? AND json_extract(details_json,'$.terminal_generation')=?`, pairID, attemptID, stopID, session, generation).Scan(&count)
	return count > 0, err
}

func verifyClearanceBasis(ctx context.Context, tx *sql.Tx, request QuarantineClearance, item LineageClearance) error {
	if item.SessionID == "" || item.TerminalGeneration == "" || item.Evidence == "" {
		return ErrAttemptLineageMismatch
	}
	switch item.Basis {
	case ClearancePhysical:
		if item.StopOperationID == "" {
			return fmt.Errorf("%w: Class A requires exact stop operation", ErrStateConflict)
		}
		var pair, session, generation, attempt string
		var attemptNull sql.NullString
		err := tx.QueryRowContext(ctx, `SELECT pair_id,session_id,terminal_generation,attempt_id FROM stop_operations WHERE operation_id=?`, item.StopOperationID).Scan(&pair, &session, &generation, &attemptNull)
		if err != nil {
			return err
		}
		if attemptNull.Valid {
			attempt = attemptNull.String
		}
		if pair != request.PairID || session != item.SessionID || generation != item.TerminalGeneration || (item.AttemptID != "" && attempt != item.AttemptID) {
			return ErrAttemptLineageMismatch
		}
		valid, err := positiveD11StopEvidence(ctx, tx, request.PairID, item.SessionID, item.TerminalGeneration, "", item.AttemptID, item.StopOperationID)
		if err != nil {
			return err
		}
		if !valid {
			return fmt.Errorf("%w: Class A lacks positive D11 evidence", ErrStateConflict)
		}
	case ClearanceAbsenceAdministrative:
		if item.StopOperationID == "" {
			return fmt.Errorf("%w: Class B requires 404 stop evidence", ErrStateConflict)
		}
		var count int
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM stop_operations WHERE operation_id=? AND pair_id=? AND session_id=? AND terminal_generation=? AND (?='' OR attempt_id=?) AND stage='STOP_TARGET_ABSENT' AND resolution_state='STOP_TARGET_ABSENT')`, item.StopOperationID, request.PairID, item.SessionID, item.TerminalGeneration, item.AttemptID, item.AttemptID).Scan(&count); err != nil {
			return err
		}
		if count != 1 {
			return fmt.Errorf("%w: Class B lacks exact 404 evidence", ErrStateConflict)
		}
		restoreLink, err := clearanceStopRestoreLink(ctx, tx, item.StopOperationID, request.PairID)
		if err != nil {
			return err
		}
		accepted, err := acceptedLineageTx(ctx, tx, item.StopOperationID, request.PairID, request.Principal, item.SessionID, item.TerminalGeneration, item.AttemptID, "CLASS_B", restoreLink)
		if err != nil {
			return err
		}
		if !accepted {
			return fmt.Errorf("%w: Class B risk not accepted for lineage", ErrStateConflict)
		}
	case ClearanceRiskAdministrative:
		if item.StopOperationID == "" {
			return fmt.Errorf("%w: Class C requires reconciled stop", ErrStateConflict)
		}
		var resolution string
		if err := tx.QueryRowContext(ctx, `SELECT resolution_state FROM stop_operations WHERE operation_id=? AND pair_id=?`, item.StopOperationID, request.PairID).Scan(&resolution); err != nil {
			return err
		}
		if resolution != string(domain.StopResolutionAdministrativeRiskAccepted) {
			return ErrStateConflict
		}
		restoreLink, err := clearanceStopRestoreLink(ctx, tx, item.StopOperationID, request.PairID)
		if err != nil {
			return err
		}
		accepted, err := acceptedLineageTx(ctx, tx, item.StopOperationID, request.PairID, request.Principal, item.SessionID, item.TerminalGeneration, item.AttemptID, "CLASS_C", restoreLink)
		if err != nil {
			return err
		}
		if !accepted {
			return fmt.Errorf("%w: Class C risk not accepted for lineage", ErrStateConflict)
		}
	default:
		return fmt.Errorf("%w: unsupported D6 basis", ErrInvalidOperationTransition)
	}
	return nil
}

func clearanceStopRestoreLink(ctx context.Context, tx *sql.Tx, stopID, pairID string) (string, error) {
	var link sql.NullString
	if err := tx.QueryRowContext(ctx, `SELECT restore_operation_id FROM stop_operations WHERE operation_id=? AND pair_id=?`, stopID, pairID).Scan(&link); err != nil {
		return "", err
	}
	return link.String, nil
}

func appendClearanceAudit(ctx context.Context, tx *sql.Tx, request QuarantineClearance, item LineageClearance) error {
	id, err := newAuditEventID()
	if err != nil {
		return err
	}
	kind := domain.AuditQuarantineResolvedAdministrative
	if item.Basis == ClearancePhysical {
		kind = domain.AuditQuarantineResolvedPhysical
	}
	details := map[string]any{"session_id": item.SessionID, "terminal_generation": item.TerminalGeneration, "basis": string(item.Basis), "evidence": item.Evidence, "principal": request.Principal, "restore_operation_id": request.RestoreOperationID, "stop_operation_id": item.StopOperationID}
	if item.Basis == ClearanceRiskAdministrative {
		details["resolution_state"] = string(domain.StopResolutionAdministrativeRiskAccepted)
	}
	var task, contract string
	if item.AttemptID != "" {
		if err = tx.QueryRowContext(ctx, `SELECT task_id,contract_id FROM task_attempts WHERE attempt_id=?`, item.AttemptID).Scan(&task, &contract); err != nil {
			return err
		}
	}
	_, err = appendAuditEventTx(ctx, tx, domain.AuditEvent{EventID: id, EventType: kind, Timestamp: request.At, PairID: request.PairID, TaskID: task, ContractID: contract, AttemptID: item.AttemptID, Actor: request.Principal, Details: details})
	return err
}
