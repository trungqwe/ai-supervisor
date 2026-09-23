package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/trungqwe/ai-supervisor/internal/audit"
	"github.com/trungqwe/ai-supervisor/internal/domain"
	"github.com/trungqwe/ai-supervisor/internal/workflow"
)

func newAuditEventID() (string, error) {
	var entropy [16]byte
	if _, err := rand.Read(entropy[:]); err != nil {
		return "", fmt.Errorf("store: generate audit event id: %w", err)
	}
	return "event-" + hex.EncodeToString(entropy[:]), nil
}

// appendAuditEventTx appends to the existing hash chain using the caller's transaction.
// Any returned error therefore rolls back the associated state transition as well.
func appendAuditEventTx(ctx context.Context, tx *sql.Tx, event domain.AuditEvent) (AuditRecord, error) {
	if event.EventID == "" || event.EventType == "" || event.Actor == "" {
		return AuditRecord{}, errors.New("store: audit event_id, event_type, and actor are required")
	}
	if len(event.EventID) > 256 || len(event.EventType) > 256 || len(event.Actor) > 256 ||
		len(event.TaskID) > 256 || len(event.ContractID) > 256 || len(event.AttemptID) > 256 || len(event.PairID) > 256 {
		return AuditRecord{}, errors.New("store: audit field exceeds maximum length of 256 characters")
	}
	sanitized, detailsJSON, err := audit.SanitizeAuditEvent(event)
	if err != nil {
		return AuditRecord{}, fmt.Errorf("store: failed to sanitize audit event: %w", err)
	}
	if sanitized.Timestamp.IsZero() {
		sanitized.Timestamp = timeNow()
	} else {
		sanitized.Timestamp = sanitized.Timestamp.UTC()
	}
	if sanitized.ContractID != "" && sanitized.TaskID == "" {
		return AuditRecord{}, fmt.Errorf("%w: contract_id requires task_id", ErrInvalidAuditLineage)
	}
	if sanitized.AttemptID != "" && (sanitized.TaskID == "" || sanitized.ContractID == "") {
		return AuditRecord{}, fmt.Errorf("%w: attempt_id requires task_id and contract_id", ErrInvalidAuditLineage)
	}
	if sanitized.ContractID != "" {
		var exists int
		if err := tx.QueryRowContext(ctx, "SELECT 1 FROM task_contracts WHERE contract_id = ? AND task_id = ?", sanitized.ContractID, sanitized.TaskID).Scan(&exists); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return AuditRecord{}, fmt.Errorf("%w: contract %s does not belong to task %s", ErrInvalidAuditLineage, sanitized.ContractID, sanitized.TaskID)
			}
			return AuditRecord{}, err
		}
	}
	if sanitized.AttemptID != "" {
		var exists int
		if err := tx.QueryRowContext(ctx, "SELECT 1 FROM task_attempts WHERE attempt_id = ? AND task_id = ? AND contract_id = ?", sanitized.AttemptID, sanitized.TaskID, sanitized.ContractID).Scan(&exists); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return AuditRecord{}, fmt.Errorf("%w: attempt %s does not belong to task %s and contract %s", ErrInvalidAuditLineage, sanitized.AttemptID, sanitized.TaskID, sanitized.ContractID)
			}
			return AuditRecord{}, err
		}
	}
	if sanitized.PairID != "" && sanitized.TaskID != "" {
		var exists int
		if err := tx.QueryRowContext(ctx, "SELECT 1 FROM tasks WHERE task_id = ? AND pair_id = ?", sanitized.TaskID, sanitized.PairID).Scan(&exists); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return AuditRecord{}, fmt.Errorf("%w: task %s does not belong to pair %s", ErrInvalidAuditLineage, sanitized.TaskID, sanitized.PairID)
			}
			return AuditRecord{}, err
		}
	}

	if _, err := tx.ExecContext(ctx, "UPDATE audit_chain_state SET id = 1 WHERE id = 1"); err != nil {
		return AuditRecord{}, err
	}
	var lastSequence int64
	var lastHash string
	if err := tx.QueryRowContext(ctx, "SELECT last_sequence, last_hash FROM audit_chain_state WHERE id = 1").Scan(&lastSequence, &lastHash); err != nil {
		return AuditRecord{}, fmt.Errorf("store: failed to query audit_chain_state: %w", err)
	}
	nextSequence := lastSequence + 1
	eventHash := ComputeEventHash(nextSequence, sanitized, detailsJSON, lastHash)

	var pairID, taskID, contractID, attemptID any
	if sanitized.PairID != "" {
		pairID = sanitized.PairID
	}
	if sanitized.TaskID != "" {
		taskID = sanitized.TaskID
	}
	if sanitized.ContractID != "" {
		contractID = sanitized.ContractID
	}
	if sanitized.AttemptID != "" {
		attemptID = sanitized.AttemptID
	}
	_, err = tx.ExecContext(ctx, `
INSERT INTO audit_events (
    sequence, event_id, event_type, timestamp, pair_id, task_id, contract_id,
    attempt_id, actor, details_json, prev_hash, event_hash
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`, nextSequence, sanitized.EventID, sanitized.EventType, formatTime(sanitized.Timestamp), pairID, taskID,
		contractID, attemptID, sanitized.Actor, detailsJSON, lastHash, eventHash)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") || strings.Contains(err.Error(), "cannot replace existing audit event") {
			return AuditRecord{}, fmt.Errorf("%w: duplicate audit event_id %s: %v", ErrDuplicateKey, sanitized.EventID, err)
		}
		return AuditRecord{}, fmt.Errorf("store: failed to insert audit_event: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "UPDATE audit_chain_state SET last_sequence = ?, last_hash = ? WHERE id = 1", nextSequence, eventHash); err != nil {
		return AuditRecord{}, fmt.Errorf("store: failed to update audit_chain_state: %w", err)
	}
	return AuditRecord{Sequence: nextSequence, Event: sanitized, PrevHash: lastHash, EventHash: eventHash}, nil
}

