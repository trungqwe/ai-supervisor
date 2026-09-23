package stop

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/trungqwe/ai-supervisor/internal/ao"
	"github.com/trungqwe/ai-supervisor/internal/domain"
	"github.com/trungqwe/ai-supervisor/internal/store"
)

type testOperator struct {
	principal string
	allowed   bool
}

type scopeOperator struct{ principal, allowedScope string }

func (o scopeOperator) VerifiedRestorePrincipal(_ context.Context, _, _, _, scope string) (string, bool, error) {
	return o.principal, scope == o.allowedScope, nil
}

func (o testOperator) VerifiedRestorePrincipal(context.Context, string, string, string, string) (string, bool, error) {
	return o.principal, o.allowed, nil
}

type testAO struct {
	mu                     sync.Mutex
	statusCalls, killCalls int
	stopErr                error
	postTerminated         bool
	onKill                 func()
}

func (f *testAO) GetWorkerStatus(_ context.Context, id string) (*ao.WorkerStatus, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.statusCalls++
	return &ao.WorkerStatus{ID: id, TerminalGeneration: "generation-stop", IsTerminated: f.postTerminated && f.killCalls > 0}, nil
}
func (f *testAO) StopWorker(_ context.Context, id string) (*ao.StopWorkerResult, error) {
	f.mu.Lock()
	f.killCalls++
	onKill := f.onKill
	err := f.stopErr
	f.mu.Unlock()
	if onKill != nil {
		onKill()
	}
	if err != nil {
		return nil, err
	}
	return &ao.StopWorkerResult{SessionID: id, Freed: true}, nil
}
func (f *testAO) calls() int { f.mu.Lock(); defer f.mu.Unlock(); return f.killCalls }

