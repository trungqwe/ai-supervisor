package ao

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestNewClient_ConstructorAndLoopbackSecurity(t *testing.T) {
	baseClient := &http.Client{
		Timeout: 42 * time.Second,
	}

	tests := []struct {
		name       string
		url        string
		client     *http.Client
		wantErr    error
		errContain string
	}{
		// PASS cases
		{
			name:    "PASS: standard IPv4 loopback with port",
			url:     "http://127.0.0.1:3001",
			client:  baseClient,
			wantErr: nil,
		},
		{
			name:    "PASS: secondary IPv4 loopback with port",
			url:     "http://127.0.0.2:8080",
			client:  baseClient,
			wantErr: nil,
		},
		{
			name:    "PASS: IPv6 loopback with port",
			url:     "http://[::1]:3001",
			client:  baseClient,
			wantErr: nil,
		},
		{
			name:    "PASS: IPv4 loopback with root slash",
			url:     "http://127.0.0.1:3001/",
			client:  baseClient,
			wantErr: nil,
		},

		// FAIL cases per contract
		{
			name:       "FAIL: nil httpClient rejected",
			url:        "http://127.0.0.1:3001",
			client:     nil,
			wantErr:    ErrBadRequest,
			errContain: "httpClient must not be nil",
		},
		{
			name:       "FAIL: empty URL rejected",
			url:        "",
			client:     baseClient,
			wantErr:    ErrBadRequest,
			errContain: "base URL cannot be empty",
		},
		{
			name:       "FAIL: localhost hostname rejected",
			url:        "http://localhost:3001",
			client:     baseClient,
			wantErr:    ErrNonLoopbackURL,
			errContain: "hostnames forbidden",
		},
		{
			name:       "FAIL: HTTPS scheme rejected",
			url:        "https://127.0.0.1:3001",
			client:     baseClient,
			wantErr:    ErrNonLoopbackURL,
			errContain: "must be http only",
		},
		{
			name:       "FAIL: remote hostname rejected",
			url:        "http://example.com:3001",
			client:     baseClient,
			wantErr:    ErrNonLoopbackURL,
			errContain: "hostnames forbidden",
		},
		{
			name:       "FAIL: remote IP rejected",
			url:        "http://192.168.1.5:3001",
			client:     baseClient,
			wantErr:    ErrNonLoopbackURL,
			errContain: "not a loopback address",
		},
		{
			name:       "FAIL: wildcard 0.0.0.0 rejected",
			url:        "http://0.0.0.0:3001",
			client:     baseClient,
			wantErr:    ErrNonLoopbackURL,
			errContain: "not a loopback address",
		},
		{
			name:       "FAIL: missing port rejected",
			url:        "http://127.0.0.1",
			client:     baseClient,
			wantErr:    ErrNonLoopbackURL,
			errContain: "explicit port required",
		},
		{
			name:       "FAIL: invalid non-numeric port rejected",
			url:        "http://127.0.0.1:abc",
			client:     baseClient,
			wantErr:    ErrNonLoopbackURL,
			errContain: "invalid port",
		},
		{
			name:       "FAIL: port 0 rejected",
			url:        "http://127.0.0.1:0",
			client:     baseClient,
			wantErr:    ErrNonLoopbackURL,
			errContain: "invalid port",
		},
		{
			name:       "FAIL: port > 65535 rejected",
			url:        "http://127.0.0.1:70000",
			client:     baseClient,
			wantErr:    ErrNonLoopbackURL,
			errContain: "invalid port",
		},
		{
			name:       "FAIL: userinfo rejected",
			url:        "http://user:pass@127.0.0.1:3001",
			client:     baseClient,
			wantErr:    ErrNonLoopbackURL,
			errContain: "userinfo in base URL is forbidden",
		},
		{
			name:       "FAIL: non-root base path rejected",
			url:        "http://127.0.0.1:3001/api",
			client:     baseClient,
			wantErr:    ErrNonLoopbackURL,
			errContain: "non-root base path",
		},
		{
			name:       "FAIL: query rejected",
			url:        "http://127.0.0.1:3001?x=1",
			client:     baseClient,
			wantErr:    ErrNonLoopbackURL,
			errContain: "query string in base URL is forbidden",
		},
		{
			name:       "FAIL: fragment rejected",
			url:        "http://127.0.0.1:3001#fragment",
			client:     baseClient,
			wantErr:    ErrNonLoopbackURL,
			errContain: "fragment in base URL is forbidden",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := NewClient(tt.url, tt.client)
			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("error %v does not match wantErr %v", err, tt.wantErr)
				}
				if tt.errContain != "" && !strings.Contains(err.Error(), tt.errContain) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errContain)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if c == nil {
					t.Fatalf("expected non-nil client")
				}
				if strings.HasSuffix(c.BaseURL(), "/") {
					t.Errorf("BaseURL() %q has trailing slash", c.BaseURL())
				}
			}
		})
	}
}

