package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// RecoveryClassification classifies restart recovery candidates per NFR-003.
type RecoveryClassification string

const (
	ClassificationExternalReconciliationRequired RecoveryClassification = "EXTERNAL_RECONCILIATION_REQUIRED"
	ClassificationInconsistentPersistedState     RecoveryClassification = "INCONSISTENT_PERSISTED_STATE"
)

// RecoveryCandidate represents a task and attempt requiring restart reconciliation.
type RecoveryCandidate struct {
	TaskID         string                 `json:"task_id"`
	AttemptID      string                 `json:"attempt_id,omitempty"`
	AttemptNumber  int                    `json:"attempt_number,omitempty"`
	ContractID     string                 `json:"contract_id,omitempty"`
	Classification RecoveryClassification `json:"classification"`
}

// ClassifyRestartRecovery detects tasks in DISPATCHED state on daemon restart.
// For tasks with an OPEN matching attempt (ended_at IS NULL), it classifies them as EXTERNAL_RECONCILIATION_REQUIRED.
// If an attempt has ended (ended_at IS NOT NULL, including empty string) or no matching attempt exists, it classifies them as INCONSISTENT_PERSISTED_STATE.
func (s *Store) ClassifyRestartRecovery(ctx context.Context) ([]RecoveryCandidate, error) {
	queryTasks := `
SELECT task_id, current_attempt
FROM tasks
WHERE state = 'DISPATCHED'
ORDER BY task_id ASC
`

	rows, err := s.db.QueryContext(ctx, queryTasks)
	if err != nil {
		return nil, fmt.Errorf("store: failed to query dispatched tasks: %w", err)
	}
	defer rows.Close()

	type dispatchedTask struct {
		taskID         string
		currentAttempt int
	}
	var tasks []dispatchedTask

	for rows.Next() {
		var dt dispatchedTask
		if err := rows.Scan(&dt.taskID, &dt.currentAttempt); err != nil {
			return nil, fmt.Errorf("store: failed to scan task: %w", err)
		}
		tasks = append(tasks, dt)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: error iterating tasks: %w", err)
	}

	var candidates []RecoveryCandidate
	var hasInconsistent bool

	queryAttempt := `
SELECT attempt_id, contract_id, ended_at
FROM task_attempts
WHERE task_id = ? AND attempt_number = ?
`

	for _, dt := range tasks {
		var attemptID, contractID string
		var endedAtStr sql.NullString
		err := s.db.QueryRowContext(ctx, queryAttempt, dt.taskID, dt.currentAttempt).Scan(&attemptID, &contractID, &endedAtStr)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				hasInconsistent = true
				candidates = append(candidates, RecoveryCandidate{
					TaskID:         dt.taskID,
					AttemptNumber:  dt.currentAttempt,
					Classification: ClassificationInconsistentPersistedState,
				})
				continue
			}
			return nil, fmt.Errorf("store: failed to query attempt for task %q: %w", dt.taskID, err)
		}

		// Finding R3-003: Open attempt requirement (ended_at must be NULL in SQL, i.e., !endedAtStr.Valid)
		// ANY non-null ended_at (valid timestamp, empty string, etc.) for a DISPATCHED task is inconsistent
		if endedAtStr.Valid {
			hasInconsistent = true
			candidates = append(candidates, RecoveryCandidate{
				TaskID:         dt.taskID,
				AttemptID:      attemptID,
				AttemptNumber:  dt.currentAttempt,
				ContractID:     contractID,
				Classification: ClassificationInconsistentPersistedState,
			})
			continue
		}

		candidates = append(candidates, RecoveryCandidate{
			TaskID:         dt.taskID,
			AttemptID:      attemptID,
			AttemptNumber:  dt.currentAttempt,
			ContractID:     contractID,
			Classification: ClassificationExternalReconciliationRequired,
		})
	}

	if hasInconsistent {
		return candidates, fmt.Errorf("%w: one or more dispatched tasks have no open matching attempt", ErrInconsistentPersistedState)
	}

	return candidates, nil
}
