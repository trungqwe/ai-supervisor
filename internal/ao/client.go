package ao

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	maxErrorBodyBytes = 64 * 1024
)

// Client provides the low-level HTTP transport client to the local Untrivial Agent Orchestrator REST daemon.
type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
}

// NewClient constructs an AO Client validating that baseURL satisfies strict loopback requirements.
// A non-nil httpClient must be explicitly provided by the caller; no implicit default client or timeout is created.
func NewClient(rawBaseURL string, httpClient *http.Client) (*Client, error) {
	if httpClient == nil {
		return nil, fmt.Errorf("%w: %w", ErrBadRequest, ErrNilHTTPClient)
	}

	trimmed := strings.TrimSpace(rawBaseURL)
	if trimmed == "" {
		return nil, fmt.Errorf("%w: base URL cannot be empty", ErrBadRequest)
	}

	parsedURL, err := url.Parse(trimmed)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid URL format: %v", ErrNonLoopbackURL, err)
	}

	// Scheme must be http only
	if parsedURL.Scheme != "http" {
		return nil, fmt.Errorf("%w: scheme %q forbidden (must be http only)", ErrNonLoopbackURL, parsedURL.Scheme)
	}

	// Userinfo forbidden
	if parsedURL.User != nil {
		return nil, fmt.Errorf("%w: userinfo in base URL is forbidden", ErrNonLoopbackURL)
	}

	// Query parameters forbidden
	if parsedURL.RawQuery != "" {
		return nil, fmt.Errorf("%w: query string in base URL is forbidden", ErrNonLoopbackURL)
	}

	// Fragment forbidden
	if parsedURL.Fragment != "" {
		return nil, fmt.Errorf("%w: fragment in base URL is forbidden", ErrNonLoopbackURL)
	}

	// Base path must be empty or exactly "/"
	if parsedURL.Path != "" && parsedURL.Path != "/" {
		return nil, fmt.Errorf("%w: non-root base path %q is forbidden", ErrNonLoopbackURL, parsedURL.Path)
	}

	// Port required and IP literal required
	hostStr := parsedURL.Host
	host, portStr, err := net.SplitHostPort(hostStr)
	if err != nil {
		return nil, fmt.Errorf("%w: explicit port required in host %q: %v", ErrNonLoopbackURL, hostStr, err)
	}

	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 || port > 65535 {
		return nil, fmt.Errorf("%w: invalid port %q (must be between 1 and 65535)", ErrNonLoopbackURL, portStr)
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return nil, fmt.Errorf("%w: host %q is not an IP literal (hostnames forbidden)", ErrNonLoopbackURL, host)
	}

	if !ip.IsLoopback() {
		return nil, fmt.Errorf("%w: IP %q is not a loopback address", ErrNonLoopbackURL, host)
	}

	// Normalize root URL without trailing slash
	normalizedURL, _ := url.Parse(fmt.Sprintf("http://%s", net.JoinHostPort(host, portStr)))

	// Shallow-copy caller's http.Client so we do not mutate caller's client,
	// while configuring adapter-local redirect rejection policy.
	adapterClient := *httpClient
	adapterClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return fmt.Errorf("%w: redirect attempted to %s", ErrRedirectAttempted, req.URL.String())
	}

	return &Client{
		baseURL:    normalizedURL,
		httpClient: &adapterClient,
	}, nil
}

// BaseURL returns the validated normalized base URL string.
func (c *Client) BaseURL() string {
	return c.baseURL.String()
}

// buildURL joins a relative path to the base URL.
func (c *Client) buildURL(relPath string) string {
	return strings.TrimRight(c.baseURL.String(), "/") + "/" + strings.TrimLeft(relPath, "/")
}

// get performs an HTTP GET request requiring exact expectedStatus and decodes the JSON response into wireTarget.
func (c *Client) get(ctx context.Context, relPath string, expectedStatus int, wireTarget any) error {
	reqURL := c.buildURL(relPath)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return fmt.Errorf("create GET request failed: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	return c.do(req, expectedStatus, wireTarget)
}

// getRaw performs an HTTP GET request requiring exact expectedStatus and returns the raw response body.
func (c *Client) getRaw(ctx context.Context, relPath string, expectedStatus int, acceptHeader string) (string, error) {
	reqURL := c.buildURL(relPath)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return "", fmt.Errorf("create raw GET request failed: %w", err)
	}
	if acceptHeader != "" {
		req.Header.Set("Accept", acceptHeader)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", &TransportError{Op: req.Method, URL: req.URL.String(), Err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode != expectedStatus {
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return "", c.decodeError(req, resp)
		}
		return "", &ProtocolError{
			StatusCode: resp.StatusCode,
			Method:     req.Method,
			Path:       req.URL.Path,
			Reason:     fmt.Sprintf("expected HTTP %d, got %d", expectedStatus, resp.StatusCode),
		}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read raw body failed: %w", err)
	}
	return string(body), nil
}

