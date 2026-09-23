package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/trungqwe/ai-supervisor/internal/domain"
)

// RecoveryExecution is an immutable observation target. Every subsequent
// mutation rechecks this tuple inside its own transaction.
type RecoveryExecution struct {
	OperationID, PairID, TaskID, ContractID, AttemptID string
	SessionID, Generation, TaskState, DispatchStage    string
	Disposition                                        sql.NullString
}

type RecoveryStop struct {
	OperationID, Stage, Purpose, SessionID, Generation string
}

type RecoverySnapshot struct {
	RestoreIDs, ProvisionIDs []string
	Executions               []RecoveryExecution
	StopOperations           []RecoveryStop
	Blocked                  []RecoveryExecution
}

// ListRecoverySnapshot reads one coherent candidate set without network calls.
// The runner must still use exact CAS on every candidate after its AO observation.
func (s *Store) ListRecoverySnapshot(ctx context.Context) (out RecoverySnapshot, err error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return out, err
	}
	defer tx.Rollback()
	var inconsistent int
	if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM tasks t LEFT JOIN task_attempts a ON a.task_id=t.task_id AND a.attempt_number=t.current_attempt LEFT JOIN dispatch_operations d ON d.attempt_id=a.attempt_id AND d.task_id=t.task_id AND d.pair_id=t.pair_id WHERE t.state IN ('DISPATCHED','RUNNING','BLOCKED') AND (a.attempt_id IS NULL OR a.ended_at IS NOT NULL OR a.session_id IS NULL OR a.terminal_generation IS NULL OR d.operation_id IS NULL OR (t.state IN ('RUNNING','BLOCKED') AND d.stage<>'SEND_CONFIRMED')))`).Scan(&inconsistent); err != nil {
		return out, err
	}
	if inconsistent != 0 {
		return out, ErrInconsistentPersistedState
	}
	collectIDs := func(query string) ([]string, error) {
		rows, e := tx.QueryContext(ctx, query)
		if e != nil {
			return nil, e
		}
		defer rows.Close()
		var ids []string
		for rows.Next() {
			var id string
			if e = rows.Scan(&id); e != nil {
				return nil, e
			}
			ids = append(ids, id)
		}
		return ids, rows.Err()
	}
	if out.RestoreIDs, err = collectIDs(`SELECT operation_id FROM pair_restore_operations WHERE resolution_state='IN_FLIGHT' AND stage IN ('RESTORE_REQUESTED','RESTORE_CONFIRMED') ORDER BY operation_id`); err != nil {
		return out, err
	}
	if out.ProvisionIDs, err = collectIDs(`SELECT operation_id FROM pair_provisioning_operations WHERE stage='PROVISION_REQUESTED' ORDER BY operation_id`); err != nil {
		return out, err
	}
	rows, e := tx.QueryContext(ctx, `SELECT d.operation_id,d.pair_id,d.task_id,a.contract_id,d.attempt_id,d.session_id,d.terminal_generation,t.state,d.stage,a.recovery_disposition FROM dispatch_operations d JOIN tasks t ON t.task_id=d.task_id JOIN task_attempts a ON a.attempt_id=d.attempt_id AND a.task_id=t.task_id AND a.attempt_number=t.current_attempt WHERE a.ended_at IS NULL AND ((t.state='DISPATCHED' AND (d.stage='DISPATCH_BOUND' OR (d.stage='SEND_REQUESTED' AND d.resolution_state IS NULL) OR d.stage='SEND_CONFIRMED')) OR (t.state='RUNNING' AND d.stage='SEND_CONFIRMED')) ORDER BY d.operation_id`)
	if e != nil {
		return out, e
	}
	for rows.Next() {
		var x RecoveryExecution
		if e = rows.Scan(&x.OperationID, &x.PairID, &x.TaskID, &x.ContractID, &x.AttemptID, &x.SessionID, &x.Generation, &x.TaskState, &x.DispatchStage, &x.Disposition); e != nil {
			rows.Close()
			return out, e
		}
		out.Executions = append(out.Executions, x)
	}
	if e = rows.Err(); e != nil {
		rows.Close()
		return out, e
	}
	rows.Close()
	rows, e = tx.QueryContext(ctx, `SELECT operation_id,stage,purpose,session_id,terminal_generation FROM stop_operations WHERE resolution_state='IN_FLIGHT' AND stage IN ('STOP_REQUESTED','STOP_CALL_SUCCEEDED') ORDER BY operation_id`)
	if e != nil {
		return out, e
	}
	for rows.Next() {
		var x RecoveryStop
		if e = rows.Scan(&x.OperationID, &x.Stage, &x.Purpose, &x.SessionID, &x.Generation); e != nil {
			rows.Close()
			return out, e
		}
		out.StopOperations = append(out.StopOperations, x)
	}
	if e = rows.Err(); e != nil {
		rows.Close()
		return out, e
	}
	rows.Close()
	rows, e = tx.QueryContext(ctx, `SELECT '',t.pair_id,t.task_id,a.contract_id,a.attempt_id,COALESCE(a.session_id,''),COALESCE(a.terminal_generation,''),t.state,'',a.recovery_disposition FROM tasks t JOIN task_attempts a ON a.task_id=t.task_id AND a.attempt_number=t.current_attempt WHERE t.state='BLOCKED' AND a.ended_at IS NULL ORDER BY t.task_id`)
	if e != nil {
		return out, e
	}
	for rows.Next() {
		var x RecoveryExecution
		if e = rows.Scan(&x.OperationID, &x.PairID, &x.TaskID, &x.ContractID, &x.AttemptID, &x.SessionID, &x.Generation, &x.TaskState, &x.DispatchStage, &x.Disposition); e != nil {
			rows.Close()
			return out, e
		}
		out.Blocked = append(out.Blocked, x)
	}
	if e = rows.Err(); e != nil {
		rows.Close()
		return out, e
	}
	rows.Close()
	return out, tx.Commit()
}

func (s *Store) RecordRecoverySweepEvent(ctx context.Context, invocationID, actor, event string, at time.Time) error {
	if invocationID == "" || actor == "" || (event != "STARTUP_RECOVERY_SWEEP_STARTED" && event != "STARTUP_RECOVERY_SWEEP_COMPLETED") {
		return ErrStateConflict
	}
	if at.IsZero() {
		at = timeNow()
	}
	id, err := newAuditEventID()
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = appendAuditEventTx(ctx, tx, domain.AuditEvent{EventID: id, EventType: event, Timestamp: at, Actor: actor, Details: map[string]any{"invocation_id": invocationID, "classification_only": true}})
	if err != nil {
		return err
	}
	return tx.Commit()
}

// RecordPostSendOutage records the approved NULL -> RECOVERY_PENDING edge.
// An existing hold is read-only and is never downgraded by an AO outage.
func (s *Store) RecordPostSendOutage(ctx context.Context, expected RecoveryExecution, actor, classification, invocation string, at time.Time) error {
	if expected.OperationID == "" || actor == "" || at.IsZero() {
		return ErrStateConflict
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var current RecoveryExecution
	err = tx.QueryRowContext(ctx, `SELECT d.operation_id,d.pair_id,d.task_id,a.contract_id,d.attempt_id,d.session_id,d.terminal_generation,t.state,d.stage,a.recovery_disposition FROM dispatch_operations d JOIN tasks t ON t.task_id=d.task_id JOIN task_attempts a ON a.attempt_id=d.attempt_id AND a.task_id=t.task_id AND a.attempt_number=t.current_attempt WHERE d.operation_id=? AND a.ended_at IS NULL`, expected.OperationID).Scan(&current.OperationID, &current.PairID, &current.TaskID, &current.ContractID, &current.AttemptID, &current.SessionID, &current.Generation, &current.TaskState, &current.DispatchStage, &current.Disposition)
	if err != nil {
		return err
	}
	if !sameRecoveryExecution(current, expected) || current.DispatchStage != "SEND_CONFIRMED" || (current.TaskState != "DISPATCHED" && current.TaskState != "RUNNING") {
		return ErrStateConflict
	}
	var runtimeSession, runtimeGeneration string
	if err = tx.QueryRowContext(ctx, `SELECT session_id,terminal_generation FROM worker_sessions WHERE pair_id=?`, current.PairID).Scan(&runtimeSession, &runtimeGeneration); err != nil {
		return err
	}
	if runtimeSession != current.SessionID || runtimeGeneration != current.Generation {
		return ErrAttemptLineageMismatch
	}
	if current.Disposition.Valid {
		if current.Disposition.String == "RECOVERY_PENDING" {
			return nil
		}
		return ErrStateConflict
	}
	res, err := tx.ExecContext(ctx, `UPDATE task_attempts SET recovery_disposition='RECOVERY_PENDING' WHERE attempt_id=? AND task_id=? AND contract_id=? AND session_id=? AND terminal_generation=? AND ended_at IS NULL AND recovery_disposition IS NULL AND attempt_number=(SELECT current_attempt FROM tasks WHERE task_id=?)`, current.AttemptID, current.TaskID, current.ContractID, current.SessionID, current.Generation, current.TaskID)
	if err = operationCASResult(res, err, current.OperationID, "post-send outage"); err != nil {
		return err
	}
	id, err := newAuditEventID()
	if err != nil {
		return err
	}
	_, err = appendAuditEventTx(ctx, tx, domain.AuditEvent{EventID: id, EventType: "POST_SEND_RECOVERY_PENDING", Timestamp: at, PairID: current.PairID, TaskID: current.TaskID, ContractID: current.ContractID, AttemptID: current.AttemptID, Actor: actor, Details: map[string]any{"dispatch_operation_id": current.OperationID, "dispatch_stage": current.DispatchStage, "session_id": current.SessionID, "generation": current.Generation, "old_task_state": current.TaskState, "new_task_state": current.TaskState, "old_disposition": nil, "new_disposition": "RECOVERY_PENDING", "observation_time": formatTime(at), "observation_source": "GetWorkerStatus", "error_classification": classification, "recovery_invocation": invocation}})
	if err != nil {
		return err
	}
	return tx.Commit()
}

func sameRecoveryExecution(a, b RecoveryExecution) bool {
	return a.OperationID == b.OperationID && a.PairID == b.PairID && a.TaskID == b.TaskID && a.ContractID == b.ContractID && a.AttemptID == b.AttemptID && a.SessionID == b.SessionID && a.Generation == b.Generation && a.TaskState == b.TaskState && a.DispatchStage == b.DispatchStage && a.Disposition == b.Disposition
}

type PostSendObservation struct {
	SessionID, Generation, Activity string
	Terminated, Absent              bool
}

// ObservePostSend commits an exact observation, recovery-hold resolution and
// any D12 terminal closure in one transaction. A fresh GET is the caller's
// responsibility and is always outside this transaction.
func (s *Store) ObservePostSend(ctx context.Context, expected RecoveryExecution, observed PostSendObservation, actor, invocation string, at time.Time) error {
	if actor == "" || at.IsZero() || expected.DispatchStage != "SEND_CONFIRMED" || observed.SessionID != expected.SessionID {
		return ErrStateConflict
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var current RecoveryExecution
	err = tx.QueryRowContext(ctx, `SELECT d.operation_id,d.pair_id,d.task_id,a.contract_id,d.attempt_id,d.session_id,d.terminal_generation,t.state,d.stage,a.recovery_disposition FROM dispatch_operations d JOIN tasks t ON t.task_id=d.task_id JOIN task_attempts a ON a.attempt_id=d.attempt_id AND a.task_id=t.task_id AND a.attempt_number=t.current_attempt WHERE d.operation_id=? AND a.ended_at IS NULL AND d.resolution_state IS NULL`, expected.OperationID).Scan(&current.OperationID, &current.PairID, &current.TaskID, &current.ContractID, &current.AttemptID, &current.SessionID, &current.Generation, &current.TaskState, &current.DispatchStage, &current.Disposition)
	if err != nil {
		return err
	}
	if !sameRecoveryExecution(current, expected) || (current.TaskState != "DISPATCHED" && current.TaskState != "RUNNING") {
		return ErrStateConflict
	}
	var runtimeSession, runtimeGeneration string
	if err = tx.QueryRowContext(ctx, `SELECT session_id,terminal_generation FROM worker_sessions WHERE pair_id=?`, current.PairID).Scan(&runtimeSession, &runtimeGeneration); err != nil {
		return err
	}
	if runtimeSession != current.SessionID || runtimeGeneration != current.Generation && observed.Generation == current.Generation {
		return ErrAttemptLineageMismatch
	}
	terminalDisposition := ""
	if observed.Absent {
		terminalDisposition = "SESSION_ABSENT"
	} else if observed.Generation != current.Generation {
		terminalDisposition = "STALE_EXECUTION_GENERATION"
	} else if observed.Terminated {
		terminalDisposition = "WORKER_TERMINATION_UNKNOWN"
	} else if current.TaskState == "DISPATCHED" && observed.Activity == "idle" {
		// This library has no P04 handoff. The approved matrix therefore takes
		// its fail-closed branch instead of claiming RUNNING or report readiness.
		terminalDisposition = "MISSED_ACTIVE_WINDOW"
	}
	newState := current.TaskState
	newDisposition := any(nil)
	if terminalDisposition != "" {
		newState = "FAILED"
		newDisposition = terminalDisposition
	} else if observed.Activity == "waiting_input" {
		newDisposition = "AO_WAITING_INPUT_OBSERVED"
	} else if observed.Activity == "blocked" {
		newDisposition = "AO_BLOCKED_DECISION"
	} else if observed.Activity != "active" && observed.Activity != "idle" && observed.Activity != "exited" {
		return ErrStateConflict
	}
	if current.Disposition.Valid && current.Disposition.String != "RECOVERY_PENDING" && terminalDisposition == "" {
		return nil
	}
	if terminalDisposition == "" && current.TaskState == "DISPATCHED" && (observed.Activity == "active" || observed.Activity == "waiting_input" || observed.Activity == "blocked") {
		newState = "RUNNING"
	}
	if current.Disposition.Valid && current.Disposition.String == "RECOVERY_PENDING" {
		id, e := newAuditEventID()
		if e != nil {
			return e
		}
		_, e = appendAuditEventTx(ctx, tx, domain.AuditEvent{EventID: id, EventType: "POST_SEND_STATUS_RECOVERED", Timestamp: at, PairID: current.PairID, TaskID: current.TaskID, ContractID: current.ContractID, AttemptID: current.AttemptID, Actor: actor, Details: map[string]any{"dispatch_operation_id": current.OperationID, "dispatch_stage": current.DispatchStage, "session_id": current.SessionID, "generation": current.Generation, "old_task_state": current.TaskState, "new_task_state": newState, "old_disposition": "RECOVERY_PENDING", "new_disposition": newDisposition, "observation_time": formatTime(at), "observation_source": "GetWorkerStatus", "observed_activity": observed.Activity, "recovery_invocation": invocation}})
		if e != nil {
			return e
		}
	}
	if newState != current.TaskState {
		res, e := tx.ExecContext(ctx, `UPDATE tasks SET state=?,updated_at=? WHERE task_id=? AND pair_id=? AND state=? AND current_attempt=(SELECT attempt_number FROM task_attempts WHERE attempt_id=?)`, newState, formatTime(at), current.TaskID, current.PairID, current.TaskState, current.AttemptID)
		if e = operationCASResult(res, e, current.OperationID, "post-send task"); e != nil {
			return e
		}
		id, e := newAuditEventID()
		if e != nil {
			return e
		}
		_, e = appendAuditEventTx(ctx, tx, domain.AuditEvent{EventID: id, EventType: domain.AuditTaskStateTransition, Timestamp: at, PairID: current.PairID, TaskID: current.TaskID, ContractID: current.ContractID, AttemptID: current.AttemptID, Actor: actor, Details: map[string]any{"from_state": current.TaskState, "to_state": newState, "recovery_disposition": newDisposition, "dispatch_operation_id": current.OperationID}})
		if e != nil {
			return e
		}
	}
	if terminalDisposition != "" {
		res, e := tx.ExecContext(ctx, `UPDATE task_attempts SET ended_at=?,recovery_disposition=?,quarantine_state=CASE WHEN ? IN ('SESSION_ABSENT','STALE_EXECUTION_GENERATION') THEN 'QUARANTINED' ELSE quarantine_state END WHERE attempt_id=? AND task_id=? AND contract_id=? AND session_id=? AND terminal_generation=? AND ended_at IS NULL AND recovery_disposition IS ?`, formatTime(at), terminalDisposition, terminalDisposition, current.AttemptID, current.TaskID, current.ContractID, current.SessionID, current.Generation, nullableString(nullStringPtr(current.Disposition)))
		if e = operationCASResult(res, e, current.OperationID, "post-send terminal attempt"); e != nil {
			return e
		}
		if terminalDisposition == "SESSION_ABSENT" || terminalDisposition == "STALE_EXECUTION_GENERATION" {
			res, e = tx.ExecContext(ctx, `UPDATE worker_sessions SET quarantine_state='QUARANTINED',updated_at=? WHERE pair_id=? AND session_id=? AND terminal_generation=?`, formatTime(at), current.PairID, current.SessionID, runtimeGeneration)
			if e = operationCASResult(res, e, current.OperationID, "post-send terminal session"); e != nil {
				return e
			}
		}
		if observed.Terminated && observed.Generation == current.Generation && !observed.Absent {
			res, e = tx.ExecContext(ctx, `UPDATE worker_sessions SET status='TERMINATED',updated_at=? WHERE pair_id=? AND session_id=? AND terminal_generation=?`, formatTime(at), current.PairID, current.SessionID, current.Generation)
			if e = operationCASResult(res, e, current.OperationID, "post-send terminated session"); e != nil {
				return e
			}
		}
	} else if current.Disposition.Valid && current.Disposition.String == "RECOVERY_PENDING" || newDisposition != nil {
		res, e := tx.ExecContext(ctx, `UPDATE task_attempts SET recovery_disposition=? WHERE attempt_id=? AND task_id=? AND contract_id=? AND session_id=? AND terminal_generation=? AND ended_at IS NULL AND recovery_disposition IS ?`, newDisposition, current.AttemptID, current.TaskID, current.ContractID, current.SessionID, current.Generation, nullableString(nullStringPtr(current.Disposition)))
		if e = operationCASResult(res, e, current.OperationID, "post-send disposition"); e != nil {
			return e
		}
	}
	if observed.Activity == "blocked" && terminalDisposition == "" {
		id, e := newAuditEventID()
		if e != nil {
			return e
		}
		_, e = appendAuditEventTx(ctx, tx, domain.AuditEvent{EventID: id, EventType: "WORKER_BLOCKED_ON_DECISION", Timestamp: at, PairID: current.PairID, TaskID: current.TaskID, ContractID: current.ContractID, AttemptID: current.AttemptID, Actor: actor, Details: map[string]any{"dispatch_operation_id": current.OperationID, "observed_activity": "blocked"}})
		if e != nil {
			return e
		}
	}
	if terminalDisposition == "MISSED_ACTIVE_WINDOW" {
		id, e := newAuditEventID()
		if e != nil {
			return e
		}
		_, e = appendAuditEventTx(ctx, tx, domain.AuditEvent{EventID: id, EventType: "MISSED_ACTIVE_WINDOW_HANDOFF_DOWNSTREAM", Timestamp: at, PairID: current.PairID, TaskID: current.TaskID, ContractID: current.ContractID, AttemptID: current.AttemptID, Actor: actor, Details: map[string]any{"dispatch_operation_id": current.OperationID, "handoff_available": false, "recovery_disposition": terminalDisposition}})
		if e != nil {
			return e
		}
	}
	return tx.Commit()
}

func (s *Store) RecordRecoveryAbsent(ctx context.Context, expected RecoveryExecution, actor string, at time.Time) error {
	if expected.DispatchStage != "SEND_CONFIRMED" {
		return ErrStateConflict
	}
	return s.ObservePostSend(ctx, expected, PostSendObservation{SessionID: expected.SessionID, Absent: true}, actor, "", at)
}

// ClassifyInterruptedRestore records the crash boundary for an unconfirmed
// /restore intent. A committed HTTP 200 remains RESTORE_CONFIRMED/IN_FLIGHT.
// Repeated classification of the exact unknown outcome is read-only.
func (s *Store) ClassifyInterruptedRestore(ctx context.Context, operationID, actor string, at time.Time) error {
	if operationID == "" || actor == "" {
		return fmt.Errorf("%w: restore classification requires operation and actor", ErrStateConflict)
	}
	if at.IsZero() {
		at = timeNow()
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var pairID, sessionID, generation, stage, resolution string
	var version int64
	err = tx.QueryRowContext(ctx, `SELECT pair_id,session_id,expected_generation,stage,resolution_state,version FROM pair_restore_operations WHERE operation_id=?`, operationID).Scan(&pairID, &sessionID, &generation, &stage, &resolution, &version)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrOperationNotFound
	}
	if err != nil {
		return err
	}
	if stage == "RESTORE_CONFIRMED" && resolution == "IN_FLIGHT" {
		return nil
	}
	if stage == "RESTORE_REQUESTED" && resolution == "RESTORE_OUTCOME_UNKNOWN" {
		return nil
	}
	if stage != "RESTORE_REQUESTED" || resolution != "IN_FLIGHT" {
		return ErrStateConflict
	}
	res, err := tx.ExecContext(ctx, `UPDATE pair_restore_operations SET resolution_state='RESTORE_OUTCOME_UNKNOWN',version=version+1 WHERE operation_id=? AND pair_id=? AND session_id=? AND expected_generation=? AND stage='RESTORE_REQUESTED' AND resolution_state='IN_FLIGHT' AND version=?`, operationID, pairID, sessionID, generation, version)
	if err = operationCASResult(res, err, operationID, "restore startup classification"); err != nil {
		return err
	}
	res, err = tx.ExecContext(ctx, `UPDATE worker_sessions SET quarantine_state='QUARANTINED',updated_at=? WHERE pair_id=? AND session_id=? AND terminal_generation=?`, formatTime(at), pairID, sessionID, generation)
	if err = operationCASResult(res, err, operationID, "restore startup session quarantine"); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE task_attempts SET quarantine_state='QUARANTINED' WHERE session_id=? AND terminal_generation=? AND task_id IN (SELECT task_id FROM tasks WHERE pair_id=?)`, sessionID, generation, pairID); err != nil {
		return err
	}
	id, err := newAuditEventID()
	if err != nil {
		return err
	}
	_, err = appendAuditEventTx(ctx, tx, domain.AuditEvent{EventID: id, EventType: domain.AuditPairRestoreOutcomeUnknown, Timestamp: at, PairID: pairID, Actor: actor, Details: map[string]any{"operation_id": operationID, "stage": stage, "old_resolution": resolution, "new_resolution": "RESTORE_OUTCOME_UNKNOWN", "session_id": sessionID, "expected_generation": generation, "source": "STARTUP_RECOVERY_SWEEP"}})
	if err != nil {
		return err
	}
	return tx.Commit()
}

