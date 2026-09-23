package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/trungqwe/ai-supervisor/internal/domain"
)

// ReservePairRestore consumes a one-shot, operation-bound operator decision and
// commits the durable restore intent before any upstream effect is attempted.
func (s *Store) ReservePairRestore(ctx context.Context, request domain.RestoreReservation) error {
	if request.AuthorizationID == "" || request.OperationID == "" || request.PairID == "" ||
		request.SessionID == "" || request.ExpectedGeneration == "" || request.AuthorizedPrincipal == "" || request.Actor == "" {
		return fmt.Errorf("store: restore authorization tuple is incomplete")
	}
	if request.RiskScope != "POSSIBLE_PROMPT_REPLAY" && request.RiskScope != "PROMPT_REPLAY_AND_UNRESOLVED_EXECUTION" {
		return fmt.Errorf("store: invalid restore risk scope %q", request.RiskScope)
	}
	now := request.At.UTC()
	if now.IsZero() {
		now = timeNow()
	}
	riskEventID, err := newAuditEventID()
	if err != nil {
		return err
	}
	requestEventID, err := newAuditEventID()
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin restore reservation: %w", err)
	}
	defer tx.Rollback()
	var sessionID, generation, quarantine, status string
	if err := tx.QueryRowContext(ctx, `SELECT session_id,terminal_generation,quarantine_state,status FROM worker_sessions WHERE pair_id=?`, request.PairID).Scan(&sessionID, &generation, &quarantine, &status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrWorkerSessionNotFound
		}
		return fmt.Errorf("store: read restore pair session: %w", err)
	}
	if sessionID != request.SessionID || generation != request.ExpectedGeneration {
		return fmt.Errorf("%w: restore identity/generation changed", ErrStateConflict)
	}
	if quarantine != string(domain.QuarantineClean) {
		return fmt.Errorf("%w: quarantined WorkerSession requires lineage-specific clearance before another restore", ErrQuarantinedExecution)
	}
	if status != string(domain.WorkerSessionTerminated) {
		return fmt.Errorf("%w: restore admission requires WorkerSession.status=TERMINATED", ErrStateConflict)
	}
	var blocked int
	err = tx.QueryRowContext(ctx, `SELECT
 EXISTS(SELECT 1 FROM task_attempts a JOIN tasks t ON t.task_id=a.task_id WHERE t.pair_id=? AND a.ended_at IS NULL)
 OR EXISTS(SELECT 1 FROM task_attempts a JOIN tasks t ON t.task_id=a.task_id WHERE t.pair_id=? AND a.quarantine_state<>'CLEAN')
 OR EXISTS(SELECT 1 FROM dispatch_operations d WHERE d.pair_id=? AND d.stage='SEND_REQUESTED' AND d.resolution_state IS NULL)
 OR EXISTS(SELECT 1 FROM tasks t WHERE t.pair_id=? AND ((t.current_attempt>0 AND NOT EXISTS(SELECT 1 FROM task_attempts a WHERE a.task_id=t.task_id AND a.attempt_number=t.current_attempt)) OR (t.state IN ('DISPATCHED','RUNNING','BLOCKED') AND NOT EXISTS(SELECT 1 FROM task_attempts a WHERE a.task_id=t.task_id AND a.attempt_number=t.current_attempt AND a.ended_at IS NULL))))
 OR EXISTS(SELECT 1 FROM pair_provisioning_operations p WHERE p.pair_id=? AND p.stage IN ('PROVISION_REQUESTED','PROVISION_FAILED'))
 OR EXISTS(SELECT 1 FROM pair_restore_operations r WHERE r.pair_id=? AND r.resolution_state<>'RESTORE_RESOLVED')
 OR EXISTS(SELECT 1 FROM stop_operations o WHERE o.pair_id=? AND o.resolution_state IN ('IN_FLIGHT','STOP_CALL_OUTCOME_UNKNOWN','STOP_CONFIRMATION_TIMEOUT','STOP_REISSUE_REQUIRES_HUMAN'))`,
		request.PairID, request.PairID, request.PairID, request.PairID, request.PairID, request.PairID, request.PairID).Scan(&blocked)
	if err != nil {
		return fmt.Errorf("store: check restore admission: %w", err)
	}
	if blocked != 0 {
		return fmt.Errorf("%w: restore admission predicate rejected Pair", ErrQuarantinedExecution)
	}
	res, err := tx.ExecContext(ctx, `UPDATE worker_sessions SET quarantine_state='QUARANTINED',updated_at=? WHERE pair_id=? AND session_id=? AND terminal_generation=? AND status='TERMINATED' AND quarantine_state=?`, formatTime(now), request.PairID, request.SessionID, request.ExpectedGeneration, quarantine)
	if err != nil {
		return fmt.Errorf("store: quarantine restore session: %w", err)
	}
	if n, err := res.RowsAffected(); err != nil || n != 1 {
		if err != nil {
			return err
		}
		return fmt.Errorf("%w: restore reservation WorkerSession CAS lost", ErrStateConflict)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE task_attempts SET quarantine_state='QUARANTINED' WHERE session_id=? AND terminal_generation=? AND task_id IN (SELECT task_id FROM tasks WHERE pair_id=?)`, request.SessionID, request.ExpectedGeneration, request.PairID); err != nil {
		return fmt.Errorf("store: quarantine restore-linked attempt lineages: %w", err)
	}
	_, err = appendAuditEventTx(ctx, tx, domain.AuditEvent{EventID: riskEventID, EventType: domain.AuditPairRestoreRiskAccepted, Timestamp: now, PairID: request.PairID, Actor: request.Actor, Details: map[string]any{"operation_id": request.OperationID, "session_id": request.SessionID, "expected_generation": request.ExpectedGeneration, "risk_scope": request.RiskScope, "authorized_principal": request.AuthorizedPrincipal}})
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO restore_authorizations(authorization_id,operation_id,pair_id,session_id,expected_generation,risk_scope,authorized_principal,authorization_event_id,issued_at,consumed_at,consumed_by_operation_id) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, request.AuthorizationID, request.OperationID, request.PairID, request.SessionID, request.ExpectedGeneration, request.RiskScope, request.AuthorizedPrincipal, riskEventID, formatTime(now), formatTime(now), request.OperationID)
	if err != nil {
		return mapLifecycleWriteError(err, "restore authorization")
	}
	_, err = appendAuditEventTx(ctx, tx, domain.AuditEvent{EventID: requestEventID, EventType: domain.AuditPairRestoreRequested, Timestamp: now, PairID: request.PairID, Actor: request.Actor, Details: map[string]any{"operation_id": request.OperationID, "authorization_id": request.AuthorizationID, "session_id": request.SessionID, "expected_generation": request.ExpectedGeneration}})
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO pair_restore_operations(operation_id,authorization_id,pair_id,session_id,expected_generation,stage,resolution_state,requested_at,version) VALUES(?,?,?,?,?,'RESTORE_REQUESTED','IN_FLIGHT',?,0)`, request.OperationID, request.AuthorizationID, request.PairID, request.SessionID, request.ExpectedGeneration, formatTime(now))
	if err != nil {
		return mapLifecycleWriteError(err, "restore operation")
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: commit restore reservation: %w", err)
	}
	return nil
}

