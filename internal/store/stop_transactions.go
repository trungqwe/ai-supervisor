package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/trungqwe/ai-supervisor/internal/domain"
)

// ReserveStopOperation commits a new unlinked stop intent and its audit event.
// A successful return belongs only to this invocation; reading the row later
// never confers permission to call the session-scoped /kill endpoint.
func (s *Store) ReserveStopOperation(ctx context.Context, stop domain.StopOperation) error {
	if stop.RestoreOperationID != nil || stop.RestorePrincipal != nil || stop.OperationID == "" || stop.PairID == "" || stop.SessionID == "" || stop.TerminalGeneration == "" || stop.Actor == "" {
		return fmt.Errorf("%w: invalid unlinked stop reservation", ErrAttemptLineageMismatch)
	}
	if stop.Stage == "" {
		stop.Stage = domain.StopRequested
	}
	if stop.ResolutionState == "" {
		stop.ResolutionState = domain.StopResolutionInFlight
	}
	if stop.Stage != domain.StopRequested || stop.ResolutionState != domain.StopResolutionInFlight {
		return fmt.Errorf("%w: stop must begin STOP_REQUESTED/IN_FLIGHT", ErrInvalidOperationTransition)
	}
	if stop.RequestedAt.IsZero() {
		stop.RequestedAt = timeNow()
	}
	if stop.InitiatingFailureReason != nil && (*stop.InitiatingFailureReason != "TIMEOUT" || stop.Purpose != domain.RunningAttemptStop) {
		return ErrStateConflict
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var session, generation string
	if err = tx.QueryRowContext(ctx, `SELECT session_id,terminal_generation FROM worker_sessions WHERE pair_id=?`, stop.PairID).Scan(&session, &generation); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrAttemptLineageMismatch
		}
		return err
	}
	if session != stop.SessionID || generation != stop.TerminalGeneration {
		return fmt.Errorf("%w: stop runtime tuple mismatch", ErrAttemptLineageMismatch)
	}
	var pending int
	if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM stop_operations WHERE pair_id=? AND resolution_state IN ('IN_FLIGHT','STOP_CALL_OUTCOME_UNKNOWN','STOP_CONFIRMATION_TIMEOUT','STOP_REISSUE_REQUIRES_HUMAN'))`, stop.PairID).Scan(&pending); err != nil {
		return err
	}
	if pending != 0 {
		return fmt.Errorf("%w: Pair already has unresolved stop", ErrStateConflict)
	}
	if stop.Purpose == domain.RunningAttemptStop {
		if stop.AttemptID == nil || stop.TaskID == nil || stop.ContractID == nil {
			return ErrAttemptLineageMismatch
		}
		var state string
		if err = tx.QueryRowContext(ctx, `SELECT t.state FROM tasks t JOIN task_attempts a ON a.task_id=t.task_id AND a.attempt_number=t.current_attempt WHERE t.task_id=? AND t.pair_id=? AND a.attempt_id=? AND a.contract_id=? AND a.ended_at IS NULL`, *stop.TaskID, stop.PairID, *stop.AttemptID, *stop.ContractID).Scan(&state); err != nil {
			return fmt.Errorf("%w: live stop requires open current attempt: %v", ErrAttemptLineageMismatch, err)
		}
		if state != string(domain.StateRunning) {
			return fmt.Errorf("%w: live stop requires RUNNING", ErrStateConflict)
		}
		if stop.InitiatingFailureReason != nil {
			var origin, deadline, dispatchSession, dispatchGeneration, stage, policyRef string
			var duration int64
			var guard int
			err = tx.QueryRowContext(ctx, `SELECT b.origin_at,b.deadline_at,b.duration_ns,b.policy_ref,d.session_id,d.terminal_generation,d.stage, (a.quarantine_state='CLEAN' AND w.quarantine_state='CLEAN' AND a.recovery_disposition IS NULL AND w.session_id=? AND w.terminal_generation=?) FROM attempt_execution_budgets b JOIN dispatch_operations d ON d.operation_id=b.dispatch_operation_id AND d.attempt_id=b.attempt_id JOIN task_attempts a ON a.attempt_id=b.attempt_id JOIN worker_sessions w ON w.pair_id=d.pair_id WHERE b.attempt_id=? AND d.pair_id=? AND d.task_id=?`, stop.SessionID, stop.TerminalGeneration, *stop.AttemptID, stop.PairID, *stop.TaskID).Scan(&origin, &deadline, &duration, &policyRef, &dispatchSession, &dispatchGeneration, &stage, &guard)
			if err != nil {
				return fmt.Errorf("%w: immutable timeout budget absent: %v", ErrStateConflict, err)
			}
			originAt, e := parseTime(origin)
			if e != nil {
				return e
			}
			deadlineAt, e := parseTime(deadline)
			if e != nil {
				return e
			}
			if duration <= 0 || policyRef == "" || !originAt.Add(time.Duration(duration)).Equal(deadlineAt) || stage != string(domain.SendConfirmed) || guard != 1 || dispatchSession != stop.SessionID || dispatchGeneration != stop.TerminalGeneration || stop.RequestedAt.Before(deadlineAt) {
				return ErrStateConflict
			}
		}
	} else if stop.Purpose == domain.QuarantineCleanup {
		if stop.AttemptID == nil || stop.TaskID == nil || stop.ContractID == nil {
			return ErrAttemptLineageMismatch
		}
		var state string
		if err = tx.QueryRowContext(ctx, `SELECT t.state FROM tasks t JOIN task_attempts a ON a.task_id=t.task_id WHERE t.task_id=? AND t.pair_id=? AND a.attempt_id=? AND a.contract_id=? AND a.ended_at IS NOT NULL AND a.quarantine_state='QUARANTINED'`, *stop.TaskID, stop.PairID, *stop.AttemptID, *stop.ContractID).Scan(&state); err != nil {
			return fmt.Errorf("%w: cleanup requires closed quarantined attempt: %v", ErrAttemptLineageMismatch, err)
		}
		if state != string(domain.StateFailed) && state != string(domain.StateHumanRequired) {
			return ErrAttemptLineageMismatch
		}
	} else if stop.Purpose != domain.PairMaintenance || stop.TaskID != nil || stop.ContractID != nil || stop.AttemptID != nil {
		return ErrAttemptLineageMismatch
	}
	if err = createStopOperationTx(ctx, tx, stop); err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, `UPDATE worker_sessions SET quarantine_state='QUARANTINED',updated_at=? WHERE pair_id=? AND session_id=? AND terminal_generation=?`, formatTime(stop.RequestedAt), stop.PairID, stop.SessionID, stop.TerminalGeneration)
	if err = operationCASResult(res, err, stop.OperationID, "stop reservation session"); err != nil {
		return err
	}
	if stop.Purpose == domain.RunningAttemptStop {
		res, err = tx.ExecContext(ctx, `UPDATE task_attempts SET quarantine_state='QUARANTINED' WHERE attempt_id=? AND task_id=? AND contract_id=? AND session_id=? AND terminal_generation=? AND ended_at IS NULL`, *stop.AttemptID, *stop.TaskID, *stop.ContractID, stop.SessionID, stop.TerminalGeneration)
		if err = operationCASResult(res, err, stop.OperationID, "stop reservation attempt"); err != nil {
			return err
		}
	}
	if err = appendStopAudit(ctx, tx, stop, domain.AuditStopOperationRequested, stop.RequestedAt, nil); err != nil {
		return err
	}
	return tx.Commit()
}

// ValidateTimeoutEffectOwner accepts only the double quarantine established by
// this winner's reservation. Any new Pair risk or changed lineage is rejected.
// It is a Supervisor precheck, not an AO generation fence.
func (s *Store) ValidateTimeoutEffectOwner(ctx context.Context, operationID string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stop, err := getStopOperationTx(ctx, tx, operationID)
	if err != nil {
		return err
	}
	if stop.InitiatingFailureReason == nil || *stop.InitiatingFailureReason != "TIMEOUT" || stop.Purpose != domain.RunningAttemptStop || stop.Stage != domain.StopRequested || stop.ResolutionState != domain.StopResolutionInFlight || stop.TaskID == nil || stop.ContractID == nil || stop.AttemptID == nil {
		return ErrStateConflict
	}
	var valid int
	err = tx.QueryRowContext(ctx, `SELECT EXISTS(
 SELECT 1 FROM tasks t JOIN task_attempts a ON a.task_id=t.task_id AND a.attempt_number=t.current_attempt
 JOIN worker_sessions w ON w.pair_id=t.pair_id
 JOIN dispatch_operations d ON d.attempt_id=a.attempt_id
 WHERE t.pair_id=? AND t.task_id=? AND t.state='RUNNING' AND a.attempt_id=? AND a.contract_id=?
 AND a.ended_at IS NULL AND a.recovery_disposition IS NULL AND a.quarantine_state='QUARANTINED'
 AND a.session_id=? AND a.terminal_generation=? AND w.session_id=? AND w.terminal_generation=?
 AND w.quarantine_state='QUARANTINED' AND d.stage='SEND_CONFIRMED' AND d.resolution_state IS NULL)
 AND NOT EXISTS(SELECT 1 FROM stop_operations WHERE pair_id=? AND operation_id<>? AND resolution_state IN ('IN_FLIGHT','STOP_CALL_OUTCOME_UNKNOWN','STOP_CONFIRMATION_TIMEOUT','STOP_REISSUE_REQUIRES_HUMAN'))
 AND NOT EXISTS(SELECT 1 FROM pair_restore_operations WHERE pair_id=? AND resolution_state<>'RESTORE_RESOLVED')
 AND NOT EXISTS(SELECT 1 FROM pair_provisioning_operations WHERE pair_id=? AND stage IN ('PROVISION_REQUESTED','PROVISION_FAILED'))
 AND NOT EXISTS(SELECT 1 FROM dispatch_operations WHERE pair_id=? AND resolution_state='DELIVERY_OUTCOME_UNKNOWN')
 AND NOT EXISTS(SELECT 1 FROM task_attempts a JOIN tasks t ON t.task_id=a.task_id WHERE t.pair_id=? AND a.attempt_id<>? AND a.quarantine_state='QUARANTINED')`, stop.PairID, *stop.TaskID, *stop.AttemptID, *stop.ContractID, stop.SessionID, stop.TerminalGeneration, stop.SessionID, stop.TerminalGeneration, stop.PairID, operationID, stop.PairID, stop.PairID, stop.PairID, stop.PairID, *stop.AttemptID).Scan(&valid)
	if err != nil {
		return err
	}
	if valid != 1 {
		return ErrQuarantinedExecution
	}
	return tx.Commit()
}

func appendStopAudit(ctx context.Context, tx *sql.Tx, stop domain.StopOperation, kind string, at time.Time, extra map[string]any) error {
	id, err := newAuditEventID()
	if err != nil {
		return err
	}
	details := map[string]any{"stop_operation_id": stop.OperationID, "purpose": string(stop.Purpose), "session_id": stop.SessionID, "target_generation": stop.TerminalGeneration}
	if stop.InitiatingFailureReason != nil {
		details["initiating_failure_reason"] = *stop.InitiatingFailureReason
	}
	if stop.RestoreOperationID != nil {
		details["restore_operation_id"] = *stop.RestoreOperationID
	} else {
		details["restore_operation_id"] = nil
	}
	for k, v := range extra {
		details[k] = v
	}
	_, err = appendAuditEventTx(ctx, tx, domain.AuditEvent{EventID: id, EventType: kind, Timestamp: at, PairID: stop.PairID, TaskID: valueOrEmpty(stop.TaskID), ContractID: valueOrEmpty(stop.ContractID), AttemptID: valueOrEmpty(stop.AttemptID), Actor: stop.Actor, Details: details})
	return err
}

// CommitStopCallAccepted persists HTTP 200 provenance and the injected deadline.
// The deadline can never be reset by replay or a later observation.
func (s *Store) CommitStopCallAccepted(ctx context.Context, operationID string, callAt, deadline time.Time) error {
	if callAt.IsZero() || deadline.IsZero() || !callAt.Before(deadline) {
		return fmt.Errorf("%w: injected stop deadline is required", ErrStateConflict)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stop, err := getStopOperationTx(ctx, tx, operationID)
	if err != nil {
		return err
	}
	if err := guardStopRestoreOwner(ctx, tx, stop); err != nil {
		return err
	}
	if stop.Stage != domain.StopRequested || stop.ResolutionState != domain.StopResolutionInFlight {
		return ErrStateConflict
	}
	res, err := tx.ExecContext(ctx, `UPDATE stop_operations SET stage='STOP_CALL_SUCCEEDED',call_completed_at=?,confirmation_deadline_at=? WHERE operation_id=? AND stage='STOP_REQUESTED' AND resolution_state='IN_FLIGHT' AND call_completed_at IS NULL AND confirmation_deadline_at IS NULL`, formatTime(callAt), formatTime(deadline), operationID)
	if err = operationCASResult(res, err, operationID, "stop"); err != nil {
		return err
	}
	if err = appendStopAudit(ctx, tx, stop, domain.AuditStopOperationCallSucceeded, callAt, map[string]any{"call_completed_at": formatTime(callAt), "confirmation_deadline_at": formatTime(deadline)}); err != nil {
		return err
	}
	return tx.Commit()
}

// StopTerminalOutcome is an observation made outside the SQLite transaction.
// The caller chooses an approved D11 result; Store validates its durable basis.
type StopTerminalOutcome struct {
	ExpectedStage        domain.StopStage
	Stage                domain.StopStage
	Resolution           domain.StopResolutionState
	At                   time.Time
	ObservedSessionID    string
	ObservedGeneration   string
	ObservedIsTerminated bool
}

// CommitStopOutcome couples every live-stop terminal result to D12 closure,
// quarantine and audit in one transaction. Linked cleanup only records stop
// evidence; restore Tx D and D6 remain separate.
func (s *Store) CommitStopOutcome(ctx context.Context, operationID string, outcome StopTerminalOutcome) error {
	if outcome.At.IsZero() {
		return fmt.Errorf("%w: stop outcome time is required", ErrStateConflict)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stop, err := getStopOperationTx(ctx, tx, operationID)
	if err != nil {
		return err
	}
	if err := guardStopRestoreOwner(ctx, tx, stop); err != nil {
		return err
	}
	if stop.Stage != outcome.ExpectedStage || stop.ResolutionState != domain.StopResolutionInFlight || !validStopTransition(outcome.ExpectedStage, outcome.Stage) {
		return ErrStateConflict
	}
	physical := outcome.Stage == domain.StopTerminationConfirmed && outcome.Resolution == domain.StopResolutionTerminationConfirmed
	if physical {
		if outcome.ExpectedStage != domain.StopCallSucceeded || stop.CallCompletedAt == nil || stop.ConfirmationDeadlineAt == nil || outcome.At.Before(*stop.CallCompletedAt) || !outcome.At.Before(*stop.ConfirmationDeadlineAt) || !outcome.ObservedIsTerminated || outcome.ObservedSessionID != stop.SessionID || outcome.ObservedGeneration != stop.TerminalGeneration {
			return fmt.Errorf("%w: D11 positive proof or strict deadline missing", ErrStateConflict)
		}
	} else if outcome.Stage == domain.StopTerminationConfirmed || outcome.Resolution == domain.StopResolutionTerminationConfirmed {
		return ErrStateConflict
	}
	logicalResolution := outcome.Resolution == domain.StopResolutionGenerationMismatch || outcome.Resolution == domain.StopResolutionEffectUnprovenAlreadyTerminated
	if logicalResolution {
		if outcome.ObservedSessionID != stop.SessionID || outcome.ObservedGeneration == "" ||
			(outcome.Resolution == domain.StopResolutionGenerationMismatch && outcome.ObservedGeneration == stop.TerminalGeneration) ||
			(outcome.Resolution == domain.StopResolutionEffectUnprovenAlreadyTerminated && (!outcome.ObservedIsTerminated || outcome.ObservedGeneration != stop.TerminalGeneration)) {
			return fmt.Errorf("%w: logical stop resolution requires exact observation", ErrStateConflict)
		}
	}
	if !validStopOutcome(outcome) {
		return fmt.Errorf("%w: invalid D11 stage/resolution", ErrInvalidOperationTransition)
	}
	res, err := tx.ExecContext(ctx, `UPDATE stop_operations SET stage=?,resolution_state=?,resolved_at=?,termination_confirmed_at=? WHERE operation_id=? AND stage=? AND resolution_state='IN_FLIGHT'`, string(outcome.Stage), string(outcome.Resolution), formatTime(outcome.At), func() any {
		if physical {
			return formatTime(outcome.At)
		}
		return nil
	}(), operationID, string(outcome.ExpectedStage))
	if err = operationCASResult(res, err, operationID, "stop"); err != nil {
		return err
	}
	event := stopOutcomeAuditType(outcome)
	if event != "" {
		extra := map[string]any{"resolution_state": string(outcome.Resolution)}
		if logicalResolution {
			extra["pair_id"] = stop.PairID
			extra["task_id"] = valueOrEmpty(stop.TaskID)
			extra["contract_id"] = valueOrEmpty(stop.ContractID)
			extra["attempt_id"] = valueOrEmpty(stop.AttemptID)
			extra["old_stage"] = string(stop.Stage)
			extra["new_stage"] = string(outcome.Stage)
			extra["old_resolution"] = string(stop.ResolutionState)
			extra["new_resolution"] = string(outcome.Resolution)
			extra["observed_session_id"] = outcome.ObservedSessionID
			extra["observed_generation"] = outcome.ObservedGeneration
			extra["is_terminated"] = outcome.ObservedIsTerminated
			extra["actor"] = stop.Actor
		}
		if physical {
			extra["confirmation_deadline_at"] = formatTime(*stop.ConfirmationDeadlineAt)
			extra["termination_confirmed_at"] = formatTime(outcome.At)
			extra["observed_session_id"] = outcome.ObservedSessionID
			extra["observed_generation"] = outcome.ObservedGeneration
			extra["is_terminated"] = true
		}
		if err = appendStopAudit(ctx, tx, stop, event, outcome.At, extra); err != nil {
			return err
		}
	}
	if stop.Purpose == domain.RunningAttemptStop {
		if err = closeLiveStopTx(ctx, tx, stop, outcome, physical); err != nil {
			return err
		}
	} else {
		// An old-generation linked cleanup is evidence for that attempt only. It
		// must not mutate the newly restored runtime or clear its quarantine.
		var runtimeGeneration string
		if err = tx.QueryRowContext(ctx, `SELECT terminal_generation FROM worker_sessions WHERE pair_id=? AND session_id=?`, stop.PairID, stop.SessionID).Scan(&runtimeGeneration); err != nil {
			return err
		}
		if runtimeGeneration == stop.TerminalGeneration {
			res, err = tx.ExecContext(ctx, `UPDATE worker_sessions SET status=CASE WHEN ? THEN 'TERMINATED' ELSE status END,quarantine_state='QUARANTINED',updated_at=? WHERE pair_id=? AND session_id=? AND terminal_generation=?`, physical, formatTime(outcome.At), stop.PairID, stop.SessionID, stop.TerminalGeneration)
			if err = operationCASResult(res, err, operationID, "stop runtime"); err != nil {
				return err
			}
		} else if stop.RestoreOperationID == nil || stop.Purpose != domain.QuarantineCleanup {
			return ErrAttemptLineageMismatch
		}
	}
	return tx.Commit()
}

func validStopOutcome(o StopTerminalOutcome) bool {
	switch o.Resolution {
	case domain.StopResolutionTerminationConfirmed:
		return o.ExpectedStage == domain.StopCallSucceeded && o.Stage == domain.StopTerminationConfirmed
	case domain.StopResolutionTargetAbsent:
		return o.Stage == domain.StopTargetAbsent
	case domain.StopResolutionCallFailed:
		return o.ExpectedStage == domain.StopRequested && o.Stage == domain.StopCallFailed
	case domain.StopResolutionConfirmationTimeout:
		return o.ExpectedStage == domain.StopCallSucceeded && o.Stage == domain.StopCallSucceeded
	case domain.StopResolutionGenerationMismatch:
		return o.Stage == o.ExpectedStage
	case domain.StopResolutionEffectUnprovenAlreadyTerminated, domain.StopResolutionCallOutcomeUnknown:
		return o.ExpectedStage == domain.StopRequested && o.Stage == domain.StopRequested
	default:
		return false
	}
}

func stopOutcomeAuditType(o StopTerminalOutcome) string {
	switch o.Resolution {
	case domain.StopResolutionTerminationConfirmed:
		return domain.AuditStopOperationConfirmed
	case domain.StopResolutionTargetAbsent:
		return domain.AuditStopOperationTargetAbsent
	case domain.StopResolutionCallFailed:
		return domain.AuditStopOperationCallFailed
	case domain.StopResolutionCallOutcomeUnknown:
		return domain.AuditStopOperationCallOutcomeUnknown
	case domain.StopResolutionConfirmationTimeout:
		return domain.AuditStopConfirmationTimeout
	case domain.StopResolutionGenerationMismatch, domain.StopResolutionEffectUnprovenAlreadyTerminated:
		return domain.AuditStopOperationResolved
	default:
		return ""
	}
}

func getStopOperationTx(ctx context.Context, tx *sql.Tx, id string) (domain.StopOperation, error) {
	var stop domain.StopOperation
	var purpose, stage, resolution, requested string
	var task, contract, attempt, restore, call, deadline, cause sql.NullString
	err := tx.QueryRowContext(ctx, `SELECT operation_id,purpose,pair_id,task_id,contract_id,attempt_id,session_id,terminal_generation,stage,actor,requested_at,call_completed_at,confirmation_deadline_at,resolution_state,restore_operation_id,initiating_failure_reason FROM stop_operations WHERE operation_id=?`, id).Scan(&stop.OperationID, &purpose, &stop.PairID, &task, &contract, &attempt, &stop.SessionID, &stop.TerminalGeneration, &stage, &stop.Actor, &requested, &call, &deadline, &resolution, &restore, &cause)
	if errors.Is(err, sql.ErrNoRows) {
		return stop, ErrOperationNotFound
	}
	if err != nil {
		return stop, err
	}
	stop.Purpose = domain.StopPurpose(purpose)
	stop.Stage = domain.StopStage(stage)
	stop.ResolutionState = domain.StopResolutionState(resolution)
	setNullableString(&stop.TaskID, task)
	setNullableString(&stop.ContractID, contract)
	setNullableString(&stop.AttemptID, attempt)
	setNullableString(&stop.RestoreOperationID, restore)
	setNullableString(&stop.InitiatingFailureReason, cause)
	stop.RequestedAt, err = parseTime(requested)
	if err != nil {
		return stop, err
	}
	stop.CallCompletedAt, err = parseNullableTime(call)
	if err != nil {
		return stop, err
	}
	stop.ConfirmationDeadlineAt, err = parseNullableTime(deadline)
	return stop, err
}

// ClassifyAbandonedStopAlive is startup-only: the caller must hold exclusive
// host quiescence, so no creator retains an in-stack effect permit. It never
// issues /kill and preserves the wire stage and call provenance.
func (s *Store) ClassifyAbandonedStopAlive(ctx context.Context, operationID, sessionID, generation, actor, invocation string, at time.Time) error {
	if operationID == "" || sessionID == "" || generation == "" || actor == "" || at.IsZero() {
		return ErrStateConflict
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stop, err := getStopOperationTx(ctx, tx, operationID)
	if err != nil {
		return err
	}
	if stop.SessionID != sessionID || stop.TerminalGeneration != generation {
		return ErrStateConflict
	}
	if stop.Stage == domain.StopRequested && stop.ResolutionState == domain.StopResolutionReissueRequiresHuman {
		return nil
	}
	if stop.Stage != domain.StopRequested || stop.ResolutionState != domain.StopResolutionInFlight || stop.SessionID != sessionID || stop.TerminalGeneration != generation {
		return ErrStateConflict
	}
	if err = guardStopRestoreOwner(ctx, tx, stop); err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, `UPDATE stop_operations SET resolution_state='STOP_REISSUE_REQUIRES_HUMAN',resolved_at=? WHERE operation_id=? AND stage='STOP_REQUESTED' AND resolution_state='IN_FLIGHT' AND purpose=? AND pair_id=? AND session_id=? AND terminal_generation=? AND COALESCE(task_id,'')=? AND COALESCE(contract_id,'')=? AND COALESCE(attempt_id,'')=? AND COALESCE(restore_operation_id,'')=?`, formatTime(at), operationID, string(stop.Purpose), stop.PairID, stop.SessionID, stop.TerminalGeneration, valueOrEmpty(stop.TaskID), valueOrEmpty(stop.ContractID), valueOrEmpty(stop.AttemptID), valueOrEmpty(stop.RestoreOperationID))
	if err = operationCASResult(res, err, operationID, "abandoned stop"); err != nil {
		return err
	}
	if err = appendStopAudit(ctx, tx, stop, domain.AuditStopOperationResolved, at, map[string]any{"old_stage": "STOP_REQUESTED", "new_stage": "STOP_REQUESTED", "old_resolution": "IN_FLIGHT", "new_resolution": "STOP_REISSUE_REQUIRES_HUMAN", "observed_session_id": sessionID, "observed_generation": generation, "is_terminated": false, "observation_source": "GetWorkerStatus", "observation_time": formatTime(at), "reason": "AUTOMATIC_REISSUE_PROHIBITED", "recovery_invocation": invocation, "actor": actor}); err != nil {
		return err
	}
	return tx.Commit()
}

func closeLiveStopTx(ctx context.Context, tx *sql.Tx, stop domain.StopOperation, outcome StopTerminalOutcome, physical bool) error {
	if stop.TaskID == nil || stop.ContractID == nil || stop.AttemptID == nil {
		return ErrAttemptLineageMismatch
	}
	var pair, contract string
	var err error
	if pair, contract, err = currentOpenAttemptLineage(ctx, tx, *stop.TaskID, *stop.AttemptID); err != nil {
		return err
	}
	if pair != stop.PairID || contract != *stop.ContractID {
		return ErrAttemptLineageMismatch
	}
	var session, generation sql.NullString
	if err = tx.QueryRowContext(ctx, `SELECT session_id,terminal_generation FROM task_attempts WHERE attempt_id=?`, *stop.AttemptID).Scan(&session, &generation); err != nil {
		return err
	}
	if !session.Valid || !generation.Valid || session.String != stop.SessionID || generation.String != stop.TerminalGeneration {
		return ErrAttemptLineageMismatch
	}
	disposition := stopDisposition(outcome, physical)
	res, err := tx.ExecContext(ctx, `UPDATE tasks SET state='FAILED',updated_at=? WHERE task_id=? AND pair_id=? AND state='RUNNING'`, formatTime(outcome.At), *stop.TaskID, stop.PairID)
	if err = operationCASResult(res, err, stop.OperationID, "live task"); err != nil {
		return err
	}
	quarantine := domain.QuarantineQuarantined
	if physical {
		quarantine = domain.QuarantineClean
	}
	res, err = tx.ExecContext(ctx, `UPDATE task_attempts SET ended_at=?,recovery_disposition=?,quarantine_state=? WHERE attempt_id=? AND task_id=? AND contract_id=? AND session_id=? AND terminal_generation=? AND ended_at IS NULL AND attempt_number=(SELECT current_attempt FROM tasks WHERE task_id=?)`, formatTime(outcome.At), disposition, string(quarantine), *stop.AttemptID, *stop.TaskID, *stop.ContractID, stop.SessionID, stop.TerminalGeneration, *stop.TaskID)
	if err = operationCASResult(res, err, stop.OperationID, "live attempt"); err != nil {
		return err
	}
	var otherRisk int
	if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM task_attempts a JOIN tasks t ON t.task_id=a.task_id WHERE t.pair_id=? AND a.attempt_id<>? AND a.quarantine_state='QUARANTINED') OR EXISTS(SELECT 1 FROM pair_restore_operations WHERE pair_id=? AND resolution_state<>'RESTORE_RESOLVED') OR EXISTS(SELECT 1 FROM dispatch_operations WHERE pair_id=? AND resolution_state='DELIVERY_OUTCOME_UNKNOWN')`, stop.PairID, *stop.AttemptID, stop.PairID, stop.PairID).Scan(&otherRisk); err != nil {
		return err
	}
	workerQuarantine := domain.QuarantineQuarantined
	if physical && otherRisk == 0 {
		workerQuarantine = domain.QuarantineClean
	}
	res, err = tx.ExecContext(ctx, `UPDATE worker_sessions SET status=CASE WHEN ? THEN 'TERMINATED' ELSE status END,quarantine_state=?,updated_at=? WHERE pair_id=? AND session_id=? AND terminal_generation=?`, physical, string(workerQuarantine), formatTime(outcome.At), stop.PairID, stop.SessionID, stop.TerminalGeneration)
	if err = operationCASResult(res, err, stop.OperationID, "live session"); err != nil {
		return err
	}
	id, err := newAuditEventID()
	if err != nil {
		return err
	}
	failureReason := disposition
	if stop.InitiatingFailureReason != nil {
		failureReason = *stop.InitiatingFailureReason
	}
	_, err = appendAuditEventTx(ctx, tx, domain.AuditEvent{EventID: id, EventType: domain.AuditTaskStateTransition, Timestamp: outcome.At, PairID: stop.PairID, TaskID: *stop.TaskID, ContractID: *stop.ContractID, AttemptID: *stop.AttemptID, Actor: stop.Actor, Details: map[string]any{"from_state": "RUNNING", "to_state": "FAILED", "failure_reason": failureReason, "recovery_disposition": disposition, "stop_operation_id": stop.OperationID}})
	if err != nil {
		return err
	}
	if physical {
		if err = appendStopAudit(ctx, tx, stop, domain.AuditQuarantineResolvedPhysical, outcome.At, map[string]any{"lineage": "attempt", "attempt_id": *stop.AttemptID}); err != nil {
			return err
		}
		if workerQuarantine == domain.QuarantineClean {
			if err = appendStopAudit(ctx, tx, stop, domain.AuditQuarantineResolvedPhysical, outcome.At, map[string]any{"lineage": "worker_session"}); err != nil {
				return err
			}
		}
	}
	return nil
}

