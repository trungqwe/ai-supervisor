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
	if request.RestoreOperationID != "" {
		var state string
		if err = tx.QueryRowContext(ctx, `SELECT resolution_state FROM pair_restore_operations WHERE operation_id=? AND pair_id=?`, request.RestoreOperationID, request.PairID).Scan(&state); err != nil {
			return err
		}
		if state != string(domain.RestoreResolved) {
			return fmt.Errorf("%w: linked Tx D must precede D6", ErrStateConflict)
		}
	} else if err = rejectUnresolvedRestore(ctx, tx, request.PairID); err != nil {
		return err
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
	type lineage struct{ session, generation string }
	expected := map[string]lineage{}
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
		expected[id] = lineage{s.String, g.String}
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	if len(expected) != len(request.Attempts) {
		return fmt.Errorf("%w: D6 requires every quarantined attempt lineage", ErrAttemptLineageMismatch)
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
		accepted, err := acceptedLineageTx(ctx, tx, item.StopOperationID, request.PairID, request.Principal, item.SessionID, item.TerminalGeneration, item.AttemptID, "CLASS_B", request.RestoreOperationID)
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
		accepted, err := acceptedLineageTx(ctx, tx, item.StopOperationID, request.PairID, request.Principal, item.SessionID, item.TerminalGeneration, item.AttemptID, "CLASS_C", request.RestoreOperationID)
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
