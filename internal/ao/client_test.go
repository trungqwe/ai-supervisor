package ao

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Helper to create a test client bound to an httptest.Server URL
func newTestClient(t *testing.T, s *httptest.Server) (*Client, *http.Client) {
	t.Helper()
	baseHTTPClient := &http.Client{Timeout: 5 * time.Second}
	client, err := NewClient(s.URL, baseHTTPClient)
	if err != nil {
		t.Fatalf("NewClient(%q) unexpected error: %v", s.URL, err)
	}
	return client, baseHTTPClient
}

// -----------------------------------------------------------------------------
// 1. Constructor, Base URL & Loopback Security Tests (P03T1-002, P03T1-003, P03T1R1-005)
// -----------------------------------------------------------------------------
func TestNewClient_ConstructorAndLoopbackSecurity(t *testing.T) {
	validHTTPClient := &http.Client{Timeout: 2 * time.Second}

	// 1.1 Nil httpClient MUST fail deterministically matching ErrNilHTTPClient and ErrBadRequest
	t.Run("nil_http_client", func(t *testing.T) {
		c, err := NewClient("http://127.0.0.1:3001", nil)
		if c != nil {
			t.Errorf("expected nil client, got %v", c)
		}
		if err == nil {
			t.Fatal("expected error on nil httpClient, got nil")
		}
		if !errors.Is(err, ErrNilHTTPClient) {
			t.Errorf("expected errors.Is(err, ErrNilHTTPClient), got: %v", err)
		}
		if !errors.Is(err, ErrBadRequest) {
			t.Errorf("expected errors.Is(err, ErrBadRequest), got: %v", err)
		}
	})

	// 1.2 Loopback base URL validation matrix
	cases := []struct {
		name        string
		rawURL      string
		expectPass  bool
		expectedErr error
	}{
		// Valid cases
		{name: "ipv4_loopback_standard", rawURL: "http://127.0.0.1:3001", expectPass: true},
		{name: "ipv4_loopback_trailing_slash", rawURL: "http://127.0.0.1:3001/", expectPass: true},
		{name: "ipv4_loopback_alternate_octets", rawURL: "http://127.0.0.2:8080", expectPass: true},
		{name: "ipv6_loopback", rawURL: "http://[::1]:3001", expectPass: true},
		{name: "ipv6_loopback_trailing_slash", rawURL: "http://[::1]:3001/", expectPass: true},

		// Prohibited hostnames & DNS resolution
		{name: "localhost_forbidden", rawURL: "http://localhost:3001", expectPass: false, expectedErr: ErrNonLoopbackURL},
		{name: "remote_hostname_forbidden", rawURL: "http://example.com:3001", expectPass: false, expectedErr: ErrNonLoopbackURL},

		// Prohibited schemes
		{name: "https_forbidden", rawURL: "https://127.0.0.1:3001", expectPass: false, expectedErr: ErrNonLoopbackURL},
		{name: "ftp_forbidden", rawURL: "ftp://127.0.0.1:3001", expectPass: false, expectedErr: ErrNonLoopbackURL},

		// Prohibited IP ranges
		{name: "private_non_loopback_ip", rawURL: "http://192.168.1.5:3001", expectPass: false, expectedErr: ErrNonLoopbackURL},
		{name: "wildcard_ip_forbidden", rawURL: "http://0.0.0.0:3001", expectPass: false, expectedErr: ErrNonLoopbackURL},
		{name: "public_ip_forbidden", rawURL: "http://8.8.8.8:3001", expectPass: false, expectedErr: ErrNonLoopbackURL},

		// Port validation
		{name: "missing_port_forbidden", rawURL: "http://127.0.0.1", expectPass: false, expectedErr: ErrNonLoopbackURL},
		{name: "non_numeric_port_forbidden", rawURL: "http://127.0.0.1:abc", expectPass: false, expectedErr: ErrNonLoopbackURL},
		{name: "zero_port_forbidden", rawURL: "http://127.0.0.1:0", expectPass: false, expectedErr: ErrNonLoopbackURL},
		{name: "port_out_of_range_forbidden", rawURL: "http://127.0.0.1:70000", expectPass: false, expectedErr: ErrNonLoopbackURL},

		// Prohibited URL components
		{name: "userinfo_forbidden", rawURL: "http://user:pass@127.0.0.1:3001", expectPass: false, expectedErr: ErrNonLoopbackURL},
		{name: "non_root_base_path_forbidden", rawURL: "http://127.0.0.1:3001/api", expectPass: false, expectedErr: ErrNonLoopbackURL},
		{name: "query_string_forbidden", rawURL: "http://127.0.0.1:3001?x=1", expectPass: false, expectedErr: ErrNonLoopbackURL},
		{name: "fragment_forbidden", rawURL: "http://127.0.0.1:3001#fragment", expectPass: false, expectedErr: ErrNonLoopbackURL},

		// Malformed & empty
		{name: "empty_url", rawURL: "", expectPass: false, expectedErr: ErrBadRequest},
		{name: "whitespace_url", rawURL: "   ", expectPass: false, expectedErr: ErrBadRequest},
		{name: "malformed_url", rawURL: "http://127.0.0.1:3001%invalid", expectPass: false, expectedErr: ErrNonLoopbackURL},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, err := NewClient(tc.rawURL, validHTTPClient)
			if tc.expectPass {
				if err != nil {
					t.Fatalf("expected pass for %q, got error: %v", tc.rawURL, err)
				}
				if c == nil {
					t.Fatal("expected non-nil Client")
				}
				if strings.HasSuffix(c.BaseURL(), "/") {
					t.Errorf("normalized baseURL %q should not have trailing slash", c.BaseURL())
				}
			} else {
				if err == nil {
					t.Fatalf("expected failure for %q, got success", tc.rawURL)
				}
				if tc.expectedErr != nil && !errors.Is(err, tc.expectedErr) {
					t.Errorf("expected errors.Is(err, %v), got %v", tc.expectedErr, err)
				}
			}
		})
	}
}

// -----------------------------------------------------------------------------
// 2. Caller Object Preservation & Redirect Safety (P03T1-002, P03T1-003)
// -----------------------------------------------------------------------------
func TestClient_CallerObjectPreservationAndRedirectSafety(t *testing.T) {
	customTransport := &http.Transport{
		MaxIdleConns: 42,
	}
	originalTimeout := 7 * time.Second
	callerClient := &http.Client{
		Transport: customTransport,
		Timeout:   originalTimeout,
	}

	c, err := NewClient("http://127.0.0.1:3001", callerClient)
	if err != nil {
		t.Fatalf("unexpected NewClient error: %v", err)
	}

	// 2.1 Verify caller-owned client object was NOT mutated
	if callerClient.Timeout != originalTimeout {
		t.Errorf("caller client Timeout mutated: got %v, want %v", callerClient.Timeout, originalTimeout)
	}
	if callerClient.CheckRedirect != nil {
		t.Errorf("caller client CheckRedirect mutated: expected nil, got non-nil")
	}
	if callerClient.Transport != customTransport {
		t.Errorf("caller client Transport mutated")
	}

	// 2.2 Verify adapter client preserved Transport and Timeout from caller
	if c.httpClient.Timeout != originalTimeout {
		t.Errorf("adapter client Timeout mismatch: got %v, want %v", c.httpClient.Timeout, originalTimeout)
	}
	if c.httpClient.Transport != customTransport {
		t.Errorf("adapter client Transport mismatch")
	}
	if c.httpClient.CheckRedirect == nil {
		t.Fatal("adapter client must install CheckRedirect policy")
	}

	// 2.3 Verify redirect fail-closed: external redirection is blocked
	externalServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("external server was reached via redirect! Request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer externalServer.Close()

	redirectServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, externalServer.URL+"/evil", http.StatusFound)
	}))
	defer redirectServer.Close()

	redirectClient, _ := newTestClient(t, redirectServer)
	_, err = redirectClient.CheckHealth(context.Background())
	if err == nil {
		t.Fatal("expected redirect to fail closed, got nil error")
	}
	if !errors.Is(err, ErrRedirectAttempted) {
		t.Errorf("expected errors.Is(err, ErrRedirectAttempted), got %v", err)
	}
}