func (s *Store) GetPairRestoreOperation(ctx context.Context, operationID string) (domain.PairRestoreOperation, error) {
	var op domain.PairRestoreOperation
	var observed, mode, basis, principal, claimed, http200, resolved, eventID sql.NullString
	var requested string
	var state, stage string
	err := s.db.QueryRowContext(ctx, `SELECT operation_id,authorization_id,pair_id,session_id,expected_generation,observed_generation,restore_mode,stage,resolution_state,resolution_basis,recovery_principal,recovery_claimed_at,requested_at,http_200_at,resolved_at,resolution_event_id,version FROM pair_restore_operations WHERE operation_id=?`, operationID).Scan(&op.OperationID, &op.AuthorizationID, &op.PairID, &op.SessionID, &op.ExpectedGeneration, &observed, &mode, &stage, &state, &basis, &principal, &claimed, &requested, &http200, &resolved, &eventID, &op.Version)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.PairRestoreOperation{}, ErrOperationNotFound
	}
	if err != nil {
		return domain.PairRestoreOperation{}, err
	}
	op.Stage = stage
	op.ResolutionState = domain.RestoreResolutionState(state)
	setNullableString(&op.ObservedGeneration, observed)
	setNullableString(&op.RestoreMode, mode)
	setNullableString(&op.RecoveryPrincipal, principal)
	if basis.Valid {
		v := domain.RestoreResolutionBasis(basis.String)
		op.ResolutionBasis = &v
	}
	op.RecoveryClaimedAt, err = parseNullableTime(claimed)
	if err != nil {
		return domain.PairRestoreOperation{}, err
	}
	op.RequestedAt, err = parseTime(requested)
	if err != nil {
		return domain.PairRestoreOperation{}, err
	}
	op.HTTP200At, err = parseNullableTime(http200)
	if err != nil {
		return domain.PairRestoreOperation{}, err
	}
	op.ResolvedAt, err = parseNullableTime(resolved)
	if err != nil {
		return domain.PairRestoreOperation{}, err
	}
	setNullableString(&op.ResolutionEventID, eventID)
	return op, nil
}

