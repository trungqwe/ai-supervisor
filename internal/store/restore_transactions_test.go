package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/trungqwe/ai-supervisor/internal/domain"
)

func setRestoreSessionTerminated(t *testing.T, s *Store, pairID string) {
	t.Helper()
	if _, err := s.db.ExecContext(context.Background(), `UPDATE worker_sessions SET status='TERMINATED' WHERE pair_id=?`, pairID); err != nil {
		t.Fatal(err)
	}
}

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
		setRestoreSessionTerminated(t, s, pairID)
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
		setRestoreSessionTerminated(t, s, pairID)
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
		setRestoreSessionTerminated(t, s, pairID)
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
			setRestoreSessionTerminated(t, s, pairID)
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

func TestRestoreAdmissionRequiresTerminatedSessionStatus(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()
	setupReadyTask(t, s, "task-restore-not-terminated", "contract-restore-not-terminated")
	pairID := "pair-d-task-restore-not-terminated"
	seedWorkerSessionForTest(t, s, domain.WorkerSession{PairID: pairID, SessionID: "session-restore-not-terminated", RuntimeType: "agy_tui", WorkerAgentID: "agy", Status: domain.WorkerSessionIdle, TerminalGeneration: "g1", QuarantineState: domain.QuarantineClean})
	req := domain.RestoreReservation{AuthorizationID: "auth-not-terminated", OperationID: "restore-not-terminated", PairID: pairID, SessionID: "session-restore-not-terminated", ExpectedGeneration: "g1", RiskScope: "POSSIBLE_PROMPT_REPLAY", AuthorizedPrincipal: "verified-subject", Actor: "supervisor"}
	if err := s.ReservePairRestore(ctx, req); !errors.Is(err, ErrStateConflict) {
		t.Fatalf("nonterminated WorkerSession restore error=%v, want state conflict", err)
	}
	var intents, authorizations int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM pair_restore_operations WHERE operation_id=?`, req.OperationID).Scan(&intents); err != nil {
		t.Fatal(err)
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM restore_authorizations WHERE authorization_id=?`, req.AuthorizationID).Scan(&authorizations); err != nil {
		t.Fatal(err)
	}
	if intents != 0 || authorizations != 0 {
		t.Fatalf("failed admission persisted intent/auth: %d/%d", intents, authorizations)
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
	principal := "verified-subject"
	operation := domain.StopOperation{OperationID: "stop-linked-maint", Purpose: domain.PairMaintenance, PairID: req.PairID, SessionID: req.SessionID, TerminalGeneration: "new", Stage: domain.StopRequested, Actor: "supervisor", RestoreOperationID: &req.OperationID, RestorePrincipal: &principal}
	observation := RestoreCleanupObservation{SessionID: req.SessionID, TerminalGeneration: "new"}
	if err := s.ClaimRestoreCleanupWithStop(ctx, operation, principal, "supervisor", time.Now().UTC(), observation); err != nil {
		t.Fatalf("atomic claim + linked PairMaintenance: %v", err)
	}
	got, err := s.GetStopOperation(ctx, operation.OperationID)
	if err != nil || got.AttemptID != nil || got.RestoreOperationID == nil || *got.RestoreOperationID != req.OperationID {
		t.Fatalf("linked maintenance round trip: %+v %v", got, err)
	}
	stale := operation
	stale.OperationID = "stop-linked-maint-stale"
	if err := s.ClaimRestoreCleanupWithStop(ctx, stale, principal, "supervisor", time.Now().UTC(), observation); !errors.Is(err, ErrStateConflict) {
		t.Fatalf("stale linked cleanup writer error=%v, want state conflict", err)
	}
	if _, err := s.GetStopOperation(ctx, stale.OperationID); !errors.Is(err, ErrOperationNotFound) {
		t.Fatalf("stale cleanup writer left duplicate stop intent: %v", err)
	}
}