// FailProvisioningAtStartup classifies only an unconfirmed create intent. The
// failed operation keeps the Pair locked by the existing unresolved index.
func (s *Store) FailProvisioningAtStartup(ctx context.Context, operationID, actor string, at time.Time) error {
	if operationID == "" || actor == "" {
		return fmt.Errorf("%w: provisioning classification requires operation and actor", ErrStateConflict)
	}
	if at.IsZero() {
		at = timeNow()
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var pairID, stage string
	err = tx.QueryRowContext(ctx, `SELECT pair_id,stage FROM pair_provisioning_operations WHERE operation_id=?`, operationID).Scan(&pairID, &stage)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrOperationNotFound
	}
	if err != nil {
		return err
	}
	if stage == "PROVISION_FAILED" {
		return nil
	}
	if stage != "PROVISION_REQUESTED" {
		return ErrStateConflict
	}
	res, err := tx.ExecContext(ctx, `UPDATE pair_provisioning_operations SET stage='PROVISION_FAILED',completed_at=?,resolved_at=?,resolution_notes='STARTUP_RECOVERY_SWEEP' WHERE operation_id=? AND pair_id=? AND stage='PROVISION_REQUESTED'`, formatTime(at), formatTime(at), operationID, pairID)
	if err = operationCASResult(res, err, operationID, "provisioning startup classification"); err != nil {
		return err
	}
	id, err := newAuditEventID()
	if err != nil {
		return err
	}
	_, err = appendAuditEventTx(ctx, tx, domain.AuditEvent{EventID: id, EventType: domain.AuditPairSessionProvisionFailed, Timestamp: at, PairID: pairID, Actor: actor, Details: map[string]any{"operation_id": operationID, "stage": "PROVISION_FAILED", "reason": "STARTUP_RECOVERY_SWEEP"}})
	if err != nil {
		return err
	}
	return tx.Commit()
}
