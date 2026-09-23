package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/trungqwe/ai-supervisor/internal/domain"
)

func setupBoundAttempt(t *testing.T, s *Store, taskID, contractID, attemptID string) (string, domain.TaskAttempt) {
	t.Helper()
	ctx := context.Background()
	setupReadyTask(t, s, taskID, contractID)
	pairID := "pair-d-" + taskID
	session := domain.WorkerSession{
		PairID:             pairID,
		SessionID:          "session-" + taskID,
		RuntimeType:        "agy_tui",
		WorkerAgentID:      "agy",
		Status:             domain.WorkerSessionIdle,
		TerminalGeneration: "generation-" + taskID,
		QuarantineState:    domain.QuarantineClean,
	}
	if err := s.CreateWorkerSession(ctx, session); err != nil {
		t.Fatalf("CreateWorkerSession: %v", err)
	}
	reportPath, err := CanonicalExpectedReportPath(taskID, attemptID)
	if err != nil {
		t.Fatalf("CanonicalExpectedReportPath: %v", err)
	}
	attempt, err := s.PrepareBoundDispatch(ctx, taskID, contractID, attemptID, reportPath, time.Now().UTC(), DispatchBinding{
		OperationID:        "dispatch-" + attemptID,
		SessionID:          session.SessionID,
		TerminalGeneration: session.TerminalGeneration,
	})
	if err != nil {
		t.Fatalf("PrepareBoundDispatch: %v", err)
	}
	return pairID, attempt
}

func TestStore_WorkerSessionCRUDAndCAS(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()
	setupReadyTask(t, s, "task-session", "contract-session")
	pairID := "pair-d-task-session"
	session := domain.WorkerSession{
		PairID:             pairID,
		SessionID:          "session-crud",
		RuntimeType:        "agy_tui",
		WorkerAgentID:      "agy",
		Status:             domain.WorkerSessionIdle,
		TerminalGeneration: "generation-1",
	}
	if err := s.CreateWorkerSession(ctx, session); err != nil {
		t.Fatalf("CreateWorkerSession: %v", err)
	}
	byPair, err := s.GetWorkerSessionByPair(ctx, pairID)
	if err != nil {
		t.Fatalf("GetWorkerSessionByPair: %v", err)
	}
	if byPair.WorktreePath != nil || byPair.QuarantineState != domain.QuarantineClean {
		t.Fatalf("nullable/default round trip mismatch: %+v", byPair)
	}
	if err := s.CreateProject(ctx, domain.Project{ProjectID: "project-session-2", Name: "Project 2", RootPath: "/session-2"}); err != nil {
		t.Fatalf("create second project: %v", err)
	}
	if err := s.CreatePair(ctx, domain.Pair{PairID: "pair-session-2", ProjectID: "project-session-2", CurrentPhaseID: "P03", State: "ACTIVE"}); err != nil {
		t.Fatalf("create second pair: %v", err)
	}
	duplicateSession := session
	duplicateSession.PairID = "pair-session-2"
	if err := s.CreateWorkerSession(ctx, duplicateSession); !errors.Is(err, ErrDuplicateKey) {
		t.Fatalf("duplicate session_id error = %v, want ErrDuplicateKey", err)
	}
	byID, err := s.GetWorkerSessionByID(ctx, "session-crud")
	if err != nil || byID.PairID != pairID {
		t.Fatalf("GetWorkerSessionByID: session=%+v err=%v", byID, err)
	}
	newGeneration := "generation-2"
	err = s.UpdateWorkerSessionStatusAndQuarantine(ctx, pairID, WorkerSessionCASUpdate{
		ExpectedStatus:          domain.WorkerSessionIdle,
		ExpectedQuarantineState: domain.QuarantineClean,
		Status:                  domain.WorkerSessionActive,
		QuarantineState:         domain.QuarantineQuarantined,
		TerminalGeneration:      &newGeneration,
	})
	if err != nil {
		t.Fatalf("UpdateWorkerSessionStatusAndQuarantine: %v", err)
	}
	updated, _ := s.GetWorkerSessionByPair(ctx, pairID)
	if updated.Status != domain.WorkerSessionActive || updated.QuarantineState != domain.QuarantineQuarantined || updated.TerminalGeneration != newGeneration {
		t.Fatalf("worker session CAS update mismatch: %+v", updated)
	}
	err = s.UpdateWorkerSessionStatusAndQuarantine(ctx, pairID, WorkerSessionCASUpdate{
		ExpectedStatus:          domain.WorkerSessionIdle,
		ExpectedQuarantineState: domain.QuarantineClean,
		Status:                  domain.WorkerSessionTerminated,
		QuarantineState:         domain.QuarantineClean,
	})
	if !errors.Is(err, ErrStateConflict) {
		t.Fatalf("stale worker session CAS error = %v, want ErrStateConflict", err)
	}
}