// post performs an HTTP POST request requiring exact expectedStatus and decodes the JSON response into wireTarget.
func (c *Client) post(ctx context.Context, relPath string, expectedStatus int, body any, wireTarget any) error {
	reqURL := c.buildURL(relPath)

	var bodyReader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body failed: %w", err)
		}
		bodyReader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bodyReader)
	if err != nil {
		return fmt.Errorf("create POST request failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/json")

	return c.do(req, expectedStatus, wireTarget)
}

// do dispatches an HTTP request, checks for exact expectedStatus, and decodes the wireTarget.
func (c *Client) do(req *http.Request, expectedStatus int, wireTarget any) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return &TransportError{Op: req.Method, URL: req.URL.String(), Err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode != expectedStatus {
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return c.decodeError(req, resp)
		}
		return &ProtocolError{
			StatusCode: resp.StatusCode,
			Method:     req.Method,
			Path:       req.URL.Path,
			Reason:     fmt.Sprintf("expected HTTP %d, got %d", expectedStatus, resp.StatusCode),
		}
	}

	if wireTarget != nil {
		if err := json.NewDecoder(resp.Body).Decode(wireTarget); err != nil {
			return &ProtocolError{
				StatusCode: resp.StatusCode,
				Method:     req.Method,
				Path:       req.URL.Path,
				Reason:     "malformed JSON response payload",
			}
		}
	}
	return nil
}

