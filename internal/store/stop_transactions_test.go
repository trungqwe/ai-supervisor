package store

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/trungqwe/ai-supervisor/internal/domain"
)

func runningStopFixture(t *testing.T) (*Store, domain.StopOperation) {
	t.Helper()
	ctx := context.Background()
	s, _ := createTestStore(t)
	pair, attempt := setupBoundAttempt(t, s, "task-live-stop", "contract-live-stop", "attempt-live-stop")
	if err := s.TransitionTask(ctx, attempt.TaskID, domain.StateDispatched, domain.StateRunning); err != nil {
		t.Fatal(err)
	}
	task, contract, id := attempt.TaskID, attempt.ContractID, attempt.AttemptID
	stop := domain.StopOperation{OperationID: "stop-live", Purpose: domain.RunningAttemptStop, PairID: pair, TaskID: &task, ContractID: &contract, AttemptID: &id, SessionID: *attempt.SessionID, TerminalGeneration: *attempt.TerminalGeneration, Actor: "supervisor"}
	if err := s.ReserveStopOperation(ctx, stop); err != nil {
		t.Fatal(err)
	}
	return s, stop
}

func TestLiveStopTerminalOutcomeAtomicAndAudited(t *testing.T) {
	ctx := context.Background()
	s, stop := runningStopFixture(t)
	defer s.Close()
	call := time.Now().UTC()
	deadline := call.Add(time.Minute)
	if err := s.CommitStopCallAccepted(ctx, stop.OperationID, call, deadline); err != nil {
		t.Fatal(err)
	}
	confirmed := call.Add(500 * time.Millisecond)
	outcome := StopTerminalOutcome{ExpectedStage: domain.StopCallSucceeded, Stage: domain.StopTerminationConfirmed, Resolution: domain.StopResolutionTerminationConfirmed, At: confirmed, ObservedSessionID: stop.SessionID, ObservedGeneration: stop.TerminalGeneration, ObservedIsTerminated: true}
	if _, err := s.db.ExecContext(ctx, `CREATE TRIGGER reject_live_transition BEFORE INSERT ON audit_events WHEN NEW.event_type='TASK_STATE_TRANSITION' BEGIN SELECT RAISE(ABORT,'injected task audit failure'); END`); err != nil {
		t.Fatal(err)
	}
	if err := s.CommitStopOutcome(ctx, stop.OperationID, outcome); err == nil || !strings.Contains(err.Error(), "injected task audit failure") {
		t.Fatalf("fault injection cause=%v", err)
	}
	assertLiveStopState(t, s, stop, domain.StateRunning, domain.StopCallSucceeded, domain.StopResolutionInFlight, false, domain.QuarantineQuarantined)
	if _, err := s.db.ExecContext(ctx, `DROP TRIGGER reject_live_transition`); err != nil {
		t.Fatal(err)
	}
	if err := s.CommitStopOutcome(ctx, stop.OperationID, outcome); err != nil {
		t.Fatal(err)
	}
	assertLiveStopState(t, s, stop, domain.StateFailed, domain.StopTerminationConfirmed, domain.StopResolutionTerminationConfirmed, true, domain.QuarantineClean)
	attempt, err := s.GetTaskAttempt(ctx, *stop.AttemptID)
	if err != nil || attempt.RecoveryDisposition == nil || *attempt.RecoveryDisposition != "WORKER_STOPPED" {
		t.Fatalf("D11 disposition: %+v %v", attempt, err)
	}
	types := map[string]int{}
	events, err := s.ListAuditEvents(ctx, 0, MaxAuditLimit)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range events {
		types[e.Event.EventType]++
	}
	for _, kind := range []string{domain.AuditStopOperationRequested, domain.AuditStopOperationCallSucceeded, domain.AuditStopOperationConfirmed, domain.AuditTaskStateTransition, domain.AuditQuarantineResolvedPhysical} {
		if types[kind] == 0 {
			t.Fatalf("missing actual audit event_type %s: %v", kind, types)
		}
	}
	if err := s.CommitStopOutcome(ctx, stop.OperationID, outcome); !errors.Is(err, ErrStateConflict) {
		t.Fatalf("stale confirmation won: %v", err)
	}
}

