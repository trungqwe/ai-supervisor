package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/trungqwe/ai-supervisor/internal/domain"
)

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
