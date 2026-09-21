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

	proj := domain.Project{ProjectID: "proj-d", Name: "Proj Dispatch", RootPath: "/dispatch"}
	_ = s.CreateProject(ctx, proj)
	pair := domain.Pair{PairID: "pair-d", ProjectID: "proj-d", CurrentPhaseID: "P02", State: "ACTIVE"}
	_ = s.CreatePair(ctx, pair)

	task := domain.Task{
		TaskID:         taskID,
		PhaseID:        "P02",
		PairID:         "pair-d",
		State:          domain.StateReady,
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
		IsImmutable:    false,
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
	attempt, err := s.PrepareDispatch(ctx, "task-100", "contract-100", "attempt-1", "reports/report.json", now)
	if err != nil {
		t.Fatalf("PrepareDispatch failed: %v", err)
	}

	if attempt.AttemptID != "attempt-1" {
		t.Errorf("attempt_id mismatch: got %s, want attempt-1", attempt.AttemptID)
	}
	if attempt.AttemptNumber != 1 {
		t.Errorf("attempt_number mismatch: got %d, want 1", attempt.AttemptNumber)
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

func TestStore_PrepareDispatch_NonReadyFails(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()

	setupReadyTask(t, s, "task-nonready", "contract-nr")

	// Transition task to RUNNING via DISPATCHED
	_, err := s.PrepareDispatch(ctx, "task-nonready", "contract-nr", "att-nr-1", "reports/r.json", time.Now())
	if err != nil {
		t.Fatalf("first dispatch failed: %v", err)
	}

	// Now task is DISPATCHED, calling PrepareDispatch again must fail
	_, err = s.PrepareDispatch(ctx, "task-nonready", "contract-nr", "att-nr-2", "reports/r.json", time.Now())
	if err == nil {
		t.Fatalf("expected dispatch on DISPATCHED task to fail, got nil")
	}
	if !errors.Is(err, ErrTaskNotReady) {
		t.Errorf("expected ErrTaskNotReady, got %v", err)
	}
}

func TestStore_PrepareDispatch_RollbackOnWrongTaskContract(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()

	setupReadyTask(t, s, "task-A", "contract-A")
	setupReadyTask(t, s, "task-B", "contract-B")

	// Attempt to dispatch task-A with contract-B (which belongs to task-B)
	_, err := s.PrepareDispatch(ctx, "task-A", "contract-B", "attempt-wrong", "reports/r.json", time.Now())
	if err == nil {
		t.Fatalf("expected dispatch with wrong contract to fail, got nil")
	}
	if !errors.Is(err, ErrContractNotOwned) {
		t.Errorf("expected ErrContractNotOwned, got %v", err)
	}

	// Verify complete rollback: task-A MUST still be READY with current_attempt = 0
	tsk, err := s.GetTask(ctx, "task-A")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if tsk.State != domain.StateReady {
		t.Errorf("task state was corrupted: got %s, want READY", tsk.State)
	}
	if tsk.CurrentAttempt != 0 {
		t.Errorf("task current_attempt was corrupted: got %d, want 0", tsk.CurrentAttempt)
	}

	// contract-B MUST NOT be frozen
	cB, err := s.GetTaskContract(ctx, "contract-B")
	if err != nil {
		t.Fatalf("GetTaskContract failed: %v", err)
	}
	if cB.IsImmutable {
		t.Errorf("contract-B was frozen on rolled-back attempt")
	}

	// No attempt record created
	_, err = s.GetTaskAttempt(ctx, "attempt-wrong")
	if !errors.Is(err, ErrAttemptNotFound) {
		t.Errorf("expected ErrAttemptNotFound, got %v", err)
	}
}

func TestStore_PrepareDispatch_RollbackOnDuplicateAttemptID(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()

	setupReadyTask(t, s, "task-1", "contract-1")
	setupReadyTask(t, s, "task-2", "contract-2")

	// Successfully dispatch task-1 with attempt-dup
	_, err := s.PrepareDispatch(ctx, "task-1", "contract-1", "attempt-dup", "r.json", time.Now())
	if err != nil {
		t.Fatalf("first dispatch failed: %v", err)
	}

	// Try to dispatch task-2 with the SAME attempt-dup ID -> fails duplicate PK
	_, err = s.PrepareDispatch(ctx, "task-2", "contract-2", "attempt-dup", "r.json", time.Now())
	if err == nil {
		t.Fatalf("expected duplicate attempt ID to fail, got nil")
	}
	if !errors.Is(err, ErrDuplicateKey) {
		t.Errorf("expected ErrDuplicateKey, got %v", err)
	}

	// Verify rollback on task-2: must still be READY, attempt = 0, contract-2 not frozen
	t2, err := s.GetTask(ctx, "task-2")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if t2.State != domain.StateReady || t2.CurrentAttempt != 0 {
		t.Errorf("task-2 rollback failed: state=%s, attempt=%d", t2.State, t2.CurrentAttempt)
	}

	c2, err := s.GetTaskContract(ctx, "contract-2")
	if err != nil {
		t.Fatalf("GetTaskContract failed: %v", err)
	}
	if c2.IsImmutable {
		t.Errorf("contract-2 was frozen on rolled-back attempt")
	}
}

func TestStore_PrepareDispatch_RetryAttemptNumberIncrements(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()

	setupReadyTask(t, s, "task-retry", "contract-retry")

	// Attempt 1: READY -> DISPATCHED
	att1, err := s.PrepareDispatch(ctx, "task-retry", "contract-retry", "att-r1", "r1.json", time.Now())
	if err != nil {
		t.Fatalf("attempt 1 failed: %v", err)
	}
	if att1.AttemptNumber != 1 {
		t.Errorf("expected attempt 1, got %d", att1.AttemptNumber)
	}

	// Canonical workflow transition: DISPATCHED -> FAILED -> READY
	if err := s.TransitionTask(ctx, "task-retry", domain.StateDispatched, domain.StateFailed); err != nil {
		t.Fatalf("transition DISPATCHED -> FAILED failed: %v", err)
	}
	if err := s.TransitionTask(ctx, "task-retry", domain.StateFailed, domain.StateReady); err != nil {
		t.Fatalf("transition FAILED -> READY failed: %v", err)
	}

	// Attempt 2: READY -> DISPATCHED
	att2, err := s.PrepareDispatch(ctx, "task-retry", "contract-retry", "att-r2", "r2.json", time.Now())
	if err != nil {
		t.Fatalf("attempt 2 failed: %v", err)
	}
	if att2.AttemptNumber != 2 {
		t.Errorf("expected attempt 2, got %d", att2.AttemptNumber)
	}

	tsk, err := s.GetTask(ctx, "task-retry")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if tsk.CurrentAttempt != 2 {
		t.Errorf("expected task current_attempt 2, got %d", tsk.CurrentAttempt)
	}
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
			att, err := s.PrepareDispatch(ctx, "task-race", "contract-race", attID, "r.json", time.Now())
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
	t.Logf("CONCURRENT_DISPATCH_SINGLE_WINNER = PASS: 1 winner, 1 conflict (%v)", results[0])

	// Verify task state is DISPATCHED and attempt is 1
	tsk, err := s.GetTask(ctx, "task-race")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if tsk.State != domain.StateDispatched || tsk.CurrentAttempt != 1 {
		t.Errorf("unexpected task state after race: state=%s, attempt=%d", tsk.State, tsk.CurrentAttempt)
	}
}

func TestStore_TaskAttempt_CrossTaskBindingFailsForeignKey(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()

	setupReadyTask(t, s, "task-X", "contract-X")
	setupReadyTask(t, s, "task-Y", "contract-Y")

	// Attempt to directly insert a task_attempt row binding task-X to contract-Y (which belongs to task-Y)
	query := `
INSERT INTO task_attempts (attempt_id, attempt_number, task_id, contract_id, expected_report_path, started_at)
VALUES ('att-cross', 1, 'task-X', 'contract-Y', 'r.json', ?)
`
	_, err := s.db.ExecContext(ctx, query, formatTime(time.Now()))
	if err == nil {
		t.Fatalf("expected composite foreign key check to block cross-task attempt binding, got nil")
	}
	t.Logf("CROSS_TASK_ATTEMPT_BINDING_REJECTED = PASS: %v", err)
}