func currentOpenAttemptLineage(ctx context.Context, tx *sql.Tx, taskID, attemptID string) (pairID, contractID string, err error) {
	err = tx.QueryRowContext(ctx, `
SELECT t.pair_id, a.contract_id
FROM tasks t
JOIN task_attempts a ON a.task_id = t.task_id AND a.attempt_number = t.current_attempt
WHERE t.task_id = ? AND a.attempt_id = ? AND a.ended_at IS NULL
`, taskID, attemptID).Scan(&pairID, &contractID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", fmt.Errorf("%w: attempt %q is not the current open attempt for task %q", ErrAttemptLineageMismatch, attemptID, taskID)
	}
	if err != nil {
		return "", "", fmt.Errorf("store: current attempt lineage query: %w", err)
	}
	return pairID, contractID, nil
}

// AtomicTerminalTransition commits TaskState CAS, current-attempt closure, disposition,
// and a hash-chained audit event in one SQLite transaction.
func (s *Store) AtomicTerminalTransition(
	ctx context.Context,
	taskID string,
	expectedCurrentState domain.TaskState,
	newState domain.TaskState,
	failureReason string,
	attemptID string,
	disposition string,
) error {
	if newState != domain.StateFailed ||
		(expectedCurrentState != domain.StateRunning && expectedCurrentState != domain.StateDispatched) {
		return fmt.Errorf("%w: AtomicTerminalTransition only permits DISPATCHED/RUNNING -> FAILED", ErrInvalidOperationTransition)
	}
	if err := workflow.Transition(expectedCurrentState, newState); err != nil {
		return fmt.Errorf("store: canonical terminal transition rejected: %w", err)
	}
	if strings.TrimSpace(disposition) == "" {
		return errors.New("store: terminal recovery disposition is required")
	}
	if strings.TrimSpace(failureReason) == "" {
		return errors.New("store: terminal failure reason is required for audit details")
	}
	eventID, err := newAuditEventID()
	if err != nil {
		return err
	}
	now := timeNow()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin atomic terminal transition: %w", err)
	}
	defer tx.Rollback()
	pairID, contractID, err := currentOpenAttemptLineage(ctx, tx, taskID, attemptID)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `
UPDATE tasks SET state = ?, updated_at = ? WHERE task_id = ? AND state = ?
`, string(newState), formatTime(now), taskID, string(expectedCurrentState))
	if err != nil {
		return fmt.Errorf("store: terminal task CAS: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("store: terminal task CAS rows affected: %w", err)
	}
	if rows != 1 {
		return fmt.Errorf("%w: task %q is not in expected state %q", ErrStateConflict, taskID, expectedCurrentState)
	}
	attemptCloseSQL := `
UPDATE task_attempts SET ended_at = ?, recovery_disposition = ?
WHERE attempt_id = ? AND task_id = ? AND ended_at IS NULL
  AND attempt_number = (SELECT current_attempt FROM tasks WHERE task_id = ?)
	`
	requiresDoubleQuarantine := disposition == "PRE_SEND_IDENTITY_MISMATCH" || disposition == "STALE_EXECUTION_GENERATION" || disposition == "SESSION_ABSENT"
	if requiresDoubleQuarantine {
		attemptCloseSQL = `UPDATE task_attempts SET ended_at = ?, recovery_disposition = ?, quarantine_state='QUARANTINED' WHERE attempt_id = ? AND task_id = ? AND ended_at IS NULL AND attempt_number = (SELECT current_attempt FROM tasks WHERE task_id = ?)`
	}
	result, err = tx.ExecContext(ctx, attemptCloseSQL, formatTime(now), disposition, attemptID, taskID, taskID)
	if err != nil {
		return fmt.Errorf("store: close current attempt: %w", err)
	}
	rows, err = result.RowsAffected()
	if err != nil || rows != 1 {
		if err != nil {
			return fmt.Errorf("store: current attempt closure rows affected: %w", err)
		}
		return fmt.Errorf("%w: attempt %q was not closed", ErrAttemptLineageMismatch, attemptID)
	}
	if requiresDoubleQuarantine {
		result, err = tx.ExecContext(ctx, `UPDATE worker_sessions SET quarantine_state='QUARANTINED',updated_at=? WHERE pair_id=? AND EXISTS(SELECT 1 FROM task_attempts a WHERE a.attempt_id=? AND a.task_id=? AND a.session_id=worker_sessions.session_id AND a.terminal_generation=worker_sessions.terminal_generation)`, formatTime(now), pairID, attemptID, taskID)
		if err != nil {
			return fmt.Errorf("store: quarantine failed attempt session: %w", err)
		}
		rows, err = result.RowsAffected()
		if err != nil || rows != 1 {
			if err != nil {
				return err
			}
			return fmt.Errorf("%w: failure session snapshot mismatch", ErrAttemptLineageMismatch)
		}
	}
	_, err = appendAuditEventTx(ctx, tx, domain.AuditEvent{
		EventID:    eventID,
		EventType:  domain.AuditTaskStateTransition,
		Timestamp:  now,
		PairID:     pairID,
		TaskID:     taskID,
		ContractID: contractID,
		AttemptID:  attemptID,
		Actor:      "supervisor",
		Details: map[string]any{
			"from_state":           string(expectedCurrentState),
			"to_state":             string(newState),
			"failure_reason":       failureReason,
			"recovery_disposition": disposition,
		},
	})
	if err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: commit atomic terminal transition: %w", err)
	}
	return nil
}

