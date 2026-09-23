package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/trungqwe/ai-supervisor/internal/domain"
	"github.com/trungqwe/ai-supervisor/internal/workflow"
)

// RecordPreSendHold durably records a disposition on the exact bound attempt.
// Protocol uncertainty has precedence over availability observations and can
// only be cleared through the authorized protocol reconciliation path.
func (s *Store) RecordPreSendHold(ctx context.Context, operationID, disposition, actor string, at time.Time) error {
	allowed := map[string]bool{"RECOVERY_PENDING": true, "PRE_SEND_PROTOCOL_UNVERIFIED": true}
	if !allowed[disposition] || actor == "" {
		return fmt.Errorf("store: invalid pre-send hold or actor")
	}
	if at.IsZero() {
		at = timeNow()
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
	var pairID, taskID, contractID, attemptID, sessionID, generation, stage string
	var current sql.NullString
	if err := tx.QueryRowContext(ctx, `SELECT d.pair_id,d.task_id,a.contract_id,d.attempt_id,d.session_id,d.terminal_generation,d.stage,a.recovery_disposition FROM dispatch_operations d JOIN task_attempts a ON a.attempt_id=d.attempt_id WHERE d.operation_id=?`, operationID).Scan(&pairID, &taskID, &contractID, &attemptID, &sessionID, &generation, &stage, &current); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrOperationNotFound
		}
		return err
	}
	if stage != string(domain.DispatchBound) {
		return fmt.Errorf("%w: pre-send hold requires DISPATCH_BOUND", ErrStateConflict)
	}
	if current.Valid {
		if current.String == disposition {
			return nil
		}
		if current.String == "PRE_SEND_PROTOCOL_UNVERIFIED" || disposition != "PRE_SEND_PROTOCOL_UNVERIFIED" {
			return fmt.Errorf("%w: pre-send hold precedence prevents downgrade", ErrStateConflict)
		}
	}
	_, err = appendAuditEventTx(ctx, tx, domain.AuditEvent{EventID: eventID, EventType: domain.AuditPreSendAdmissibilityRejected, Timestamp: at, PairID: pairID, TaskID: taskID, ContractID: contractID, AttemptID: attemptID, Actor: actor, Details: map[string]any{"dispatch_operation_id": operationID, "pre_send_disposition": disposition}})
	if err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, `UPDATE task_attempts SET recovery_disposition=? WHERE attempt_id=? AND ended_at IS NULL AND recovery_disposition IS ? AND session_id=? AND terminal_generation=?`, disposition, attemptID, nullableString(nullStringPtr(current)), sessionID, generation)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("%w: pre-send hold CAS lost", ErrStateConflict)
	}
	return tx.Commit()
}

func nullStringPtr(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

// RecordPreSendAdmissibilityRejected writes the ADR-016 §22 rejection evidence
// while preserving DISPATCH_BOUND, DISPATCHED, and the open attempt.
func (s *Store) RecordPreSendAdmissibilityRejected(ctx context.Context, operationID, observedActivity, actor string, at time.Time) error {
	if observedActivity != "active" && observedActivity != "blocked" && observedActivity != "exited" {
		return fmt.Errorf("store: invalid pre-send inadmissible activity")
	}
	if actor == "" {
		return fmt.Errorf("store: actor required")
	}
	if at.IsZero() {
		at = timeNow()
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
	var pairID, taskID, contractID, attemptID, stage string
	if err := tx.QueryRowContext(ctx, `SELECT d.pair_id,d.task_id,a.contract_id,d.attempt_id,d.stage FROM dispatch_operations d JOIN task_attempts a ON a.attempt_id=d.attempt_id WHERE d.operation_id=? AND a.ended_at IS NULL`, operationID).Scan(&pairID, &taskID, &contractID, &attemptID, &stage); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrAttemptLineageMismatch
		}
		return err
	}
	if stage != string(domain.DispatchBound) {
		return fmt.Errorf("%w: rejection requires DISPATCH_BOUND", ErrStateConflict)
	}
	_, err = appendAuditEventTx(ctx, tx, domain.AuditEvent{EventID: eventID, EventType: domain.AuditPreSendAdmissibilityRejected, Timestamp: at, PairID: pairID, TaskID: taskID, ContractID: contractID, AttemptID: attemptID, Actor: actor, Details: map[string]any{"operation_id": operationID, "error_code": "PRE_SEND_ADMISSIBILITY_REJECTED", "observed_activity": observedActivity, "task_state": string(domain.StateDispatched), "attempt_remains_open": true}})
	if err != nil {
		return err
	}
	return tx.Commit()
}

