package store

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/trungqwe/ai-supervisor/internal/domain"
)

func TestStore_BoundDispatchGuardsRollback(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(*testing.T, *Store, string, string)
	}{
		{"session quarantined", func(t *testing.T, s *Store, pairID, taskID string) {
			_, err := s.db.ExecContext(context.Background(), `UPDATE worker_sessions SET quarantine_state = 'QUARANTINED' WHERE pair_id = ?`, pairID)
			if err != nil {
				t.Fatal(err)
			}
		}},
		{"prior attempt quarantined while pair clean", func(t *testing.T, s *Store, pairID, taskID string) {
			seedPriorQuarantinedAttempt(t, s, taskID)
		}},
		{"provision requested", func(t *testing.T, s *Store, pairID, taskID string) {
			seedUnresolvedProvisioning(t, s, pairID, domain.ProvisionRequested)
		}},
		{"provision failed", func(t *testing.T, s *Store, pairID, taskID string) {
			seedUnresolvedProvisioning(t, s, pairID, domain.ProvisionFailed)
		}},
		{"all guards", func(t *testing.T, s *Store, pairID, taskID string) {
			seedPriorQuarantinedAttempt(t, s, taskID)
			seedUnresolvedProvisioning(t, s, pairID, domain.ProvisionFailed)
			if _, err := s.db.ExecContext(context.Background(), `UPDATE worker_sessions SET quarantine_state = 'QUARANTINED' WHERE pair_id = ?`, pairID); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			s, _ := createTestStore(t)
			defer s.Close()
			const taskID, contractID = "task-guard", "contract-guard"
			setupReadyTask(t, s, taskID, contractID)
			pairID := "pair-d-" + taskID
			seedWorkerSessionForTest(t, s, domain.WorkerSession{
				PairID: pairID, SessionID: "session-guard", RuntimeType: "agy_tui", WorkerAgentID: "agy",
				Status: domain.WorkerSessionIdle, TerminalGeneration: "generation-guard", QuarantineState: domain.QuarantineClean,
			})
			tc.setup(t, s, pairID, taskID)
			before, err := s.GetTask(ctx, taskID)
			if err != nil {
				t.Fatal(err)
			}
			contract, err := s.GetTaskContract(ctx, contractID)
			if err != nil {
				t.Fatal(err)
			}
			path, err := CanonicalExpectedReportPath(taskID, "attempt-rejected")
			if err != nil {
				t.Fatal(err)
			}
			_, err = s.PrepareBoundDispatch(ctx, taskID, contractID, "attempt-rejected", path, time.Now().UTC(), DispatchBinding{
				OperationID: "dispatch-rejected", SessionID: "session-guard", TerminalGeneration: "generation-guard",
			})
			if !errors.Is(err, ErrQuarantinedExecution) {
				t.Fatalf("dispatch error = %v", err)
			}
			after, err := s.GetTask(ctx, taskID)
			if err != nil {
				t.Fatal(err)
			}
			contractAfter, err := s.GetTaskContract(ctx, contractID)
			if err != nil {
				t.Fatal(err)
			}
			if after.State != before.State || after.CurrentAttempt != before.CurrentAttempt || contractAfter.IsImmutable != contract.IsImmutable {
				t.Fatalf("dispatch guard did not roll back task/contract: before=%+v after=%+v contract=%+v contractAfter=%+v", before, after, contract, contractAfter)
			}
			for _, q := range []string{
				`SELECT COUNT(*) FROM task_attempts WHERE attempt_id = 'attempt-rejected'`,
				`SELECT COUNT(*) FROM dispatch_operations WHERE operation_id = 'dispatch-rejected'`,
				`SELECT COUNT(*) FROM audit_events WHERE attempt_id = 'attempt-rejected'`,
			} {
				var count int
				if err := s.db.QueryRowContext(ctx, q).Scan(&count); err != nil || count != 0 {
					t.Fatalf("rollback query %q: count=%d err=%v", q, count, err)
				}
			}
		})
	}
}