func TestClient_CallerObjectPreservationAndRedirectSafety(t *testing.T) {
	callerTransport := &http.Transport{}
	callerClient := &http.Client{
		Transport: callerTransport,
		Timeout:   27 * time.Second,
	}

	c, err := NewClient("http://127.0.0.1:3001", callerClient)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	// Verify caller-owned client object is NOT mutated
	if callerClient.Timeout != 27*time.Second {
		t.Errorf("callerClient.Timeout mutated to %v", callerClient.Timeout)
	}
	if callerClient.CheckRedirect != nil {
		t.Errorf("callerClient.CheckRedirect was mutated")
	}

	// Verify adapter client preserved Transport and Timeout
	if c.httpClient.Timeout != 27*time.Second {
		t.Errorf("adapter client timeout = %v, want 27s", c.httpClient.Timeout)
	}
	if c.httpClient.Transport != callerTransport {
		t.Errorf("adapter client Transport was not preserved")
	}

	// Redirect safety: loopback endpoint returns redirect to external host
	externalDestinationHit := false
	externalServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		externalDestinationHit = true
		w.WriteHeader(http.StatusOK)
	}))
	defer externalServer.Close()

	loopbackServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, externalServer.URL+"/escaped", http.StatusFound)
	}))
	defer loopbackServer.Close()

	// Parse loopbackServer URL to extract IP and port
	u, err := url.Parse(loopbackServer.URL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}
	host, port, _ := net.SplitHostPort(u.Host)
	loopbackURL := fmt.Sprintf("http://%s:%s", host, port)

	adapter, err := NewClient(loopbackURL, callerClient)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	var dummy map[string]any
	err = adapter.get(context.Background(), "/test-redirect", &dummy)
	if err == nil {
		t.Fatalf("expected redirect error, got nil")
	}
	if !errors.Is(err, ErrRedirectAttempted) {
		t.Errorf("expected ErrRedirectAttempted, got %v", err)
	}
	if externalDestinationHit {
		t.Fatalf("SECURITY VIOLATION: redirect to external host was followed!")
	}
}

func TestClient_CheckHealth(t *testing.T) {
	// Success case
	tsOK := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/healthz" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":  "ok",
			"service": "agent-orchestrator-daemon",
			"pid":     8888,
		})
	}))
	defer tsOK.Close()

	c, err := NewClient(tsOK.URL, &http.Client{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	h, err := c.CheckHealth(context.Background())
	if err != nil {
		t.Fatalf("CheckHealth: %v", err)
	}
	if h.Status != "ok" || h.Service != "agent-orchestrator-daemon" || h.PID != 8888 {
		t.Errorf("unexpected health status: %+v", h)
	}

	// Malformed JSON failure
	tsBadJSON := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{invalid-json"))
	}))
	defer tsBadJSON.Close()

	cBadJSON, _ := NewClient(tsBadJSON.URL, &http.Client{Timeout: 5 * time.Second})
	_, err = cBadJSON.CheckHealth(context.Background())
	if err == nil || !errors.Is(err, ErrProtocolViolation) {
		t.Errorf("expected ErrProtocolViolation on malformed JSON, got %v", err)
	}

	// Missing status failure
	tsMissing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"service": "agent-orchestrator-daemon",
		})
	}))
	defer tsMissing.Close()

	cMissing, _ := NewClient(tsMissing.URL, &http.Client{Timeout: 5 * time.Second})
	_, err = cMissing.CheckHealth(context.Background())
	if err == nil || !errors.Is(err, ErrProtocolViolation) {
		t.Errorf("expected ErrProtocolViolation on missing status, got %v", err)
	}

	// Wrong status failure
	tsWrong := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":  "degraded",
			"service": "agent-orchestrator-daemon",
		})
	}))
	defer tsWrong.Close()

	cWrong, _ := NewClient(tsWrong.URL, &http.Client{Timeout: 5 * time.Second})
	_, err = cWrong.CheckHealth(context.Background())
	if err == nil || !errors.Is(err, ErrProtocolViolation) {
		t.Errorf("expected ErrProtocolViolation on status != ok, got %v", err)
	}
}

