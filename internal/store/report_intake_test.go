package store

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/trungqwe/ai-supervisor/internal/domain"
)

func setupRunningAttemptWithBinding(t *testing.T, s *Store, taskID, contractID, attemptID string) (string, DispatchBinding) {
	t.Helper()
	ctx := context.Background()

	pairID := "pair-" + taskID
	sessionID := "sess-" + taskID
	generation := "gen-" + taskID
	_, _ = setupReadyTaskAndSession(t, s, taskID, pairID, sessionID, generation)

	reportPath, _ := CanonicalExpectedReportPath(taskID, attemptID)
	snapshot := validTestSnapshot()
	binding := DispatchBinding{
		OperationID:        "op-" + attemptID,
		SessionID:          sessionID,
		TerminalGeneration: generation,
		Workspace:          snapshot,
	}

	_, err := s.PrepareBoundDispatch(ctx, taskID, contractID, attemptID, reportPath, time.Now().UTC(), binding)
	if err != nil {
		t.Fatalf("PrepareBoundDispatch failed: %v", err)
	}

	err = s.RecordSendRequested(ctx, binding.OperationID, sessionID, generation, "idle", false, "supervisor", time.Now().UTC(), snapshot)
	if err != nil {
		t.Fatalf("RecordSendRequested failed: %v", err)
	}

	err = s.RecordSendConfirmed(ctx, binding.OperationID, "supervisor", true, time.Now().UTC(), domain.ExecutionBudgetPolicy{
		Duration:  time.Hour,
		PolicyRef: "fixture-policy",
	})
	if err != nil {
		t.Fatalf("RecordSendConfirmed failed: %v", err)
	}

	// Advance task to RUNNING
	err = s.TransitionTask(ctx, taskID, domain.StateDispatched, domain.StateRunning)
	if err != nil {
		t.Fatalf("TransitionTask to RUNNING failed: %v", err)
	}

	return pairID, binding
}

func TestReportIntake_CleanSuccessTransactionA(t *testing.T) {
	s, _ := createTestStore(t)
	defer s.Close()
	ctx := context.Background()

	taskID := "task-clean-1"
	contractID := "contract-" + taskID
	attemptID := "attempt-clean-1"
	_, _ = setupRunningAttemptWithBinding(t, s, taskID, contractID, attemptID)

	headSHA := "1234567890abcdef"
	report := domain.WorkerReport{
		HeadSHA:      headSHA,
		FilesChanged: []string{"internal/store/migrations.go"},
		Tests: []domain.ClaimedTestResult{
			{Name: "TestA", Passed: true, ExitCode: 0},
		},
		WorkerClaims: []string{"Implemented feature X"},
		BuildStatus:  "SUCCESS",
	}

	gitEvidence := domain.GitEvidenceResult{
		IsClean:       true,
		ActualHeadSHA: "1234567890abcdef",
		ActualBaseSHA: "35909d7b21cdfe6b9f5c309ea565c5f9f9fedeea",
		ChangedFiles:  []string{"internal/store/migrations.go"},
	}

	err := s.IngestWorkerReport(ctx, taskID, contractID, attemptID, report, gitEvidence, "supervisor", time.Now().UTC())
	if err != nil {
		t.Fatalf("IngestWorkerReport failed: %v", err)
	}

	// AC-P04A-14: Task state advanced to REPORT_READY
	task, err := s.GetTask(ctx, taskID)
	if err != nil || task.State != domain.StateReportReady {
		t.Fatalf("expected task REPORT_READY, got %s, err=%v", task.State, err)
	}

	// Binding advanced to RETAINED_FOR_VERIFICATION
	wb, err := s.GetAttemptWorkspaceBinding(ctx, attemptID)
	if err != nil || wb.BindingState != domain.BindingStateRetainedForVerification {
		t.Fatalf("expected binding RETAINED_FOR_VERIFICATION, got %s, err=%v", wb.BindingState, err)
	}

	// worker_claims persisted with verbatim head_sha
	claim, err := s.GetWorkerClaim(ctx, attemptID)
	if err != nil {
		t.Fatalf("GetWorkerClaim failed: %v", err)
	}
	if claim.ReportedHeadSHA != headSHA {
		t.Fatalf("expected reported_head_sha %q, got %q", headSHA, claim.ReportedHeadSHA)
	}
	if !strings.Contains(claim.PayloadJSON, "claimed_files_changed") || !strings.Contains(claim.PayloadJSON, "tests") {
		t.Fatalf("invalid payload_json: %s", claim.PayloadJSON)
	}
}

