package ao

import (
	"context"
	"fmt"
	"strings"
)

// RegisterProject registers a local codebase path with AO via POST /api/v1/projects.
func (c *Client) RegisterProject(ctx context.Context, req RegisterProjectRequest) (*Project, error) {
	trimmedPath := strings.TrimSpace(req.Path)
	if trimmedPath == "" {
		return nil, fmt.Errorf("%w: project path cannot be empty", ErrBadRequest)
	}
	req.Path = trimmedPath

	var resp ProjectResponse
	if err := c.post(ctx, "/api/v1/projects", req, &resp); err != nil {
		return nil, fmt.Errorf("register project failed: %w", err)
	}
	return &resp.Project, nil
}

// GetProject retrieves project details from AO via GET /api/v1/projects/{id}.
func (c *Client) GetProject(ctx context.Context, projectID string) (*Project, error) {
	trimmedID := strings.TrimSpace(projectID)
	if trimmedID == "" {
		return nil, fmt.Errorf("%w: project ID cannot be empty", ErrBadRequest)
	}

	var resp GetProjectResponse
	path := "/api/v1/projects/" + trimmedID
	if err := c.get(ctx, path, &resp); err != nil {
		return nil, fmt.Errorf("get project failed: %w", err)
	}
	return &resp.Project, nil
}
