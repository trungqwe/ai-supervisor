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

	if err := s.TransitionTask(ctx, taskID, domain.StateDraft, domain.StateReady); err != nil {
		t.Fatalf("TransitionTask DRAFT -> READY failed: %v", err)
	}
}

// prepareLegacyDispatchForTest exercises the frozen P02 allocator internals.
// Production callers must use PrepareBoundDispatch.
func prepareLegacyDispatchForTest(s *Store, ctx context.Context, taskID, contractID, attemptID, expectedReportPath string, startedAt time.Time) (domain.TaskAttempt, error) {
	return s.prepareDispatch(ctx, taskID, contractID, attemptID, expectedReportPath, startedAt, nil)
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

	attempt, err := prepareLegacyDispatchForTest(s, ctx, "task-100", "contract-100", "attempt-1", canonicalPath, now)
	if err != nil {
		t.Fatalf("PrepareDispatch failed: %v", err)
	}

	if attempt.AttemptID != "attempt-1" {
		t.Errorf("attempt_id mismatch: got %s, want attempt-1", attempt.AttemptID)
	}
	if attempt.AttemptNumber != 1 {
		t.Errorf("attempt_number mismatch: got %d, want 1", attempt.AttemptNumber)
	}
	if attempt.ContractID != "contract-100" {
		t.Errorf("contract_id mismatch: got %s, want contract-100", attempt.ContractID)
	}
	if attempt.ExpectedReportPath != canonicalPath {
		t.Errorf("expected_report_path mismatch: got %s, want %s", attempt.ExpectedReportPath, canonicalPath)
	}
	if !attempt.StartedAt.Equal(now) {
		t.Errorf("started_at mismatch: got %v, want %v", attempt.StartedAt, now)
	}

	// Verify task transitioned to DISPATCHED and attempt incremented
	task, err := s.GetTask(ctx, "task-100")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if task.State != domain.StateDispatched {
		t.Errorf("task state mismatch: got %s, want DISPATCHED", task.State)
	}
	if task.CurrentAttempt != 1 {
		t.Errorf("current_attempt mismatch: got %d, want 1", task.CurrentAttempt)
	}

	// Verify contract was frozen
	c, err := s.GetTaskContract(ctx, "contract-100")
	if err != nil {
		t.Fatalf("GetTaskContract failed: %v", err)
	}
	if !c.IsImmutable {
		t.Errorf("contract is_immutable mismatch: got false, want true")
	}

	// Verify attempt persisted in store
	gotAttempt, err := s.GetTaskAttempt(ctx, "attempt-1")
	if err != nil {
		t.Fatalf("GetTaskAttempt failed: %v", err)
	}
	if gotAttempt.AttemptID != "attempt-1" || gotAttempt.AttemptNumber != 1 {
		t.Errorf("persisted attempt mismatch: %+v", gotAttempt)
	}
	t.Logf("ATOMIC_READY_TO_DISPATCHED = PASS")
}

func TestStore_LegacyPrepareDispatchIsRejected(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()
	_, err := s.PrepareDispatch(ctx, "task", "contract", "attempt", "report", time.Now().UTC())
	if !errors.Is(err, ErrBoundDispatchRequired) {
		t.Fatalf("legacy unbound API error=%v, want ErrBoundDispatchRequired", err)
	}
}

