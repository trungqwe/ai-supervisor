package ao

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
		return nil, fmt.Errorf("%w: %v", ErrBadRequest, ErrNilHTTPClient)
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

// get performs an HTTP GET request and decodes the JSON response into target.
func (c *Client) get(ctx context.Context, relPath string, target any) error {
	reqURL := c.buildURL(relPath)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return fmt.Errorf("create GET request failed: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	return c.do(req, target)
}

// getRaw performs an HTTP GET request and returns the raw response body.
func (c *Client) getRaw(ctx context.Context, relPath string, acceptHeader string) (string, error) {
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

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", c.decodeError(req, resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read raw body failed: %w", err)
	}
	return string(body), nil
}

// post performs an HTTP POST request and decodes the JSON response into target.
func (c *Client) post(ctx context.Context, relPath string, body any, target any) (int, error) {
	reqURL := c.buildURL(relPath)

	var bodyReader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return 0, fmt.Errorf("marshal request body failed: %w", err)
		}
		bodyReader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bodyReader)
	if err != nil {
		return 0, fmt.Errorf("create POST request failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, &TransportError{Op: req.Method, URL: req.URL.String(), Err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp.StatusCode, c.decodeError(req, resp)
	}

	if target != nil {
		if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
			return resp.StatusCode, &ProtocolError{
				StatusCode: resp.StatusCode,
				Method:     req.Method,
				Path:       req.URL.Path,
				Reason:     "malformed JSON response payload",
			}
		}
	}
	return resp.StatusCode, nil
}

// do dispatches an HTTP request, checks for success, and decodes the target.
func (c *Client) do(req *http.Request, target any) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return &TransportError{Op: req.Method, URL: req.URL.String(), Err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return c.decodeError(req, resp)
	}

	if target != nil {
		if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
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
// If the payload does not match the canonical envelope, it returns a sanitized ProtocolError
// without copying raw response body text.
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

	var apiErr APIError
	if jsonErr := json.Unmarshal(body, &apiErr); jsonErr == nil && apiErr.ErrorType != "" && apiErr.Code != "" {
		apiErr.StatusCode = resp.StatusCode
		return &apiErr
	}

	return &ProtocolError{
		StatusCode: resp.StatusCode,
		Method:     req.Method,
		Path:       req.URL.Path,
		Reason:     "malformed or non-conforming AO error envelope",
	}
}
