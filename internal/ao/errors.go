package ao

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

var (
	// ErrNilHTTPClient is returned when NewClient is invoked with a nil *http.Client.
	ErrNilHTTPClient = errors.New("httpClient must not be nil")

	// ErrNonLoopbackURL is returned when the configured AO daemon base URL does not satisfy the strict loopback specification.
	ErrNonLoopbackURL = errors.New("ao base URL must be an explicit loopback IP literal with port (e.g. http://127.0.0.1:3001 or http://[::1]:3001)")

	// ErrDaemonUnavailable is returned when the AO daemon cannot be reached or returns a 503/transient failure.
	ErrDaemonUnavailable = errors.New("ao daemon is unavailable")

	// ErrNotFound is returned when an AO resource cannot be found (HTTP 404).
	ErrNotFound = errors.New("resource not found")

	// ErrSessionNotFound is returned when a requested session does not exist in AO.
	ErrSessionNotFound = errors.New("session not found")

	// ErrProjectNotFound is returned when a requested project does not exist in AO.
	ErrProjectNotFound = errors.New("project not found")

	// ErrAgentNotFound is returned when a requested agent ID is not found in the readiness inventory.
	ErrAgentNotFound = errors.New("agent not found in readiness inventory")

	// ErrBadRequest is returned when a request is rejected due to invalid input (HTTP 400) or client-side validation failure.
	ErrBadRequest = errors.New("bad request")

	// ErrUnexpectedStatus is returned when AO returns an unhandled HTTP status code.
	ErrUnexpectedStatus = errors.New("unexpected HTTP status code")

	// ErrProtocolViolation is returned when an AO response violates expected structure, envelope format, or semantic constraints.
	ErrProtocolViolation = errors.New("ao protocol violation or incompatible payload")

	// ErrRedirectAttempted is returned when an AO endpoint attempts an HTTP redirect.
	ErrRedirectAttempted = errors.New("redirects from loopback base URL are forbidden")
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

// ProtocolError represents a sanitized failure when an AO response fails semantic or structural contract validation.
// It deliberately excludes raw response bodies to prevent leaking arbitrary upstream plaintext, HTML, or credentials.
type ProtocolError struct {
	StatusCode int
	Method     string
	Path       string
	Reason     string
}

// Error formats the sanitized protocol error.
func (e *ProtocolError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.StatusCode > 0 {
		return fmt.Sprintf("ao protocol error on %s %s (status %d): %s", e.Method, e.Path, e.StatusCode, e.Reason)
	}
	return fmt.Sprintf("ao protocol error on %s %s: %s", e.Method, e.Path, e.Reason)
}

// Is enables errors.Is matching against ErrProtocolViolation.
func (e *ProtocolError) Is(target error) bool {
	return target == ErrProtocolViolation
}

// TransportError wraps network and transport failures while preserving the underlying error cause.
type TransportError struct {
	Op  string
	URL string
	Err error
}

// Error formats the transport error.
func (e *TransportError) Error() string {
	if e == nil {
		return "<nil>"
	}
	return fmt.Sprintf("ao transport error on %s %s: %v", e.Op, e.URL, e.Err)
}

// Unwrap returns the underlying cause, allowing errors.Is and errors.As to match context cancellation/deadline errors.
func (e *TransportError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// Is enables matching against ErrDaemonUnavailable while preserving underlying unwrap chaining.
func (e *TransportError) Is(target error) bool {
	if target == ErrDaemonUnavailable {
		return true
	}
	if e == nil {
		return false
	}
	return errors.Is(e.Err, target)
}
