package ao

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type rawSessionWireResponse struct {
	Session struct {
		ID            string `json:"id"`
		ProjectID     string `json:"projectId,omitempty"`
		Status        string `json:"status"`
		DisplayStatus string `json:"displayStatus,omitempty"`
		IsTerminated  bool   `json:"isTerminated"`
		Activity      struct {
			State          string    `json:"state"`
			LastActivityAt time.Time `json:"lastActivityAt"`
		} `json:"activity"`
		Harness            string `json:"harness,omitempty"`
		Branch             string `json:"branch,omitempty"`
		Model              string `json:"model,omitempty"`
		TerminalGeneration string `json:"terminalGeneration,omitempty"`
		PreviewURL         string `json:"previewUrl,omitempty"`
	} `json:"session"`
}

// GetWorkerStatus queries GET /api/v1/sessions/{sessionId} to retrieve the authoritative read model of an AO session.
// sessionID is safely URL-path escaped.
// Activity states are strictly validated against canonical states (active, idle, waiting_input, blocked, exited).
// An unknown activity state causes fail-closed protocol rejection.
func (c *Client) GetWorkerStatus(ctx context.Context, sessionID string) (*WorkerStatus, error) {
	trimmedID := strings.TrimSpace(sessionID)
	if trimmedID == "" {
		return nil, fmt.Errorf("%w: sessionID cannot be empty", ErrBadRequest)
	}

	escapedID := url.PathEscape(trimmedID)
	path := "/api/v1/sessions/" + escapedID

	var wire rawSessionWireResponse
	if err := c.get(ctx, path, &wire); err != nil {
		return nil, err
	}

	// Validate activity state fail-closed against canonical specification
	var state ActivityState
	switch wire.Session.Activity.State {
	case string(ActivityStateActive):
		state = ActivityStateActive
	case string(ActivityStateIdle):
		state = ActivityStateIdle
	case string(ActivityStateWaitingInput):
		state = ActivityStateWaitingInput
	case string(ActivityStateBlocked):
		state = ActivityStateBlocked
	case string(ActivityStateExited):
		state = ActivityStateExited
	default:
		return nil, &ProtocolError{
			StatusCode: http.StatusOK,
			Method:     "GET",
			Path:       path,
			Reason:     fmt.Sprintf("unknown activity state %q (must be active, idle, waiting_input, blocked, or exited)", wire.Session.Activity.State),
		}
	}

	return &WorkerStatus{
		ID:            wire.Session.ID,
		ProjectID:     wire.Session.ProjectID,
		Status:        wire.Session.Status,
		DisplayStatus: wire.Session.DisplayStatus,
		IsTerminated:  wire.Session.IsTerminated,
		Activity: ActivitySnapshot{
			State:          state,
			LastActivityAt: wire.Session.Activity.LastActivityAt,
		},
		Harness:            wire.Session.Harness,
		Branch:             wire.Session.Branch,
		Model:              wire.Session.Model,
		TerminalGeneration: wire.Session.TerminalGeneration,
		PreviewURL:         wire.Session.PreviewURL,
	}, nil
}
