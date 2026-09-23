package store

import (
	"context"
	"errors"
	"github.com/trungqwe/ai-supervisor/internal/domain"
	"testing"
	"time"
)

func TestStartupRestoreClassificationAtomicReplayAndConfirmedProvenance(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()
	setupReadyTask(t, s, "task-startup-restore", "contract-startup-restore")
	pair := "pair-d-task-startup-restore"
	seedWorkerSessionForTest(t, s, domain.WorkerSession{PairID: pair, SessionID: "session-startup-restore", RuntimeType: "agy_tui", WorkerAgentID: "agy", Status: domain.WorkerSessionTerminated, TerminalGeneration: "gen-old", QuarantineState: domain.QuarantineClean})
	req := domain.RestoreReservation{AuthorizationID: "auth-startup-restore", OperationID: "restore-startup", PairID: pair, SessionID: "session-startup-restore", ExpectedGeneration: "gen-old", RiskScope: "POSSIBLE_PROMPT_REPLAY", AuthorizedPrincipal: "verified-subject", Actor: "supervisor"}
	if err := s.ReservePairRestore(ctx, req); err != nil {
		t.Fatal(err)
	}
	var version int64
	if err := s.db.QueryRowContext(ctx, `SELECT version FROM pair_restore_operations WHERE operation_id=?`, req.OperationID).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, `CREATE TRIGGER reject_startup_restore_audit BEFORE INSERT ON audit_events WHEN NEW.event_type='PAIR_RESTORE_OUTCOME_UNKNOWN' BEGIN SELECT RAISE(ABORT,'injected audit failure'); END`); err != nil {
		t.Fatal(err)
	}
	if err := s.ClassifyInterruptedRestore(ctx, req.OperationID, "supervisor", time.Now()); err == nil {
		t.Fatal("audit failure did not roll back restore classification")
	}
	op, err := s.GetPairRestoreOperation(ctx, req.OperationID)
	if err != nil || op.ResolutionState != domain.RestoreInFlight || op.Version != version {
		t.Fatalf("partial restore classification: %+v %v", op, err)
	}
	if _, err := s.db.ExecContext(ctx, `DROP TRIGGER reject_startup_restore_audit`); err != nil {
		t.Fatal(err)
	}
	if err := s.ClassifyInterruptedRestore(ctx, req.OperationID, "supervisor", time.Now()); err != nil {
		t.Fatal(err)
	}
	op, err = s.GetPairRestoreOperation(ctx, req.OperationID)
	if err != nil || op.ResolutionState != domain.RestoreOutcomeUnknown || op.Version != version+1 {
		t.Fatalf("unknown restore state: %+v %v", op, err)
	}
	ws, err := s.GetWorkerSessionByPair(ctx, pair)
	if err != nil || ws.QuarantineState != domain.QuarantineQuarantined {
		t.Fatalf("session quarantine: %+v %v", ws, err)
	}
	if err := s.ClassifyInterruptedRestore(ctx, req.OperationID, "supervisor", time.Now()); err != nil {
		t.Fatal(err)
	}
	var events int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_events WHERE event_type='PAIR_RESTORE_OUTCOME_UNKNOWN' AND json_extract(details_json,'$.operation_id')=?`, req.OperationID).Scan(&events); err != nil || events != 1 {
		t.Fatalf("replay audit count=%d err=%v", events, err)
	}
}

func TestStartupProvisioningFailureAtomicReplay(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()
	pair := "pair-startup-provision"
	setupProvisioningPair(t, s, pair)
	op := domain.PairProvisioningOperation{OperationID: "provision-startup", PairID: pair, ClientToken: "token-startup"}
	if err := s.ReservePairProvisioning(ctx, op, "supervisor"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, `CREATE TRIGGER reject_startup_provision_audit BEFORE INSERT ON audit_events WHEN NEW.event_type='PAIR_SESSION_PROVISION_FAILED' BEGIN SELECT RAISE(ABORT,'injected audit failure'); END`); err != nil {
		t.Fatal(err)
	}
	if err := s.FailProvisioningAtStartup(ctx, op.OperationID, "supervisor", time.Now()); err == nil {
		t.Fatal("audit failure did not roll back provisioning")
	}
	got, err := s.GetPairProvisioningOperation(ctx, op.OperationID)
	if err != nil || got.Stage != domain.ProvisionRequested {
		t.Fatalf("partial provisioning classification: %+v %v", got, err)
	}
	if _, err := s.db.ExecContext(ctx, `DROP TRIGGER reject_startup_provision_audit`); err != nil {
		t.Fatal(err)
	}
	if err := s.FailProvisioningAtStartup(ctx, op.OperationID, "supervisor", time.Now()); err != nil {
		t.Fatal(err)
	}
	got, err = s.GetPairProvisioningOperation(ctx, op.OperationID)
	if err != nil || got.Stage != domain.ProvisionFailed {
		t.Fatalf("failed provisioning state: %+v %v", got, err)
	}
	var notes string
	if err := s.db.QueryRowContext(ctx, `SELECT resolution_notes FROM pair_provisioning_operations WHERE operation_id=?`, op.OperationID).Scan(&notes); err != nil || notes != "STARTUP_RECOVERY_SWEEP" {
		t.Fatalf("resolution notes=%q err=%v", notes, err)
	}
	if err := s.FailProvisioningAtStartup(ctx, op.OperationID, "supervisor", time.Now()); err != nil {
		t.Fatal(err)
	}
	var events int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_events WHERE event_type='PAIR_SESSION_PROVISION_FAILED' AND json_extract(details_json,'$.operation_id')=?`, op.OperationID).Scan(&events); err != nil || events != 1 {
		t.Fatalf("replay audit count=%d err=%v", events, err)
	}
}