// -----------------------------------------------------------------------------
// 3. Exact Status Contract Tests (P03T1R1-001)
// -----------------------------------------------------------------------------
func TestClient_ExactSuccessStatusContract(t *testing.T) {
	// Pinned endpoint contract requires exact status codes:
	// GET /healthz -> 200
	// GET /readyz -> 200
	// GET /api/v1/agents -> 200
	// GET /api/v1/agents/readiness -> 200
	// GET /api/v1/openapi.yaml -> 200
	// GET /api/v1/projects/{id} -> 200
	// GET /api/v1/sessions/{id} -> 200
	// POST /api/v1/projects -> 201

	cases := []struct {
		name       string
		method     string
		path       string
		respStatus int
		respBody   string
		call       func(c *Client) error
	}{
		{
			name:       "health_wrong_status_201",
			method:     http.MethodGet,
			path:       "/healthz",
			respStatus: http.StatusCreated,
			respBody:   `{"status":"ok","service":"agent-orchestrator-daemon","pid":1234}`,
			call:       func(c *Client) error { _, err := c.CheckHealth(context.Background()); return err },
		},
		{
			name:       "readiness_wrong_status_202",
			method:     http.MethodGet,
			path:       "/readyz",
			respStatus: http.StatusAccepted,
			respBody:   `{"status":"ready","service":"agent-orchestrator-daemon","pid":1234}`,
			call:       func(c *Client) error { _, err := c.CheckReadiness(context.Background()); return err },
		},
		{
			name:       "agents_wrong_status_201",
			method:     http.MethodGet,
			path:       "/api/v1/agents",
			respStatus: http.StatusCreated,
			respBody:   `{"supported":[],"installed":[],"authorized":[]}`,
			call:       func(c *Client) error { _, err := c.ListAgents(context.Background()); return err },
		},
		{
			name:       "agents_readiness_wrong_status_201",
			method:     http.MethodGet,
			path:       "/api/v1/agents/readiness",
			respStatus: http.StatusCreated,
			respBody:   `{"agents":[]}`,
			call:       func(c *Client) error { _, err := c.GetAgentReadiness(context.Background(), "claude-code"); return err },
		},
		{
			name:       "openapi_wrong_status_206",
			method:     http.MethodGet,
			path:       "/api/v1/openapi.yaml",
			respStatus: http.StatusPartialContent,
			respBody:   `openapi: 3.1.0`,
			call:       func(c *Client) error { _, err := c.GetAPIContract(context.Background()); return err },
		},
		{
			name:       "get_project_wrong_status_201",
			method:     http.MethodGet,
			path:       "/api/v1/projects/p1",
			respStatus: http.StatusCreated,
			respBody:   `{"status":"ok","project":{"id":"p1","name":"proj1","path":"/tmp"}}`,
			call:       func(c *Client) error { _, err := c.GetProject(context.Background(), "p1"); return err },
		},
		{
			name:       "get_worker_status_wrong_status_201",
			method:     http.MethodGet,
			path:       "/api/v1/sessions/s1",
			respStatus: http.StatusCreated,
			respBody:   `{"session":{"id":"s1","status":"working","activity":{"state":"active","lastActivityAt":"2026-09-22T10:00:00Z"}}}`,
			call:       func(c *Client) error { _, err := c.GetWorkerStatus(context.Background(), "s1"); return err },
		},
		{
			name:       "register_project_wrong_status_200",
			method:     http.MethodPost,
			path:       "/api/v1/projects",
			respStatus: http.StatusOK,
			respBody:   `{"project":{"id":"p1","name":"p1","path":"/tmp"}}`,
			call:       func(c *Client) error { _, err := c.RegisterProject(context.Background(), "p1", "/tmp"); return err },
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.respStatus)
				w.Write([]byte(tc.respBody))
			}))
			defer s.Close()

			c, _ := newTestClient(t, s)
			err := tc.call(c)
			if err == nil {
				t.Fatalf("expected ProtocolError on status %d, got nil", tc.respStatus)
			}
			if !errors.Is(err, ErrProtocolViolation) {
				t.Errorf("expected errors.Is(err, ErrProtocolViolation), got %v", err)
			}
			var protoErr *ProtocolError
			if !errors.As(err, &protoErr) {
				t.Errorf("expected *ProtocolError, got %T: %v", err, err)
			} else if protoErr.StatusCode != tc.respStatus {
				t.Errorf("expected ProtocolError.StatusCode %d, got %d", tc.respStatus, protoErr.StatusCode)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// 4. Preflight Daemon Identity Tests (P03T1-005, P03T1R1-002)
// -----------------------------------------------------------------------------
func TestClient_CheckHealthAndReadiness_DaemonIdentity(t *testing.T) {
	cases := []struct {
		name         string
		probe        string // "health" or "readiness"
		statusCode   int
		body         string
		expectPass   bool
		expectedDesc string
	}{
		// Valid health
		{
			name:       "health_valid",
			probe:      "health",
			statusCode: http.StatusOK,
			body:       `{"status":"ok","service":"agent-orchestrator-daemon","pid":12345,"executablePath":"/bin/ao"}`,
			expectPass: true,
		},
		// Health failures
		{
			name:         "health_missing_service",
			probe:        "health",
			statusCode:   http.StatusOK,
			body:         `{"status":"ok","service":"","pid":12345}`,
			expectPass:   false,
			expectedDesc: "service",
		},
		{
			name:         "health_wrong_service",
			probe:        "health",
			statusCode:   http.StatusOK,
			body:         `{"status":"ok","service":"rogue-daemon","pid":12345}`,
			expectPass:   false,
			expectedDesc: "service",
		},
		{
			name:         "health_zero_pid",
			probe:        "health",
			statusCode:   http.StatusOK,
			body:         `{"status":"ok","service":"agent-orchestrator-daemon","pid":0}`,
			expectPass:   false,
			expectedDesc: "pid",
		},
		{
			name:         "health_negative_pid",
			probe:        "health",
			statusCode:   http.StatusOK,
			body:         `{"status":"ok","service":"agent-orchestrator-daemon","pid":-1}`,
			expectPass:   false,
			expectedDesc: "pid",
		},
		{
			name:         "health_missing_status",
			probe:        "health",
			statusCode:   http.StatusOK,
			body:         `{"service":"agent-orchestrator-daemon","pid":12345}`,
			expectPass:   false,
			expectedDesc: "status",
		},
		{
			name:         "health_wrong_status",
			probe:        "health",
			statusCode:   http.StatusOK,
			body:         `{"status":"not_ok","service":"agent-orchestrator-daemon","pid":12345}`,
			expectPass:   false,
			expectedDesc: "status",
		},
		{
			name:         "health_malformed_json",
			probe:        "health",
			statusCode:   http.StatusOK,
			body:         `{invalid-json}`,
			expectPass:   false,
			expectedDesc: "malformed",
		},

		// Valid readiness
		{
			name:       "readiness_valid",
			probe:      "readiness",
			statusCode: http.StatusOK,
			body:       `{"status":"ready","service":"agent-orchestrator-daemon","pid":54321}`,
			expectPass: true,
		},
		// Readiness failures
		{
			name:         "readiness_missing_service",
			probe:        "readiness",
			statusCode:   http.StatusOK,
			body:         `{"status":"ready","service":"","pid":54321}`,
			expectPass:   false,
			expectedDesc: "service",
		},
		{
			name:         "readiness_wrong_service",
			probe:        "readiness",
			statusCode:   http.StatusOK,
			body:         `{"status":"ready","service":"other-daemon","pid":54321}`,
			expectPass:   false,
			expectedDesc: "service",
		},
		{
			name:         "readiness_zero_pid",
			probe:        "readiness",
			statusCode:   http.StatusOK,
			body:         `{"status":"ready","service":"agent-orchestrator-daemon","pid":0}`,
			expectPass:   false,
			expectedDesc: "pid",
		},
		{
			name:         "readiness_negative_pid",
			probe:        "readiness",
			statusCode:   http.StatusOK,
			body:         `{"status":"ready","service":"agent-orchestrator-daemon","pid":-42}`,
			expectPass:   false,
			expectedDesc: "pid",
		},
		{
			name:         "readiness_missing_status",
			probe:        "readiness",
			statusCode:   http.StatusOK,
			body:         `{"service":"agent-orchestrator-daemon","pid":54321}`,
			expectPass:   false,
			expectedDesc: "status",
		},
		{
			name:         "readiness_wrong_status",
			probe:        "readiness",
			statusCode:   http.StatusOK,
			body:         `{"status":"ok","service":"agent-orchestrator-daemon","pid":54321}`,
			expectPass:   false,
			expectedDesc: "status",
		},
		{
			name:         "readiness_malformed_json",
			probe:        "readiness",
			statusCode:   http.StatusOK,
			body:         `{"status":`,
			expectPass:   false,
			expectedDesc: "malformed",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.statusCode)
				w.Write([]byte(tc.body))
			}))
			defer s.Close()

			c, _ := newTestClient(t, s)
			var err error
			if tc.probe == "health" {
				res, hErr := c.CheckHealth(context.Background())
				err = hErr
				if tc.expectPass {
					if err != nil {
						t.Fatalf("unexpected CheckHealth error: %v", err)
					}
					if res.Status != "ok" || res.Service != DaemonServiceAO || res.PID <= 0 {
						t.Errorf("unexpected health status result: %+v", res)
					}
				}
			} else {
				res, rErr := c.CheckReadiness(context.Background())
				err = rErr
				if tc.expectPass {
					if err != nil {
						t.Fatalf("unexpected CheckReadiness error: %v", err)
					}
					if res.Status != "ready" || res.Service != DaemonServiceAO || res.PID <= 0 {
						t.Errorf("unexpected readiness status result: %+v", res)
					}
				}
			}

			if !tc.expectPass {
				if err == nil {
					t.Fatalf("expected error for %s, got nil", tc.name)
				}
				if !errors.Is(err, ErrProtocolViolation) {
					t.Errorf("expected errors.Is(err, ErrProtocolViolation), got %v", err)
				}
			}
		})
	}
}