func TestStore_PrepareDispatch_CanonicalReportPathEnforcement(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()

	setupReadyTask(t, s, "task-path", "contract-path")

	// Canonical report path: .supervisor/reports/task-path/att-1.json
	canonicalPath, err := CanonicalExpectedReportPath("task-path", "att-1")
	if err != nil {
		t.Fatalf("CanonicalExpectedReportPath failed: %v", err)
	}

	// 1. Non-canonical directory path must fail
	badPath1 := ".supervisor/other/task-path/att-1.json"
	_, err = prepareLegacyDispatchForTest(s, ctx, "task-path", "contract-path", "att-1", badPath1, time.Now())
	if err == nil || !errors.Is(err, ErrReportPathMismatch) {
		t.Fatalf("expected ErrReportPathMismatch for bad path, got: %v", err)
	}

	// 2. Non-canonical filename must fail
	badPath2 := ".supervisor/reports/task-path/other.json"
	_, err = prepareLegacyDispatchForTest(s, ctx, "task-path", "contract-path", "att-1", badPath2, time.Now())
	if err == nil || !errors.Is(err, ErrReportPathMismatch) {
		t.Fatalf("expected ErrReportPathMismatch for bad filename, got: %v", err)
	}

	// 3. Traversal attack in path must fail
	badPath3 := ".supervisor/reports/task-path/../att-1.json"
	_, err = prepareLegacyDispatchForTest(s, ctx, "task-path", "contract-path", "att-1", badPath3, time.Now())
	if err == nil {
		t.Fatalf("expected error for traversal in expected_report_path, got nil")
	}

	// 4. Absolute path must fail
	badPath4 := "D:/.supervisor/reports/task-path/att-1.json"
	_, err = prepareLegacyDispatchForTest(s, ctx, "task-path", "contract-path", "att-1", badPath4, time.Now())
	if err == nil {
		t.Fatalf("expected error for absolute path in expected_report_path, got nil")
	}

	// Task must remain in READY state
	task, err := s.GetTask(ctx, "task-path")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if task.State != domain.StateReady || task.CurrentAttempt != 0 {
		t.Errorf("task state was corrupted after rejected dispatch: %+v", task)
	}

	// 5. Exact canonical path must succeed
	att, err := prepareLegacyDispatchForTest(s, ctx, "task-path", "contract-path", "att-1", canonicalPath, time.Now())
	if err != nil {
		t.Fatalf("PrepareDispatch with canonical path failed: %v", err)
	}
	if att.ExpectedReportPath != canonicalPath {
		t.Errorf("expected %s, got %s", canonicalPath, att.ExpectedReportPath)
	}
	t.Logf("CANONICAL_REPORT_PATH_ENFORCEMENT = PASS (%s)", canonicalPath)
	t.Logf("CANONICAL_REPORT_PATH = PASS")

	// 6. Section 21: Unsafe attemptID fails PrepareDispatch before transaction mutation
	setupReadyTask(t, s, "task-unsafe", "contract-unsafe")
	_, err = prepareLegacyDispatchForTest(s, ctx, "task-unsafe", "contract-unsafe", "CON", ".supervisor/reports/task-unsafe/CON.json", time.Now())
	if err == nil || !errors.Is(err, ErrReportPathMismatch) {
		t.Fatalf("expected ErrReportPathMismatch for unsafe CON attempt ID, got: %v", err)
	}

	// Verify task remains in READY, current_attempt unchanged (0), contract mutable, 0 attempts inserted
	taskUnsafe, err := s.GetTask(ctx, "task-unsafe")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if taskUnsafe.State != domain.StateReady || taskUnsafe.CurrentAttempt != 0 {
		t.Errorf("task state corrupted after rejected unsafe dispatch: %+v", taskUnsafe)
	}
	contractUnsafe, err := s.GetTaskContract(ctx, "contract-unsafe")
	if err != nil {
		t.Fatalf("GetTaskContract failed: %v", err)
	}
	if contractUnsafe.IsImmutable {
		t.Errorf("contract was frozen after rejected unsafe dispatch")
	}
	var attemptCount int
	_ = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM task_attempts WHERE task_id = 'task-unsafe'").Scan(&attemptCount)
	if attemptCount != 0 {
		t.Errorf("expected 0 attempt rows inserted, got %d", attemptCount)
	}
}

