package store

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/trungqwe/ai-supervisor/internal/domain"
)

func validTestSnapshot() domain.WorkspaceBindingSnapshot {
	return domain.WorkspaceBindingSnapshot{
		CanonicalWorktreePath:       "C:\\repo\\worktree",
		WorktreeVolumeSerialHex:     "0000000012345678",
		WorktreeFileIDHex:           "000000000000000012345678abcdef01",
		LinkedGitDirPath:            "C:\\repo\\worktree\\.git",
		LinkedGitDirVolumeSerialHex: "0000000012345678",
		LinkedGitDirFileIDHex:       "000000000000000012345678abcdef02",
		PinnedAOCommit:              strings.Repeat("a", 40),
	}
}

func setupReadyTaskAndSession(t *testing.T, s *Store, taskID, pairID, sessionID, generation string) (string, string) {
	t.Helper()
	ctx := context.Background()
	projID := "proj-" + pairID

	_ = s.CreateProject(ctx, domain.Project{ProjectID: projID, Name: projID, RootPath: "/workspace"})
	_ = s.CreatePair(ctx, domain.Pair{PairID: pairID, ProjectID: projID, CurrentPhaseID: "P04", State: "ACTIVE"})

	provOp := domain.PairProvisioningOperation{OperationID: "prov-" + pairID, PairID: pairID, ClientToken: "tok-" + pairID}
	_ = s.ReservePairProvisioning(ctx, provOp, "supervisor")
	_ = s.ConfirmPairProvisioning(ctx, provOp.OperationID, domain.WorkerSession{
		PairID:             pairID,
		SessionID:          sessionID,
		RuntimeType:        "agy",
		WorkerAgentID:      "agy",
		Status:             domain.WorkerSessionIdle,
		TerminalGeneration: generation,
		QuarantineState:    domain.QuarantineClean,
	}, "supervisor", time.Now().UTC())

	_ = s.CreateTask(ctx, domain.Task{TaskID: taskID, PhaseID: "P04", PairID: pairID, State: domain.StateDraft})
	contractID := "contract-" + taskID
	_ = s.InsertTaskContract(ctx, domain.TaskContract{
		ContractID:     contractID,
		TaskID:         taskID,
		RevisionNumber: 1,
		BaseSHA:        "35909d7b21cdfe6b9f5c309ea565c5f9f9fedeea",
		AllowedScope:   []string{"internal/**"},
	})
	_ = s.TransitionTask(ctx, taskID, domain.StateDraft, domain.StateReady)
	return contractID, projID
}

func TestPrepareBoundDispatch_AtomicSuccessAndRollback(t *testing.T) {
	s, _ := createTestStore(t)
	defer s.Close()
	ctx := context.Background()

	taskID := "task-bound-1"
	pairID := "pair-bound-1"
	sessionID := "sess-bound-1"
	generation := "gen-bound-1"
	contractID, _ := setupReadyTaskAndSession(t, s, taskID, pairID, sessionID, generation)

	attemptID := "attempt-bound-1"
	reportPath, _ := CanonicalExpectedReportPath(taskID, attemptID)
	snapshot := validTestSnapshot()

	binding := DispatchBinding{
		OperationID:        "op-bound-1",
		SessionID:          sessionID,
		TerminalGeneration: generation,
		Workspace:          snapshot,
	}

	attempt, err := s.PrepareBoundDispatch(ctx, taskID, contractID, attemptID, reportPath, time.Now().UTC(), binding)
	if err != nil {
		t.Fatalf("PrepareBoundDispatch failed: %v", err)
	}
	if attempt.AttemptID != attemptID {
		t.Fatalf("expected attemptID %s, got %s", attemptID, attempt.AttemptID)
	}

	// Verify Task state is DISPATCHED
	task, err := s.GetTask(ctx, taskID)
	if err != nil || task.State != domain.StateDispatched {
		t.Fatalf("expected task DISPATCHED, got %s, err=%v", task.State, err)
	}

	// Verify DispatchOperation is DISPATCH_BOUND
	op, err := s.GetDispatchOperation(ctx, binding.OperationID)
	if err != nil || op.Stage != domain.DispatchBound {
		t.Fatalf("expected operation DISPATCH_BOUND, got %s, err=%v", op.Stage, err)
	}

	// Verify attempt_workspace_bindings is ACTIVE
	wb, err := s.GetAttemptWorkspaceBinding(ctx, attemptID)
	if err != nil || wb.BindingState != domain.BindingStateActive {
		t.Fatalf("expected binding ACTIVE, got %v, err=%v", wb, err)
	}
	if !snapshot.MatchesBinding(wb) {
		t.Fatalf("binding row does not match snapshot: %+v vs %+v", wb, snapshot)
	}

	// AC-P04A-09: Verify audits appended
	events, err := s.ListAuditEvents(ctx, 0, 100)
	if err != nil {
		t.Fatalf("failed to list audits: %v", err)
	}
	foundDispatchBound := false
	foundBindingCreated := false
	for _, e := range events {
		if e.Event.EventType == domain.AuditTaskDispatchBound && e.Event.AttemptID == attemptID {
			foundDispatchBound = true
		}
		if e.Event.EventType == domain.AuditWorkspaceBindingCreated && e.Event.AttemptID == attemptID {
			foundBindingCreated = true
		}
	}
	if !foundDispatchBound || !foundBindingCreated {
		t.Fatalf("expected both DISPATCH_BOUND and WORKSPACE_BINDING_CREATED audits, found: bound=%t, created=%t", foundDispatchBound, foundBindingCreated)
	}
}

