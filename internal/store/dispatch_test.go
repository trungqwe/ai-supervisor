package store

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/trungqwe/ai-supervisor/internal/domain"
)

func setupReadyTask(t *testing.T, s *Store, taskID, contractID string) {
	t.Helper()
	ctx := context.Background()

	proj := domain.Project{ProjectID: "proj-d-" + taskID, Name: "Proj Dispatch", RootPath: "/dispatch"}
	_ = s.CreateProject(ctx, proj)
	pair := domain.Pair{PairID: "pair-d-" + taskID, ProjectID: "proj-d-" + taskID, CurrentPhaseID: "P02", State: "ACTIVE"}
	_ = s.CreatePair(ctx, pair)

	task := domain.Task{
		TaskID:         taskID,
		PhaseID:        "P02",
		PairID:         "pair-d-" + taskID,
		State:          domain.StateDraft,
		CurrentAttempt: 0,
	}
	if err := s.CreateTask(ctx, task); err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}
	if err := s.TransitionTask(ctx, taskID, domain.StateDraft, domain.StateReady); err != nil {
		t.Fatalf("TransitionTask DRAFT -> READY failed: %v", err)
	}

	contract := domain.TaskContract{
		ContractID:     contractID,
		TaskID:         taskID,
		RevisionNumber: 1,
		BaseSHA:        "87fa16a001a825753bec9e3c5dd36d511e4c5318",
		AllowedScope:   []string{"internal/store/**"},
	}
	if err := s.InsertTaskContract(ctx, contract); err != nil {
		t.Fatalf("InsertTaskContract failed: %v", err)
	}
}

func TestStore_PrepareDispatch_Success(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()

	setupReadyTask(t, s, "task-100", "contract-100")

	now := time.Now().UTC()
	canonicalPath, err := CanonicalExpectedReportPath("task-100", "attempt-1")
	if err != nil {
		t.Fatalf("CanonicalExpectedReportPath failed: %v", err)
	}

	attempt, err := s.PrepareDispatch(ctx, "task-100", "contract-100", "attempt-1", canonicalPath, now)
	if err != nil {
		t.Fatalf("PrepareDispatch failed: %v", err)
	}

	if attempt.AttemptID != "attempt-1" {
		t.Errorf("attempt_id mismatch: got %s, want attempt-1", attempt.AttemptID)
	}
	if attempt.AttemptNumber != 1 {
		t.Errorf("attempt_number mismatch: got %d, want 1", attempt.AttemptNumber)
	}
	if attempt.ExpectedReportPath != canonicalPath {
		t.Errorf("expected_report_path mismatch: got %s, want %s", attempt.ExpectedReportPath, canonicalPath)
	}

	// Task must now be DISPATCHED with current_attempt = 1
	tsk, err := s.GetTask(ctx, "task-100")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if tsk.State != domain.StateDispatched {
		t.Errorf("expected state DISPATCHED, got %s", tsk.State)
	}
	if tsk.CurrentAttempt != 1 {
		t.Errorf("expected current_attempt 1, got %d", tsk.CurrentAttempt)
	}

	// Contract must now be immutable
	c, err := s.GetTaskContract(ctx, "contract-100")
	if err != nil {
		t.Fatalf("GetTaskContract failed: %v", err)
	}
	if !c.IsImmutable {
		t.Errorf("expected contract to be frozen (is_immutable = true)")
	}

	// TaskAttempt row must exist and match
	att, err := s.GetTaskAttempt(ctx, "attempt-1")
	if err != nil {
		t.Fatalf("GetTaskAttempt failed: %v", err)
	}
	if att.AttemptNumber != 1 || att.TaskID != "task-100" || att.ContractID != "contract-100" {
		t.Errorf("persisted attempt mismatch: %+v", att)
	}
}