func TestStore_LifecycleOperationCRUDAndConstraints(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()
	pairID, attempt := setupBoundAttempt(t, s, "task-ops", "contract-ops", "attempt-ops")

	provision := domain.PairProvisioningOperation{
		OperationID: "provision-1",
		PairID:      pairID,
		Stage:       domain.ProvisionRequested,
		ClientToken: "client-token-1",
	}
	if err := s.CreatePairProvisioningOperation(ctx, provision); err != nil {
		t.Fatalf("CreatePairProvisioningOperation: %v", err)
	}
	duplicate := provision
	duplicate.OperationID = "provision-2"
	duplicate.Stage = domain.ProvisionFailed
	if err := s.CreatePairProvisioningOperation(ctx, duplicate); !errors.Is(err, ErrDuplicateKey) {
		t.Fatalf("unresolved provisioning uniqueness error = %v, want ErrDuplicateKey", err)
	}
	completedAt := time.Now().UTC()
	sessionID := "session-task-ops"
	if err := s.UpdatePairProvisioningOperationStage(ctx, provision.OperationID, domain.ProvisionRequested, domain.ProvisionConfirmed, PairProvisioningStageUpdate{
		SessionID:   &sessionID,
		CompletedAt: &completedAt,
	}); err != nil {
		t.Fatalf("UpdatePairProvisioningOperationStage: %v", err)
	}
	gotProvision, err := s.GetPairProvisioningOperation(ctx, provision.OperationID)
	if err != nil || gotProvision.Stage != domain.ProvisionConfirmed || gotProvision.SessionID == nil || *gotProvision.SessionID != sessionID {
		t.Fatalf("provision operation round trip: operation=%+v err=%v", gotProvision, err)
	}

	gotDispatch, err := s.GetDispatchOperation(ctx, "dispatch-attempt-ops")
	if err != nil || gotDispatch.Stage != domain.DispatchBound || gotDispatch.AttemptID != attempt.AttemptID {
		t.Fatalf("bound dispatch round trip: operation=%+v err=%v", gotDispatch, err)
	}
	if err := s.UpdateDispatchOperationStage(ctx, gotDispatch.OperationID, domain.DispatchBound, domain.SendRequested, DispatchStageUpdate{}); err != nil {
		t.Fatalf("dispatch DISPATCH_BOUND -> SEND_REQUESTED: %v", err)
	}
	confirmedAt := time.Now().UTC()
	if err := s.UpdateDispatchOperationStage(ctx, gotDispatch.OperationID, domain.SendRequested, domain.SendConfirmed, DispatchStageUpdate{ConfirmedAt: &confirmedAt}); err != nil {
		t.Fatalf("dispatch SEND_REQUESTED -> SEND_CONFIRMED: %v", err)
	}
	if err := s.CreateDispatchOperation(ctx, domain.DispatchOperation{
		OperationID:        "dispatch-duplicate-attempt",
		AttemptID:          attempt.AttemptID,
		PairID:             pairID,
		TaskID:             attempt.TaskID,
		SessionID:          *attempt.SessionID,
		TerminalGeneration: *attempt.TerminalGeneration,
		Stage:              domain.DispatchBound,
	}); !errors.Is(err, ErrDuplicateKey) {
		t.Fatalf("duplicate dispatch attempt error = %v, want ErrDuplicateKey", err)
	}

	taskID, contractID, attemptID := attempt.TaskID, attempt.ContractID, attempt.AttemptID
	stop := domain.StopOperation{
		OperationID:        "stop-1",
		Purpose:            domain.RunningAttemptStop,
		PairID:             pairID,
		TaskID:             &taskID,
		ContractID:         &contractID,
		AttemptID:          &attemptID,
		SessionID:          *attempt.SessionID,
		TerminalGeneration: *attempt.TerminalGeneration,
		Stage:              domain.StopRequested,
		Actor:              "supervisor",
	}
	if err := s.CreateStopOperation(ctx, stop); err != nil {
		t.Fatalf("CreateStopOperation: %v", err)
	}
	callCompletedAt := time.Now().UTC()
	deadline := callCompletedAt.Add(time.Minute)
	resolution := domain.StopResolutionInFlight
	if err := s.UpdateStopOperationStage(ctx, stop.OperationID, domain.StopRequested, domain.StopCallSucceeded, StopStageUpdate{
		CallCompletedAt:        &callCompletedAt,
		ConfirmationDeadlineAt: &deadline,
		ResolutionState:        &resolution,
	}); err != nil {
		t.Fatalf("UpdateStopOperationStage: %v", err)
	}
	gotStop, err := s.GetStopOperation(ctx, stop.OperationID)
	if err != nil || gotStop.Stage != domain.StopCallSucceeded || gotStop.ConfirmationDeadlineAt == nil || !gotStop.ConfirmationDeadlineAt.Equal(deadline) {
		t.Fatalf("stop operation round trip: operation=%+v err=%v", gotStop, err)
	}

	for _, statement := range []string{
		"DELETE FROM task_attempts WHERE attempt_id = 'attempt-ops'",
		"DELETE FROM tasks WHERE task_id = 'task-ops'",
		"DELETE FROM pairs WHERE pair_id = '" + pairID + "'",
	} {
		if _, err := s.db.ExecContext(ctx, statement); err == nil {
			t.Fatalf("ON DELETE RESTRICT did not reject %q", statement)
		}
	}
	if inUse := s.db.Stats().InUse; inUse != 0 {
		t.Fatalf("database connections still in use after lifecycle CRUD: %d", inUse)
	}
}

