package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/trungqwe/ai-supervisor/internal/domain"
)

func TestStore_RestartRecovery_OpenAttempt(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()

	setupReadyTask(t, s, "task-rec-open", "contract-rec-open")

	// Dispatch task-rec-open -> state is DISPATCHED with attempt 1 and ended_at is NULL
	p, err := CanonicalExpectedReportPath("task-rec-open", "att-rec-open")
	if err != nil {
		t.Fatalf("CanonicalExpectedReportPath failed: %v", err)
	}

	_, err = prepareLegacyDispatchForTest(s, ctx, "task-rec-open", "contract-rec-open", "att-rec-open", p, time.Now())
	if err != nil {
		t.Fatalf("PrepareDispatch failed: %v", err)
	}

	// Classify recovery candidates on simulated restart
	candidates, err := s.ClassifyRestartRecovery(ctx)
	if err != nil {
		t.Fatalf("ClassifyRestartRecovery failed: %v", err)
	}

	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}
	c := candidates[0]
	if c.TaskID != "task-rec-open" || c.AttemptID != "att-rec-open" || c.AttemptNumber != 1 || c.ContractID != "contract-rec-open" {
		t.Errorf("unexpected candidate fields: %+v", c)
	}
	if c.Classification != ClassificationExternalReconciliationRequired {
		t.Errorf("expected EXTERNAL_RECONCILIATION_REQUIRED, got %s", c.Classification)
	}
	t.Logf("RECOVERY_NULL_ENDED_AT = EXTERNAL_RECONCILIATION_REQUIRED (%+v)", c)
}

func TestStore_RestartRecovery_InconsistentNoAttempt(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()

	// Finding R2-002: Use raw package-internal SQL fixture to simulate corrupted DB state, NOT production CreateTask
	proj := domain.Project{ProjectID: "p-raw-1", Name: "Raw Proj 1", RootPath: "/raw1"}
	s.CreateProject(ctx, proj)
	pair := domain.Pair{PairID: "pair-raw-1", ProjectID: "p-raw-1", CurrentPhaseID: "P02", State: "ACTIVE"}
	s.CreatePair(ctx, pair)

	now := formatTime(time.Now().UTC())
	rawTaskInsert := `
INSERT INTO tasks (task_id, phase_id, pair_id, state, current_attempt, created_at, updated_at)
VALUES ('task-corrupt-noatt', 'P02', 'pair-raw-1', 'DISPATCHED', 1, ?, ?)
`
	if _, err := s.db.ExecContext(ctx, rawTaskInsert, now, now); err != nil {
		t.Fatalf("failed to insert raw corrupted task: %v", err)
	}

	// Classify recovery candidates
	candidates, err := s.ClassifyRestartRecovery(ctx)
	if err == nil || !errors.Is(err, ErrInconsistentPersistedState) {
		t.Fatalf("expected ErrInconsistentPersistedState, got %v", err)
	}

	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}
	if candidates[0].Classification != ClassificationInconsistentPersistedState {
		t.Errorf("expected INCONSISTENT_PERSISTED_STATE, got %s", candidates[0].Classification)
	}
	t.Logf("RECOVERY_NO_ATTEMPT = INCONSISTENT_PERSISTED_STATE (%+v)", candidates[0])
}

func TestStore_RestartRecovery_InconsistentEndedAttempt(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()

	setupReadyTask(t, s, "task-rec-ended", "contract-rec-ended")

	p, _ := CanonicalExpectedReportPath("task-rec-ended", "att-rec-ended")
	_, err := prepareLegacyDispatchForTest(s, ctx, "task-rec-ended", "contract-rec-ended", "att-rec-ended", p, time.Now())
	if err != nil {
		t.Fatalf("PrepareDispatch failed: %v", err)
	}

	// Finding R2-007 / R3-003: Simulate corrupt state where attempt has valid timestamp ended_at set, but task is still in DISPATCHED state
	endedTime := formatTime(time.Now().UTC())
	_, err = s.db.ExecContext(ctx, "UPDATE task_attempts SET ended_at = ? WHERE attempt_id = 'att-rec-ended'", endedTime)
	if err != nil {
		t.Fatalf("failed to set ended_at on attempt: %v", err)
	}

	// Classify recovery candidates
	candidates, err := s.ClassifyRestartRecovery(ctx)
	if err == nil || !errors.Is(err, ErrInconsistentPersistedState) {
		t.Fatalf("expected ErrInconsistentPersistedState for ended attempt with DISPATCHED task, got %v", err)
	}

	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}
	c := candidates[0]
	if c.Classification != ClassificationInconsistentPersistedState {
		t.Errorf("expected INCONSISTENT_PERSISTED_STATE for ended attempt, got %s", c.Classification)
	}
	t.Logf("RECOVERY_TIMESTAMP_ENDED_AT = INCONSISTENT_PERSISTED_STATE (%+v)", c)
}

func TestStore_RestartRecovery_InconsistentEmptyEndedAttempt(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()

	setupReadyTask(t, s, "task-rec-empty", "contract-rec-empty")

	p, _ := CanonicalExpectedReportPath("task-rec-empty", "att-rec-empty")
	_, err := prepareLegacyDispatchForTest(s, ctx, "task-rec-empty", "contract-rec-empty", "att-rec-empty", p, time.Now())
	if err != nil {
		t.Fatalf("PrepareDispatch failed: %v", err)
	}

	// Finding R3-003: Simulate corrupt state where attempt has empty string ended_at = '', but task is still in DISPATCHED state
	_, err = s.db.ExecContext(ctx, "UPDATE task_attempts SET ended_at = '' WHERE attempt_id = 'att-rec-empty'")
	if err != nil {
		t.Fatalf("failed to set empty ended_at on attempt: %v", err)
	}

	// Classify recovery candidates
	candidates, err := s.ClassifyRestartRecovery(ctx)
	if err == nil || !errors.Is(err, ErrInconsistentPersistedState) {
		t.Fatalf("expected ErrInconsistentPersistedState for empty string ended_at with DISPATCHED task, got %v", err)
	}

	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}
	c := candidates[0]
	if c.Classification != ClassificationInconsistentPersistedState {
		t.Errorf("expected INCONSISTENT_PERSISTED_STATE for empty string ended_at, got %s", c.Classification)
	}
	t.Logf("RECOVERY_EMPTY_STRING_ENDED_AT = INCONSISTENT_PERSISTED_STATE (%+v)", c)
}
