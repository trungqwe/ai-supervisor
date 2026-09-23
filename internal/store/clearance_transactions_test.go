package store

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/trungqwe/ai-supervisor/internal/domain"
)

func TestNoAttemptClassBClearanceAndAuditRollback(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()
	setupReadyTask(t, s, "task-clear-none", "contract-clear-none")
	pair := "pair-d-task-clear-none"
	session := "session-clear-none"
	generation := "generation-clear-none"
	seedWorkerSessionForTest(t, s, domain.WorkerSession{PairID: pair, SessionID: session, RuntimeType: "agy_tui", WorkerAgentID: "agy", Status: domain.WorkerSessionTerminated, TerminalGeneration: generation, QuarantineState: domain.QuarantineQuarantined})
	stop := domain.StopOperation{OperationID: "stop-clear-404", Purpose: domain.PairMaintenance, PairID: pair, SessionID: session, TerminalGeneration: generation, Actor: "verified-principal"}
	if err := s.ReserveStopOperation(ctx, stop); err != nil {
		t.Fatal(err)
	}
	if err := s.CommitStopOutcome(ctx, stop.OperationID, StopTerminalOutcome{ExpectedStage: domain.StopRequested, Stage: domain.StopTargetAbsent, Resolution: domain.StopResolutionTargetAbsent, At: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	if err := s.AcceptStopAdministrativeRisk(ctx, AdministrativeStopDecision{StopOperationID: stop.OperationID, PairID: pair, SessionID: session, TerminalGeneration: generation, Purpose: domain.PairMaintenance, ExpectedStage: domain.StopTargetAbsent, ExpectedResolution: domain.StopResolutionTargetAbsent, Principal: "verified-principal", AuthorityScope: "STOP_ADMINISTRATIVE_RECONCILIATION", Reason: "404 absence risk", EvidenceReferences: []string{"HTTP 404"}, Lineages: []LineageClearance{{SessionID: session, TerminalGeneration: generation}}, At: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	req := QuarantineClearance{PairID: pair, Principal: "verified-principal", Session: LineageClearance{SessionID: session, TerminalGeneration: generation, Basis: ClearanceAbsenceAdministrative, StopOperationID: stop.OperationID, Evidence: "HTTP 404 and operator accepted absence risk"}}
	if _, err := s.db.ExecContext(ctx, `CREATE TRIGGER reject_d6_audit BEFORE INSERT ON audit_events WHEN NEW.event_type='QUARANTINE_RESOLVED_ADMINISTRATIVE' BEGIN SELECT RAISE(ABORT,'injected D6 audit failure'); END`); err != nil {
		t.Fatal(err)
	}
	if err := s.ClearQuarantineWithEvidence(ctx, req); err == nil || !strings.Contains(err.Error(), "injected D6 audit failure") {
		t.Fatalf("D6 fault cause=%v", err)
	}
	worker, err := s.GetWorkerSessionByPair(ctx, pair)
	if err != nil || worker.QuarantineState != domain.QuarantineQuarantined {
		t.Fatalf("D6 rollback released lane: %+v %v", worker, err)
	}
	if _, err := s.db.ExecContext(ctx, `DROP TRIGGER reject_d6_audit`); err != nil {
		t.Fatal(err)
	}
	if err := s.ClearQuarantineWithEvidence(ctx, req); err != nil {
		t.Fatal(err)
	}
	worker, err = s.GetWorkerSessionByPair(ctx, pair)
	if err != nil || worker.QuarantineState != domain.QuarantineClean {
		t.Fatalf("D6 did not clear: %+v %v", worker, err)
	}
	events, err := s.ListAuditEvents(ctx, 0, MaxAuditLimit)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range events {
		if e.Event.EventType == domain.AuditQuarantineResolvedAdministrative {
			found = true
		}
	}
	if !found {
		t.Fatal("administrative audit event_type absent")
	}
}

func TestMixedLineageD6RequiresEveryBasis(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()
	pair, attempt := setupBoundAttempt(t, s, "task-clear-mixed", "contract-clear-mixed", "attempt-clear-mixed")
	if err := s.AtomicTerminalTransition(ctx, attempt.TaskID, domain.StateDispatched, domain.StateFailed, "generation changed", attempt.AttemptID, "STALE_EXECUTION_GENERATION"); err != nil {
		t.Fatal(err)
	}
	oldStop := domain.StopOperation{OperationID: "stop-old-attempt", Purpose: domain.QuarantineCleanup, PairID: pair, TaskID: &attempt.TaskID, ContractID: &attempt.ContractID, AttemptID: &attempt.AttemptID, SessionID: *attempt.SessionID, TerminalGeneration: *attempt.TerminalGeneration, Actor: "verified-principal"}
	if err := s.ReserveStopOperation(ctx, oldStop); err != nil {
		t.Fatal(err)
	}
	if err := s.CommitStopOutcome(ctx, oldStop.OperationID, StopTerminalOutcome{ExpectedStage: domain.StopRequested, Stage: domain.StopCallFailed, Resolution: domain.StopResolutionCallFailed, At: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	if err := s.AcceptStopAdministrativeRisk(ctx, AdministrativeStopDecision{StopOperationID: oldStop.OperationID, PairID: pair, SessionID: oldStop.SessionID, TerminalGeneration: oldStop.TerminalGeneration, Purpose: domain.QuarantineCleanup, AttemptID: attempt.AttemptID, ExpectedStage: domain.StopCallFailed, ExpectedResolution: domain.StopResolutionCallFailed, Principal: "verified-principal", AuthorityScope: "STOP_ADMINISTRATIVE_RECONCILIATION", Reason: "old execution cannot be proven stopped", EvidenceReferences: []string{"failed stop call"}, Lineages: []LineageClearance{{AttemptID: attempt.AttemptID, SessionID: oldStop.SessionID, TerminalGeneration: oldStop.TerminalGeneration}}, At: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	// Fixture: a restored runtime has a new generation while the old attempt
	// retains its immutable snapshot and quarantine.
	if _, err := s.db.ExecContext(ctx, `UPDATE worker_sessions SET terminal_generation='generation-new',status='ACTIVE',quarantine_state='QUARANTINED' WHERE pair_id=?`, pair); err != nil {
		t.Fatal(err)
	}
	stop := domain.StopOperation{OperationID: "stop-new-runtime", Purpose: domain.PairMaintenance, PairID: pair, SessionID: *attempt.SessionID, TerminalGeneration: "generation-new", Actor: "verified-principal"}
	if err := s.ReserveStopOperation(ctx, stop); err != nil {
		t.Fatal(err)
	}
	call := time.Now().UTC()
	deadline := call.Add(time.Minute)
	if err := s.CommitStopCallAccepted(ctx, stop.OperationID, call, deadline); err != nil {
		t.Fatal(err)
	}
	if err := s.CommitStopOutcome(ctx, stop.OperationID, StopTerminalOutcome{ExpectedStage: domain.StopCallSucceeded, Stage: domain.StopTerminationConfirmed, Resolution: domain.StopResolutionTerminationConfirmed, At: call.Add(time.Second), ObservedSessionID: stop.SessionID, ObservedGeneration: stop.TerminalGeneration, ObservedIsTerminated: true}); err != nil {
		t.Fatal(err)
	}
	req := QuarantineClearance{PairID: pair, Principal: "verified-principal", Session: LineageClearance{SessionID: stop.SessionID, TerminalGeneration: stop.TerminalGeneration, Basis: ClearancePhysical, StopOperationID: stop.OperationID, Evidence: "new runtime D11 proof"}}
	if err := s.ClearQuarantineWithEvidence(ctx, req); !errors.Is(err, ErrAttemptLineageMismatch) {
		t.Fatalf("missing old lineage admitted: %v", err)
	}
	old, err := s.GetTaskAttempt(ctx, attempt.AttemptID)
	if err != nil || old.QuarantineState != domain.QuarantineQuarantined {
		t.Fatalf("old lineage released by new proof: %+v %v", old, err)
	}
	req.Attempts = []LineageClearance{{AttemptID: attempt.AttemptID, SessionID: *attempt.SessionID, TerminalGeneration: *attempt.TerminalGeneration, Basis: ClearancePhysical, StopOperationID: stop.OperationID, Evidence: "wrong-generation proof"}}
	if err := s.ClearQuarantineWithEvidence(ctx, req); !errors.Is(err, ErrAttemptLineageMismatch) {
		t.Fatalf("new stop cleared old attempt: %v", err)
	}
	req.Attempts[0].Basis = ClearanceRiskAdministrative
	req.Attempts[0].StopOperationID = oldStop.OperationID
	req.Attempts[0].Evidence = "operator explicitly accepts unresolved old execution risk"
	if _, err := s.db.ExecContext(ctx, `CREATE TRIGGER fail_mixed_d6 BEFORE INSERT ON audit_events WHEN NEW.event_type='QUARANTINE_RESOLVED_PHYSICAL' BEGIN SELECT RAISE(ABORT,'injected mixed D6 failure'); END`); err != nil {
		t.Fatal(err)
	}
	if err := s.ClearQuarantineWithEvidence(ctx, req); err == nil || !strings.Contains(err.Error(), "injected mixed D6 failure") {
		t.Fatalf("mixed D6 fault=%v", err)
	}
	stillOld, err := s.GetTaskAttempt(ctx, attempt.AttemptID)
	if err != nil || stillOld.QuarantineState != domain.QuarantineQuarantined {
		t.Fatalf("mixed rollback released attempt: %+v %v", stillOld, err)
	}
	stillWorker, err := s.GetWorkerSessionByPair(ctx, pair)
	if err != nil || stillWorker.QuarantineState != domain.QuarantineQuarantined {
		t.Fatalf("mixed rollback released WorkerSession: %+v %v", stillWorker, err)
	}
	if _, err := s.db.ExecContext(ctx, `DROP TRIGGER fail_mixed_d6`); err != nil {
		t.Fatal(err)
	}
	if err := s.ClearQuarantineWithEvidence(ctx, req); err != nil {
		t.Fatal(err)
	}
	old, err = s.GetTaskAttempt(ctx, attempt.AttemptID)
	if err != nil || old.QuarantineState != domain.QuarantineClean {
		t.Fatalf("old lineage not cleared by Class C: %+v %v", old, err)
	}
	worker, err := s.GetWorkerSessionByPair(ctx, pair)
	if err != nil || worker.QuarantineState != domain.QuarantineClean {
		t.Fatalf("runtime not cleared by Class A: %+v %v", worker, err)
	}
	events, err := s.ListAuditEvents(ctx, 0, MaxAuditLimit)
	if err != nil {
		t.Fatal(err)
	}
	types := map[string]int{}
	for _, e := range events {
		types[e.Event.EventType]++
	}
	if types[domain.AuditQuarantineResolvedPhysical] != 1 || types[domain.AuditQuarantineResolvedAdministrative] != 1 {
		t.Fatalf("mixed audit basis collapsed: %v", types)
	}
}