func assertLiveStopState(t *testing.T, s *Store, stop domain.StopOperation, wantTask domain.TaskState, wantStage domain.StopStage, wantResolution domain.StopResolutionState, closed bool, wantQ domain.QuarantineState) {
	t.Helper()
	ctx := context.Background()
	task, err := s.GetTask(ctx, *stop.TaskID)
	if err != nil || task.State != wantTask {
		t.Fatalf("task state: %+v %v", task, err)
	}
	attempt, err := s.GetTaskAttempt(ctx, *stop.AttemptID)
	if err != nil || ((attempt.EndedAt != nil) != closed) || attempt.QuarantineState != wantQ {
		t.Fatalf("attempt state: %+v %v", attempt, err)
	}
	op, err := s.GetStopOperation(ctx, stop.OperationID)
	if err != nil || op.Stage != wantStage || op.ResolutionState != wantResolution {
		t.Fatalf("stop state: %+v %v", op, err)
	}
}

func TestLiveStopStrictNanosecondDeadline(t *testing.T) {
	for _, tc := range []struct {
		name  string
		delta time.Duration
		valid bool
	}{{"before_1ns", -time.Nanosecond, true}, {"equal", 0, false}, {"after_1ns", time.Nanosecond, false}} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			s, stop := runningStopFixture(t)
			defer s.Close()
			call := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
			deadline := call.Add(500 * time.Millisecond)
			if err := s.CommitStopCallAccepted(ctx, stop.OperationID, call, deadline); err != nil {
				t.Fatal(err)
			}
			outcome := StopTerminalOutcome{ExpectedStage: domain.StopCallSucceeded, Stage: domain.StopTerminationConfirmed, Resolution: domain.StopResolutionTerminationConfirmed, At: deadline.Add(tc.delta), ObservedSessionID: stop.SessionID, ObservedGeneration: stop.TerminalGeneration, ObservedIsTerminated: true}
			err := s.CommitStopOutcome(ctx, stop.OperationID, outcome)
			if (err == nil) != tc.valid {
				t.Fatalf("strict Before result: %v", err)
			}
			if tc.valid {
				assertLiveStopState(t, s, stop, domain.StateFailed, domain.StopTerminationConfirmed, domain.StopResolutionTerminationConfirmed, true, domain.QuarantineClean)
			} else {
				assertLiveStopState(t, s, stop, domain.StateRunning, domain.StopCallSucceeded, domain.StopResolutionInFlight, false, domain.QuarantineQuarantined)
			}
		})
	}
}

func TestGenericStopStageCannotSplitLiveClosure(t *testing.T) {
	ctx := context.Background()
	s, stop := runningStopFixture(t)
	defer s.Close()
	call := time.Now().UTC()
	deadline := call.Add(time.Minute)
	if err := s.CommitStopCallAccepted(ctx, stop.OperationID, call, deadline); err != nil {
		t.Fatal(err)
	}
	resolution := domain.StopResolutionTerminationConfirmed
	confirmed := call.Add(time.Second)
	err := s.UpdateStopOperationStage(ctx, stop.OperationID, domain.StopCallSucceeded, domain.StopTerminationConfirmed, StopStageUpdate{ResolutionState: &resolution, TerminationConfirmedAt: &confirmed, ResolvedAt: &confirmed, ObservedSessionID: stop.SessionID, ObservedGeneration: stop.TerminalGeneration, ObservedIsTerminated: true})
	if err != nil {
		t.Fatalf("generic update failed to delegate atomic closure: %v", err)
	}
	assertLiveStopState(t, s, stop, domain.StateFailed, domain.StopTerminationConfirmed, domain.StopResolutionTerminationConfirmed, true, domain.QuarantineClean)
}

