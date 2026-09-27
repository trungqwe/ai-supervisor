package store

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/trungqwe/ai-supervisor/internal/contract"
	"github.com/trungqwe/ai-supervisor/internal/domain"
)

var (
	shaHexCheckRegex         = regexp.MustCompile(`^[0-9a-f]{7,40}$`)
	ErrDirtyWorktreeDetected = errors.New("store: dirty worktree detected at report intake")
)

//go:embed worker_report_schema.json
var workerReportSchemaBytes []byte

var (
	workerReportValidatorOnce sync.Once
	workerReportValidatorFunc func([]byte) (any, error)
	workerReportValidatorErr  error

	// Test seam for validating schema compile failure fail-closed behavior
	testSchemaOverrideBytes []byte
)

func getWorkerReportValidator() (func([]byte) (any, error), error) {
	if testSchemaOverrideBytes != nil {
		resolved, err := contract.CompileSchema(testSchemaOverrideBytes)
		if err != nil {
			return nil, fmt.Errorf("store: failed to compile worker report schema: %w", err)
		}
		if resolved == nil {
			return nil, errors.New("store: compiled worker report schema is nil")
		}
		return func(rawJSON []byte) (any, error) {
			return contract.ParseAndValidateRaw(rawJSON, resolved)
		}, nil
	}

	workerReportValidatorOnce.Do(func() {
		resolved, err := contract.CompileSchema(workerReportSchemaBytes)
		if err != nil {
			workerReportValidatorErr = fmt.Errorf("store: failed to compile worker report schema: %w", err)
			return
		}
		if resolved == nil {
			workerReportValidatorErr = errors.New("store: compiled worker report schema is nil")
			return
		}
		workerReportValidatorFunc = func(rawJSON []byte) (any, error) {
			return contract.ParseAndValidateRaw(rawJSON, resolved)
		}
	})
	if workerReportValidatorErr != nil {
		return nil, workerReportValidatorErr
	}
	if workerReportValidatorFunc == nil {
		return nil, errors.New("store: worker report validator unavailable")
	}
	return workerReportValidatorFunc, nil
}

// IngestWorkerReportRaw parses and validates raw JSON against schema before ingesting.
// It is the sole exported admission entry point for worker reports.
func (s *Store) IngestWorkerReportRaw(
	ctx context.Context,
	taskID string,
	contractID string,
	attemptID string,
	rawReport []byte,
	gitEvidence domain.GitEvidenceResult,
	actor string,
	at time.Time,
) error {
	validate, err := getWorkerReportValidator()
	if err != nil {
		return fmt.Errorf("store: worker report schema validation unavailable: %w", err)
	}
	if _, err := validate(rawReport); err != nil {
		return fmt.Errorf("store: worker report schema validation failed: %w", err)
	}

	var report domain.WorkerReport
	if err := json.Unmarshal(rawReport, &report); err != nil {
		return fmt.Errorf("store: failed to unmarshal worker report: %w", err)
	}

	return s.ingestWorkerReportDecoded(ctx, taskID, contractID, attemptID, report, gitEvidence, actor, at)
}

// ingestWorkerReportDecoded executes report intake: Transaction A on clean evidence,
// or intake diagnostic transaction on dirty evidence. It is a package-private helper
// called exclusively after successful raw JSON schema validation.
func (s *Store) ingestWorkerReportDecoded(
	ctx context.Context,
	taskID string,
	contractID string,
	attemptID string,
	report domain.WorkerReport,
	gitEvidence domain.GitEvidenceResult,
	actor string,
	at time.Time,
) error {
	if actor == "" {
		actor = "supervisor"
	}
	if at.IsZero() {
		at = timeNow()
	}
	epochMS := at.UnixMilli()

	if err := report.Validate(); err != nil {
		return fmt.Errorf("store: invalid worker report: %w", err)
	}
	if report.TaskID != taskID {
		return fmt.Errorf("%w: report task_id %q does not match task_id %q", ErrAttemptLineageMismatch, report.TaskID, taskID)
	}
	if report.AttemptID != attemptID {
		return fmt.Errorf("%w: report attempt_id %q does not match attempt_id %q", ErrAttemptLineageMismatch, report.AttemptID, attemptID)
	}

	contract, err := s.GetTaskContract(ctx, contractID)
	if err != nil {
		return fmt.Errorf("store: failed to get task contract: %w", err)
	}
	if report.BaseSHA != contract.BaseSHA {
		return fmt.Errorf("%w: report base_sha %q does not match contract base_sha %q", ErrAttemptLineageMismatch, report.BaseSHA, contract.BaseSHA)
	}

	// Case 1: Dirty Worktree -> execute separate intake diagnostic transaction
	if !gitEvidence.IsClean {
		return s.recordDirtyIntakeDiagnostic(ctx, taskID, contractID, attemptID, gitEvidence, actor, at)
	}

	// Case 2: Clean Worktree -> execute Transaction A atomically
	return s.executeTransactionA(ctx, taskID, contractID, attemptID, report, actor, at, epochMS)
}