func TestStore_PrepareDispatch_CanonicalReportPathEnforcement(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()

	setupReadyTask(t, s, "task-path", "contract-path")

	// 1. Noncanonical path rejected (Finding R2-003)
	_, err := s.PrepareDispatch(ctx, "task-path", "contract-path", "att-1", "reports/report.json", time.Now())
	if err == nil || !errors.Is(err, ErrReportPathMismatch) {
		t.Errorf("expected ErrReportPathMismatch for arbitrary path 'reports/report.json', got %v", err)
	}

	// 2. Static singleton path rejected
	_, err = s.PrepareDispatch(ctx, "task-path", "contract-path", "att-1", ".supervisor/worker-report.json", time.Now())
	if err == nil || !errors.Is(err, ErrReportPathMismatch) {
		t.Errorf("expected ErrReportPathMismatch for static singleton path, got %v", err)
	}

	// 3. Wrong attempt ID in path rejected
	wrongAttemptPath := ".supervisor/reports/task-path/other-att.json"
	_, err = s.PrepareDispatch(ctx, "task-path", "contract-path", "att-1", wrongAttemptPath, time.Now())
	if err == nil || !errors.Is(err, ErrReportPathMismatch) {
		t.Errorf("expected ErrReportPathMismatch for wrong attempt in path, got %v", err)
	}

	// 4. Attempt ID with path traversal rejected
	_, err = s.PrepareDispatch(ctx, "task-path", "contract-path", "../att", ".supervisor/reports/task-path/../att.json", time.Now())
	if err == nil || !errors.Is(err, ErrReportPathMismatch) {
		t.Errorf("expected ErrReportPathMismatch for attempt ID with path traversal, got %v", err)
	}

	// 5. Attempt ID with whitespace rejected
	_, err = s.PrepareDispatch(ctx, "task-path", "contract-path", " att-1 ", ".supervisor/reports/task-path/ att-1 .json", time.Now())
	if err == nil || !errors.Is(err, ErrReportPathMismatch) {
		t.Errorf("expected ErrReportPathMismatch for whitespace attempt ID, got %v", err)
	}

	// 6. Verify task remains READY with current_attempt = 0 and contract mutable
	tsk, err := s.GetTask(ctx, "task-path")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if tsk.State != domain.StateReady || tsk.CurrentAttempt != 0 {
		t.Errorf("task corrupted after path rejection: state=%s, attempt=%d", tsk.State, tsk.CurrentAttempt)
	}
	c, err := s.GetTaskContract(ctx, "contract-path")
	if err != nil {
		t.Fatalf("GetTaskContract failed: %v", err)
	}
	if c.IsImmutable {
		t.Errorf("contract was frozen on rejected path")
	}

	// 7. Canonical path succeeds
	canonicalPath, _ := CanonicalExpectedReportPath("task-path", "att-1")
	_, err = s.PrepareDispatch(ctx, "task-path", "contract-path", "att-1", canonicalPath, time.Now())
	if err != nil {
		t.Fatalf("canonical path dispatch failed: %v", err)
	}
	t.Logf("CANONICAL_REPORT_PATH = PASS")
}