func stopDisposition(o StopTerminalOutcome, physical bool) string {
	if physical {
		return "WORKER_STOPPED"
	}
	switch o.Resolution {
	case domain.StopResolutionTargetAbsent:
		return "SESSION_ABSENT"
	case domain.StopResolutionConfirmationTimeout:
		return "STOP_CONFIRMATION_TIMEOUT"
	case domain.StopResolutionGenerationMismatch:
		return "STOP_GENERATION_MISMATCH"
	case domain.StopResolutionCallOutcomeUnknown:
		return "STOP_CALL_OUTCOME_UNKNOWN"
	default:
		return "WORKER_TERMINATION_UNKNOWN"
	}
}

func guardStopRestoreOwner(ctx context.Context, tx *sql.Tx, stop domain.StopOperation) error {
	var id, state string
	err := tx.QueryRowContext(ctx, `SELECT operation_id,resolution_state FROM pair_restore_operations WHERE pair_id=? AND resolution_state<>'RESTORE_RESOLVED'`, stop.PairID).Scan(&id, &state)
	if errors.Is(err, sql.ErrNoRows) {
		if stop.RestoreOperationID != nil {
			return ErrStateConflict
		}
		return nil
	}
	if err != nil {
		return err
	}
	if stop.RestoreOperationID == nil || *stop.RestoreOperationID != id || state != string(domain.RestoreCleanupClaimed) {
		return ErrQuarantinedExecution
	}
	return nil
}

