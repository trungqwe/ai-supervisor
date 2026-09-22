package ao

import (
	"context"
	"fmt"
	"strings"
)

// GetWorkerStatus queries GET /api/v1/sessions/{id} to retrieve the authoritative read model of a session.
// This implements the read-only session inspection requirement for TASK-P03-001.
// Note: Session mutation (spawn, send, kill, restore) is strictly reserved for TASK-P03-002.
func (c *Client) GetWorkerStatus(ctx context.Context, sessionID string) (*WorkerStatus, error) {
	trimmedID := strings.TrimSpace(sessionID)
	if trimmedID == "" {
		return nil, fmt.Errorf("%w: session ID cannot be empty", ErrBadRequest)
	}

	var resp SessionResponse
	path := "/api/v1/sessions/" + trimmedID
	if err := c.get(ctx, path, &resp); err != nil {
		return nil, fmt.Errorf("get worker status failed: %w", err)
	}
	return &resp.Session, nil
}