func TestClient_CheckReadiness(t *testing.T) {
	// Success case
	tsOK := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/readyz" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":  "ready",
			"service": "agent-orchestrator-daemon",
			"pid":     8888,
		})
	}))
	defer tsOK.Close()

	c, err := NewClient(tsOK.URL, &http.Client{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	r, err := c.CheckReadiness(context.Background())
	if err != nil {
		t.Fatalf("CheckReadiness: %v", err)
	}
	if r.Status != "ready" || r.PID != 8888 {
		t.Errorf("unexpected readiness status: %+v", r)
	}

	// Wrong status failure
	tsWrong := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "not_ready",
		})
	}))
	defer tsWrong.Close()

	cWrong, _ := NewClient(tsWrong.URL, &http.Client{Timeout: 5 * time.Second})
	_, err = cWrong.CheckReadiness(context.Background())
	if err == nil || !errors.Is(err, ErrProtocolViolation) {
		t.Errorf("expected ErrProtocolViolation on wrong readyz status, got %v", err)
	}
}

func TestClient_ListAgents(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/agents" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(AgentInventory{
			Supported: []AgentInfo{
				{ID: "agy", Label: "Antigravity", AuthStatus: "authorized"},
				{ID: "codex", Label: "Codex", AuthStatus: "authorized"},
			},
			Installed: []AgentInfo{
				{ID: "agy", Label: "Antigravity", AuthStatus: "authorized"},
			},
			Authorized: []AgentInfo{
				{ID: "agy", Label: "Antigravity", AuthStatus: "authorized"},
			},
		})
	}))
	defer ts.Close()

	c, err := NewClient(ts.URL, &http.Client{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	inv, err := c.ListAgents(context.Background())
	if err != nil {
		t.Fatalf("ListAgents: %v", err)
	}
	if len(inv.Supported) != 2 || len(inv.Installed) != 1 || len(inv.Authorized) != 1 {
		t.Errorf("unexpected agent inventory counts: %+v", inv)
	}
	if inv.Installed[0].ID != "agy" {
		t.Errorf("installed agent ID = %q, want agy", inv.Installed[0].ID)
	}
}

func TestClient_GetAgentReadiness(t *testing.T) {
	postCalled := false
	checkedAt := time.Now().UTC().Add(-1 * time.Minute).Truncate(time.Second)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			postCalled = true
			t.Errorf("FORBIDDEN: POST was called on readiness endpoint")
		}
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/agents/readiness" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(rawAgentReadinessResponse{
			Agents: []AgentReadinessSnapshot{
				{
					ID:    "agy",
					Label: "Antigravity",
					Installation: AgentInstallationObservation{
						State:      "installed",
						Freshness:  "fresh",
						CheckedAt:  &checkedAt,
						ReasonCode: "FOUND",
					},
					Authentication: AgentAuthenticationObservation{
						State:     "authorized",
						Freshness: "fresh",
						CheckedAt: &checkedAt,
					},
					EffectiveReadiness: "ready",
					UsageCount:         10,
				},
				{
					ID:    "claude",
					Label: "Claude Code",
					Installation: AgentInstallationObservation{
						State:     "not_installed",
						Freshness: "fresh",
					},
					Authentication: AgentAuthenticationObservation{
						State:     "unknown",
						Freshness: "stale",
					},
					EffectiveReadiness: "not_ready",
				},
			},
		})
	}))
	defer ts.Close()

	c, err := NewClient(ts.URL, &http.Client{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	// 1. Success matching agentID
	snap, err := c.GetAgentReadiness(context.Background(), "agy")
	if err != nil {
		t.Fatalf("GetAgentReadiness('agy'): %v", err)
	}
	if snap.ID != "agy" || snap.EffectiveReadiness != "ready" {
		t.Errorf("unexpected snapshot: %+v", snap)
	}
	if snap.Installation.State != "installed" || snap.Authentication.State != "authorized" {
		t.Errorf("unexpected observation states: %+v", snap)
	}
	if snap.UsageCount != 10 {
		t.Errorf("UsageCount = %d, want 10", snap.UsageCount)
	}

	// 2. Absent agent returns ErrAgentNotFound
	_, err = c.GetAgentReadiness(context.Background(), "missing-agent")
	if err == nil || !errors.Is(err, ErrAgentNotFound) {
		t.Errorf("expected ErrAgentNotFound on missing agent, got %v", err)
	}

	// 3. Empty agentID returns ErrBadRequest
	_, err = c.GetAgentReadiness(context.Background(), "   ")
	if err == nil || !errors.Is(err, ErrBadRequest) {
		t.Errorf("expected ErrBadRequest on empty agentID, got %v", err)
	}

	// 4. Verify no POST was called
	if postCalled {
		t.Errorf("POST /readiness/ensure was called during readiness inspection")
	}
}

func TestClient_GetAPIContract(t *testing.T) {
	mockSchema := "openapi: 3.0.3\ninfo:\n  title: Agent Orchestrator\n  version: 0.13.0\n"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/openapi.yaml" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/yaml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockSchema))
	}))
	defer ts.Close()

	c, err := NewClient(ts.URL, &http.Client{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	contract, err := c.GetAPIContract(context.Background())
	if err != nil {
		t.Fatalf("GetAPIContract: %v", err)
	}
	if contract != mockSchema {
		t.Errorf("contract = %q, want %q", contract, mockSchema)
	}
}

func TestClient_RegisterProject(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/projects" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}

		bodyBytes, _ := io.ReadAll(r.Body)
		var wireMap map[string]any
		if err := json.Unmarshal(bodyBytes, &wireMap); err != nil {
			t.Errorf("unmarshal request body: %v", err)
		}

		// Verify EXACT wire keys: path, projectId only
		if len(wireMap) != 2 {
			t.Errorf("wire payload has %d keys, want exactly 2 (path, projectId). Keys: %+v", len(wireMap), wireMap)
		}
		if _, ok := wireMap["path"]; !ok {
			t.Errorf("missing path key")
		}
		if _, ok := wireMap["projectId"]; !ok {
			t.Errorf("missing projectId key")
		}
		if _, forbidden := wireMap["name"]; forbidden {
			t.Errorf("forbidden key 'name' present in request")
		}
		if _, forbidden := wireMap["config"]; forbidden {
			t.Errorf("forbidden key 'config' present in request")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated) // HTTP 201
		_ = json.NewEncoder(w).Encode(map[string]any{
			"project": map[string]any{
				"id":            wireMap["projectId"],
				"name":          "My Project",
				"kind":          "single_repo",
				"path":          wireMap["path"],
				"repo":          "my-repo",
				"defaultBranch": "main",
			},
		})
	}))
	defer ts.Close()

	c, err := NewClient(ts.URL, &http.Client{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	// 1. Success case
	p, err := c.RegisterProject(context.Background(), "proj-1", "/work/proj-1")
	if err != nil {
		t.Fatalf("RegisterProject: %v", err)
	}
	if p.ID != "proj-1" || p.Path != "/work/proj-1" || p.IsDegraded {
		t.Errorf("unexpected registered project: %+v", p)
	}

	// 2. Reject empty projectID
	_, err = c.RegisterProject(context.Background(), "  ", "/work/proj-1")
	if err == nil || !errors.Is(err, ErrBadRequest) {
		t.Errorf("expected ErrBadRequest on empty projectID, got %v", err)
	}

	// 3. Reject empty rootPath
	_, err = c.RegisterProject(context.Background(), "proj-1", "   ")
	if err == nil || !errors.Is(err, ErrBadRequest) {
		t.Errorf("expected ErrBadRequest on empty rootPath, got %v", err)
	}
}

func TestClient_GetProject(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", r.Method)
		}

		reqURI := r.URL.RequestURI()

		switch {
		case strings.HasPrefix(reqURI, "/api/v1/projects/healthy-proj"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": "ok",
				"project": map[string]any{
					"id":            "healthy-proj",
					"name":          "Healthy Project",
					"path":          "/repo/healthy",
					"defaultBranch": "main",
					"folderMissing": false,
				},
			})

		case strings.HasPrefix(reqURI, "/api/v1/projects/degraded-proj"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": "degraded",
				"project": map[string]any{
					"id":           "degraded-proj",
					"name":         "Degraded Project",
					"path":         "/repo/missing",
					"resolveError": "git repository folder missing from disk",
				},
			})

		case strings.HasPrefix(reqURI, "/api/v1/projects/unknown-status"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": "pending_validation",
				"project": map[string]any{
					"id": "unknown-status",
				},
			})

		case strings.HasPrefix(reqURI, "/api/v1/projects/missing-proj"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(APIError{
				ErrorType: "not_found",
				Code:      "PROJECT_NOT_FOUND",
				Message:   "project missing-proj does not exist",
			})

		case strings.HasPrefix(reqURI, "/api/v1/projects/complex%2Fname%3Fwith%23chars"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": "ok",
				"project": map[string]any{
					"id":   "complex/name?with#chars",
					"name": "Escaped Project",
					"path": "/repo/complex",
				},
			})

		default:
			t.Errorf("unhandled route: %s (Raw: %s)", r.URL.Path, reqURI)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	c, err := NewClient(ts.URL, &http.Client{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	// 1. Healthy project
	p, err := c.GetProject(context.Background(), "healthy-proj")
	if err != nil {
		t.Fatalf("GetProject healthy: %v", err)
	}
	if p.Status != "ok" || p.IsDegraded || p.FolderMissing {
		t.Errorf("healthy project unexpectedly degraded: %+v", p)
	}

	// 2. Degraded project: visibly degraded with ResolveError
	deg, err := c.GetProject(context.Background(), "degraded-proj")
	if err != nil {
		t.Fatalf("GetProject degraded: %v", err)
	}
	if deg.Status != "degraded" || !deg.IsDegraded {
		t.Errorf("degraded project not marked degraded: %+v", deg)
	}
	if deg.ResolveError == "" {
		t.Errorf("degraded project missing ResolveError: %+v", deg)
	}

	// 3. Unknown discriminator fails closed
	_, err = c.GetProject(context.Background(), "unknown-status")
	if err == nil || !errors.Is(err, ErrProtocolViolation) {
		t.Errorf("expected ErrProtocolViolation on unknown status, got %v", err)
	}

	// 4. Missing project (404)
	_, err = c.GetProject(context.Background(), "missing-proj")
	if err == nil || !errors.Is(err, ErrProjectNotFound) {
		t.Errorf("expected ErrProjectNotFound on 404, got %v", err)
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound matching, got %v", err)
	}

	// 5. Safe path segment escaping
	esc, err := c.GetProject(context.Background(), "complex/name?with#chars")
	if err != nil {
		t.Fatalf("GetProject with special chars: %v", err)
	}
	if esc.ID != "complex/name?with#chars" {
		t.Errorf("ID = %q, want complex/name?with#chars", esc.ID)
	}
}

func TestClient_GetWorkerStatus(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", r.Method)
		}

		reqURI := r.URL.RequestURI()

		switch {
		case strings.HasPrefix(reqURI, "/api/v1/sessions/session-active"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"session": map[string]any{
					"id":        "session-active",
					"projectId": "proj-1",
					"status":    "working",
					"activity": map[string]any{
						"state":          "active",
						"lastActivityAt": now,
					},
					"harness":      "agy",
					"branch":       "feat/new-task",
					"isTerminated": false,
				},
			})

		case strings.HasPrefix(reqURI, "/api/v1/sessions/session-idle"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"session": map[string]any{
					"id":     "session-idle",
					"status": "idle",
					"activity": map[string]any{
						"state":          "idle",
						"lastActivityAt": now,
					},
					"isTerminated": false,
				},
			})

		case strings.HasPrefix(reqURI, "/api/v1/sessions/session-waiting"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"session": map[string]any{
					"id":     "session-waiting",
					"status": "needs_input",
					"activity": map[string]any{
						"state":          "waiting_input",
						"lastActivityAt": now,
					},
					"isTerminated": false,
				},
			})

		case strings.HasPrefix(reqURI, "/api/v1/sessions/session-blocked"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"session": map[string]any{
					"id":     "session-blocked",
					"status": "blocked",
					"activity": map[string]any{
						"state":          "blocked",
						"lastActivityAt": now,
					},
					"isTerminated": false,
				},
			})

		case strings.HasPrefix(reqURI, "/api/v1/sessions/session-exited"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"session": map[string]any{
					"id":     "session-exited",
					"status": "exited",
					"activity": map[string]any{
						"state":          "exited",
						"lastActivityAt": now,
					},
					"isTerminated": true,
				},
			})

		case strings.HasPrefix(reqURI, "/api/v1/sessions/session-unknown-state"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"session": map[string]any{
					"id": "session-unknown-state",
					"activity": map[string]any{
						"state":          "hyperspace_running",
						"lastActivityAt": now,
					},
				},
			})

		case strings.HasPrefix(reqURI, "/api/v1/sessions/missing-session"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(APIError{
				ErrorType: "not_found",
				Code:      "SESSION_NOT_FOUND",
				Message:   "session missing-session not found",
			})

		case strings.HasPrefix(reqURI, "/api/v1/sessions/sess%2Fspecial%3Fpath"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"session": map[string]any{
					"id": "sess/special?path",
					"activity": map[string]any{
						"state":          "active",
						"lastActivityAt": now,
					},
				},
			})

		default:
			t.Errorf("unhandled session route: %s", reqURI)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	c, err := NewClient(ts.URL, &http.Client{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	// 1. All 5 canonical activity states
	canonicalStates := []struct {
		id        string
		wantState ActivityState
		wantTerm  bool
	}{
		{"session-active", ActivityStateActive, false},
		{"session-idle", ActivityStateIdle, false},
		{"session-waiting", ActivityStateWaitingInput, false},
		{"session-blocked", ActivityStateBlocked, false},
		{"session-exited", ActivityStateExited, true},
	}

	for _, cs := range canonicalStates {
		status, err := c.GetWorkerStatus(context.Background(), cs.id)
		if err != nil {
			t.Fatalf("GetWorkerStatus(%q): %v", cs.id, err)
		}
		if status.Activity.State != cs.wantState {
			t.Errorf("status %q state = %q, want %q", cs.id, status.Activity.State, cs.wantState)
		}
		if status.IsTerminated != cs.wantTerm {
			t.Errorf("status %q IsTerminated = %v, want %v", cs.id, status.IsTerminated, cs.wantTerm)
		}
	}

	// 2. Unknown activity state fails closed
	_, err = c.GetWorkerStatus(context.Background(), "session-unknown-state")
	if err == nil || !errors.Is(err, ErrProtocolViolation) {
		t.Errorf("expected ErrProtocolViolation on unknown activity state, got %v", err)
	}

	// 3. Missing session (404)
	_, err = c.GetWorkerStatus(context.Background(), "missing-session")
	if err == nil || !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("expected ErrSessionNotFound on 404, got %v", err)
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound matching, got %v", err)
	}

	// 4. Safe URL escaping
	esc, err := c.GetWorkerStatus(context.Background(), "sess/special?path")
	if err != nil {
		t.Fatalf("GetWorkerStatus escaped: %v", err)
	}
	if esc.ID != "sess/special?path" {
		t.Errorf("ID = %q, want sess/special?path", esc.ID)
	}

	// 5. Empty sessionID
	_, err = c.GetWorkerStatus(context.Background(), "   ")
	if err == nil || !errors.Is(err, ErrBadRequest) {
		t.Errorf("expected ErrBadRequest on empty sessionID, got %v", err)
	}
}