func stopFixture(t *testing.T) (*store.Store, domain.StopOperation) {
	t.Helper()
	ctx := context.Background()
	s, err := store.Open(ctx, store.Config{DBPath: filepath.Join(t.TempDir(), "stop.db"), BusyTimeoutMs: 5000})
	if err != nil {
		t.Fatal(err)
	}
	pair := "pair-stop"
	project := "project-stop"
	session := "session-stop"
	if err = s.CreateProject(ctx, domain.Project{ProjectID: project, Name: project, RootPath: "/" + project}); err != nil {
		t.Fatal(err)
	}
	if err = s.CreatePair(ctx, domain.Pair{PairID: pair, ProjectID: project, CurrentPhaseID: "P03", State: "ACTIVE"}); err != nil {
		t.Fatal(err)
	}
	provision := domain.PairProvisioningOperation{OperationID: "provision-stop", PairID: pair, ClientToken: "token-stop"}
	if err = s.ReservePairProvisioning(ctx, provision, "supervisor"); err != nil {
		t.Fatal(err)
	}
	if err = s.ConfirmPairProvisioning(ctx, provision.OperationID, domain.WorkerSession{PairID: pair, SessionID: session, RuntimeType: "agy_tui", WorkerAgentID: "agy", Status: domain.WorkerSessionIdle, TerminalGeneration: "generation-stop", QuarantineState: domain.QuarantineClean}, "supervisor", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	return s, domain.StopOperation{OperationID: "stop-operation", Purpose: domain.PairMaintenance, PairID: pair, SessionID: session, TerminalGeneration: "generation-stop", Actor: "supervisor"}
}

func TestNewIntentAloneGrantsOneEffect(t *testing.T) {
	ctx := context.Background()
	s, op := stopFixture(t)
	defer s.Close()
	fake := &testAO{postTerminated: true}
	fake.onKill = func() {
		got, err := s.GetStopOperation(ctx, op.OperationID)
		if err != nil || got.Stage != domain.StopRequested || got.ResolutionState != domain.StopResolutionInFlight {
			t.Errorf("effect preceded durable intent: %+v %v", got, err)
		}
	}
	c := Coordinator{Store: s, AO: fake, Operator: testOperator{principal: "verified-subject", allowed: true}, KillTimeout: time.Minute}
	if err := c.Start(ctx, op); err != nil {
		t.Fatal(err)
	}
	if fake.calls() != 1 {
		t.Fatalf("kill calls=%d", fake.calls())
	}
	got, err := s.GetStopOperation(ctx, op.OperationID)
	if err != nil || got.Stage != domain.StopTerminationConfirmed || got.ResolutionState != domain.StopResolutionTerminationConfirmed {
		t.Fatalf("stop outcome: %+v %v", got, err)
	}
	if err := c.Start(ctx, op); err == nil || fake.calls() != 1 {
		t.Fatalf("replay effect: %v calls=%d", err, fake.calls())
	}
	other := op
	other.OperationID = "stop-other"
	if err := c.Start(ctx, other); err != nil || fake.calls() != 1 {
		t.Fatalf("already terminated runtime caused another effect: %v calls=%d", err, fake.calls())
	}
}

func TestMissingAuthorityOrTimeoutLeavesNoIntentOrEffect(t *testing.T) {
	ctx := context.Background()
	s, op := stopFixture(t)
	defer s.Close()
	fake := &testAO{}
	c := Coordinator{Store: s, AO: fake, Operator: testOperator{principal: "fake", allowed: false}, KillTimeout: time.Minute}
	if err := c.Start(ctx, op); err == nil {
		t.Fatal("missing authority admitted")
	}
	c.Operator = testOperator{principal: "fake", allowed: true}
	c.KillTimeout = 0
	if err := c.Start(ctx, op); err == nil {
		t.Fatal("missing injected timeout admitted")
	}
	if _, err := s.GetStopOperation(ctx, op.OperationID); !errors.Is(err, store.ErrOperationNotFound) {
		t.Fatalf("unexpected intent: %v", err)
	}
	if fake.calls() != 0 {
		t.Fatalf("effect calls=%d", fake.calls())
	}
}

func TestAmbiguousEffectAndReplayStayQuarantined(t *testing.T) {
	ctx := context.Background()
	s, op := stopFixture(t)
	defer s.Close()
	fake := &testAO{stopErr: errors.New("lost response")}
	c := Coordinator{Store: s, AO: fake, Operator: testOperator{principal: "verified-subject", allowed: true}, KillTimeout: time.Minute}
	if err := c.Start(ctx, op); err == nil {
		t.Fatal("ambiguous stop reported success")
	}
	got, err := s.GetStopOperation(ctx, op.OperationID)
	if err != nil || got.Stage != domain.StopRequested || got.ResolutionState != domain.StopResolutionCallOutcomeUnknown {
		t.Fatalf("unknown durable result: %+v %v", got, err)
	}
	session, err := s.GetWorkerSessionByPair(ctx, op.PairID)
	if err != nil || session.QuarantineState != domain.QuarantineQuarantined {
		t.Fatalf("unknown cleared quarantine: %+v %v", session, err)
	}
	if err := c.Start(ctx, op); err == nil || fake.calls() != 1 {
		t.Fatalf("ambiguous effect replayed: err=%v calls=%d", err, fake.calls())
	}
}

func TestLinkedClaimEffectTxDThenD6(t *testing.T) {
	ctx := context.Background()
	s, stop := stopFixture(t)
	defer s.Close()
	if err := s.UpdateWorkerSessionStatusAndQuarantine(ctx, stop.PairID, store.WorkerSessionCASUpdate{ExpectedStatus: domain.WorkerSessionIdle, ExpectedQuarantineState: domain.QuarantineClean, Status: domain.WorkerSessionTerminated, QuarantineState: domain.QuarantineClean}); err != nil {
		t.Fatal(err)
	}
	restoreID := "restore-linked-stop"
	principal := "verified-subject"
	req := domain.RestoreReservation{AuthorizationID: "auth-linked-stop", OperationID: restoreID, PairID: stop.PairID, SessionID: stop.SessionID, ExpectedGeneration: stop.TerminalGeneration, RiskScope: "POSSIBLE_PROMPT_REPLAY", AuthorizedPrincipal: principal, Actor: principal}
	if err := s.ReservePairRestore(ctx, req); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordPairRestoreUnknown(ctx, restoreID, principal); err != nil {
		t.Fatal(err)
	}
	if err := s.ClaimRestoreRecovery(ctx, restoreID, stop.PairID, stop.SessionID, stop.TerminalGeneration, principal, principal, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	stop.RestoreOperationID = &restoreID
	stop.RestorePrincipal = &principal
	fake := &testAO{postTerminated: true}
	c := Coordinator{Store: s, AO: fake, Operator: testOperator{principal: principal, allowed: true}, KillTimeout: time.Minute}
	if err := c.Start(ctx, stop); err != nil {
		t.Fatal(err)
	}
	if fake.calls() != 1 {
		t.Fatalf("linked effects=%d", fake.calls())
	}
	worker, err := s.GetWorkerSessionByPair(ctx, stop.PairID)
	if err != nil || worker.QuarantineState != domain.QuarantineQuarantined {
		t.Fatalf("linked stop cleared before Tx D: %+v %v", worker, err)
	}
	if err := s.ResolveAmbiguousRestore(ctx, restoreID, stop.PairID, stop.SessionID, stop.TerminalGeneration, domain.PhysicalExecutionResolution, "exact linked D11 runtime proof", principal, principal, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	worker, err = s.GetWorkerSessionByPair(ctx, stop.PairID)
	if err != nil || worker.QuarantineState != domain.QuarantineQuarantined {
		t.Fatalf("Tx D cleared D6 gate: %+v %v", worker, err)
	}
	clear := store.QuarantineClearance{PairID: stop.PairID, RestoreOperationID: restoreID, Session: store.LineageClearance{SessionID: stop.SessionID, TerminalGeneration: stop.TerminalGeneration, Basis: store.ClearancePhysical, StopOperationID: stop.OperationID, Evidence: "linked physical runtime proof"}}
	if err := c.ClearQuarantine(ctx, clear); err != nil {
		t.Fatal(err)
	}
	worker, err = s.GetWorkerSessionByPair(ctx, stop.PairID)
	if err != nil || worker.QuarantineState != domain.QuarantineClean {
		t.Fatalf("D6 did not clear: %+v %v", worker, err)
	}
	if err := c.Start(ctx, stop); err == nil || fake.calls() != 1 {
		t.Fatalf("linked replay reissued /kill: %v calls=%d", err, fake.calls())
	}
}

func TestPreexistingLinkedIntentNeverGrantsEffect(t *testing.T) {
	ctx := context.Background()
	s, stop := stopFixture(t)
	defer s.Close()
	if err := s.UpdateWorkerSessionStatusAndQuarantine(ctx, stop.PairID, store.WorkerSessionCASUpdate{ExpectedStatus: domain.WorkerSessionIdle, ExpectedQuarantineState: domain.QuarantineClean, Status: domain.WorkerSessionTerminated, QuarantineState: domain.QuarantineClean}); err != nil {
		t.Fatal(err)
	}
	restoreID := "restore-preexisting"
	principal := "verified-subject"
	req := domain.RestoreReservation{AuthorizationID: "auth-preexisting", OperationID: restoreID, PairID: stop.PairID, SessionID: stop.SessionID, ExpectedGeneration: stop.TerminalGeneration, RiskScope: "POSSIBLE_PROMPT_REPLAY", AuthorizedPrincipal: principal, Actor: principal}
	if err := s.ReservePairRestore(ctx, req); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordPairRestoreUnknown(ctx, restoreID, principal); err != nil {
		t.Fatal(err)
	}
	if err := s.ClaimRestoreRecovery(ctx, restoreID, stop.PairID, stop.SessionID, stop.TerminalGeneration, principal, principal, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	stop.RestoreOperationID = &restoreID
	stop.RestorePrincipal = &principal
	stop.Actor = principal
	if err := s.ClaimRestoreCleanupWithStop(ctx, stop, principal, principal, time.Now().UTC(), store.RestoreCleanupObservation{SessionID: stop.SessionID, TerminalGeneration: stop.TerminalGeneration}); err != nil {
		t.Fatal(err)
	}
	fake := &testAO{}
	c := Coordinator{Store: s, AO: fake, Operator: testOperator{principal: principal, allowed: true}, KillTimeout: time.Minute}
	if err := c.Start(ctx, stop); err == nil || fake.calls() != 0 {
		t.Fatalf("existing linked intent issued effect: %v calls=%d", err, fake.calls())
	}
	got, err := s.GetStopOperation(ctx, stop.OperationID)
	if err != nil || got.Stage != domain.StopRequested {
		t.Fatalf("existing intent mutated: %+v %v", got, err)
	}
}

func TestCompetingPairCallersAndRestartReplayDoNotDuplicateKill(t *testing.T) {
	ctx := context.Background()
	s, stop := stopFixture(t)
	defer s.Close()
	fake := &testAO{postTerminated: false}
	c := Coordinator{Store: s, AO: fake, Operator: testOperator{principal: "verified-subject", allowed: true}, KillTimeout: time.Minute}
	start := make(chan struct{})
	results := make(chan error, 2)
	for _, id := range []string{stop.OperationID, "stop-competing"} {
		op := stop
		op.OperationID = id
		go func() { <-start; results <- c.Start(ctx, op) }()
	}
	close(start)
	a, b := <-results, <-results
	if (a == nil) == (b == nil) || fake.calls() != 1 {
		t.Fatalf("Pair race outcomes=%v/%v effects=%d", a, b, fake.calls())
	}
	// Simulate a new process seeing only the persisted STOP_CALL_SUCCEEDED.
	restarted := Coordinator{Store: s, AO: fake, Operator: c.Operator, KillTimeout: time.Minute}
	if err := restarted.Start(ctx, stop); err == nil || fake.calls() != 1 {
		t.Fatalf("restart replay effect: %v calls=%d", err, fake.calls())
	}
}

func TestCrashAfterIntentBeforeEffectFailsClosed(t *testing.T) {
	ctx := context.Background()
	s, stop := stopFixture(t)
	defer s.Close()
	if err := s.ReserveStopOperation(ctx, stop); err != nil {
		t.Fatal(err)
	}
	fake := &testAO{}
	c := Coordinator{Store: s, AO: fake, Operator: testOperator{principal: "verified-subject", allowed: true}, KillTimeout: time.Minute}
	if err := c.Start(ctx, stop); err == nil || fake.calls() != 0 {
		t.Fatalf("persisted intent granted effect: %v calls=%d", err, fake.calls())
	}
	worker, err := s.GetWorkerSessionByPair(ctx, stop.PairID)
	if err != nil || worker.QuarantineState != domain.QuarantineQuarantined {
		t.Fatalf("crash intent did not hold Pair: %+v %v", worker, err)
	}
}

func TestCallerCancellationAfterKillKeepsDurableConfirmation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	s, stop := stopFixture(t)
	defer s.Close()
	fake := &testAO{}
	fake.onKill = cancel
	c := Coordinator{Store: s, AO: fake, Operator: testOperator{principal: "verified-subject", allowed: true}, KillTimeout: time.Minute}
	if err := c.Start(ctx, stop); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetStopOperation(context.Background(), stop.OperationID)
	if err != nil || got.Stage != domain.StopCallSucceeded || got.ConfirmationDeadlineAt == nil || fake.calls() != 1 {
		t.Fatalf("cancellation lost confirmation or retried: %+v %v calls=%d", got, err, fake.calls())
	}
}

func TestLinkedAdministrativeReconciliationBeforeTxDAndD6(t *testing.T) {
	ctx := context.Background()
	s, stop := stopFixture(t)
	defer s.Close()
	if err := s.UpdateWorkerSessionStatusAndQuarantine(ctx, stop.PairID, store.WorkerSessionCASUpdate{ExpectedStatus: domain.WorkerSessionIdle, ExpectedQuarantineState: domain.QuarantineClean, Status: domain.WorkerSessionTerminated, QuarantineState: domain.QuarantineClean}); err != nil {
		t.Fatal(err)
	}
	principal, restoreID := "verified-subject", "restore-admin-linked"
	req := domain.RestoreReservation{AuthorizationID: "auth-admin-linked", OperationID: restoreID, PairID: stop.PairID, SessionID: stop.SessionID, ExpectedGeneration: stop.TerminalGeneration, RiskScope: "POSSIBLE_PROMPT_REPLAY", AuthorizedPrincipal: principal, Actor: principal}
	if err := s.ReservePairRestore(ctx, req); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordPairRestoreUnknown(ctx, restoreID, principal); err != nil {
		t.Fatal(err)
	}
	if err := s.ClaimRestoreRecovery(ctx, restoreID, stop.PairID, stop.SessionID, stop.TerminalGeneration, principal, principal, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	stop.RestoreOperationID = &restoreID
	stop.RestorePrincipal = &principal
	stop.Actor = principal
	if err := s.ClaimRestoreCleanupWithStop(ctx, stop, principal, principal, time.Now().UTC(), store.RestoreCleanupObservation{SessionID: stop.SessionID, TerminalGeneration: stop.TerminalGeneration}); err != nil {
		t.Fatal(err)
	}
	if err := s.CommitStopOutcome(ctx, stop.OperationID, store.StopTerminalOutcome{ExpectedStage: domain.StopRequested, Stage: domain.StopCallFailed, Resolution: domain.StopResolutionCallFailed, At: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	if err := s.ResolveAmbiguousRestore(ctx, restoreID, stop.PairID, stop.SessionID, stop.TerminalGeneration, domain.AdministrativeRiskResolution, "operator case", principal, principal, time.Now().UTC()); err == nil {
		t.Fatal("Tx D accepted unreconciled linked stop")
	}
	clear := store.QuarantineClearance{PairID: stop.PairID, RestoreOperationID: restoreID, Principal: principal, Session: store.LineageClearance{SessionID: stop.SessionID, TerminalGeneration: stop.TerminalGeneration, Basis: store.ClearanceRiskAdministrative, StopOperationID: stop.OperationID, Evidence: "operator case"}}
	if err := s.ClearQuarantineWithEvidence(ctx, clear); err == nil {
		t.Fatal("D6 ran before Tx D")
	}
	fake := &testAO{}
	c := Coordinator{Store: s, AO: fake, Operator: testOperator{principal: principal, allowed: false}, KillTimeout: time.Minute}
	d := store.AdministrativeStopDecision{StopOperationID: stop.OperationID, PairID: stop.PairID, SessionID: stop.SessionID, TerminalGeneration: stop.TerminalGeneration, Purpose: stop.Purpose, RestoreOperationID: restoreID, ExpectedStage: domain.StopCallFailed, ExpectedResolution: domain.StopResolutionCallFailed, Reason: "residual risk", EvidenceReferences: []string{"operator case"}, Lineages: []store.LineageClearance{{SessionID: stop.SessionID, TerminalGeneration: stop.TerminalGeneration}}, At: time.Now().UTC()}
	if err := c.AcceptAdministrativeRisk(ctx, d); err == nil {
		t.Fatal("unverified principal admitted")
	}
	c.Operator = scopeOperator{principal: principal, allowedScope: "RESTORE_ONLY"}
	if err := c.AcceptAdministrativeRisk(ctx, d); err == nil {
		t.Fatal("wrong authority scope admitted")
	}
	c.Operator = testOperator{principal: principal, allowed: true}
	d.Principal = "wrong"
	if err := c.AcceptAdministrativeRisk(ctx, d); err == nil {
		t.Fatal("wrong principal admitted")
	}
	d.Principal = ""
	if err := c.AcceptAdministrativeRisk(ctx, d); err != nil {
		t.Fatal(err)
	}
	worker, err := s.GetWorkerSessionByPair(ctx, stop.PairID)
	if err != nil || worker.QuarantineState != domain.QuarantineQuarantined {
		t.Fatalf("risk acceptance cleared quarantine: %+v %v", worker, err)
	}
	if err := s.ResolveAmbiguousRestore(ctx, restoreID, stop.PairID, stop.SessionID, stop.TerminalGeneration, domain.AdministrativeRiskResolution, "operator case", principal, principal, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	worker, err = s.GetWorkerSessionByPair(ctx, stop.PairID)
	if err != nil || worker.QuarantineState != domain.QuarantineQuarantined {
		t.Fatalf("Tx D cleared quarantine: %+v %v", worker, err)
	}
	clear.Principal = ""
	if err := c.ClearQuarantine(ctx, clear); err != nil {
		t.Fatal(err)
	}
	worker, err = s.GetWorkerSessionByPair(ctx, stop.PairID)
	if err != nil || worker.QuarantineState != domain.QuarantineClean {
		t.Fatalf("D6 did not clear: %+v %v", worker, err)
	}
	if err := c.AcceptAdministrativeRisk(ctx, d); err != nil {
		t.Fatalf("exact replay after D6: %v", err)
	}
	if fake.calls() != 0 {
		t.Fatalf("administrative path issued /kill %d times", fake.calls())
	}
}