// -----------------------------------------------------------------------------
// 5. Agent Catalog & Complete Readiness Observation (P03T1-006, P03T1R1-003)
// -----------------------------------------------------------------------------
func TestClient_AgentsAndReadiness(t *testing.T) {
	now := time.Date(2026, 9, 22, 10, 0, 0, 0, time.UTC)
	nowStr := now.Format(time.RFC3339)

	t.Run("list_agents_success", func(t *testing.T) {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet || r.URL.Path != "/api/v1/agents" {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"supported":[{"id":"claude-code","label":"Claude Code"}],
				"installed":[{"id":"claude-code","label":"Claude Code","authStatus":"authorized","usageCount":3}],
				"authorized":[{"id":"claude-code","label":"Claude Code","authStatus":"authorized"}]
			}`))
		}))
		defer s.Close()

		c, _ := newTestClient(t, s)
		inv, err := c.ListAgents(context.Background())
		if err != nil {
			t.Fatalf("unexpected ListAgents error: %v", err)
		}
		if len(inv.Supported) != 1 || inv.Supported[0].ID != "claude-code" {
			t.Errorf("unexpected supported inventory: %+v", inv.Supported)
		}
		if len(inv.Installed) != 1 || inv.Installed[0].UsageCount != 3 {
			t.Errorf("unexpected installed inventory: %+v", inv.Installed)
		}
		if len(inv.Authorized) != 1 || inv.Authorized[0].AuthStatus != "authorized" {
			t.Errorf("unexpected authorized inventory: %+v", inv.Authorized)
		}
	})

	t.Run("agent_readiness_full_coverage", func(t *testing.T) {
		ensureCalled := false
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/v1/agents/readiness/ensure" {
				ensureCalled = true
				t.Errorf("POST /readiness/ensure MUST NOT be called in read-only task!")
			}
			if r.Method != http.MethodGet || r.URL.Path != "/api/v1/agents/readiness" {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			// Return snapshots covering actual pinned values:
			// Installation: installed, not_installed, unknown
			// Authentication: authorized, unauthorized, unknown, not_applicable
			// Freshness: fresh, stale, checking
			// Effective: ready, not_ready, unknown
			// Pinned style reason codes: installed, authorized, checking, not_installed, auth_not_applicable
			w.Write([]byte(fmt.Sprintf(`{
				"agents": [
					{
						"id": "claude-code",
						"label": "Claude Code",
						"installation": {
							"state": "installed",
							"freshness": "fresh",
							"checkedAt": "%s",
							"attemptedAt": "%s",
							"reasonCode": "installed",
							"reason": "binary located at /usr/local/bin/claude"
						},
						"authentication": {
							"state": "authorized",
							"freshness": "fresh",
							"checkedAt": "%s",
							"attemptedAt": "%s",
							"reasonCode": "authorized",
							"reason": "token verified"
						},
						"effectiveReadiness": "ready",
						"usageCount": 12,
						"lastUsedAt": "%s"
					},
					{
						"id": "codex",
						"label": "Codex",
						"installation": {
							"state": "not_installed",
							"freshness": "stale",
							"reasonCode": "not_installed",
							"reason": "binary missing"
						},
						"authentication": {
							"state": "unknown",
							"freshness": "stale",
							"reasonCode": "auth_skipped_not_installed"
						},
						"effectiveReadiness": "not_ready",
						"usageCount": 0
					},
					{
						"id": "agy",
						"label": "Antigravity CLI",
						"installation": {
							"state": "installed",
							"freshness": "checking",
							"reasonCode": "checking"
						},
						"authentication": {
							"state": "not_applicable",
							"freshness": "fresh",
							"reasonCode": "auth_not_applicable"
						},
						"effectiveReadiness": "unknown",
						"usageCount": 5
					}
				]
			}`, nowStr, nowStr, nowStr, nowStr, nowStr)))
		}))
		defer s.Close()

		c, _ := newTestClient(t, s)

		// 5.1 Requesting claude-code: exact match and field preservation
		snap, err := c.GetAgentReadiness(context.Background(), "claude-code")
		if err != nil {
			t.Fatalf("unexpected GetAgentReadiness error: %v", err)
		}
		if snap.ID != "claude-code" || snap.Label != "Claude Code" {
			t.Errorf("unexpected agent identity: %+v", snap)
		}
		if snap.Installation.State != "installed" || snap.Installation.Freshness != "fresh" || snap.Installation.ReasonCode != "installed" {
			t.Errorf("unexpected installation observation: %+v", snap.Installation)
		}
		if snap.Authentication.State != "authorized" || snap.Authentication.Freshness != "fresh" || snap.Authentication.ReasonCode != "authorized" {
			t.Errorf("unexpected authentication observation: %+v", snap.Authentication)
		}
		if snap.EffectiveReadiness != "ready" || snap.UsageCount != 12 {
			t.Errorf("unexpected effective readiness or usageCount: %+v", snap)
		}
		if snap.Installation.CheckedAt == nil || !snap.Installation.CheckedAt.Equal(now) {
			t.Errorf("checkedAt timestamp not preserved: %v", snap.Installation.CheckedAt)
		}
		if snap.LastUsedAt == nil || !snap.LastUsedAt.Equal(now) {
			t.Errorf("lastUsedAt timestamp not preserved: %v", snap.LastUsedAt)
		}

		// 5.2 Requesting codex: exact match
		codexSnap, err := c.GetAgentReadiness(context.Background(), "codex")
		if err != nil {
			t.Fatalf("unexpected GetAgentReadiness(codex): %v", err)
		}
		if codexSnap.Installation.State != "not_installed" || codexSnap.EffectiveReadiness != "not_ready" {
			t.Errorf("unexpected codex snapshot: %+v", codexSnap)
		}

		// 5.3 Requesting agy: exact match
		agySnap, err := c.GetAgentReadiness(context.Background(), "agy")
		if err != nil {
			t.Fatalf("unexpected GetAgentReadiness(agy): %v", err)
		}
		if agySnap.Authentication.State != "not_applicable" || agySnap.EffectiveReadiness != "unknown" {
			t.Errorf("unexpected agy snapshot: %+v", agySnap)
		}

		// 5.4 Requesting absent agent returns ErrAgentNotFound
		_, err = c.GetAgentReadiness(context.Background(), "nonexistent-agent")
		if err == nil {
			t.Fatal("expected ErrAgentNotFound, got nil")
		}
		if !errors.Is(err, ErrAgentNotFound) {
			t.Errorf("expected errors.Is(err, ErrAgentNotFound), got %v", err)
		}

		// 5.5 Empty or whitespace agent ID returns ErrBadRequest
		_, err = c.GetAgentReadiness(context.Background(), "")
		if !errors.Is(err, ErrBadRequest) {
			t.Errorf("expected ErrBadRequest on empty agentID, got %v", err)
		}
		_, err = c.GetAgentReadiness(context.Background(), "   ")
		if !errors.Is(err, ErrBadRequest) {
			t.Errorf("expected ErrBadRequest on whitespace agentID, got %v", err)
		}

		// Verify ensure was never called
		if ensureCalled {
			t.Errorf("ensureReadiness was unexpectedly called")
		}
	})
}

// -----------------------------------------------------------------------------
// 6. OpenAPI Raw Contract & Representative Fixture (P03T1-006, P03T1R1-004)
// -----------------------------------------------------------------------------
func TestClient_GetAPIContract(t *testing.T) {
	// Pinned source accurate fixture beginning with openapi: 3.1.0 and info.version: 0.1.0-route-shell
	pinnedSchema := `openapi: 3.1.0
info:
  description: Loopback-only HTTP surface served by the Go daemon. Generated from Go (code-first) — do not edit by hand; run go generate ./...
  title: Agent Orchestrator HTTP daemon
  version: 0.1.0-route-shell
servers:
- description: Local daemon (loopback only)
  url: http://127.0.0.1:3001
paths:
  /api/v1/agents:
    get:
      operationId: listAgents
`

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/openapi.yaml" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/yaml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(pinnedSchema))
	}))
	defer s.Close()

	c, _ := newTestClient(t, s)
	schema, err := c.GetAPIContract(context.Background())
	if err != nil {
		t.Fatalf("unexpected GetAPIContract error: %v", err)
	}

	if schema != pinnedSchema {
		t.Errorf("GetAPIContract did not return raw schema unchanged")
	}
}

// -----------------------------------------------------------------------------
// 7. Project Transport & Resource Identity Validation (P03T1-007, P03T1R1-004)
// -----------------------------------------------------------------------------
func TestClient_Projects_TransportAndResourceIdentity(t *testing.T) {
	// 7.1 RegisterProject validation
	t.Run("register_project_contract_and_identity", func(t *testing.T) {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.Path != "/api/v1/projects" {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			}

			body, _ := io.ReadAll(r.Body)
			var rawMap map[string]any
			if err := json.Unmarshal(body, &rawMap); err != nil {
				t.Fatalf("failed to unmarshal request body: %v", err)
			}

			// Wire JSON key set must be EXACTLY: path, projectId
			if len(rawMap) != 2 {
				t.Errorf("expected exactly 2 keys in payload, got %d: %v", len(rawMap), rawMap)
			}
			if _, ok := rawMap["path"]; !ok {
				t.Errorf("missing required 'path' key")
			}
			if _, ok := rawMap["projectId"]; !ok {
				t.Errorf("missing required 'projectId' key")
			}

			// For out-of-contract key tests
			for k := range rawMap {
				if k != "path" && k != "projectId" {
					t.Errorf("unauthorized key %q present in wire body", k)
				}
			}

			// Mock response based on input
			pID := rawMap["projectId"].(string)
			switch pID {
			case "empty-resp-id":
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(`{"project":{"id":"","path":"/tmp"}}`))
			case "mismatched-resp-id":
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(`{"project":{"id":"different-id","path":"/tmp"}}`))
			case "null-project":
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(`{"project":null}`))
			case "empty-project":
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(`{"project":{}}`))
			default:
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(fmt.Sprintf(`{"project":{"id":%q,"name":"my-proj","path":%q,"repo":"git@github.com:foo/bar","defaultBranch":"main"}}`, pID, rawMap["path"])))
			}
		}))
		defer s.Close()

		c, _ := newTestClient(t, s)

		// Successful registration
		p, err := c.RegisterProject(context.Background(), "my-proj-id", "/workspace/repo")
		if err != nil {
			t.Fatalf("unexpected RegisterProject error: %v", err)
		}
		if p.ID != "my-proj-id" || p.Path != "/workspace/repo" || p.Status != "ok" || p.IsDegraded {
			t.Errorf("unexpected registered project: %+v", p)
		}

		// Input validation: empty / whitespace rejected before network
		if _, err := c.RegisterProject(context.Background(), "", "/tmp"); !errors.Is(err, ErrBadRequest) {
			t.Errorf("expected ErrBadRequest on empty projectID, got %v", err)
		}
		if _, err := c.RegisterProject(context.Background(), "   ", "/tmp"); !errors.Is(err, ErrBadRequest) {
			t.Errorf("expected ErrBadRequest on whitespace projectID, got %v", err)
		}
		if _, err := c.RegisterProject(context.Background(), "p1", ""); !errors.Is(err, ErrBadRequest) {
			t.Errorf("expected ErrBadRequest on empty rootPath, got %v", err)
		}
		if _, err := c.RegisterProject(context.Background(), "p1", "   "); !errors.Is(err, ErrBadRequest) {
			t.Errorf("expected ErrBadRequest on whitespace rootPath, got %v", err)
		}

		// Response identity validation: empty ID fails closed
		_, err = c.RegisterProject(context.Background(), "empty-resp-id", "/tmp")
		if !errors.Is(err, ErrProtocolViolation) {
			t.Errorf("expected ErrProtocolViolation on empty response ID, got %v", err)
		}

		// Response identity validation: mismatched ID fails closed
		_, err = c.RegisterProject(context.Background(), "mismatched-resp-id", "/tmp")
		if !errors.Is(err, ErrProtocolViolation) {
			t.Errorf("expected ErrProtocolViolation on mismatched response ID, got %v", err)
		}

		// Response identity validation: null project fails closed
		_, err = c.RegisterProject(context.Background(), "null-project", "/tmp")
		if !errors.Is(err, ErrProtocolViolation) {
			t.Errorf("expected ErrProtocolViolation on null project, got %v", err)
		}

		// Response identity validation: empty project object fails closed
		_, err = c.RegisterProject(context.Background(), "empty-project", "/tmp")
		if !errors.Is(err, ErrProtocolViolation) {
			t.Errorf("expected ErrProtocolViolation on empty project object, got %v", err)
		}
	})

	// 7.2 GetProject validation & URL path segment escaping
	t.Run("get_project_escaping_and_degraded", func(t *testing.T) {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("unexpected method: %s", r.Method)
			}

			w.Header().Set("Content-Type", "application/json")

			// Check exact RequestURI escaping
			switch r.RequestURI {
			case "/api/v1/projects/normal-proj":
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"status":"ok","project":{"id":"normal-proj","name":"Normal","path":"/path"}}`))
			case "/api/v1/projects/degraded-proj":
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"status":"degraded","project":{"id":"degraded-proj","name":"Degraded","path":"/path","resolveError":"git config corrupt"}}`))
			case "/api/v1/projects/degraded-missing-error":
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"status":"degraded","project":{"id":"degraded-missing-error","name":"Degraded","path":"/path"}}`))
			case "/api/v1/projects/mismatched-id":
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"status":"ok","project":{"id":"other-id","name":"Mismatch","path":"/path"}}`))
			case "/api/v1/projects/null-project":
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"status":"ok","project":null}`))
			case "/api/v1/projects/unknown-discriminator":
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"status":"unsupported_state","project":{"id":"unknown-discriminator"}}`))
			case "/api/v1/projects/proj%2Fslash":
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"status":"ok","project":{"id":"proj/slash","name":"Slash","path":"/path"}}`))
			case "/api/v1/projects/proj%3Fquery":
				if r.URL.RawQuery != "" {
					t.Errorf("query string was injected! RawQuery=%q", r.URL.RawQuery)
				}
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"status":"ok","project":{"id":"proj?query","name":"Query","path":"/path"}}`))
			case "/api/v1/projects/proj%23frag":
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"status":"ok","project":{"id":"proj#frag","name":"Frag","path":"/path"}}`))
			case "/api/v1/projects/proj%25pct":
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"status":"ok","project":{"id":"proj%pct","name":"Pct","path":"/path"}}`))
			default:
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"error":"not_found","code":"PROJECT_NOT_FOUND","message":"project not found"}`))
			}
		}))
		defer s.Close()

		c, _ := newTestClient(t, s)

		// Healthy project
		p, err := c.GetProject(context.Background(), "normal-proj")
		if err != nil {
			t.Fatalf("unexpected GetProject error: %v", err)
		}
		if p.ID != "normal-proj" || p.Status != "ok" || p.IsDegraded || p.ResolveError != "" {
			t.Errorf("unexpected healthy project: %+v", p)
		}

		// Degraded project
		deg, err := c.GetProject(context.Background(), "degraded-proj")
		if err != nil {
			t.Fatalf("unexpected degraded GetProject error: %v", err)
		}
		if deg.ID != "degraded-proj" || deg.Status != "degraded" || !deg.IsDegraded || deg.ResolveError != "git config corrupt" {
			t.Errorf("unexpected degraded project: %+v", deg)
		}

		// Degraded project missing ResolveError fails closed
		_, err = c.GetProject(context.Background(), "degraded-missing-error")
		if !errors.Is(err, ErrProtocolViolation) {
			t.Errorf("expected ErrProtocolViolation when degraded project lacks resolveError, got %v", err)
		}

		// Mismatched ID fails closed
		_, err = c.GetProject(context.Background(), "mismatched-id")
		if !errors.Is(err, ErrProtocolViolation) {
			t.Errorf("expected ErrProtocolViolation on mismatched ID, got %v", err)
		}

		// Null project fails closed
		_, err = c.GetProject(context.Background(), "null-project")
		if !errors.Is(err, ErrProtocolViolation) {
			t.Errorf("expected ErrProtocolViolation on null project, got %v", err)
		}

		// Unknown discriminator fails closed
		_, err = c.GetProject(context.Background(), "unknown-discriminator")
		if !errors.Is(err, ErrProtocolViolation) {
			t.Errorf("expected ErrProtocolViolation on unknown discriminator, got %v", err)
		}

		// URL path segment escaping for /, ?, #, %
		escapedCases := []string{
			"proj/slash",
			"proj?query",
			"proj#frag",
			"proj%pct",
		}
		for _, id := range escapedCases {
			proj, err := c.GetProject(context.Background(), id)
			if err != nil {
				t.Fatalf("GetProject(%q) failed: %v", id, err)
			}
			if proj.ID != id {
				t.Errorf("expected project ID %q, got %q", id, proj.ID)
			}
		}

		// 404 PROJECT_NOT_FOUND classification
		_, err = c.GetProject(context.Background(), "nonexistent")
		if !errors.Is(err, ErrProjectNotFound) {
			t.Errorf("expected errors.Is(err, ErrProjectNotFound), got %v", err)
		}
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expected errors.Is(err, ErrNotFound), got %v", err)
		}
	})
}