func TestLiveStopNonPhysicalPurposeMatrix(t *testing.T) {
	for _, tc := range []struct {
		name               string
		afterHTTP200       bool
		stage              domain.StopStage
		resolution         domain.StopResolutionState
		disposition, audit string
	}{
		{"404", false, domain.StopTargetAbsent, domain.StopResolutionTargetAbsent, "SESSION_ABSENT", domain.AuditStopOperationTargetAbsent},
		{"generation mismatch", false, domain.StopRequested, domain.StopResolutionGenerationMismatch, "STOP_GENERATION_MISMATCH", ""},
		{"already terminated", false, domain.StopRequested, domain.StopResolutionEffectUnprovenAlreadyTerminated, "WORKER_TERMINATION_UNKNOWN", ""},
		{"timeout", true, domain.StopCallSucceeded, domain.StopResolutionConfirmationTimeout, "STOP_CONFIRMATION_TIMEOUT", domain.AuditStopConfirmationTimeout},
		{"definite failure", false, domain.StopCallFailed, domain.StopResolutionCallFailed, "WORKER_TERMINATION_UNKNOWN", domain.AuditStopOperationCallFailed},
		{"ambiguous call", false, domain.StopRequested, domain.StopResolutionCallOutcomeUnknown, "STOP_CALL_OUTCOME_UNKNOWN", domain.AuditStopOperationCallOutcomeUnknown},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			s, stop := runningStopFixture(t)
			defer s.Close()
			expected := domain.StopRequested
			if tc.afterHTTP200 {
				call := time.Now().UTC()
				if err := s.CommitStopCallAccepted(ctx, stop.OperationID, call, call.Add(time.Minute)); err != nil {
					t.Fatal(err)
				}
				expected = domain.StopCallSucceeded
			}
			outcome := StopTerminalOutcome{ExpectedStage: expected, Stage: tc.stage, Resolution: tc.resolution, At: time.Now().UTC()}
			if err := s.CommitStopOutcome(ctx, stop.OperationID, outcome); err != nil {
				t.Fatal(err)
			}
			assertLiveStopState(t, s, stop, domain.StateFailed, tc.stage, tc.resolution, true, domain.QuarantineQuarantined)
			attempt, err := s.GetTaskAttempt(ctx, *stop.AttemptID)
			if err != nil || attempt.RecoveryDisposition == nil || *attempt.RecoveryDisposition != tc.disposition {
				t.Fatalf("disposition: %+v %v", attempt, err)
			}
			worker, err := s.GetWorkerSessionByPair(ctx, stop.PairID)
			if err != nil || worker.QuarantineState != domain.QuarantineQuarantined {
				t.Fatalf("Pair risk released: %+v %v", worker, err)
			}
			events, err := s.ListAuditEvents(ctx, 0, MaxAuditLimit)
			if err != nil {
				t.Fatal(err)
			}
			types := map[string]int{}
			for _, e := range events {
				types[e.Event.EventType]++
			}
			if types[domain.AuditTaskStateTransition] != 1 || tc.audit != "" && types[tc.audit] != 1 {
				t.Fatalf("actual event_type mismatch: %v", types)
			}
		})
	}
}

