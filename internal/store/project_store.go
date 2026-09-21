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

// CreateProject inserts a new Project record into the store.
func (s *Store) CreateProject(ctx context.Context, p domain.Project) error {
	if strings.TrimSpace(p.ProjectID) == "" {
		return errors.New("store: project_id must not be empty")
	}
	if strings.TrimSpace(p.Name) == "" {
		return errors.New("store: project name must not be empty")
	}
	if strings.TrimSpace(p.RootPath) == "" {
		return errors.New("store: root_path must not be empty")
	}

	registeredAt := p.RegisteredAt
	if registeredAt.IsZero() {
		registeredAt = timeNow()
	}

	query := `
INSERT INTO projects (project_id, name, root_path, repo_url, registered_at)
VALUES (?, ?, ?, ?, ?)
`

	var repoURL *string
	if strings.TrimSpace(p.RepoURL) != "" {
		trimmed := strings.TrimSpace(p.RepoURL)
		repoURL = &trimmed
	}

	_, err := s.db.ExecContext(ctx, query,
		p.ProjectID,
		p.Name,
		p.RootPath,
		repoURL,
		formatTime(registeredAt),
	)
	if err != nil {
		var sqliteErr *sqlite.Error
		if errors.As(err, &sqliteErr) && strings.Contains(sqliteErr.Error(), "UNIQUE constraint failed") {
			return fmt.Errorf("%w: project %q already exists", ErrDuplicateKey, p.ProjectID)
		}
		return fmt.Errorf("store: failed to insert project: %w", err)
	}

	return nil
}

// GetProject retrieves a Project by its projectID.
func (s *Store) GetProject(ctx context.Context, projectID string) (domain.Project, error) {
	query := `
SELECT project_id, name, root_path, repo_url, registered_at
FROM projects
WHERE project_id = ?
`

	var p domain.Project
	var repoURL sql.NullString
	var regAtStr string

	err := s.db.QueryRowContext(ctx, query, projectID).Scan(
		&p.ProjectID,
		&p.Name,
		&p.RootPath,
		&repoURL,
		&regAtStr,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Project{}, ErrProjectNotFound
		}
		return domain.Project{}, fmt.Errorf("store: failed to query project: %w", err)
	}

	if repoURL.Valid {
		p.RepoURL = repoURL.String
	}

	regAt, err := parseTime(regAtStr)
	if err != nil {
		return domain.Project{}, fmt.Errorf("store: invalid registered_at timestamp: %w", err)
	}
	p.RegisteredAt = regAt

	return p, nil
}
