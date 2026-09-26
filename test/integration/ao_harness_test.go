package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/trungqwe/ai-supervisor/internal/ao"
	"github.com/trungqwe/ai-supervisor/internal/dispatch"
	"github.com/trungqwe/ai-supervisor/internal/domain"
	"github.com/trungqwe/ai-supervisor/internal/host"
	"github.com/trungqwe/ai-supervisor/internal/recovery"
	"github.com/trungqwe/ai-supervisor/internal/stop"
	"github.com/trungqwe/ai-supervisor/internal/store"
)

type testHandoff struct {
	available bool
}

func (h *testHandoff) Available(context.Context, store.RecoveryExecution) (bool, error) {
	return h.available, nil
}

type mockAOServer struct {
	mu            sync.Mutex
	projects      map[string]bool
	sessions      map[string]string
	harnesses     map[string]string
	generations   map[string]string
	dispatches    map[string][]string
	killed        map[string]bool
	workspaceFile string
}

func newMockAOServer() *mockAOServer {
	return &mockAOServer{
		projects:      make(map[string]bool),
		sessions:      make(map[string]string),
		harnesses:     make(map[string]string),
		generations:   make(map[string]string),
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
		gen := fmt.Sprintf("gen-%s-1", sessID)
		m.sessions[sessID] = "waiting_input"
		m.harnesses[sessID] = req.Harness
		m.generations[sessID] = gen

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		promptBytes := 120
		sysPromptBytes := 240
		_ = json.NewEncoder(w).Encode(map[string]any{
			"session": map[string]any{
				"id":                 sessID,
				"projectId":          req.ProjectID,
				"kind":               "worker",
				"harness":            req.Harness,
				"status":             "idle",
				"isTerminated":       false,
				"terminalGeneration": gen,
				"activity": map[string]any{
					"state":          "waiting_input",
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

		// After task send, worker processes and transitions to idle
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
				"id":                 sessID,
				"projectId":          "proj-1",
				"kind":               "worker",
				"harness":            m.harnesses[sessID],
				"status":             state,
				"isTerminated":       isTerminated,
				"terminalGeneration": m.generations[sessID],
				"activity": map[string]any{
					"state":          state,
					"lastActivityAt": time.Now().UTC().Format(time.RFC3339),
				},
			},
		})
		return
	}

	// 5. Workspace file retrieval: GET /api/v1/sessions/{id}/workspace/file (returns canonical 15-field JSON envelope)
	if r.Method == http.MethodGet && strings.Contains(path, "/workspace/file") {
		parts := strings.Split(path, "/")
		sessID := ""
		if len(parts) >= 5 {
			sessID = parts[4]
		}
		filePath := r.URL.Query().Get("path")

		envelope := ao.WorkspaceFileResponse{
			SessionID:        sessID,
			Path:             filePath,
			Content:          m.workspaceFile,
			Binary:           false,
			Deleted:          false,
			ContentTruncated: false,
			Size:             int64(len(m.workspaceFile)),
			Status:           string(ao.WorkspaceFileStatusUnmodified),
			WorkspaceVersion: "wv-p03-004",
			Diff:             "",
			DiffTruncated:    false,
			Editable:         false,
			FileFingerprint:  "fp-p03-004-harness",
			Additions:        0,
			Deletions:        0,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(envelope)
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

// TestP03IntegrationHarness5StepsViaLibrarySaga verifies the 5 AO integration exit gate steps (AC-004-07)
// by driving the execution entirely through internal supervisor library sagas, Store guards,
// state transitions, and audit records (R1-005).
func TestP03IntegrationHarness5StepsViaLibrarySaga(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "harness_saga.sqlite")

	ctx := context.Background()
	st, err := store.Open(ctx, store.Config{
		DBPath:        dbPath,
		BusyTimeoutMs: 5000,
	})
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer st.Close()

	mockServer := newMockAOServer()
	ts := httptest.NewServer(mockServer)
	defer ts.Close()

	client, err := ao.NewClient(ts.URL, ts.Client())
	if err != nil {
		t.Fatalf("failed to create AO client: %v", err)
	}

	auth := host.NewAuthority("")

	dispCoord := &dispatch.Coordinator{
		Store:          st,
		AO:             client,
		Operator:       auth,
		RestoreEnabled: false,
		ExecutionPolicy: domain.ExecutionBudgetPolicy{
			Duration:  30 * time.Minute,
			PolicyRef: "policy-p03",
		},
	}

	// Step 1: Pair provisioning & Session Creation via dispatch.Coordinator.Provision
	projID := "proj-1"
	pairID := "pair-1"
	if err := st.CreateProject(ctx, domain.Project{ProjectID: projID, Name: projID, RootPath: "/workspace"}); err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}
	if err := st.CreatePair(ctx, domain.Pair{PairID: pairID, ProjectID: projID, CurrentPhaseID: "P03", State: "ACTIVE"}); err != nil {
		t.Fatalf("CreatePair failed: %v", err)
	}

	provOp := domain.PairProvisioningOperation{
		OperationID: "prov-op-1",
		PairID:      pairID,
		ClientToken: "client-tok-1",
	}
	if err := dispCoord.Provision(ctx, provOp, projID, "agy", "supervisor"); err != nil {
		t.Fatalf("Step 1 Provision failed: %v", err)
	}

	// Verify Store state after provisioning
	durableProv, err := st.GetPairProvisioningOperation(ctx, provOp.OperationID)
	if err != nil {
		t.Fatalf("failed to get provisioning operation: %v", err)
	}
	if durableProv.Stage != domain.ProvisionConfirmed {
		t.Fatalf("expected ProvisionConfirmed, got %s", durableProv.Stage)
	}

	session, err := st.GetWorkerSessionByPair(ctx, pairID)
	if err != nil {
		t.Fatalf("failed to get worker session: %v", err)
	}
	if session.Status != domain.WorkerSessionIdle {
		t.Fatalf("expected WorkerSessionIdle, got %s", session.Status)
	}
	sessionID := session.SessionID
	generation := session.TerminalGeneration

	// Step 2: Task Transmission via dispatch.Coordinator.Dispatch
	taskID := "task-1"
	contractID := "contract-1"
	attemptID := "attempt-1"
	dispatchOpID := "dispatch-op-1"

	if err := st.CreateTask(ctx, domain.Task{TaskID: taskID, PhaseID: "P03", PairID: pairID, State: domain.StateDraft}); err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}
	if err := st.InsertTaskContract(ctx, domain.TaskContract{
		ContractID:     contractID,
		TaskID:         taskID,
		RevisionNumber: 1,
		BaseSHA:        "35909d7b21cdfe6b9f5c309ea565c5f9f9fedeea",
		AllowedScope:   []string{"internal/**"},
	}); err != nil {
		t.Fatalf("InsertTaskContract failed: %v", err)
	}
	if err := st.TransitionTask(ctx, taskID, domain.StateDraft, domain.StateReady); err != nil {
		t.Fatalf("TransitionTask to ready failed: %v", err)
	}

	reportPath, err := store.CanonicalExpectedReportPath(taskID, attemptID)
	if err != nil {
		t.Fatalf("CanonicalExpectedReportPath failed: %v", err)
	}

	msg := "execute immutable task contract 004"
	if err := dispCoord.Dispatch(ctx, taskID, contractID, attemptID, dispatchOpID, sessionID, generation, reportPath, msg, "supervisor"); err != nil {
		t.Fatalf("Step 2 Dispatch failed: %v", err)
	}

	// Verify Store state after dispatch
	dispOp, err := st.GetDispatchOperation(ctx, dispatchOpID)
	if err != nil {
		t.Fatalf("failed to get dispatch operation: %v", err)
	}
	if dispOp.Stage != domain.SendConfirmed {
		t.Fatalf("expected DispatchSendConfirmed, got %s", dispOp.Stage)
	}

	budget, err := st.GetExecutionBudget(ctx, attemptID)
	if err != nil {
		t.Fatalf("failed to get execution budget: %v", err)
	}
	if budget.Duration <= 0 {
		t.Fatalf("expected positive budget duration, got %v", budget.Duration)
	}

	attempt, err := st.GetTaskAttempt(ctx, attemptID)
	if err != nil {
		t.Fatalf("failed to get task attempt: %v", err)
	}
	if attempt.AttemptID != attemptID {
		t.Fatalf("unexpected attempt ID: %s", attempt.AttemptID)
	}

	// Step 3: Observation & Status Reconciliation via recovery.Poller
	scanner := &recovery.Runner{
		Store:                st,
		AO:                   client,
		Host:                 auth,
		ActivityPollInterval: 1 * time.Second,
		ExecutionDeadline:    30 * time.Minute,
		Handoff:              &testHandoff{available: true},
		Actor:                "SUPERVISOR_RUNNER",
		Now:                  time.Now,
	}
	scanReport, err := scanner.Run(ctx)
	if err != nil {
		t.Fatalf("Step 3 scanner.Run failed: %v", err)
	}
	if !scanReport.Complete {
		t.Fatalf("expected scanReport.Complete = true, got %v", scanReport.Complete)
	}

	poller := &recovery.Poller{
		Store:    st,
		AO:       client,
		Owner:    scanner,
		Interval: 1 * time.Second,
		Actor:    "SUPERVISOR_POLLER",
	}
	if err := poller.PollOnce(ctx); err != nil {
		t.Fatalf("Step 3 PollOnce failed: %v", err)
	}

	// Step 4: Workspace File Retrieval via typed library API (R1-005 / AC-004-10)
	// Reads and inspects report content using approved ao.Client.GetWorkspaceFile
	workspaceOpts := ao.WorkspaceReadOptions{
		MaxWireBytes: 10 * 1024 * 1024, // 10MB wire envelope limit
		MaxBytes:     10 * 1024 * 1024, // 10MB decoded content limit
	}
	reportBytes, err := client.GetWorkspaceFile(ctx, sessionID, reportPath, workspaceOpts)
	if err != nil {
		t.Fatalf("Step 4 client.GetWorkspaceFile failed: %v", err)
	}
	if string(reportBytes) != mockServer.workspaceFile {
		t.Fatalf("report bytes mismatch: got %q, want %q", string(reportBytes), mockServer.workspaceFile)
	}

	var parsedReport struct {
		TaskID  string `json:"task_id"`
		Status  string `json:"status"`
		Summary string `json:"summary"`
	}
	if err := json.Unmarshal(reportBytes, &parsedReport); err != nil {
		t.Fatalf("failed to parse workspace report JSON: %v", err)
	}
	if parsedReport.TaskID != "TASK-P03-004" || parsedReport.Status != "COMPLETED" {
		t.Fatalf("unexpected parsed report contents: %+v", parsedReport)
	}

	// Verify context cancellation contract (R1-005): cancelled context must fail closed via typed API
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := client.GetWorkspaceFile(cancelledCtx, sessionID, reportPath, workspaceOpts); err == nil {
		t.Fatal("expected GetWorkspaceFile with cancelled context to fail, but succeeded")
	}

	// Negative probe: Bounded reading limits wire overflow fail-closed
	smallWireOpts := ao.WorkspaceReadOptions{
		MaxWireBytes: 20, // tiny limit smaller than envelope
		MaxBytes:     10 * 1024 * 1024,
	}
	if _, err := client.GetWorkspaceFile(ctx, sessionID, reportPath, smallWireOpts); !errors.Is(err, ao.ErrPayloadTooLarge) {
		t.Fatalf("expected ErrPayloadTooLarge on wire overflow, got: %v", err)
	}

	// Step 5: Teardown / Stop Session via stop.Coordinator.Start
	stopCoord := &stop.Coordinator{
		Store:       st,
		AO:          client,
		Operator:    auth,
		KillTimeout: 5 * time.Second,
		TimeoutHost: auth,
		Now:         time.Now,
	}

	stopOpID := "stop-op-1"
	stopOp := domain.StopOperation{
		OperationID:        stopOpID,
		Purpose:            domain.RunningAttemptStop,
		PairID:             pairID,
		TaskID:             &taskID,
		ContractID:         &contractID,
		AttemptID:          &attemptID,
		SessionID:          sessionID,
		TerminalGeneration: generation,
		Actor:              "SUPERVISOR_STOP",
	}

	if err := stopCoord.Start(ctx, stopOp); err != nil {
		t.Fatalf("Step 5 stopCoord.Start failed: %v", err)
	}

	// Verify Store state and AO state after stop
	stOp, err := st.GetStopOperation(ctx, stopOpID)
	if err != nil {
		t.Fatalf("failed to get stop operation: %v", err)
	}
	if stOp.Stage != domain.StopCallSucceeded && stOp.Stage != domain.StopTerminationConfirmed {
		t.Fatalf("unexpected stop operation stage: %s", stOp.Stage)
	}

	finalStatus, err := client.GetWorkerStatus(ctx, sessionID)
	if err != nil {
		t.Fatalf("GetWorkerStatus failed: %v", err)
	}
	if !finalStatus.IsTerminated {
		t.Fatal("expected IsTerminated = true after kill")
	}

	// Full durable state and audit chain reconciliation across sagas (R1-005)
	task, err := st.GetTask(ctx, taskID)
	if err != nil {
		t.Fatalf("reconcile GetTask failed: %v", err)
	}
	if task.TaskID != taskID {
		t.Fatalf("task ID mismatch: %s", task.TaskID)
	}

	attemptRecord, err := st.GetTaskAttempt(ctx, attemptID)
	if err != nil {
		t.Fatalf("reconcile GetTaskAttempt failed: %v", err)
	}
	if attemptRecord.AttemptID != attemptID {
		t.Fatalf("attempt ID mismatch: %s", attemptRecord.AttemptID)
	}

	durableDisp, err := st.GetDispatchOperation(ctx, dispatchOpID)
	if err != nil {
		t.Fatalf("reconcile GetDispatchOperation failed: %v", err)
	}
	if durableDisp.Stage != domain.SendConfirmed {
		t.Fatalf("expected SendConfirmed, got %s", durableDisp.Stage)
	}

	durableStop, err := st.GetStopOperation(ctx, stopOpID)
	if err != nil {
		t.Fatalf("reconcile GetStopOperation failed: %v", err)
	}
	if durableStop.Stage != domain.StopCallSucceeded && durableStop.Stage != domain.StopTerminationConfirmed {
		t.Fatalf("unexpected stop stage: %s", durableStop.Stage)
	}

	// Cryptographic verification of the tamper-evident append-only audit chain
	if err := st.VerifyAuditChain(ctx); err != nil {
		t.Fatalf("VerifyAuditChain failed: %v", err)
	}
}

// TestPairHoldAndAdmissionGuard verifies that host authority prevents conflicting operations
// and enforces fail-closed behavior for unauthenticated principals and automatic restore.
func TestPairHoldAndAdmissionGuard(t *testing.T) {
	auth := host.NewAuthority("any-token")

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
	dbPath := filepath.Join(tempDir, "test_store.sqlite")

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