func TestStore_TaskAttemptNullableAndBoundSnapshotRoundTrip(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()

	setupReadyTask(t, s, "task-null-snapshot", "contract-null-snapshot")
	path, _ := CanonicalExpectedReportPath("task-null-snapshot", "attempt-null-snapshot")
	if _, err := s.PrepareDispatch(ctx, "task-null-snapshot", "contract-null-snapshot", "attempt-null-snapshot", path, time.Now().UTC()); err != nil {
		t.Fatalf("PrepareDispatch nullable snapshot: %v", err)
	}
	nullable, err := s.GetTaskAttempt(ctx, "attempt-null-snapshot")
	if err != nil || nullable.SessionID != nil || nullable.TerminalGeneration != nil || nullable.RecoveryDisposition != nil || nullable.QuarantineState != domain.QuarantineClean {
		t.Fatalf("nullable snapshot round trip: attempt=%+v err=%v", nullable, err)
	}

	_, bound := setupBoundAttempt(t, s, "task-bound-snapshot", "contract-bound-snapshot", "attempt-bound-snapshot")
	got, err := s.GetTaskAttempt(ctx, bound.AttemptID)
	if err != nil || got.SessionID == nil || got.TerminalGeneration == nil || *got.SessionID != *bound.SessionID || *got.TerminalGeneration != *bound.TerminalGeneration {
		t.Fatalf("bound snapshot round trip: attempt=%+v err=%v", got, err)
	}
}