func TestStore_PrepareDispatch_StaleContractRevisionRejected(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()

	setupReadyTask(t, s, "task-stale", "contract-rev-1")

	// Rev 1 dispatched and frozen
	path1, _ := CanonicalExpectedReportPath("task-stale", "att-1")
	_, err := s.PrepareDispatch(ctx, "task-stale", "contract-rev-1", "att-1", path1, time.Now())
	if err != nil {
		t.Fatalf("first dispatch failed: %v", err)
	}

	// Task progresses through review cycle to REVISION_REQUIRED -> READY
	if err := s.TransitionTask(ctx, "task-stale", domain.StateDispatched, domain.StateRunning); err != nil {
		t.Fatalf("transition to RUNNING failed: %v", err)
	}
	if err := s.TransitionTask(ctx, "task-stale", domain.StateRunning, domain.StateReportReady); err != nil {
		t.Fatalf("transition to REPORT_READY failed: %v", err)
	}
	if err := s.TransitionTask(ctx, "task-stale", domain.StateReportReady, domain.StateEvidenceReady); err != nil {
		t.Fatalf("transition to EVIDENCE_READY failed: %v", err)
	}
	if err := s.TransitionTask(ctx, "task-stale", domain.StateEvidenceReady, domain.StateReviewing); err != nil {
		t.Fatalf("transition to REVIEWING failed: %v", err)
	}
	if err := s.TransitionTask(ctx, "task-stale", domain.StateReviewing, domain.StateRevisionRequired); err != nil {
		t.Fatalf("transition to REVISION_REQUIRED failed: %v", err)
	}
	if err := s.TransitionTask(ctx, "task-stale", domain.StateRevisionRequired, domain.StateReady); err != nil {
		t.Fatalf("transition to READY failed: %v", err)
	}

	// Insert Rev 2
	rev1ID := "contract-rev-1"
	contractRev2 := domain.TaskContract{
		ContractID:           "contract-rev-2",
		TaskID:               "task-stale",
		RevisionNumber:       2,
		SupersedesContractID: &rev1ID,
		BaseSHA:              "87fa16a001a825753bec9e3c5dd36d511e4c5318",
		AllowedScope:         []string{"internal/store/**"},
	}
	if err := s.InsertTaskContract(ctx, contractRev2); err != nil {
		t.Fatalf("InsertTaskContract rev 2 failed: %v", err)
	}

	// Finding R2-004: Dispatching with stale revision (Rev 1) must be rejected!
	path2, _ := CanonicalExpectedReportPath("task-stale", "att-2")
	_, err = s.PrepareDispatch(ctx, "task-stale", "contract-rev-1", "att-2", path2, time.Now())
	if err == nil || !errors.Is(err, ErrStaleContractRevision) {
		t.Fatalf("expected ErrStaleContractRevision when dispatching superseded contract, got %v", err)
	}
	t.Logf("STALE_CONTRACT_REVISION = REJECTED (%v)", err)

	// Task must remain READY
	tsk, err := s.GetTask(ctx, "task-stale")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if tsk.State != domain.StateReady {
		t.Errorf("task state was corrupted after rejected dispatch: %s", tsk.State)
	}

	// Dispatching with latest revision (Rev 2) must succeed
	att2, err := s.PrepareDispatch(ctx, "task-stale", "contract-rev-2", "att-2", path2, time.Now())
	if err != nil {
		t.Fatalf("PrepareDispatch with latest revision failed: %v", err)
	}
	if att2.ContractID != "contract-rev-2" {
		t.Errorf("expected contract-rev-2, got %s", att2.ContractID)
	}
	t.Logf("LATEST_CONTRACT_REVISION_DISPATCH = PASS")
}

func TestStore_PrepareDispatch_RetryReusesLatestImmutableContract(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()

	setupReadyTask(t, s, "task-retry-imm", "c-retry-1")

	// Attempt 1 dispatch freezes c-retry-1
	p1, _ := CanonicalExpectedReportPath("task-retry-imm", "att-1")
	att1, err := s.PrepareDispatch(ctx, "task-retry-imm", "c-retry-1", "att-1", p1, time.Now())
	if err != nil {
		t.Fatalf("attempt 1 failed: %v", err)
	}
	if att1.AttemptNumber != 1 {
		t.Errorf("expected attempt 1, got %d", att1.AttemptNumber)
	}

	// Canonical failure cycle: DISPATCHED -> FAILED -> READY
	if err := s.TransitionTask(ctx, "task-retry-imm", domain.StateDispatched, domain.StateFailed); err != nil {
		t.Fatalf("transition DISPATCHED -> FAILED failed: %v", err)
	}
	if err := s.TransitionTask(ctx, "task-retry-imm", domain.StateFailed, domain.StateReady); err != nil {
		t.Fatalf("transition FAILED -> READY failed: %v", err)
	}

	// Attempt 2 retry: c-retry-1 is already frozen (is_immutable = 1).
	// Per Section 18, retry MUST succeed by reusing the already-frozen latest revision!
	p2, _ := CanonicalExpectedReportPath("task-retry-imm", "att-2")
	att2, err := s.PrepareDispatch(ctx, "task-retry-imm", "c-retry-1", "att-2", p2, time.Now())
	if err != nil {
		t.Fatalf("retry dispatch failed: %v", err)
	}
	if att2.AttemptNumber != 2 {
		t.Errorf("expected attempt 2, got %d", att2.AttemptNumber)
	}
	t.Logf("RETRY_LATEST_IMMUTABLE_CONTRACT = PASS (attempt 2 allocated)")
}