// -----------------------------------------------------------------------------
// 8. Session Read Model & Activity State Validation (P03T1-008, P03T1R1-004)
// -----------------------------------------------------------------------------
func TestClient_Sessions_ReadModelAndActivityValidation(t *testing.T) {
	now := time.Date(2026, 9, 22, 11, 0, 0, 0, time.UTC)
	nowStr := now.Format(time.RFC3339)

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")

		switch r.RequestURI {
		case "/api/v1/sessions/sess-active":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(fmt.Sprintf(`{"session":{"id":"sess-active","projectId":"p1","status":"working","isTerminated":false,"branch":"feat/x","activity":{"state":"active","lastActivityAt":%q}}}`, nowStr)))
		case "/api/v1/sessions/sess-idle":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(fmt.Sprintf(`{"session":{"id":"sess-idle","projectId":"p1","status":"idle","isTerminated":false,"activity":{"state":"idle","lastActivityAt":%q}}}`, nowStr)))
		case "/api/v1/sessions/sess-waiting-input":
			// Pinned AO derives ActivityWaitingInput -> status "needs_input"
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(fmt.Sprintf(`{"session":{"id":"sess-waiting-input","projectId":"p1","status":"needs_input","isTerminated":false,"activity":{"state":"waiting_input","lastActivityAt":%q}}}`, nowStr)))
		case "/api/v1/sessions/sess-blocked":
			// Pinned AO derives ActivityBlocked -> status "needs_input"
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(fmt.Sprintf(`{"session":{"id":"sess-blocked","projectId":"p1","status":"needs_input","isTerminated":false,"activity":{"state":"blocked","lastActivityAt":%q}}}`, nowStr)))
		case "/api/v1/sessions/sess-exited":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(fmt.Sprintf(`{"session":{"id":"sess-exited","projectId":"p1","status":"exited","isTerminated":true,"activity":{"state":"exited","lastActivityAt":%q}}}`, nowStr)))
		case "/api/v1/sessions/sess-unknown-state":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(fmt.Sprintf(`{"session":{"id":"sess-unknown-state","projectId":"p1","status":"working","activity":{"state":"sleeping","lastActivityAt":%q}}}`, nowStr)))
		case "/api/v1/sessions/sess-mismatch":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(fmt.Sprintf(`{"session":{"id":"other-sess","projectId":"p1","status":"working","activity":{"state":"active","lastActivityAt":%q}}}`, nowStr)))
		case "/api/v1/sessions/sess-null":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"session":null}`))
		case "/api/v1/sessions/sess%2Fslash":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(fmt.Sprintf(`{"session":{"id":"sess/slash","projectId":"p1","status":"working","activity":{"state":"active","lastActivityAt":%q}}}`, nowStr)))
		case "/api/v1/sessions/sess%3Fquery":
			if r.URL.RawQuery != "" {
				t.Errorf("query string injected! RawQuery=%q", r.URL.RawQuery)
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(fmt.Sprintf(`{"session":{"id":"sess?query","projectId":"p1","status":"working","activity":{"state":"active","lastActivityAt":%q}}}`, nowStr)))
		case "/api/v1/sessions/sess%23frag":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(fmt.Sprintf(`{"session":{"id":"sess#frag","projectId":"p1","status":"working","activity":{"state":"active","lastActivityAt":%q}}}`, nowStr)))
		case "/api/v1/sessions/sess%25pct":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(fmt.Sprintf(`{"session":{"id":"sess%%pct","projectId":"p1","status":"working","activity":{"state":"active","lastActivityAt":%q}}}`, nowStr)))
		default:
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":"not_found","code":"SESSION_NOT_FOUND","message":"session not found"}`))
		}
	}))
	defer s.Close()

	c, _ := newTestClient(t, s)

	// Test 5 canonical activity states
	canonicalStates := []struct {
		id            string
		expectedState ActivityState
		expectedTerm  bool
	}{
		{"sess-active", ActivityStateActive, false},
		{"sess-idle", ActivityStateIdle, false},
		{"sess-waiting-input", ActivityStateWaitingInput, false},
		{"sess-blocked", ActivityStateBlocked, false},
		{"sess-exited", ActivityStateExited, true},
	}

	for _, tc := range canonicalStates {
		t.Run(string(tc.expectedState), func(t *testing.T) {
			st, err := c.GetWorkerStatus(context.Background(), tc.id)
			if err != nil {
				t.Fatalf("unexpected GetWorkerStatus(%q) error: %v", tc.id, err)
			}
			if st.ID != tc.id {
				t.Errorf("expected session ID %q, got %q", tc.id, st.ID)
			}
			if st.Activity.State != tc.expectedState {
				t.Errorf("expected activity state %q, got %q", tc.expectedState, st.Activity.State)
			}
			if st.IsTerminated != tc.expectedTerm {
				t.Errorf("expected isTerminated %v, got %v", tc.expectedTerm, st.IsTerminated)
			}
			if !st.Activity.LastActivityAt.Equal(now) {
				t.Errorf("lastActivityAt not preserved: %v", st.Activity.LastActivityAt)
			}
		})
	}

	// Unknown activity state fails closed
	_, err := c.GetWorkerStatus(context.Background(), "sess-unknown-state")
	if !errors.Is(err, ErrProtocolViolation) {
		t.Errorf("expected ErrProtocolViolation on unknown activity state, got %v", err)
	}
	var getProtoErr *ProtocolError
	if !errors.As(err, &getProtoErr) {
		t.Fatalf("expected *ProtocolError for GetWorkerStatus unknown activity, got %T (%v)", err, err)
	}
	if getProtoErr.StatusCode != http.StatusOK {
		t.Errorf("expected StatusCode %d for GetWorkerStatus unknown activity, got %d", http.StatusOK, getProtoErr.StatusCode)
	}
	if getProtoErr.Method != http.MethodGet {
		t.Errorf("expected Method %s, got %s", http.MethodGet, getProtoErr.Method)
	}
	if getProtoErr.Path != "/api/v1/sessions/sess-unknown-state" {
		t.Errorf("expected Path %s, got %s", "/api/v1/sessions/sess-unknown-state", getProtoErr.Path)
	}

	// Mismatched session ID fails closed
	_, err = c.GetWorkerStatus(context.Background(), "sess-mismatch")
	if !errors.Is(err, ErrProtocolViolation) {
		t.Errorf("expected ErrProtocolViolation on mismatched session ID, got %v", err)
	}

	// Null session fails closed
	_, err = c.GetWorkerStatus(context.Background(), "sess-null")
	if !errors.Is(err, ErrProtocolViolation) {
		t.Errorf("expected ErrProtocolViolation on null session, got %v", err)
	}

	// Input validation: empty session ID fails
	if _, err := c.GetWorkerStatus(context.Background(), ""); !errors.Is(err, ErrBadRequest) {
		t.Errorf("expected ErrBadRequest on empty sessionID, got %v", err)
	}
	if _, err := c.GetWorkerStatus(context.Background(), "   "); !errors.Is(err, ErrBadRequest) {
		t.Errorf("expected ErrBadRequest on whitespace sessionID, got %v", err)
	}

	// Path segment escaping for /, ?, #, %
	escapedIDs := []string{
		"sess/slash",
		"sess?query",
		"sess#frag",
		"sess%pct",
	}
	for _, id := range escapedIDs {
		st, err := c.GetWorkerStatus(context.Background(), id)
		if err != nil {
			t.Fatalf("GetWorkerStatus(%q) failed: %v", id, err)
		}
		if st.ID != id {
			t.Errorf("expected session ID %q, got %q", id, st.ID)
		}
	}

	// 404 SESSION_NOT_FOUND matches ErrSessionNotFound and ErrNotFound
	_, err = c.GetWorkerStatus(context.Background(), "nonexistent")
	if !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("expected errors.Is(err, ErrSessionNotFound), got %v", err)
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected errors.Is(err, ErrNotFound), got %v", err)
	}
}

