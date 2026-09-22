package ao

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// CreateWorkerSession spawns a minimal worker session in AO via POST /api/v1/sessions.
// Wire serialization is strictly {"projectId": projectID, "kind": "worker", "harness": harness}.
// Success requires HTTP 201 Created and validated response session identity.
func (c *Client) CreateWorkerSession(ctx context.Context, projectID string, harness string) (*CreateWorkerSessionResult, error) {
	if strings.TrimSpace(projectID) == "" {
		return nil, fmt.Errorf("%w: projectID cannot be empty or whitespace", ErrBadRequest)
	}
	if strings.TrimSpace(harness) == "" {
		return nil, fmt.Errorf("%w: harness cannot be empty or whitespace", ErrBadRequest)
	}

	payload := wireSpawnWorkerRequest{
		ProjectID: projectID,
		Kind:      "worker",
		Harness:   harness,
	}

	path := "/api/v1/sessions"
	var resp wireSpawnSessionResponse
	if err := c.post(ctx, path, http.StatusCreated, payload, &resp); err != nil {
		return nil, err
	}

	if resp.Session == nil {
		return nil, &ProtocolError{
			StatusCode: http.StatusCreated,
			Method:     http.MethodPost,
			Path:       path,
			Reason:     "spawn response is null or missing session object",
		}
	}
	if resp.Session.ID == "" {
		return nil, &ProtocolError{
			StatusCode: http.StatusCreated,
			Method:     http.MethodPost,
			Path:       path,
			Reason:     "spawn response contains empty session ID",
		}
	}
	if resp.Session.ProjectID != projectID {
		return nil, &ProtocolError{
			StatusCode: http.StatusCreated,
			Method:     http.MethodPost,
			Path:       path,
			Reason:     fmt.Sprintf("spawn response project ID %q does not match requested projectID %q", resp.Session.ProjectID, projectID),
		}
	}
	if resp.Session.Kind != "worker" {
		return nil, &ProtocolError{
			StatusCode: http.StatusCreated,
			Method:     http.MethodPost,
			Path:       path,
			Reason:     fmt.Sprintf("spawn response session kind %q is not worker", resp.Session.Kind),
		}
	}
	if resp.Session.Harness != harness {
		return nil, &ProtocolError{
			StatusCode: http.StatusCreated,
			Method:     http.MethodPost,
			Path:       path,
			Reason:     fmt.Sprintf("spawn response harness %q does not match requested harness %q", resp.Session.Harness, harness),
		}
	}
	if resp.PromptBytes == nil {
		return nil, &ProtocolError{
			StatusCode: http.StatusCreated,
			Method:     http.MethodPost,
			Path:       path,
			Reason:     "spawn response missing promptBytes field",
		}
	}
	if *resp.PromptBytes < 0 {
		return nil, &ProtocolError{
			StatusCode: http.StatusCreated,
			Method:     http.MethodPost,
			Path:       path,
			Reason:     fmt.Sprintf("spawn response promptBytes %d must not be negative", *resp.PromptBytes),
		}
	}
	if resp.SystemPromptBytes == nil {
		return nil, &ProtocolError{
			StatusCode: http.StatusCreated,
			Method:     http.MethodPost,
			Path:       path,
			Reason:     "spawn response missing systemPromptBytes field",
		}
	}
	if *resp.SystemPromptBytes < 0 {
		return nil, &ProtocolError{
			StatusCode: http.StatusCreated,
			Method:     http.MethodPost,
			Path:       path,
			Reason:     fmt.Sprintf("spawn response systemPromptBytes %d must not be negative", *resp.SystemPromptBytes),
		}
	}

	state, err := validateCanonicalActivityState(resp.Session.Activity.State, http.MethodPost, path)
	if err != nil {
		return nil, err
	}

	return toNormalizedCreateWorkerSessionResult(&resp, state), nil
}

// DispatchTaskContract dispatches an immutable task contract / instruction message to an AO session via POST /api/v1/sessions/{sessionId}/send.
// sessionID is safely URL-path escaped. message is transmitted without mutation.
// Success requires HTTP 200 OK and validated response identity.
func (c *Client) DispatchTaskContract(ctx context.Context, sessionID string, message string) (*DispatchTaskResult, error) {
	if strings.TrimSpace(sessionID) == "" {
		return nil, fmt.Errorf("%w: sessionID cannot be empty or whitespace", ErrBadRequest)
	}
	if message == "" {
		return nil, fmt.Errorf("%w: message cannot be empty", ErrBadRequest)
	}

	escapedID := url.PathEscape(sessionID)
	path := "/api/v1/sessions/" + escapedID + "/send"

	payload := wireSendSessionMessageRequest{
		Message: message,
	}

	var resp wireSendSessionMessageResponse
	if err := c.post(ctx, path, http.StatusOK, payload, &resp); err != nil {
		return nil, err
	}

	if !resp.OK {
		return nil, &ProtocolError{
			StatusCode: http.StatusOK,
			Method:     http.MethodPost,
			Path:       path,
			Reason:     "send response ok is false",
		}
	}
	if resp.SessionID == "" {
		return nil, &ProtocolError{
			StatusCode: http.StatusOK,
			Method:     http.MethodPost,
			Path:       path,
			Reason:     "send response contains empty session ID",
		}
	}
	if resp.SessionID != sessionID {
		return nil, &ProtocolError{
			StatusCode: http.StatusOK,
			Method:     http.MethodPost,
			Path:       path,
			Reason:     fmt.Sprintf("send response session ID %q does not match requested sessionID %q", resp.SessionID, sessionID),
		}
	}

	return &DispatchTaskResult{
		SessionID: resp.SessionID,
		Message:   resp.Message,
	}, nil
}