func administrativeStopFixture(t *testing.T, absent bool) (*Store, domain.StopOperation, AdministrativeStopDecision) {
	t.Helper()
	ctx := context.Background()
	s, _ := createTestStore(t)
	setupReadyTask(t, s, "task-admin-stop", "contract-admin-stop")
	pair, session, generation := "pair-d-task-admin-stop", "session-admin-stop", "generation-admin-stop"
	seedWorkerSessionForTest(t, s, domain.WorkerSession{PairID: pair, SessionID: session, RuntimeType: "agy_tui", WorkerAgentID: "agy", Status: domain.WorkerSessionTerminated, TerminalGeneration: generation, QuarantineState: domain.QuarantineQuarantined})
	stop := domain.StopOperation{OperationID: "stop-admin", Purpose: domain.PairMaintenance, PairID: pair, SessionID: session, TerminalGeneration: generation, Actor: "operator"}
	if err := s.ReserveStopOperation(ctx, stop); err != nil {
		t.Fatal(err)
	}
	outcome := StopTerminalOutcome{ExpectedStage: domain.StopRequested, Stage: domain.StopCallFailed, Resolution: domain.StopResolutionCallFailed, At: time.Now().UTC()}
	if absent {
		outcome.Stage = domain.StopTargetAbsent
		outcome.Resolution = domain.StopResolutionTargetAbsent
	}
	if err := s.CommitStopOutcome(ctx, stop.OperationID, outcome); err != nil {
		t.Fatal(err)
	}
	decision := AdministrativeStopDecision{StopOperationID: stop.OperationID, PairID: pair, SessionID: session, TerminalGeneration: generation, Purpose: domain.PairMaintenance, ExpectedStage: outcome.Stage, ExpectedResolution: outcome.Resolution, Principal: "operator", AuthorityScope: "STOP_ADMINISTRATIVE_RECONCILIATION", Reason: "residual execution risk", EvidenceReferences: []string{"operator case 1"}, Lineages: []LineageClearance{{SessionID: session, TerminalGeneration: generation}}, At: time.Now().UTC()}
	return s, stop, decision
}