func TestLinkedOldGenerationCleanupRequiresExactClosedQuarantinedAttempt(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()
	pairID, attempt := setupBoundAttempt(t, s, "task-linked-cleanup", "contract-linked-cleanup", "attempt-linked-cleanup")
	setRestoreSessionTerminated(t, s, pairID)
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
	if err := s.ResolveAmbiguousRestore(ctx, req.OperationID, pairID, req.SessionID, req.ExpectedGeneration, domain.PhysicalExecutionResolution, "operator says process stopped", "verified-subject", "supervisor", time.Now().UTC()); err == nil {
		t.Fatal("Class A accepted operator assertion without any D11 stop evidence")
	}
	taskID, contractID, attemptID := attempt.TaskID, attempt.ContractID, attempt.AttemptID
	principal := "verified-subject"
	bad := domain.StopOperation{OperationID: "stop-wrong-generation", Purpose: domain.QuarantineCleanup, PairID: pairID, TaskID: &taskID, ContractID: &contractID, AttemptID: &attemptID, SessionID: req.SessionID, TerminalGeneration: "new-generation", Stage: domain.StopRequested, Actor: "supervisor", RestoreOperationID: &req.OperationID, RestorePrincipal: &principal}
	if err := s.ClaimRestoreCleanupWithStop(ctx, bad, principal, "supervisor", time.Now().UTC()); !errors.Is(err, ErrAttemptLineageMismatch) {
		t.Fatalf("new generation was attached to old attempt: %v", err)
	}
	good := bad
	good.OperationID = "stop-exact-old-generation"
	good.TerminalGeneration = req.ExpectedGeneration
	if err := s.ClaimRestoreCleanupWithStop(ctx, good, principal, "supervisor", time.Now().UTC()); err != nil {
		t.Fatalf("atomic exact old-generation cleanup rejected: %v", err)
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

func TestRestorePhysicalResolutionRequiresPositiveD11StopEvidence(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()
	pairID, attempt := setupBoundAttempt(t, s, "task-restore-physical", "contract-restore-physical", "attempt-restore-physical")
	if err := s.AtomicTerminalTransition(ctx, attempt.TaskID, domain.StateDispatched, domain.StateFailed, "prepare closed lineage", attempt.AttemptID, "ORDINARY_FAILURE"); err != nil {
		t.Fatal(err)
	}
	setRestoreSessionTerminated(t, s, pairID)
	req := domain.RestoreReservation{AuthorizationID: "auth-restore-physical", OperationID: "restore-physical", PairID: pairID, SessionID: *attempt.SessionID, ExpectedGeneration: *attempt.TerminalGeneration, RiskScope: "POSSIBLE_PROMPT_REPLAY", AuthorizedPrincipal: "verified-subject", Actor: "supervisor"}
	if err := s.ReservePairRestore(ctx, req); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordPairRestoreUnknown(ctx, req.OperationID, "supervisor"); err != nil {
		t.Fatal(err)
	}
	if err := s.ClaimRestoreRecovery(ctx, req.OperationID, pairID, req.SessionID, req.ExpectedGeneration, "verified-subject", "supervisor", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	principal := "verified-subject"
	taskID, contractID, attemptID := attempt.TaskID, attempt.ContractID, attempt.AttemptID
	stop := domain.StopOperation{OperationID: "stop-restore-physical", Purpose: domain.QuarantineCleanup, PairID: pairID, TaskID: &taskID, ContractID: &contractID, AttemptID: &attemptID, SessionID: req.SessionID, TerminalGeneration: req.ExpectedGeneration, Stage: domain.StopRequested, Actor: "supervisor", RestoreOperationID: &req.OperationID, RestorePrincipal: &principal}
	if err := s.ClaimRestoreCleanupWithStop(ctx, stop, principal, "supervisor", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	callAt := time.Now().UTC()
	deadline := callAt.Add(time.Minute)
	if err := s.UpdateStopOperationStage(ctx, stop.OperationID, domain.StopRequested, domain.StopCallSucceeded, StopStageUpdate{ExpectedResolutionState: domain.StopResolutionInFlight, CallCompletedAt: &callAt, ConfirmationDeadlineAt: &deadline}); err != nil {
		t.Fatal(err)
	}
	confirmedAt := callAt.Add(time.Second)
	resolved := domain.StopResolutionTerminationConfirmed
	if err := s.UpdateStopOperationStage(ctx, stop.OperationID, domain.StopCallSucceeded, domain.StopTerminationConfirmed, StopStageUpdate{ExpectedResolutionState: domain.StopResolutionInFlight, TerminationConfirmedAt: &confirmedAt, ResolvedAt: &confirmedAt, ResolutionState: &resolved, ObservedSessionID: stop.SessionID, ObservedGeneration: stop.TerminalGeneration, ObservedIsTerminated: true}); err != nil {
		t.Fatal(err)
	}
	if err := s.ResolveAmbiguousRestore(ctx, req.OperationID, pairID, req.SessionID, req.ExpectedGeneration, domain.PhysicalExecutionResolution, "D11 stop record and confirmation audit", "verified-subject", "supervisor", confirmedAt); err != nil {
		t.Fatalf("positive D11 evidence rejected: %v", err)
	}
	op, err := s.GetPairRestoreOperation(ctx, req.OperationID)
	if err != nil || op.ResolutionBasis == nil || *op.ResolutionBasis != domain.PhysicalExecutionResolution {
		t.Fatalf("Class A basis was not persisted: %+v %v", op, err)
	}
}

func TestRestoreCleanupClaimAndStopIntentRollbackTogether(t *testing.T) {
	for _, inject := range []bool{false, true} {
		t.Run(map[bool]string{false: "positive", true: "audit_failure"}[inject], func(t *testing.T) {
			ctx := context.Background()
			s, _ := createTestStore(t)
			defer s.Close()
			setupReadyTask(t, s, "task-cleanup-rollback", "contract-cleanup-rollback")
			pairID := "pair-d-task-cleanup-rollback"
			seedWorkerSessionForTest(t, s, domain.WorkerSession{PairID: pairID, SessionID: "session-cleanup-rollback", RuntimeType: "agy_tui", WorkerAgentID: "agy", Status: domain.WorkerSessionTerminated, TerminalGeneration: "g1", QuarantineState: domain.QuarantineClean})
			req := domain.RestoreReservation{AuthorizationID: "auth-cleanup-rollback", OperationID: "restore-cleanup-rollback", PairID: pairID, SessionID: "session-cleanup-rollback", ExpectedGeneration: "g1", RiskScope: "POSSIBLE_PROMPT_REPLAY", AuthorizedPrincipal: "verified-subject", Actor: "supervisor"}
			if err := s.ReservePairRestore(ctx, req); err != nil {
				t.Fatal(err)
			}
			if err := s.RecordPairRestoreUnknown(ctx, req.OperationID, "supervisor"); err != nil {
				t.Fatal(err)
			}
			if err := s.ClaimRestoreRecovery(ctx, req.OperationID, pairID, req.SessionID, "g1", "verified-subject", "supervisor", time.Now().UTC()); err != nil {
				t.Fatal(err)
			}
			before, err := s.GetPairRestoreOperation(ctx, req.OperationID)
			if err != nil {
				t.Fatal(err)
			}
			principal := "verified-subject"
			stop := domain.StopOperation{OperationID: "stop-cleanup-rollback", Purpose: domain.PairMaintenance, PairID: pairID, SessionID: req.SessionID, TerminalGeneration: "g2", Stage: domain.StopRequested, Actor: "supervisor", RestoreOperationID: &req.OperationID, RestorePrincipal: &principal}
			observation := RestoreCleanupObservation{SessionID: req.SessionID, TerminalGeneration: "g2"}
			if err := s.ClaimRestoreCleanupWithStop(ctx, stop, principal, "supervisor", time.Now().UTC()); !errors.Is(err, ErrStateConflict) {
				t.Fatalf("missing runtime observation was accepted: %v", err)
			}
			if err := s.ClaimRestoreCleanupWithStop(ctx, stop, principal, "supervisor", time.Now().UTC(), RestoreCleanupObservation{SessionID: req.SessionID, TerminalGeneration: "g1"}); !errors.Is(err, ErrStateConflict) {
				t.Fatalf("expected generation was treated as observed runtime: %v", err)
			}
			if inject {
				if _, err := s.db.ExecContext(ctx, `CREATE TRIGGER reject_linked_stop_audit BEFORE INSERT ON audit_events WHEN NEW.event_type='STOP_OPERATION_REQUESTED' BEGIN SELECT RAISE(ABORT,'injected stop audit failure'); END`); err != nil {
					t.Fatal(err)
				}
			}
			err = s.ClaimRestoreCleanupWithStop(ctx, stop, principal, "supervisor", time.Now().UTC(), observation)
			if inject && (err == nil || !strings.Contains(err.Error(), "injected stop audit failure")) {
				t.Fatalf("wrong failure: %v", err)
			}
			if !inject && err != nil {
				t.Fatalf("positive control rejected: %v", err)
			}
			after, err := s.GetPairRestoreOperation(ctx, req.OperationID)
			if err != nil {
				t.Fatal(err)
			}
			var stopRows, claimAudits, stopAudits int
			if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM stop_operations WHERE operation_id=?`, stop.OperationID).Scan(&stopRows); err != nil {
				t.Fatal(err)
			}
			if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_events WHERE event_type='PAIR_RESTORE_CLEANUP_CLAIMED' AND json_extract(details_json,'$.operation_id')=?`, req.OperationID).Scan(&claimAudits); err != nil {
				t.Fatal(err)
			}
			if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_events WHERE event_type='STOP_OPERATION_REQUESTED' AND json_extract(details_json,'$.stop_operation_id')=?`, stop.OperationID).Scan(&stopAudits); err != nil {
				t.Fatal(err)
			}
			if inject {
				if after.ResolutionState != before.ResolutionState || after.Version != before.Version || stopRows != 0 || claimAudits != 0 || stopAudits != 0 {
					t.Fatalf("partial rollback: before=%+v after=%+v rows=%d audits=%d/%d", before, after, stopRows, claimAudits, stopAudits)
				}
			} else if after.ResolutionState != domain.RestoreCleanupClaimed || after.Version != before.Version+1 || stopRows != 1 || claimAudits != 1 || stopAudits != 1 {
				t.Fatalf("positive atomic claim missing: before=%+v after=%+v rows=%d audits=%d/%d", before, after, stopRows, claimAudits, stopAudits)
			}
		})
	}
}

func confirmD11StopForRestoreTest(t *testing.T, s *Store, stop domain.StopOperation) {
	t.Helper()
	ctx := context.Background()
	callAt := time.Now().UTC().Truncate(time.Microsecond)
	deadline := callAt.Add(time.Minute)
	if err := s.CommitStopCallAccepted(ctx, stop.OperationID, callAt, deadline); err != nil {
		t.Fatal(err)
	}
	confirmedAt := callAt.Add(time.Second)
	if err := s.CommitStopOutcome(ctx, stop.OperationID, StopTerminalOutcome{ExpectedStage: domain.StopCallSucceeded, Stage: domain.StopTerminationConfirmed, Resolution: domain.StopResolutionTerminationConfirmed, At: confirmedAt, ObservedSessionID: stop.SessionID, ObservedGeneration: stop.TerminalGeneration, ObservedIsTerminated: true}); err != nil {
		t.Fatal(err)
	}
}

func TestRestorePhysicalResolutionCoversRuntimeAndEveryQuarantinedLineage(t *testing.T) {
	for _, tc := range []struct {
		name                 string
		oldProof, newRuntime bool
		wantPhysical         bool
	}{
		{"old_only_new_runtime_active", true, false, false},
		{"new_only_old_attempt_unresolved", false, true, false},
		{"complete_coverage", true, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			s, _ := createTestStore(t)
			defer s.Close()
			pairID, attempt := setupBoundAttempt(t, s, "task-coverage", "contract-coverage", "attempt-coverage")
			taskID, contractID, attemptID := attempt.TaskID, attempt.ContractID, attempt.AttemptID
			if tc.oldProof {
				if err := s.TransitionTask(ctx, taskID, domain.StateDispatched, domain.StateRunning); err != nil {
					t.Fatal(err)
				}
				old := domain.StopOperation{OperationID: "stop-old", Purpose: domain.RunningAttemptStop, PairID: pairID, TaskID: &taskID, ContractID: &contractID, AttemptID: &attemptID, SessionID: *attempt.SessionID, TerminalGeneration: *attempt.TerminalGeneration, Actor: "supervisor"}
				if err := s.CreateStopOperation(ctx, old); err != nil {
					t.Fatal(err)
				}
				confirmD11StopForRestoreTest(t, s, old)
			} else {
				if err := s.AtomicTerminalTransition(ctx, attempt.TaskID, domain.StateDispatched, domain.StateFailed, "closed lineage", attempt.AttemptID, "ORDINARY_FAILURE"); err != nil {
					t.Fatal(err)
				}
			}
			setRestoreSessionTerminated(t, s, pairID)
			req := domain.RestoreReservation{AuthorizationID: "auth-coverage", OperationID: "restore-coverage", PairID: pairID, SessionID: *attempt.SessionID, ExpectedGeneration: *attempt.TerminalGeneration, RiskScope: "POSSIBLE_PROMPT_REPLAY", AuthorizedPrincipal: "verified-subject", Actor: "supervisor"}
			if err := s.ReservePairRestore(ctx, req); err != nil {
				t.Fatal(err)
			}
			if err := s.ConfirmPairRestore(ctx, req.OperationID, req.SessionID, req.ExpectedGeneration, "new-generation", "native", domain.WorkerSessionActive, "supervisor", time.Now().UTC()); err != nil {
				t.Fatal(err)
			}
			if _, err := s.db.ExecContext(ctx, `UPDATE task_attempts SET quarantine_state='QUARANTINED' WHERE attempt_id=?`, attemptID); err != nil {
				t.Fatal(err)
			}
			if err := s.ClaimRestoreRecovery(ctx, req.OperationID, pairID, req.SessionID, "new-generation", "verified-subject", "supervisor", time.Now().UTC()); err != nil {
				t.Fatal(err)
			}
			if tc.newRuntime {
				principal := "verified-subject"
				stop := domain.StopOperation{OperationID: "stop-new", Purpose: domain.PairMaintenance, PairID: pairID, SessionID: req.SessionID, TerminalGeneration: "new-generation", RestoreOperationID: &req.OperationID, RestorePrincipal: &principal, Actor: "supervisor"}
				if err := s.ClaimRestoreCleanupWithStop(ctx, stop, principal, "supervisor", time.Now().UTC(), RestoreCleanupObservation{SessionID: req.SessionID, TerminalGeneration: "new-generation"}); err != nil {
					t.Fatal(err)
				}
				confirmD11StopForRestoreTest(t, s, stop)
			}
			resolve := func() error {
				return s.ResolveAmbiguousRestore(ctx, req.OperationID, pairID, req.SessionID, req.ExpectedGeneration, domain.PhysicalExecutionResolution, "exact D11 runtime and lineage evidence", "verified-subject", "supervisor", time.Now().UTC())
			}
			if tc.wantPhysical {
				if _, err := s.db.ExecContext(ctx, `CREATE TRIGGER reject_physical_resolution_audit BEFORE INSERT ON audit_events WHEN NEW.event_type='PAIR_RESTORE_RESOLVED' BEGIN SELECT RAISE(ABORT,'injected resolution audit failure'); END`); err != nil {
					t.Fatal(err)
				}
				if err := resolve(); err == nil || !strings.Contains(err.Error(), "injected resolution audit failure") {
					t.Fatalf("audit rollback error=%v", err)
				}
				rolled, err := s.GetPairRestoreOperation(ctx, req.OperationID)
				if err != nil || rolled.ResolutionState != domain.RestoreCleanupClaimed || rolled.ResolutionBasis != nil {
					t.Fatalf("Tx D partial commit: %+v %v", rolled, err)
				}
				var resolvedAudits int
				if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_events WHERE event_type='PAIR_RESTORE_RESOLVED' AND json_extract(details_json,'$.operation_id')=?`, req.OperationID).Scan(&resolvedAudits); err != nil {
					t.Fatal(err)
				}
				if resolvedAudits != 0 {
					t.Fatalf("resolution audit survived rollback: %d", resolvedAudits)
				}
				if _, err := s.db.ExecContext(ctx, `DROP TRIGGER reject_physical_resolution_audit`); err != nil {
					t.Fatal(err)
				}
				if err := resolve(); err != nil {
					t.Fatalf("complete Class A rejected: %v", err)
				}
				if err := s.ResolveAmbiguousRestore(ctx, req.OperationID, pairID, req.SessionID, req.ExpectedGeneration, domain.AdministrativeRiskResolution, "stale writer", "verified-subject", "supervisor", time.Now().UTC()); !errors.Is(err, ErrStateConflict) {
					t.Fatalf("stale basis writer accepted: %v", err)
				}
			} else if err := resolve(); !errors.Is(err, ErrStateConflict) {
				t.Fatalf("incomplete Class A accepted: %v", err)
			}
			op, err := s.GetPairRestoreOperation(ctx, req.OperationID)
			if err != nil {
				t.Fatal(err)
			}
			want := domain.RestoreRecoveryClaimed
			if tc.newRuntime {
				want = domain.RestoreCleanupClaimed
			}
			if tc.wantPhysical {
				want = domain.RestoreResolved
			}
			if op.ResolutionState != want {
				t.Fatalf("wrong resolution after coverage check: %+v", op)
			}
			worker, err := s.GetWorkerSessionByPair(ctx, pairID)
			if err != nil || worker.QuarantineState != domain.QuarantineQuarantined {
				t.Fatalf("Tx D cleared quarantine: %+v %v", worker, err)
			}
			closed, err := s.GetTaskAttempt(ctx, attemptID)
			if err != nil || closed.QuarantineState != domain.QuarantineQuarantined {
				t.Fatalf("Tx D cleared attempt quarantine: %+v %v", closed, err)
			}
		})
	}
}

func TestPhysicalEvidenceRejectsPersistedEqualDeadline(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()
	pairID, attempt := setupBoundAttempt(t, s, "task-equal-evidence", "contract-equal-evidence", "attempt-equal-evidence")
	taskID, contractID, attemptID := attempt.TaskID, attempt.ContractID, attempt.AttemptID
	stop := domain.StopOperation{OperationID: "stop-equal-evidence", Purpose: domain.RunningAttemptStop, PairID: pairID, TaskID: &taskID, ContractID: &contractID, AttemptID: &attemptID, SessionID: *attempt.SessionID, TerminalGeneration: *attempt.TerminalGeneration, Actor: "supervisor"}
	if err := s.TransitionTask(ctx, taskID, domain.StateDispatched, domain.StateRunning); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateStopOperation(ctx, stop); err != nil {
		t.Fatal(err)
	}
	confirmD11StopForRestoreTest(t, s, stop)
	setRestoreSessionTerminated(t, s, pairID)
	req := domain.RestoreReservation{AuthorizationID: "auth-equal-evidence", OperationID: "restore-equal-evidence", PairID: pairID, SessionID: stop.SessionID, ExpectedGeneration: stop.TerminalGeneration, RiskScope: "POSSIBLE_PROMPT_REPLAY", AuthorizedPrincipal: "verified-subject", Actor: "supervisor"}
	if err := s.ReservePairRestore(ctx, req); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE task_attempts SET quarantine_state='QUARANTINED' WHERE attempt_id=?`, attemptID); err != nil {
		t.Fatal(err)
	}
	// Simulate a legacy persisted equality row plus matching audit. The normal
	// confirmation API rejects equality; Tx D must reject it independently.
	confirmed, err := s.GetStopOperation(ctx, stop.OperationID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE stop_operations SET confirmation_deadline_at=? WHERE operation_id=?`, formatTime(*confirmed.TerminationConfirmedAt), stop.OperationID); err != nil {
		t.Fatal(err)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	eventID, err := newAuditEventID()
	if err != nil {
		t.Fatal(err)
	}
	_, err = appendAuditEventTx(ctx, tx, domain.AuditEvent{EventID: eventID, EventType: domain.AuditStopOperationConfirmed, Timestamp: *confirmed.TerminationConfirmedAt, PairID: pairID, TaskID: taskID, ContractID: contractID, AttemptID: attemptID, Actor: "supervisor", Details: map[string]any{"stop_operation_id": stop.OperationID, "restore_operation_id": nil, "session_id": stop.SessionID, "target_generation": stop.TerminalGeneration, "confirmation_deadline_at": formatTime(*confirmed.TerminationConfirmedAt), "termination_confirmed_at": formatTime(*confirmed.TerminationConfirmedAt), "observed_session_id": stop.SessionID, "observed_generation": stop.TerminalGeneration, "is_terminated": true}})
	if err != nil {
		tx.Rollback()
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	tx, err = s.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	valid, err := positiveD11StopEvidence(ctx, tx, pairID, stop.SessionID, stop.TerminalGeneration, "", attemptID)
	tx.Rollback()
	if err != nil || valid {
		t.Fatalf("equality accepted as Class A physical proof: valid=%v err=%v", valid, err)
	}
}

func TestRestorePhysicalEvidenceTimestampPrecisionAndFailClosed(t *testing.T) {
	base := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name, confirmedRaw, deadlineRaw string
		accepted                        bool
	}{
		{"whole_second_before_500ms", formatTime(base), formatTime(base.Add(500 * time.Millisecond)), true},
		{"different_fraction_digits", formatTime(base.Add(900 * time.Millisecond)), formatTime(base.Add(950 * time.Millisecond)), true},
		{"before_1ns", formatTime(base.Add(500*time.Millisecond - time.Nanosecond)), formatTime(base.Add(500 * time.Millisecond)), true},
		{"equal", formatTime(base.Add(500 * time.Millisecond)), formatTime(base.Add(500 * time.Millisecond)), false},
		{"after_1ns", formatTime(base.Add(500*time.Millisecond + time.Nanosecond)), formatTime(base.Add(500 * time.Millisecond)), false},
		{"malformed_confirmation", "invalid-timestamp", formatTime(base.Add(500 * time.Millisecond)), false},
		{"malformed_deadline", formatTime(base), "invalid-timestamp", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			s, _ := createTestStore(t)
			defer s.Close()
			setupReadyTask(t, s, "task-precision", "contract-precision")
			pairID := "pair-d-task-precision"
			seedWorkerSessionForTest(t, s, domain.WorkerSession{PairID: pairID, SessionID: "session-precision", RuntimeType: "agy_tui", WorkerAgentID: "agy", Status: domain.WorkerSessionTerminated, TerminalGeneration: "g1", QuarantineState: domain.QuarantineClean})
			req := domain.RestoreReservation{AuthorizationID: "auth-precision", OperationID: "restore-precision", PairID: pairID, SessionID: "session-precision", ExpectedGeneration: "g1", RiskScope: "POSSIBLE_PROMPT_REPLAY", AuthorizedPrincipal: "verified-subject", Actor: "supervisor"}
			if err := s.ReservePairRestore(ctx, req); err != nil {
				t.Fatal(err)
			}
			if err := s.RecordPairRestoreUnknown(ctx, req.OperationID, "supervisor"); err != nil {
				t.Fatal(err)
			}
			principal := "verified-subject"
			if err := s.ClaimRestoreRecovery(ctx, req.OperationID, pairID, req.SessionID, req.ExpectedGeneration, principal, "supervisor", time.Now().UTC()); err != nil {
				t.Fatal(err)
			}
			stop := domain.StopOperation{OperationID: "stop-precision", Purpose: domain.PairMaintenance, PairID: pairID, SessionID: req.SessionID, TerminalGeneration: req.ExpectedGeneration, RestoreOperationID: &req.OperationID, RestorePrincipal: &principal, Actor: "supervisor"}
			if err := s.ClaimRestoreCleanupWithStop(ctx, stop, principal, "supervisor", time.Now().UTC(), RestoreCleanupObservation{SessionID: req.SessionID, TerminalGeneration: req.ExpectedGeneration}); err != nil {
				t.Fatal(err)
			}
			confirmD11StopForRestoreTest(t, s, stop)
			// The normal stop API cannot persist equality or malformed times.
			// Build a legacy candidate whose matching audit tuple is complete.
			if _, err := s.db.ExecContext(ctx, `UPDATE stop_operations SET termination_confirmed_at=?,confirmation_deadline_at=? WHERE operation_id=?`, tc.confirmedRaw, tc.deadlineRaw, stop.OperationID); err != nil {
				t.Fatal(err)
			}
			tx, err := s.db.BeginTx(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			eventID, err := newAuditEventID()
			if err != nil {
				tx.Rollback()
				t.Fatal(err)
			}
			_, err = appendAuditEventTx(ctx, tx, domain.AuditEvent{EventID: eventID, EventType: domain.AuditStopOperationConfirmed, Timestamp: time.Now().UTC(), PairID: pairID, Actor: "supervisor", Details: map[string]any{"stop_operation_id": stop.OperationID, "restore_operation_id": req.OperationID, "session_id": stop.SessionID, "target_generation": stop.TerminalGeneration, "confirmation_deadline_at": tc.deadlineRaw, "termination_confirmed_at": tc.confirmedRaw, "observed_session_id": stop.SessionID, "observed_generation": stop.TerminalGeneration, "is_terminated": true}})
			if err != nil {
				tx.Rollback()
				t.Fatal(err)
			}
			if err := tx.Commit(); err != nil {
				t.Fatal(err)
			}
			err = s.ResolveAmbiguousRestore(ctx, req.OperationID, pairID, req.SessionID, req.ExpectedGeneration, domain.PhysicalExecutionResolution, "matching D11 evidence", principal, "supervisor", time.Now().UTC())
			if tc.accepted && err != nil {
				t.Fatalf("valid physical evidence rejected: %v", err)
			}
			if !tc.accepted && !errors.Is(err, ErrStateConflict) {
				t.Fatalf("invalid physical evidence accepted: %v", err)
			}
			op, err := s.GetPairRestoreOperation(ctx, req.OperationID)
			if err != nil {
				t.Fatal(err)
			}
			var stage, resolution string
			if err := s.db.QueryRowContext(ctx, `SELECT stage,resolution_state FROM stop_operations WHERE operation_id=?`, stop.OperationID).Scan(&stage, &resolution); err != nil {
				t.Fatal(err)
			}
			var confirmationAudits, resolutionAudits int
			if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_events WHERE event_type='STOP_OPERATION_CONFIRMED' AND json_extract(details_json,'$.stop_operation_id')=?`, stop.OperationID).Scan(&confirmationAudits); err != nil {
				t.Fatal(err)
			}
			if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_events WHERE event_type='PAIR_RESTORE_RESOLVED' AND json_extract(details_json,'$.operation_id')=?`, req.OperationID).Scan(&resolutionAudits); err != nil {
				t.Fatal(err)
			}
			if stage != string(domain.StopTerminationConfirmed) || resolution != string(domain.StopResolutionTerminationConfirmed) || confirmationAudits != 2 {
				t.Fatalf("stop proof changed: stage=%s resolution=%s audits=%d", stage, resolution, confirmationAudits)
			}
			if tc.accepted {
				if op.ResolutionState != domain.RestoreResolved || op.ResolutionBasis == nil || *op.ResolutionBasis != domain.PhysicalExecutionResolution || resolutionAudits != 1 {
					t.Fatalf("Tx D physical result missing: %+v audits=%d", op, resolutionAudits)
				}
			} else if op.ResolutionState != domain.RestoreCleanupClaimed || op.ResolutionBasis != nil || resolutionAudits != 0 {
				t.Fatalf("Tx D partial commit: %+v audits=%d", op, resolutionAudits)
			}
			worker, err := s.GetWorkerSessionByPair(ctx, pairID)
			if err != nil || worker.QuarantineState != domain.QuarantineQuarantined {
				t.Fatalf("quarantine cleared by Tx D: %+v %v", worker, err)
			}
		})
	}
}