func TestStore_PrepareDispatch_StaleContractRevisionRejected(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()

	// Initial task setup with Revision 1
	setupReadyTask(t, s, "task-stale", "contract-rev-1")

	path1, _ := CanonicalExpectedReportPath("task-stale", "att-1")
	att1, err := prepareLegacyDispatchForTest(s, ctx, "task-stale", "contract-rev-1", "att-1", path1, time.Now())
	if err != nil {
		t.Fatalf("initial dispatch failed: %v", err)
	}
	if att1.AttemptNumber != 1 {
		t.Errorf("expected attempt 1, got %d", att1.AttemptNumber)
	}

	// Progress task canonically to REVISION_REQUIRED
	if err := s.TransitionTask(ctx, "task-stale", domain.StateDispatched, domain.StateRunning); err != nil {
		t.Fatalf("TransitionTask DISPATCHED -> RUNNING failed: %v", err)
	}
	if err := s.TransitionTask(ctx, "task-stale", domain.StateRunning, domain.StateReportReady); err != nil {
		t.Fatalf("TransitionTask RUNNING -> REPORT_READY failed: %v", err)
	}
	if err := s.TransitionTask(ctx, "task-stale", domain.StateReportReady, domain.StateEvidenceReady); err != nil {
		t.Fatalf("TransitionTask REPORT_READY -> EVIDENCE_READY failed: %v", err)
	}
	if err := s.TransitionTask(ctx, "task-stale", domain.StateEvidenceReady, domain.StateReviewing); err != nil {
		t.Fatalf("TransitionTask EVIDENCE_READY -> REVIEWING failed: %v", err)
	}
	if err := s.TransitionTask(ctx, "task-stale", domain.StateReviewing, domain.StateRevisionRequired); err != nil {
		t.Fatalf("TransitionTask REVIEWING -> REVISION_REQUIRED failed: %v", err)
	}

	// Finding R3-001: Attempting REVISION_REQUIRED -> READY without inserting new mutable revision must FAIL
	err = s.TransitionTask(ctx, "task-stale", domain.StateRevisionRequired, domain.StateReady)
	if err == nil || !errors.Is(err, ErrReadyContractRequired) {
		t.Fatalf("expected ErrReadyContractRequired when transitioning REVISION_REQUIRED -> READY without new revision, got: %v", err)
	}
	t.Logf("REVISION_REQUIRED_READY_WITHOUT_NEW_REVISION = REJECTED (%v)", err)

	// Verify task remains in REVISION_REQUIRED
	staleTask, err := s.GetTask(ctx, "task-stale")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if staleTask.State != domain.StateRevisionRequired {
		t.Fatalf("task state corrupted: got %s, want REVISION_REQUIRED", staleTask.State)
	}

	// Insert valid Rev 2 while in REVISION_REQUIRED (Finding R3-002)
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

	// Now REVISION_REQUIRED -> READY must succeed (Finding R3-001)
	if err := s.TransitionTask(ctx, "task-stale", domain.StateRevisionRequired, domain.StateReady); err != nil {
		t.Fatalf("TransitionTask REVISION_REQUIRED -> READY failed after rev 2: %v", err)
	}
	t.Logf("REVISION_REQUIRED_READY_WITH_NEW_REVISION = PASS")

	// Attempting to dispatch with stale Rev 1 must fail with ErrStaleContractRevision
	path2, _ := CanonicalExpectedReportPath("task-stale", "att-2")
	_, err = prepareLegacyDispatchForTest(s, ctx, "task-stale", "contract-rev-1", "att-2", path2, time.Now())
	if err == nil || !errors.Is(err, ErrStaleContractRevision) {
		t.Fatalf("expected ErrStaleContractRevision when dispatching superseded revision, got: %v", err)
	}
	t.Logf("STALE_CONTRACT_REVISION_DISPATCH = REJECTED (%v)", err)

	// Task must remain in READY state
	task, err := s.GetTask(ctx, "task-stale")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if task.State != domain.StateReady || task.CurrentAttempt != 1 {
		t.Errorf("task state corrupted after rejected dispatch: %+v", task)
	}

	// Dispatching with latest revision (Rev 2) must succeed
	att2, err := prepareLegacyDispatchForTest(s, ctx, "task-stale", "contract-rev-2", "att-2", path2, time.Now())
	if err != nil {
		t.Fatalf("PrepareDispatch with latest revision failed: %v", err)
	}
	if att2.ContractID != "contract-rev-2" {
		t.Errorf("expected contract-rev-2, got %s", att2.ContractID)
	}
	t.Logf("LATEST_CONTRACT_REVISION_BINDING = PASS")
}

