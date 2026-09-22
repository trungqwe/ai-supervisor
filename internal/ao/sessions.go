package ao

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// validateCanonicalActivityState validates raw state string fail-closed against canonical specification.
// statusCode preserves the actual HTTP response status observed on the wire.
func validateCanonicalActivityState(rawState string, statusCode int, method string, path string) (ActivityState, error) {
	switch rawState {
	case string(ActivityStateActive):
		return ActivityStateActive, nil
	case string(ActivityStateIdle):
		return ActivityStateIdle, nil
	case string(ActivityStateWaitingInput):
		return ActivityStateWaitingInput, nil
	case string(ActivityStateBlocked):
		return ActivityStateBlocked, nil
	case string(ActivityStateExited):
		return ActivityStateExited, nil
	default:
		return "", &ProtocolError{
			StatusCode: statusCode,
			Method:     method,
			Path:       path,
			Reason:     fmt.Sprintf("unknown activity state %q (must be active, idle, waiting_input, blocked, or exited)", rawState),
		}
	}
}

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

	state, err := validateCanonicalActivityState(wire.Session.Activity.State, http.StatusOK, http.MethodGet, path)
	if err != nil {
		return nil, err
	}

	return toNormalizedWorkerStatus(&wire, state), nil
}
