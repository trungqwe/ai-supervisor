package ao

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewClient_LoopbackValidation(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr error
	}{
		{
			name:    "valid IPv4 loopback",
			url:     "http://127.0.0.1:3001",
			wantErr: nil,
		},
		{
			name:    "valid IPv4 loopback secondary IP",
			url:     "http://127.0.0.2:8080",
			wantErr: nil,
		},
		{
			name:    "valid localhost hostname",
			url:     "http://localhost:3001",
			wantErr: nil,
		},
		{
			name:    "valid localhost with path",
			url:     "http://localhost:3001/base/",
			wantErr: nil,
		},
		{
			name:    "valid IPv6 loopback",
			url:     "http://[::1]:3001",
			wantErr: nil,
		},
		{
			name:    "valid HTTPS loopback",
			url:     "https://127.0.0.1:8443",
			wantErr: nil,
		},
		{
			name:    "empty URL rejected",
			url:     "",
			wantErr: ErrBadRequest,
		},
		{
			name:    "whitespace URL rejected",
			url:     "   ",
			wantErr: ErrBadRequest,
		},
		{
			name:    "remote IP rejected",
			url:     "http://192.168.1.100:3001",
			wantErr: ErrNonLoopbackURL,
		},
		{
			name:    "remote hostname rejected",
			url:     "http://example.com:3001",
			wantErr: ErrNonLoopbackURL,
		},
		{
			name:    "invalid scheme rejected",
			url:     "ftp://127.0.0.1:21",
			wantErr: ErrNonLoopbackURL,
		},
		{
			name:    "wildcard 0.0.0.0 rejected",
			url:     "http://0.0.0.0:3001",
			wantErr: ErrNonLoopbackURL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.url)
			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("NewClient(%q) expected error %v, got nil", tt.url, tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("NewClient(%q) error = %v, want %v", tt.url, err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Fatalf("NewClient(%q) unexpected error: %v", tt.url, err)
				}
				if client == nil {
					t.Fatalf("NewClient(%q) returned nil client", tt.url)
				}
				if !strings.HasPrefix(client.BaseURL(), "http") {
					t.Fatalf("BaseURL() = %q, expected http prefix", client.BaseURL())
				}
			}
		})
	}
}

func TestClient_CheckHealth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/healthz" {
			t.Errorf("expected path /healthz, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":  "ok",
			"service": "agent-orchestrator-daemon",
			"pid":     12345,
		})
	}))
	defer server.Close()

	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	health, err := client.CheckHealth(context.Background())
	if err != nil {
		t.Fatalf("CheckHealth: %v", err)
	}
	if health.Status != "ok" {
		t.Errorf("health.Status = %q, want ok", health.Status)
	}
	if health.Service != "agent-orchestrator-daemon" {
		t.Errorf("health.Service = %q, want agent-orchestrator-daemon", health.Service)
	}
	if health.PID != 12345 {
		t.Errorf("health.PID = %d, want 12345", health.PID)
	}
}

func TestClient_CheckReadiness(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/readyz" {
			t.Errorf("expected path /readyz, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":  "ready",
			"service": "agent-orchestrator-daemon",
			"pid":     12345,
		})
	}))
	defer server.Close()

	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	ready, err := client.CheckReadiness(context.Background())
	if err != nil {
		t.Fatalf("CheckReadiness: %v", err)
	}
	if ready.Status != "ready" {
		t.Errorf("ready.Status = %q, want ready", ready.Status)
	}
	if ready.PID != 12345 {
		t.Errorf("ready.PID = %d, want 12345", ready.PID)
	}
}