func TestReportIntake_DirtyWorktreeDiagnostics(t *testing.T) {
	s, _ := createTestStore(t)
	defer s.Close()
	ctx := context.Background()

	taskID := "task-dirty-1"
	contractID := "contract-" + taskID
	attemptID := "attempt-dirty-1"
	_, _ = setupRunningAttemptWithBinding(t, s, taskID, contractID, attemptID)

	headSHA := "1234567890abcdef"
	report := domain.WorkerReport{
		HeadSHA:      headSHA,
		FilesChanged: []string{"internal/foo.go"},
	}

	dirtyEvidence := domain.GitEvidenceResult{
		IsClean:          false,
		UncommittedFiles: []string{"internal/foo.go (modified)"},
	}

	// AC-P04A-15: Dirty intake triggers intake diagnostic transaction
	err := s.IngestWorkerReport(ctx, taskID, contractID, attemptID, report, dirtyEvidence, "supervisor", time.Now().UTC())
	if !errors.Is(err, ErrDirtyWorktreeDetected) {
		t.Fatalf("expected ErrDirtyWorktreeDetected, got: %v", err)
	}

	// Task state remains RUNNING
	task, err := s.GetTask(ctx, taskID)
	if err != nil || task.State != domain.StateRunning {
		t.Fatalf("expected task RUNNING, got %s, err=%v", task.State, err)
	}

	// Binding remains ACTIVE
	wb, err := s.GetAttemptWorkspaceBinding(ctx, attemptID)
	if err != nil || wb.BindingState != domain.BindingStateActive {
		t.Fatalf("expected binding ACTIVE, got %s, err=%v", wb.BindingState, err)
	}

	// Active hold exists with DIRTY_WORKTREE_DETECTED
	holds, err := s.GetActiveReviewIntegrityHolds(ctx, attemptID)
	if err != nil || len(holds) != 1 || holds[0].HoldReason != domain.HoldReasonDirtyWorktreeDetected {
		t.Fatalf("expected 1 active DIRTY_WORKTREE_DETECTED hold, got: %+v, err=%v", holds, err)
	}

	// Exact replay returns ErrDirtyWorktreeDetected without inserting duplicate hold
	err = s.IngestWorkerReport(ctx, taskID, contractID, attemptID, report, dirtyEvidence, "supervisor", time.Now().UTC())
	if !errors.Is(err, ErrDirtyWorktreeDetected) {
		t.Fatalf("expected replay ErrDirtyWorktreeDetected, got: %v", err)
	}
	holdsAfterReplay, err := s.GetActiveReviewIntegrityHolds(ctx, attemptID)
	if err != nil || len(holdsAfterReplay) != 1 {
		t.Fatalf("replay created duplicate hold: count=%d", len(holdsAfterReplay))
	}
}

func TestReportIntake_ClaimEvidenceZeroTrustSeparation(t *testing.T) {
	s, _ := createTestStore(t)
	defer s.Close()
	ctx := context.Background()

	taskID := "task-zt-1"
	contractID := "contract-" + taskID
	attemptID := "attempt-zt-1"
	_, _ = setupRunningAttemptWithBinding(t, s, taskID, contractID, attemptID)

	reportedSHA := "aaaa1111"
	actualSHA := "bbbb2222"

	report := domain.WorkerReport{
		HeadSHA:      reportedSHA,
		FilesChanged: []string{"test.go"},
	}

	gitEvidence := domain.GitEvidenceResult{
		IsClean:       true,
		ActualHeadSHA: actualSHA,
		ActualBaseSHA: "35909d7b21cdfe6b9f5c309ea565c5f9f9fedeea",
		ChangedFiles:  []string{"test.go"},
	}

	// AC-P04A-18: Zero-trust separation
	err := s.IngestWorkerReport(ctx, taskID, contractID, attemptID, report, gitEvidence, "supervisor", time.Now().UTC())
	if err != nil {
		t.Fatalf("IngestWorkerReport failed: %v", err)
	}

	claim, err := s.GetWorkerClaim(ctx, attemptID)
	if err != nil {
		t.Fatalf("GetWorkerClaim failed: %v", err)
	}
	if claim.ReportedHeadSHA != reportedSHA {
		t.Fatalf("expected reported_head_sha %q, got %q", reportedSHA, claim.ReportedHeadSHA)
	}
	if gitEvidence.ActualHeadSHA != actualSHA {
		t.Fatalf("expected actual_head_sha %q, got %q", actualSHA, gitEvidence.ActualHeadSHA)
	}
	if claim.ReportedHeadSHA == gitEvidence.ActualHeadSHA {
		t.Fatalf("reported and actual SHA unexpectedly collided")
	}
}
