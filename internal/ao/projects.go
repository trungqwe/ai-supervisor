package ao

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type registerProjectPayload struct {
	Path      string `json:"path"`
	ProjectID string `json:"projectId"`
}

type registerProjectResponse struct {
	Project Project `json:"project"`
}

type getProjectWireResponse struct {
	Status  string          `json:"status"`
	Project json.RawMessage `json:"project"`
}

type rawProjectOK struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Kind          string `json:"kind,omitempty"`
	Path          string `json:"path"`
	Repo          string `json:"repo,omitempty"`
	DefaultBranch string `json:"defaultBranch,omitempty"`
	Agent         string `json:"agent,omitempty"`
	FolderMissing bool   `json:"folderMissing,omitempty"`
}

type rawProjectDegraded struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Kind         string `json:"kind,omitempty"`
	Path         string `json:"path"`
	ResolveError string `json:"resolveError"`
}

// RegisterProject registers a local codebase path with AO via POST /api/v1/projects.
// Wire serialization is strictly {"path": rootPath, "projectId": projectID}.
func (c *Client) RegisterProject(ctx context.Context, projectID string, rootPath string) (*Project, error) {
	trimmedID := strings.TrimSpace(projectID)
	if trimmedID == "" {
		return nil, fmt.Errorf("%w: projectID cannot be empty", ErrBadRequest)
	}

	trimmedPath := strings.TrimSpace(rootPath)
	if trimmedPath == "" {
		return nil, fmt.Errorf("%w: rootPath cannot be empty", ErrBadRequest)
	}

	payload := registerProjectPayload{
		Path:      trimmedPath,
		ProjectID: trimmedID,
	}

	var resp registerProjectResponse
	statusCode, err := c.post(ctx, "/api/v1/projects", payload, &resp)
	if err != nil {
		return nil, err
	}
	if statusCode != http.StatusCreated {
		return nil, &ProtocolError{
			StatusCode: statusCode,
			Method:     "POST",
			Path:       "/api/v1/projects",
			Reason:     fmt.Sprintf("expected HTTP 201 Created, got %d", statusCode),
		}
	}

	resp.Project.Status = "ok"
	resp.Project.IsDegraded = false
	return &resp.Project, nil
}

// GetProject retrieves project details from AO via GET /api/v1/projects/{id}.
// projectID is safely URL-path escaped.
func (c *Client) GetProject(ctx context.Context, projectID string) (*Project, error) {
	trimmedID := strings.TrimSpace(projectID)
	if trimmedID == "" {
		return nil, fmt.Errorf("%w: projectID cannot be empty", ErrBadRequest)
	}

	escapedID := url.PathEscape(trimmedID)
	path := "/api/v1/projects/" + escapedID

	var wire getProjectWireResponse
	if err := c.get(ctx, path, &wire); err != nil {
		return nil, err
	}

	switch wire.Status {
	case "ok":
		var p rawProjectOK
		if err := json.Unmarshal(wire.Project, &p); err != nil {
			return nil, &ProtocolError{
				StatusCode: http.StatusOK,
				Method:     "GET",
				Path:       path,
				Reason:     "failed to decode healthy project JSON",
			}
		}
		return &Project{
			ID:            p.ID,
			Name:          p.Name,
			Kind:          p.Kind,
			Path:          p.Path,
			Repo:          p.Repo,
			DefaultBranch: p.DefaultBranch,
			Agent:         p.Agent,
			FolderMissing: p.FolderMissing,
			Status:        "ok",
			IsDegraded:    false,
		}, nil

	case "degraded":
		var p rawProjectDegraded
		if err := json.Unmarshal(wire.Project, &p); err != nil {
			return nil, &ProtocolError{
				StatusCode: http.StatusOK,
				Method:     "GET",
				Path:       path,
				Reason:     "failed to decode degraded project JSON",
			}
		}
		return &Project{
			ID:           p.ID,
			Name:         p.Name,
			Kind:         p.Kind,
			Path:         p.Path,
			Status:       "degraded",
			IsDegraded:   true,
			ResolveError: p.ResolveError,
		}, nil

	default:
		return nil, &ProtocolError{
			StatusCode: http.StatusOK,
			Method:     "GET",
			Path:       path,
			Reason:     fmt.Sprintf("unknown project status discriminator %q", wire.Status),
		}
	}
}
