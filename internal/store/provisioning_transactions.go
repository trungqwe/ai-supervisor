package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/trungqwe/ai-supervisor/internal/domain"
)

// ReservePairProvisioning commits the single Pair lane reservation and audit
// before an AO create-session request may be issued.
func (s *Store) ReservePairProvisioning(ctx context.Context, operation domain.PairProvisioningOperation, actor string) error {
	if operation.OperationID == "" || operation.PairID == "" || operation.ClientToken == "" || actor == "" {
		return fmt.Errorf("store: provisioning reservation is incomplete")
	}
	now := operation.RequestedAt
	if now.IsZero() {
		now = timeNow()
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
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM worker_sessions WHERE pair_id=?`, operation.PairID).Scan(&count); err != nil {
		return err
	}
	if count != 0 {
		return fmt.Errorf("%w: Pair already has a WorkerSession", ErrStateConflict)
	}
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM pair_restore_operations WHERE pair_id=? AND resolution_state<>'RESTORE_RESOLVED') OR EXISTS(SELECT 1 FROM pair_provisioning_operations WHERE pair_id=? AND stage IN ('PROVISION_REQUESTED','PROVISION_FAILED')) OR EXISTS(SELECT 1 FROM stop_operations WHERE pair_id=? AND resolution_state IN ('IN_FLIGHT','STOP_CALL_OUTCOME_UNKNOWN','STOP_CONFIRMATION_TIMEOUT','STOP_REISSUE_REQUIRES_HUMAN'))`, operation.PairID, operation.PairID, operation.PairID).Scan(&count); err != nil {
		return err
	}
	if count != 0 {
		return fmt.Errorf("%w: Pair has unresolved operation", ErrQuarantinedExecution)
	}
	if err := tx.QueryRowContext(ctx, `SELECT
EXISTS(SELECT 1 FROM task_attempts a JOIN tasks t ON t.task_id=a.task_id WHERE t.pair_id=? AND a.ended_at IS NULL)
OR EXISTS(SELECT 1 FROM tasks t WHERE t.pair_id=? AND (
 (t.current_attempt>0 AND NOT EXISTS(SELECT 1 FROM task_attempts a WHERE a.task_id=t.task_id AND a.attempt_number=t.current_attempt))
 OR (t.state IN ('DISPATCHED','RUNNING','BLOCKED','REVIEWING') AND (t.current_attempt=0 OR NOT EXISTS(SELECT 1 FROM task_attempts a WHERE a.task_id=t.task_id AND a.attempt_number=t.current_attempt)))))`, operation.PairID, operation.PairID).Scan(&count); err != nil {
		return err
	}
	if count != 0 {
		return fmt.Errorf("%w: active or inconsistent task lineage blocks provisioning", ErrQuarantinedExecution)
	}
	var quarantined int
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM worker_sessions WHERE pair_id=? AND quarantine_state<>'CLEAN') OR EXISTS(SELECT 1 FROM task_attempts a JOIN tasks t ON t.task_id=a.task_id WHERE t.pair_id=? AND a.quarantine_state<>'CLEAN')`, operation.PairID, operation.PairID).Scan(&quarantined); err != nil {
		return err
	}
	if quarantined != 0 {
		return fmt.Errorf("%w: Pair has quarantined lineage", ErrQuarantinedExecution)
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO pair_provisioning_operations(operation_id,pair_id,stage,client_token,session_id,requested_at) VALUES(?,?, 'PROVISION_REQUESTED', ?, NULL, ?)`, operation.OperationID, operation.PairID, operation.ClientToken, formatTime(now))
	if err != nil {
		return mapLifecycleWriteError(err, "provisioning reservation")
	}
	_, err = appendAuditEventTx(ctx, tx, domain.AuditEvent{EventID: eventID, EventType: domain.AuditPairSessionProvisionRequested, Timestamp: now, PairID: operation.PairID, Actor: actor, Details: map[string]any{"operation_id": operation.OperationID, "stage": "PROVISION_REQUESTED", "client_token": operation.ClientToken}})
	if err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: commit provisioning reservation: %w", err)
	}
	return nil
}

