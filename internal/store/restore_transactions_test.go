package store

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/trungqwe/ai-supervisor/internal/domain"
)

func TestRestoreReservationConfirmationAndPairGuard(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()
	setupReadyTask(t, s, "task-restore-tx", "contract-restore-tx")
	seedWorkerSessionForTest(t, s, domain.WorkerSession{PairID: "pair-d-task-restore-tx", SessionID: "session-restore-tx", RuntimeType: "agy_tui", WorkerAgentID: "agy", Status: domain.WorkerSessionTerminated, TerminalGeneration: "generation-old", QuarantineState: domain.QuarantineClean})
	request := domain.RestoreReservation{AuthorizationID: "auth-restore-tx", OperationID: "restore-tx", PairID: "pair-d-task-restore-tx", SessionID: "session-restore-tx", ExpectedGeneration: "generation-old", RiskScope: "POSSIBLE_PROMPT_REPLAY", AuthorizedPrincipal: "verified-subject", Actor: "supervisor", At: time.Now().UTC()}
	if err := s.ReservePairRestore(ctx, request); err != nil {
		t.Fatalf("reserve restore: %v", err)
	}
	var consumedBy string
	if err := s.db.QueryRowContext(ctx, `SELECT consumed_by_operation_id FROM restore_authorizations WHERE authorization_id=?`, request.AuthorizationID).Scan(&consumedBy); err != nil || consumedBy != request.OperationID {
		t.Fatalf("one-shot authorization was not consumed by exact operation: consumed_by=%q err=%v", consumedBy, err)
	}
	if err := s.ReservePairRestore(ctx, request); err == nil {
		t.Fatal("consumed restore authorization was replayed")
	}
	if err := s.UpdateWorkerSessionStatusAndQuarantine(ctx, request.PairID, WorkerSessionCASUpdate{ExpectedStatus: domain.WorkerSessionTerminated, ExpectedQuarantineState: domain.QuarantineQuarantined, Status: domain.WorkerSessionTerminated, QuarantineState: domain.QuarantineQuarantined}); !errors.Is(err, ErrQuarantinedExecution) {
		t.Fatalf("unresolved restore allowed existing Store mutation: %v", err)
	}
	if _, err := s.db.ExecContext(ctx, `INSERT INTO stop_operations(operation_id,purpose,pair_id,session_id,terminal_generation,stage,actor,requested_at,resolution_state) VALUES('legacy-unlinked-stop','PAIR_MAINTENANCE',?,'session-restore-tx','generation-old','STOP_REQUESTED','legacy','2026-09-23T00:00:00Z','IN_FLIGHT')`, request.PairID); err != nil {
		t.Fatal(err)
	}
	callFailedResolution := domain.StopResolutionCallFailed
	if err := s.UpdateStopOperationStage(ctx, "legacy-unlinked-stop", domain.StopRequested, domain.StopCallFailed, StopStageUpdate{ResolutionState: &callFailedResolution}); !errors.Is(err, ErrQuarantinedExecution) {
		t.Fatalf("unlinked stop mutation bypassed unresolved restore guard: %v", err)
	}
	if err := s.RecordPairRestoreUnknown(ctx, request.OperationID, "supervisor"); err != nil {
		t.Fatalf("mark unknown: %v", err)
	}
	if err := s.ConfirmPairRestore(ctx, request.OperationID, request.SessionID, request.ExpectedGeneration, "generation-new", "native", domain.WorkerSessionIdle, "supervisor", time.Now().UTC()); !errors.Is(err, ErrStateConflict) {
		t.Fatalf("stale confirmation after unknown=%v", err)
	}
	if err := s.ResolveAmbiguousRestore(ctx, request.OperationID, request.PairID, request.SessionID, request.ExpectedGeneration, domain.AdministrativeRiskResolution, "operator accepted unresolved execution risk", "verified-subject", "supervisor", time.Now().UTC()); err != nil {
		t.Fatalf("authorized risk resolution: %v", err)
	}
	got, err := s.GetWorkerSessionByPair(ctx, request.PairID)
	if err != nil || got.TerminalGeneration != "generation-old" || got.QuarantineState != domain.QuarantineQuarantined {
		t.Fatalf("unknown restore must retain old quarantined binding: %+v, %v", got, err)
	}
	replay := request
	replay.AuthorizationID = "auth-restore-after-resolution"
	replay.OperationID = "restore-after-resolution"
	if err := s.ReservePairRestore(ctx, replay); !errors.Is(err, ErrQuarantinedExecution) {
		t.Fatalf("quarantined Pair accepted another restore: %v", err)
	}
}

