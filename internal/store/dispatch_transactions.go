package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
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
func (s *Store) RecordSendConfirmed(ctx context.Context, operationID, actor string, validatedHTTP200 bool, at time.Time, policies ...domain.ExecutionBudgetPolicy) error {
	if !validatedHTTP200 {
		return fmt.Errorf("store: send confirmation requires validated HTTP 200")
	}
	if actor == "" {
		return fmt.Errorf("store: send confirmation requires actor")
	}
	if len(policies) != 1 || policies[0].Duration <= 0 || strings.TrimSpace(policies[0].PolicyRef) == "" {
		return fmt.Errorf("%w: immutable execution budget policy required", ErrStateConflict)
	}
	if at.IsZero() {
		at = timeNow()
	}
	at = at.UTC()
	deadline := at.Add(policies[0].Duration)
	if !deadline.After(at) || at.Year() < 1 || deadline.Year() > 9999 {
		return fmt.Errorf("%w: execution budget deadline invalid or overflowed", ErrStateConflict)
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
	_, err = tx.ExecContext(ctx, `INSERT INTO attempt_execution_budgets(attempt_id,dispatch_operation_id,origin_at,deadline_at,duration_ns,policy_ref,binding_basis,bound_at) VALUES(?,?,?,?,?,?,'SEND_CONFIRMATION_ATOMIC',?)`, attemptID, operationID, formatTime(at), formatTime(deadline), int64(policies[0].Duration), policies[0].PolicyRef, formatTime(at))
	if err != nil {
		return err
	}
	_, err = appendAuditEventTx(ctx, tx, domain.AuditEvent{EventID: eventID, EventType: domain.AuditDispatchSendConfirmed, Timestamp: at, PairID: pairID, TaskID: taskID, ContractID: contractID, AttemptID: attemptID, Actor: actor, Details: map[string]any{"dispatch_operation_id": operationID, "stage": "SEND_CONFIRMED", "http_status": 200, "budget_origin_at": formatTime(at), "execution_deadline_at": formatTime(deadline), "duration_ns": int64(policies[0].Duration), "policy_ref": policies[0].PolicyRef}})
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

// GetExecutionBudget reads the immutable policy and checks its causal lineage.
// A missing or malformed budget never falls back to the current host policy.
func (s *Store) GetExecutionBudget(ctx context.Context, attemptID string) (domain.ExecutionBudget, error) {
	var b domain.ExecutionBudget
	var origin, deadline, bound string
	var duration int64
	var principal, evidence sql.NullString
	var dispatchStage, confirmedAt, taskState, snapshotSession, snapshotGeneration, opSession, opGeneration string
	var open int
	err := s.db.QueryRowContext(ctx, `SELECT b.attempt_id,b.dispatch_operation_id,b.origin_at,b.deadline_at,b.duration_ns,b.policy_ref,b.binding_basis,b.authorized_principal,b.evidence_ref,b.bound_at,d.stage,d.confirmed_at,t.state,a.session_id,a.terminal_generation,d.session_id,d.terminal_generation,(a.ended_at IS NULL AND t.current_attempt=a.attempt_number) FROM attempt_execution_budgets b JOIN dispatch_operations d ON d.operation_id=b.dispatch_operation_id AND d.attempt_id=b.attempt_id JOIN task_attempts a ON a.attempt_id=b.attempt_id JOIN tasks t ON t.task_id=a.task_id AND t.task_id=d.task_id AND t.pair_id=d.pair_id WHERE b.attempt_id=?`, attemptID).Scan(&b.AttemptID, &b.DispatchOperationID, &origin, &deadline, &duration, &b.PolicyRef, &b.BindingBasis, &principal, &evidence, &bound, &dispatchStage, &confirmedAt, &taskState, &snapshotSession, &snapshotGeneration, &opSession, &opGeneration, &open)
	if errors.Is(err, sql.ErrNoRows) {
		return b, ErrOperationNotFound
	}
	if err != nil {
		return b, err
	}
	b.OriginAt, err = parseTime(origin)
	if err != nil {
		return b, err
	}
	b.DeadlineAt, err = parseTime(deadline)
	if err != nil {
		return b, err
	}
	b.BoundAt, err = parseTime(bound)
	if err != nil {
		return b, err
	}
	b.Duration = time.Duration(duration)
	if principal.Valid {
		b.AuthorizedPrincipal = &principal.String
	}
	if evidence.Valid {
		b.EvidenceRef = &evidence.String
	}
	if dispatchStage != string(domain.SendConfirmed) || confirmedAt != origin || (taskState != string(domain.StateDispatched) && taskState != string(domain.StateRunning)) || open != 1 || snapshotSession != opSession || snapshotGeneration != opGeneration || duration <= 0 || strings.TrimSpace(b.PolicyRef) == "" || !b.OriginAt.Add(b.Duration).Equal(b.DeadlineAt) || !b.OriginAt.Before(b.DeadlineAt) || b.OriginAt.Year() < 1 || b.DeadlineAt.Year() > 9999 {
		return domain.ExecutionBudget{}, fmt.Errorf("%w: execution budget lineage or timestamp invalid", ErrStateConflict)
	}
	return b, nil
}

// BindLegacyExecutionBudget accepts a separately verified historical policy.
// The caller must hold trusted maintenance ownership and authenticate the
// principal; Store still enforces exact lineage, one-shot CAS and audit atomicity.
func (s *Store) BindLegacyExecutionBudget(ctx context.Context, attemptID, principal, scope, evidenceRef string, policy domain.ExecutionBudgetPolicy, at time.Time) error {
	if principal == "" || scope != "LEGACY_EXECUTION_BUDGET_RECONCILIATION" || strings.TrimSpace(evidenceRef) == "" || strings.TrimSpace(policy.PolicyRef) == "" || policy.Duration <= 0 || at.IsZero() {
		return fmt.Errorf("%w: verified historical binding inputs required", ErrStateConflict)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var pairID, taskID, contractID, operationID, sessionID, generation, confirmedAt, stage, state string
	var currentAttempt, attemptNumber int
	var ended sql.NullString
	err = tx.QueryRowContext(ctx, `SELECT t.pair_id,t.task_id,a.contract_id,d.operation_id,a.session_id,a.terminal_generation,d.confirmed_at,d.stage,t.state,t.current_attempt,a.attempt_number,a.ended_at FROM task_attempts a JOIN tasks t ON t.task_id=a.task_id JOIN dispatch_operations d ON d.attempt_id=a.attempt_id WHERE a.attempt_id=?`, attemptID).Scan(&pairID, &taskID, &contractID, &operationID, &sessionID, &generation, &confirmedAt, &stage, &state, &currentAttempt, &attemptNumber, &ended)
	if err != nil {
		return err
	}
	if ended.Valid || currentAttempt != attemptNumber || stage != string(domain.SendConfirmed) || (state != string(domain.StateDispatched) && state != string(domain.StateRunning)) {
		return ErrStateConflict
	}
	var currentSession, currentGeneration string
	if err = tx.QueryRowContext(ctx, `SELECT session_id,terminal_generation FROM worker_sessions WHERE pair_id=?`, pairID).Scan(&currentSession, &currentGeneration); err != nil {
		return err
	}
	if sessionID != currentSession || generation != currentGeneration {
		return ErrAttemptLineageMismatch
	}
	origin, err := parseTime(confirmedAt)
	if err != nil {
		return err
	}
	deadline := origin.Add(policy.Duration)
	if !origin.Before(deadline) || origin.Year() < 1 || deadline.Year() > 9999 {
		return ErrStateConflict
	}
	var count int
	if err = tx.QueryRowContext(ctx, `SELECT count(*) FROM attempt_execution_budgets WHERE attempt_id=? OR dispatch_operation_id=?`, attemptID, operationID).Scan(&count); err != nil {
		return err
	}
	if count != 0 {
		if count != 1 {
			return ErrStateConflict
		}
		var existingOrigin, existingDeadline, existingPolicy, existingBasis, existingPrincipal, existingEvidence, existingOperation string
		var existingDuration int64
		err = tx.QueryRowContext(ctx, `SELECT dispatch_operation_id,origin_at,deadline_at,duration_ns,policy_ref,binding_basis,authorized_principal,evidence_ref FROM attempt_execution_budgets WHERE attempt_id=?`, attemptID).Scan(&existingOperation, &existingOrigin, &existingDeadline, &existingDuration, &existingPolicy, &existingBasis, &existingPrincipal, &existingEvidence)
		if err != nil {
			return ErrStateConflict
		}
		if existingOperation != operationID || existingOrigin != confirmedAt || existingDeadline != formatTime(deadline) || existingDuration != int64(policy.Duration) || existingPolicy != policy.PolicyRef || existingBasis != "LEGACY_OPERATOR_VERIFIED" || existingPrincipal != principal || existingEvidence != evidenceRef {
			return ErrStateConflict
		}
		var auditCount int
		if err = tx.QueryRowContext(ctx, `SELECT count(*) FROM audit_events WHERE event_type=? AND attempt_id=? AND actor=? AND json_extract(details_json,'$.evidence_ref')=? AND json_extract(details_json,'$.dispatch_operation_id')=?`, domain.AuditExecutionBudgetLegacyBound, attemptID, principal, evidenceRef, operationID).Scan(&auditCount); err != nil {
			return err
		}
		if auditCount != 1 {
			return ErrStateConflict
		}
		return nil // exact replay is read-only
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO attempt_execution_budgets(attempt_id,dispatch_operation_id,origin_at,deadline_at,duration_ns,policy_ref,binding_basis,authorized_principal,evidence_ref,bound_at) VALUES(?,?,?,?,?,?,'LEGACY_OPERATOR_VERIFIED',?,?,?)`, attemptID, operationID, confirmedAt, formatTime(deadline), int64(policy.Duration), policy.PolicyRef, principal, evidenceRef, formatTime(at)); err != nil {
		return err
	}
	id, err := newAuditEventID()
	if err != nil {
		return err
	}
	_, err = appendAuditEventTx(ctx, tx, domain.AuditEvent{EventID: id, EventType: domain.AuditExecutionBudgetLegacyBound, Timestamp: at, PairID: pairID, TaskID: taskID, ContractID: contractID, AttemptID: attemptID, Actor: principal, Details: map[string]any{"dispatch_operation_id": operationID, "session_id": sessionID, "terminal_generation": generation, "origin_at": confirmedAt, "deadline_at": formatTime(deadline), "duration_ns": int64(policy.Duration), "policy_ref": policy.PolicyRef, "binding_basis": "LEGACY_OPERATOR_VERIFIED", "authorized_principal": principal, "authority_scope": scope, "evidence_ref": evidenceRef}})
	if err != nil {
		return err
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