func TestClient_ErrorHandlingAndSanitization(t *testing.T) {
	// 1. Structured AO API errors (400, 404, 500, 503)
	tests := []struct {
		status    int
		errorType string
		code      string
		msg       string
		reqID     string
		wantErr   error
	}{
		{400, "bad_request", "INVALID_PARAMS", "param missing", "req-1", ErrBadRequest},
		{404, "not_found", "ROUTE_NOT_FOUND", "route not found", "req-2", ErrNotFound},
		{503, "unavailable", "SERVICE_UNAVAILABLE", "daemon busy", "req-3", ErrDaemonUnavailable},
		{500, "internal", "INTERNAL_ERROR", "internal server error", "req-4", nil},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("HTTP_%d_%s", tt.status, tt.code), func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				_ = json.NewEncoder(w).Encode(APIError{
					ErrorType: tt.errorType,
					Code:      tt.code,
					Message:   tt.msg,
					RequestID: tt.reqID,
				})
			}))
			defer ts.Close()

			c, _ := NewClient(ts.URL, &http.Client{Timeout: 5 * time.Second})
			var dummy map[string]any
			err := c.get(context.Background(), "/test-err", &dummy)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}

			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Errorf("expected matching %v, got %v", tt.wantErr, err)
			}

			var apiErr *APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("expected *APIError, got %T: %v", err, err)
			}
			if apiErr.StatusCode != tt.status || apiErr.Code != tt.code || apiErr.RequestID != tt.reqID {
				t.Errorf("apiErr mismatch: %+v", apiErr)
			}
		})
	}

	// 2. Malformed JSON or HTML error body does NOT leak into error string
	sensitiveLeakSecret := "TOP_SECRET_INTERNAL_DATABASE_PASSWORD_XYZ"
	tsHTML := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("<html><body>Fatal crash: " + sensitiveLeakSecret + "</body></html>"))
	}))
	defer tsHTML.Close()

	cHTML, _ := NewClient(tsHTML.URL, &http.Client{Timeout: 5 * time.Second})
	var dummy map[string]any
	err := cHTML.get(context.Background(), "/html-error", &dummy)
	if err == nil {
		t.Fatalf("expected error on HTML 500, got nil")
	}
	if !errors.Is(err, ErrProtocolViolation) {
		t.Errorf("expected ErrProtocolViolation on non-conforming error envelope, got %v", err)
	}
	if strings.Contains(err.Error(), sensitiveLeakSecret) {
		t.Fatalf("SECURITY VIOLATION: raw upstream HTML error was leaked in error message: %s", err.Error())
	}

	// 3. Transport error cause preservation (context.Canceled & context.DeadlineExceeded)
	// Deadline exceeded
	ctxTimeout, cancelTimeout := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancelTimeout()

	tsSleep := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer tsSleep.Close()

	cSleep, _ := NewClient(tsSleep.URL, &http.Client{Timeout: 5 * time.Second})
	err = cSleep.get(ctxTimeout, "/sleep", &dummy)
	if err == nil {
		t.Fatalf("expected deadline exceeded error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected errors.Is(err, context.DeadlineExceeded) == true, got %v", err)
	}
	if !errors.Is(err, ErrDaemonUnavailable) {
		t.Errorf("expected errors.Is(err, ErrDaemonUnavailable) == true, got %v", err)
	}

	// Context canceled
	ctxCancel, cancelNow := context.WithCancel(context.Background())
	cancelNow()

	err = cSleep.get(ctxCancel, "/cancel", &dummy)
	if err == nil {
		t.Fatalf("expected canceled error, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected errors.Is(err, context.Canceled) == true, got %v", err)
	}
	if !errors.Is(err, ErrDaemonUnavailable) {
		t.Errorf("expected errors.Is(err, ErrDaemonUnavailable) == true, got %v", err)
	}
}
