package store

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
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

func validTestReport(taskID, contractID, attemptID, baseSHA, headSHA string) domain.WorkerReport {
	return domain.WorkerReport{
		TaskID:         taskID,
		AttemptID:      attemptID,
		Status:         "COMPLETED",
		Branch:         "codex/task-branch",
		BaseSHA:        baseSHA,
		HeadSHA:        headSHA,
		FilesChanged:   []string{"internal/store/migrations.go"},
		CommandsRun:    []domain.CommandRun{{Command: "go test ./...", ExitCode: 0}},
		Tests:          []domain.WorkerReportTest{{TestSuite: "unit", Passed: 1, Failed: 0}},
		BuildStatus:    "PASSED",
		WorkerClaims:   []string{"Implemented feature X"},
		ReadyForReview: true,
	}
}

func validTestReportJSON(t *testing.T, r domain.WorkerReport) []byte {
	t.Helper()
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("failed to marshal worker report: %v", err)
	}
	return data
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
	baseSHA := "35909d7b21cdfe6b9f5c309ea565c5f9f9fedeea"
	report := validTestReport(taskID, contractID, attemptID, baseSHA, headSHA)

	gitEvidence := domain.GitEvidenceResult{
		IsClean:       true,
		ActualHeadSHA: "1234567890abcdef",
		ActualBaseSHA: baseSHA,
		ChangedFiles:  []string{"internal/store/migrations.go"},
	}

	err := s.IngestWorkerReportRaw(ctx, taskID, contractID, attemptID, validTestReportJSON(t, report), gitEvidence, "supervisor", time.Now().UTC())
	if err != nil {
		t.Fatalf("IngestWorkerReportRaw failed: %v", err)
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
	baseSHA := "35909d7b21cdfe6b9f5c309ea565c5f9f9fedeea"
	report := validTestReport(taskID, contractID, attemptID, baseSHA, headSHA)

	dirtyEvidence := domain.GitEvidenceResult{
		IsClean:          false,
		UncommittedFiles: []string{"internal/foo.go (modified)"},
	}

	// AC-P04A-15: Dirty intake triggers intake diagnostic transaction
	err := s.IngestWorkerReportRaw(ctx, taskID, contractID, attemptID, validTestReportJSON(t, report), dirtyEvidence, "supervisor", time.Now().UTC())
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
	err = s.IngestWorkerReportRaw(ctx, taskID, contractID, attemptID, validTestReportJSON(t, report), dirtyEvidence, "supervisor", time.Now().UTC())
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
	baseSHA := "35909d7b21cdfe6b9f5c309ea565c5f9f9fedeea"

	report := validTestReport(taskID, contractID, attemptID, baseSHA, reportedSHA)

	gitEvidence := domain.GitEvidenceResult{
		IsClean:       true,
		ActualHeadSHA: actualSHA,
		ActualBaseSHA: baseSHA,
		ChangedFiles:  []string{"test.go"},
	}

	// AC-P04A-18: Zero-trust separation
	err := s.IngestWorkerReportRaw(ctx, taskID, contractID, attemptID, validTestReportJSON(t, report), gitEvidence, "supervisor", time.Now().UTC())
	if err != nil {
		t.Fatalf("IngestWorkerReportRaw failed: %v", err)
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

func TestReportIntake_NegativeSchemaAndLineage(t *testing.T) {
	s, _ := createTestStore(t)
	defer s.Close()
	ctx := context.Background()

	taskID := "task-neg-1"
	contractID := "contract-" + taskID
	attemptID := "attempt-neg-1"
	_, _ = setupRunningAttemptWithBinding(t, s, taskID, contractID, attemptID)

	baseSHA := "35909d7b21cdfe6b9f5c309ea565c5f9f9fedeea"
	headSHA := "1234567890abcdef"

	cleanEvidence := domain.GitEvidenceResult{IsClean: true}

	// 1. Lineage mismatch: TaskID
	badTaskReport := validTestReport("wrong-task", contractID, attemptID, baseSHA, headSHA)
	if err := s.IngestWorkerReportRaw(ctx, taskID, contractID, attemptID, validTestReportJSON(t, badTaskReport), cleanEvidence, "supervisor", time.Now().UTC()); err == nil {
		t.Fatal("expected error on TaskID lineage mismatch")
	}

	// 2. Lineage mismatch: AttemptID
	badAttemptReport := validTestReport(taskID, contractID, "wrong-attempt", baseSHA, headSHA)
	if err := s.IngestWorkerReportRaw(ctx, taskID, contractID, attemptID, validTestReportJSON(t, badAttemptReport), cleanEvidence, "supervisor", time.Now().UTC()); err == nil {
		t.Fatal("expected error on AttemptID lineage mismatch")
	}

	// 3. Lineage mismatch: BaseSHA
	badBaseReport := validTestReport(taskID, contractID, attemptID, "wrongbase12345678", headSHA)
	if err := s.IngestWorkerReportRaw(ctx, taskID, contractID, attemptID, validTestReportJSON(t, badBaseReport), cleanEvidence, "supervisor", time.Now().UTC()); err == nil {
		t.Fatal("expected error on BaseSHA lineage mismatch")
	}

	// 4. IngestWorkerReportRaw with additional property
	extraFieldJSON := []byte(`{
		"task_id": "` + taskID + `",
		"attempt_id": "` + attemptID + `",
		"status": "COMPLETED",
		"branch": "main",
		"base_sha": "` + baseSHA + `",
		"head_sha": "` + headSHA + `",
		"files_changed": [],
		"commands_run": [],
		"tests": [],
		"build_status": "PASSED",
		"worker_claims": [],
		"ready_for_review": true,
		"disallowed_key": "fail"
	}`)
	if err := s.IngestWorkerReportRaw(ctx, taskID, contractID, attemptID, extraFieldJSON, cleanEvidence, "supervisor", time.Now().UTC()); err == nil {
		t.Fatal("expected error on extra property in raw report JSON")
	}

	// 5. IngestWorkerReportRaw with missing required property
	missingFieldJSON := []byte(`{
		"task_id": "` + taskID + `",
		"attempt_id": "` + attemptID + `",
		"status": "COMPLETED",
		"branch": "main",
		"base_sha": "` + baseSHA + `",
		"head_sha": "` + headSHA + `"
	}`)
	if err := s.IngestWorkerReportRaw(ctx, taskID, contractID, attemptID, missingFieldJSON, cleanEvidence, "supervisor", time.Now().UTC()); err == nil {
		t.Fatal("expected error on missing required fields in raw report JSON")
	}
}

func TestReportIntake_NegativeDirtyIntakeGuard(t *testing.T) {
	s, _ := createTestStore(t)
	defer s.Close()
	ctx := context.Background()

	taskID := "task-dirty-guard"
	contractID := "contract-" + taskID
	attemptID := "attempt-dirty-guard"
	pairID, _ := setupRunningAttemptWithBinding(t, s, taskID, contractID, attemptID)

	baseSHA := "35909d7b21cdfe6b9f5c309ea565c5f9f9fedeea"
	headSHA := "1234567890abcdef"
	report := validTestReport(taskID, contractID, attemptID, baseSHA, headSHA)
	dirtyEvidence := domain.GitEvidenceResult{
		IsClean:          false,
		UncommittedFiles: []string{"file.go"},
	}

	// Case 1: Task not in RUNNING (simulate task moved to REPORT_READY or FAILED)
	_ = s.TransitionTask(ctx, taskID, domain.StateRunning, domain.StateReportReady)

	err := s.IngestWorkerReportRaw(ctx, taskID, contractID, attemptID, validTestReportJSON(t, report), dirtyEvidence, "supervisor", time.Now().UTC())
	if err == nil {
		t.Fatal("expected error for dirty intake when task is not RUNNING")
	}

	// Verify ZERO holds and ZERO audit events created
	holds, _ := s.GetActiveReviewIntegrityHolds(ctx, attemptID)
	if len(holds) != 0 {
		t.Fatalf("expected 0 holds on rejected dirty intake, got %d", len(holds))
	}

	// Case 2: Attempt ended
	taskID2 := "task-dirty-ended"
	contractID2 := "contract-" + taskID2
	attemptID2 := "attempt-dirty-ended"
	_, _ = setupRunningAttemptWithBinding(t, s, taskID2, contractID2, attemptID2)
	now := time.Now().UTC()
	_, _ = s.db.ExecContext(ctx, "UPDATE task_attempts SET ended_at = ? WHERE attempt_id = ?", now.Format(time.RFC3339Nano), attemptID2)

	report2 := validTestReport(taskID2, contractID2, attemptID2, baseSHA, headSHA)
	err = s.IngestWorkerReportRaw(ctx, taskID2, contractID2, attemptID2, validTestReportJSON(t, report2), dirtyEvidence, "supervisor", now)
	if err == nil {
		t.Fatal("expected error for dirty intake when attempt is ended")
	}
	holds2, _ := s.GetActiveReviewIntegrityHolds(ctx, attemptID2)
	if len(holds2) != 0 {
		t.Fatalf("expected 0 holds on ended attempt, got %d", len(holds2))
	}

	// Case 3: Contract lineage mismatch
	taskID3 := "task-dirty-lineage"
	contractID3 := "contract-" + taskID3
	attemptID3 := "attempt-dirty-lineage"
	_, _ = setupRunningAttemptWithBinding(t, s, taskID3, contractID3, attemptID3)
	report3 := validTestReport(taskID3, "wrong-contract", attemptID3, baseSHA, headSHA)

	err = s.IngestWorkerReportRaw(ctx, taskID3, "wrong-contract", attemptID3, validTestReportJSON(t, report3), dirtyEvidence, "supervisor", now)
	if err == nil {
		t.Fatal("expected error on contract lineage mismatch")
	}
	holds3, _ := s.GetActiveReviewIntegrityHolds(ctx, attemptID3)
	if len(holds3) != 0 {
		t.Fatalf("expected 0 holds on mismatched contract, got %d", len(holds3))
	}
	_ = pairID
}

func TestWorkerReportSchemaParity(t *testing.T) {
	canonicalPath := filepath.Join("..", "..", "docs", "schemas", "worker-report.schema.json")
	canonicalBytes, err := os.ReadFile(canonicalPath)
	if err != nil {
		t.Fatalf("failed to read canonical schema: %v", err)
	}

	if !bytes.Equal(workerReportSchemaBytes, canonicalBytes) {
		t.Fatalf("embedded worker_report_schema.json is not byte-identical to %s", canonicalPath)
	}

	embeddedSum := sha256.Sum256(workerReportSchemaBytes)
	canonicalSum := sha256.Sum256(canonicalBytes)
	if embeddedSum != canonicalSum {
		t.Fatalf("SHA-256 mismatch: embedded=%x, canonical=%x", embeddedSum, canonicalSum)
	}
}

func TestIngestWorkerReportRaw_NegativeValidation(t *testing.T) {
	s, _ := createTestStore(t)
	defer s.Close()
	ctx := context.Background()

	taskID := "task-neg-val"
	contractID := "contract-" + taskID
	attemptID := "attempt-neg-val"
	_, _ = setupRunningAttemptWithBinding(t, s, taskID, contractID, attemptID)

	cleanEvidence := domain.GitEvidenceResult{IsClean: true}

	countTable := func(table string) int {
		var count int
		_ = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(&count)
		return count
	}

	tasksCountBefore := countTable("tasks")
	auditCountBefore := countTable("audit_events")
	holdsCountBefore := countTable("review_integrity_holds")
	claimsCountBefore := countTable("worker_claims")

	// 1. Unknown property
	unknownPropJSON := []byte(`{
		"task_id": "task-neg-val",
		"attempt_id": "attempt-neg-val",
		"status": "COMPLETED",
		"branch": "main",
		"base_sha": "35909d7b21cdfe6b9f5c309ea565c5f9f9fedeea",
		"head_sha": "1234567890abcdef",
		"files_changed": [],
		"commands_run": [],
		"tests": [],
		"build_status": "PASSED",
		"worker_claims": [],
		"ready_for_review": true,
		"disallowed_field": "intruder"
	}`)
	err := s.IngestWorkerReportRaw(ctx, taskID, contractID, attemptID, unknownPropJSON, cleanEvidence, "supervisor", time.Now().UTC())
	if err == nil {
		t.Fatal("expected error on unknown property")
	}

	// 2. Missing required field
	missingReqJSON := []byte(`{
		"task_id": "task-neg-val",
		"attempt_id": "attempt-neg-val",
		"status": "COMPLETED",
		"branch": "main",
		"base_sha": "35909d7b21cdfe6b9f5c309ea565c5f9f9fedeea",
		"head_sha": "1234567890abcdef"
	}`)
	err = s.IngestWorkerReportRaw(ctx, taskID, contractID, attemptID, missingReqJSON, cleanEvidence, "supervisor", time.Now().UTC())
	if err == nil {
		t.Fatal("expected error on missing required fields")
	}

	// 3. Wrong type
	wrongTypeJSON := []byte(`{
		"task_id": "task-neg-val",
		"attempt_id": "attempt-neg-val",
		"status": "COMPLETED",
		"branch": "main",
		"base_sha": "35909d7b21cdfe6b9f5c309ea565c5f9f9fedeea",
		"head_sha": "1234567890abcdef",
		"files_changed": [],
		"commands_run": [],
		"tests": "not-an-array",
		"build_status": "PASSED",
		"worker_claims": [],
		"ready_for_review": true
	}`)
	err = s.IngestWorkerReportRaw(ctx, taskID, contractID, attemptID, wrongTypeJSON, cleanEvidence, "supervisor", time.Now().UTC())
	if err == nil {
		t.Fatal("expected error on wrong type")
	}

	// 4. Invalid enum/pattern
	invalidEnumJSON := []byte(`{
		"task_id": "task-neg-val",
		"attempt_id": "attempt-neg-val",
		"status": "INVALID_STATUS_ENUM",
		"branch": "main",
		"base_sha": "35909d7b21cdfe6b9f5c309ea565c5f9f9fedeea",
		"head_sha": "1234567890abcdef",
		"files_changed": [],
		"commands_run": [],
		"tests": [],
		"build_status": "PASSED",
		"worker_claims": [],
		"ready_for_review": true
	}`)
	err = s.IngestWorkerReportRaw(ctx, taskID, contractID, attemptID, invalidEnumJSON, cleanEvidence, "supervisor", time.Now().UTC())
	if err == nil {
		t.Fatal("expected error on invalid status enum")
	}

	// 5. Trailing JSON
	trailingJSON := []byte(`{
		"task_id": "task-neg-val",
		"attempt_id": "attempt-neg-val",
		"status": "COMPLETED",
		"branch": "main",
		"base_sha": "35909d7b21cdfe6b9f5c309ea565c5f9f9fedeea",
		"head_sha": "1234567890abcdef",
		"files_changed": [],
		"commands_run": [],
		"tests": [],
		"build_status": "PASSED",
		"worker_claims": [],
		"ready_for_review": true
	} {"extra": "payload"}`)
	err = s.IngestWorkerReportRaw(ctx, taskID, contractID, attemptID, trailingJSON, cleanEvidence, "supervisor", time.Now().UTC())
	if err == nil {
		t.Fatal("expected error on trailing JSON content")
	}

	// 6. Schema compile failure test seam
	testSchemaOverrideBytes = []byte(`{"type": "invalid json schema syntax`)
	defer func() { testSchemaOverrideBytes = nil }()

	validJSON := []byte(`{
		"task_id": "task-neg-val",
		"attempt_id": "attempt-neg-val",
		"status": "COMPLETED",
		"branch": "main",
		"base_sha": "35909d7b21cdfe6b9f5c309ea565c5f9f9fedeea",
		"head_sha": "1234567890abcdef",
		"files_changed": [],
		"commands_run": [],
		"tests": [],
		"build_status": "PASSED",
		"worker_claims": [],
		"ready_for_review": true
	}`)
	err = s.IngestWorkerReportRaw(ctx, taskID, contractID, attemptID, validJSON, cleanEvidence, "supervisor", time.Now().UTC())
	if err == nil {
		t.Fatal("expected error on schema compile failure")
	}
	testSchemaOverrideBytes = nil

	// 7. Verify zero DB mutation across all failure probes
	if c := countTable("tasks"); c != tasksCountBefore {
		t.Fatalf("tasks table mutated on validation failure: before=%d, after=%d", tasksCountBefore, c)
	}
	if c := countTable("audit_events"); c != auditCountBefore {
		t.Fatalf("audit_events table mutated on validation failure: before=%d, after=%d", auditCountBefore, c)
	}
	if c := countTable("review_integrity_holds"); c != holdsCountBefore {
		t.Fatalf("review_integrity_holds table mutated on validation failure: before=%d, after=%d", holdsCountBefore, c)
	}
	if c := countTable("worker_claims"); c != claimsCountBefore {
		t.Fatalf("worker_claims table mutated on validation failure: before=%d, after=%d", claimsCountBefore, c)
	}
}