func TestStartupConfirmedRestoreRetainsHTTP200(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()
	setupReadyTask(t, s, "task-startup-confirmed", "contract-startup-confirmed")
	pair := "pair-d-task-startup-confirmed"
	seedWorkerSessionForTest(t, s, domain.WorkerSession{PairID: pair, SessionID: "session-startup-confirmed", RuntimeType: "agy_tui", WorkerAgentID: "agy", Status: domain.WorkerSessionTerminated, TerminalGeneration: "old", QuarantineState: domain.QuarantineClean})
	req := domain.RestoreReservation{AuthorizationID: "auth-startup-confirmed", OperationID: "restore-startup-confirmed", PairID: pair, SessionID: "session-startup-confirmed", ExpectedGeneration: "old", RiskScope: "POSSIBLE_PROMPT_REPLAY", AuthorizedPrincipal: "verified-subject", Actor: "supervisor"}
	if err := s.ReservePairRestore(ctx, req); err != nil {
		t.Fatal(err)
	}
	at := time.Now().UTC()
	if err := s.ConfirmPairRestore(ctx, req.OperationID, req.SessionID, req.ExpectedGeneration, "new", "native", domain.WorkerSessionIdle, "supervisor", at); err != nil {
		t.Fatal(err)
	}
	before, err := s.GetPairRestoreOperation(ctx, req.OperationID)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ClassifyInterruptedRestore(ctx, req.OperationID, "supervisor", at.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	after, err := s.GetPairRestoreOperation(ctx, req.OperationID)
	if err != nil {
		t.Fatal(err)
	}
	if before.Stage != after.Stage || before.ResolutionState != after.ResolutionState || before.Version != after.Version || after.Stage != "RESTORE_CONFIRMED" || after.ResolutionState != domain.RestoreInFlight || after.HTTP200At == nil {
		t.Fatalf("confirmed restore provenance changed: before=%+v after=%+v", before, after)
	}
	var events int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_events WHERE event_type='PAIR_RESTORE_OUTCOME_UNKNOWN' AND json_extract(details_json,'$.operation_id')=?`, req.OperationID).Scan(&events); err != nil || events != 0 {
		t.Fatalf("confirmed restore misclassified: count=%d err=%v", events, err)
	}
}

func TestPostSendOutageAndFreshRecoveryCAS(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()
	_, attempt := setupBoundAttempt(t, s, "task-post-outage", "contract-post-outage", "attempt-post-outage")
	opID := "dispatch-attempt-post-outage"
	if err := s.RecordSendRequested(ctx, opID, *attempt.SessionID, *attempt.TerminalGeneration, "idle", false, "supervisor", time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordSendConfirmed(ctx, opID, "supervisor", true, time.Now(), domain.ExecutionBudgetPolicy{Duration: time.Hour, PolicyRef: "fixture-policy"}); err != nil {
		t.Fatal(err)
	}
	snap, err := s.ListRecoverySnapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Executions) != 1 || snap.Executions[0].DispatchStage != "SEND_CONFIRMED" {
		t.Fatalf("snapshot: %+v", snap.Executions)
	}
	x := snap.Executions[0]
	if _, err = s.db.ExecContext(ctx, `CREATE TRIGGER fail_post_outage BEFORE INSERT ON audit_events WHEN NEW.event_type='POST_SEND_RECOVERY_PENDING' BEGIN SELECT RAISE(ABORT,'injected outage audit failure'); END`); err != nil {
		t.Fatal(err)
	}
	if err = s.RecordPostSendOutage(ctx, x, "supervisor", "AO_UNAVAILABLE", "run-1", time.Now()); err == nil {
		t.Fatal("outage audit failure accepted")
	}
	a, err := s.GetTaskAttempt(ctx, attempt.AttemptID)
	if err != nil || a.RecoveryDisposition != nil {
		t.Fatalf("partial outage hold: %+v %v", a, err)
	}
	if _, err = s.db.ExecContext(ctx, `DROP TRIGGER fail_post_outage`); err != nil {
		t.Fatal(err)
	}
	if err = s.RecordPostSendOutage(ctx, x, "supervisor", "AO_UNAVAILABLE", "run-1", time.Now()); err != nil {
		t.Fatal(err)
	}
	snap, err = s.ListRecoverySnapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	held := snap.Executions[0]
	if err = s.RecordPostSendOutage(ctx, held, "supervisor", "AO_UNAVAILABLE", "run-2", time.Now()); err != nil {
		t.Fatal(err)
	}
	if err = s.ObservePostSend(ctx, x, PostSendObservation{SessionID: x.SessionID, Generation: x.Generation, Activity: "active"}, "supervisor", "run-1", time.Now()); err == nil {
		t.Fatal("stale snapshot cleared hold")
	}
	if _, err = s.db.ExecContext(ctx, `CREATE TRIGGER fail_post_recovery BEFORE INSERT ON audit_events WHEN NEW.event_type='POST_SEND_STATUS_RECOVERED' BEGIN SELECT RAISE(ABORT,'injected recovery audit failure'); END`); err != nil {
		t.Fatal(err)
	}
	if err = s.ObservePostSend(ctx, held, PostSendObservation{SessionID: x.SessionID, Generation: x.Generation, Activity: "active"}, "supervisor", "run-2", time.Now()); err == nil {
		t.Fatal("recovery audit failure accepted")
	}
	task, _ := s.GetTask(ctx, x.TaskID)
	if task.State != domain.StateDispatched {
		t.Fatalf("partial task transition: %s", task.State)
	}
	if _, err = s.db.ExecContext(ctx, `DROP TRIGGER fail_post_recovery`); err != nil {
		t.Fatal(err)
	}
	if err = s.ObservePostSend(ctx, held, PostSendObservation{SessionID: x.SessionID, Generation: x.Generation, Activity: "active"}, "supervisor", "run-2", time.Now()); err != nil {
		t.Fatal(err)
	}
	task, _ = s.GetTask(ctx, x.TaskID)
	a, _ = s.GetTaskAttempt(ctx, x.AttemptID)
	if task.State != domain.StateRunning || a.RecoveryDisposition != nil || a.EndedAt != nil {
		t.Fatalf("recovery: task=%+v attempt=%+v", task, a)
	}
	var pending, recovered int
	if err = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_events WHERE event_type='POST_SEND_RECOVERY_PENDING'`).Scan(&pending); err != nil {
		t.Fatal(err)
	}
	if err = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_events WHERE event_type='POST_SEND_STATUS_RECOVERED'`).Scan(&recovered); err != nil {
		t.Fatal(err)
	}
	if pending != 1 || recovered != 1 {
		t.Fatalf("audit counts pending=%d recovered=%d", pending, recovered)
	}
}

func TestMissedActiveWindowAvailableHandoffAuditRollback(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()
	_, attempt := setupBoundAttempt(t, s, "task-handoff-rollback", "contract-handoff-rollback", "attempt-handoff-rollback")
	opID := "dispatch-attempt-handoff-rollback"
	if err := s.RecordSendRequested(ctx, opID, *attempt.SessionID, *attempt.TerminalGeneration, "idle", false, "supervisor", time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordSendConfirmed(ctx, opID, "supervisor", true, time.Now(), domain.ExecutionBudgetPolicy{Duration: time.Hour, PolicyRef: "fixture-policy"}); err != nil {
		t.Fatal(err)
	}
	snap, err := s.ListRecoverySnapshot(ctx)
	if err != nil || len(snap.Executions) != 1 {
		t.Fatalf("snapshot: %+v %v", snap, err)
	}
	x := snap.Executions[0]
	if _, err = s.db.ExecContext(ctx, `CREATE TRIGGER fail_handoff_audit BEFORE INSERT ON audit_events WHEN NEW.event_type='MISSED_ACTIVE_WINDOW_HANDOFF_DOWNSTREAM' BEGIN SELECT RAISE(ABORT,'injected handoff audit failure'); END`); err != nil {
		t.Fatal(err)
	}
	observation := PostSendObservation{SessionID: x.SessionID, Generation: x.Generation, Activity: "idle", HandoffAvailable: true}
	if err := s.ObservePostSend(ctx, x, observation, "supervisor", "poll", time.Now()); err == nil {
		t.Fatal("handoff audit failure accepted")
	}
	task, _ := s.GetTask(ctx, x.TaskID)
	a, _ := s.GetTaskAttempt(ctx, x.AttemptID)
	if task.State != domain.StateDispatched || a.RecoveryDisposition != nil || a.EndedAt != nil {
		t.Fatalf("partial handoff commit: task=%+v attempt=%+v", task, a)
	}
	if _, err = s.db.ExecContext(ctx, `DROP TRIGGER fail_handoff_audit`); err != nil {
		t.Fatal(err)
	}
	if err := s.ObservePostSend(ctx, x, observation, "supervisor", "poll", time.Now()); err != nil {
		t.Fatal(err)
	}
	task, _ = s.GetTask(ctx, x.TaskID)
	a, _ = s.GetTaskAttempt(ctx, x.AttemptID)
	if task.State != domain.StateRunning || a.RecoveryDisposition == nil || *a.RecoveryDisposition != "MISSED_ACTIVE_WINDOW" || a.EndedAt != nil {
		t.Fatalf("handoff commit: task=%+v attempt=%+v", task, a)
	}
}

func TestGenericPostSendObservationCannotCloseStopOwnedAttempt(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()
	pair, attempt := setupBoundAttempt(t, s, "task-stop-owned", "contract-stop-owned", "attempt-stop-owned")
	opID := "dispatch-attempt-stop-owned"
	if err := s.RecordSendRequested(ctx, opID, *attempt.SessionID, *attempt.TerminalGeneration, "idle", false, "supervisor", time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordSendConfirmed(ctx, opID, "supervisor", true, time.Now(), domain.ExecutionBudgetPolicy{Duration: time.Hour, PolicyRef: "fixture-policy"}); err != nil {
		t.Fatal(err)
	}
	if err := s.TransitionTask(ctx, attempt.TaskID, domain.StateDispatched, domain.StateRunning); err != nil {
		t.Fatal(err)
	}
	snap, err := s.ListRecoverySnapshot(ctx)
	if err != nil || len(snap.Executions) != 1 {
		t.Fatalf("snapshot: %+v %v", snap, err)
	}
	x := snap.Executions[0]
	task, contract, id := attempt.TaskID, attempt.ContractID, attempt.AttemptID
	stop := domain.StopOperation{OperationID: "stop-owned", Purpose: domain.RunningAttemptStop, PairID: pair, TaskID: &task, ContractID: &contract, AttemptID: &id, SessionID: x.SessionID, TerminalGeneration: x.Generation, Actor: "supervisor"}
	if err := s.ReserveStopOperation(ctx, stop); err != nil {
		t.Fatal(err)
	}
	if err := s.ObservePostSend(ctx, x, PostSendObservation{SessionID: x.SessionID, Generation: x.Generation, Activity: "active", Terminated: true}, "supervisor", "stale-poll", time.Now()); !errors.Is(err, ErrStateConflict) {
		t.Fatalf("generic observation bypassed stop: %v", err)
	}
	before, _ := s.GetTask(ctx, x.TaskID)
	a, _ := s.GetTaskAttempt(ctx, x.AttemptID)
	if before.State != domain.StateRunning || a.EndedAt != nil {
		t.Fatalf("generic partial closure: task=%+v attempt=%+v", before, a)
	}
	call := time.Now().UTC()
	if err := s.CommitStopCallAccepted(ctx, stop.OperationID, call, call.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, `CREATE TRIGGER fail_stop_owned_audit BEFORE INSERT ON audit_events WHEN NEW.event_type='STOP_OPERATION_CONFIRMED' BEGIN SELECT RAISE(ABORT,'injected stop audit failure'); END`); err != nil {
		t.Fatal(err)
	}
	outcome := StopTerminalOutcome{ExpectedStage: domain.StopCallSucceeded, Stage: domain.StopTerminationConfirmed, Resolution: domain.StopResolutionTerminationConfirmed, At: call.Add(time.Second), ObservedSessionID: x.SessionID, ObservedGeneration: x.Generation, ObservedIsTerminated: true}
	if err := s.CommitStopOutcome(ctx, stop.OperationID, outcome); err == nil {
		t.Fatal("stop audit failure accepted")
	}
	st, _ := s.GetStopOperation(ctx, stop.OperationID)
	a, _ = s.GetTaskAttempt(ctx, x.AttemptID)
	if st.ResolutionState != domain.StopResolutionInFlight || a.EndedAt != nil {
		t.Fatalf("partial stop failure: stop=%+v attempt=%+v", st, a)
	}
	if _, err := s.db.ExecContext(ctx, `DROP TRIGGER fail_stop_owned_audit`); err != nil {
		t.Fatal(err)
	}
	if err := s.CommitStopOutcome(ctx, stop.OperationID, outcome); err != nil {
		t.Fatal(err)
	}
	st, _ = s.GetStopOperation(ctx, stop.OperationID)
	a, _ = s.GetTaskAttempt(ctx, x.AttemptID)
	if st.ResolutionState != domain.StopResolutionTerminationConfirmed || a.EndedAt == nil || a.RecoveryDisposition == nil || *a.RecoveryDisposition != "WORKER_STOPPED" {
		t.Fatalf("stop winner lost: stop=%+v attempt=%+v", st, a)
	}
}

func TestAbandonedStopAliveResolutionAuditAndRollback(t *testing.T) {
	ctx := context.Background()
	s, stop := runningStopFixture(t)
	defer s.Close()
	now := time.Now().UTC()
	if _, err := s.db.ExecContext(ctx, `CREATE TRIGGER fail_abandoned_stop BEFORE INSERT ON audit_events WHEN NEW.event_type='STOP_OPERATION_RESOLVED' BEGIN SELECT RAISE(ABORT,'injected stop audit failure'); END`); err != nil {
		t.Fatal(err)
	}
	if err := s.ClassifyAbandonedStopAlive(ctx, stop.OperationID, stop.SessionID, stop.TerminalGeneration, "supervisor", "run-1", now); err == nil {
		t.Fatal("audit failure accepted")
	}
	op, err := s.GetStopOperation(ctx, stop.OperationID)
	if err != nil || op.ResolutionState != domain.StopResolutionInFlight {
		t.Fatalf("partial stop resolution: %+v %v", op, err)
	}
	if _, err = s.db.ExecContext(ctx, `DROP TRIGGER fail_abandoned_stop`); err != nil {
		t.Fatal(err)
	}
	if err = s.ClassifyAbandonedStopAlive(ctx, stop.OperationID, stop.SessionID, "wrong", "supervisor", "run-1", now); err == nil {
		t.Fatal("wrong generation accepted")
	}
	if err = s.ClassifyAbandonedStopAlive(ctx, stop.OperationID, stop.SessionID, stop.TerminalGeneration, "supervisor", "run-1", now); err != nil {
		t.Fatal(err)
	}
	if err = s.ClassifyAbandonedStopAlive(ctx, stop.OperationID, stop.SessionID, stop.TerminalGeneration, "supervisor", "run-1", now); err != nil {
		t.Fatal(err)
	}
	op, _ = s.GetStopOperation(ctx, stop.OperationID)
	if op.Stage != domain.StopRequested || op.ResolutionState != domain.StopResolutionReissueRequiresHuman || op.ResolvedAt == nil {
		t.Fatalf("stop resolution: %+v", op)
	}
	task, _ := s.GetTask(ctx, *stop.TaskID)
	if task.State != domain.StateRunning {
		t.Fatalf("live task changed: %s", task.State)
	}
	var events int
	if err = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_events WHERE event_type='STOP_OPERATION_RESOLVED' AND json_extract(details_json,'$.recovery_invocation')='run-1'`).Scan(&events); err != nil || events != 1 {
		t.Fatalf("audit count=%d err=%v", events, err)
	}
}

