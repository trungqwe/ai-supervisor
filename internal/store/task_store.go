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
// In accordance with the canonical lifecycle initial edge ([*] -> DRAFT),
// a new task must be created in the DRAFT state with current_attempt = 0.
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

	// Canonical initial edge enforcement: [*] -> DRAFT with current_attempt = 0
	if t.State != domain.StateDraft {
		return fmt.Errorf("%w: new task state must be DRAFT, got %q", ErrInvalidInitialTaskState, t.State)
	}
	if t.CurrentAttempt != 0 {
		return fmt.Errorf("%w: new task current_attempt must be 0, got %d", ErrInvalidInitialTaskState, t.CurrentAttempt)
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
// The READY -> DISPATCHED transition is exclusively owned by PrepareDispatch and is rejected here.
// When transitioning into active lane states (RUNNING or REVIEWING), the pair active-lane invariant is enforced.
// When transitioning into READY, governing contract preconditions are enforced inside the same transaction (Finding R3-001).
func (s *Store) TransitionTask(ctx context.Context, taskID string, expectedFrom, to domain.TaskState) error {
	// Reject generic READY -> DISPATCHED bypass (Finding R2-001)
	if expectedFrom == domain.StateReady && to == domain.StateDispatched {
		return ErrAtomicDispatchRequired
	}

	if err := workflow.Transition(expectedFrom, to); err != nil {
		return fmt.Errorf("store: canonical transition rejected: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: failed to begin transition transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Query pair_id and current state of taskID
	var pairID string
	var currentState string
	err = tx.QueryRowContext(ctx, "SELECT pair_id, state FROM tasks WHERE task_id = ?", taskID).Scan(&pairID, &currentState)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrTaskNotFound
		}
		return fmt.Errorf("store: failed to query task: %w", err)
	}
	if currentState != string(expectedFrom) {
		return fmt.Errorf("%w: task %q is in state %q, expected %q", ErrStateConflict, taskID, currentState, expectedFrom)
	}

	// 2. READY-entry contract preconditions inside same transaction (Finding R3-001)
	if to == domain.StateReady {
		var revNum int64
		var isImmutableInt int
		contractQuery := `
SELECT revision_number, is_immutable
FROM task_contracts
WHERE task_id = ?
ORDER BY revision_number DESC
LIMIT 1`
		err = tx.QueryRowContext(ctx, contractQuery, taskID).Scan(&revNum, &isImmutableInt)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("%w: task %q has no persisted contract", ErrReadyContractRequired, taskID)
			}
			return fmt.Errorf("store: failed to query task contract for ready check: %w", err)
		}

		isImmutable := (isImmutableInt == 1)

		switch expectedFrom {
		case domain.StateDraft:
			// Finding R4-001: DRAFT -> READY requires latest contract revision to be mutable (is_immutable = false).
			// Disallows reusing already-dispatched frozen contracts on human replanning flows (HUMAN_REQUIRED -> DRAFT -> READY).
			if isImmutable {
				return fmt.Errorf("%w: transition from DRAFT to READY requires mutable contract revision (latest rev %d is immutable)", ErrReadyContractRequired, revNum)
			}
		case domain.StateRevisionRequired:
			// REVISION_REQUIRED -> READY requires a NEW latest contract revision that is still is_immutable = false
			if isImmutable {
				return fmt.Errorf("%w: transition from REVISION_REQUIRED to READY requires new mutable contract revision (latest rev %d is immutable)", ErrReadyContractRequired, revNum)
			}
		case domain.StateFailed:
			// FAILED -> READY is retry-without-spec-change: requires latest contract is_immutable = true
			if !isImmutable {
				return fmt.Errorf("%w: retry from FAILED to READY requires immutable contract revision (latest rev %d is mutable)", ErrReadyContractRequired, revNum)
			}
		default:
			// Any other transition to READY (fail closed)
			return fmt.Errorf("%w: unsupported transition to READY from %q", ErrReadyContractRequired, expectedFrom)
		}
	}

	// 3. Pair active-lane invariant: exactly one task may be in DISPATCHED, RUNNING, or REVIEWING per Pair (Finding R2-006)
	if to == domain.StateRunning || to == domain.StateReviewing {
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
			return fmt.Errorf("%w: pair %q already has active task %q in state %q", ErrPairBusy, pairID, activeTaskID, activeState)
		} else if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("store: failed to check pair active lane: %w", err)
		}
	}

	// 4. Execute compare-and-set update
	now := formatTime(timeNow())
	query := `
UPDATE tasks
SET state = ?, updated_at = ?
WHERE task_id = ? AND state = ?
`

	res, err := tx.ExecContext(ctx, query, string(to), now, taskID, string(expectedFrom))
	if err != nil {
		return fmt.Errorf("store: failed to update task state: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("store: failed to get rows affected: %w", err)
	}
	if rows == 0 {
		var actualState string
		err := tx.QueryRowContext(ctx, "SELECT state FROM tasks WHERE task_id = ?", taskID).Scan(&actualState)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrTaskNotFound
			}
			return fmt.Errorf("store: failed to query task state: %w", err)
		}
		return fmt.Errorf("%w: task %q is in state %q, expected %q", ErrStateConflict, taskID, actualState, expectedFrom)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: failed to commit transition: %w", err)
	}

	return nil
}