func (s *Store) recordDirtyIntakeDiagnostic(
	ctx context.Context,
	taskID string,
	contractID string,
	attemptID string,
	gitEvidence domain.GitEvidenceResult,
	actor string,
	at time.Time,
) error {
	epochMS := at.UnixMilli()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var pairID, taskState, attemptContractID string
	var currentAttempt, attemptNumber int
	var endedAt sql.NullString
	err = tx.QueryRowContext(ctx, `
SELECT t.pair_id, t.state, t.current_attempt, a.attempt_number, a.contract_id, a.ended_at
FROM tasks t
JOIN task_attempts a ON a.task_id = t.task_id
WHERE t.task_id = ? AND a.attempt_id = ?
`, taskID, attemptID).Scan(&pairID, &taskState, &currentAttempt, &attemptNumber, &attemptContractID, &endedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrAttemptNotFound
		}
		return err
	}

	if taskState != string(domain.StateRunning) {
		return fmt.Errorf("%w: task state is %s, expected RUNNING", ErrStateConflict, taskState)
	}
	if currentAttempt != attemptNumber || endedAt.Valid {
		return fmt.Errorf("%w: attempt is not current open attempt", ErrAttemptLineageMismatch)
	}
	if attemptContractID != contractID {
		return fmt.Errorf("%w: contract lineage mismatch (got %q, attempt has %q)", ErrAttemptLineageMismatch, contractID, attemptContractID)
	}

	diagnosticInput := map[string]any{
		"attempt_id":        attemptID,
		"hold_reason":       string(domain.HoldReasonDirtyWorktreeDetected),
		"uncommitted_files": gitEvidence.UncommittedFiles,
	}
	sanitizedFP, err := domain.ComputeFingerprint(diagnosticInput)
	if err != nil {
		return fmt.Errorf("store: failed to compute diagnostic fingerprint: %w", err)
	}
	diagnosticFP := sanitizedFP

	// Check if active hold with identical diagnostic fingerprint already exists (replay)
	var existingHoldID string
	err = tx.QueryRowContext(ctx, `
SELECT hold_id FROM review_integrity_holds
WHERE attempt_id = ? AND hold_reason = 'DIRTY_WORKTREE_DETECTED' AND diagnostic_fingerprint = ? AND hold_state = 'ACTIVE'
`, attemptID, diagnosticFP).Scan(&existingHoldID)
	if err == nil {
		// Exact replay: active hold already recorded
		return ErrDirtyWorktreeDetected
	} else if !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	// Calculate occurrence_number
	var occurrenceCount int
	err = tx.QueryRowContext(ctx, `
SELECT COUNT(*) FROM review_integrity_holds
WHERE attempt_id = ? AND hold_reason = 'DIRTY_WORKTREE_DETECTED' AND diagnostic_fingerprint = ?
`, attemptID, diagnosticFP).Scan(&occurrenceCount)
	if err != nil {
		return err
	}
	occurrenceNumber := occurrenceCount + 1

	holdDesc := domain.HoldIdentityDescriptor{
		AttemptID:             attemptID,
		ContractID:            contractID,
		DiagnosticFingerprint: diagnosticFP,
		HoldReason:            string(domain.HoldReasonDirtyWorktreeDetected),
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

	rejDesc := domain.RejectionEventDescriptor{
		AttemptID:                 attemptID,
		ContractID:                contractID,
		DiagnosticFingerprint:     diagnosticFP,
		EventType:                 domain.AuditEvidenceCollectionFailed,
		HoldID:                    holdID,
		OccurrenceNumber:          occurrenceNumber,
		PairID:                    pairID,
		Reason:                    string(domain.HoldReasonDirtyWorktreeDetected),
		SanitizedInputFingerprint: sanitizedFP,
		TaskID:                    taskID,
		Version:                   1,
	}
	rejectionEventID, err := domain.DeriveRejectionEventID(rejDesc)
	if err != nil {
		return fmt.Errorf("store: failed to derive rejection_event_id: %w", err)
	}

	// 1. Append rejection audit event EVIDENCE_COLLECTION_FAILED
	_, err = appendAuditEventTx(ctx, tx, domain.AuditEvent{
		EventID:    rejectionEventID,
		EventType:  domain.AuditEvidenceCollectionFailed,
		Timestamp:  at,
		PairID:     pairID,
		TaskID:     taskID,
		ContractID: contractID,
		AttemptID:  attemptID,
		Actor:      actor,
		Details: map[string]any{
			"actor_role":                  "SUPERVISOR",
			"hold_id":                     holdID,
			"hold_reason":                 string(domain.HoldReasonDirtyWorktreeDetected),
			"diagnostic_fingerprint":      diagnosticFP,
			"sanitized_input_fingerprint": sanitizedFP,
			"uncommitted_files":           gitEvidence.UncommittedFiles,
			"occurrence_number":           occurrenceNumber,
		},
	})
	if err != nil {
		return fmt.Errorf("store: failed to append evidence collection failed audit: %w", err)
	}

	// 2. Insert ACTIVE hold row into review_integrity_holds
	insertHoldQuery := `
INSERT INTO review_integrity_holds (
    hold_id, task_id, attempt_id, contract_id, hold_reason, hold_state,
    diagnostic_fingerprint, occurrence_number, rejection_audit_event_id,
    created_at_epoch_ms
) VALUES (?, ?, ?, ?, 'DIRTY_WORKTREE_DETECTED', 'ACTIVE', ?, ?, ?, ?)
`
	_, err = tx.ExecContext(ctx, insertHoldQuery,
		holdID, taskID, attemptID, contractID,
		diagnosticFP, occurrenceNumber, rejectionEventID,
		epochMS,
	)
	if err != nil {
		return fmt.Errorf("store: failed to insert dirty worktree hold: %w", err)
	}

	// Commit diagnostic transaction; task remains RUNNING
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: failed to commit dirty intake diagnostic transaction: %w", err)
	}

	return ErrDirtyWorktreeDetected
}

