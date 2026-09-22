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
	"strings"
	"time"
)

const (
	// DefaultTimeout is the default HTTP client timeout for AO requests.
	DefaultTimeout = 15 * time.Second

	// maxErrorBodyBytes caps the maximum response size read for error bodies.
	maxErrorBodyBytes = 64 * 1024
)

// Client provides the low-level HTTP transport client to the local Untrivial Agent Orchestrator REST daemon.
type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
}

// Option configures a Client.
type Option func(*Client)

// WithHTTPClient configures a custom http.Client.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		if httpClient != nil {
			c.httpClient = httpClient
		}
	}
}

// WithTimeout configures a custom timeout on the default http.Client.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		if c.httpClient != nil && timeout > 0 {
			c.httpClient.Timeout = timeout
		}
	}
}

// NewClient constructs an AO Client validating that rawBaseURL targets a local loopback interface.
func NewClient(rawBaseURL string, opts ...Option) (*Client, error) {
	trimmed := strings.TrimSpace(rawBaseURL)
	if trimmed == "" {
		return nil, fmt.Errorf("%w: base URL cannot be empty", ErrBadRequest)
	}

	parsedURL, err := url.Parse(trimmed)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid URL format: %v", ErrBadRequest, err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, fmt.Errorf("%w: invalid scheme %q (must be http or https)", ErrNonLoopbackURL, parsedURL.Scheme)
	}

	if !isLoopbackURL(parsedURL) {
		return nil, fmt.Errorf("%w: host %q is not a loopback address", ErrNonLoopbackURL, parsedURL.Host)
	}

	parsedURL.Path = strings.TrimRight(parsedURL.Path, "/")

	c := &Client{
		baseURL: parsedURL,
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c, nil
}

// isLoopbackURL verifies that the URL hostname is a valid local loopback address.
func isLoopbackURL(u *url.URL) bool {
	host := u.Hostname()
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	if ip != nil && ip.IsLoopback() {
		return true
	}
	return false
}

// BaseURL returns the validated base URL as a string.
func (c *Client) BaseURL() string {
	return c.baseURL.String()
}

// buildURL joins relPath to baseURL.
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

// getRaw performs an HTTP GET request and returns the raw response body as a string.
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
		return "", fmt.Errorf("%w: %v", ErrDaemonUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", c.decodeError(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read raw body failed: %w", err)
	}
	return string(body), nil
}

// post performs an HTTP POST request and decodes the JSON response into target.
func (c *Client) post(ctx context.Context, relPath string, body any, target any) error {
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

	return c.do(req, target)
}

// do dispatches an HTTP request, checks for success, and decodes the target.
func (c *Client) do(req *http.Request, target any) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDaemonUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return c.decodeError(resp)
	}

	if target != nil {
		if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
			return fmt.Errorf("decode response JSON failed: %w", err)
		}
	}
	return nil
}

// decodeError reads an error response and decodes the AO APIError envelope.
func (c *Client) decodeError(resp *http.Response) error {
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes))
	if err != nil {
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("failed to read error response: %v", err),
		}
	}

	var apiErr APIError
	if jsonErr := json.Unmarshal(body, &apiErr); jsonErr == nil && (apiErr.Code != "" || apiErr.Message != "" || apiErr.ErrorType != "") {
		apiErr.StatusCode = resp.StatusCode
		return &apiErr
	}

	return &APIError{
		StatusCode: resp.StatusCode,
		Message:    strings.TrimSpace(string(body)),
	}
}