func TestClient_ListAgents(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/agents" {
			t.Errorf("expected path /api/v1/agents, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
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
	defer server.Close()

	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	inv, err := client.ListAgents(context.Background())
	if err != nil {
		t.Fatalf("ListAgents: %v", err)
	}
	if len(inv.Supported) != 2 {
		t.Errorf("Supported count = %d, want 2", len(inv.Supported))
	}
	if len(inv.Installed) != 1 || inv.Installed[0].ID != "agy" {
		t.Errorf("Installed = %#v, want [agy]", inv.Installed)
	}
}

func TestClient_GetAgentReadiness(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/agents/readiness" {
			t.Errorf("expected path /api/v1/agents/readiness, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(AgentReadinessResponse{
			Agents: []AgentReadinessSnapshot{
				{ID: "agy", Label: "Antigravity", EffectiveReadiness: "ready", UsageCount: 5},
			},
		})
	}))
	defer server.Close()

	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	resp, err := client.GetAgentReadiness(context.Background())
	if err != nil {
		t.Fatalf("GetAgentReadiness: %v", err)
	}
	if len(resp.Agents) != 1 {
		t.Fatalf("len(resp.Agents) = %d, want 1", len(resp.Agents))
	}
	if resp.Agents[0].EffectiveReadiness != "ready" {
		t.Errorf("EffectiveReadiness = %q, want ready", resp.Agents[0].EffectiveReadiness)
	}
}

func TestClient_GetAPIContract(t *testing.T) {
	mockYAML := `openapi: "3.0.3"
info:
  title: "Agent Orchestrator Daemon"
  version: "v0.13.0"
`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/openapi.yaml" {
			t.Errorf("expected path /api/v1/openapi.yaml, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockYAML))
	}))
	defer server.Close()

	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	contract, err := client.GetAPIContract(context.Background())
	if err != nil {
		t.Fatalf("GetAPIContract: %v", err)
	}
	if !strings.Contains(contract, `openapi: "3.0.3"`) {
		t.Errorf("unexpected contract content: %s", contract)
	}
}

func TestClient_RegisterProject(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/projects" {
			t.Errorf("expected path /api/v1/projects, got %s", r.URL.Path)
		}

		var in RegisterProjectRequest
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			t.Errorf("decode request body: %v", err)
		}
		if in.Path != "D:/TU_CODE/project-x" {
			t.Errorf("Path = %q, want D:/TU_CODE/project-x", in.Path)
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(ProjectResponse{
			Project: Project{
				ID:            "project-x",
				Name:          "Project X",
				Kind:          "single_repo",
				Path:          in.Path,
				Repo:          "project-x",
				DefaultBranch: "main",
			},
		})
	}))
	defer server.Close()

	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	p, err := client.RegisterProject(context.Background(), RegisterProjectRequest{
		Path: "D:/TU_CODE/project-x",
	})
	if err != nil {
		t.Fatalf("RegisterProject: %v", err)
	}
	if p.ID != "project-x" {
		t.Errorf("p.ID = %q, want project-x", p.ID)
	}
	if p.Path != "D:/TU_CODE/project-x" {
		t.Errorf("p.Path = %q, want D:/TU_CODE/project-x", p.Path)
	}

	// Test empty path rejection
	_, err = client.RegisterProject(context.Background(), RegisterProjectRequest{Path: "   "})
	if err == nil || !errors.Is(err, ErrBadRequest) {
		t.Fatalf("expected ErrBadRequest on empty path, got %v", err)
	}
}

func TestClient_GetProject(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path == "/api/v1/projects/project-x" {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(GetProjectResponse{
				Status: "ok",
				Project: Project{
					ID:            "project-x",
					Name:          "Project X",
					Path:          "D:/TU_CODE/project-x",
					DefaultBranch: "main",
				},
			})
			return
		}
		if r.URL.Path == "/api/v1/projects/missing" {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(APIError{
				ErrorType: "not_found",
				Code:      "PROJECT_NOT_FOUND",
				Message:   "project missing does not exist",
			})
			return
		}
		t.Errorf("unexpected path: %s", r.URL.Path)
	}))
	defer server.Close()

	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	// Success case
	p, err := client.GetProject(context.Background(), "project-x")
	if err != nil {
		t.Fatalf("GetProject: %v", err)
	}
	if p.ID != "project-x" {
		t.Errorf("p.ID = %q, want project-x", p.ID)
	}

	// Not found case
	_, err = client.GetProject(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error on missing project, got nil")
	}
	if !errors.Is(err, ErrProjectNotFound) {
		t.Errorf("err = %v, want ErrProjectNotFound", err)
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}

	// Empty ID case
	_, err = client.GetProject(context.Background(), "  ")
	if err == nil || !errors.Is(err, ErrBadRequest) {
		t.Fatalf("expected ErrBadRequest on empty project ID, got %v", err)
	}
}

