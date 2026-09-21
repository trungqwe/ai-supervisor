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

// GetTaskAttempt retrieves a TaskAttempt by attemptID.
func (s *Store) GetTaskAttempt(ctx context.Context, attemptID string) (domain.TaskAttempt, error) {
	query := `
SELECT attempt_id, attempt_number, task_id, contract_id, expected_report_path, started_at, ended_at, worker_report_raw
FROM task_attempts
WHERE attempt_id = ?
`

	var a domain.TaskAttempt
	var startedAtStr string
	var endedAtStr sql.NullString
	var reportRaw sql.NullString

	err := s.db.QueryRowContext(ctx, query, attemptID).Scan(
		&a.AttemptID,
		&a.AttemptNumber,
		&a.TaskID,
		&a.ContractID,
		&a.ExpectedReportPath,
		&startedAtStr,
		&endedAtStr,
		&reportRaw,
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

	return a, nil
}

// PrepareDispatch executes an atomic pre-dispatch transaction:
// 1. Conditionally updates Task state from READY to DISPATCHED and increments current_attempt by 1.
// 2. Reads the allocated attempt_number.
// 3. Verifies that contractID exists and belongs to taskID.
// 4. Freezes the contract revision (is_immutable = 1).
// 5. Inserts the TaskAttempt record.
// 6. Commits the transaction.
func (s *Store) PrepareDispatch(
	ctx context.Context,
	taskID string,
	contractID string,
	attemptID string,
	expectedReportPath string,
	startedAt time.Time,
) (domain.TaskAttempt, error) {
	if strings.TrimSpace(taskID) == "" {
		return domain.TaskAttempt{}, errors.New("store: taskID must not be empty")
	}
	if strings.TrimSpace(contractID) == "" {
		return domain.TaskAttempt{}, errors.New("store: contractID must not be empty")
	}
	if strings.TrimSpace(attemptID) == "" {
		return domain.TaskAttempt{}, errors.New("store: attemptID must not be empty")
	}
	if strings.TrimSpace(expectedReportPath) == "" {
		return domain.TaskAttempt{}, errors.New("store: expectedReportPath must not be empty")
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
		if errors.Is(err, sql.ErrNoRows) {
			return domain.TaskAttempt{}, ErrTaskNotFound
		}
		return domain.TaskAttempt{}, fmt.Errorf("%w: task %q is in state %q, expected READY", ErrTaskNotReady, taskID, currentState)
	}

	// 2. Read new attempt number
	var allocatedAttemptNumber int
	err = tx.QueryRowContext(ctx, "SELECT current_attempt FROM tasks WHERE task_id = ?", taskID).Scan(&allocatedAttemptNumber)
	if err != nil {
		return domain.TaskAttempt{}, fmt.Errorf("store: failed to read updated attempt number: %w", err)
	}

	// 3. Verify contract existence and ownership
	var contractTaskID string
	var isImmutableInt int
	err = tx.QueryRowContext(ctx, "SELECT task_id, is_immutable FROM task_contracts WHERE contract_id = ?", contractID).Scan(&contractTaskID, &isImmutableInt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.TaskAttempt{}, ErrContractNotFound
		}
		return domain.TaskAttempt{}, fmt.Errorf("store: failed to verify contract: %w", err)
	}

	if contractTaskID != taskID {
		return domain.TaskAttempt{}, fmt.Errorf("%w: contract %q belongs to task %q, not %q", ErrContractNotOwned, contractID, contractTaskID, taskID)
	}

	// 4. Freeze contract if not already frozen
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

	// 5. Insert TaskAttempt
	insertAttemptQuery := `
INSERT INTO task_attempts (
    attempt_id, attempt_number, task_id, contract_id, expected_report_path, started_at
) VALUES (?, ?, ?, ?, ?, ?)
`

	_, err = tx.ExecContext(ctx, insertAttemptQuery,
		attemptID,
		allocatedAttemptNumber,
		taskID,
		contractID,
		expectedReportPath,
		formatTime(now),
	)
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

	// 6. Commit
	if err := tx.Commit(); err != nil {
		return domain.TaskAttempt{}, fmt.Errorf("store: failed to commit dispatch transaction: %w", err)
	}

	return domain.TaskAttempt{
		AttemptID:          attemptID,
		AttemptNumber:      allocatedAttemptNumber,
		TaskID:             taskID,
		ContractID:         contractID,
		ExpectedReportPath: expectedReportPath,
		StartedAt:          now,
	}, nil
}
