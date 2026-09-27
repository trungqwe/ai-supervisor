package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/trungqwe/ai-supervisor/internal/domain"
)

var (
	ErrWorkspaceBindingNotFound   = errors.New("store: workspace binding not found")
	ErrPreSendGuardRejected       = errors.New("store: pre-send guard rejected")
	ErrReviewIntegrityHoldActive   = errors.New("store: active review integrity hold blocks operation")
	ErrWorkerClaimNotFound        = errors.New("store: worker claim not found")
	ErrReviewIntegrityHoldNotFound = errors.New("store: review integrity hold not found")
)

// GetAttemptWorkspaceBinding retrieves a WorkspaceBinding by attemptID.
func (s *Store) GetAttemptWorkspaceBinding(ctx context.Context, attemptID string) (*domain.WorkspaceBinding, error) {
	query := `
SELECT attempt_id, task_id, contract_id, session_id, terminal_generation,
       canonical_worktree_path, volume_serial_hex, file_id_hex,
       linked_gitdir_path, linked_gitdir_volume_serial_hex, linked_gitdir_file_id_hex,
       pinned_ao_commit, binding_state, created_at_epoch_ms, released_at_epoch_ms
FROM attempt_workspace_bindings
WHERE attempt_id = ?
`
	var b domain.WorkspaceBinding
	var releasedAt sql.NullInt64

	err := s.db.QueryRowContext(ctx, query, attemptID).Scan(
		&b.AttemptID,
		&b.TaskID,
		&b.ContractID,
		&b.SessionID,
		&b.TerminalGeneration,
		&b.CanonicalWorktreePath,
		&b.VolumeSerialHex,
		&b.FileIDHex,
		&b.LinkedGitDirPath,
		&b.LinkedGitDirVolumeSerialHex,
		&b.LinkedGitDirFileIDHex,
		&b.PinnedAOCommit,
		&b.BindingState,
		&b.CreatedAtEpochMS,
		&releasedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrWorkspaceBindingNotFound
		}
		return nil, fmt.Errorf("store: failed to query workspace binding: %w", err)
	}

	if releasedAt.Valid {
		val := releasedAt.Int64
		b.ReleasedAtEpochMS = &val
	}

	return &b, nil
}

// GetActiveWorkspaceBinding retrieves an ACTIVE WorkspaceBinding by attemptID.
func (s *Store) GetActiveWorkspaceBinding(ctx context.Context, attemptID string) (*domain.WorkspaceBinding, error) {
	b, err := s.GetAttemptWorkspaceBinding(ctx, attemptID)
	if err != nil {
		return nil, err
	}
	if b.BindingState != domain.BindingStateActive {
		return nil, fmt.Errorf("%w: binding state is %s", ErrWorkspaceBindingNotFound, b.BindingState)
	}
	return b, nil
}