// ResolvePreSendHold clears only the exact open-attempt hold with matching
// lineage, operator principal, and fresh admissible AO status.
func (s *Store) ResolvePreSendHold(ctx context.Context, operationID, attemptID, expectedDisposition, sessionID, generation, activity, principal, actor string, at time.Time) error {
	if attemptID == "" || sessionID == "" || generation == "" || actor == "" || (activity != "idle" && activity != "waiting_input") || (expectedDisposition != "RECOVERY_PENDING" && principal == "") {
		return fmt.Errorf("store: pre-send recovery requires verified exact observation")
	}
	if expectedDisposition != "RECOVERY_PENDING" && expectedDisposition != "PRE_SEND_PROTOCOL_UNVERIFIED" {
		return fmt.Errorf("store: invalid pre-send disposition")
	}
	if at.IsZero() {
		at = timeNow()
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
	var pairID, taskID, contractID, snapshotSession, snapshotGeneration string
	var currentDisposition sql.NullString
	if err := tx.QueryRowContext(ctx, `SELECT t.pair_id,a.task_id,a.contract_id,a.session_id,a.terminal_generation,a.recovery_disposition FROM task_attempts a JOIN tasks t ON t.task_id=a.task_id JOIN dispatch_operations d ON d.attempt_id=a.attempt_id WHERE a.attempt_id=? AND d.operation_id=? AND d.stage='DISPATCH_BOUND' AND d.task_id=a.task_id AND d.pair_id=t.pair_id AND d.session_id=a.session_id AND d.terminal_generation=a.terminal_generation AND a.attempt_number=t.current_attempt AND a.ended_at IS NULL`, attemptID, operationID).Scan(&pairID, &taskID, &contractID, &snapshotSession, &snapshotGeneration, &currentDisposition); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrAttemptLineageMismatch
		}
		return err
	}
	if !currentDisposition.Valid || currentDisposition.String != expectedDisposition || snapshotSession != sessionID || snapshotGeneration != generation {
		return fmt.Errorf("%w: hold disposition or immutable lineage changed", ErrStateConflict)
	}
	var currentSession, currentGeneration string
	if err := tx.QueryRowContext(ctx, `SELECT session_id,terminal_generation FROM worker_sessions WHERE pair_id=?`, pairID).Scan(&currentSession, &currentGeneration); err != nil {
		return err
	}
	if currentSession != sessionID || currentGeneration != generation {
		return fmt.Errorf("%w: session lineage changed", ErrStateConflict)
	}
	_, err = appendAuditEventTx(ctx, tx, domain.AuditEvent{EventID: eventID, EventType: domain.AuditPreSendStatusRecovered, Timestamp: at, PairID: pairID, TaskID: taskID, ContractID: contractID, AttemptID: attemptID, Actor: actor, Details: map[string]any{"expected_disposition": expectedDisposition, "session_id": sessionID, "terminal_generation": generation, "activity_state": activity, "authorized_principal": principal}})
	if err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, `UPDATE task_attempts SET recovery_disposition=NULL WHERE attempt_id=? AND ended_at IS NULL AND recovery_disposition=? AND session_id=? AND terminal_generation=?`, attemptID, expectedDisposition, sessionID, generation)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("%w: attempt hold CAS lost", ErrStateConflict)
	}
	return tx.Commit()
}

