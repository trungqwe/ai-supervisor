package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/trungqwe/ai-supervisor/internal/domain"
	"github.com/trungqwe/ai-supervisor/internal/workflow"
	"modernc.org/sqlite"
)

// CreateTask inserts a new Task record into the store.
func (s *Store) CreateTask(ctx context.Context, t domain.Task) error {
	if strings.TrimSpace(t.TaskID) == "" {
		return errors.New("store: task_id must not be empty")
	}
	if strings.TrimSpace(t.PhaseID) == "" {
		return errors.New("store: phase_id must not be empty")
	}
	if strings.TrimSpace(t.PairID) == "" {
		return errors.New("store: pair_id must not be empty")
	}
	if !t.State.IsValid() {
		return fmt.Errorf("store: invalid task state %q", t.State)
	}
	if t.CurrentAttempt < 0 {
		return fmt.Errorf("store: current_attempt must be non-negative (got %d)", t.CurrentAttempt)
	}

	createdAt := t.CreatedAt
	if createdAt.IsZero() {
		createdAt = timeNow()
	}
	updatedAt := t.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = createdAt
	}

	query := `
INSERT INTO tasks (task_id, phase_id, pair_id, state, current_attempt, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
`

	_, err := s.db.ExecContext(ctx, query,
		t.TaskID,
		t.PhaseID,
		t.PairID,
		string(t.State),
		t.CurrentAttempt,
		formatTime(createdAt),
		formatTime(updatedAt),
	)
	if err != nil {
		var sqliteErr *sqlite.Error
		if errors.As(err, &sqliteErr) {
			if strings.Contains(sqliteErr.Error(), "FOREIGN KEY constraint failed") {
				return fmt.Errorf("%w: pair %q does not exist", ErrForeignKeyViolation, t.PairID)
			}
			if strings.Contains(sqliteErr.Error(), "UNIQUE constraint failed") {
				return fmt.Errorf("%w: task %q already exists", ErrDuplicateKey, t.TaskID)
			}
		}
		return fmt.Errorf("store: failed to insert task: %w", err)
	}

	return nil
}

// GetTask retrieves a Task by its taskID.
func (s *Store) GetTask(ctx context.Context, taskID string) (domain.Task, error) {
	query := `
SELECT task_id, phase_id, pair_id, state, current_attempt, created_at, updated_at
FROM tasks
WHERE task_id = ?
`

	var t domain.Task
	var stateStr string
	var createdAtStr, updatedAtStr string

	err := s.db.QueryRowContext(ctx, query, taskID).Scan(
		&t.TaskID,
		&t.PhaseID,
		&t.PairID,
		&stateStr,
		&t.CurrentAttempt,
		&createdAtStr,
		&updatedAtStr,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Task{}, ErrTaskNotFound
		}
		return domain.Task{}, fmt.Errorf("store: failed to query task: %w", err)
	}

	t.State = domain.TaskState(stateStr)
	createdAt, err := parseTime(createdAtStr)
	if err != nil {
		return domain.Task{}, fmt.Errorf("store: invalid created_at: %w", err)
	}
	t.CreatedAt = createdAt

	updatedAt, err := parseTime(updatedAtStr)
	if err != nil {
		return domain.Task{}, fmt.Errorf("store: invalid updated_at: %w", err)
	}
	t.UpdatedAt = updatedAt

	return t, nil
}

// TransitionTask executes a compare-and-set state transition on a Task using canonical workflow validation.
func (s *Store) TransitionTask(ctx context.Context, taskID string, expectedFrom, to domain.TaskState) error {
	if err := workflow.Transition(expectedFrom, to); err != nil {
		return fmt.Errorf("store: canonical transition rejected: %w", err)
	}

	now := formatTime(timeNow())

	query := `
UPDATE tasks
SET state = ?, updated_at = ?
WHERE task_id = ? AND state = ?
`

	res, err := s.db.ExecContext(ctx, query, string(to), now, taskID, string(expectedFrom))
	if err != nil {
		return fmt.Errorf("store: failed to update task state: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("store: failed to get rows affected: %w", err)
	}
	if rows == 0 {
		var actualState string
		err := s.db.QueryRowContext(ctx, "SELECT state FROM tasks WHERE task_id = ?", taskID).Scan(&actualState)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrTaskNotFound
		}
		return fmt.Errorf("%w: task %q is in state %q, expected %q", ErrStateConflict, taskID, actualState, expectedFrom)
	}

	return nil
}