// RecordPairRestoreUnknown records an ambiguous transport/protocol result. It
// never authorizes replay of /restore and leaves the Pair locked.
func (s *Store) RecordPairRestoreUnknown(ctx context.Context, operationID, actor string) error {
	return s.transitionRestore(ctx, operationID, actor, domain.AuditPairRestoreOutcomeUnknown,
		`UPDATE pair_restore_operations SET resolution_state='RESTORE_OUTCOME_UNKNOWN',version=version+1 WHERE operation_id=? AND resolution_state='IN_FLIGHT' AND version=?`, nil)
}

func (s *Store) transitionRestore(ctx context.Context, operationID, actor, eventType, update string, fields []any) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var pairID string
	var version int64
	if err := tx.QueryRowContext(ctx, "SELECT pair_id,version FROM pair_restore_operations WHERE operation_id=?", operationID).Scan(&pairID, &version); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrOperationNotFound
		}
		return err
	}
	eventID, err := newAuditEventID()
	if err != nil {
		return err
	}
	now := timeNow()
	_, err = appendAuditEventTx(ctx, tx, domain.AuditEvent{EventID: eventID, EventType: eventType, Timestamp: now, PairID: pairID, Actor: actor, Details: map[string]any{"operation_id": operationID}})
	if err != nil {
		return err
	}
	args := append(fields, operationID, version)
	res, err := tx.ExecContext(ctx, update, args...)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("%w: restore operation CAS lost", ErrStateConflict)
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