func (s *Store) executeTransactionA(
	ctx context.Context,
	taskID string,
	contractID string,
	attemptID string,
	report domain.WorkerReport,
	actor string,
	at time.Time,
	epochMS int64,
) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Verify task is RUNNING
	var taskState, attemptContractID string
	var currentAttempt, attemptNumber int
	var endedAt sql.NullString
	err = tx.QueryRowContext(ctx, `
SELECT t.state, t.current_attempt, a.attempt_number, a.contract_id, a.ended_at
FROM tasks t
JOIN task_attempts a ON a.task_id = t.task_id
WHERE t.task_id = ? AND a.attempt_id = ?
`, taskID, attemptID).Scan(&taskState, &currentAttempt, &attemptNumber, &attemptContractID, &endedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrAttemptNotFound
		}
		return err
	}

	if taskState != string(domain.StateRunning) {
		return fmt.Errorf("%w: task state is %s, expected RUNNING", ErrStateConflict, taskState)
	}
	if currentAttempt != attemptNumber || endedAt.Valid {
		return fmt.Errorf("%w: attempt is not current open attempt", ErrAttemptLineageMismatch)
	}
	if attemptContractID != contractID {
		return fmt.Errorf("%w: contract lineage mismatch (got %q, attempt has %q)", ErrAttemptLineageMismatch, contractID, attemptContractID)
	}

	// 2. Verify workspace binding is ACTIVE
	var bindingState string
	err = tx.QueryRowContext(ctx, `
SELECT binding_state FROM attempt_workspace_bindings WHERE attempt_id = ?
`, attemptID).Scan(&bindingState)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrWorkspaceBindingNotFound
		}
		return err
	}
	if bindingState != string(domain.BindingStateActive) {
		return fmt.Errorf("%w: binding state is %s, expected ACTIVE", ErrStateConflict, bindingState)
	}

	// 3. Canonicalize worker claim payload
	claimPayload := domain.WorkerClaimPayload{
		ClaimedFilesChanged: report.FilesChanged,
		Tests:               report.Tests,
		TextualClaims:       report.WorkerClaims,
		BuildStatus:         report.BuildStatus,
	}
	canonicalJSON, err := domain.CanonicalizeWorkerClaimPayload(claimPayload)
	if err != nil {
		return fmt.Errorf("store: failed to canonicalize worker claim payload: %w", err)
	}

	// 4. Insert into worker_claims
	claimID := "claim-" + attemptID
	insertClaimQuery := `
INSERT INTO worker_claims (
    claim_id, task_id, attempt_id, contract_id, reported_head_sha,
    payload_json, created_at_epoch_ms
) VALUES (?, ?, ?, ?, ?, ?, ?)
`
	_, err = tx.ExecContext(ctx, insertClaimQuery,
		claimID, taskID, attemptID, contractID, report.HeadSHA,
		canonicalJSON, epochMS,
	)
	if err != nil {
		return fmt.Errorf("store: failed to persist worker_claims: %w", err)
	}

	// 5. CAS advance attempt_workspace_bindings: ACTIVE -> RETAINED_FOR_VERIFICATION
	updateBindingQuery := `
UPDATE attempt_workspace_bindings
SET binding_state = 'RETAINED_FOR_VERIFICATION'
WHERE attempt_id = ? AND binding_state = 'ACTIVE'
`
	res, err := tx.ExecContext(ctx, updateBindingQuery, attemptID)
	if err != nil {
		return fmt.Errorf("store: failed to advance binding to RETAINED_FOR_VERIFICATION: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil || n != 1 {
		return fmt.Errorf("%w: workspace binding CAS lost", ErrStateConflict)
	}

	// 6. CAS advance tasks: RUNNING -> REPORT_READY
	updateTaskQuery := `
UPDATE tasks
SET state = 'REPORT_READY', updated_at = ?
WHERE task_id = ? AND state = 'RUNNING'
`
	res, err = tx.ExecContext(ctx, updateTaskQuery, formatTime(at), taskID)
	if err != nil {
		return fmt.Errorf("store: failed to advance task to REPORT_READY: %w", err)
	}
	n, err = res.RowsAffected()
	if err != nil || n != 1 {
		return fmt.Errorf("%w: task state CAS lost", ErrStateConflict)
	}

	// Commit Transaction A
	return tx.Commit()
}
