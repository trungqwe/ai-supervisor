package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/trungqwe/ai-supervisor/internal/domain"
)

func TestStore_RestartRecovery_ExternalReconciliationRequired(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()

	setupReadyTask(t, s, "task-rec-1", "contract-rec-1")

	// Dispatch task-rec-1 -> state is DISPATCHED with attempt 1
	_, err := s.PrepareDispatch(ctx, "task-rec-1", "contract-rec-1", "att-rec-1", "r.json", time.Now())
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
	if c.TaskID != "task-rec-1" || c.AttemptID != "att-rec-1" || c.AttemptNumber != 1 || c.ContractID != "contract-rec-1" {
		t.Errorf("unexpected candidate fields: %+v", c)
	}
	if c.Classification != ClassificationExternalReconciliationRequired {
		t.Errorf("expected EXTERNAL_RECONCILIATION_REQUIRED, got %s", c.Classification)
	}
	t.Logf("RESTART_RECOVERY_CLASSIFIER = PASS: candidate=%+v", c)
}

func TestStore_RestartRecovery_InconsistentPersistedState(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()

	// Manually insert a task in DISPATCHED state without inserting an attempt row
	proj := domain.Project{ProjectID: "p-corrupt", Name: "Corrupt", RootPath: "/corrupt"}
	s.CreateProject(ctx, proj)
	pair := domain.Pair{PairID: "pair-corrupt", ProjectID: "p-corrupt", CurrentPhaseID: "P02", State: "ACTIVE"}
	s.CreatePair(ctx, pair)

	corruptTask := domain.Task{
		TaskID:         "task-corrupt",
		PhaseID:        "P02",
		PairID:         "pair-corrupt",
		State:          domain.StateDispatched,
		CurrentAttempt: 1, // attempt 1 claimed, but no row in task_attempts!
	}
	if err := s.CreateTask(ctx, corruptTask); err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	// Classify recovery candidates
	candidates, err := s.ClassifyRestartRecovery(ctx)
	if err == nil {
		t.Fatalf("expected ErrInconsistentPersistedState, got nil")
	}
	if !errors.Is(err, ErrInconsistentPersistedState) {
		t.Errorf("expected ErrInconsistentPersistedState, got %v", err)
	}

	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}
	if candidates[0].Classification != ClassificationInconsistentPersistedState {
		t.Errorf("expected INCONSISTENT_PERSISTED_STATE, got %s", candidates[0].Classification)
	}
	t.Logf("INCONSISTENT_PERSISTED_STATE = PASS: error=%v, candidate=%+v", err, candidates[0])
}
