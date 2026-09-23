package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/trungqwe/ai-supervisor/internal/domain"
	"modernc.org/sqlite"
)

var timeNow = func() time.Time {
	return time.Now().UTC()
}

// DispatchBinding supplies the immutable execution snapshot for DISPATCH_BOUND.
type DispatchBinding struct {
	OperationID        string
	SessionID          string
	TerminalGeneration string
}

// GetTaskAttempt retrieves a TaskAttempt by attemptID.
func (s *Store) GetTaskAttempt(ctx context.Context, attemptID string) (domain.TaskAttempt, error) {
	query := `
SELECT attempt_id, attempt_number, task_id, contract_id, expected_report_path, started_at,
       ended_at, worker_report_raw, session_id, terminal_generation, recovery_disposition, quarantine_state
FROM task_attempts
WHERE attempt_id = ?
`

	var a domain.TaskAttempt
	var startedAtStr string
	var endedAtStr sql.NullString
	var reportRaw sql.NullString
	var sessionID, terminalGeneration, recoveryDisposition sql.NullString
	var quarantineState string

	err := s.db.QueryRowContext(ctx, query, attemptID).Scan(
		&a.AttemptID,
		&a.AttemptNumber,
		&a.TaskID,
		&a.ContractID,
		&a.ExpectedReportPath,
		&startedAtStr,
		&endedAtStr,
		&reportRaw,
		&sessionID,
		&terminalGeneration,
		&recoveryDisposition,
		&quarantineState,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.TaskAttempt{}, ErrAttemptNotFound
		}
		return domain.TaskAttempt{}, fmt.Errorf("store: failed to query task attempt: %w", err)
	}

	startedAt, err := parseTime(startedAtStr)
	if err != nil {
		return domain.TaskAttempt{}, fmt.Errorf("store: invalid started_at timestamp: %w", err)
	}
	a.StartedAt = startedAt

	if endedAtStr.Valid {
		endedAt, err := parseTime(endedAtStr.String)
		if err != nil {
			return domain.TaskAttempt{}, fmt.Errorf("store: invalid ended_at timestamp: %w", err)
		}
		a.EndedAt = &endedAt
	}

	if reportRaw.Valid {
		a.WorkerReportRaw = reportRaw.String
	}
	if sessionID.Valid {
		a.SessionID = &sessionID.String
	}
	if terminalGeneration.Valid {
		a.TerminalGeneration = &terminalGeneration.String
	}
	if recoveryDisposition.Valid {
		a.RecoveryDisposition = &recoveryDisposition.String
	}
	a.QuarantineState = domain.QuarantineState(quarantineState)

	return a, nil
}

// PrepareDispatch executes the atomic pre-dispatch transaction:
// 1. Validates that expectedReportPath strictly matches CanonicalExpectedReportPath(taskID, attemptID).
// 2. Conditionally updates Task state from READY to DISPATCHED and increments current_attempt by 1.
// 3. Reads the pair_id and enforces the pair active-lane invariant (at most one DISPATCHED/RUNNING/REVIEWING task per Pair).
// 4. Verifies contract existence, task ownership, and that the contract is the latest revision for the task.
// 5. Freezes the contract revision (is_immutable = 1) if not already frozen.
// 6. Inserts the TaskAttempt record.
// 7. Commits the transaction.
func (s *Store) PrepareDispatch(
	ctx context.Context,
	taskID string,
	contractID string,
	attemptID string,
	expectedReportPath string,
	startedAt time.Time,
) (domain.TaskAttempt, error) {
	return s.prepareDispatch(ctx, taskID, contractID, attemptID, expectedReportPath, startedAt, nil)
}

// PrepareBoundDispatch extends the P02 dispatch allocation transaction with the
// immutable ADR-016 session snapshot, dispatch operation, and audit event.
func (s *Store) PrepareBoundDispatch(
	ctx context.Context,
	taskID string,
	contractID string,
	attemptID string,
	expectedReportPath string,
	startedAt time.Time,
	binding DispatchBinding,
) (domain.TaskAttempt, error) {
	if strings.TrimSpace(binding.OperationID) == "" || strings.TrimSpace(binding.SessionID) == "" || strings.TrimSpace(binding.TerminalGeneration) == "" {
		return domain.TaskAttempt{}, errors.New("store: dispatch binding operation_id, session_id, and terminal_generation are required")
	}
	return s.prepareDispatch(ctx, taskID, contractID, attemptID, expectedReportPath, startedAt, &binding)
}