// StopWorker terminates an active AO worker session via POST /api/v1/sessions/{sessionId}/kill.
// sessionID is safely URL-path escaped. Request body is empty.
// Success requires HTTP 200 OK and validated response identity.
func (c *Client) StopWorker(ctx context.Context, sessionID string) (*StopWorkerResult, error) {
	if strings.TrimSpace(sessionID) == "" {
		return nil, fmt.Errorf("%w: sessionID cannot be empty or whitespace", ErrBadRequest)
	}

	escapedID := url.PathEscape(sessionID)
	path := "/api/v1/sessions/" + escapedID + "/kill"

	var resp wireKillSessionResponse
	if err := c.post(ctx, path, http.StatusOK, nil, &resp); err != nil {
		return nil, err
	}

	if !resp.OK {
		return nil, &ProtocolError{
			StatusCode: http.StatusOK,
			Method:     http.MethodPost,
			Path:       path,
			Reason:     "kill response ok is false",
		}
	}
	if resp.SessionID == "" {
		return nil, &ProtocolError{
			StatusCode: http.StatusOK,
			Method:     http.MethodPost,
			Path:       path,
			Reason:     "kill response contains empty session ID",
		}
	}
	if resp.SessionID != sessionID {
		return nil, &ProtocolError{
			StatusCode: http.StatusOK,
			Method:     http.MethodPost,
			Path:       path,
			Reason:     fmt.Sprintf("kill response session ID %q does not match requested sessionID %q", resp.SessionID, sessionID),
		}
	}

	return &StopWorkerResult{
		SessionID: resp.SessionID,
		Freed:     resp.Freed,
	}, nil
}

// ResumeWorker restores a terminated AO worker session via POST /api/v1/sessions/{sessionId}/restore.
// sessionID is safely URL-path escaped. Request body is empty.
// Success requires HTTP 200 OK, valid restoreMode (native, saved_prompt, fresh), and validated session snapshot.
func (c *Client) ResumeWorker(ctx context.Context, sessionID string) (*ResumeWorkerResult, error) {
	if strings.TrimSpace(sessionID) == "" {
		return nil, fmt.Errorf("%w: sessionID cannot be empty or whitespace", ErrBadRequest)
	}

	escapedID := url.PathEscape(sessionID)
	path := "/api/v1/sessions/" + escapedID + "/restore"

	var resp wireRestoreSessionResponse
	if err := c.post(ctx, path, http.StatusOK, nil, &resp); err != nil {
		return nil, err
	}

	if !resp.OK {
		return nil, &ProtocolError{
			StatusCode: http.StatusOK,
			Method:     http.MethodPost,
			Path:       path,
			Reason:     "restore response ok is false",
		}
	}
	if resp.SessionID == "" {
		return nil, &ProtocolError{
			StatusCode: http.StatusOK,
			Method:     http.MethodPost,
			Path:       path,
			Reason:     "restore response contains empty session ID",
		}
	}
	if resp.SessionID != sessionID {
		return nil, &ProtocolError{
			StatusCode: http.StatusOK,
			Method:     http.MethodPost,
			Path:       path,
			Reason:     fmt.Sprintf("restore response top-level session ID %q does not match requested sessionID %q", resp.SessionID, sessionID),
		}
	}

	var mode RestoreMode
	switch resp.RestoreMode {
	case string(RestoreModeNative):
		mode = RestoreModeNative
	case string(RestoreModeSavedPrompt):
		mode = RestoreModeSavedPrompt
	case string(RestoreModeFresh):
		mode = RestoreModeFresh
	default:
		return nil, &ProtocolError{
			StatusCode: http.StatusOK,
			Method:     http.MethodPost,
			Path:       path,
			Reason:     fmt.Sprintf("unknown restoreMode %q (must be native, saved_prompt, or fresh)", resp.RestoreMode),
		}
	}

	if resp.Session == nil {
		return nil, &ProtocolError{
			StatusCode: http.StatusOK,
			Method:     http.MethodPost,
			Path:       path,
			Reason:     "restore response is null or missing nested session object",
		}
	}
	if resp.Session.ID == "" {
		return nil, &ProtocolError{
			StatusCode: http.StatusOK,
			Method:     http.MethodPost,
			Path:       path,
			Reason:     "restore response contains empty nested session ID",
		}
	}
	if resp.Session.ID != sessionID {
		return nil, &ProtocolError{
			StatusCode: http.StatusOK,
			Method:     http.MethodPost,
			Path:       path,
			Reason:     fmt.Sprintf("restore response nested session ID %q does not match requested sessionID %q", resp.Session.ID, sessionID),
		}
	}

	state, err := validateCanonicalActivityState(resp.Session.Activity.State, http.MethodPost, path)
	if err != nil {
		return nil, err
	}

	return &ResumeWorkerResult{
		SessionID:   resp.SessionID,
		RestoreMode: mode,
		Session:     toNormalizedWorkerStatusFromView(resp.Session, state),
	}, nil
}