func TestStore_PrepareDispatch_RetryReusesLatestImmutableContract(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()

	setupReadyTask(t, s, "task-retry-imm", "c-retry-1")

	// Attempt 1 dispatch freezes c-retry-1
	p1, _ := CanonicalExpectedReportPath("task-retry-imm", "att-1")
	att1, err := prepareLegacyDispatchForTest(s, ctx, "task-retry-imm", "c-retry-1", "att-1", p1, time.Now())
	if err != nil {
		t.Fatalf("attempt 1 failed: %v", err)
	}
	if att1.AttemptNumber != 1 {
		t.Errorf("expected attempt 1, got %d", att1.AttemptNumber)
	}

	// Transition DISPATCHED -> FAILED
	if err := s.TransitionTask(ctx, "task-retry-imm", domain.StateDispatched, domain.StateFailed); err != nil {
		t.Fatalf("transition DISPATCHED -> FAILED failed: %v", err)
	}

	// Finding R3-002: Attempting to insert a new contract while FAILED must FAIL
	cRetry1ID := "c-retry-1"
	cMutated := domain.TaskContract{
		ContractID:           "c-retry-2-smuggle",
		TaskID:               "task-retry-imm",
		RevisionNumber:       2,
		SupersedesContractID: &cRetry1ID,
		BaseSHA:              "87fa16a001a825753bec9e3c5dd36d511e4c5318",
	}
	err = s.InsertTaskContract(ctx, cMutated)
	if err == nil || !errors.Is(err, ErrContractInsertState) {
		t.Fatalf("expected ErrContractInsertState when inserting contract in FAILED state, got: %v", err)
	}
	t.Logf("CONTRACT_INSERT_WHILE_FAILED = REJECTED (%v)", err)

	// FAILED -> READY retry: requires latest contract is immutable (c-retry-1 is immutable)
	if err := s.TransitionTask(ctx, "task-retry-imm", domain.StateFailed, domain.StateReady); err != nil {
		t.Fatalf("transition FAILED -> READY failed: %v", err)
	}

	// Attempt 2 retry: c-retry-1 is already frozen (is_immutable = 1).
	// Per Section 19, retry MUST succeed by reusing the already-frozen latest revision!
	p2, _ := CanonicalExpectedReportPath("task-retry-imm", "att-2")
	att2, err := prepareLegacyDispatchForTest(s, ctx, "task-retry-imm", "c-retry-1", "att-2", p2, time.Now())
	if err != nil {
		t.Fatalf("retry dispatch failed: %v", err)
	}
	if att2.AttemptNumber != 2 {
		t.Errorf("expected attempt 2, got %d", att2.AttemptNumber)
	}
	t.Logf("FAILED_RETRY_REUSES_IMMUTABLE_CONTRACT = PASS (attempt 2 allocated)")
	t.Logf("FAILED_DIRECT_RETRY_IMMUTABLE_CONTRACT = PASS")
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

	// Create Task A and Task B under the SAME pair in DRAFT
	taskA := domain.Task{TaskID: "task-A", PhaseID: "P02", PairID: "pair-lane", State: domain.StateDraft, CurrentAttempt: 0}
	s.CreateTask(ctx, taskA)

	taskB := domain.Task{TaskID: "task-B", PhaseID: "P02", PairID: "pair-lane", State: domain.StateDraft, CurrentAttempt: 0}
	s.CreateTask(ctx, taskB)

	// Insert contracts while tasks are in DRAFT
	cA := domain.TaskContract{ContractID: "cA", TaskID: "task-A", RevisionNumber: 1, BaseSHA: "87fa16a001a825753bec9e3c5dd36d511e4c5318"}
	if err := s.InsertTaskContract(ctx, cA); err != nil {
		t.Fatalf("insert contract A failed: %v", err)
	}

	cB := domain.TaskContract{ContractID: "cB", TaskID: "task-B", RevisionNumber: 1, BaseSHA: "87fa16a001a825753bec9e3c5dd36d511e4c5318"}
	if err := s.InsertTaskContract(ctx, cB); err != nil {
		t.Fatalf("insert contract B failed: %v", err)
	}

	// Transition both tasks DRAFT -> READY
	if err := s.TransitionTask(ctx, "task-A", domain.StateDraft, domain.StateReady); err != nil {
		t.Fatalf("transition task A to READY failed: %v", err)
	}
	if err := s.TransitionTask(ctx, "task-B", domain.StateDraft, domain.StateReady); err != nil {
		t.Fatalf("transition task B to READY failed: %v", err)
	}

	// 1. Dispatch Task A -> Task A enters DISPATCHED (active lane)
	pA, _ := CanonicalExpectedReportPath("task-A", "att-A1")
	_, err := prepareLegacyDispatchForTest(s, ctx, "task-A", "cA", "att-A1", pA, time.Now())
	if err != nil {
		t.Fatalf("dispatch task A failed: %v", err)
	}

	// 2. Finding R2-006: Dispatching Task B on same pair while Task A is DISPATCHED must fail with ErrPairBusy
	pB, _ := CanonicalExpectedReportPath("task-B", "att-B1")
	_, err = prepareLegacyDispatchForTest(s, ctx, "task-B", "cB", "att-B1", pB, time.Now())
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

	// 3. Progress Task A: DISPATCHED -> FAILED (leaves active lane)
	if err := s.TransitionTask(ctx, "task-A", domain.StateDispatched, domain.StateFailed); err != nil {
		t.Fatalf("transition task A to FAILED failed: %v", err)
	}

	// 4. Now that Task A left active lane, dispatching Task B must succeed
	_, err = prepareLegacyDispatchForTest(s, ctx, "task-B", "cB", "att-B1", pB, time.Now())
	if err != nil {
		t.Fatalf("dispatch task B after task A failed should succeed, got: %v", err)
	}
	t.Logf("PAIR_ACTIVE_LANE_SINGLE_TASK = PASS")
}