func TestStore_PairActiveLaneInvariant(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()

	// Create 1 Project and 1 Pair
	proj := domain.Project{ProjectID: "proj-lane", Name: "Lane Proj", RootPath: "/lane"}
	s.CreateProject(ctx, proj)
	pair := domain.Pair{PairID: "pair-lane", ProjectID: "proj-lane", CurrentPhaseID: "P02", State: "ACTIVE"}
	s.CreatePair(ctx, pair)

	// Create Task A and Task B under the SAME pair
	taskA := domain.Task{TaskID: "task-A", PhaseID: "P02", PairID: "pair-lane", State: domain.StateDraft, CurrentAttempt: 0}
	s.CreateTask(ctx, taskA)
	s.TransitionTask(ctx, "task-A", domain.StateDraft, domain.StateReady)

	taskB := domain.Task{TaskID: "task-B", PhaseID: "P02", PairID: "pair-lane", State: domain.StateDraft, CurrentAttempt: 0}
	s.CreateTask(ctx, taskB)
	s.TransitionTask(ctx, "task-B", domain.StateDraft, domain.StateReady)

	cA := domain.TaskContract{ContractID: "cA", TaskID: "task-A", RevisionNumber: 1, BaseSHA: "87fa16a001a825753bec9e3c5dd36d511e4c5318"}
	s.InsertTaskContract(ctx, cA)

	cB := domain.TaskContract{ContractID: "cB", TaskID: "task-B", RevisionNumber: 1, BaseSHA: "87fa16a001a825753bec9e3c5dd36d511e4c5318"}
	s.InsertTaskContract(ctx, cB)

	// 1. Dispatch Task A -> Task A enters DISPATCHED (active lane)
	pA, _ := CanonicalExpectedReportPath("task-A", "att-A1")
	_, err := s.PrepareDispatch(ctx, "task-A", "cA", "att-A1", pA, time.Now())
	if err != nil {
		t.Fatalf("dispatch task A failed: %v", err)
	}

	// 2. Finding R2-006: Dispatching Task B on same pair while Task A is DISPATCHED must fail with ErrPairBusy
	pB, _ := CanonicalExpectedReportPath("task-B", "att-B1")
	_, err = s.PrepareDispatch(ctx, "task-B", "cB", "att-B1", pB, time.Now())
	if err == nil || !errors.Is(err, ErrPairBusy) {
		t.Fatalf("expected ErrPairBusy when dispatching second task on same pair, got %v", err)
	}
	t.Logf("SECOND_ACTIVE_TASK_SAME_PAIR = REJECTED (%v)", err)

	// Verify Task B remains READY, attempt = 0, cB mutable
	tB, err := s.GetTask(ctx, "task-B")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if tB.State != domain.StateReady || tB.CurrentAttempt != 0 {
		t.Errorf("task B corrupted: %+v", tB)
	}
	cBGot, err := s.GetTaskContract(ctx, "cB")
	if err != nil {
		t.Fatalf("GetTaskContract failed: %v", err)
	}
	if cBGot.IsImmutable {
		t.Errorf("contract B was frozen on busy pair")
	}

	// 3. While Task A is DISPATCHED, Task B cannot transition to RUNNING or REVIEWING
	// (Even if Task B were at EVIDENCE_READY, it would be blocked from entering REVIEWING)
	// Progress Task A: DISPATCHED -> FAILED (leaves active lane: DISPATCHED/RUNNING/REVIEWING)
	if err := s.TransitionTask(ctx, "task-A", domain.StateDispatched, domain.StateFailed); err != nil {
		t.Fatalf("transition task A to FAILED failed: %v", err)
	}

	// 4. Now that Task A left active lane, dispatching Task B must succeed
	_, err = s.PrepareDispatch(ctx, "task-B", "cB", "att-B1", pB, time.Now())
	if err != nil {
		t.Fatalf("dispatch task B after task A failed should succeed, got: %v", err)
	}
	t.Logf("PAIR_ACTIVE_LANE_SINGLE_TASK = PASS")
}

