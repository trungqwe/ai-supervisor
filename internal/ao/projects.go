package ao

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// RegisterProject registers a local codebase path with AO via POST /api/v1/projects.
// Wire serialization is strictly {"path": rootPath, "projectId": projectID}.
// Success requires HTTP 201 and non-empty matching project ID.
func (c *Client) RegisterProject(ctx context.Context, projectID string, rootPath string) (*Project, error) {
	if strings.TrimSpace(projectID) == "" {
		return nil, fmt.Errorf("%w: projectID cannot be empty or whitespace", ErrBadRequest)
	}
	if strings.TrimSpace(rootPath) == "" {
		return nil, fmt.Errorf("%w: rootPath cannot be empty or whitespace", ErrBadRequest)
	}

	payload := wireRegisterProjectRequest{
		Path:      rootPath,
		ProjectID: projectID,
	}

	var resp wireRegisterProjectResponse
	if err := c.post(ctx, "/api/v1/projects", http.StatusCreated, payload, &resp); err != nil {
		return nil, err
	}

	if resp.Project == nil {
		return nil, &ProtocolError{
			StatusCode: http.StatusCreated,
			Method:     http.MethodPost,
			Path:       "/api/v1/projects",
			Reason:     "registered project response is null or missing project object",
		}
	}
	if resp.Project.ID == "" {
		return nil, &ProtocolError{
			StatusCode: http.StatusCreated,
			Method:     http.MethodPost,
			Path:       "/api/v1/projects",
			Reason:     "registered project response contains empty project ID",
		}
	}
	if resp.Project.ID != projectID {
		return nil, &ProtocolError{
			StatusCode: http.StatusCreated,
			Method:     http.MethodPost,
			Path:       "/api/v1/projects",
			Reason:     fmt.Sprintf("registered project ID %q does not match requested projectID %q", resp.Project.ID, projectID),
		}
	}

	return toNormalizedProjectFromRegister(resp.Project), nil
}

// GetProject retrieves project details from AO via GET /api/v1/projects/{id}.
// projectID is safely URL-path escaped.
// Success requires HTTP 200 and non-empty matching project ID.
func (c *Client) GetProject(ctx context.Context, projectID string) (*Project, error) {
	if strings.TrimSpace(projectID) == "" {
		return nil, fmt.Errorf("%w: projectID cannot be empty or whitespace", ErrBadRequest)
	}

	escapedID := url.PathEscape(projectID)
	path := "/api/v1/projects/" + escapedID

	var wire wireGetProjectResponse
	if err := c.get(ctx, path, http.StatusOK, &wire); err != nil {
		return nil, err
	}

	if len(wire.Project) == 0 || string(wire.Project) == "null" || string(wire.Project) == "{}" {
		return nil, &ProtocolError{
			StatusCode: http.StatusOK,
			Method:     http.MethodGet,
			Path:       path,
			Reason:     "project object is null, empty, or missing in response",
		}
	}

	switch wire.Status {
	case "ok":
		var p wireProjectOK
		if err := json.Unmarshal(wire.Project, &p); err != nil {
			return nil, &ProtocolError{
				StatusCode: http.StatusOK,
				Method:     http.MethodGet,
				Path:       path,
				Reason:     "failed to decode healthy project JSON",
			}
		}
		if p.ID == "" {
			return nil, &ProtocolError{
				StatusCode: http.StatusOK,
				Method:     http.MethodGet,
				Path:       path,
				Reason:     "healthy project object contains empty project ID",
			}
		}
		if p.ID != projectID {
			return nil, &ProtocolError{
				StatusCode: http.StatusOK,
				Method:     http.MethodGet,
				Path:       path,
				Reason:     fmt.Sprintf("healthy project ID %q does not match requested projectID %q", p.ID, projectID),
			}
		}
		return toNormalizedProjectOK(&p), nil

	case "degraded":
		var p wireProjectDegraded
		if err := json.Unmarshal(wire.Project, &p); err != nil {
			return nil, &ProtocolError{
				StatusCode: http.StatusOK,
				Method:     http.MethodGet,
				Path:       path,
				Reason:     "failed to decode degraded project JSON",
			}
		}
		if p.ID == "" {
			return nil, &ProtocolError{
				StatusCode: http.StatusOK,
				Method:     http.MethodGet,
				Path:       path,
				Reason:     "degraded project object contains empty project ID",
			}
		}
		if p.ID != projectID {
			return nil, &ProtocolError{
				StatusCode: http.StatusOK,
				Method:     http.MethodGet,
				Path:       path,
				Reason:     fmt.Sprintf("degraded project ID %q does not match requested projectID %q", p.ID, projectID),
			}
		}
		if p.ResolveError == "" {
			return nil, &ProtocolError{
				StatusCode: http.StatusOK,
				Method:     http.MethodGet,
				Path:       path,
				Reason:     "degraded project object is missing required resolveError",
			}
		}
		return toNormalizedProjectDegraded(&p), nil

	default:
		return nil, &ProtocolError{
			StatusCode: http.StatusOK,
			Method:     http.MethodGet,
			Path:       path,
			Reason:     fmt.Sprintf("unknown project status discriminator %q", wire.Status),
		}
	}
}