func TestClient_GetWorkerStatus(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path == "/api/v1/sessions/session-123" {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(SessionResponse{
				Session: WorkerStatus{
					ID:            "session-123",
					ProjectID:     "project-x",
					Status:        "working",
					DisplayStatus: "Working",
					IsTerminated:  false,
					Activity: ActivitySnapshot{
						State:          "active",
						LastActivityAt: now,
					},
					Harness: "agy",
					Branch:  "main",
				},
			})
			return
		}
		if r.URL.Path == "/api/v1/sessions/session-missing" {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(APIError{
				ErrorType: "not_found",
				Code:      "SESSION_NOT_FOUND",
				Message:   "session session-missing not found",
			})
			return
		}
		t.Errorf("unexpected path: %s", r.URL.Path)
	}))
	defer server.Close()

	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	// Success case
	status, err := client.GetWorkerStatus(context.Background(), "session-123")
	if err != nil {
		t.Fatalf("GetWorkerStatus: %v", err)
	}
	if status.ID != "session-123" {
		t.Errorf("status.ID = %q, want session-123", status.ID)
	}
	if status.Activity.State != "active" {
		t.Errorf("status.Activity.State = %q, want active", status.Activity.State)
	}
	if status.IsTerminated {
		t.Errorf("status.IsTerminated = true, want false")
	}

	// Not found case
	_, err = client.GetWorkerStatus(context.Background(), "session-missing")
	if err == nil {
		t.Fatal("expected error on missing session, got nil")
	}
	if !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("err = %v, want ErrSessionNotFound", err)
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}

	// Empty ID case
	_, err = client.GetWorkerStatus(context.Background(), "  ")
	if err == nil || !errors.Is(err, ErrBadRequest) {
		t.Fatalf("expected ErrBadRequest on empty session ID, got %v", err)
	}
}

func TestClient_ErrorEnvelopeDecoding(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    error
		checkCode  string
	}{
		{
			name:       "400 bad request structured",
			statusCode: http.StatusBadRequest,
			body:       `{"error":"bad_request","code":"INVALID_JSON","message":"malformed json input"}`,
			wantErr:    ErrBadRequest,
			checkCode:  "INVALID_JSON",
		},
		{
			name:       "404 not found structured",
			statusCode: http.StatusNotFound,
			body:       `{"error":"not_found","code":"ROUTE_NOT_FOUND","message":"handler not found"}`,
			wantErr:    ErrNotFound,
			checkCode:  "ROUTE_NOT_FOUND",
		},
		{
			name:       "503 service unavailable",
			statusCode: http.StatusServiceUnavailable,
			body:       `{"error":"unavailable","code":"SERVICE_UNAVAILABLE","message":"server busy"}`,
			wantErr:    ErrDaemonUnavailable,
			checkCode:  "SERVICE_UNAVAILABLE",
		},
		{
			name:       "500 internal error with unstructured plain text",
			statusCode: http.StatusInternalServerError,
			body:       "Internal Server Error: memory fault",
			wantErr:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			client, err := NewClient(server.URL)
			if err != nil {
				t.Fatalf("NewClient: %v", err)
			}

			err = client.get(context.Background(), "/test-endpoint", nil)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Errorf("errors.Is(err, %v) = false, got %v", tt.wantErr, err)
			}

			var apiErr *APIError
			if errors.As(err, &apiErr) {
				if apiErr.StatusCode != tt.statusCode {
					t.Errorf("apiErr.StatusCode = %d, want %d", apiErr.StatusCode, tt.statusCode)
				}
				if tt.checkCode != "" && apiErr.Code != tt.checkCode {
					t.Errorf("apiErr.Code = %q, want %q", apiErr.Code, tt.checkCode)
				}
			} else if tt.checkCode != "" {
				t.Fatalf("expected *APIError, got %T: %v", err, err)
			}
		})
	}
}

func TestClient_ContextTimeoutAndCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, WithTimeout(50*time.Millisecond))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	// Timeout should trigger ErrDaemonUnavailable wrap
	_, err = client.CheckHealth(context.Background())
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !errors.Is(err, ErrDaemonUnavailable) {
		t.Errorf("expected ErrDaemonUnavailable, got %v", err)
	}

	// Context cancellation
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately
	_, err = client.CheckHealth(ctx)
	if err == nil {
		t.Fatal("expected context canceled error, got nil")
	}
}