func seedPriorQuarantinedAttempt(t *testing.T, s *Store, taskID string) {
	t.Helper()
	ctx := context.Background()
	path, _ := CanonicalExpectedReportPath(taskID, "attempt-prior")
	if _, err := s.PrepareBoundDispatch(ctx, taskID, "contract-guard", "attempt-prior", path, time.Now().UTC(), DispatchBinding{
		OperationID: "dispatch-prior", SessionID: "session-guard", TerminalGeneration: "generation-guard",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE task_attempts SET ended_at = ?, quarantine_state = 'QUARANTINED' WHERE attempt_id = 'attempt-prior'`, formatTime(time.Now().UTC())); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE tasks SET state = 'READY' WHERE task_id = ?`, taskID); err != nil {
		t.Fatal(err)
	}
}

func seedUnresolvedProvisioning(t *testing.T, s *Store, pairID string, stage domain.PairProvisioningStage) {
	t.Helper()
	if _, err := s.db.ExecContext(context.Background(), `INSERT INTO pair_provisioning_operations(operation_id,pair_id,stage,client_token,requested_at) VALUES('provision-guard',?,?,?,?)`, pairID, string(stage), "token-guard", formatTime(timeNow())); err != nil {
		t.Fatal(err)
	}
}

func TestStore_StopLineageAndPurpose(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()
	pairID, attempt := setupBoundAttempt(t, s, "task-stop-lineage", "contract-stop-lineage", "attempt-stop-lineage")
	taskID, contractID, attemptID := attempt.TaskID, attempt.ContractID, attempt.AttemptID
	base := domain.StopOperation{
		Purpose: domain.RunningAttemptStop, PairID: pairID, TaskID: &taskID, ContractID: &contractID,
		AttemptID: &attemptID, SessionID: *attempt.SessionID, TerminalGeneration: *attempt.TerminalGeneration,
		Stage: domain.StopRequested, Actor: "supervisor",
	}
	for _, tc := range []struct {
		name   string
		mutate func(*domain.StopOperation)
	}{
		{"wrong session", func(o *domain.StopOperation) { o.SessionID = "another-session" }},
		{"wrong generation", func(o *domain.StopOperation) { o.TerminalGeneration = "another-generation" }},
		{"cross pair", func(o *domain.StopOperation) { o.PairID = "pair-d-task-other-stop" }},
		{"task only cross pair", func(o *domain.StopOperation) {
			o.Purpose = domain.PairMaintenance
			o.AttemptID = nil
			o.ContractID = nil
			o.PairID = "pair-d-task-other-stop"
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o := base
			o.OperationID = "stop-reject-" + tc.name
			tc.mutate(&o)
			if err := s.CreateStopOperation(ctx, o); !errors.Is(err, ErrAttemptLineageMismatch) {
				t.Fatalf("stop lineage error = %v", err)
			}
			var count int
			if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM stop_operations WHERE operation_id = ?`, o.OperationID).Scan(&count); err != nil || count != 0 {
				t.Fatalf("rejected stop persisted: count=%d err=%v", count, err)
			}
		})
	}
	base.OperationID = "stop-live-valid"
	if err := s.CreateStopOperation(ctx, base); err != nil {
		t.Fatalf("live stop: %v", err)
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE task_attempts SET ended_at = ? WHERE attempt_id = ?`, formatTime(time.Now().UTC()), attemptID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE tasks SET state = 'FAILED' WHERE task_id = ?`, taskID); err != nil {
		t.Fatal(err)
	}
	cleanup := base
	cleanup.OperationID = "stop-cleanup-closed"
	cleanup.Purpose = domain.QuarantineCleanup
	if err := s.CreateStopOperation(ctx, cleanup); err != nil {
		t.Fatalf("cleanup of closed attempt: %v", err)
	}
	maintenance := base
	maintenance.OperationID = "stop-maintenance-no-attempt"
	maintenance.Purpose = domain.PairMaintenance
	maintenance.TaskID, maintenance.ContractID, maintenance.AttemptID = nil, nil, nil
	if err := s.CreateStopOperation(ctx, maintenance); err != nil {
		t.Fatalf("pair maintenance: %v", err)
	}
}

func TestStore_StopDeadlineImmutableAndResolutionCAS(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()
	pairID, attempt := setupBoundAttempt(t, s, "task-stop-cas", "contract-stop-cas", "attempt-stop-cas")
	taskID, contractID, attemptID := attempt.TaskID, attempt.ContractID, attempt.AttemptID
	stop := domain.StopOperation{OperationID: "stop-cas", Purpose: domain.RunningAttemptStop,
		PairID: pairID, TaskID: &taskID, ContractID: &contractID, AttemptID: &attemptID,
		SessionID: *attempt.SessionID, TerminalGeneration: *attempt.TerminalGeneration,
		Stage: domain.StopRequested, Actor: "supervisor"}
	if err := s.CreateStopOperation(ctx, stop); err != nil {
		t.Fatal(err)
	}
	completed := time.Now().UTC().Truncate(time.Microsecond)
	deadline := completed.Add(time.Minute)
	first := StopStageUpdate{CallCompletedAt: &completed, ConfirmationDeadlineAt: &deadline}
	if err := s.UpdateStopOperationStage(ctx, stop.OperationID, domain.StopRequested, domain.StopCallSucceeded, first); err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateStopOperationStage(ctx, stop.OperationID, domain.StopCallSucceeded, domain.StopCallSucceeded, first); err != nil {
		t.Fatalf("identical replay: %v", err)
	}
	for _, tc := range []struct {
		name   string
		update StopStageUpdate
	}{
		{"call completed changed", StopStageUpdate{CallCompletedAt: ptrTime(completed.Add(time.Second))}},
		{"deadline extended", StopStageUpdate{ConfirmationDeadlineAt: ptrTime(deadline.Add(time.Second))}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := s.UpdateStopOperationStage(ctx, stop.OperationID, domain.StopCallSucceeded, domain.StopCallSucceeded, tc.update); !errors.Is(err, ErrStateConflict) {
				t.Fatalf("rewrite error = %v", err)
			}
			got, err := s.GetStopOperation(ctx, stop.OperationID)
			if err != nil || !got.CallCompletedAt.Equal(completed) || !got.ConfirmationDeadlineAt.Equal(deadline) {
				t.Fatalf("deadline mutated: %+v err=%v", got, err)
			}
		})
	}
	resolvedAt := completed.Add(time.Second)
	timeout := domain.StopResolutionConfirmationTimeout
	if err := s.UpdateStopOperationStage(ctx, stop.OperationID, domain.StopCallSucceeded, domain.StopCallSucceeded, StopStageUpdate{ResolvedAt: &resolvedAt, ResolutionState: &timeout}); err != nil {
		t.Fatal(err)
	}
	confirmed := domain.StopResolutionTerminationConfirmed
	if err := s.UpdateStopOperationStage(ctx, stop.OperationID, domain.StopCallSucceeded, domain.StopTerminationConfirmed, StopStageUpdate{ResolutionState: &confirmed}); !errors.Is(err, ErrStateConflict) {
		t.Fatalf("stale confirmation error = %v", err)
	}
	if err := s.UpdateStopOperationStage(ctx, stop.OperationID, domain.StopCallSucceeded, domain.StopCallSucceeded, StopStageUpdate{ResolutionState: ptrStopResolution(domain.StopResolutionInFlight)}); !errors.Is(err, ErrStateConflict) {
		t.Fatalf("stale IN_FLIGHT error = %v", err)
	}
	got, err := s.GetStopOperation(ctx, stop.OperationID)
	if err != nil || got.Stage != domain.StopCallSucceeded || got.ResolutionState != timeout || !got.ResolvedAt.Equal(resolvedAt) {
		t.Fatalf("timeout overwritten: %+v err=%v", got, err)
	}
}

func TestStore_StopResolutionCompetingWriters(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()
	pairID, attempt := setupBoundAttempt(t, s, "task-stop-race", "contract-stop-race", "attempt-stop-race")
	taskID, contractID, attemptID := attempt.TaskID, attempt.ContractID, attempt.AttemptID
	if err := s.CreateStopOperation(ctx, domain.StopOperation{OperationID: "stop-race", Purpose: domain.RunningAttemptStop,
		PairID: pairID, TaskID: &taskID, ContractID: &contractID, AttemptID: &attemptID,
		SessionID: *attempt.SessionID, TerminalGeneration: *attempt.TerminalGeneration,
		Stage: domain.StopRequested, Actor: "supervisor"}); err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateStopOperationStage(ctx, "stop-race", domain.StopRequested, domain.StopCallSucceeded, StopStageUpdate{}); err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for _, resolution := range []domain.StopResolutionState{domain.StopResolutionConfirmationTimeout, domain.StopResolutionGenerationMismatch} {
		wg.Add(1)
		go func(resolution domain.StopResolutionState) {
			defer wg.Done()
			<-start
			results <- s.UpdateStopOperationStage(ctx, "stop-race", domain.StopCallSucceeded, domain.StopCallSucceeded, StopStageUpdate{ResolutionState: &resolution})
		}(resolution)
	}
	close(start)
	wg.Wait()
	close(results)
	succeeded, conflicts := 0, 0
	for err := range results {
		if err == nil {
			succeeded++
		} else if errors.Is(err, ErrStateConflict) {
			conflicts++
		} else {
			t.Fatalf("unexpected writer error: %v", err)
		}
	}
	if succeeded != 1 || conflicts != 1 {
		t.Fatalf("writer outcomes: success=%d conflict=%d", succeeded, conflicts)
	}
	got, err := s.GetStopOperation(ctx, "stop-race")
	if err != nil || got.ResolutionState == domain.StopResolutionInFlight {
		t.Fatalf("winner result lost: %+v err=%v", got, err)
	}
}

func TestStopTerminationConfirmationAndAuditRollbackAtomically(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()
	pairID, attempt := setupBoundAttempt(t, s, "task-stop-confirm-audit", "contract-stop-confirm-audit", "attempt-stop-confirm-audit")
	taskID, contractID, attemptID := attempt.TaskID, attempt.ContractID, attempt.AttemptID
	stop := domain.StopOperation{OperationID: "stop-confirm-audit", Purpose: domain.RunningAttemptStop, PairID: pairID, TaskID: &taskID, ContractID: &contractID, AttemptID: &attemptID, SessionID: *attempt.SessionID, TerminalGeneration: *attempt.TerminalGeneration, Stage: domain.StopRequested, Actor: "supervisor"}
	if err := s.CreateStopOperation(ctx, stop); err != nil {
		t.Fatal(err)
	}
	callAt := time.Now().UTC().Truncate(time.Microsecond)
	deadline := callAt.Add(time.Minute)
	if err := s.UpdateStopOperationStage(ctx, stop.OperationID, domain.StopRequested, domain.StopCallSucceeded, StopStageUpdate{CallCompletedAt: &callAt, ConfirmationDeadlineAt: &deadline}); err != nil {
		t.Fatal(err)
	}
	confirmedAt := callAt.Add(time.Second)
	resolution := domain.StopResolutionTerminationConfirmed
	update := StopStageUpdate{ExpectedResolutionState: domain.StopResolutionInFlight, TerminationConfirmedAt: &confirmedAt, ResolvedAt: &confirmedAt, ResolutionState: &resolution, ObservedSessionID: stop.SessionID, ObservedGeneration: stop.TerminalGeneration, ObservedIsTerminated: true}
	if _, err := s.db.ExecContext(ctx, `CREATE TRIGGER reject_stop_confirmation_audit BEFORE INSERT ON audit_events WHEN NEW.event_type='STOP_OPERATION_CONFIRMED' BEGIN SELECT RAISE(ABORT,'injected stop confirmation audit failure'); END`); err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateStopOperationStage(ctx, stop.OperationID, domain.StopCallSucceeded, domain.StopTerminationConfirmed, update); err == nil {
		t.Fatal("stop confirmation survived audit rollback")
	}
	got, err := s.GetStopOperation(ctx, stop.OperationID)
	if err != nil || got.Stage != domain.StopCallSucceeded || got.ResolutionState != domain.StopResolutionInFlight || got.TerminationConfirmedAt != nil {
		t.Fatalf("partial stop confirmation state: %+v %v", got, err)
	}
}

func ptrTime(value time.Time) *time.Time                                             { return &value }
func ptrStopResolution(value domain.StopResolutionState) *domain.StopResolutionState { return &value }