// AtomicAttemptClosureTransition commits BLOCKED -> HUMAN_REQUIRED, current-attempt
// closure, AO_BLOCKED_ESCALATED disposition, and audit append in one transaction.
func (s *Store) AtomicAttemptClosureTransition(ctx context.Context, taskID, attemptID, disposition string) error {
	if disposition != domain.RecoveryAOBlockedEscalated {
		return fmt.Errorf("%w: closure disposition must be %q", ErrInvalidOperationTransition, domain.RecoveryAOBlockedEscalated)
	}
	eventID, err := newAuditEventID()
	if err != nil {
		return err
	}
	now := timeNow()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin atomic attempt closure: %w", err)
	}
	defer tx.Rollback()
	pairID, contractID, err := currentOpenAttemptLineage(ctx, tx, taskID, attemptID)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `
UPDATE tasks SET state = ?, updated_at = ? WHERE task_id = ? AND state = ?
`, string(domain.StateHumanRequired), formatTime(now), taskID, string(domain.StateBlocked))
	if err != nil {
		return fmt.Errorf("store: attempt closure task CAS: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("store: attempt closure task CAS rows affected: %w", err)
	}
	if rows != 1 {
		return fmt.Errorf("%w: task %q is not in expected state BLOCKED", ErrStateConflict, taskID)
	}
	result, err = tx.ExecContext(ctx, `
UPDATE task_attempts SET ended_at = ?, recovery_disposition = ?
WHERE attempt_id = ? AND task_id = ? AND ended_at IS NULL
  AND attempt_number = (SELECT current_attempt FROM tasks WHERE task_id = ?)
`, formatTime(now), domain.RecoveryAOBlockedEscalated, attemptID, taskID, taskID)
	if err != nil {
		return fmt.Errorf("store: close blocked attempt: %w", err)
	}
	rows, err = result.RowsAffected()
	if err != nil || rows != 1 {
		if err != nil {
			return fmt.Errorf("store: blocked attempt closure rows affected: %w", err)
		}
		return fmt.Errorf("%w: attempt %q was not closed", ErrAttemptLineageMismatch, attemptID)
	}
	_, err = appendAuditEventTx(ctx, tx, domain.AuditEvent{
		EventID:    eventID,
		EventType:  domain.AuditWorkerBlockedEscalated,
		Timestamp:  now,
		PairID:     pairID,
		TaskID:     taskID,
		ContractID: contractID,
		AttemptID:  attemptID,
		Actor:      "supervisor",
		Details: map[string]any{
			"from_state":           string(domain.StateBlocked),
			"to_state":             string(domain.StateHumanRequired),
			"recovery_disposition": domain.RecoveryAOBlockedEscalated,
		},
	})
	if err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: commit atomic attempt closure: %w", err)
	}
	return nil
}