// decodeError reads up to maxErrorBodyBytes and decodes the AO APIError envelope.
// If the payload does not match the canonical envelope (including missing required fields),
// it returns a sanitized ProtocolError without copying raw response body text.
func (c *Client) decodeError(req *http.Request, resp *http.Response) error {
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes))
	if err != nil {
		return &ProtocolError{
			StatusCode: resp.StatusCode,
			Method:     req.Method,
			Path:       req.URL.Path,
			Reason:     "failed to read error response",
		}
	}

	var wire wireAPIError
	if jsonErr := json.Unmarshal(body, &wire); jsonErr == nil && wire.Error != "" && wire.Code != "" && wire.Message != "" {
		return &APIError{
			StatusCode:     resp.StatusCode,
			ErrorType:      wire.Error,
			Code:           wire.Code,
			Message:        wire.Message,
			RequestID:      wire.RequestID,
			Details:        wire.Details,
			ReportingOwner: wire.ReportingOwner,
		}
	}

	return &ProtocolError{
		StatusCode: resp.StatusCode,
		Method:     req.Method,
		Path:       req.URL.Path,
		Reason:     "malformed or non-conforming AO error envelope",
	}
}
// GetWorkspaceFile retrieves a workspace file from an active AO session using the canonical JSON envelope
// endpoint GET /api/v1/sessions/{sessionId}/workspace/file?path={filePath}.
//
// Invariants enforced:
// - MaxWireBytes and MaxBytes are independent, strictly positive (> 0), with integer overflow pre-checks.
// - Bounded reading reads at most MaxWireBytes + 1; rejects wire overflow with ErrPayloadTooLarge fail-closed.
// - Single JSON value decoded with trailing payload rejection (io.EOF check).
// - Validates presence and types of all 15 required fields of WorkspaceFileResponse without zero-value masking.
// - Validates sessionID and filePath match the requested values (ProtocolError on mismatch).
// - Validates status is a valid enum value ("unmodified", "modified", "added", "deleted").
// - Rejects binary files with ProtocolError.
// - Rejects truncated content with ProtocolError.
// - Rejects deleted files (deleted=true or status="deleted") returning ErrWorkspaceFileDeleted (never fake 404 APIError).
// - Verifies decoded content byte length <= MaxBytes; rejects content overflow with ErrPayloadTooLarge.
// - Preserves TransportError, APIError, ProtocolError error taxonomy without domain report parsing or task state mutation.
func (c *Client) GetWorkspaceFile(ctx context.Context, sessionID, filePath string, opts WorkspaceReadOptions) ([]byte, error) {
	if strings.TrimSpace(sessionID) == "" {
		return nil, fmt.Errorf("%w: sessionID cannot be empty", ErrBadRequest)
	}
	if strings.TrimSpace(filePath) == "" {
		return nil, fmt.Errorf("%w: filePath cannot be empty", ErrBadRequest)
	}
	if opts.MaxWireBytes <= 0 || opts.MaxBytes <= 0 {
		return nil, fmt.Errorf("%w: MaxWireBytes and MaxBytes must both be strictly positive (> 0)", ErrBadRequest)
	}
	if opts.MaxWireBytes >= math.MaxInt64 {
		return nil, fmt.Errorf("%w: MaxWireBytes causes integer overflow", ErrBadRequest)
	}
	if opts.MaxBytes >= math.MaxInt64 {
		return nil, fmt.Errorf("%w: MaxBytes causes integer overflow", ErrBadRequest)
	}

	reqURL := fmt.Sprintf("%s/api/v1/sessions/%s/workspace/file?path=%s",
		c.baseURL.String(),
		url.PathEscape(sessionID),
		url.QueryEscape(filePath),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create GET request failed: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &TransportError{Op: http.MethodGet, URL: req.URL.String(), Err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, c.decodeError(req, resp)
		}
		return nil, &ProtocolError{
			StatusCode: resp.StatusCode,
			Method:     req.Method,
			Path:       req.URL.Path,
			Reason:     fmt.Sprintf("expected HTTP 200, got %d", resp.StatusCode),
		}
	}

	limit := opts.MaxWireBytes + 1
	lr := io.LimitReader(resp.Body, limit)
	wireBytes, err := io.ReadAll(lr)
	if err != nil {
		return nil, &TransportError{Op: http.MethodGet, URL: req.URL.String(), Err: err}
	}
	if int64(len(wireBytes)) > opts.MaxWireBytes {
		return nil, ErrPayloadTooLarge
	}

	type rawWorkspaceEnvelope struct {
		SessionID        *string `json:"sessionId"`
		Path             *string `json:"path"`
		Content          *string `json:"content"`
		Binary           *bool   `json:"binary"`
		Deleted          *bool   `json:"deleted"`
		ContentTruncated *bool   `json:"contentTruncated"`
		Size             *int64  `json:"size"`
		Status           *string `json:"status"`
		WorkspaceVersion *string `json:"workspaceVersion"`
		Diff             *string `json:"diff"`
		DiffTruncated    *bool   `json:"diffTruncated"`
		Editable         *bool   `json:"editable"`
		FileFingerprint  *string `json:"fileFingerprint"`
		Additions        *int    `json:"additions"`
		Deletions        *int    `json:"deletions"`
	}

	dec := json.NewDecoder(bytes.NewReader(wireBytes))
	var raw rawWorkspaceEnvelope
	if err := dec.Decode(&raw); err != nil {
		return nil, &ProtocolError{
			StatusCode: resp.StatusCode,
			Method:     req.Method,
			Path:       req.URL.Path,
			Reason:     "malformed JSON response payload",
		}
	}

	var trailing json.RawMessage
	if err := dec.Decode(&trailing); err != io.EOF {
		return nil, &ProtocolError{
			StatusCode: resp.StatusCode,
			Method:     req.Method,
			Path:       req.URL.Path,
			Reason:     "unexpected trailing data after JSON envelope",
		}
	}

	if raw.SessionID == nil || raw.Path == nil || raw.Content == nil ||
		raw.Binary == nil || raw.Deleted == nil || raw.ContentTruncated == nil ||
		raw.Size == nil || raw.Status == nil || raw.WorkspaceVersion == nil ||
		raw.Diff == nil || raw.DiffTruncated == nil || raw.Editable == nil ||
		raw.FileFingerprint == nil || raw.Additions == nil || raw.Deletions == nil {
		return nil, &ProtocolError{
			StatusCode: resp.StatusCode,
			Method:     req.Method,
			Path:       req.URL.Path,
			Reason:     "missing required field in WorkspaceFileResponse envelope",
		}
	}

	if *raw.SessionID != sessionID {
		return nil, &ProtocolError{
			StatusCode: resp.StatusCode,
			Method:     req.Method,
			Path:       req.URL.Path,
			Reason:     fmt.Sprintf("envelope sessionId %q does not match requested %q", *raw.SessionID, sessionID),
		}
	}
	if *raw.Path != filePath {
		return nil, &ProtocolError{
			StatusCode: resp.StatusCode,
			Method:     req.Method,
			Path:       req.URL.Path,
			Reason:     fmt.Sprintf("envelope path %q does not match requested %q", *raw.Path, filePath),
		}
	}

	switch *raw.Status {
	case string(WorkspaceFileStatusUnmodified),
		string(WorkspaceFileStatusModified),
		string(WorkspaceFileStatusAdded),
		string(WorkspaceFileStatusDeleted):
		// valid status enum
	default:
		return nil, &ProtocolError{
			StatusCode: resp.StatusCode,
			Method:     req.Method,
			Path:       req.URL.Path,
			Reason:     fmt.Sprintf("invalid status value %q in workspace file envelope", *raw.Status),
		}
	}

	if *raw.Binary {
		return nil, &ProtocolError{
			StatusCode: resp.StatusCode,
			Method:     req.Method,
			Path:       req.URL.Path,
			Reason:     "binary workspace file is unsupported in text reader",
		}
	}

	if *raw.ContentTruncated {
		return nil, &ProtocolError{
			StatusCode: resp.StatusCode,
			Method:     req.Method,
			Path:       req.URL.Path,
			Reason:     "workspace file content truncated by upstream",
		}
	}

	if *raw.Deleted || *raw.Status == string(WorkspaceFileStatusDeleted) {
		return nil, ErrWorkspaceFileDeleted
	}

	contentBytes := []byte(*raw.Content)
	if int64(len(contentBytes)) > opts.MaxBytes {
		return nil, ErrPayloadTooLarge
	}

	return contentBytes, nil
}