func (s *Store) prepareDispatch(
	ctx context.Context,
	taskID string,
	contractID string,
	attemptID string,
	expectedReportPath string,
	startedAt time.Time,
	binding *DispatchBinding,
) (domain.TaskAttempt, error) {
	// Finding R2-003: Canonical report path enforcement
	canonicalPath, err := CanonicalExpectedReportPath(taskID, attemptID)
	if err != nil {
		return domain.TaskAttempt{}, err
	}
	if expectedReportPath != canonicalPath {
		return domain.TaskAttempt{}, fmt.Errorf("%w: expected %q, got %q", ErrReportPathMismatch, canonicalPath, expectedReportPath)
	}

	now := startedAt.UTC()
	if now.IsZero() {
		now = timeNow()
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.TaskAttempt{}, fmt.Errorf("store: failed to begin dispatch transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Conditional compare-and-set state claim: READY -> DISPATCHED, current_attempt = current_attempt + 1
	claimQuery := `
UPDATE tasks
SET state = 'DISPATCHED', current_attempt = current_attempt + 1, updated_at = ?
WHERE task_id = ? AND state = 'READY'
`

	res, err := tx.ExecContext(ctx, claimQuery, formatTime(now), taskID)
	if err != nil {
		return domain.TaskAttempt{}, fmt.Errorf("store: failed to execute state claim: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return domain.TaskAttempt{}, fmt.Errorf("store: failed to read rows affected: %w", err)
	}
	if rows == 0 {
		var currentState string
		err := tx.QueryRowContext(ctx, "SELECT state FROM tasks WHERE task_id = ?", taskID).Scan(&currentState)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return domain.TaskAttempt{}, ErrTaskNotFound
			}
			return domain.TaskAttempt{}, fmt.Errorf("store: failed to query task state: %w", err)
		}
		return domain.TaskAttempt{}, fmt.Errorf("%w: task %q is in state %q, expected READY", ErrTaskNotReady, taskID, currentState)
	}

	// 2. Read pair_id and new attempt number
	var pairID string
	var allocatedAttemptNumber int
	err = tx.QueryRowContext(ctx, "SELECT pair_id, current_attempt FROM tasks WHERE task_id = ?", taskID).Scan(&pairID, &allocatedAttemptNumber)
	if err != nil {
		return domain.TaskAttempt{}, fmt.Errorf("store: failed to read updated task info: %w", err)
	}

	// 3. Pair active-lane invariant: at most one task in DISPATCHED, RUNNING, or REVIEWING per Pair (Finding R2-006)
	var activeTaskID, activeState string
	checkPairQuery := `
SELECT task_id, state
FROM tasks
WHERE pair_id = ?
  AND task_id != ?
  AND state IN ('DISPATCHED', 'RUNNING', 'REVIEWING')
LIMIT 1`
	err = tx.QueryRowContext(ctx, checkPairQuery, pairID, taskID).Scan(&activeTaskID, &activeState)
	if err == nil {
		return domain.TaskAttempt{}, fmt.Errorf("%w: pair %q already has active task %q in state %q", ErrPairBusy, pairID, activeTaskID, activeState)
	} else if !errors.Is(err, sql.ErrNoRows) {
		return domain.TaskAttempt{}, fmt.Errorf("store: failed to check pair active lane: %w", err)
	}

	// 4. Verify contract existence, ownership, and latest revision (Finding R2-004)
	var contractTaskID string
	var selectedRevision int
	var isImmutableInt int
	queryContract := `
SELECT task_id, revision_number, is_immutable
FROM task_contracts
WHERE contract_id = ?`
	err = tx.QueryRowContext(ctx, queryContract, contractID).Scan(&contractTaskID, &selectedRevision, &isImmutableInt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.TaskAttempt{}, ErrContractNotFound
		}
		return domain.TaskAttempt{}, fmt.Errorf("store: failed to verify contract: %w", err)
	}

	if contractTaskID != taskID {
		return domain.TaskAttempt{}, fmt.Errorf("%w: contract %q belongs to task %q, not %q", ErrContractNotOwned, contractID, contractTaskID, taskID)
	}

	var latestRevision int
	err = tx.QueryRowContext(ctx, "SELECT MAX(revision_number) FROM task_contracts WHERE task_id = ?", taskID).Scan(&latestRevision)
	if err != nil {
		return domain.TaskAttempt{}, fmt.Errorf("store: failed to query latest contract revision: %w", err)
	}
	if selectedRevision != latestRevision {
		return domain.TaskAttempt{}, fmt.Errorf("%w: contract revision %d is superseded by latest revision %d", ErrStaleContractRevision, selectedRevision, latestRevision)
	}

	// 5. Freeze contract if not already frozen (retries reuse already-frozen latest revision)
	if isImmutableInt == 0 {
		freezeQuery := `
UPDATE task_contracts
SET is_immutable = 1
WHERE contract_id = ? AND is_immutable = 0
`
		_, err = tx.ExecContext(ctx, freezeQuery, contractID)
		if err != nil {
			return domain.TaskAttempt{}, fmt.Errorf("store: failed to freeze contract: %w", err)
		}
	}

	// 6. Insert TaskAttempt, including immutable execution snapshot when bound.
	insertAttemptQuery := `
INSERT INTO task_attempts (
    attempt_id, attempt_number, task_id, contract_id, expected_report_path, started_at
) VALUES (?, ?, ?, ?, ?, ?)
`
	insertArgs := []any{attemptID, allocatedAttemptNumber, taskID, contractID, expectedReportPath, formatTime(now)}
	if binding != nil {
		var sessionID, terminalGeneration, quarantineState string
		err = tx.QueryRowContext(ctx, `
SELECT session_id, terminal_generation, quarantine_state FROM worker_sessions
WHERE pair_id = ?
`, pairID).Scan(&sessionID, &terminalGeneration, &quarantineState)
		if errors.Is(err, sql.ErrNoRows) {
			return domain.TaskAttempt{}, ErrWorkerSessionNotFound
		}
		if err != nil {
			return domain.TaskAttempt{}, fmt.Errorf("store: verify dispatch worker session: %w", err)
		}
		if sessionID != binding.SessionID || terminalGeneration != binding.TerminalGeneration {
			return domain.TaskAttempt{}, fmt.Errorf("%w: dispatch binding does not match current pair session", ErrWorkerSessionNotFound)
		}
		if quarantineState != string(domain.QuarantineClean) {
			return domain.TaskAttempt{}, fmt.Errorf("%w: pair %q session is %s", ErrQuarantinedExecution, pairID, quarantineState)
		}
		var blockedAttempts int
		err = tx.QueryRowContext(ctx, `
SELECT COUNT(*) FROM task_attempts
WHERE task_id = ? AND quarantine_state <> 'CLEAN'
`, taskID).Scan(&blockedAttempts)
		if err != nil {
			return domain.TaskAttempt{}, fmt.Errorf("store: verify prior attempt quarantine: %w", err)
		}
		if blockedAttempts != 0 {
			return domain.TaskAttempt{}, fmt.Errorf("%w: task %q has %d quarantined prior attempt(s)", ErrQuarantinedExecution, taskID, blockedAttempts)
		}
		var unresolvedProvisioning int
		err = tx.QueryRowContext(ctx, `
SELECT COUNT(*) FROM pair_provisioning_operations
WHERE pair_id = ? AND stage IN ('PROVISION_REQUESTED', 'PROVISION_FAILED')
`, pairID).Scan(&unresolvedProvisioning)
		if err != nil {
			return domain.TaskAttempt{}, fmt.Errorf("store: verify unresolved pair provisioning: %w", err)
		}
		if unresolvedProvisioning != 0 {
			return domain.TaskAttempt{}, fmt.Errorf("%w: pair %q has unresolved provisioning", ErrQuarantinedExecution, pairID)
		}
		insertAttemptQuery = `
INSERT INTO task_attempts (
    attempt_id, attempt_number, task_id, contract_id, expected_report_path, started_at,
    session_id, terminal_generation
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
`
		insertArgs = append(insertArgs, binding.SessionID, binding.TerminalGeneration)
	}

	_, err = tx.ExecContext(ctx, insertAttemptQuery, insertArgs...)
	if err != nil {
		var sqliteErr *sqlite.Error
		if errors.As(err, &sqliteErr) {
			if strings.Contains(sqliteErr.Error(), "UNIQUE constraint failed") {
				return domain.TaskAttempt{}, fmt.Errorf("%w: duplicate attempt identity", ErrDuplicateKey)
			}
			if strings.Contains(sqliteErr.Error(), "FOREIGN KEY constraint failed") {
				return domain.TaskAttempt{}, fmt.Errorf("%w: attempt contract binding violation", ErrForeignKeyViolation)
			}
		}
		return domain.TaskAttempt{}, fmt.Errorf("store: failed to insert task attempt: %w", err)
	}

	if binding != nil {
		_, err = tx.ExecContext(ctx, `
INSERT INTO dispatch_operations (
    operation_id, attempt_id, pair_id, task_id, session_id, terminal_generation,
    stage, requested_at
) VALUES (?, ?, ?, ?, ?, ?, 'DISPATCH_BOUND', ?)
`, binding.OperationID, attemptID, pairID, taskID, binding.SessionID, binding.TerminalGeneration, formatTime(now))
		if err != nil {
			return domain.TaskAttempt{}, mapLifecycleWriteError(err, "bound dispatch operation")
		}
		eventID, err := newAuditEventID()
		if err != nil {
			return domain.TaskAttempt{}, err
		}
		_, err = appendAuditEventTx(ctx, tx, domain.AuditEvent{
			EventID:    eventID,
			EventType:  domain.AuditTaskDispatchBound,
			Timestamp:  now,
			PairID:     pairID,
			TaskID:     taskID,
			ContractID: contractID,
			AttemptID:  attemptID,
			Actor:      "supervisor",
			Details: map[string]any{
				"session_id":          binding.SessionID,
				"terminal_generation": binding.TerminalGeneration,
			},
		})
		if err != nil {
			return domain.TaskAttempt{}, err
		}
	}

	// 7. Commit
	if err := tx.Commit(); err != nil {
		return domain.TaskAttempt{}, fmt.Errorf("store: failed to commit dispatch transaction: %w", err)
	}

	attempt := domain.TaskAttempt{
		AttemptID:          attemptID,
		AttemptNumber:      allocatedAttemptNumber,
		TaskID:             taskID,
		ContractID:         contractID,
		ExpectedReportPath: expectedReportPath,
		StartedAt:          now,
		QuarantineState:    domain.QuarantineClean,
	}
	if binding != nil {
		attempt.SessionID = &binding.SessionID
		attempt.TerminalGeneration = &binding.TerminalGeneration
	}
	return attempt, nil
}
