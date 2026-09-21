package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/trungqwe/ai-supervisor/internal/domain"
	"modernc.org/sqlite"
)

// CreatePair inserts a new Pair record into the store.
// In accordance with the 1:1 project-pair invariant, only one pair is allowed per project.
func (s *Store) CreatePair(ctx context.Context, p domain.Pair) error {
	if strings.TrimSpace(p.PairID) == "" {
		return errors.New("store: pair_id must not be empty")
	}
	if strings.TrimSpace(p.ProjectID) == "" {
		return errors.New("store: project_id must not be empty")
	}
	if strings.TrimSpace(p.CurrentPhaseID) == "" {
		return errors.New("store: current_phase_id must not be empty")
	}
	if strings.TrimSpace(p.State) == "" {
		return errors.New("store: state must not be empty")
	}

	createdAt := p.CreatedAt
	if createdAt.IsZero() {
		createdAt = timeNow()
	}

	var activeTaskID *string
	if strings.TrimSpace(p.ActiveTaskID) != "" {
		trimmed := strings.TrimSpace(p.ActiveTaskID)
		activeTaskID = &trimmed
	}

	query := `
INSERT INTO pairs (pair_id, project_id, current_phase_id, active_task_id, state, created_at)
VALUES (?, ?, ?, ?, ?, ?)
`

	_, err := s.db.ExecContext(ctx, query,
		p.PairID,
		p.ProjectID,
		p.CurrentPhaseID,
		activeTaskID,
		p.State,
		formatTime(createdAt),
	)
	if err != nil {
		var sqliteErr *sqlite.Error
		if errors.As(err, &sqliteErr) {
			if strings.Contains(sqliteErr.Error(), "FOREIGN KEY constraint failed") {
				return fmt.Errorf("%w: project %q does not exist", ErrForeignKeyViolation, p.ProjectID)
			}
			if strings.Contains(sqliteErr.Error(), "UNIQUE constraint failed") {
				return fmt.Errorf("%w: pair %q or project %q already bound", ErrDuplicateKey, p.PairID, p.ProjectID)
			}
		}
		return fmt.Errorf("store: failed to insert pair: %w", err)
	}

	return nil
}

// GetPair retrieves a Pair by its pairID.
func (s *Store) GetPair(ctx context.Context, pairID string) (domain.Pair, error) {
	query := `
SELECT pair_id, project_id, current_phase_id, active_task_id, state, created_at
FROM pairs
WHERE pair_id = ?
`

	var p domain.Pair
	var activeTaskID sql.NullString
	var createdAtStr string

	err := s.db.QueryRowContext(ctx, query, pairID).Scan(
		&p.PairID,
		&p.ProjectID,
		&p.CurrentPhaseID,
		&activeTaskID,
		&p.State,
		&createdAtStr,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Pair{}, ErrPairNotFound
		}
		return domain.Pair{}, fmt.Errorf("store: failed to query pair: %w", err)
	}

	if activeTaskID.Valid {
		p.ActiveTaskID = activeTaskID.String
	}

	createdAt, err := parseTime(createdAtStr)
	if err != nil {
		return domain.Pair{}, fmt.Errorf("store: invalid created_at timestamp: %w", err)
	}
	p.CreatedAt = createdAt

	return p, nil
}
