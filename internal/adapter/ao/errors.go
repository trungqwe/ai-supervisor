package ao

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

var (
	// ErrNonLoopbackURL is returned when the configured AO daemon base URL does not resolve to a local loopback address.
	ErrNonLoopbackURL = errors.New("ao base URL must be loopback (127.0.0.1, localhost, or [::1])")

	// ErrDaemonUnavailable is returned when the AO daemon cannot be reached or returns a 503/transient failure.
	ErrDaemonUnavailable = errors.New("ao daemon is unavailable")

	// ErrNotFound is returned when an AO resource cannot be found (HTTP 404).
	ErrNotFound = errors.New("resource not found")

	// ErrSessionNotFound is returned when a requested session does not exist in AO.
	ErrSessionNotFound = errors.New("session not found")

	// ErrProjectNotFound is returned when a requested project does not exist in AO.
	ErrProjectNotFound = errors.New("project not found")

	// ErrBadRequest is returned when AO rejects a request due to invalid input (HTTP 400).
	ErrBadRequest = errors.New("bad request")

	// ErrUnexpectedStatus is returned when AO returns an unhandled HTTP status code.
	ErrUnexpectedStatus = errors.New("unexpected HTTP status code")
)

// APIError represents the structured error response returned by the Agent Orchestrator REST API.
type APIError struct {
	StatusCode     int            `json:"-"`
	ErrorType      string         `json:"error"`
	Code           string         `json:"code"`
	Message        string         `json:"message"`
	RequestID      string         `json:"requestId,omitempty"`
	Details        map[string]any `json:"details,omitempty"`
	ReportingOwner string         `json:"reporting_owner,omitempty"`
}

// Error formats the API error into a human-readable string.
func (e *APIError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Code != "" {
		return fmt.Sprintf("ao api error (status %d, code %s): %s", e.StatusCode, e.Code, e.Message)
	}
	if e.Message != "" {
		return fmt.Sprintf("ao api error (status %d): %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("ao api error (status %d)", e.StatusCode)
}

// Is enables errors.Is matching against common sentinel errors.
func (e *APIError) Is(target error) bool {
	if e == nil {
		return false
	}
	switch target {
	case ErrNotFound:
		return e.StatusCode == http.StatusNotFound
	case ErrSessionNotFound:
		return e.StatusCode == http.StatusNotFound && (e.Code == "SESSION_NOT_FOUND" || strings.Contains(strings.ToLower(e.Message), "session"))
	case ErrProjectNotFound:
		return e.StatusCode == http.StatusNotFound && (e.Code == "PROJECT_NOT_FOUND" || e.Code == "PROJECT_FOLDER_MISSING" || strings.Contains(strings.ToLower(e.Message), "project"))
	case ErrBadRequest:
		return e.StatusCode == http.StatusBadRequest
	case ErrDaemonUnavailable:
		return e.StatusCode == http.StatusServiceUnavailable || e.StatusCode == http.StatusBadGateway || e.StatusCode == http.StatusGatewayTimeout
	default:
		return false
	}
}