// ConfirmPairProvisioning commits the validated HTTP 201 session and operation
// confirmation in one transaction. The caller must never repeat create after an
// ambiguous result.
func (s *Store) ConfirmPairProvisioning(ctx context.Context, operationID string, session domain.WorkerSession, actor string, completedAt time.Time) error {
	if actor == "" || session.PairID == "" || session.SessionID == "" || session.TerminalGeneration == "" {
		return fmt.Errorf("store: provisioning confirmation is incomplete")
	}
	if completedAt.IsZero() {
		completedAt = timeNow()
	}
	eventID, err := newAuditEventID()
	if err != nil {
		return err
	}
	if session.QuarantineState == "" {
		session.QuarantineState = domain.QuarantineClean
	}
	if session.CreatedAt.IsZero() {
		session.CreatedAt = completedAt
	}
	session.UpdatedAt = completedAt
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var pairID string
	if err := tx.QueryRowContext(ctx, `SELECT pair_id FROM pair_provisioning_operations WHERE operation_id=? AND stage='PROVISION_REQUESTED'`, operationID).Scan(&pairID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%w: provisioning intent absent or stale", ErrStateConflict)
		}
		return err
	}
	if pairID != session.PairID {
		return fmt.Errorf("%w: provisioned WorkerSession Pair mismatch", ErrStateConflict)
	}
	_, err = appendAuditEventTx(ctx, tx, domain.AuditEvent{EventID: eventID, EventType: domain.AuditPairSessionProvisioned, Timestamp: completedAt, PairID: pairID, Actor: actor, Details: map[string]any{"operation_id": operationID, "stage": "PROVISION_CONFIRMED", "session_id": session.SessionID, "terminal_generation": session.TerminalGeneration}})
	if err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, `UPDATE pair_provisioning_operations SET stage='PROVISION_CONFIRMED',session_id=?,completed_at=? WHERE operation_id=? AND pair_id=? AND stage='PROVISION_REQUESTED'`, session.SessionID, formatTime(completedAt), operationID, pairID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("%w: provisioning confirmation CAS lost", ErrStateConflict)
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO worker_sessions(pair_id,session_id,runtime_type,worktree_path,worker_agent_id,status,terminal_generation,quarantine_state,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?)`, session.PairID, session.SessionID, session.RuntimeType, nullableString(session.WorktreePath), session.WorkerAgentID, string(session.Status), session.TerminalGeneration, string(session.QuarantineState), formatTime(session.CreatedAt), formatTime(session.UpdatedAt))
	if err != nil {
		return mapLifecycleWriteError(err, "provisioned WorkerSession")
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: commit provisioning confirmation: %w", err)
	}
	return nil
}

// FailPairProvisioning records a single failed create attempt; it deliberately
// leaves the Pair unavailable until an authorized resolver handles the intent.
func (s *Store) FailPairProvisioning(ctx context.Context, operationID, actor, reason string, failedAt time.Time) error {
	if actor == "" || reason == "" {
		return fmt.Errorf("store: provisioning failure requires actor and reason")
	}
	if failedAt.IsZero() {
		failedAt = timeNow()
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
	if err := tx.QueryRowContext(ctx, `SELECT pair_id FROM pair_provisioning_operations WHERE operation_id=? AND stage='PROVISION_REQUESTED'`, operationID).Scan(&pairID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%w: provisioning intent absent or stale", ErrStateConflict)
		}
		return err
	}
	_, err = appendAuditEventTx(ctx, tx, domain.AuditEvent{EventID: eventID, EventType: domain.AuditPairSessionProvisionFailed, Timestamp: failedAt, PairID: pairID, Actor: actor, Details: map[string]any{"operation_id": operationID, "stage": "PROVISION_FAILED", "reason": reason}})
	if err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, `UPDATE pair_provisioning_operations SET stage='PROVISION_FAILED',completed_at=? WHERE operation_id=? AND stage='PROVISION_REQUESTED'`, formatTime(failedAt), operationID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("%w: provisioning failure CAS lost", ErrStateConflict)
	}
	return tx.Commit()
}
