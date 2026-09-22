package ao

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// GetWorkerStatus queries GET /api/v1/sessions/{sessionId} to retrieve the authoritative read model of an AO session.
// sessionID is safely URL-path escaped.
// Activity states are strictly validated against canonical states (active, idle, waiting_input, blocked, exited).
// An unknown activity state causes fail-closed protocol rejection.
func (c *Client) GetWorkerStatus(ctx context.Context, sessionID string) (*WorkerStatus, error) {
	if strings.TrimSpace(sessionID) == "" {
		return nil, fmt.Errorf("%w: sessionID cannot be empty or whitespace", ErrBadRequest)
	}

	escapedID := url.PathEscape(sessionID)
	path := "/api/v1/sessions/" + escapedID

	var wire wireSessionResponse
	if err := c.get(ctx, path, http.StatusOK, &wire); err != nil {
		return nil, err
	}

	if wire.Session == nil {
		return nil, &ProtocolError{
			StatusCode: http.StatusOK,
			Method:     http.MethodGet,
			Path:       path,
			Reason:     "session response is null or missing session object",
		}
	}
	if wire.Session.ID == "" {
		return nil, &ProtocolError{
			StatusCode: http.StatusOK,
			Method:     http.MethodGet,
			Path:       path,
			Reason:     "session response contains empty session ID",
		}
	}
	if wire.Session.ID != sessionID {
		return nil, &ProtocolError{
			StatusCode: http.StatusOK,
			Method:     http.MethodGet,
			Path:       path,
			Reason:     fmt.Sprintf("returned session ID %q does not match requested sessionID %q", wire.Session.ID, sessionID),
		}
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
			Method:     http.MethodGet,
			Path:       path,
			Reason:     fmt.Sprintf("unknown activity state %q (must be active, idle, waiting_input, blocked, or exited)", wire.Session.Activity.State),
		}
	}

	return toNormalizedWorkerStatus(&wire, state), nil
}
