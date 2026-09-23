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

func TestD6RejectsNewPairStopRiskAndStaleEvidence(t *testing.T) {
	ctx := context.Background()
	s, old, d := administrativeStopFixture(t, true)
	defer s.Close()
	if err := s.AcceptStopAdministrativeRisk(ctx, d); err != nil {
		t.Fatal(err)
	}
	oldReq := QuarantineClearance{PairID: old.PairID, Principal: d.Principal, Session: LineageClearance{SessionID: old.SessionID, TerminalGeneration: old.TerminalGeneration, Basis: ClearanceAbsenceAdministrative, StopOperationID: old.OperationID, Evidence: "old HTTP 404"}}
	newer := old
	newer.OperationID = "stop-newer-risk"
	if err := s.ReserveStopOperation(ctx, newer); err != nil {
		t.Fatal(err)
	}
	if err := s.ClearQuarantineWithEvidence(ctx, oldReq); err == nil {
		t.Fatal("old acceptance bypassed new STOP_REQUESTED/IN_FLIGHT")
	}
	if err := s.CommitStopOutcome(ctx, newer.OperationID, StopTerminalOutcome{ExpectedStage: domain.StopRequested, Stage: domain.StopRequested, Resolution: domain.StopResolutionCallOutcomeUnknown, At: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	if err := s.ClearQuarantineWithEvidence(ctx, oldReq); err == nil {
		t.Fatal("old acceptance bypassed new unknown outcome")
	}
	newDecision := AdministrativeStopDecision{StopOperationID: newer.OperationID, PairID: newer.PairID, SessionID: newer.SessionID, TerminalGeneration: newer.TerminalGeneration, Purpose: newer.Purpose, ExpectedStage: domain.StopRequested, ExpectedResolution: domain.StopResolutionCallOutcomeUnknown, Principal: d.Principal, AuthorityScope: d.AuthorityScope, Reason: "new unknown effect risk", EvidenceReferences: []string{"new stop observation"}, Lineages: []LineageClearance{{SessionID: newer.SessionID, TerminalGeneration: newer.TerminalGeneration}}, At: time.Now().UTC()}
	if err := s.AcceptStopAdministrativeRisk(ctx, newDecision); err != nil {
		t.Fatal(err)
	}
	if err := s.ClearQuarantineWithEvidence(ctx, oldReq); err == nil {
		t.Fatal("old selected evidence cleared newer reconciled stop")
	}
	req := oldReq
	req.Session.Basis = ClearanceRiskAdministrative
	req.Session.StopOperationID = newer.OperationID
	req.Session.Evidence = "new operator acceptance"
	if _, err := s.db.ExecContext(ctx, `CREATE TRIGGER fail_new_d6_cas BEFORE UPDATE ON worker_sessions WHEN NEW.quarantine_state='CLEAN' BEGIN SELECT RAISE(ABORT,'injected new D6 CAS failure'); END`); err != nil {
		t.Fatal(err)
	}
	if err := s.ClearQuarantineWithEvidence(ctx, req); err == nil || !strings.Contains(err.Error(), "injected new D6 CAS failure") {
		t.Fatalf("D6 CAS fault=%v", err)
	}
	worker, err := s.GetWorkerSessionByPair(ctx, old.PairID)
	if err != nil || worker.QuarantineState != domain.QuarantineQuarantined {
		t.Fatalf("D6 CAS rollback released lane: %+v %v", worker, err)
	}
	if _, err := s.db.ExecContext(ctx, `DROP TRIGGER fail_new_d6_cas`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, `CREATE TRIGGER fail_new_d6 BEFORE INSERT ON audit_events WHEN NEW.event_type='QUARANTINE_RESOLVED_ADMINISTRATIVE' BEGIN SELECT RAISE(ABORT,'injected new D6 audit failure'); END`); err != nil {
		t.Fatal(err)
	}
	if err := s.ClearQuarantineWithEvidence(ctx, req); err == nil || !strings.Contains(err.Error(), "injected new D6 audit failure") {
		t.Fatalf("D6 audit fault=%v", err)
	}
	worker, err = s.GetWorkerSessionByPair(ctx, old.PairID)
	if err != nil || worker.QuarantineState != domain.QuarantineQuarantined {
		t.Fatalf("D6 rollback released lane: %+v %v", worker, err)
	}
	if _, err := s.db.ExecContext(ctx, `DROP TRIGGER fail_new_d6`); err != nil {
		t.Fatal(err)
	}
	if err := s.ClearQuarantineWithEvidence(ctx, req); err != nil {
		t.Fatalf("new reconciled evidence rejected: %v", err)
	}
}

func TestD6OldResolvedRestoreCannotBypassNewUnresolved(t *testing.T) {
	ctx := context.Background()
	s, stop, d := administrativeStopFixture(t, true)
	defer s.Close()
	if err := s.AcceptStopAdministrativeRisk(ctx, d); err != nil {
		t.Fatal(err)
	}
	rows, err := s.db.QueryContext(ctx, `SELECT event_id FROM audit_events ORDER BY sequence LIMIT 3`)
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		t.Fatal(err)
	}
	rows.Close()
	if len(ids) != 3 {
		t.Fatal("audit fixture lacks FK targets")
	}
	at := formatTime(time.Now().UTC())
	for i, op := range []string{"restore-old-resolved", "restore-new-unresolved"} {
		_, err = s.db.ExecContext(ctx, `INSERT INTO restore_authorizations(authorization_id,operation_id,pair_id,session_id,expected_generation,risk_scope,authorized_principal,authorization_event_id,issued_at) VALUES(?,?,?,?,?,'POSSIBLE_PROMPT_REPLAY','operator',?,?)`, "auth-"+op, op, stop.PairID, stop.SessionID, stop.TerminalGeneration, ids[i], at)
		if err != nil {
			t.Fatal(err)
		}
	}
	if _, err = s.db.ExecContext(ctx, `INSERT INTO pair_restore_operations(operation_id,authorization_id,pair_id,session_id,expected_generation,stage,resolution_state,resolution_basis,requested_at,resolved_at,resolution_event_id) VALUES('restore-old-resolved','auth-restore-old-resolved',?, ?, ?,'RESTORE_REQUESTED','RESTORE_RESOLVED','ADMINISTRATIVE_RISK_RESOLUTION',?,?,?)`, stop.PairID, stop.SessionID, stop.TerminalGeneration, at, at, ids[2]); err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.ExecContext(ctx, `INSERT INTO pair_restore_operations(operation_id,authorization_id,pair_id,session_id,expected_generation,stage,resolution_state,requested_at) VALUES('restore-new-unresolved','auth-restore-new-unresolved',?, ?, ?,'RESTORE_REQUESTED','IN_FLIGHT',?)`, stop.PairID, stop.SessionID, stop.TerminalGeneration, at); err != nil {
		t.Fatal(err)
	}
	req := QuarantineClearance{PairID: stop.PairID, RestoreOperationID: "restore-old-resolved", Principal: d.Principal, Session: LineageClearance{SessionID: stop.SessionID, TerminalGeneration: stop.TerminalGeneration, Basis: ClearanceAbsenceAdministrative, StopOperationID: stop.OperationID, Evidence: "old absence acceptance"}}
	if err := s.ClearQuarantineWithEvidence(ctx, req); err == nil {
		t.Fatal("historical resolved restore bypassed new unresolved restore")
	}
	worker, err := s.GetWorkerSessionByPair(ctx, stop.PairID)
	if err != nil || worker.QuarantineState != domain.QuarantineQuarantined {
		t.Fatalf("restore guard released lane: %+v %v", worker, err)
	}
}

func classBAttemptFixture(t *testing.T, live bool) (*Store, domain.StopOperation, AdministrativeStopDecision, QuarantineClearance) {
	t.Helper()
	ctx := context.Background()
	var s *Store
	var stop domain.StopOperation
	if live {
		s, stop = runningStopFixture(t)
	} else {
		var attempt domain.TaskAttempt
		var pair string
		s, _ = createTestStore(t)
		pair, attempt = setupBoundAttempt(t, s, "task-class-b-clean", "contract-class-b-clean", "attempt-class-b-clean")
		if err := s.AtomicTerminalTransition(ctx, attempt.TaskID, domain.StateDispatched, domain.StateFailed, "generation changed", attempt.AttemptID, "STALE_EXECUTION_GENERATION"); err != nil {
			t.Fatal(err)
		}
		stop = domain.StopOperation{OperationID: "stop-class-b-clean", Purpose: domain.QuarantineCleanup, PairID: pair, TaskID: &attempt.TaskID, ContractID: &attempt.ContractID, AttemptID: &attempt.AttemptID, SessionID: *attempt.SessionID, TerminalGeneration: *attempt.TerminalGeneration, Actor: "supervisor"}
		if err := s.ReserveStopOperation(ctx, stop); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.CommitStopOutcome(ctx, stop.OperationID, StopTerminalOutcome{ExpectedStage: domain.StopRequested, Stage: domain.StopTargetAbsent, Resolution: domain.StopResolutionTargetAbsent, At: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	principal := "verified-operator"
	decision := AdministrativeStopDecision{StopOperationID: stop.OperationID, PairID: stop.PairID, SessionID: stop.SessionID, TerminalGeneration: stop.TerminalGeneration, Purpose: stop.Purpose, AttemptID: *stop.AttemptID, ExpectedStage: domain.StopTargetAbsent, ExpectedResolution: domain.StopResolutionTargetAbsent, Principal: principal, AuthorityScope: "STOP_ADMINISTRATIVE_RECONCILIATION", Reason: "404 absence with residual risk", EvidenceReferences: []string{"HTTP 404"}, Lineages: []LineageClearance{{SessionID: stop.SessionID, TerminalGeneration: stop.TerminalGeneration}, {AttemptID: *stop.AttemptID, SessionID: stop.SessionID, TerminalGeneration: stop.TerminalGeneration}}, At: time.Now().UTC()}
	req := QuarantineClearance{PairID: stop.PairID, Principal: principal, Session: LineageClearance{SessionID: stop.SessionID, TerminalGeneration: stop.TerminalGeneration, Basis: ClearanceAbsenceAdministrative, StopOperationID: stop.OperationID, Evidence: "session 404 accepted"}, Attempts: []LineageClearance{{AttemptID: *stop.AttemptID, SessionID: stop.SessionID, TerminalGeneration: stop.TerminalGeneration, Basis: ClearanceAbsenceAdministrative, StopOperationID: stop.OperationID, Evidence: "attempt 404 accepted"}}}
	return s, stop, decision, req
}

func TestClassBLiveAndCleanupRequireDistinctLineageAcceptance(t *testing.T) {
	for _, purpose := range []struct {
		name string
		live bool
	}{{"live", true}, {"cleanup", false}} {
		t.Run(purpose.name, func(t *testing.T) {
			for _, missing := range []struct {
				name string
				keep int
			}{{"session_missing", 1}, {"attempt_missing", 0}} {
				t.Run(missing.name, func(t *testing.T) {
					ctx := context.Background()
					s, stop, d, req := classBAttemptFixture(t, purpose.live)
					defer s.Close()
					d.Lineages = d.Lineages[missing.keep : missing.keep+1]
					if err := s.AcceptStopAdministrativeRisk(ctx, d); err != nil {
						t.Fatal(err)
					}
					if err := s.ClearQuarantineWithEvidence(ctx, req); err == nil {
						t.Fatal("one lineage acceptance cleared both gates")
					}
					w, err := s.GetWorkerSessionByPair(ctx, stop.PairID)
					if err != nil || w.QuarantineState != domain.QuarantineQuarantined {
						t.Fatalf("session released: %+v %v", w, err)
					}
					a, err := s.GetTaskAttempt(ctx, *stop.AttemptID)
					if err != nil || a.QuarantineState != domain.QuarantineQuarantined {
						t.Fatalf("attempt released: %+v %v", a, err)
					}
				})
			}
			t.Run("both_accepted", func(t *testing.T) {
				ctx := context.Background()
				s, stop, d, req := classBAttemptFixture(t, purpose.live)
				defer s.Close()
				bad := d
				bad.Lineages = []LineageClearance{{AttemptID: "wrong-attempt", SessionID: stop.SessionID, TerminalGeneration: stop.TerminalGeneration}}
				if err := s.AcceptStopAdministrativeRisk(ctx, bad); err == nil {
					t.Fatal("wrong attempt accepted")
				}
				bad = d
				bad.Lineages = []LineageClearance{{SessionID: stop.SessionID, TerminalGeneration: "wrong-generation"}}
				if err := s.AcceptStopAdministrativeRisk(ctx, bad); err == nil {
					t.Fatal("wrong generation accepted")
				}
				if err := s.AcceptStopAdministrativeRisk(ctx, d); err != nil {
					t.Fatal(err)
				}
				if err := s.AcceptStopAdministrativeRisk(ctx, d); err != nil {
					t.Fatalf("exact replay: %v", err)
				}
				badReq := req
				badReq.Attempts = []LineageClearance{req.Attempts[0]}
				badReq.Attempts[0].TerminalGeneration = "wrong-generation"
				if err := s.ClearQuarantineWithEvidence(ctx, badReq); err == nil {
					t.Fatal("wrong snapshot cleared")
				}
				if _, err := s.db.ExecContext(ctx, `CREATE TRIGGER fail_class_b_d6 BEFORE INSERT ON audit_events WHEN NEW.event_type='QUARANTINE_RESOLVED_ADMINISTRATIVE' BEGIN SELECT RAISE(ABORT,'injected Class B D6 failure'); END`); err != nil {
					t.Fatal(err)
				}
				if err := s.ClearQuarantineWithEvidence(ctx, req); err == nil || !strings.Contains(err.Error(), "injected Class B D6 failure") {
					t.Fatalf("D6 rollback cause=%v", err)
				}
				w, err := s.GetWorkerSessionByPair(ctx, stop.PairID)
				if err != nil || w.QuarantineState != domain.QuarantineQuarantined {
					t.Fatalf("session rollback: %+v %v", w, err)
				}
				a, err := s.GetTaskAttempt(ctx, *stop.AttemptID)
				if err != nil || a.QuarantineState != domain.QuarantineQuarantined {
					t.Fatalf("attempt rollback: %+v %v", a, err)
				}
				if _, err := s.db.ExecContext(ctx, `DROP TRIGGER fail_class_b_d6`); err != nil {
					t.Fatal(err)
				}
				if err := s.ClearQuarantineWithEvidence(ctx, req); err != nil {
					t.Fatal(err)
				}
				op, err := s.GetStopOperation(ctx, stop.OperationID)
				if err != nil || op.Stage != domain.StopTargetAbsent || op.ResolutionState != domain.StopResolutionTargetAbsent {
					t.Fatalf("absence provenance changed: %+v %v", op, err)
				}
				events, err := s.ListAuditEvents(ctx, 0, MaxAuditLimit)
				if err != nil {
					t.Fatal(err)
				}
				acceptance, absence, physical := 0, 0, 0
				for _, e := range events {
					switch e.Event.EventType {
					case domain.AuditAdministrativeRiskAccepted:
						acceptance++
					case domain.AuditStopOperationTargetAbsent:
						absence++
					case domain.AuditQuarantineResolvedPhysical:
						physical++
					}
				}
				if acceptance != 2 || absence != 1 || physical != 0 {
					t.Fatalf("Class B audit types: acceptance=%d absence=%d physical=%d", acceptance, absence, physical)
				}
			})
		})
	}
}

func TestD6PreviouslyClearedStopDoesNotRequireOldPrincipal(t *testing.T) {
	ctx := context.Background()
	s, old, d := administrativeStopFixture(t, true)
	defer s.Close()
	if err := s.AcceptStopAdministrativeRisk(ctx, d); err != nil {
		t.Fatal(err)
	}
	oldClear := QuarantineClearance{PairID: old.PairID, Principal: d.Principal, Session: LineageClearance{SessionID: old.SessionID, TerminalGeneration: old.TerminalGeneration, Basis: ClearanceAbsenceAdministrative, StopOperationID: old.OperationID, Evidence: "old 404"}}
	if err := s.ClearQuarantineWithEvidence(ctx, oldClear); err != nil {
		t.Fatal(err)
	}
	newer := old
	newer.OperationID = "stop-next-operator"
	if err := s.ReserveStopOperation(ctx, newer); err != nil {
		t.Fatal(err)
	}
	if err := s.CommitStopOutcome(ctx, newer.OperationID, StopTerminalOutcome{ExpectedStage: domain.StopRequested, Stage: domain.StopCallFailed, Resolution: domain.StopResolutionCallFailed, At: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	next := AdministrativeStopDecision{StopOperationID: newer.OperationID, PairID: newer.PairID, SessionID: newer.SessionID, TerminalGeneration: newer.TerminalGeneration, Purpose: newer.Purpose, ExpectedStage: domain.StopCallFailed, ExpectedResolution: domain.StopResolutionCallFailed, Principal: "other-verified-operator", AuthorityScope: d.AuthorityScope, Reason: "new residual risk", EvidenceReferences: []string{"new case"}, Lineages: []LineageClearance{{SessionID: newer.SessionID, TerminalGeneration: newer.TerminalGeneration}}, At: time.Now().UTC()}
	if err := s.AcceptStopAdministrativeRisk(ctx, next); err != nil {
		t.Fatal(err)
	}
	req := QuarantineClearance{PairID: newer.PairID, Principal: next.Principal, Session: LineageClearance{SessionID: newer.SessionID, TerminalGeneration: newer.TerminalGeneration, Basis: ClearanceRiskAdministrative, StopOperationID: newer.OperationID, Evidence: "new accepted risk"}}
	if err := s.ClearQuarantineWithEvidence(ctx, req); err != nil {
		t.Fatalf("new fully reconciled risk blocked by historical principal: %v", err)
	}
}