func TestAdministrativeStopClassBCASReplayAndAudit(t *testing.T) {
	ctx := context.Background()
	s, stop, d := administrativeStopFixture(t, true)
	defer s.Close()
	bad := d
	bad.AuthorityScope = "RESTORE"
	if err := s.AcceptStopAdministrativeRisk(ctx, bad); err == nil {
		t.Fatal("wrong scope admitted")
	}
	bad = d
	bad.TerminalGeneration = "wrong"
	if err := s.AcceptStopAdministrativeRisk(ctx, bad); err == nil {
		t.Fatal("wrong generation admitted")
	}
	bad = d
	bad.Lineages = []LineageClearance{{SessionID: d.SessionID, TerminalGeneration: "wrong"}}
	if err := s.AcceptStopAdministrativeRisk(ctx, bad); err == nil {
		t.Fatal("wrong lineage admitted")
	}
	if err := s.AcceptStopAdministrativeRisk(ctx, d); err != nil {
		t.Fatal(err)
	}
	if err := s.AcceptStopAdministrativeRisk(ctx, d); err != nil {
		t.Fatalf("exact replay: %v", err)
	}
	bad = d
	bad.Reason = "different decision"
	if err := s.AcceptStopAdministrativeRisk(ctx, bad); err == nil {
		t.Fatal("conflicting replay admitted")
	}
	got, err := s.GetStopOperation(ctx, stop.OperationID)
	if err != nil || got.Stage != domain.StopTargetAbsent || got.ResolutionState != domain.StopResolutionTargetAbsent {
		t.Fatalf("absence provenance lost: %+v %v", got, err)
	}
	worker, err := s.GetWorkerSessionByPair(ctx, stop.PairID)
	if err != nil || worker.QuarantineState != domain.QuarantineQuarantined {
		t.Fatalf("acceptance cleared quarantine: %+v %v", worker, err)
	}
	events, err := s.ListAuditEvents(ctx, 0, MaxAuditLimit)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, e := range events {
		if e.Event.EventType == domain.AuditAdministrativeRiskAccepted {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("acceptance event_type count=%d", count)
	}
}

func TestAdministrativeStopClassCRollbackAndWireProvenance(t *testing.T) {
	ctx := context.Background()
	s, stop, d := administrativeStopFixture(t, false)
	defer s.Close()
	if _, err := s.db.ExecContext(ctx, `CREATE TRIGGER fail_admin_audit BEFORE INSERT ON audit_events WHEN NEW.event_type='ADMINISTRATIVE_RISK_ACCEPTED' BEGIN SELECT RAISE(ABORT,'injected acceptance audit failure'); END`); err != nil {
		t.Fatal(err)
	}
	if err := s.AcceptStopAdministrativeRisk(ctx, d); err == nil || !strings.Contains(err.Error(), "injected acceptance audit failure") {
		t.Fatalf("audit fault=%v", err)
	}
	got, err := s.GetStopOperation(ctx, stop.OperationID)
	if err != nil || got.ResolutionState != domain.StopResolutionCallFailed || got.Stage != domain.StopCallFailed {
		t.Fatalf("CAS rollback failed: %+v %v", got, err)
	}
	if _, err := s.db.ExecContext(ctx, `DROP TRIGGER fail_admin_audit`); err != nil {
		t.Fatal(err)
	}
	if err := s.AcceptStopAdministrativeRisk(ctx, d); err != nil {
		t.Fatal(err)
	}
	got, err = s.GetStopOperation(ctx, stop.OperationID)
	if err != nil || got.Stage != domain.StopCallFailed || got.ResolutionState != domain.StopResolutionAdministrativeRiskAccepted || got.ResolvedAt == nil {
		t.Fatalf("wire provenance/state: %+v %v", got, err)
	}
	stale := d
	stale.ExpectedResolution = domain.StopResolutionConfirmationTimeout
	if err := s.AcceptStopAdministrativeRisk(ctx, stale); err == nil {
		t.Fatal("stale writer admitted")
	}
	if err := s.AcceptStopAdministrativeRisk(ctx, d); err != nil {
		t.Fatalf("Class C replay: %v", err)
	}
	worker, err := s.GetWorkerSessionByPair(ctx, stop.PairID)
	if err != nil || worker.QuarantineState != domain.QuarantineQuarantined {
		t.Fatalf("administrative stop cleared quarantine: %+v %v", worker, err)
	}
}

func TestAdministrativeStopCompetingWritersOneDecision(t *testing.T) {
	ctx := context.Background()
	s, stop, d := administrativeStopFixture(t, false)
	defer s.Close()
	start := make(chan struct{})
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for _, reason := range []string{"risk case A", "risk case B"} {
		candidate := d
		candidate.Reason = reason
		wg.Add(1)
		go func() { defer wg.Done(); <-start; results <- s.AcceptStopAdministrativeRisk(ctx, candidate) }()
	}
	close(start)
	wg.Wait()
	close(results)
	success := 0
	for err := range results {
		if err == nil {
			success++
		}
	}
	if success != 1 {
		t.Fatalf("competing administrative writers success=%d", success)
	}
	op, err := s.GetStopOperation(ctx, stop.OperationID)
	if err != nil || op.ResolutionState != domain.StopResolutionAdministrativeRiskAccepted {
		t.Fatalf("winning resolution lost: %+v %v", op, err)
	}
	events, err := s.ListAuditEvents(ctx, 0, MaxAuditLimit)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, e := range events {
		if e.Event.EventType == domain.AuditAdministrativeRiskAccepted {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("competing acceptance events=%d", count)
	}
}

func TestLiveStopTerminalFaultBoundariesRollbackTogether(t *testing.T) {
	probes := []struct{ name, create string }{
		{"stop_CAS", `CREATE TRIGGER fail_live_stop BEFORE UPDATE ON stop_operations WHEN NEW.stage='STOP_TERMINATION_CONFIRMED' BEGIN SELECT RAISE(ABORT,'injected stop CAS failure'); END`},
		{"attempt_CAS", `CREATE TRIGGER fail_live_stop BEFORE UPDATE ON task_attempts WHEN NEW.ended_at IS NOT NULL BEGIN SELECT RAISE(ABORT,'injected attempt CAS failure'); END`},
		{"session_CAS", `CREATE TRIGGER fail_live_stop BEFORE UPDATE ON worker_sessions WHEN NEW.status='TERMINATED' BEGIN SELECT RAISE(ABORT,'injected session CAS failure'); END`},
		{"stop_audit", `CREATE TRIGGER fail_live_stop BEFORE INSERT ON audit_events WHEN NEW.event_type='STOP_OPERATION_CONFIRMED' BEGIN SELECT RAISE(ABORT,'injected stop audit failure'); END`},
		{"clearance_audit", `CREATE TRIGGER fail_live_stop BEFORE INSERT ON audit_events WHEN NEW.event_type='QUARANTINE_RESOLVED_PHYSICAL' BEGIN SELECT RAISE(ABORT,'injected clearance audit failure'); END`},
	}
	for _, probe := range probes {
		t.Run(probe.name, func(t *testing.T) {
			ctx := context.Background()
			s, stop := runningStopFixture(t)
			defer s.Close()
			call := time.Now().UTC()
			if err := s.CommitStopCallAccepted(ctx, stop.OperationID, call, call.Add(time.Minute)); err != nil {
				t.Fatal(err)
			}
			outcome := StopTerminalOutcome{ExpectedStage: domain.StopCallSucceeded, Stage: domain.StopTerminationConfirmed, Resolution: domain.StopResolutionTerminationConfirmed, At: call.Add(time.Second), ObservedSessionID: stop.SessionID, ObservedGeneration: stop.TerminalGeneration, ObservedIsTerminated: true}
			if _, err := s.db.ExecContext(ctx, probe.create); err != nil {
				t.Fatal(err)
			}
			if err := s.CommitStopOutcome(ctx, stop.OperationID, outcome); err == nil || !strings.Contains(err.Error(), "injected") {
				t.Fatalf("fault cause=%v", err)
			}
			assertLiveStopState(t, s, stop, domain.StateRunning, domain.StopCallSucceeded, domain.StopResolutionInFlight, false, domain.QuarantineQuarantined)
			events, err := s.ListAuditEvents(ctx, 0, MaxAuditLimit)
			if err != nil {
				t.Fatal(err)
			}
			for _, e := range events {
				if e.Event.EventType == domain.AuditStopOperationConfirmed || e.Event.EventType == domain.AuditQuarantineResolvedPhysical {
					t.Fatalf("partial terminal audit survived: %s", e.Event.EventType)
				}
			}
			if _, err := s.db.ExecContext(ctx, `DROP TRIGGER fail_live_stop`); err != nil {
				t.Fatal(err)
			}
			if err := s.CommitStopOutcome(ctx, stop.OperationID, outcome); err != nil {
				t.Fatalf("positive control: %v", err)
			}
			assertLiveStopState(t, s, stop, domain.StateFailed, domain.StopTerminationConfirmed, domain.StopResolutionTerminationConfirmed, true, domain.QuarantineClean)
		})
	}
}

func TestGenericStopAPICannotPersistUnboundCallMetadata(t *testing.T) {
	ctx := context.Background()
	s, stop, _ := administrativeStopFixture(t, false)
	defer s.Close()
	now := time.Now().UTC()
	deadline := now.Add(time.Minute)
	if err := s.UpdateStopOperationStage(ctx, stop.OperationID, domain.StopCallFailed, domain.StopCallFailed, StopStageUpdate{CallCompletedAt: &now, ConfirmationDeadlineAt: &deadline}); err == nil {
		t.Fatal("generic API changed resolved stop metadata")
	}
	got, err := s.GetStopOperation(ctx, stop.OperationID)
	if err != nil || got.CallCompletedAt != nil || got.ConfirmationDeadlineAt != nil || got.ResolutionState != domain.StopResolutionCallFailed {
		t.Fatalf("metadata bypass: %+v %v", got, err)
	}
}