// -----------------------------------------------------------------------------
// 9. Error Envelope, Sanitization & Transport Cause Preservation (P03T1-004, P03T1R1-005)
// -----------------------------------------------------------------------------
func TestClient_ErrorHandlingAndSanitization(t *testing.T) {
	// 9.1 Structured AO API Error Envelope preservation
	t.Run("structured_api_errors", func(t *testing.T) {
		cases := []struct {
			name           string
			statusCode     int
			body           string
			expectedTarget error
			checkCode      string
		}{
			{
				name:           "bad_request_400",
				statusCode:     http.StatusBadRequest,
				body:           `{"error":"bad_request","code":"INVALID_PROMPT","message":"prompt exceeds max length","requestId":"req-123"}`,
				expectedTarget: ErrBadRequest,
				checkCode:      "INVALID_PROMPT",
			},
			{
				name:           "session_not_found_404",
				statusCode:     http.StatusNotFound,
				body:           `{"error":"not_found","code":"SESSION_NOT_FOUND","message":"session does not exist"}`,
				expectedTarget: ErrSessionNotFound,
				checkCode:      "SESSION_NOT_FOUND",
			},
			{
				name:           "project_not_found_404",
				statusCode:     http.StatusNotFound,
				body:           `{"error":"not_found","code":"PROJECT_NOT_FOUND","message":"project does not exist"}`,
				expectedTarget: ErrProjectNotFound,
				checkCode:      "PROJECT_NOT_FOUND",
			},
			{
				name:           "project_folder_missing_404",
				statusCode:     http.StatusNotFound,
				body:           `{"error":"not_found","code":"PROJECT_FOLDER_MISSING","message":"repo folder deleted from disk"}`,
				expectedTarget: ErrProjectNotFound,
				checkCode:      "PROJECT_FOLDER_MISSING",
			},
			{
				name:           "service_unavailable_503",
				statusCode:     http.StatusServiceUnavailable,
				body:           `{"error":"unavailable","code":"SERVICE_UNAVAILABLE","message":"daemon database locked"}`,
				expectedTarget: ErrDaemonUnavailable,
				checkCode:      "SERVICE_UNAVAILABLE",
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(tc.statusCode)
					w.Write([]byte(tc.body))
				}))
				defer s.Close()

				c, _ := newTestClient(t, s)
				_, err := c.CheckHealth(context.Background())
				if err == nil {
					t.Fatalf("expected error on status %d, got nil", tc.statusCode)
				}

				if !errors.Is(err, tc.expectedTarget) {
					t.Errorf("expected errors.Is(err, %v), got: %v", tc.expectedTarget, err)
				}

				var apiErr *APIError
				if !errors.As(err, &apiErr) {
					t.Fatalf("expected *APIError, got %T: %v", err, err)
				}
				if apiErr.Code != tc.checkCode {
					t.Errorf("expected APIError.Code %q, got %q", tc.checkCode, apiErr.Code)
				}
			})
		}
	})

	// 9.2 Verification that free-text substring does NOT classify error
	t.Run("no_substring_classification", func(t *testing.T) {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			// Notice: Code is ROUTE_NOT_FOUND, but message contains the words "session" and "project"
			w.Write([]byte(`{"error":"not_found","code":"ROUTE_NOT_FOUND","message":"route /api/v1/session/xyz is not found for project abc"}`))
		}))
		defer s.Close()

		c, _ := newTestClient(t, s)
		_, err := c.CheckHealth(context.Background())
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		// Must match generic ErrNotFound (404)
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expected errors.Is(err, ErrNotFound), got %v", err)
		}
		// MUST NOT match ErrSessionNotFound merely because message contains "session"
		if errors.Is(err, ErrSessionNotFound) {
			t.Errorf("violates contract: 404 ROUTE_NOT_FOUND matched ErrSessionNotFound based on message substring!")
		}
		// MUST NOT match ErrProjectNotFound merely because message contains "project"
		if errors.Is(err, ErrProjectNotFound) {
			t.Errorf("violates contract: 404 ROUTE_NOT_FOUND matched ErrProjectNotFound based on message substring!")
		}
	})

	// 9.3 Incomplete envelope missing required fields fails closed to sanitized ProtocolError
	t.Run("incomplete_envelope_fails_closed", func(t *testing.T) {
		incompleteCases := []struct {
			name string
			body string
		}{
			{"missing_error_field", `{"code":"FAIL","message":"something failed"}`},
			{"missing_code_field", `{"error":"internal","message":"something failed"}`},
			{"missing_message_field", `{"error":"internal","code":"FAIL"}`},
		}

		for _, tc := range incompleteCases {
			t.Run(tc.name, func(t *testing.T) {
				s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(tc.body))
				}))
				defer s.Close()

				c, _ := newTestClient(t, s)
				_, err := c.CheckHealth(context.Background())
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !errors.Is(err, ErrProtocolViolation) {
					t.Errorf("expected errors.Is(err, ErrProtocolViolation), got %v", err)
				}
				var protoErr *ProtocolError
				if !errors.As(err, &protoErr) {
					t.Errorf("expected *ProtocolError, got %T: %v", err, err)
				}
			})
		}
	})

	// 9.4 Sanitization: malformed JSON and arbitrary HTML/plaintext do NOT leak into error string
	t.Run("sanitized_error_does_not_leak_raw_body", func(t *testing.T) {
		secretText := "SECRET_AWS_KEY=AKIAIOSFODNN7EXAMPLE_DO_NOT_LEAK"
		htmlBody := fmt.Sprintf("<html><body><h1>502 Bad Gateway</h1><p>%s</p></body></html>", secretText)

		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusBadGateway)
			w.Write([]byte(htmlBody))
		}))
		defer s.Close()

		c, _ := newTestClient(t, s)
		_, err := c.CheckHealth(context.Background())
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		errStr := err.Error()
		if strings.Contains(errStr, secretText) {
			t.Fatalf("SECURITY VIOLATION: upstream error body leaked in error string: %s", errStr)
		}
		if strings.Contains(errStr, "<html>") {
			t.Fatalf("SECURITY VIOLATION: raw HTML leaked in error string: %s", errStr)
		}

		if !errors.Is(err, ErrProtocolViolation) {
			t.Errorf("expected errors.Is(err, ErrProtocolViolation), got %v", err)
		}
	})

	// 9.5 Transport Error cause preservation & unwrap chaining
	t.Run("transport_error_unwrap_chain", func(t *testing.T) {
		// Server that immediately closes connection
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hj, ok := w.(http.Hijacker)
			if ok {
				conn, _, _ := hj.Hijack()
				conn.Close()
				return
			}
		}))
		defer s.Close()

		c, _ := newTestClient(t, s)

		// 9.5.1 Network failure is distinguishable from API error and matches ErrDaemonUnavailable
		_, netErr := c.CheckHealth(context.Background())
		if netErr == nil {
			t.Fatal("expected network error, got nil")
		}
		var transportErr *TransportError
		if !errors.As(netErr, &transportErr) {
			t.Fatalf("expected *TransportError, got %T: %v", netErr, netErr)
		}
		if !errors.Is(netErr, ErrDaemonUnavailable) {
			t.Errorf("expected transport error to match ErrDaemonUnavailable")
		}

		// 9.5.2 Context cancellation is discoverable via errors.Is(err, context.Canceled)
		cancelCtx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, cancelErr := c.CheckHealth(cancelCtx)
		if cancelErr == nil {
			t.Fatal("expected error on cancelled context, got nil")
		}
		if !errors.Is(cancelErr, context.Canceled) {
			t.Errorf("expected errors.Is(cancelErr, context.Canceled) == true, got: %v", cancelErr)
		}

		// 9.5.3 Context deadline exceeded is discoverable via errors.Is(err, context.DeadlineExceeded)
		deadlineCtx, dCancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
		defer dCancel()

		_, deadlineErr := c.CheckHealth(deadlineCtx)
		if deadlineErr == nil {
			t.Fatal("expected error on expired context, got nil")
		}
		if !errors.Is(deadlineErr, context.DeadlineExceeded) {
			t.Errorf("expected errors.Is(deadlineErr, context.DeadlineExceeded) == true, got: %v", deadlineErr)
		}
	})
}
func TestClient_GetWorkspaceFile(t *testing.T) {
	validEnvelopeMap := func(sessionID, filePath, content string) map[string]any {
		return map[string]any{
			"sessionId":        sessionID,
			"path":             filePath,
			"content":          content,
			"binary":           false,
			"deleted":          false,
			"contentTruncated": false,
			"size":             int64(len(content)),
			"status":           "unmodified",
			"workspaceVersion": "wv-1",
			"diff":             "--- old\n+++ new\n+added diff lines\n",
			"diffTruncated":    false,
			"editable":         true,
			"fileFingerprint":  "fp-12345",
			"additions":        1,
			"deletions":        0,
		}
	}

	t.Run("successful_retrieval_with_large_diff_and_small_content", func(t *testing.T) {
		sessionID := "sess-ws-1"
		filePath := "src/main.go"
		content := "package main\n\nfunc main() {}\n"
		largeDiff := strings.Repeat("+diff line\n", 500)

		m := validEnvelopeMap(sessionID, filePath, content)
		m["diff"] = largeDiff

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Fatalf("expected GET, got %s", r.Method)
			}
			if r.URL.Path != "/api/v1/sessions/sess-ws-1/workspace/file" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			if r.URL.Query().Get("path") != filePath {
				t.Fatalf("unexpected query path: %s", r.URL.Query().Get("path"))
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(m)
		}))
		defer ts.Close()

		client, err := NewClient(ts.URL, ts.Client())
		if err != nil {
			t.Fatalf("NewClient failed: %v", err)
		}

		opts := WorkspaceReadOptions{
			MaxWireBytes: 64 * 1024,
			MaxBytes:     1024,
		}
		gotBytes, err := client.GetWorkspaceFile(context.Background(), sessionID, filePath, opts)
		if err != nil {
			t.Fatalf("GetWorkspaceFile failed: %v", err)
		}
		if string(gotBytes) != content {
			t.Fatalf("content mismatch: got %q, want %q", string(gotBytes), content)
		}
	})

	t.Run("wire_overflow_rejects_with_ErrPayloadTooLarge", func(t *testing.T) {
		sessionID := "sess-ws-wire-overflow"
		filePath := "test.txt"
		content := "small content"
		largeDiff := strings.Repeat("x", 5000)

		m := validEnvelopeMap(sessionID, filePath, content)
		m["diff"] = largeDiff

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(m)
		}))
		defer ts.Close()

		client, err := NewClient(ts.URL, ts.Client())
		if err != nil {
			t.Fatalf("NewClient failed: %v", err)
		}

		opts := WorkspaceReadOptions{
			MaxWireBytes: 200, // Small wire limit
			MaxBytes:     1000,
		}
		_, err = client.GetWorkspaceFile(context.Background(), sessionID, filePath, opts)
		if err == nil {
			t.Fatal("expected ErrPayloadTooLarge on wire overflow, got nil")
		}
		if !errors.Is(err, ErrPayloadTooLarge) {
			t.Fatalf("expected ErrPayloadTooLarge, got: %v", err)
		}
	})

	t.Run("content_overflow_rejects_with_ErrPayloadTooLarge_even_with_valid_wire", func(t *testing.T) {
		sessionID := "sess-ws-content-overflow"
		filePath := "large.txt"
		content := strings.Repeat("a", 2000)

		m := validEnvelopeMap(sessionID, filePath, content)

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(m)
		}))
		defer ts.Close()

		client, err := NewClient(ts.URL, ts.Client())
		if err != nil {
			t.Fatalf("NewClient failed: %v", err)
		}

		opts := WorkspaceReadOptions{
			MaxWireBytes: 64 * 1024, // Wire envelope fits
			MaxBytes:     500,       // Content limit exceeded
		}
		_, err = client.GetWorkspaceFile(context.Background(), sessionID, filePath, opts)
		if err == nil {
			t.Fatal("expected ErrPayloadTooLarge on content overflow, got nil")
		}
		if !errors.Is(err, ErrPayloadTooLarge) {
			t.Fatalf("expected ErrPayloadTooLarge, got: %v", err)
		}
	})

	t.Run("exact_boundaries_wire_and_content", func(t *testing.T) {
		sessionID := "sess-ws-boundaries"
		filePath := "boundary.txt"
		content := "exact10chr"

		m := validEnvelopeMap(sessionID, filePath, content)
		bodyBytes, err := json.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		exactWireLen := int64(len(bodyBytes))

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(bodyBytes)
		}))
		defer ts.Close()

		client, err := NewClient(ts.URL, ts.Client())
		if err != nil {
			t.Fatalf("NewClient failed: %v", err)
		}

		// Exact wire limit matches: should succeed
		got, err := client.GetWorkspaceFile(context.Background(), sessionID, filePath, WorkspaceReadOptions{
			MaxWireBytes: exactWireLen,
			MaxBytes:     int64(len(content)),
		})
		if err != nil {
			t.Fatalf("exact boundary match failed: %v", err)
		}
		if string(got) != content {
			t.Fatalf("content mismatch: got %q, want %q", string(got), content)
		}

		// Wire limit is 1 byte less: should fail with ErrPayloadTooLarge
		_, err = client.GetWorkspaceFile(context.Background(), sessionID, filePath, WorkspaceReadOptions{
			MaxWireBytes: exactWireLen - 1,
			MaxBytes:     int64(len(content)),
		})
		if !errors.Is(err, ErrPayloadTooLarge) {
			t.Fatalf("expected ErrPayloadTooLarge for wire-1, got %v", err)
		}

		// Content limit is 1 byte less: should fail with ErrPayloadTooLarge
		_, err = client.GetWorkspaceFile(context.Background(), sessionID, filePath, WorkspaceReadOptions{
			MaxWireBytes: exactWireLen,
			MaxBytes:     int64(len(content)) - 1,
		})
		if !errors.Is(err, ErrPayloadTooLarge) {
			t.Fatalf("expected ErrPayloadTooLarge for content-1, got %v", err)
		}
	})

	t.Run("missing_required_fields_fails_closed", func(t *testing.T) {
		requiredFields := []string{
			"sessionId", "path", "content", "binary", "deleted",
			"contentTruncated", "size", "status", "workspaceVersion", "diff",
			"diffTruncated", "editable", "fileFingerprint", "additions", "deletions",
		}

		for _, missingField := range requiredFields {
			t.Run("missing_"+missingField, func(t *testing.T) {
				m := validEnvelopeMap("sess-1", "file.txt", "content")
				delete(m, missingField)

				ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode(m)
				}))
				defer ts.Close()

				client, err := NewClient(ts.URL, ts.Client())
				if err != nil {
					t.Fatal(err)
				}

				_, err = client.GetWorkspaceFile(context.Background(), "sess-1", "file.txt", WorkspaceReadOptions{
					MaxWireBytes: 10000,
					MaxBytes:     1000,
				})
				if err == nil {
					t.Fatalf("expected ProtocolError when %s is missing, got nil", missingField)
				}
				var protoErr *ProtocolError
				if !errors.As(err, &protoErr) {
					t.Fatalf("expected *ProtocolError when %s is missing, got: %T (%v)", missingField, err, err)
				}
				if !strings.Contains(protoErr.Reason, "missing required field") {
					t.Fatalf("unexpected reason: %s", protoErr.Reason)
				}
			})
		}
	})

	t.Run("trailing_payload_and_second_json_value_rejected", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			m := validEnvelopeMap("sess-1", "file.txt", "hello")
			b, _ := json.Marshal(m)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(b)
			_, _ = w.Write([]byte(" {\"second\":\"value\"}"))
		}))
		defer ts.Close()

		client, err := NewClient(ts.URL, ts.Client())
		if err != nil {
			t.Fatal(err)
		}

		_, err = client.GetWorkspaceFile(context.Background(), "sess-1", "file.txt", WorkspaceReadOptions{
			MaxWireBytes: 10000,
			MaxBytes:     1000,
		})
		if err == nil {
			t.Fatal("expected ProtocolError on trailing payload, got nil")
		}
		var protoErr *ProtocolError
		if !errors.As(err, &protoErr) {
			t.Fatalf("expected *ProtocolError, got %v", err)
		}
		if !strings.Contains(protoErr.Reason, "trailing data") {
			t.Fatalf("unexpected reason: %s", protoErr.Reason)
		}
	})

	t.Run("status_enum_validation", func(t *testing.T) {
		for _, status := range []string{"unmodified", "modified", "added"} {
			t.Run("valid_"+status, func(t *testing.T) {
				m := validEnvelopeMap("sess-1", "file.txt", "content")
				m["status"] = status

				ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode(m)
				}))
				defer ts.Close()

				client, _ := NewClient(ts.URL, ts.Client())
				_, err := client.GetWorkspaceFile(context.Background(), "sess-1", "file.txt", WorkspaceReadOptions{
					MaxWireBytes: 10000,
					MaxBytes:     1000,
				})
				if err != nil {
					t.Fatalf("valid status %s failed: %v", status, err)
				}
			})
		}

		t.Run("invalid_status_enum", func(t *testing.T) {
			m := validEnvelopeMap("sess-1", "file.txt", "content")
			m["status"] = "corrupted_status"

			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(m)
			}))
			defer ts.Close()

			client, _ := NewClient(ts.URL, ts.Client())
			_, err := client.GetWorkspaceFile(context.Background(), "sess-1", "file.txt", WorkspaceReadOptions{
				MaxWireBytes: 10000,
				MaxBytes:     1000,
			})
			if err == nil {
				t.Fatal("expected ProtocolError for invalid status, got nil")
			}
			var protoErr *ProtocolError
			if !errors.As(err, &protoErr) {
				t.Fatalf("expected ProtocolError, got: %v", err)
			}
		})
	})

	t.Run("deleted_file_returns_ErrWorkspaceFileDeleted_not_APIError_404", func(t *testing.T) {
		// Test deleted=true with status="unmodified"
		m1 := validEnvelopeMap("sess-1", "file.txt", "")
		m1["deleted"] = true

		ts1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(m1)
		}))
		defer ts1.Close()

		client1, _ := NewClient(ts1.URL, ts1.Client())
		_, err1 := client1.GetWorkspaceFile(context.Background(), "sess-1", "file.txt", WorkspaceReadOptions{
			MaxWireBytes: 10000,
			MaxBytes:     1000,
		})
		if !errors.Is(err1, ErrWorkspaceFileDeleted) {
			t.Fatalf("expected ErrWorkspaceFileDeleted for deleted=true, got: %v", err1)
		}
		if errors.Is(err1, ErrNotFound) {
			t.Fatal("ErrWorkspaceFileDeleted must NOT match ErrNotFound or fake APIError/404")
		}

		// Test deleted=false with status="deleted"
		m2 := validEnvelopeMap("sess-1", "file.txt", "")
		m2["status"] = "deleted"

		ts2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(m2)
		}))
		defer ts2.Close()

		client2, _ := NewClient(ts2.URL, ts2.Client())
		_, err2 := client2.GetWorkspaceFile(context.Background(), "sess-1", "file.txt", WorkspaceReadOptions{
			MaxWireBytes: 10000,
			MaxBytes:     1000,
		})
		if !errors.Is(err2, ErrWorkspaceFileDeleted) {
			t.Fatalf("expected ErrWorkspaceFileDeleted for status=deleted, got: %v", err2)
		}
	})

	t.Run("binary_file_returns_ProtocolError", func(t *testing.T) {
		m := validEnvelopeMap("sess-1", "file.bin", "binarycontent")
		m["binary"] = true

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(m)
		}))
		defer ts.Close()

		client, _ := NewClient(ts.URL, ts.Client())
		_, err := client.GetWorkspaceFile(context.Background(), "sess-1", "file.bin", WorkspaceReadOptions{
			MaxWireBytes: 10000,
			MaxBytes:     1000,
		})
		var protoErr *ProtocolError
		if !errors.As(err, &protoErr) || !strings.Contains(protoErr.Reason, "binary") {
			t.Fatalf("expected ProtocolError with binary reason, got: %v", err)
		}
	})

	t.Run("content_truncated_returns_ProtocolError", func(t *testing.T) {
		m := validEnvelopeMap("sess-1", "file.txt", "truncated content")
		m["contentTruncated"] = true

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(m)
		}))
		defer ts.Close()

		client, _ := NewClient(ts.URL, ts.Client())
		_, err := client.GetWorkspaceFile(context.Background(), "sess-1", "file.txt", WorkspaceReadOptions{
			MaxWireBytes: 10000,
			MaxBytes:     1000,
		})
		var protoErr *ProtocolError
		if !errors.As(err, &protoErr) || !strings.Contains(protoErr.Reason, "truncated") {
			t.Fatalf("expected ProtocolError with truncated reason, got: %v", err)
		}
	})

	t.Run("session_and_path_mismatch_returns_ProtocolError", func(t *testing.T) {
		m := validEnvelopeMap("sess-different", "file.txt", "content")

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(m)
		}))
		defer ts.Close()

		client, _ := NewClient(ts.URL, ts.Client())
		_, err := client.GetWorkspaceFile(context.Background(), "sess-expected", "file.txt", WorkspaceReadOptions{
			MaxWireBytes: 10000,
			MaxBytes:     1000,
		})
		var protoErr *ProtocolError
		if !errors.As(err, &protoErr) || !strings.Contains(protoErr.Reason, "sessionId") {
			t.Fatalf("expected ProtocolError for sessionId mismatch, got: %v", err)
		}
	})

	t.Run("upstream_http_errors_properly_decoded_into_APIError", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error":     "NotFound",
				"code":      "FILE_NOT_FOUND",
				"message":   "the requested file does not exist",
				"requestId": "req-1234",
			})
		}))
		defer ts.Close()

		client, _ := NewClient(ts.URL, ts.Client())
		_, err := client.GetWorkspaceFile(context.Background(), "sess-1", "missing.txt", WorkspaceReadOptions{
			MaxWireBytes: 10000,
			MaxBytes:     1000,
		})
		var apiErr *APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("expected *APIError, got %T: %v", err, err)
		}
		if apiErr.StatusCode != http.StatusNotFound || apiErr.Code != "FILE_NOT_FOUND" {
			t.Fatalf("unexpected APIError fields: %+v", apiErr)
		}
		if !errors.Is(err, ErrNotFound) {
			t.Fatal("expected errors.Is(err, ErrNotFound) to match")
		}
	})

	t.Run("options_validation_rejects_invalid_parameters", func(t *testing.T) {
		client, _ := NewClient("http://127.0.0.1:3001", http.DefaultClient)

		// Empty sessionID
		_, err := client.GetWorkspaceFile(context.Background(), "", "file.txt", WorkspaceReadOptions{MaxWireBytes: 10, MaxBytes: 10})
		if !errors.Is(err, ErrBadRequest) {
			t.Fatalf("expected ErrBadRequest for empty sessionID, got %v", err)
		}

		// Empty filePath
		_, err = client.GetWorkspaceFile(context.Background(), "sess", "", WorkspaceReadOptions{MaxWireBytes: 10, MaxBytes: 10})
		if !errors.Is(err, ErrBadRequest) {
			t.Fatalf("expected ErrBadRequest for empty filePath, got %v", err)
		}

		// MaxWireBytes <= 0
		_, err = client.GetWorkspaceFile(context.Background(), "sess", "file.txt", WorkspaceReadOptions{MaxWireBytes: 0, MaxBytes: 10})
		if !errors.Is(err, ErrBadRequest) {
			t.Fatalf("expected ErrBadRequest for MaxWireBytes=0, got %v", err)
		}

		// MaxBytes <= 0
		_, err = client.GetWorkspaceFile(context.Background(), "sess", "file.txt", WorkspaceReadOptions{MaxWireBytes: 10, MaxBytes: -5})
		if !errors.Is(err, ErrBadRequest) {
			t.Fatalf("expected ErrBadRequest for MaxBytes < 0, got %v", err)
		}

		// MaxWireBytes integer overflow
		_, err = client.GetWorkspaceFile(context.Background(), "sess", "file.txt", WorkspaceReadOptions{MaxWireBytes: math.MaxInt64, MaxBytes: 10})
		if !errors.Is(err, ErrBadRequest) {
			t.Fatalf("expected ErrBadRequest for MaxWireBytes=MaxInt64, got %v", err)
		}
	})

	t.Run("context_cancellation_fails_closed", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(50 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		client, _ := NewClient(ts.URL, ts.Client())
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := client.GetWorkspaceFile(ctx, "sess-1", "file.txt", WorkspaceReadOptions{
			MaxWireBytes: 10000,
			MaxBytes:     1000,
		})
		if err == nil {
			t.Fatal("expected error on cancelled context, got nil")
		}
		var transportErr *TransportError
		if !errors.As(err, &transportErr) {
			t.Fatalf("expected *TransportError, got %T: %v", err, err)
		}
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled unwrap, got %v", err)
		}
	})
}