func TestConcurrentRestoreReservationsProduceOnePairOwner(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()
	setupReadyTask(t, s, "task-restore-race", "contract-restore-race")
	seedWorkerSessionForTest(t, s, domain.WorkerSession{PairID: "pair-d-task-restore-race", SessionID: "session-restore-race", RuntimeType: "agy_tui", WorkerAgentID: "agy", Status: domain.WorkerSessionTerminated, TerminalGeneration: "g1", QuarantineState: domain.QuarantineClean})
	requests := []domain.RestoreReservation{{AuthorizationID: "auth-race-a", OperationID: "restore-race-a", PairID: "pair-d-task-restore-race", SessionID: "session-restore-race", ExpectedGeneration: "g1", RiskScope: "POSSIBLE_PROMPT_REPLAY", AuthorizedPrincipal: "principal-a", Actor: "supervisor"}, {AuthorizationID: "auth-race-b", OperationID: "restore-race-b", PairID: "pair-d-task-restore-race", SessionID: "session-restore-race", ExpectedGeneration: "g1", RiskScope: "POSSIBLE_PROMPT_REPLAY", AuthorizedPrincipal: "principal-b", Actor: "supervisor"}}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, request := range requests {
		request := request
		wg.Add(1)
		go func() { defer wg.Done(); results <- s.ReservePairRestore(ctx, request) }()
	}
	wg.Wait()
	close(results)
	success := 0
	for err := range results {
		if err == nil {
			success++
		}
	}
	if success != 1 {
		t.Fatalf("Pair restore reservation winners=%d, want 1", success)
	}
	var n int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM pair_restore_operations WHERE pair_id=? AND resolution_state<>'RESTORE_RESOLVED'`, requests[0].PairID).Scan(&n); err != nil || n != 1 {
		t.Fatalf("unresolved owner count=%d err=%v", n, err)
	}
}

func TestRestoreAdmissionRejectsOpenOrUnresolvedLineageAndAllowsCleanHistory(t *testing.T) {
	ctx := context.Background()
	t.Run("open attempt", func(t *testing.T) {
		s, _ := createTestStore(t)
		defer s.Close()
		pairID, attempt := setupBoundAttempt(t, s, "task-restore-open", "contract-restore-open", "attempt-restore-open")
		req := domain.RestoreReservation{AuthorizationID: "auth-open", OperationID: "restore-open", PairID: pairID, SessionID: *attempt.SessionID, ExpectedGeneration: *attempt.TerminalGeneration, RiskScope: "POSSIBLE_PROMPT_REPLAY", AuthorizedPrincipal: "principal", Actor: "supervisor"}
		if err := s.ReservePairRestore(ctx, req); !errors.Is(err, ErrQuarantinedExecution) {
			t.Fatalf("open attempt did not block restore: %v", err)
		}
	})
	t.Run("inconsistent current attempt pointer", func(t *testing.T) {
		s, _ := createTestStore(t)
		defer s.Close()
		setupReadyTask(t, s, "task-restore-pointer", "contract-restore-pointer")
		pairID := "pair-d-task-restore-pointer"
		seedWorkerSessionForTest(t, s, domain.WorkerSession{PairID: pairID, SessionID: "session-pointer", RuntimeType: "agy_tui", WorkerAgentID: "agy", Status: domain.WorkerSessionIdle, TerminalGeneration: "g1", QuarantineState: domain.QuarantineClean})
		if _, err := s.db.ExecContext(ctx, `UPDATE tasks SET current_attempt=1 WHERE task_id='task-restore-pointer'`); err != nil {
			t.Fatal(err)
		}
		req := domain.RestoreReservation{AuthorizationID: "auth-pointer", OperationID: "restore-pointer", PairID: pairID, SessionID: "session-pointer", ExpectedGeneration: "g1", RiskScope: "POSSIBLE_PROMPT_REPLAY", AuthorizedPrincipal: "principal", Actor: "supervisor"}
		if err := s.ReservePairRestore(ctx, req); !errors.Is(err, ErrQuarantinedExecution) {
			t.Fatalf("inconsistent current_attempt did not block restore: %v", err)
		}
	})
	t.Run("closed SEND_REQUESTED remains unresolved", func(t *testing.T) {
		s, _ := createTestStore(t)
		defer s.Close()
		pairID, attempt := setupBoundAttempt(t, s, "task-restore-send-requested", "contract-restore-send-requested", "attempt-restore-send-requested")
		opID := "dispatch-attempt-restore-send-requested"
		if err := s.RecordSendRequested(ctx, opID, *attempt.SessionID, *attempt.TerminalGeneration, "idle", false, "supervisor", time.Now()); err != nil {
			t.Fatal(err)
		}
		if _, err := s.db.ExecContext(ctx, `UPDATE tasks SET state='FAILED' WHERE task_id=?; UPDATE task_attempts SET ended_at='2026-09-23T00:00:00Z' WHERE attempt_id=?`, attempt.TaskID, attempt.AttemptID); err != nil {
			t.Fatal(err)
		}
		req := domain.RestoreReservation{AuthorizationID: "auth-send-requested", OperationID: "restore-send-requested", PairID: pairID, SessionID: *attempt.SessionID, ExpectedGeneration: *attempt.TerminalGeneration, RiskScope: "POSSIBLE_PROMPT_REPLAY", AuthorizedPrincipal: "principal", Actor: "supervisor"}
		if err := s.ReservePairRestore(ctx, req); !errors.Is(err, ErrQuarantinedExecution) {
			t.Fatalf("closed unresolved SEND_REQUESTED did not block restore: %v", err)
		}
	})
	for _, confirmed := range []bool{false, true} {
		name, taskID, contractID, attemptID := "historical DISPATCH_BOUND", "task-restore-bound-history", "contract-restore-bound-history", "attempt-restore-bound-history"
		if confirmed {
			name, taskID, contractID, attemptID = "historical SEND_CONFIRMED", "task-restore-confirmed-history", "contract-restore-confirmed-history", "attempt-restore-confirmed-history"
		}
		t.Run(name, func(t *testing.T) {
			s, _ := createTestStore(t)
			defer s.Close()
			pairID, attempt := setupBoundAttempt(t, s, taskID, contractID, attemptID)
			opID := "dispatch-" + attemptID
			if confirmed {
				if err := s.RecordSendRequested(ctx, opID, *attempt.SessionID, *attempt.TerminalGeneration, "idle", false, "supervisor", time.Now()); err != nil {
					t.Fatal(err)
				}
				if err := s.RecordSendConfirmed(ctx, opID, "supervisor", true, time.Now()); err != nil {
					t.Fatal(err)
				}
			}
			if err := s.AtomicTerminalTransition(ctx, taskID, domain.StateDispatched, domain.StateFailed, "test closed historical attempt", attemptID, "ORDINARY_FAILURE"); err != nil {
				t.Fatal(err)
			}
			req := domain.RestoreReservation{AuthorizationID: "auth-" + attemptID, OperationID: "restore-" + attemptID, PairID: pairID, SessionID: *attempt.SessionID, ExpectedGeneration: *attempt.TerminalGeneration, RiskScope: "POSSIBLE_PROMPT_REPLAY", AuthorizedPrincipal: "principal", Actor: "supervisor"}
			if err := s.ReservePairRestore(ctx, req); err != nil {
				t.Fatalf("clean closed history permanently blocked restore: %v", err)
			}
		})
	}
}

func TestRestoreReservationAuditFailureRollsBackAllState(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()
	setupReadyTask(t, s, "task-restore-audit-fail", "contract-restore-audit-fail")
	seedWorkerSessionForTest(t, s, domain.WorkerSession{PairID: "pair-d-task-restore-audit-fail", SessionID: "session-restore-audit-fail", RuntimeType: "agy_tui", WorkerAgentID: "agy", Status: domain.WorkerSessionTerminated, TerminalGeneration: "g1", QuarantineState: domain.QuarantineClean})
	if _, err := s.db.ExecContext(ctx, `CREATE TRIGGER reject_restore_audit BEFORE INSERT ON audit_events WHEN NEW.event_type='PAIR_RESTORE_RISK_ACCEPTED' BEGIN SELECT RAISE(ABORT,'injected audit failure'); END`); err != nil {
		t.Fatal(err)
	}
	req := domain.RestoreReservation{AuthorizationID: "auth-restore-audit-fail", OperationID: "restore-audit-fail", PairID: "pair-d-task-restore-audit-fail", SessionID: "session-restore-audit-fail", ExpectedGeneration: "g1", RiskScope: "POSSIBLE_PROMPT_REPLAY", AuthorizedPrincipal: "verified-subject", Actor: "supervisor"}
	if err := s.ReservePairRestore(ctx, req); err == nil {
		t.Fatal("reservation unexpectedly survived audit failure")
	}
	var operations, authorizations int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM pair_restore_operations`).Scan(&operations); err != nil {
		t.Fatal(err)
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM restore_authorizations`).Scan(&authorizations); err != nil {
		t.Fatal(err)
	}
	session, err := s.GetWorkerSessionByPair(ctx, req.PairID)
	if err != nil || operations != 0 || authorizations != 0 || session.QuarantineState != domain.QuarantineClean {
		t.Fatalf("partial Tx A state: operations=%d auth=%d session=%+v err=%v", operations, authorizations, session, err)
	}
}

func TestRestoreConfirmedHTTP200ProvenanceAndNoAutoClearance(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()
	setupReadyTask(t, s, "task-restore-confirm", "contract-restore-confirm")
	seedWorkerSessionForTest(t, s, domain.WorkerSession{PairID: "pair-d-task-restore-confirm", SessionID: "session-restore-confirm", RuntimeType: "agy_tui", WorkerAgentID: "agy", Status: domain.WorkerSessionTerminated, TerminalGeneration: "old", QuarantineState: domain.QuarantineClean})
	req := domain.RestoreReservation{AuthorizationID: "auth-restore-confirm", OperationID: "restore-confirm", PairID: "pair-d-task-restore-confirm", SessionID: "session-restore-confirm", ExpectedGeneration: "old", RiskScope: "POSSIBLE_PROMPT_REPLAY", AuthorizedPrincipal: "verified-subject", Actor: "supervisor"}
	if err := s.ReservePairRestore(ctx, req); err != nil {
		t.Fatal(err)
	}
	if err := s.ConfirmPairRestore(ctx, req.OperationID, req.SessionID, "old", "new", "saved_prompt", domain.WorkerSessionIdle, "supervisor", time.Now().UTC()); err != nil {
		t.Fatalf("confirm restore: %v", err)
	}
	if err := s.ResolveConfirmedRestore(ctx, req.OperationID, req.SessionID, "new", "idle", "verified-subject", "supervisor", time.Now().UTC()); err != nil {
		t.Fatalf("resolve confirmed restore: %v", err)
	}
	session, err := s.GetWorkerSessionByPair(ctx, req.PairID)
	if err != nil || session.TerminalGeneration != "new" || session.QuarantineState != domain.QuarantineQuarantined {
		t.Fatalf("confirmation incorrectly cleared quarantine: %+v %v", session, err)
	}
	var stage, resolution, basis string
	var http200 sql.NullString
	if err := s.db.QueryRowContext(ctx, `SELECT stage,resolution_state,resolution_basis,http_200_at FROM pair_restore_operations WHERE operation_id=?`, req.OperationID).Scan(&stage, &resolution, &basis, &http200); err != nil {
		t.Fatal(err)
	}
	if stage != "RESTORE_CONFIRMED" || resolution != "RESTORE_RESOLVED" || basis != "RESTORE_HTTP_200_CONFIRMED" || !http200.Valid {
		t.Fatalf("restore provenance not durable: %s %s %s %+v", stage, resolution, basis, http200)
	}
}

func TestLinkedPairMaintenancePersistsRestoreLinkWithoutAttempt(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()
	setupReadyTask(t, s, "task-linked-maint", "contract-linked-maint")
	seedWorkerSessionForTest(t, s, domain.WorkerSession{PairID: "pair-d-task-linked-maint", SessionID: "session-linked-maint", RuntimeType: "agy_tui", WorkerAgentID: "agy", Status: domain.WorkerSessionTerminated, TerminalGeneration: "old", QuarantineState: domain.QuarantineClean})
	req := domain.RestoreReservation{AuthorizationID: "auth-linked-maint", OperationID: "restore-linked-maint", PairID: "pair-d-task-linked-maint", SessionID: "session-linked-maint", ExpectedGeneration: "old", RiskScope: "POSSIBLE_PROMPT_REPLAY", AuthorizedPrincipal: "verified-subject", Actor: "supervisor"}
	if err := s.ReservePairRestore(ctx, req); err != nil {
		t.Fatal(err)
	}
	if err := s.ConfirmPairRestore(ctx, req.OperationID, req.SessionID, "old", "new", "native", domain.WorkerSessionIdle, "supervisor", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if err := s.ClaimRestoreRecovery(ctx, req.OperationID, req.PairID, req.SessionID, "new", "verified-subject", "supervisor", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if err := s.ClaimRestoreCleanup(ctx, req.OperationID, req.PairID, req.SessionID, "new", "verified-subject", "supervisor", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	principal := "verified-subject"
	operation := domain.StopOperation{OperationID: "stop-linked-maint", Purpose: domain.PairMaintenance, PairID: req.PairID, SessionID: req.SessionID, TerminalGeneration: "new", Stage: domain.StopRequested, Actor: "supervisor", RestoreOperationID: &req.OperationID, RestorePrincipal: &principal}
	if err := s.CreateStopOperation(ctx, operation); err != nil {
		t.Fatalf("linked PairMaintenance: %v", err)
	}
	got, err := s.GetStopOperation(ctx, operation.OperationID)
	if err != nil || got.AttemptID != nil || got.RestoreOperationID == nil || *got.RestoreOperationID != req.OperationID {
		t.Fatalf("linked maintenance round trip: %+v %v", got, err)
	}
}

func TestLinkedOldGenerationCleanupRequiresExactClosedQuarantinedAttempt(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()
	pairID, attempt := setupBoundAttempt(t, s, "task-linked-cleanup", "contract-linked-cleanup", "attempt-linked-cleanup")
	if err := s.AtomicTerminalTransition(ctx, attempt.TaskID, domain.StateDispatched, domain.StateFailed, "prior failure", attempt.AttemptID, "ORDINARY_FAILURE"); err != nil {
		t.Fatal(err)
	}
	req := domain.RestoreReservation{AuthorizationID: "auth-linked-cleanup", OperationID: "restore-linked-cleanup", PairID: pairID, SessionID: *attempt.SessionID, ExpectedGeneration: *attempt.TerminalGeneration, RiskScope: "POSSIBLE_PROMPT_REPLAY", AuthorizedPrincipal: "verified-subject", Actor: "supervisor"}
	if err := s.ReservePairRestore(ctx, req); err != nil {
		t.Fatal(err)
	}
	if err := s.ConfirmPairRestore(ctx, req.OperationID, req.SessionID, req.ExpectedGeneration, "new-generation", "native", domain.WorkerSessionIdle, "supervisor", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if err := s.ClaimRestoreRecovery(ctx, req.OperationID, pairID, req.SessionID, req.ExpectedGeneration, "verified-subject", "supervisor", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if err := s.ClaimRestoreCleanup(ctx, req.OperationID, pairID, req.SessionID, req.ExpectedGeneration, "verified-subject", "supervisor", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	taskID, contractID, attemptID := attempt.TaskID, attempt.ContractID, attempt.AttemptID
	principal := "verified-subject"
	bad := domain.StopOperation{OperationID: "stop-wrong-generation", Purpose: domain.QuarantineCleanup, PairID: pairID, TaskID: &taskID, ContractID: &contractID, AttemptID: &attemptID, SessionID: req.SessionID, TerminalGeneration: "new-generation", Stage: domain.StopRequested, Actor: "supervisor", RestoreOperationID: &req.OperationID, RestorePrincipal: &principal}
	if err := s.CreateStopOperation(ctx, bad); !errors.Is(err, ErrAttemptLineageMismatch) {
		t.Fatalf("new generation was attached to old attempt: %v", err)
	}
	good := bad
	good.OperationID = "stop-exact-old-generation"
	good.TerminalGeneration = req.ExpectedGeneration
	if err := s.CreateStopOperation(ctx, good); err != nil {
		t.Fatalf("exact old-generation cleanup rejected: %v", err)
	}
	got, err := s.GetStopOperation(ctx, good.OperationID)
	if err != nil || got.RestoreOperationID == nil || *got.RestoreOperationID != req.OperationID || got.AttemptID == nil || *got.AttemptID != attemptID {
		t.Fatalf("linked cleanup lineage mismatch: %+v %v", got, err)
	}
	closed, err := s.GetTaskAttempt(ctx, attemptID)
	if err != nil || closed.EndedAt == nil || closed.TerminalGeneration == nil || *closed.TerminalGeneration != req.ExpectedGeneration || closed.QuarantineState != domain.QuarantineQuarantined {
		t.Fatalf("restore/stop mutated old attempt snapshot or quarantine: %+v %v", closed, err)
	}
}