func TestStore_PairActiveLane_ReviewingTransitionBlocked(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()

	// Create 1 Project and 1 Pair
	proj := domain.Project{ProjectID: "proj-rev-busy", Name: "Rev Busy Proj", RootPath: "/revbusy"}
	s.CreateProject(ctx, proj)
	pair := domain.Pair{PairID: "pair-rev-busy", ProjectID: "proj-rev-busy", CurrentPhaseID: "P02", State: "ACTIVE"}
	s.CreatePair(ctx, pair)

	// Task A and Task B under the same Pair
	taskA := domain.Task{TaskID: "task-A-rev", PhaseID: "P02", PairID: "pair-rev-busy", State: domain.StateDraft, CurrentAttempt: 0}
	s.CreateTask(ctx, taskA)
	taskB := domain.Task{TaskID: "task-B-rev", PhaseID: "P02", PairID: "pair-rev-busy", State: domain.StateDraft, CurrentAttempt: 0}
	s.CreateTask(ctx, taskB)

	cA := domain.TaskContract{ContractID: "cA-rev", TaskID: "task-A-rev", RevisionNumber: 1, BaseSHA: "87fa16a001a825753bec9e3c5dd36d511e4c5318"}
	s.InsertTaskContract(ctx, cA)
	cB := domain.TaskContract{ContractID: "cB-rev", TaskID: "task-B-rev", RevisionNumber: 1, BaseSHA: "87fa16a001a825753bec9e3c5dd36d511e4c5318"}
	s.InsertTaskContract(ctx, cB)

	s.TransitionTask(ctx, "task-A-rev", domain.StateDraft, domain.StateReady)
	s.TransitionTask(ctx, "task-B-rev", domain.StateDraft, domain.StateReady)

	// Step 1: Progress Task B canonically to EVIDENCE_READY while Task A is not yet active
	pB, _ := CanonicalExpectedReportPath("task-B-rev", "att-B1")
	_, err := prepareLegacyDispatchForTest(s, ctx, "task-B-rev", "cB-rev", "att-B1", pB, time.Now())
	if err != nil {
		t.Fatalf("dispatch task B failed: %v", err)
	}
	s.TransitionTask(ctx, "task-B-rev", domain.StateDispatched, domain.StateRunning)
	s.TransitionTask(ctx, "task-B-rev", domain.StateRunning, domain.StateReportReady)
	s.TransitionTask(ctx, "task-B-rev", domain.StateReportReady, domain.StateEvidenceReady)

	// Step 2: Now Task B is in EVIDENCE_READY (not an active lane state).
	// Task A can now be dispatched to DISPATCHED (entering active lane).
	pA, _ := CanonicalExpectedReportPath("task-A-rev", "att-A1")
	_, err = prepareLegacyDispatchForTest(s, ctx, "task-A-rev", "cA-rev", "att-A1", pA, time.Now())
	if err != nil {
		t.Fatalf("dispatch task A failed: %v", err)
	}

	// Step 3: Section 23 — While Task A is DISPATCHED, Task B attempts EVIDENCE_READY -> REVIEWING
	busyErr := s.TransitionTask(ctx, "task-B-rev", domain.StateEvidenceReady, domain.StateReviewing)
	if busyErr == nil || !errors.Is(busyErr, ErrPairBusy) {
		t.Fatalf("expected ErrPairBusy when transitioning to REVIEWING while another task is DISPATCHED, got: %v", busyErr)
	}

	// Verify Task B remains EVIDENCE_READY
	tB, errGet := s.GetTask(ctx, "task-B-rev")
	if errGet != nil {
		t.Fatalf("GetTask failed: %v", errGet)
	}
	if tB.State != domain.StateEvidenceReady {
		t.Errorf("task B state corrupted: got %s, want EVIDENCE_READY", tB.State)
	}
	t.Logf("PAIR_REVIEWING_ENTRY_WHILE_BUSY = REJECTED (%v)", busyErr)
}