// AdministrativeStopDecision is a one-shot, exact-lineage operator decision.
// The trusted host boundary authenticates Principal before invoking this API.
type AdministrativeStopDecision struct {
	StopOperationID, PairID, SessionID, TerminalGeneration string
	Purpose                                                domain.StopPurpose
	AttemptID, RestoreOperationID                          string
	ExpectedStage                                          domain.StopStage
	ExpectedResolution                                     domain.StopResolutionState
	Principal, AuthorityScope, Reason                      string
	EvidenceReferences                                     []string
	Lineages                                               []LineageClearance
	At                                                     time.Time
}

func (s *Store) AcceptStopAdministrativeRisk(ctx context.Context, d AdministrativeStopDecision) error {
	if d.StopOperationID == "" || d.PairID == "" || d.SessionID == "" || d.TerminalGeneration == "" || d.Principal == "" || d.AuthorityScope != "STOP_ADMINISTRATIVE_RECONCILIATION" || d.Reason == "" || len(d.EvidenceReferences) == 0 || len(d.Lineages) == 0 || d.At.IsZero() {
		return fmt.Errorf("%w: verified, scoped stop decision required", ErrStateConflict)
	}
	for _, ref := range d.EvidenceReferences {
		if ref == "" {
			return ErrStateConflict
		}
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stop, err := getStopOperationTx(ctx, tx, d.StopOperationID)
	if err != nil {
		return err
	}
	if stop.PairID != d.PairID || stop.SessionID != d.SessionID || stop.TerminalGeneration != d.TerminalGeneration || stop.Purpose != d.Purpose || valueOrEmpty(stop.AttemptID) != d.AttemptID || valueOrEmpty(stop.RestoreOperationID) != d.RestoreOperationID || stop.Stage != d.ExpectedStage {
		return ErrAttemptLineageMismatch
	}
	class := "CLASS_C"
	if d.ExpectedResolution == domain.StopResolutionTargetAbsent && d.ExpectedStage == domain.StopTargetAbsent {
		class = "CLASS_B"
	}
	if class == "CLASS_C" && (d.ExpectedResolution == domain.StopResolutionInFlight || d.ExpectedResolution == domain.StopResolutionTerminationConfirmed || d.ExpectedResolution == domain.StopResolutionAdministrativeRiskAccepted || d.ExpectedResolution == domain.StopResolutionTargetAbsent) {
		return ErrStateConflict
	}
	prior, err := stopAcceptanceEventsTx(ctx, tx, d.StopOperationID)
	if err != nil {
		return err
	}
	if len(prior) != 0 {
		if len(prior) != len(d.Lineages) {
			return ErrStateConflict
		}
		seenReplay := map[string]bool{}
		for _, l := range d.Lineages {
			if seenReplay[l.AttemptID] {
				return ErrStateConflict
			}
			seenReplay[l.AttemptID] = true
			v, ok := prior[l.AttemptID]
			if !ok || v.Principal != d.Principal || v.Scope != d.AuthorityScope || v.Reason != d.Reason || v.Class != class || v.SessionID != l.SessionID || v.Generation != l.TerminalGeneration || v.RestoreID != d.RestoreOperationID || v.OldResolution != string(d.ExpectedResolution) || !sameStrings(v.Evidence, d.EvidenceReferences) {
				return ErrStateConflict
			}
		}
		if class == "CLASS_C" && stop.ResolutionState != domain.StopResolutionAdministrativeRiskAccepted {
			return ErrStateConflict
		}
		if class == "CLASS_B" && stop.ResolutionState != domain.StopResolutionTargetAbsent {
			return ErrStateConflict
		}
		return nil
	}
	if err = guardStopRestoreOwner(ctx, tx, stop); err != nil {
		return err
	}
	if stop.RestoreOperationID != nil {
		var owner, state string
		if err = tx.QueryRowContext(ctx, `SELECT recovery_principal,resolution_state FROM pair_restore_operations WHERE operation_id=? AND pair_id=?`, *stop.RestoreOperationID, d.PairID).Scan(&owner, &state); err != nil {
			return err
		}
		if owner != d.Principal || state != string(domain.RestoreCleanupClaimed) {
			return ErrStateConflict
		}
	}
	if class == "CLASS_B" && (stop.ResolutionState != domain.StopResolutionTargetAbsent || d.ExpectedResolution != stop.ResolutionState) {
		return ErrStateConflict
	}
	if class == "CLASS_B" {
		var absenceAudit int
		if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM audit_events WHERE event_type=? AND pair_id=? AND json_extract(details_json,'$.stop_operation_id')=?)`, domain.AuditStopOperationTargetAbsent, d.PairID, d.StopOperationID).Scan(&absenceAudit); err != nil {
			return err
		}
		if absenceAudit != 1 {
			return fmt.Errorf("%w: Class B lacks audited HTTP 404", ErrStateConflict)
		}
	}
	seen := map[string]bool{}
	for _, l := range d.Lineages {
		key := l.AttemptID
		if key == "" {
			key = "@session"
		}
		if seen[key] || l.SessionID == "" || l.TerminalGeneration == "" || l.SessionID != d.SessionID {
			return ErrAttemptLineageMismatch
		}
		seen[key] = true
		// HTTP 404 is session absence evidence. A linked attempt additionally
		// needs its own acceptance; neither acceptance implies the other.
		if class == "CLASS_B" && (l.SessionID != stop.SessionID || l.TerminalGeneration != stop.TerminalGeneration || l.AttemptID != "" && l.AttemptID != valueOrEmpty(stop.AttemptID)) {
			return ErrAttemptLineageMismatch
		}
		if l.AttemptID == "" {
			var sid, gen, q string
			if err = tx.QueryRowContext(ctx, `SELECT session_id,terminal_generation,quarantine_state FROM worker_sessions WHERE pair_id=?`, d.PairID).Scan(&sid, &gen, &q); err != nil {
				return err
			}
			if sid != l.SessionID || gen != l.TerminalGeneration || q != string(domain.QuarantineQuarantined) {
				return ErrAttemptLineageMismatch
			}
		} else {
			var sid, gen, q string
			if err = tx.QueryRowContext(ctx, `SELECT a.session_id,a.terminal_generation,a.quarantine_state FROM task_attempts a JOIN tasks t ON t.task_id=a.task_id WHERE a.attempt_id=? AND t.pair_id=?`, l.AttemptID, d.PairID).Scan(&sid, &gen, &q); err != nil {
				return err
			}
			if sid != l.SessionID || gen != l.TerminalGeneration || q != string(domain.QuarantineQuarantined) {
				return ErrAttemptLineageMismatch
			}
		}
	}
	if stop.ResolutionState != d.ExpectedResolution {
		return ErrStateConflict
	}
	newResolution := d.ExpectedResolution
	if class == "CLASS_C" {
		res, e := tx.ExecContext(ctx, `UPDATE stop_operations SET resolution_state='ADMINISTRATIVE_RISK_ACCEPTED',resolved_at=? WHERE operation_id=? AND pair_id=? AND session_id=? AND terminal_generation=? AND purpose=? AND stage=? AND resolution_state=?`, formatTime(d.At), d.StopOperationID, d.PairID, d.SessionID, d.TerminalGeneration, string(d.Purpose), string(d.ExpectedStage), string(d.ExpectedResolution))
		if e = operationCASResult(res, e, d.StopOperationID, "administrative stop resolution"); e != nil {
			return e
		}
		newResolution = domain.StopResolutionAdministrativeRiskAccepted
	} else {
		res, e := tx.ExecContext(ctx, `UPDATE stop_operations SET resolution_state=resolution_state WHERE operation_id=? AND pair_id=? AND session_id=? AND terminal_generation=? AND purpose=? AND stage=? AND resolution_state='STOP_TARGET_ABSENT'`, d.StopOperationID, d.PairID, d.SessionID, d.TerminalGeneration, string(d.Purpose), string(d.ExpectedStage))
		if e = operationCASResult(res, e, d.StopOperationID, "Class B stop absence CAS"); e != nil {
			return e
		}
	}
	for _, l := range d.Lineages {
		id, e := newAuditEventID()
		if e != nil {
			return e
		}
		var taskID, contractID string
		if l.AttemptID != "" {
			if e = tx.QueryRowContext(ctx, `SELECT task_id,contract_id FROM task_attempts WHERE attempt_id=?`, l.AttemptID).Scan(&taskID, &contractID); e != nil {
				return e
			}
		}
		details := map[string]any{"stop_operation_id": d.StopOperationID, "purpose": string(d.Purpose), "session_id": l.SessionID, "terminal_generation": l.TerminalGeneration, "attempt_id": l.AttemptID, "principal": d.Principal, "authority_scope": d.AuthorityScope, "reason": d.Reason, "evidence_references": d.EvidenceReferences, "old_resolution": string(d.ExpectedResolution), "new_resolution": string(newResolution), "restore_operation_id": d.RestoreOperationID, "class": class}
		if _, e = appendAuditEventTx(ctx, tx, domain.AuditEvent{EventID: id, EventType: domain.AuditAdministrativeRiskAccepted, Timestamp: d.At, PairID: d.PairID, TaskID: taskID, ContractID: contractID, AttemptID: l.AttemptID, Actor: d.Principal, Details: details}); e != nil {
			return e
		}
	}
	return tx.Commit()
}

type stopAcceptance struct {
	Principal, Scope, Reason, Class, SessionID, Generation, RestoreID string
	OldResolution                                                     string
	Evidence                                                          []string
}

func stopAcceptanceEventsTx(ctx context.Context, tx *sql.Tx, stopID string) (map[string]stopAcceptance, error) {
	rows, err := tx.QueryContext(ctx, `SELECT details_json FROM audit_events WHERE event_type=? AND json_extract(details_json,'$.stop_operation_id')=?`, domain.AuditAdministrativeRiskAccepted, stopID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := map[string]stopAcceptance{}
	for rows.Next() {
		var raw string
		if err = rows.Scan(&raw); err != nil {
			return nil, err
		}
		var v struct {
			AttemptID     string   `json:"attempt_id"`
			Principal     string   `json:"principal"`
			Scope         string   `json:"authority_scope"`
			Reason        string   `json:"reason"`
			Class         string   `json:"class"`
			SessionID     string   `json:"session_id"`
			Generation    string   `json:"terminal_generation"`
			RestoreID     string   `json:"restore_operation_id"`
			OldResolution string   `json:"old_resolution"`
			Evidence      []string `json:"evidence_references"`
		}
		if err = json.Unmarshal([]byte(raw), &v); err != nil {
			return nil, err
		}
		if _, exists := result[v.AttemptID]; exists {
			return nil, ErrStateConflict
		}
		result[v.AttemptID] = stopAcceptance{Principal: v.Principal, Scope: v.Scope, Reason: v.Reason, Class: v.Class, SessionID: v.SessionID, Generation: v.Generation, RestoreID: v.RestoreID, OldResolution: v.OldResolution, Evidence: v.Evidence}
	}
	return result, rows.Err()
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// A lineage's administrative acceptance must be committed before Tx D or D6.
func acceptedLineageTx(ctx context.Context, tx *sql.Tx, stopID, pairID, principal, session, generation, attempt, class, restoreID string) (bool, error) {
	var count int
	err := tx.QueryRowContext(ctx, `SELECT count(*) FROM audit_events WHERE event_type=? AND pair_id=? AND actor=? AND json_extract(details_json,'$.stop_operation_id')=? AND json_extract(details_json,'$.session_id')=? AND json_extract(details_json,'$.terminal_generation')=? AND json_extract(details_json,'$.attempt_id')=? AND json_extract(details_json,'$.class')=? AND json_extract(details_json,'$.restore_operation_id')=? AND json_extract(details_json,'$.authority_scope')='STOP_ADMINISTRATIVE_RECONCILIATION'`, domain.AuditAdministrativeRiskAccepted, pairID, principal, stopID, session, generation, attempt, class, restoreID).Scan(&count)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return count == 1, err
}