// ConfirmPairRestore records validated HTTP 200 provenance and updates the
// session generation atomically. It deliberately leaves quarantine in place.
func (s *Store) ConfirmPairRestore(ctx context.Context, operationID, sessionID, oldGeneration, newGeneration, restoreMode string, sessionStatus domain.WorkerSessionStatus, actor string, confirmedAt time.Time) error {
	if sessionID == "" || oldGeneration == "" || newGeneration == "" || restoreMode == "" || actor == "" || (sessionStatus != domain.WorkerSessionActive && sessionStatus != domain.WorkerSessionIdle && sessionStatus != domain.WorkerSessionTerminated) {
		return fmt.Errorf("store: incomplete restore confirmation")
	}
	if restoreMode != "native" && restoreMode != "saved_prompt" && restoreMode != "fresh" {
		return fmt.Errorf("store: invalid restore mode")
	}
	if confirmedAt.IsZero() {
		confirmedAt = timeNow()
	}
	eventID, err := newAuditEventID()
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var pairID string
	var version int64
	var expected string
	var stage, resolution string
	if err := tx.QueryRowContext(ctx, `SELECT pair_id,version,expected_generation,stage,resolution_state FROM pair_restore_operations WHERE operation_id=?`, operationID).Scan(&pairID, &version, &expected, &stage, &resolution); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrOperationNotFound
		}
		return err
	}
	if expected != oldGeneration || stage != "RESTORE_REQUESTED" || resolution != "IN_FLIGHT" {
		return fmt.Errorf("%w: restore confirmation is stale", ErrStateConflict)
	}
	var currentSession, currentGeneration string
	if err := tx.QueryRowContext(ctx, `SELECT session_id,terminal_generation FROM worker_sessions WHERE pair_id=?`, pairID).Scan(&currentSession, &currentGeneration); err != nil {
		return err
	}
	if currentSession != sessionID || currentGeneration != oldGeneration {
		return fmt.Errorf("%w: worker session snapshot changed", ErrStateConflict)
	}
	_, err = appendAuditEventTx(ctx, tx, domain.AuditEvent{EventID: eventID, EventType: domain.AuditPairRestoreConfirmed, Timestamp: confirmedAt, PairID: pairID, Actor: actor, Details: map[string]any{"operation_id": operationID, "session_id": sessionID, "old_generation": oldGeneration, "observed_generation": newGeneration, "restore_mode": restoreMode, "basis": "RESTORE_HTTP_200_CONFIRMED"}})
	if err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, `UPDATE pair_restore_operations SET stage='RESTORE_CONFIRMED',observed_generation=?,restore_mode=?,http_200_at=?,version=version+1 WHERE operation_id=? AND version=? AND stage='RESTORE_REQUESTED' AND resolution_state='IN_FLIGHT'`, newGeneration, restoreMode, formatTime(confirmedAt), operationID, version)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("%w: restore confirmation CAS lost", ErrStateConflict)
	}
	res, err = tx.ExecContext(ctx, `UPDATE worker_sessions SET terminal_generation=?,status=?,quarantine_state='QUARANTINED',updated_at=? WHERE pair_id=? AND session_id=? AND terminal_generation=?`, newGeneration, string(sessionStatus), formatTime(confirmedAt), pairID, sessionID, oldGeneration)
	if err != nil {
		return err
	}
	n, err = res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("%w: worker session restore CAS lost", ErrStateConflict)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: commit restore confirmation: %w", err)
	}
	return nil
}