// RecordVariantBDiagnostic executes the atomic pre-send diagnostic transaction (Variant B).
// It appends REVIEW_INTEGRITY_CONFLICT, inserts an ACTIVE hold with reason INVARIANT_MISMATCH,
// CAS invalidates any ACTIVE binding for the attempt, and keeps task DISPATCHED and attempt open.
func (s *Store) RecordVariantBDiagnostic(ctx context.Context, operationID, attemptedReason, actor string, at time.Time) error {
	switch attemptedReason {
	case "WORKSPACE_BINDING_MISSING",
		"WORKSPACE_BINDING_LINEAGE_MISMATCH",
		"WORKSPACE_BINDING_NOT_ACTIVE",
		"WORKSPACE_BINDING_PHYSICAL_IDENTITY_MISMATCH":
	default:
		return fmt.Errorf("store: invalid attempted_reason %q for Variant B", attemptedReason)
	}

	if actor == "" {
		actor = "SUPERVISOR"
	}
	if at.IsZero() {
		at = timeNow()
	}
	epochMS := at.UnixMilli()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: failed to begin diagnostic transaction: %w", err)
	}
	defer tx.Rollback()

	var pairID, taskID, contractID, attemptID, sessionID, generation, stage string
	err = tx.QueryRowContext(ctx, `
SELECT d.pair_id, d.task_id, a.contract_id, d.attempt_id, d.session_id, d.terminal_generation, d.stage
FROM dispatch_operations d
JOIN task_attempts a ON a.attempt_id = d.attempt_id
WHERE d.operation_id = ?
`, operationID).Scan(&pairID, &taskID, &contractID, &attemptID, &sessionID, &generation, &stage)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrOperationNotFound
		}
		return err
	}

	diagnosticInput := map[string]any{
		"attempt_id":            attemptID,
		"attempted_reason":      attemptedReason,
		"dispatch_operation_id": operationID,
	}
	sanitizedFP, err := domain.ComputeFingerprint(diagnosticInput)
	if err != nil {
		return fmt.Errorf("store: failed to compute sanitized input fingerprint: %w", err)
	}
	diagnosticFP := sanitizedFP

	// Check if active hold with identical diagnostic fingerprint already exists for idempotency/replay
	var existingActiveHoldID string
	var existingRejectionEventID string
	err = tx.QueryRowContext(ctx, `
SELECT hold_id, rejection_audit_event_id FROM review_integrity_holds
WHERE attempt_id = ? AND hold_reason = 'INVARIANT_MISMATCH' AND diagnostic_fingerprint = ? AND hold_state = 'ACTIVE'
`, attemptID, diagnosticFP).Scan(&existingActiveHoldID, &existingRejectionEventID)
	if err == nil {
		// Idempotent replay: already held
		return nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	// Calculate occurrence_number
	var occurrenceCount int
	err = tx.QueryRowContext(ctx, `
SELECT COUNT(*) FROM review_integrity_holds
WHERE attempt_id = ? AND hold_reason = 'INVARIANT_MISMATCH' AND diagnostic_fingerprint = ?
`, attemptID, diagnosticFP).Scan(&occurrenceCount)
	if err != nil {
		return err
	}
	occurrenceNumber := occurrenceCount + 1

	holdDesc := domain.HoldIdentityDescriptor{
		AttemptID:             attemptID,
		ContractID:            contractID,
		DiagnosticFingerprint: diagnosticFP,
		HoldReason:            string(domain.HoldReasonInvariantMismatch),
		Kind:                  "review_integrity_hold",
		OccurrenceNumber:      occurrenceNumber,
		PairID:                pairID,
		TaskID:                taskID,
		Version:               1,
	}
	holdID, err := domain.DeriveHoldID(holdDesc)
	if err != nil {
		return fmt.Errorf("store: failed to derive hold_id: %w", err)
	}

	rejDesc := domain.RejectionEventDescriptorVariantB{
		AttemptID:                 attemptID,
		AttemptedReason:           attemptedReason,
		ConflictSource:            "WORKSPACE_BINDING_GUARD",
		ConflictType:              "LINEAGE_MISMATCH",
		ContractID:                contractID,
		DiagnosticFingerprint:     diagnosticFP,
		DispatchOperationID:       operationID,
		EventType:                 domain.AuditReviewIntegrityConflict,
		HoldID:                    holdID,
		OccurrenceNumber:          occurrenceNumber,
		PairID:                    pairID,
		SanitizedInputFingerprint: sanitizedFP,
		TaskID:                    taskID,
		Version:                   2,
	}
	rejectionEventID, err := domain.DeriveRejectionEventIDVariantB(rejDesc)
	if err != nil {
		return fmt.Errorf("store: failed to derive rejection_event_id: %w", err)
	}

	auditDetails := map[string]any{
		"conflict_source":             "WORKSPACE_BINDING_GUARD",
		"dispatch_operation_id":       operationID,
		"attempted_reason":            attemptedReason,
		"conflict_type":               "LINEAGE_MISMATCH",
		"sanitized_input_fingerprint": sanitizedFP,
		"diagnostic_fingerprint":      diagnosticFP,
		"actor_role":                  "SUPERVISOR",
		"hold_id":                     holdID,
		"occurrence_number":           occurrenceNumber,
	}

	// 1. Append audit event
	_, err = appendAuditEventTx(ctx, tx, domain.AuditEvent{
		EventID:    rejectionEventID,
		EventType:  domain.AuditReviewIntegrityConflict,
		Timestamp:  at,
		PairID:     pairID,
		TaskID:     taskID,
		ContractID: contractID,
		AttemptID:  attemptID,
		Actor:      actor,
		Details:    auditDetails,
	})
	if err != nil {
		return fmt.Errorf("store: failed to append conflict audit event: %w", err)
	}

	// 2. Insert ACTIVE hold
	insertHoldQuery := `
INSERT INTO review_integrity_holds (
    hold_id, task_id, attempt_id, contract_id, hold_reason, hold_state,
    diagnostic_fingerprint, occurrence_number, rejection_audit_event_id,
    created_at_epoch_ms
) VALUES (?, ?, ?, ?, 'INVARIANT_MISMATCH', 'ACTIVE', ?, ?, ?, ?)
`
	_, err = tx.ExecContext(ctx, insertHoldQuery,
		holdID, taskID, attemptID, contractID,
		diagnosticFP, occurrenceNumber, rejectionEventID,
		epochMS,
	)
	if err != nil {
		return fmt.Errorf("store: failed to insert review integrity hold: %w", err)
	}

	// 3. CAS invalidate active binding if row exists
	updateBindingQuery := `
UPDATE attempt_workspace_bindings
SET binding_state = 'INVALIDATED', released_at_epoch_ms = ?
WHERE attempt_id = ? AND binding_state = 'ACTIVE'
`
	_, err = tx.ExecContext(ctx, updateBindingQuery, epochMS, attemptID)
	if err != nil {
		return fmt.Errorf("store: failed to CAS invalidate workspace binding: %w", err)
	}

	return tx.Commit()
}