func TestStore_StopOperationRejectsCrossTaskLineage(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()
	pairA, attemptA := setupBoundAttempt(t, s, "task-lineage-a", "contract-lineage-a", "attempt-lineage-a")
	_, attemptB := setupBoundAttempt(t, s, "task-lineage-b", "contract-lineage-b", "attempt-lineage-b")
	taskID, contractID, attemptID := attemptA.TaskID, attemptA.ContractID, attemptB.AttemptID
	err := s.CreateStopOperation(ctx, domain.StopOperation{
		OperationID:        "stop-cross-lineage",
		Purpose:            domain.RunningAttemptStop,
		PairID:             pairA,
		TaskID:             &taskID,
		ContractID:         &contractID,
		AttemptID:          &attemptID,
		SessionID:          *attemptA.SessionID,
		TerminalGeneration: *attemptA.TerminalGeneration,
		Stage:              domain.StopRequested,
		Actor:              "supervisor",
	})
	if !errors.Is(err, ErrAttemptLineageMismatch) {
		t.Fatalf("cross-task stop lineage error = %v, want ErrAttemptLineageMismatch", err)
	}
	var count int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM stop_operations WHERE operation_id = 'stop-cross-lineage'").Scan(&count); err != nil || count != 0 {
		t.Fatalf("cross-lineage stop operation persisted: count=%d err=%v", count, err)
	}
}

func TestStore_PrepareBoundDispatchAuditFailureRollsBackBinding(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()
	setupReadyTask(t, s, "task-bound-rollback", "contract-bound-rollback")
	pairID := "pair-d-task-bound-rollback"
	if err := s.CreateWorkerSession(ctx, domain.WorkerSession{
		PairID:             pairID,
		SessionID:          "session-bound-rollback",
		RuntimeType:        "agy_tui",
		WorkerAgentID:      "agy",
		Status:             domain.WorkerSessionIdle,
		TerminalGeneration: "generation-bound-rollback",
		QuarantineState:    domain.QuarantineClean,
	}); err != nil {
		t.Fatalf("CreateWorkerSession: %v", err)
	}
	if _, err := s.db.ExecContext(ctx, `
CREATE TRIGGER reject_dispatch_bound_audit
BEFORE INSERT ON audit_events
BEGIN
    SELECT RAISE(ABORT, 'forced dispatch audit failure');
END;
`); err != nil {
		t.Fatalf("create audit rejection trigger: %v", err)
	}
	reportPath, _ := CanonicalExpectedReportPath("task-bound-rollback", "attempt-bound-rollback")
	_, err := s.PrepareBoundDispatch(ctx, "task-bound-rollback", "contract-bound-rollback", "attempt-bound-rollback", reportPath, time.Now().UTC(), DispatchBinding{
		OperationID:        "dispatch-bound-rollback",
		SessionID:          "session-bound-rollback",
		TerminalGeneration: "generation-bound-rollback",
	})
	if err == nil {
		t.Fatal("PrepareBoundDispatch unexpectedly succeeded with rejected audit insert")
	}
	task, _ := s.GetTask(ctx, "task-bound-rollback")
	contract, _ := s.GetTaskContract(ctx, "contract-bound-rollback")
	if task.State != domain.StateReady || task.CurrentAttempt != 0 || contract.IsImmutable {
		t.Fatalf("dispatch audit failure did not roll back task/contract: task=%+v contract=%+v", task, contract)
	}
	for table, key := range map[string][2]string{
		"task_attempts":       {"attempt_id", "attempt-bound-rollback"},
		"dispatch_operations": {"operation_id", "dispatch-bound-rollback"},
	} {
		var count int
		query := "SELECT COUNT(*) FROM " + table + " WHERE " + key[0] + " = ?"
		if err := s.db.QueryRowContext(ctx, query, key[1]).Scan(&count); err != nil || count != 0 {
			t.Fatalf("%s row survived dispatch rollback: count=%d err=%v key=%s", table, count, err, key[1])
		}
	}
}