// ResolveConfirmedRestore performs the narrow post-HTTP-200 operator resolution
// CAS after a fresh admissible observation. It does not clear any quarantine.
func (s *Store) ResolveConfirmedRestore(ctx context.Context, operationID, sessionID, observedGeneration, activityState, principal, actor string, resolvedAt time.Time) error {
	if sessionID == "" || observedGeneration == "" || principal == "" || actor == "" || (activityState != "idle" && activityState != "waiting_input") {
		return fmt.Errorf("store: restore resolution requires exact idle/waiting observation")
	}
	if resolvedAt.IsZero() {
		resolvedAt = timeNow()
	}
	eventID, err := newAuditEventID()
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var pairID, opSession, opObserved, stage, resolution string
	var basis, owner sql.NullString
	var version int64
	if err := tx.QueryRowContext(ctx, `SELECT pair_id,session_id,observed_generation,stage,resolution_state,version,resolution_basis,recovery_principal FROM pair_restore_operations WHERE operation_id=?`, operationID).Scan(&pairID, &opSession, &opObserved, &stage, &resolution, &version, &basis, &owner); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrOperationNotFound
		}
		return err
	}
	if opSession != sessionID || opObserved != observedGeneration || stage != "RESTORE_CONFIRMED" {
		return fmt.Errorf("%w: restore resolution observation is stale", ErrStateConflict)
	}
	alreadyResolved := resolution == string(domain.RestoreResolved) && basis.Valid && basis.String == string(domain.RestoreHTTP200Confirmed) && owner.Valid && owner.String == principal
	if !alreadyResolved && resolution != string(domain.RestoreInFlight) && resolution != string(domain.RestoreRecoveryClaimed) {
		return fmt.Errorf("%w: restore is not owned for resolution", ErrStateConflict)
	}
	if !alreadyResolved && resolution == string(domain.RestoreRecoveryClaimed) && (!owner.Valid || owner.String != principal) {
		return fmt.Errorf("%w: restore recovery claim belongs to another principal", ErrStateConflict)
	}
	var currentSession, currentGeneration string
	if err := tx.QueryRowContext(ctx, `SELECT session_id,terminal_generation FROM worker_sessions WHERE pair_id=?`, pairID).Scan(&currentSession, &currentGeneration); err != nil {
		return err
	}
	if currentSession != sessionID || currentGeneration != observedGeneration {
		return fmt.Errorf("%w: current WorkerSession differs from observed restore generation", ErrStateConflict)
	}
	var pendingStop int
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM stop_operations WHERE restore_operation_id=? AND resolution_state NOT IN ('TERMINATION_CONFIRMED','STOP_TARGET_ABSENT','ADMINISTRATIVE_RISK_ACCEPTED'))`, operationID).Scan(&pendingStop); err != nil {
		return err
	}
	if pendingStop != 0 {
		return fmt.Errorf("%w: linked stop must be terminal before restore resolution", ErrStateConflict)
	}
	if alreadyResolved {
		return nil
	}
	if resolution == string(domain.RestoreInFlight) {
		claimEventID, claimErr := newAuditEventID()
		if claimErr != nil {
			return claimErr
		}
		_, claimErr = appendAuditEventTx(ctx, tx, domain.AuditEvent{EventID: claimEventID, EventType: domain.AuditPairRestoreRecoveryClaimed, Timestamp: resolvedAt, PairID: pairID, Actor: actor, Details: map[string]any{"operation_id": operationID, "session_id": sessionID, "observed_generation": observedGeneration, "principal": principal}})
		if claimErr != nil {
			return claimErr
		}
		res, err := tx.ExecContext(ctx, `UPDATE pair_restore_operations SET resolution_state='RESTORE_RECOVERY_CLAIMED',recovery_principal=?,recovery_claimed_at=?,version=version+1 WHERE operation_id=? AND stage='RESTORE_CONFIRMED' AND resolution_state='IN_FLIGHT' AND version=?`, principal, formatTime(resolvedAt), operationID, version)
		if err != nil {
			return err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if n != 1 {
			return fmt.Errorf("%w: restore recovery claim CAS lost", ErrStateConflict)
		}
		version++
	}
	_, err = appendAuditEventTx(ctx, tx, domain.AuditEvent{EventID: eventID, EventType: domain.AuditPairRestoreResolved, Timestamp: resolvedAt, PairID: pairID, Actor: actor, Details: map[string]any{"operation_id": operationID, "session_id": sessionID, "observed_generation": observedGeneration, "activity_state": activityState, "basis": "RESTORE_HTTP_200_CONFIRMED"}})
	if err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, `UPDATE pair_restore_operations SET resolution_state='RESTORE_RESOLVED',resolution_basis='RESTORE_HTTP_200_CONFIRMED',resolved_at=?,resolution_event_id=?,version=version+1 WHERE operation_id=? AND stage='RESTORE_CONFIRMED' AND resolution_state='RESTORE_RECOVERY_CLAIMED' AND version=? AND observed_generation=?`, formatTime(resolvedAt), eventID, operationID, version, observedGeneration)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("%w: restore resolution CAS lost", ErrStateConflict)
	}
	return tx.Commit()
}

// ResolveAmbiguousRestore records a separately authorized physical resolution
// or administrative risk acceptance. It never mutates/clears quarantine.
func (s *Store) ResolveAmbiguousRestore(ctx context.Context, operationID, pairID, sessionID, generation string, basis domain.RestoreResolutionBasis, evidence, principal, actor string, resolvedAt time.Time) error {
	if operationID == "" || pairID == "" || sessionID == "" || generation == "" || (basis != domain.PhysicalExecutionResolution && basis != domain.AdministrativeRiskResolution) || evidence == "" || principal == "" || actor == "" {
		return fmt.Errorf("store: ambiguous restore resolution requires exact tuple, basis, evidence, and verified principal")
	}
	if resolvedAt.IsZero() {
		resolvedAt = timeNow()
	}
	claimEvent, err := newAuditEventID()
	if err != nil {
		return err
	}
	resolutionEvent, err := newAuditEventID()
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var storedPair, storedSession, storedGeneration, stage, state string
	var storedBasis, storedOwner sql.NullString
	var version int64
	if err := tx.QueryRowContext(ctx, `SELECT pair_id,session_id,expected_generation,stage,resolution_state,version,resolution_basis,recovery_principal FROM pair_restore_operations WHERE operation_id=?`, operationID).Scan(&storedPair, &storedSession, &storedGeneration, &stage, &state, &version, &storedBasis, &storedOwner); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrOperationNotFound
		}
		return err
	}
	if storedPair != pairID || storedSession != sessionID || storedGeneration != generation {
		return fmt.Errorf("%w: restore authorization tuple mismatch", ErrAttemptLineageMismatch)
	}
	var pendingStop int
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM stop_operations WHERE restore_operation_id=? AND resolution_state NOT IN ('TERMINATION_CONFIRMED','STOP_TARGET_ABSENT','ADMINISTRATIVE_RISK_ACCEPTED'))`, operationID).Scan(&pendingStop); err != nil {
		return err
	}
	if pendingStop != 0 {
		return fmt.Errorf("%w: linked stop must be terminal before restore resolution", ErrStateConflict)
	}
	if state == string(domain.RestoreResolved) && storedBasis.Valid && storedBasis.String == string(basis) && storedOwner.Valid && storedOwner.String == principal {
		return nil
	}
	if state != "RESTORE_OUTCOME_UNKNOWN" && state != "RESTORE_RECOVERY_CLAIMED" && state != "RESTORE_CLEANUP_CLAIMED" {
		return fmt.Errorf("%w: restore is not awaiting risk resolution", ErrStateConflict)
	}
	if basis == domain.PhysicalExecutionResolution {
		var positiveStop int
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(
			SELECT 1 FROM stop_operations o
			JOIN audit_events e ON e.event_type='STOP_OPERATION_CONFIRMED'
			 AND json_extract(e.details_json,'$.stop_operation_id')=o.operation_id
			 AND json_extract(e.details_json,'$.restore_operation_id')=o.restore_operation_id
			 AND e.pair_id=o.pair_id AND e.task_id IS o.task_id AND e.contract_id IS o.contract_id AND e.attempt_id IS o.attempt_id
			 AND json_extract(e.details_json,'$.session_id')=o.session_id
			 AND json_extract(e.details_json,'$.target_generation')=o.terminal_generation
			 AND json_extract(e.details_json,'$.confirmation_deadline_at')=o.confirmation_deadline_at
			 AND json_extract(e.details_json,'$.termination_confirmed_at')=o.termination_confirmed_at
			 AND json_extract(e.details_json,'$.observed_session_id')=o.session_id
			 AND json_extract(e.details_json,'$.observed_generation')=o.terminal_generation
			 AND json_extract(e.details_json,'$.is_terminated')=1
			WHERE o.restore_operation_id=? AND o.pair_id=? AND o.session_id=? AND o.terminal_generation=?
			 AND o.stage='STOP_TERMINATION_CONFIRMED' AND o.resolution_state='TERMINATION_CONFIRMED'
			 AND o.call_completed_at IS NOT NULL AND o.confirmation_deadline_at IS NOT NULL
			 AND o.termination_confirmed_at IS NOT NULL AND o.resolved_at IS NOT NULL
			 AND o.termination_confirmed_at<=o.confirmation_deadline_at
			 AND ((o.purpose='PAIR_MAINTENANCE' AND o.task_id IS NULL AND o.contract_id IS NULL AND o.attempt_id IS NULL)
			       OR (o.purpose='QUARANTINE_CLEANUP' AND o.attempt_id IS NOT NULL
			       AND EXISTS(SELECT 1 FROM task_attempts a JOIN tasks t ON t.task_id=a.task_id
			         WHERE a.attempt_id=o.attempt_id AND a.task_id=o.task_id AND a.contract_id=o.contract_id
			           AND t.pair_id=o.pair_id AND a.session_id=o.session_id
			           AND a.terminal_generation=o.terminal_generation AND a.ended_at IS NOT NULL
			           AND a.quarantine_state='QUARANTINED'))))`, operationID, pairID, sessionID, generation).Scan(&positiveStop); err != nil {
			return fmt.Errorf("store: verify positive D11 physical stop evidence: %w", err)
		}
		if positiveStop != 1 {
			return fmt.Errorf("%w: Class A requires positive D11 stop evidence matching restore/session/generation/deadline/lineage; assertion is insufficient", ErrStateConflict)
		}
	}
	if _, err := appendAuditEventTx(ctx, tx, domain.AuditEvent{EventID: claimEvent, EventType: domain.AuditPairRestoreRecoveryClaimed, Timestamp: resolvedAt, PairID: pairID, Actor: actor, Details: map[string]any{"operation_id": operationID, "basis": string(basis), "principal": principal, "evidence": evidence}}); err != nil {
		return err
	}
	resolutionSource := state
	if state == "RESTORE_OUTCOME_UNKNOWN" {
		res, err := tx.ExecContext(ctx, `UPDATE pair_restore_operations SET resolution_state='RESTORE_RECOVERY_CLAIMED',recovery_principal=?,recovery_claimed_at=?,version=version+1 WHERE operation_id=? AND resolution_state=? AND version=?`, principal, formatTime(resolvedAt), operationID, state, version)
		if err != nil {
			return err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if n != 1 {
			return fmt.Errorf("%w: restore recovery claim lost", ErrStateConflict)
		}
		version++
		resolutionSource = "RESTORE_RECOVERY_CLAIMED"
	} else {
		var owner string
		if err := tx.QueryRowContext(ctx, `SELECT recovery_principal FROM pair_restore_operations WHERE operation_id=?`, operationID).Scan(&owner); err != nil {
			return err
		}
		if owner != principal {
			return fmt.Errorf("%w: restore recovery is owned by another principal", ErrStateConflict)
		}
	}
	if _, err := appendAuditEventTx(ctx, tx, domain.AuditEvent{EventID: resolutionEvent, EventType: domain.AuditPairRestoreResolved, Timestamp: resolvedAt, PairID: pairID, Actor: actor, Details: map[string]any{"operation_id": operationID, "basis": string(basis), "principal": principal, "evidence": evidence}}); err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, `UPDATE pair_restore_operations SET resolution_state='RESTORE_RESOLVED',resolution_basis=?,resolved_at=?,resolution_event_id=?,version=version+1 WHERE operation_id=? AND resolution_state=? AND version=?`, string(basis), formatTime(resolvedAt), resolutionEvent, operationID, resolutionSource, version)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("%w: restore resolution lost", ErrStateConflict)
	}
	_ = stage
	_ = sessionID
	return tx.Commit()
}

// ClaimRestoreCleanup transfers one unresolved restore operation to the narrow
// linked-stop path. It does not authorize a /kill call by itself.
func (s *Store) ClaimRestoreCleanup(ctx context.Context, operationID, pairID, sessionID, targetGeneration, principal, actor string, claimedAt time.Time) error {
	_ = ctx
	_ = operationID
	_ = pairID
	_ = sessionID
	_ = targetGeneration
	_ = principal
	_ = actor
	_ = claimedAt
	return fmt.Errorf("%w: cleanup ownership and linked STOP_REQUESTED must commit via ClaimRestoreCleanupWithStop", ErrAtomicDispatchRequired)
}

// ClaimRestoreCleanupWithStop atomically transfers restore cleanup ownership,
// persists the linked STOP_REQUESTED row, and writes both required audit events.
func (s *Store) ClaimRestoreCleanupWithStop(ctx context.Context, stop domain.StopOperation, principal, actor string, claimedAt time.Time) error {
	if stop.OperationID == "" || stop.RestoreOperationID == nil || *stop.RestoreOperationID == "" || stop.PairID == "" || stop.SessionID == "" || stop.TerminalGeneration == "" || principal == "" || actor == "" {
		return fmt.Errorf("store: linked restore cleanup stop tuple is incomplete")
	}
	if stop.RestorePrincipal == nil || *stop.RestorePrincipal != principal {
		return fmt.Errorf("%w: linked stop principal must match cleanup owner", ErrStateConflict)
	}
	if claimedAt.IsZero() {
		claimedAt = timeNow()
	}
	if stop.RequestedAt.IsZero() {
		stop.RequestedAt = claimedAt
	}
	if stop.Actor == "" {
		stop.Actor = actor
	}
	if stop.Stage == "" {
		stop.Stage = domain.StopRequested
	}
	if stop.ResolutionState == "" {
		stop.ResolutionState = domain.StopResolutionInFlight
	}
	if stop.Stage != domain.StopRequested || stop.ResolutionState != domain.StopResolutionInFlight {
		return fmt.Errorf("store: linked cleanup must begin at STOP_REQUESTED/IN_FLIGHT")
	}
	claimEventID, err := newAuditEventID()
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var pairID, sessionID, expected string
	var observed sql.NullString
	var state string
	var version int64
	var owner sql.NullString
	if err := tx.QueryRowContext(ctx, `SELECT pair_id,session_id,expected_generation,observed_generation,resolution_state,version,recovery_principal FROM pair_restore_operations WHERE operation_id=?`, *stop.RestoreOperationID).Scan(&pairID, &sessionID, &expected, &observed, &state, &version, &owner); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrOperationNotFound
		}
		return err
	}
	if pairID != stop.PairID || sessionID != stop.SessionID || state != string(domain.RestoreRecoveryClaimed) || !owner.Valid || owner.String != principal || (stop.TerminalGeneration != expected && (!observed.Valid || stop.TerminalGeneration != observed.String)) {
		return fmt.Errorf("%w: cleanup claim tuple/owner/generation mismatch", ErrStateConflict)
	}
	if _, err := appendAuditEventTx(ctx, tx, domain.AuditEvent{EventID: claimEventID, EventType: domain.AuditPairRestoreCleanupClaimed, Timestamp: claimedAt, PairID: pairID, Actor: actor, Details: map[string]any{"operation_id": *stop.RestoreOperationID, "session_id": sessionID, "target_generation": stop.TerminalGeneration, "principal": principal}}); err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, `UPDATE pair_restore_operations SET resolution_state='RESTORE_CLEANUP_CLAIMED',version=version+1 WHERE operation_id=? AND pair_id=? AND session_id=? AND resolution_state='RESTORE_RECOVERY_CLAIMED' AND recovery_principal=? AND version=?`, *stop.RestoreOperationID, pairID, sessionID, principal, version)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err != nil || n != 1 {
		if err != nil {
			return err
		}
		return fmt.Errorf("%w: cleanup ownership CAS lost", ErrStateConflict)
	}
	if err := createStopOperationTx(ctx, tx, stop); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) ClaimRestoreRecovery(ctx context.Context, operationID, pairID, sessionID, generation, principal, actor string, claimedAt time.Time) error {
	if operationID == "" || pairID == "" || sessionID == "" || generation == "" || principal == "" || actor == "" {
		return fmt.Errorf("store: restore recovery claim tuple is incomplete")
	}
	if claimedAt.IsZero() {
		claimedAt = timeNow()
	}
	eventID, err := newAuditEventID()
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var storedPair, storedSession, expected, stage, state string
	var observed sql.NullString
	var version int64
	if err := tx.QueryRowContext(ctx, `SELECT pair_id,session_id,expected_generation,observed_generation,stage,resolution_state,version FROM pair_restore_operations WHERE operation_id=?`, operationID).Scan(&storedPair, &storedSession, &expected, &observed, &stage, &state, &version); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrOperationNotFound
		}
		return err
	}
	allowed := state == string(domain.RestoreOutcomeUnknown) || (state == string(domain.RestoreInFlight) && stage == "RESTORE_CONFIRMED")
	genMatches := generation == expected || (observed.Valid && generation == observed.String)
	if storedPair != pairID || storedSession != sessionID || !allowed || !genMatches {
		return fmt.Errorf("%w: restore recovery claim tuple/state mismatch", ErrStateConflict)
	}
	_, err = appendAuditEventTx(ctx, tx, domain.AuditEvent{EventID: eventID, EventType: domain.AuditPairRestoreRecoveryClaimed, Timestamp: claimedAt, PairID: pairID, Actor: actor, Details: map[string]any{"operation_id": operationID, "session_id": sessionID, "generation": generation, "principal": principal}})
	if err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, `UPDATE pair_restore_operations SET resolution_state='RESTORE_RECOVERY_CLAIMED',recovery_principal=?,recovery_claimed_at=?,version=version+1 WHERE operation_id=? AND pair_id=? AND session_id=? AND resolution_state=? AND version=?`, principal, formatTime(claimedAt), operationID, pairID, sessionID, state, version)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("%w: restore recovery claim CAS lost", ErrStateConflict)
	}
	return tx.Commit()
}
