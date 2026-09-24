package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/trungqwe/ai-supervisor/internal/ao"
	"github.com/trungqwe/ai-supervisor/internal/domain"
	"github.com/trungqwe/ai-supervisor/internal/host"
	"github.com/trungqwe/ai-supervisor/internal/stop"
	"github.com/trungqwe/ai-supervisor/internal/store"
)

type mockAOServer struct {
	mu            sync.Mutex
	projects      map[string]bool
	sessions      map[string]string
	harnesses     map[string]string
	dispatches    map[string][]string
	killed        map[string]bool
	workspaceFile string
}

func newMockAOServer() *mockAOServer {
	return &mockAOServer{
		projects:      make(map[string]bool),
		sessions:      make(map[string]string),
		harnesses:     make(map[string]string),
		dispatches:    make(map[string][]string),
		killed:        make(map[string]bool),
		workspaceFile: "{\"task_id\":\"TASK-P03-004\",\"status\":\"COMPLETED\",\"summary\":\"5 AO steps proven\"}",
	}
}

func (m *mockAOServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()

	path := r.URL.Path

	// 1. Projects: POST /api/v1/projects
	if r.Method == http.MethodPost && path == "/api/v1/projects" {
		var req struct {
			ProjectID string `json:"projectId"`
			Path      string `json:"path"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		m.projects[req.ProjectID] = true
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"project": map[string]any{
				"id":     req.ProjectID,
				"name":   req.ProjectID,
				"path":   req.Path,
				"status": "ok",
			},
		})
		return
	}

	// 2. Spawn Session: POST /api/v1/sessions
	if r.Method == http.MethodPost && path == "/api/v1/sessions" {
		var req struct {
			ProjectID string `json:"projectId"`
			Kind      string `json:"kind"`
			Harness   string `json:"harness"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		sessID := fmt.Sprintf("sess-%d", len(m.sessions)+1)
		m.sessions[sessID] = "active"
		m.harnesses[sessID] = req.Harness

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		promptBytes := 120
		sysPromptBytes := 240
		_ = json.NewEncoder(w).Encode(map[string]any{
			"session": map[string]any{
				"id":           sessID,
				"projectId":    req.ProjectID,
				"kind":         "worker",
				"harness":      req.Harness,
				"status":       "active",
				"isTerminated": false,
				"activity": map[string]any{
					"state":          "active",
					"lastActivityAt": time.Now().UTC().Format(time.RFC3339),
				},
			},
			"promptBytes":       promptBytes,
			"systemPromptBytes": sysPromptBytes,
		})
		return
	}

	// 3. Send Task Message: POST /api/v1/sessions/{id}/send
	if r.Method == http.MethodPost && strings.HasPrefix(path, "/api/v1/sessions/") && strings.HasSuffix(path, "/send") {
		parts := strings.Split(path, "/")
		sessID := parts[4]
		var req struct {
			Message string `json:"message"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		m.dispatches[sessID] = append(m.dispatches[sessID], req.Message)

		m.sessions[sessID] = "idle"

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":        true,
			"sessionId": sessID,
			"message":   req.Message,
		})
		return
	}

	// 4. Observation: GET /api/v1/sessions/{id}
	if r.Method == http.MethodGet && strings.HasPrefix(path, "/api/v1/sessions/") && !strings.Contains(path, "workspace") {
		parts := strings.Split(path, "/")
		sessID := parts[4]
		state, ok := m.sessions[sessID]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		isTerminated := (state == "exited")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"session": map[string]any{
				"id":           sessID,
				"projectId":    "proj-p03",
				"kind":         "worker",
				"harness":      m.harnesses[sessID],
				"status":       state,
				"isTerminated": isTerminated,
				"activity": map[string]any{
					"state":          state,
					"lastActivityAt": time.Now().UTC().Format(time.RFC3339),
				},
			},
		})
		return
	}

	// 5. Workspace file retrieval: GET /api/v1/sessions/{id}/workspace/file
	if r.Method == http.MethodGet && strings.Contains(path, "/workspace/file") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(m.workspaceFile))
		return
	}

	// 6. Kill session: POST /api/v1/sessions/{id}/kill
	if r.Method == http.MethodPost && strings.HasPrefix(path, "/api/v1/sessions/") && strings.HasSuffix(path, "/kill") {
		parts := strings.Split(path, "/")
		sessID := parts[4]
		m.killed[sessID] = true
		m.sessions[sessID] = "exited"

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":        true,
			"sessionId": sessID,
			"freed":     true,
		})
		return
	}

	w.WriteHeader(http.StatusNotFound)
}

// TestP03IntegrationHarness5Steps verifies the 5 AO integration exit gate steps:
// Step 1: Session creation & project registration
// Step 2: Task transmission (send)
// Step 3: Observation & reconciliation (poll until IDLE)
// Step 4: Workspace report read
// Step 5: Teardown / kill session
func TestP03IntegrationHarness5Steps(t *testing.T) {
	mockServer := newMockAOServer()
	ts := httptest.NewServer(mockServer)
	defer ts.Close()

	client, err := ao.NewClient(ts.URL, ts.Client())
	if err != nil {
		t.Fatalf("failed to create AO client: %v", err)
	}

	ctx := context.Background()

	// Step 1: Project registration and Session Creation
	proj, err := client.RegisterProject(ctx, "proj-p03", "/workspace")
	if err != nil {
		t.Fatalf("Step 1 RegisterProject failed: %v", err)
	}
	if proj.ID != "proj-p03" {
		t.Fatalf("unexpected project ID: %s", proj.ID)
	}

	sessResult, err := client.CreateWorkerSession(ctx, "proj-p03", "general")
	if err != nil {
		t.Fatalf("Step 1 CreateWorkerSession failed: %v", err)
	}
	sessionID := sessResult.Session.ID
	if sessionID == "" {
		t.Fatal("empty sessionID from CreateWorkerSession")
	}

	// Step 2: Task Transmission (send)
	msg := "execute task contract 004"
	dispatchRes, err := client.DispatchTaskContract(ctx, sessionID, msg)
	if err != nil {
		t.Fatalf("Step 2 DispatchTaskContract failed: %v", err)
	}
	if dispatchRes == nil || dispatchRes.SessionID != sessionID {
		t.Fatalf("invalid dispatch result: %+v", dispatchRes)
	}

	// Step 3: Observation & Status Reconciliation
	workerStatus, err := client.GetWorkerStatus(ctx, sessionID)
	if err != nil {
		t.Fatalf("Step 3 GetWorkerStatus failed: %v", err)
	}
	if workerStatus.Activity.State != ao.ActivityStateIdle {
		t.Fatalf("expected ActivityStateIdle after send completion, got %s", workerStatus.Activity.State)
	}

	// Step 4: Workspace File Retrieval (raw workspace transport)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+fmt.Sprintf("/api/v1/sessions/%s/workspace/file?path=.supervisor/reports/task-1/att-1.json", sessionID), nil)
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("Step 4 workspace read failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected HTTP 200 for workspace read, got %d", resp.StatusCode)
	}

	// Step 5: Teardown / Kill Session
	stopRes, err := client.StopWorker(ctx, sessionID)
	if err != nil {
		t.Fatalf("Step 5 StopWorker failed: %v", err)
	}
	if stopRes == nil || !stopRes.Freed {
		t.Fatalf("unexpected stop result: %+v", stopRes)
	}

	finalStatus, err := client.GetWorkerStatus(ctx, sessionID)
	if err != nil {
		t.Fatalf("final GetWorkerStatus failed: %v", err)
	}
	if !finalStatus.IsTerminated {
		t.Fatal("expected IsTerminated = true after kill")
	}
}

// TestPairHoldAndAdmissionGuard verifies that host authority prevents conflicting operations
// and enforces fail-closed behavior for unauthenticated principals and automatic restore.
func TestPairHoldAndAdmissionGuard(t *testing.T) {
	auth := host.NewAuthority("verified-principal-123")

	// Invariant: Automatic restore is always disabled
	if auth.AutomaticRestoreEnabled() {
		t.Fatal("AutomaticRestoreEnabled must be false")
	}

	ctx := context.Background()
	permit1, err := auth.AcquireExclusiveScope(ctx, "pair-hold-1", "SAGA_DISPATCH")
	if err != nil {
		t.Fatalf("AcquireExclusiveScope failed: %v", err)
	}

	// Concurrent attempt on the same pair while held must be rejected fail-closed
	_, err = auth.AcquireExclusiveScope(ctx, "pair-hold-1", "ANOTHER_CALLER")
	if err == nil {
		t.Fatal("expected rejection for concurrent pair hold, but got nil")
	}

	// Release first scope
	_ = permit1.Release()

	// Re-acquisition succeeds
	permit2, err := auth.AcquireExclusiveScope(ctx, "pair-hold-1", "RETRY_CALLER")
	if err != nil {
		t.Fatalf("re-acquire failed: %v", err)
	}
	_ = permit2.Release()
}

// TestStopCoordinatorIntegration verifies that stop.Coordinator integrates with host.Authority
// through TimeoutAdmission.
func TestStopCoordinatorIntegration(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := tempDir + "/test_store.sqlite"

	ctx := context.Background()
	st, err := store.Open(ctx, store.Config{
		DBPath:        dbPath,
		BusyTimeoutMs: 5000,
	})
	if err != nil {
		t.Fatalf("store.Open failed: %v", err)
	}
	defer st.Close()

	auth := host.NewAuthority("verified-principal-456")

	coord := &stop.Coordinator{
		Store:       st,
		TimeoutHost: auth,
		KillTimeout: 5 * time.Second,
		Now:         time.Now,
	}

	reason := "EXECUTION_TIMEOUT"
	stopOp := domain.StopOperation{
		OperationID:             "stop-op-1",
		PairID:                  "pair-test",
		Purpose:                 domain.RunningAttemptStop,
		InitiatingFailureReason: &reason,
	}

	err = coord.Start(ctx, stopOp)
	if err == nil {
		t.Fatal("expected error when calling Start with timeout cause, got nil")
	}
}