// RecordSendRequested commits the durable send intent and audit before the AO
// network call. It rejects stale binding, unresolved Pair state, and any hold.
func (s *Store) RecordSendRequested(ctx context.Context, operationID, observedSessionID, observedGeneration, activity string, isTerminated bool, actor string, at time.Time) error {
	if actor == "" || observedSessionID == "" || observedGeneration == "" || (activity != "idle" && activity != "waiting_input") || isTerminated {
		return fmt.Errorf("store: send intent requires a positive exact idle/waiting pre-send observation")
	}
	if at.IsZero() {
		at = timeNow()
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
	var pairID, taskID, contractID, attemptID, sessionID, generation, stage string
	var disposition sql.NullString
	err = tx.QueryRowContext(ctx, `SELECT d.pair_id,d.task_id,a.contract_id,d.attempt_id,d.session_id,d.terminal_generation,d.stage,a.recovery_disposition FROM dispatch_operations d JOIN task_attempts a ON a.attempt_id=d.attempt_id WHERE d.operation_id=?`, operationID).Scan(&pairID, &taskID, &contractID, &attemptID, &sessionID, &generation, &stage, &disposition)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrOperationNotFound
		}
		return err
	}
	if stage != string(domain.DispatchBound) || disposition.Valid {
		return fmt.Errorf("%w: dispatch stage/hold precondition", ErrStateConflict)
	}
	if sessionID != observedSessionID || generation != observedGeneration {
		return fmt.Errorf("%w: GET observation does not match bound dispatch", ErrAttemptLineageMismatch)
	}
	var valid int
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM worker_sessions w WHERE w.pair_id=? AND w.session_id=? AND w.terminal_generation=? AND w.quarantine_state='CLEAN') AND NOT EXISTS(SELECT 1 FROM pair_restore_operations r WHERE r.pair_id=? AND r.resolution_state<>'RESTORE_RESOLVED') AND NOT EXISTS(SELECT 1 FROM pair_provisioning_operations p WHERE p.pair_id=? AND p.stage IN ('PROVISION_REQUESTED','PROVISION_FAILED')) AND EXISTS(SELECT 1 FROM task_attempts a JOIN tasks t ON t.task_id=a.task_id WHERE t.pair_id=? AND a.attempt_id=? AND a.attempt_number=t.current_attempt AND t.state='DISPATCHED' AND a.ended_at IS NULL AND a.session_id=? AND a.terminal_generation=?)`, pairID, sessionID, generation, pairID, pairID, pairID, attemptID, sessionID, generation).Scan(&valid); err != nil {
		return err
	}
	if valid != 1 {
		return fmt.Errorf("%w: dispatch binding or Pair guard rejected", ErrQuarantinedExecution)
	}
	_, err = appendAuditEventTx(ctx, tx, domain.AuditEvent{EventID: eventID, EventType: domain.AuditDispatchSendRequested, Timestamp: at, PairID: pairID, TaskID: taskID, ContractID: contractID, AttemptID: attemptID, Actor: actor, Details: map[string]any{"dispatch_operation_id": operationID, "stage": "SEND_REQUESTED", "session_id": sessionID, "terminal_generation": generation}})
	if err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, `UPDATE dispatch_operations SET stage='SEND_REQUESTED' WHERE operation_id=? AND stage='DISPATCH_BOUND' AND resolution_state IS NULL`, operationID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("%w: send intent CAS lost", ErrStateConflict)
	}
	return tx.Commit()
}

// RecordSendConfirmed accepts only a validated HTTP 200 response and leaves the
// Task in DISPATCHED; the response says nothing about execution state.
func (s *Store) RecordSendConfirmed(ctx context.Context, operationID, actor string, validatedHTTP200 bool, at time.Time) error {
	if !validatedHTTP200 {
		return fmt.Errorf("store: send confirmation requires validated HTTP 200")
	}
	if actor == "" {
		return fmt.Errorf("store: send confirmation requires actor")
	}
	if at.IsZero() {
		at = timeNow()
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
	var pairID, taskID, contractID, attemptID, stage string
	var taskState, opSession, opGeneration, currentSession, currentGeneration string
	var currentAttempt, attemptNumber int
	if err := tx.QueryRowContext(ctx, `SELECT d.pair_id,d.task_id,d.attempt_id,d.stage,d.session_id,d.terminal_generation,t.state,t.current_attempt,a.attempt_number,w.session_id,w.terminal_generation FROM dispatch_operations d JOIN tasks t ON t.task_id=d.task_id JOIN task_attempts a ON a.attempt_id=d.attempt_id JOIN worker_sessions w ON w.pair_id=d.pair_id WHERE d.operation_id=?`, operationID).Scan(&pairID, &taskID, &attemptID, &stage, &opSession, &opGeneration, &taskState, &currentAttempt, &attemptNumber, &currentSession, &currentGeneration); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrOperationNotFound
		}
		return err
	}
	if stage != string(domain.SendRequested) {
		return fmt.Errorf("%w: dispatch is not awaiting confirmation", ErrStateConflict)
	}
	if taskState != string(domain.StateDispatched) || currentAttempt != attemptNumber || currentSession != opSession || currentGeneration != opGeneration {
		return fmt.Errorf("%w: stale dispatch/session snapshot at confirmation", ErrAttemptLineageMismatch)
	}
	if err := tx.QueryRowContext(ctx, `SELECT contract_id FROM task_attempts WHERE attempt_id=? AND ended_at IS NULL`, attemptID).Scan(&contractID); err != nil {
		return fmt.Errorf("%w: attempt is closed", ErrAttemptLineageMismatch)
	}
	_, err = appendAuditEventTx(ctx, tx, domain.AuditEvent{EventID: eventID, EventType: domain.AuditDispatchSendConfirmed, Timestamp: at, PairID: pairID, TaskID: taskID, ContractID: contractID, AttemptID: attemptID, Actor: actor, Details: map[string]any{"dispatch_operation_id": operationID, "stage": "SEND_CONFIRMED", "http_status": 200}})
	if err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, `UPDATE dispatch_operations SET stage='SEND_CONFIRMED',confirmed_at=? WHERE operation_id=? AND stage='SEND_REQUESTED' AND resolution_state IS NULL`, formatTime(at), operationID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("%w: send confirmation CAS lost", ErrStateConflict)
	}
	return tx.Commit()
}

// RecordUnknownDelivery closes the attempt and quarantines both lineages in the
// same transaction as the terminal operation resolution and audit append.
func (s *Store) RecordUnknownDelivery(ctx context.Context, operationID, actor string, at time.Time) error {
	if actor == "" {
		return fmt.Errorf("store: unknown delivery requires actor")
	}
	if at.IsZero() {
		at = timeNow()
	}
	eventID, err := newAuditEventID()
	if err != nil {
		return err
	}
	escalationEventID, err := newAuditEventID()
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var pairID, taskID, contractID, attemptID, sessionID, generation, stage string
	if err := tx.QueryRowContext(ctx, `SELECT d.pair_id,d.task_id,a.contract_id,d.attempt_id,d.session_id,d.terminal_generation,d.stage FROM dispatch_operations d JOIN task_attempts a ON a.attempt_id=d.attempt_id WHERE d.operation_id=?`, operationID).Scan(&pairID, &taskID, &contractID, &attemptID, &sessionID, &generation, &stage); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrOperationNotFound
		}
		return err
	}
	if stage != string(domain.SendRequested) {
		return fmt.Errorf("%w: no unresolved send intent", ErrStateConflict)
	}
	var current string
	if err := tx.QueryRowContext(ctx, `SELECT state FROM tasks WHERE task_id=?`, taskID).Scan(&current); err != nil {
		return err
	}
	if current != string(domain.StateDispatched) {
		return fmt.Errorf("%w: task is not DISPATCHED", ErrStateConflict)
	}
	if err := workflow.Transition(domain.StateDispatched, domain.StateFailed); err != nil {
		return err
	}
	if err := workflow.Transition(domain.StateFailed, domain.StateHumanRequired); err != nil {
		return err
	}
	_, err = appendAuditEventTx(ctx, tx, domain.AuditEvent{EventID: eventID, EventType: domain.AuditTaskStateTransition, Timestamp: at, PairID: pairID, TaskID: taskID, ContractID: contractID, AttemptID: attemptID, Actor: actor, Details: map[string]any{"dispatch_operation_id": operationID, "from_state": "DISPATCHED", "to_state": "FAILED", "failure_reason": domain.RecoveryUncertainDeliveryCrash, "recovery_disposition": domain.RecoveryUncertainDeliveryCrash, "resolution_state": domain.RecoveryDeliveryOutcomeUnknown}})
	if err != nil {
		return err
	}
	quarantineEventID, err := newAuditEventID()
	if err != nil {
		return err
	}
	_, err = appendAuditEventTx(ctx, tx, domain.AuditEvent{EventID: quarantineEventID, EventType: domain.AuditUncertainDeliveryQuarantine, Timestamp: at, PairID: pairID, TaskID: taskID, ContractID: contractID, AttemptID: attemptID, Actor: actor, Details: map[string]any{"dispatch_operation_id": operationID, "resolution_state": domain.RecoveryDeliveryOutcomeUnknown, "session_id": sessionID, "terminal_generation": generation}})
	if err != nil {
		return err
	}
	_, err = appendAuditEventTx(ctx, tx, domain.AuditEvent{EventID: escalationEventID, EventType: domain.AuditTaskStateTransition, Timestamp: at, PairID: pairID, TaskID: taskID, ContractID: contractID, AttemptID: attemptID, Actor: actor, Details: map[string]any{"dispatch_operation_id": operationID, "from_state": "FAILED", "to_state": "HUMAN_REQUIRED", "reason": domain.RecoveryUncertainDeliveryCrash}})
	if err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, `UPDATE tasks SET state='FAILED',updated_at=? WHERE task_id=? AND state='DISPATCHED'`, formatTime(at), taskID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("%w: task terminal CAS lost", ErrStateConflict)
	}
	res, err = tx.ExecContext(ctx, `UPDATE tasks SET state='HUMAN_REQUIRED',updated_at=? WHERE task_id=? AND state='FAILED'`, formatTime(at), taskID)
	if err != nil {
		return err
	}
	n, err = res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("%w: human-required escalation CAS lost", ErrStateConflict)
	}
	res, err = tx.ExecContext(ctx, `UPDATE task_attempts SET ended_at=?,recovery_disposition=?,quarantine_state='QUARANTINED' WHERE attempt_id=? AND task_id=? AND ended_at IS NULL AND attempt_number=(SELECT current_attempt FROM tasks WHERE task_id=?)`, formatTime(at), domain.RecoveryUncertainDeliveryCrash, attemptID, taskID, taskID)
	if err != nil {
		return err
	}
	n, err = res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("%w: current attempt close CAS lost", ErrAttemptLineageMismatch)
	}
	res, err = tx.ExecContext(ctx, `UPDATE dispatch_operations SET resolution_state='DELIVERY_OUTCOME_UNKNOWN' WHERE operation_id=? AND stage='SEND_REQUESTED' AND resolution_state IS NULL`, operationID)
	if err != nil {
		return err
	}
	n, err = res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("%w: dispatch resolution CAS lost", ErrStateConflict)
	}
	res, err = tx.ExecContext(ctx, `UPDATE worker_sessions SET quarantine_state='QUARANTINED',updated_at=? WHERE pair_id=? AND session_id=? AND terminal_generation=?`, formatTime(at), pairID, sessionID, generation)
	if err != nil {
		return err
	}
	n, err = res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("%w: session quarantine target changed", ErrStateConflict)
	}
	return tx.Commit()
}