func TestPostSendLifecycleMatrixAndHoldPrecedence(t *testing.T) {
	for _, tc := range []struct {
		name        string
		state       domain.TaskState
		obs         PostSendObservation
		want        domain.TaskState
		closed      bool
		disposition string
	}{
		{"active", domain.StateDispatched, PostSendObservation{Activity: "active"}, domain.StateRunning, false, ""},
		{"waiting", domain.StateDispatched, PostSendObservation{Activity: "waiting_input"}, domain.StateRunning, false, "AO_WAITING_INPUT_OBSERVED"},
		{"blocked", domain.StateDispatched, PostSendObservation{Activity: "blocked"}, domain.StateRunning, false, "AO_BLOCKED_DECISION"},
		{"idle", domain.StateDispatched, PostSendObservation{Activity: "idle"}, domain.StateFailed, true, "MISSED_ACTIVE_WINDOW"},
		{"exited", domain.StateDispatched, PostSendObservation{Activity: "exited"}, domain.StateDispatched, false, ""},
		{"terminated", domain.StateDispatched, PostSendObservation{Terminated: true, Activity: "exited"}, domain.StateFailed, true, "WORKER_TERMINATION_UNKNOWN"},
		{"absent", domain.StateDispatched, PostSendObservation{Absent: true}, domain.StateFailed, true, "SESSION_ABSENT"},
		{"mismatch", domain.StateDispatched, PostSendObservation{Generation: "wrong", Activity: "active"}, domain.StateFailed, true, "STALE_EXECUTION_GENERATION"},
		{"running_waiting", domain.StateRunning, PostSendObservation{Activity: "waiting_input"}, domain.StateRunning, false, "AO_WAITING_INPUT_OBSERVED"},
		{"running_blocked", domain.StateRunning, PostSendObservation{Activity: "blocked"}, domain.StateRunning, false, "AO_BLOCKED_DECISION"},
		{"running_terminated", domain.StateRunning, PostSendObservation{Terminated: true}, domain.StateFailed, true, "WORKER_TERMINATION_UNKNOWN"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			s, _ := createTestStore(t)
			defer s.Close()
			_, a := setupBoundAttempt(t, s, "task-matrix", "contract-matrix", "attempt-matrix")
			op := "dispatch-attempt-matrix"
			if err := s.RecordSendRequested(ctx, op, *a.SessionID, *a.TerminalGeneration, "idle", false, "fixture", time.Now()); err != nil {
				t.Fatal(err)
			}
			if err := s.RecordSendConfirmed(ctx, op, "fixture", true, time.Now(), domain.ExecutionBudgetPolicy{Duration: time.Hour, PolicyRef: "fixture-policy"}); err != nil {
				t.Fatal(err)
			}
			if tc.state == domain.StateRunning {
				if err := s.TransitionTask(ctx, a.TaskID, domain.StateDispatched, domain.StateRunning); err != nil {
					t.Fatal(err)
				}
			}
			snap, err := s.ListRecoverySnapshot(ctx)
			if err != nil || len(snap.Executions) != 1 {
				t.Fatalf("snapshot %+v %v", snap.Executions, err)
			}
			o := tc.obs
			o.SessionID = *a.SessionID
			if o.Generation == "" && !o.Absent {
				o.Generation = *a.TerminalGeneration
			}
			if err := s.ObservePostSend(ctx, snap.Executions[0], o, "supervisor", "matrix", time.Now().UTC()); err != nil {
				t.Fatal(err)
			}
			task, _ := s.GetTask(ctx, a.TaskID)
			got, _ := s.GetTaskAttempt(ctx, a.AttemptID)
			disposition := ""
			if got.RecoveryDisposition != nil {
				disposition = *got.RecoveryDisposition
			}
			if task.State != tc.want || (got.EndedAt != nil) != tc.closed || disposition != tc.disposition {
				t.Fatalf("state=%s closed=%v disposition=%q", task.State, got.EndedAt != nil, disposition)
			}
			if tc.name == "idle" {
				var n int
				if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_events WHERE event_type='MISSED_ACTIVE_WINDOW_HANDOFF_DOWNSTREAM'`).Scan(&n); err != nil || n != 1 {
					t.Fatalf("idle handoff audit=%d %v", n, err)
				}
			}
		})
	}
}

func TestPostSendStrongerHoldRejectsOutageAndTerminalRecoveryRollback(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()
	_, a := setupBoundAttempt(t, s, "task-strong", "contract-strong", "attempt-strong")
	op := "dispatch-attempt-strong"
	if err := s.RecordSendRequested(ctx, op, *a.SessionID, *a.TerminalGeneration, "idle", false, "fixture", time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordSendConfirmed(ctx, op, "fixture", true, time.Now(), domain.ExecutionBudgetPolicy{Duration: time.Hour, PolicyRef: "fixture-policy"}); err != nil {
		t.Fatal(err)
	}
	snap, _ := s.ListRecoverySnapshot(ctx)
	x := snap.Executions[0]
	if err := s.ObservePostSend(ctx, x, PostSendObservation{SessionID: x.SessionID, Generation: x.Generation, Activity: "blocked"}, "supervisor", "matrix", time.Now()); err != nil {
		t.Fatal(err)
	}
	snap, _ = s.ListRecoverySnapshot(ctx)
	held := snap.Executions[0]
	if err := s.RecordPostSendOutage(ctx, held, "supervisor", "AO_UNAVAILABLE", "matrix", time.Now()); err == nil {
		t.Fatal("stronger diagnostic overwritten")
	}
	after, _ := s.GetTaskAttempt(ctx, a.AttemptID)
	if after.RecoveryDisposition == nil || *after.RecoveryDisposition != "AO_BLOCKED_DECISION" {
		t.Fatalf("stronger hold: %+v", after)
	}
	if err := s.ObservePostSend(ctx, held, PostSendObservation{SessionID: held.SessionID, Generation: held.Generation, Activity: "active"}, "supervisor", "matrix", time.Now()); err != nil {
		t.Fatal(err)
	}
	after, _ = s.GetTaskAttempt(ctx, a.AttemptID)
	if after.RecoveryDisposition == nil || *after.RecoveryDisposition != "AO_BLOCKED_DECISION" {
		t.Fatalf("fresh GET downgraded diagnostic: %+v", after)
	}
	if err := s.ObservePostSend(ctx, held, PostSendObservation{SessionID: held.SessionID, Generation: held.Generation, Terminated: true}, "supervisor", "matrix", time.Now()); err != nil {
		t.Fatal(err)
	}
	after, _ = s.GetTaskAttempt(ctx, a.AttemptID)
	if after.EndedAt == nil || after.RecoveryDisposition == nil || *after.RecoveryDisposition != "WORKER_TERMINATION_UNKNOWN" {
		t.Fatalf("terminal observation ignored behind diagnostic: %+v", after)
	}
	// Independent fixture: terminal observation must roll back closure with its audit.
	s2, _ := createTestStore(t)
	defer s2.Close()
	_, b := setupBoundAttempt(t, s2, "task-terminal-hold", "contract-terminal-hold", "attempt-terminal-hold")
	op2 := "dispatch-attempt-terminal-hold"
	if err := s2.RecordSendRequested(ctx, op2, *b.SessionID, *b.TerminalGeneration, "idle", false, "fixture", time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := s2.RecordSendConfirmed(ctx, op2, "fixture", true, time.Now(), domain.ExecutionBudgetPolicy{Duration: time.Hour, PolicyRef: "fixture-policy"}); err != nil {
		t.Fatal(err)
	}
	snap, _ = s2.ListRecoverySnapshot(ctx)
	x = snap.Executions[0]
	if err := s2.RecordPostSendOutage(ctx, x, "supervisor", "AO_UNAVAILABLE", "run", time.Now()); err != nil {
		t.Fatal(err)
	}
	snap, _ = s2.ListRecoverySnapshot(ctx)
	held = snap.Executions[0]
	if _, err := s2.db.ExecContext(ctx, `CREATE TRIGGER fail_terminal_recovery BEFORE INSERT ON audit_events WHEN NEW.event_type='POST_SEND_STATUS_RECOVERED' BEGIN SELECT RAISE(ABORT,'injected terminal recovery audit failure'); END`); err != nil {
		t.Fatal(err)
	}
	if err := s2.ObservePostSend(ctx, held, PostSendObservation{SessionID: held.SessionID, Generation: held.Generation, Terminated: true}, "supervisor", "run", time.Now()); err == nil {
		t.Fatal("terminal audit failure accepted")
	}
	task, _ := s2.GetTask(ctx, b.TaskID)
	after, _ = s2.GetTaskAttempt(ctx, b.AttemptID)
	if task.State != domain.StateDispatched || after.EndedAt != nil || after.RecoveryDisposition == nil || *after.RecoveryDisposition != "RECOVERY_PENDING" {
		t.Fatalf("terminal rollback: %+v %+v", task, after)
	}
	if _, err := s2.db.ExecContext(ctx, `DROP TRIGGER fail_terminal_recovery`); err != nil {
		t.Fatal(err)
	}
	if err := s2.ObservePostSend(ctx, held, PostSendObservation{SessionID: held.SessionID, Generation: held.Generation, Terminated: true}, "supervisor", "run", time.Now()); err != nil {
		t.Fatal(err)
	}
	task, _ = s2.GetTask(ctx, b.TaskID)
	after, _ = s2.GetTaskAttempt(ctx, b.AttemptID)
	if task.State != domain.StateFailed || after.EndedAt == nil || after.RecoveryDisposition == nil || *after.RecoveryDisposition != "WORKER_TERMINATION_UNKNOWN" {
		t.Fatalf("terminal recovery: %+v %+v", task, after)
	}
	var recovered int
	if err := s2.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_events WHERE event_type='POST_SEND_STATUS_RECOVERED'`).Scan(&recovered); err != nil || recovered != 1 {
		t.Fatalf("terminal recovery audit=%d %v", recovered, err)
	}
}

func TestRecoverySnapshotRejectsOpenTaskWithoutBoundOperation(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()
	setupReadyTask(t, s, "task-legacy-sweep", "contract-legacy-sweep")
	path, err := CanonicalExpectedReportPath("task-legacy-sweep", "attempt-legacy-sweep")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = prepareLegacyDispatchForTest(s, ctx, "task-legacy-sweep", "contract-legacy-sweep", "attempt-legacy-sweep", path, time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err = s.ListRecoverySnapshot(ctx); !errors.Is(err, ErrInconsistentPersistedState) {
		t.Fatalf("inconsistent open attempt accepted: %v", err)
	}
}

func TestPostSendGenerationChangedQuarantinesCurrentRuntimeAndOldAttempt(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()
	pair, a := setupBoundAttempt(t, s, "task-new-gen", "contract-new-gen", "attempt-new-gen")
	op := "dispatch-attempt-new-gen"
	if err := s.RecordSendRequested(ctx, op, *a.SessionID, *a.TerminalGeneration, "idle", false, "fixture", time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordSendConfirmed(ctx, op, "fixture", true, time.Now(), domain.ExecutionBudgetPolicy{Duration: time.Hour, PolicyRef: "fixture-policy"}); err != nil {
		t.Fatal(err)
	}
	snap, _ := s.ListRecoverySnapshot(ctx)
	x := snap.Executions[0]
	if _, err := s.db.ExecContext(ctx, `UPDATE worker_sessions SET terminal_generation='new-runtime-generation' WHERE pair_id=?`, pair); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordPostSendOutage(ctx, x, "supervisor", "AO_UNAVAILABLE", "run", time.Now()); err == nil {
		t.Fatal("outage wrote hold against stale runtime")
	}
	if err := s.ObservePostSend(ctx, x, PostSendObservation{SessionID: x.SessionID, Generation: "new-runtime-generation", Activity: "active"}, "supervisor", "run", time.Now()); err != nil {
		t.Fatal(err)
	}
	task, _ := s.GetTask(ctx, a.TaskID)
	attempt, _ := s.GetTaskAttempt(ctx, a.AttemptID)
	ws, _ := s.GetWorkerSessionByPair(ctx, pair)
	if task.State != domain.StateFailed || attempt.EndedAt == nil || attempt.QuarantineState != domain.QuarantineQuarantined || ws.QuarantineState != domain.QuarantineQuarantined || attempt.TerminalGeneration == nil || *attempt.TerminalGeneration != x.Generation {
		t.Fatalf("mixed lineage containment: %+v %+v %+v", task, attempt, ws)
	}
}