func TestStore_PrepareDispatch_ConcurrentSamePairSingleWinner(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()

	// Create 1 Project and 1 Pair
	proj := domain.Project{ProjectID: "proj-pair-race", Name: "Pair Race Proj", RootPath: "/race"}
	s.CreateProject(ctx, proj)
	pair := domain.Pair{PairID: "pair-race", ProjectID: "proj-pair-race", CurrentPhaseID: "P02", State: "ACTIVE"}
	s.CreatePair(ctx, pair)

	// Create Task A and Task B on the SAME pair
	taskA := domain.Task{TaskID: "task-pair-A", PhaseID: "P02", PairID: "pair-race", State: domain.StateDraft, CurrentAttempt: 0}
	s.CreateTask(ctx, taskA)
	taskB := domain.Task{TaskID: "task-pair-B", PhaseID: "P02", PairID: "pair-race", State: domain.StateDraft, CurrentAttempt: 0}
	s.CreateTask(ctx, taskB)

	cA := domain.TaskContract{ContractID: "c-pair-A", TaskID: "task-pair-A", RevisionNumber: 1, BaseSHA: "87fa16a001a825753bec9e3c5dd36d511e4c5318"}
	s.InsertTaskContract(ctx, cA)
	cB := domain.TaskContract{ContractID: "c-pair-B", TaskID: "task-pair-B", RevisionNumber: 1, BaseSHA: "87fa16a001a825753bec9e3c5dd36d511e4c5318"}
	s.InsertTaskContract(ctx, cB)

	s.TransitionTask(ctx, "task-pair-A", domain.StateDraft, domain.StateReady)
	s.TransitionTask(ctx, "task-pair-B", domain.StateDraft, domain.StateReady)

	// Section 24: Concurrent dispatch on two READY tasks of the same Pair
	var wg sync.WaitGroup
	wg.Add(2)

	errs := make([]error, 2)
	attempts := make([]domain.TaskAttempt, 2)

	go func() {
		defer wg.Done()
		pA, _ := CanonicalExpectedReportPath("task-pair-A", "att-pair-A1")
		att, err := prepareLegacyDispatchForTest(s, ctx, "task-pair-A", "c-pair-A", "att-pair-A1", pA, time.Now())
		errs[0] = err
		attempts[0] = att
	}()

	go func() {
		defer wg.Done()
		pB, _ := CanonicalExpectedReportPath("task-pair-B", "att-pair-B1")
		att, err := prepareLegacyDispatchForTest(s, ctx, "task-pair-B", "c-pair-B", "att-pair-B1", pB, time.Now())
		errs[1] = err
		attempts[1] = att
	}()

	wg.Wait()

	successCount := 0
	pairBusyCount := 0
	var winningTaskID, losingTaskID string
	var losingContractID string

	if errs[0] == nil {
		successCount++
		winningTaskID = "task-pair-A"
	} else if errors.Is(errs[0], ErrPairBusy) {
		pairBusyCount++
		losingTaskID = "task-pair-A"
		losingContractID = "c-pair-A"
	}

	if errs[1] == nil {
		successCount++
		winningTaskID = "task-pair-B"
	} else if errors.Is(errs[1], ErrPairBusy) {
		pairBusyCount++
		losingTaskID = "task-pair-B"
		losingContractID = "c-pair-B"
	}

	if successCount != 1 || pairBusyCount != 1 {
		t.Fatalf("expected exactly 1 success and 1 ErrPairBusy, got %d successes and %d busy errors (errs: %v, %v)",
			successCount, pairBusyCount, errs[0], errs[1])
	}

	// Winning task must be DISPATCHED
	winTask, err := s.GetTask(ctx, winningTaskID)
	if err != nil {
		t.Fatalf("GetTask winning failed: %v", err)
	}
	if winTask.State != domain.StateDispatched || winTask.CurrentAttempt != 1 {
		t.Errorf("winning task state mismatch: %+v", winTask)
	}

	// Losing task must remain READY with current_attempt = 0
	loseTask, err := s.GetTask(ctx, losingTaskID)
	if err != nil {
		t.Fatalf("GetTask losing failed: %v", err)
	}
	if loseTask.State != domain.StateReady || loseTask.CurrentAttempt != 0 {
		t.Errorf("losing task state mismatch: %+v", loseTask)
	}

	// Losing contract must remain mutable
	loseContract, err := s.GetTaskContract(ctx, losingContractID)
	if err != nil {
		t.Fatalf("GetTaskContract losing failed: %v", err)
	}
	if loseContract.IsImmutable {
		t.Errorf("losing contract was frozen on ErrPairBusy")
	}

	// Total TaskAttempt rows for pair must be exactly 1
	var totalAttempts int
	err = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM task_attempts WHERE task_id IN ('task-pair-A', 'task-pair-B')").Scan(&totalAttempts)
	if err != nil {
		t.Fatalf("failed to count attempts: %v", err)
	}
	if totalAttempts != 1 {
		t.Errorf("expected exactly 1 attempt row in DB, got %d", totalAttempts)
	}

	t.Logf("CONCURRENT_SAME_PAIR_DISPATCH = SINGLE_WINNER (winner=%s, loser=%s)", winningTaskID, losingTaskID)
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
			att, err := prepareLegacyDispatchForTest(s, ctx, "task-race", "contract-race", attID, repPath, time.Now())
			results[idx] = err
			attempts[idx] = att
		}()
	}

	wg.Wait()

	successCount := 0
	failureCount := 0
	var loserErr error
	for _, res := range results {
		if res == nil {
			successCount++
		} else {
			failureCount++
			loserErr = res
		}
	}

	if successCount != 1 || failureCount != 1 {
		t.Fatalf("expected exactly 1 success and 1 failure, got %d successes and %d failures (errors: %v, %v)",
			successCount, failureCount, results[0], results[1])
	}
	t.Logf("SAME_TASK_CONCURRENT_DISPATCH_SINGLE_WINNER = PASS: 1 winner, 1 conflict (%v)", loserErr)
}