// GetActiveReviewIntegrityHolds returns all ACTIVE review integrity holds for an attempt.
func (s *Store) GetActiveReviewIntegrityHolds(ctx context.Context, attemptID string) ([]domain.ReviewIntegrityHold, error) {
	query := `
SELECT hold_id, task_id, attempt_id, contract_id, hold_reason, hold_state,
       diagnostic_fingerprint, occurrence_number, rejection_audit_event_id,
       resolution_audit_event_id, resolved_by_principal, created_at_epoch_ms,
       resolved_at_epoch_ms
FROM review_integrity_holds
WHERE attempt_id = ? AND hold_state = 'ACTIVE'
ORDER BY created_at_epoch_ms ASC
`
	rows, err := s.db.QueryContext(ctx, query, attemptID)
	if err != nil {
		return nil, fmt.Errorf("store: failed to query active review integrity holds: %w", err)
	}
	defer rows.Close()

	var holds []domain.ReviewIntegrityHold
	for rows.Next() {
		var h domain.ReviewIntegrityHold
		var resEventID, resPrincipal sql.NullString
		var resAt sql.NullInt64

		err := rows.Scan(
			&h.HoldID,
			&h.TaskID,
			&h.AttemptID,
			&h.ContractID,
			&h.HoldReason,
			&h.HoldState,
			&h.DiagnosticFingerprint,
			&h.OccurrenceNumber,
			&h.RejectionAuditEventID,
			&resEventID,
			&resPrincipal,
			&h.CreatedAtEpochMS,
			&resAt,
		)
		if err != nil {
			return nil, fmt.Errorf("store: failed to scan review integrity hold: %w", err)
		}

		if resEventID.Valid {
			h.ResolutionAuditEventID = &resEventID.String
		}
		if resPrincipal.Valid {
			h.ResolvedByPrincipal = &resPrincipal.String
		}
		if resAt.Valid {
			val := resAt.Int64
			h.ResolvedAtEpochMS = &val
		}

		holds = append(holds, h)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return holds, nil
}

// ResolveReviewIntegrityHold resolves an ACTIVE hold following governed terminalization.
func (s *Store) ResolveReviewIntegrityHold(ctx context.Context, holdID, resolutionEventID, resolvedByPrincipal string, at time.Time) error {
	if strings.TrimSpace(resolvedByPrincipal) == "" {
		return errors.New("store: resolvedByPrincipal is required and must not be blank")
	}
	if at.IsZero() {
		at = timeNow()
	}
	epochMS := at.UnixMilli()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var taskID, attemptID, contractID, holdReason, diagnosticFP string
	var occurrenceNumber int
	var createdAtEpochMS int64
	err = tx.QueryRowContext(ctx, `
SELECT task_id, attempt_id, contract_id, hold_reason, diagnostic_fingerprint, occurrence_number, created_at_epoch_ms
FROM review_integrity_holds
WHERE hold_id = ? AND hold_state = 'ACTIVE'
`, holdID).Scan(&taskID, &attemptID, &contractID, &holdReason, &diagnosticFP, &occurrenceNumber, &createdAtEpochMS)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrReviewIntegrityHoldNotFound
		}
		return err
	}

	// Verify governed terminalization: attempt must be terminated / ended
	var endedAt sql.NullString
	var recoveryDisposition sql.NullString
	err = tx.QueryRowContext(ctx, `
SELECT ended_at, recovery_disposition FROM task_attempts WHERE attempt_id = ?
`, attemptID).Scan(&endedAt, &recoveryDisposition)
	if err != nil {
		return err
	}
	if !endedAt.Valid {
		return errors.New("store: governed terminalization required: attempt must be ended before resolving hold")
	}

	var pairID string
	err = tx.QueryRowContext(ctx, `SELECT pair_id FROM tasks WHERE task_id = ?`, taskID).Scan(&pairID)
	if err != nil {
		return err
	}

	if resolutionEventID == "" {
		resDesc := domain.ResolutionEventDescriptor{
			AttemptID:                              attemptID,
			ContractID:                             contractID,
			EventType:                              domain.AuditReviewIntegrityHoldResolved,
			HoldID:                                 holdID,
			Kind:                                   "review_integrity_resolution_event",
			OccurrenceNumber:                       occurrenceNumber,
			PairID:                                 pairID,
			ResolvedByPrincipal:                    resolvedByPrincipal,
			SanitizedResolutionRationaleFingerprint: diagnosticFP,
			TaskID:                                 taskID,
			Version:                                1,
		}
		derived, err := domain.DeriveResolutionEventID(resDesc)
		if err != nil {
			return err
		}
		resolutionEventID = derived
	}

	// Append resolution audit event
	_, err = appendAuditEventTx(ctx, tx, domain.AuditEvent{
		EventID:    resolutionEventID,
		EventType:  domain.AuditReviewIntegrityHoldResolved,
		Timestamp:  at,
		PairID:     pairID,
		TaskID:     taskID,
		ContractID: contractID,
		AttemptID:  attemptID,
		Actor:      resolvedByPrincipal,
		Details: map[string]any{
			"hold_id":               holdID,
			"resolved_by_principal": resolvedByPrincipal,
			"hold_reason":           holdReason,
		},
	})
	if err != nil {
		return fmt.Errorf("store: failed to append resolution audit event: %w", err)
	}

	// CAS update hold to RESOLVED
	res, err := tx.ExecContext(ctx, `
UPDATE review_integrity_holds
SET hold_state = 'RESOLVED',
    resolution_audit_event_id = ?,
    resolved_by_principal = ?,
    resolved_at_epoch_ms = ?
WHERE hold_id = ? AND hold_state = 'ACTIVE'
`, resolutionEventID, resolvedByPrincipal, epochMS, holdID)
	if err != nil {
		return fmt.Errorf("store: failed to resolve review integrity hold: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("%w: hold resolution CAS lost", ErrStateConflict)
	}

	return tx.Commit()
}

// GetWorkerClaim retrieves a WorkerClaimRecord by attemptID.
func (s *Store) GetWorkerClaim(ctx context.Context, attemptID string) (*domain.WorkerClaimRecord, error) {
	query := `
SELECT claim_id, task_id, attempt_id, contract_id, reported_head_sha,
       payload_json, created_at_epoch_ms
FROM worker_claims
WHERE attempt_id = ?
`
	var c domain.WorkerClaimRecord
	err := s.db.QueryRowContext(ctx, query, attemptID).Scan(
		&c.ClaimID,
		&c.TaskID,
		&c.AttemptID,
		&c.ContractID,
		&c.ReportedHeadSHA,
		&c.PayloadJSON,
		&c.CreatedAtEpochMS,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrWorkerClaimNotFound
		}
		return nil, fmt.Errorf("store: failed to query worker claim: %w", err)
	}
	return &c, nil
}