func TestRecordSendRequested_GuardsAndVariantBDiagnostic(t *testing.T) {
	s, _ := createTestStore(t)
	defer s.Close()
	ctx := context.Background()

	taskID := "task-presend-1"
	pairID := "pair-presend-1"
	sessionID := "sess-presend-1"
	generation := "gen-presend-1"
	contractID, _ := setupReadyTaskAndSession(t, s, taskID, pairID, sessionID, generation)

	attemptID := "attempt-presend-1"
	reportPath, _ := CanonicalExpectedReportPath(taskID, attemptID)
	snapshot := validTestSnapshot()

	opID := "op-presend-1"
	binding := DispatchBinding{
		OperationID:        opID,
		SessionID:          sessionID,
		TerminalGeneration: generation,
		Workspace:          snapshot,
	}

	_, err := s.PrepareBoundDispatch(ctx, taskID, contractID, attemptID, reportPath, time.Now().UTC(), binding)
	if err != nil {
		t.Fatalf("PrepareBoundDispatch failed: %v", err)
	}

	// 1. Snapshot identity mismatch fails with WORKSPACE_BINDING_PHYSICAL_IDENTITY_MISMATCH and triggers Variant B
	mismatchedSnapshot := snapshot
	mismatchedSnapshot.WorktreeFileIDHex = "000000000000000099999999abcdef01"

	err = s.RecordSendRequested(ctx, opID, sessionID, generation, "idle", false, "supervisor", time.Now().UTC(), mismatchedSnapshot)
	if err == nil || !strings.Contains(err.Error(), "WORKSPACE_BINDING_PHYSICAL_IDENTITY_MISMATCH") {
		t.Fatalf("expected physical identity mismatch error, got: %v", err)
	}

	// AC-P04A-11: Verify Variant B results:
	// - Task remains DISPATCHED
	task, err := s.GetTask(ctx, taskID)
	if err != nil || task.State != domain.StateDispatched {
		t.Fatalf("expected task DISPATCHED, got %s, err=%v", task.State, err)
	}
	// - Operation remains DISPATCH_BOUND
	op, err := s.GetDispatchOperation(ctx, opID)
	if err != nil || op.Stage != domain.DispatchBound {
		t.Fatalf("expected operation DISPATCH_BOUND, got %s, err=%v", op.Stage, err)
	}
	// - Attempt remains open
	attempt, err := s.GetTaskAttempt(ctx, attemptID)
	if err != nil || attempt.EndedAt != nil {
		t.Fatalf("expected attempt open, got ended_at=%v, err=%v", attempt.EndedAt, err)
	}
	// - Binding is CAS invalidated
	wb, err := s.GetAttemptWorkspaceBinding(ctx, attemptID)
	if err != nil || wb.BindingState != domain.BindingStateInvalidated {
		t.Fatalf("expected binding INVALIDATED, got %s, err=%v", wb.BindingState, err)
	}
	// - Active hold exists with INVARIANT_MISMATCH
	holds, err := s.GetActiveReviewIntegrityHolds(ctx, attemptID)
	if err != nil || len(holds) != 1 || holds[0].HoldReason != domain.HoldReasonInvariantMismatch {
		t.Fatalf("expected 1 active INVARIANT_MISMATCH hold, got: %+v, err=%v", holds, err)
	}

	// 2. Further RecordSendRequested calls are blocked by active hold (Guard 14)
	err = s.RecordSendRequested(ctx, opID, sessionID, generation, "idle", false, "supervisor", time.Now().UTC(), snapshot)
	if err == nil || !errors.Is(err, ErrReviewIntegrityHoldActive) {
		t.Fatalf("expected ErrReviewIntegrityHoldActive, got: %v", err)
	}

	// 3. Governed Terminalization & Hold Resolution
	// Resolving hold before attempt terminalization must fail
	holdID := holds[0].HoldID
	err = s.ResolveReviewIntegrityHold(ctx, holdID, "", "operator-1", time.Now().UTC())
	if err == nil {
		t.Fatalf("expected ResolveReviewIntegrityHold to fail before terminalization")
	}

	// Execute D12 terminal transition DISPATCHED -> FAILED
	err = s.AtomicTerminalTransition(ctx, taskID, domain.StateDispatched, domain.StateFailed, "binding integrity failure", attemptID, "WORKSPACE_BINDING_INTEGRITY_FAILURE")
	if err != nil {
		t.Fatalf("AtomicTerminalTransition failed: %v", err)
	}

	// Now hold resolution succeeds
	err = s.ResolveReviewIntegrityHold(ctx, holdID, "", "operator-1", time.Now().UTC())
	if err != nil {
		t.Fatalf("ResolveReviewIntegrityHold failed after terminalization: %v", err)
	}

	// Active holds should now be 0
	activeHolds, err := s.GetActiveReviewIntegrityHolds(ctx, attemptID)
	if err != nil || len(activeHolds) != 0 {
		t.Fatalf("expected 0 active holds, got: %+v", activeHolds)
	}
}