func TestStore_PrepareDispatch_RollbackOnWrongTaskContract(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()

	setupReadyTask(t, s, "task-A-wb", "contract-A-wb")
	setupReadyTask(t, s, "task-B-wb", "contract-B-wb")

	pathWrong, _ := CanonicalExpectedReportPath("task-A-wb", "attempt-wrong")
	_, err := prepareLegacyDispatchForTest(s, ctx, "task-A-wb", "contract-B-wb", "attempt-wrong", pathWrong, time.Now())
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
	_, err := prepareLegacyDispatchForTest(s, ctx, "task-dup-1", "contract-dup-1", "att-dup-same", p1, time.Now())
	if err != nil {
		t.Fatalf("first dispatch failed: %v", err)
	}

	// Progress task-dup-1 out of active lane so pair-lane check doesn't block first
	_ = s.TransitionTask(ctx, "task-dup-1", domain.StateDispatched, domain.StateFailed)

	p2, _ := CanonicalExpectedReportPath("task-dup-2", "att-dup-same")
	_, err = prepareLegacyDispatchForTest(s, ctx, "task-dup-2", "contract-dup-2", "att-dup-same", p2, time.Now())
	if err == nil || !errors.Is(err, ErrDuplicateKey) {
		t.Fatalf("expected ErrDuplicateKey, got %v", err)
	}

	t2, _ := s.GetTask(ctx, "task-dup-2")
	if t2.State != domain.StateReady || t2.CurrentAttempt != 0 {
		t.Errorf("task-dup-2 rollback failed: %+v", t2)
	}
}