func TestStore_PrepareDispatch_ConcurrentSingleWinner(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()

	setupReadyTask(t, s, "task-race", "contract-race")

	var wg sync.WaitGroup
	wg.Add(2)

	results := make([]error, 2)
	attempts := make([]domain.TaskAttempt, 2)

	for i := 0; i < 2; i++ {
		idx := i
		go func() {
			defer wg.Done()
			attID := "att-race-1"
			if idx == 1 {
				attID = "att-race-2"
			}
			repPath, _ := CanonicalExpectedReportPath("task-race", attID)
			att, err := s.PrepareDispatch(ctx, "task-race", "contract-race", attID, repPath, time.Now())
			results[idx] = err
			attempts[idx] = att
		}()
	}

	wg.Wait()

	successCount := 0
	failureCount := 0
	for _, res := range results {
		if res == nil {
			successCount++
		} else {
			failureCount++
		}
	}

	if successCount != 1 || failureCount != 1 {
		t.Fatalf("expected exactly 1 success and 1 failure, got %d successes and %d failures (errors: %v, %v)",
			successCount, failureCount, results[0], results[1])
	}
	t.Logf("SAME_TASK_CONCURRENT_DISPATCH_SINGLE_WINNER = PASS: 1 winner, 1 conflict (%v)", results[0])
}

func TestStore_PrepareDispatch_RollbackOnWrongTaskContract(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()

	setupReadyTask(t, s, "task-A-wb", "contract-A-wb")
	setupReadyTask(t, s, "task-B-wb", "contract-B-wb")

	pathWrong, _ := CanonicalExpectedReportPath("task-A-wb", "attempt-wrong")
	_, err := s.PrepareDispatch(ctx, "task-A-wb", "contract-B-wb", "attempt-wrong", pathWrong, time.Now())
	if err == nil || !errors.Is(err, ErrContractNotOwned) {
		t.Fatalf("expected ErrContractNotOwned, got %v", err)
	}

	// Verify rollback
	tskA, _ := s.GetTask(ctx, "task-A-wb")
	if tskA.State != domain.StateReady || tskA.CurrentAttempt != 0 {
		t.Errorf("task-A rollback failed: %+v", tskA)
	}
	cBWb, _ := s.GetTaskContract(ctx, "contract-B-wb")
	if cBWb.IsImmutable {
		t.Errorf("contract-B was frozen on rolled-back attempt")
	}
}

func TestStore_PrepareDispatch_RollbackOnDuplicateAttemptID(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()

	setupReadyTask(t, s, "task-dup-1", "contract-dup-1")
	setupReadyTask(t, s, "task-dup-2", "contract-dup-2")

	p1, _ := CanonicalExpectedReportPath("task-dup-1", "att-dup-same")
	_, err := s.PrepareDispatch(ctx, "task-dup-1", "contract-dup-1", "att-dup-same", p1, time.Now())
	if err != nil {
		t.Fatalf("first dispatch failed: %v", err)
	}

	// Progress task-dup-1 out of active lane so pair-lane check doesn't block first
	_ = s.TransitionTask(ctx, "task-dup-1", domain.StateDispatched, domain.StateFailed)

	p2, _ := CanonicalExpectedReportPath("task-dup-2", "att-dup-same")
	_, err = s.PrepareDispatch(ctx, "task-dup-2", "contract-dup-2", "att-dup-same", p2, time.Now())
	if err == nil || !errors.Is(err, ErrDuplicateKey) {
		t.Fatalf("expected ErrDuplicateKey, got %v", err)
	}

	t2, _ := s.GetTask(ctx, "task-dup-2")
	if t2.State != domain.StateReady || t2.CurrentAttempt != 0 {
		t.Errorf("task-dup-2 rollback failed: %+v", t2)
	}
}
