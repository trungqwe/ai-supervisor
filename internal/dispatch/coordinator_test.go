package dispatch

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/trungqwe/ai-supervisor/internal/ao"
	"github.com/trungqwe/ai-supervisor/internal/domain"
	"github.com/trungqwe/ai-supervisor/internal/store"
)

type fakeAO struct {
	restores, sends, creates int
	createResult             *ao.CreateWorkerSessionResult
	createErr                error
	statusResult             *ao.WorkerStatus
	statusErr                error
	dispatchResult           *ao.DispatchTaskResult
	dispatchErr              error
	restoreErr               error
	resumeResult             *ao.ResumeWorkerResult
	onResume                 func(context.Context, string)
}

func (f *fakeAO) CreateWorkerSession(context.Context, string, string) (*ao.CreateWorkerSessionResult, error) {
	f.creates++
	return f.createResult, f.createErr
}
func (f *fakeAO) GetWorkerStatus(context.Context, string) (*ao.WorkerStatus, error) {
	if f.statusResult != nil || f.statusErr != nil {
		return f.statusResult, f.statusErr
	}
	if f.restores > 0 && f.resumeResult != nil {
		status := f.resumeResult.Session
		return &status, nil
	}
	return f.statusResult, f.statusErr
}
func (f *fakeAO) ResumeWorker(ctx context.Context, sessionID string) (*ao.ResumeWorkerResult, error) {
	f.restores++
	if f.onResume != nil {
		f.onResume(ctx, sessionID)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return f.resumeResult, f.restoreErr
}
func (f *fakeAO) DispatchTaskContract(context.Context, string, string) (*ao.DispatchTaskResult, error) {
	f.sends++
	return f.dispatchResult, f.dispatchErr
}

type verifiedOperator struct {
	principal     string
	ok            bool
	requiredScope string
}

func (v verifiedOperator) VerifiedRestorePrincipal(_ context.Context, _, _, _, scope string) (string, bool, error) {
	if v.requiredScope != "" && v.requiredScope != scope {
		return "", false, nil
	}
	return v.principal, v.ok, nil
}

func TestRestoreIsDisabledWithoutExplicitHostBoundaryEnablement(t *testing.T) {
	upstream := &fakeAO{}
	coordinator := Coordinator{AO: upstream, Operator: verifiedOperator{principal: "test-subject", ok: true}}
	err := coordinator.Restore(context.Background(), domain.RestoreReservation{OperationID: "restore", PairID: "pair", SessionID: "session", ExpectedGeneration: "generation", RiskScope: "POSSIBLE_PROMPT_REPLAY", Actor: "test"})
	if err == nil || upstream.restores != 0 {
		t.Fatalf("default-disabled restore made AO call: err=%v calls=%d", err, upstream.restores)
	}
}

func seedRestorablePair(t *testing.T, s *store.Store, pairID, sessionID, generation string) {
	t.Helper()
	ctx := context.Background()
	projectID := "project-" + pairID
	if err := s.CreateProject(ctx, domain.Project{ProjectID: projectID, Name: projectID, RootPath: "/" + projectID}); err != nil {
		t.Fatal(err)
	}
	if err := s.CreatePair(ctx, domain.Pair{PairID: pairID, ProjectID: projectID, CurrentPhaseID: "P03", State: "ACTIVE"}); err != nil {
		t.Fatal(err)
	}
	provision := domain.PairProvisioningOperation{OperationID: "provision-" + pairID, PairID: pairID, ClientToken: "token-" + pairID}
	if err := s.ReservePairProvisioning(ctx, provision, "supervisor"); err != nil {
		t.Fatal(err)
	}
	if err := s.ConfirmPairProvisioning(ctx, provision.OperationID, domain.WorkerSession{PairID: pairID, SessionID: sessionID, RuntimeType: "agy_tui", WorkerAgentID: "agy", Status: domain.WorkerSessionTerminated, TerminalGeneration: generation, QuarantineState: domain.QuarantineClean}, "supervisor", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
}

func restoreRequest(pairID, sessionID, generation, operationID string) domain.RestoreReservation {
	return domain.RestoreReservation{AuthorizationID: "auth-" + operationID, OperationID: operationID, PairID: pairID, SessionID: sessionID, ExpectedGeneration: generation, RiskScope: "POSSIBLE_PROMPT_REPLAY", Actor: "request-actor"}
}

func TestRestoreIntentCommitsBeforeOneAOCallAndHTTP200KeepsHold(t *testing.T) {
	ctx := context.Background()
	s, err := store.Open(ctx, store.Config{DBPath: filepath.Join(t.TempDir(), "restore-success.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	seedRestorablePair(t, s, "pair-restore-success", "session-restore-success", "generation-old")
	req := restoreRequest("pair-restore-success", "session-restore-success", "generation-old", "restore-success")
	upstream := &fakeAO{resumeResult: &ao.ResumeWorkerResult{SessionID: req.SessionID, RestoreMode: ao.RestoreModeNative, Session: ao.WorkerStatus{ID: req.SessionID, TerminalGeneration: "generation-new", Activity: ao.ActivitySnapshot{State: ao.ActivityStateIdle}}}}
	upstream.onResume = func(ctx context.Context, sessionID string) {
		op, err := s.GetPairRestoreOperation(ctx, req.OperationID)
		if err != nil || op.Stage != "RESTORE_REQUESTED" || op.ResolutionState != domain.RestoreInFlight {
			t.Errorf("AO call preceded durable restore intent: %+v %v", op, err)
		}
		session, err := s.GetWorkerSessionByPair(ctx, req.PairID)
		if err != nil || session.SessionID != sessionID || session.QuarantineState != domain.QuarantineQuarantined {
			t.Errorf("AO call preceded Pair quarantine: %+v %v", session, err)
		}
	}
	c := Coordinator{Store: s, AO: upstream, Operator: verifiedOperator{principal: "host-principal", ok: true, requiredScope: "WRONG_SCOPE"}, RestoreEnabled: true}
	if err := c.Restore(ctx, req); err == nil || upstream.restores != 0 {
		t.Fatalf("unverified host boundary allowed restore: err=%v calls=%d", err, upstream.restores)
	}
	if _, err := s.GetPairRestoreOperation(ctx, req.OperationID); !errors.Is(err, store.ErrOperationNotFound) {
		t.Fatalf("unverified boundary persisted restore intent: %v", err)
	}
	c.Operator = verifiedOperator{principal: "", ok: true, requiredScope: req.RiskScope}
	if err := c.Restore(ctx, req); err == nil || upstream.restores != 0 {
		t.Fatalf("empty verified principal allowed restore: err=%v calls=%d", err, upstream.restores)
	}
	c.Operator = verifiedOperator{principal: "host-principal", ok: true, requiredScope: req.RiskScope}
	if err := c.Restore(ctx, req); err != nil {
		t.Fatal(err)
	}
	if upstream.restores != 1 {
		t.Fatalf("restore AO calls=%d, want 1", upstream.restores)
	}
	op, err := s.GetPairRestoreOperation(ctx, req.OperationID)
	if err != nil || op.Stage != "RESTORE_CONFIRMED" || op.ResolutionState != domain.RestoreInFlight || op.HTTP200At == nil {
		t.Fatalf("restore confirmation state: %+v %v", op, err)
	}
	session, err := s.GetWorkerSessionByPair(ctx, req.PairID)
	if err != nil || session.TerminalGeneration != "generation-new" || session.QuarantineState != domain.QuarantineQuarantined {
		t.Fatalf("HTTP 200 incorrectly cleared Pair quarantine: %+v %v", session, err)
	}
}

func TestRestoreAmbiguousAndInvalidOutcomesAreNeverRetried(t *testing.T) {
	for _, tc := range []struct {
		name   string
		result *ao.ResumeWorkerResult
		err    error
		cancel bool
	}{{name: "transport or HTTP 409", err: errors.New("HTTP 409 or lost response")}, {name: "canceled caller context", cancel: true}, {name: "invalid identity", result: &ao.ResumeWorkerResult{SessionID: "different-session", RestoreMode: ao.RestoreModeNative, Session: ao.WorkerStatus{ID: "different-session", TerminalGeneration: "new", Activity: ao.ActivitySnapshot{State: ao.ActivityStateIdle}}}}} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			s, err := store.Open(ctx, store.Config{DBPath: filepath.Join(t.TempDir(), "restore-unknown.db")})
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			seedRestorablePair(t, s, "pair-restore-unknown", "session-restore-unknown", "old")
			req := restoreRequest("pair-restore-unknown", "session-restore-unknown", "old", "restore-unknown")
			upstream := &fakeAO{restoreErr: tc.err, resumeResult: tc.result}
			callCtx := ctx
			if tc.cancel {
				var cancel context.CancelFunc
				callCtx, cancel = context.WithCancel(ctx)
				upstream.onResume = func(context.Context, string) { cancel() }
			}
			c := Coordinator{Store: s, AO: upstream, Operator: verifiedOperator{principal: "host-principal", ok: true}, RestoreEnabled: true}
			if err := c.Restore(callCtx, req); err == nil {
				t.Fatal("ambiguous/invalid restore unexpectedly succeeded")
			}
			op, err := s.GetPairRestoreOperation(ctx, req.OperationID)
			if err != nil || op.ResolutionState != domain.RestoreOutcomeUnknown {
				t.Fatalf("restore uncertainty not durable: %+v %v", op, err)
			}
			if err := c.Restore(ctx, req); err == nil {
				t.Fatal("unresolved restore was replayed")
			}
			if upstream.restores != 1 {
				t.Fatalf("ambiguous restore AO calls=%d, want 1", upstream.restores)
			}
		})
	}
}

func TestRestoreConfirmationAuditFailurePersistsUnknownAndDoesNotRetry(t *testing.T) {
	ctx := context.Background()
	cfg := store.Config{DBPath: filepath.Join(t.TempDir(), "restore-confirm-audit.db"), BusyTimeoutMs: 5000}
	s, err := store.Open(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	seedRestorablePair(t, s, "pair-restore-confirm-audit", "session-restore-confirm-audit", "old")
	req := restoreRequest("pair-restore-confirm-audit", "session-restore-confirm-audit", "old", "restore-confirm-audit")
	raw, err := sql.Open("sqlite", cfg.DSN())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.ExecContext(ctx, `CREATE TRIGGER reject_restore_confirmation BEFORE INSERT ON audit_events WHEN NEW.event_type='PAIR_RESTORE_CONFIRMED' BEGIN SELECT RAISE(ABORT,'injected confirmation audit failure'); END`); err != nil {
		t.Fatal(err)
	}
	if err := raw.Close(); err != nil {
		t.Fatal(err)
	}
	upstream := &fakeAO{resumeResult: &ao.ResumeWorkerResult{SessionID: req.SessionID, RestoreMode: ao.RestoreModeNative, Session: ao.WorkerStatus{ID: req.SessionID, TerminalGeneration: "new", Activity: ao.ActivitySnapshot{State: ao.ActivityStateIdle}}}}
	c := Coordinator{Store: s, AO: upstream, Operator: verifiedOperator{principal: "host-principal", ok: true}, RestoreEnabled: true}
	if err := c.Restore(ctx, req); err == nil {
		t.Fatal("restore confirmation audit failure unexpectedly succeeded")
	}
	op, err := s.GetPairRestoreOperation(ctx, req.OperationID)
	if err != nil || op.Stage != "RESTORE_REQUESTED" || op.ResolutionState != domain.RestoreOutcomeUnknown {
		t.Fatalf("confirmation failure did not remain unknown: %+v %v", op, err)
	}
	if upstream.restores != 1 {
		t.Fatalf("confirmation failure restore AO calls=%d, want 1", upstream.restores)
	}
}

func TestRestoreRequiresPostHTTP200GetToMatchIdentityAndGeneration(t *testing.T) {
	ctx := context.Background()
	s, err := store.Open(ctx, store.Config{DBPath: filepath.Join(t.TempDir(), "restore-post-get.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	seedRestorablePair(t, s, "pair-restore-post-get", "session-restore-post-get", "old")
	req := restoreRequest("pair-restore-post-get", "session-restore-post-get", "old", "restore-post-get")
	upstream := &fakeAO{
		resumeResult: &ao.ResumeWorkerResult{SessionID: req.SessionID, RestoreMode: ao.RestoreModeNative, Session: ao.WorkerStatus{ID: req.SessionID, TerminalGeneration: "new", Activity: ao.ActivitySnapshot{State: ao.ActivityStateIdle}}},
		statusResult: &ao.WorkerStatus{ID: req.SessionID, TerminalGeneration: "other-generation", Activity: ao.ActivitySnapshot{State: ao.ActivityStateIdle}},
	}
	c := Coordinator{Store: s, AO: upstream, Operator: verifiedOperator{principal: "verified-host", ok: true}, RestoreEnabled: true}
	if err := c.Restore(ctx, req); err == nil {
		t.Fatal("mismatched post-restore generation was accepted")
	}
	op, err := s.GetPairRestoreOperation(ctx, req.OperationID)
	if err != nil || op.Stage != "RESTORE_REQUESTED" || op.ResolutionState != domain.RestoreOutcomeUnknown {
		t.Fatalf("mismatched GET was not contained: %+v %v", op, err)
	}
	ws, err := s.GetWorkerSessionByPair(ctx, req.PairID)
	if err != nil || ws.TerminalGeneration != "old" || ws.QuarantineState != domain.QuarantineQuarantined {
		t.Fatalf("DB claimed unconfirmed generation: %+v %v", ws, err)
	}
	if upstream.restores != 1 {
		t.Fatalf("restore was retried after ambiguous GET: %d calls", upstream.restores)
	}
	if err := c.Restore(ctx, req); err == nil || upstream.restores != 1 {
		t.Fatalf("unresolved restore replayed: err=%v calls=%d", err, upstream.restores)
	}
}

func TestRestoreDurableIntentSurvivesCrashBeforeAOAndIsNotReissued(t *testing.T) {
	ctx := context.Background()
	cfg := store.Config{DBPath: filepath.Join(t.TempDir(), "restore-crash-before-effect.db"), BusyTimeoutMs: 5000}
	s, err := store.Open(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if s != nil {
			_ = s.Close()
		}
	}()
	seedRestorablePair(t, s, "pair-restore-crash", "session-restore-crash", "old")
	req := restoreRequest("pair-restore-crash", "session-restore-crash", "old", "restore-crash")
	req.AuthorizedPrincipal = "host-principal"
	req.Actor = "host-principal"
	if err := s.ReservePairRestore(ctx, req); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s = nil
	s, err = store.Open(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	upstream := &fakeAO{resumeResult: &ao.ResumeWorkerResult{SessionID: req.SessionID, RestoreMode: ao.RestoreModeNative, Session: ao.WorkerStatus{ID: req.SessionID, TerminalGeneration: "new", Activity: ao.ActivitySnapshot{State: ao.ActivityStateIdle}}}}
	c := Coordinator{Store: s, AO: upstream, Operator: verifiedOperator{principal: "host-principal", ok: true}, RestoreEnabled: true}
	if err := c.Restore(ctx, req); err == nil {
		t.Fatal("restarted coordinator replayed unissued durable restore")
	}
	if upstream.restores != 0 {
		t.Fatalf("crash-before-effect caused %d AO restore calls", upstream.restores)
	}
	op, err := s.GetPairRestoreOperation(ctx, req.OperationID)
	if err != nil || op.Stage != "RESTORE_REQUESTED" || op.ResolutionState != domain.RestoreInFlight {
		t.Fatalf("crash intent was not preserved: %+v %v", op, err)
	}
}

func TestProvisioningPersistsBeforeOneAOCallAndRejectsDuplicate(t *testing.T) {
	ctx := context.Background()
	s, err := store.Open(ctx, store.Config{DBPath: filepath.Join(t.TempDir(), "state.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.CreateProject(ctx, domain.Project{ProjectID: "project-coordinator", Name: "coordinator", RootPath: "/coordinator"}); err != nil {
		t.Fatal(err)
	}
	if err := s.CreatePair(ctx, domain.Pair{PairID: "pair-coordinator", ProjectID: "project-coordinator", CurrentPhaseID: "P03", State: "ACTIVE"}); err != nil {
		t.Fatal(err)
	}
	upstream := &fakeAO{createResult: &ao.CreateWorkerSessionResult{Session: ao.WorkerStatus{ID: "session-coordinator", TerminalGeneration: "generation-1", Activity: ao.ActivitySnapshot{State: ao.ActivityStateIdle}}}}
	c := Coordinator{Store: s, AO: upstream}
	op := domain.PairProvisioningOperation{OperationID: "provision-coordinator", PairID: "pair-coordinator", ClientToken: "token-1"}
	if err := c.Provision(ctx, op, "project-coordinator", "agy_tui", "supervisor"); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetPairProvisioningOperation(ctx, op.OperationID)
	if err != nil || got.Stage != domain.ProvisionConfirmed {
		t.Fatalf("confirmed provisioning state: %+v %v", got, err)
	}
	events, err := s.ListAuditEvents(ctx, 0, store.MaxAuditLimit)
	if err != nil {
		t.Fatal(err)
	}
	provisionAudit := map[string]bool{}
	for _, event := range events {
		if id, _ := event.Event.Details["operation_id"].(string); id == op.OperationID {
			provisionAudit[event.Event.EventType] = true
		}
	}
	if !provisionAudit[domain.AuditPairSessionProvisionRequested] || !provisionAudit[domain.AuditPairSessionProvisioned] {
		t.Fatalf("approved provisioning event_type evidence missing: %v", provisionAudit)
	}
	if upstream.creates != 1 {
		t.Fatalf("AO create calls=%d, want 1", upstream.creates)
	}
	second := op
	second.OperationID = "provision-coordinator-2"
	second.ClientToken = "token-2"
	if err := c.Provision(ctx, second, "project-coordinator", "agy_tui", "supervisor"); err == nil {
		t.Fatal("duplicate provisioning unexpectedly succeeded")
	}
	if upstream.creates != 1 {
		t.Fatalf("duplicate call reached AO: calls=%d", upstream.creates)
	}
}

func TestProvisioningAmbiguousCreateIsFailedAndNeverRespawned(t *testing.T) {
	ctx := context.Background()
	s, err := store.Open(ctx, store.Config{DBPath: filepath.Join(t.TempDir(), "state.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.CreateProject(ctx, domain.Project{ProjectID: "project-ambiguous", Name: "ambiguous", RootPath: "/ambiguous"}); err != nil {
		t.Fatal(err)
	}
	if err := s.CreatePair(ctx, domain.Pair{PairID: "pair-ambiguous", ProjectID: "project-ambiguous", CurrentPhaseID: "P03", State: "ACTIVE"}); err != nil {
		t.Fatal(err)
	}
	upstream := &fakeAO{createErr: errors.New("transport outcome unknown")}
	c := Coordinator{Store: s, AO: upstream}
	op := domain.PairProvisioningOperation{OperationID: "provision-ambiguous", PairID: "pair-ambiguous", ClientToken: "token-ambiguous"}
	if err := c.Provision(ctx, op, "project-ambiguous", "agy_tui", "supervisor"); err == nil {
		t.Fatal("ambiguous AO create unexpectedly succeeded")
	}
	got, err := s.GetPairProvisioningOperation(ctx, op.OperationID)
	if err != nil || got.Stage != domain.ProvisionFailed {
		t.Fatalf("ambiguous provisioning disposition: %+v %v", got, err)
	}
	if err := c.Provision(ctx, op, "project-ambiguous", "agy_tui", "supervisor"); err == nil {
		t.Fatal("failed provisioning was retried")
	}
	if upstream.creates != 1 {
		t.Fatalf("ambiguous create calls=%d, want 1", upstream.creates)
	}
	events, err := s.ListAuditEvents(ctx, 0, store.MaxAuditLimit)
	if err != nil {
		t.Fatal(err)
	}
	foundFailureEvent := false
	for _, event := range events {
		if id, _ := event.Event.Details["operation_id"].(string); id == op.OperationID && event.Event.EventType == domain.AuditPairSessionProvisionFailed {
			foundFailureEvent = true
		}
	}
	if !foundFailureEvent {
		t.Fatal("approved PAIR_SESSION_PROVISION_FAILED event_type missing")
	}
}

func TestDispatchAmbiguousSendIsPersistedAndNeverRepeated(t *testing.T) {
	ctx := context.Background()
	s, err := store.Open(ctx, store.Config{DBPath: filepath.Join(t.TempDir(), "state.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.CreateProject(ctx, domain.Project{ProjectID: "project-send", Name: "send", RootPath: "/send"}); err != nil {
		t.Fatal(err)
	}
	if err := s.CreatePair(ctx, domain.Pair{PairID: "pair-send", ProjectID: "project-send", CurrentPhaseID: "P03", State: "ACTIVE"}); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateTask(ctx, domain.Task{TaskID: "task-send", PhaseID: "P03", PairID: "pair-send", State: domain.StateDraft}); err != nil {
		t.Fatal(err)
	}
	if err := s.InsertTaskContract(ctx, domain.TaskContract{ContractID: "contract-send", TaskID: "task-send", RevisionNumber: 1, BaseSHA: "base", AllowedScope: []string{"internal/domain/**"}}); err != nil {
		t.Fatal(err)
	}
	if err := s.TransitionTask(ctx, "task-send", domain.StateDraft, domain.StateReady); err != nil {
		t.Fatal(err)
	}
	provision := domain.PairProvisioningOperation{OperationID: "provision-send", PairID: "pair-send", ClientToken: "token-send"}
	if err := s.ReservePairProvisioning(ctx, provision, "supervisor"); err != nil {
		t.Fatal(err)
	}
	if err := s.ConfirmPairProvisioning(ctx, provision.OperationID, domain.WorkerSession{PairID: "pair-send", SessionID: "session-send", RuntimeType: "agy_tui", WorkerAgentID: "agy", Status: domain.WorkerSessionIdle, TerminalGeneration: "generation-send", QuarantineState: domain.QuarantineClean}, "supervisor", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	upstream := &fakeAO{statusResult: &ao.WorkerStatus{ID: "session-send", TerminalGeneration: "generation-send", Activity: ao.ActivitySnapshot{State: ao.ActivityStateIdle}}, dispatchErr: errors.New("connection lost after send")}
	c := Coordinator{Store: s, AO: upstream}
	report, err := store.CanonicalExpectedReportPath("task-send", "attempt-send")
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Dispatch(ctx, "task-send", "contract-send", "attempt-send", "dispatch-send", "session-send", "generation-send", report, "immutable contract", "supervisor"); err == nil {
		t.Fatal("ambiguous send unexpectedly returned success")
	}
	if upstream.sends != 1 {
		t.Fatalf("AO send calls=%d, want 1", upstream.sends)
	}
	op, err := s.GetDispatchOperation(ctx, "dispatch-send")
	if err != nil || op.ResolutionState == nil || *op.ResolutionState != domain.RecoveryDeliveryOutcomeUnknown {
		t.Fatalf("durable send resolution: %+v %v", op, err)
	}
	task, err := s.GetTask(ctx, "task-send")
	if err != nil || task.State != domain.StateHumanRequired {
		t.Fatalf("unknown delivery Task state: %+v %v", task, err)
	}
	if err := c.Dispatch(ctx, "task-send", "contract-send", "attempt-send", "dispatch-send", "session-send", "generation-send", report, "immutable contract", "supervisor"); err == nil {
		t.Fatal("terminal unknown attempt was resent")
	}
	if upstream.sends != 1 {
		t.Fatalf("retry reached AO; sends=%d", upstream.sends)
	}
}

func TestDispatchHTTP200ConfirmsAcceptanceWithoutRunningTask(t *testing.T) {
	ctx := context.Background()
	s, err := store.Open(ctx, store.Config{DBPath: filepath.Join(t.TempDir(), "send-confirmed.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.CreateProject(ctx, domain.Project{ProjectID: "project-send-confirmed", Name: "send-confirmed", RootPath: "/send-confirmed"}); err != nil {
		t.Fatal(err)
	}
	if err := s.CreatePair(ctx, domain.Pair{PairID: "pair-send-confirmed", ProjectID: "project-send-confirmed", CurrentPhaseID: "P03", State: "ACTIVE"}); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateTask(ctx, domain.Task{TaskID: "task-send-confirmed", PhaseID: "P03", PairID: "pair-send-confirmed", State: domain.StateDraft}); err != nil {
		t.Fatal(err)
	}
	if err := s.InsertTaskContract(ctx, domain.TaskContract{ContractID: "contract-send-confirmed", TaskID: "task-send-confirmed", RevisionNumber: 1, BaseSHA: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", AllowedScope: []string{"internal/domain/**"}}); err != nil {
		t.Fatal(err)
	}
	if err := s.TransitionTask(ctx, "task-send-confirmed", domain.StateDraft, domain.StateReady); err != nil {
		t.Fatal(err)
	}
	provision := domain.PairProvisioningOperation{OperationID: "provision-send-confirmed", PairID: "pair-send-confirmed", ClientToken: "token-send-confirmed"}
	if err := s.ReservePairProvisioning(ctx, provision, "supervisor"); err != nil {
		t.Fatal(err)
	}
	if err := s.ConfirmPairProvisioning(ctx, provision.OperationID, domain.WorkerSession{PairID: "pair-send-confirmed", SessionID: "session-send-confirmed", RuntimeType: "agy_tui", WorkerAgentID: "agy", Status: domain.WorkerSessionIdle, TerminalGeneration: "generation-send-confirmed", QuarantineState: domain.QuarantineClean}, "supervisor", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	upstream := &fakeAO{statusResult: &ao.WorkerStatus{ID: "session-send-confirmed", TerminalGeneration: "generation-send-confirmed", Activity: ao.ActivitySnapshot{State: ao.ActivityStateWaitingInput}}, dispatchResult: &ao.DispatchTaskResult{SessionID: "session-send-confirmed"}}
	c := Coordinator{Store: s, AO: upstream}
	report, err := store.CanonicalExpectedReportPath("task-send-confirmed", "attempt-send-confirmed")
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Dispatch(ctx, "task-send-confirmed", "contract-send-confirmed", "attempt-send-confirmed", "dispatch-send-confirmed", "session-send-confirmed", "generation-send-confirmed", report, "contract text", "supervisor"); err != nil {
		t.Fatal(err)
	}
	if upstream.sends != 1 {
		t.Fatalf("send AO calls=%d, want 1", upstream.sends)
	}
	task, err := s.GetTask(ctx, "task-send-confirmed")
	if err != nil || task.State != domain.StateDispatched {
		t.Fatalf("HTTP 200 incorrectly claims task running: %+v %v", task, err)
	}
	op, err := s.GetDispatchOperation(ctx, "dispatch-send-confirmed")
	if err != nil || op.Stage != domain.SendConfirmed {
		t.Fatalf("send confirmation state: %+v %v", op, err)
	}
	if err := c.Dispatch(ctx, "task-send-confirmed", "contract-send-confirmed", "attempt-send-confirmed", "dispatch-send-confirmed", "session-send-confirmed", "generation-send-confirmed", report, "contract text", "supervisor"); err == nil {
		t.Fatal("duplicate dispatch call unexpectedly succeeded")
	}
	if upstream.sends != 1 {
		t.Fatalf("duplicate dispatch reached AO; send calls=%d", upstream.sends)
	}
}

func TestHTTP200ConfirmationRollbackRunsFreshD5ContainmentWithoutResend(t *testing.T) {
	ctx := context.Background()
	cfg := store.Config{DBPath: filepath.Join(t.TempDir(), "dispatch-confirm-rollback.db")}
	s, err := store.Open(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	prepareDispatchCoordinatorFixture(t, s, "confirm-rollback")
	upstream := &fakeAO{statusResult: &ao.WorkerStatus{ID: "session-confirm-rollback", TerminalGeneration: "generation-confirm-rollback", Activity: ao.ActivitySnapshot{State: ao.ActivityStateIdle}}, dispatchResult: &ao.DispatchTaskResult{SessionID: "session-confirm-rollback"}}
	c := Coordinator{Store: s, AO: upstream}
	raw, err := sql.Open("sqlite", cfg.DSN())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.ExecContext(ctx, `CREATE TRIGGER reject_send_confirm_for_remediation BEFORE INSERT ON audit_events WHEN NEW.event_type='DISPATCH_SEND_CONFIRMED' BEGIN SELECT RAISE(ABORT,'injected confirmation rollback'); END`); err != nil {
		t.Fatal(err)
	}
	_ = raw.Close()
	report, _ := store.CanonicalExpectedReportPath("task-confirm-rollback", "attempt-confirm-rollback")
	if err := c.Dispatch(ctx, "task-confirm-rollback", "contract-confirm-rollback", "attempt-confirm-rollback", "dispatch-confirm-rollback", "session-confirm-rollback", "generation-confirm-rollback", report, "contract", "supervisor"); err == nil {
		t.Fatal("HTTP 200 transaction rollback was reported as success")
	}
	if upstream.sends != 1 {
		t.Fatalf("send call count=%d, want exactly one", upstream.sends)
	}
	op, err := s.GetDispatchOperation(ctx, "dispatch-confirm-rollback")
	if err != nil || op.Stage != domain.SendRequested || op.ResolutionState == nil || *op.ResolutionState != domain.RecoveryDeliveryOutcomeUnknown {
		t.Fatalf("fresh D5 CAS did not contain confirmation rollback: %+v %v", op, err)
	}
	task, err := s.GetTask(ctx, "task-confirm-rollback")
	if err != nil || task.State != domain.StateHumanRequired {
		t.Fatalf("D5 Task containment: %+v %v", task, err)
	}
	assertDispatchAuditTypes(t, s, "dispatch-confirm-rollback", domain.AuditDispatchSendRequested, domain.AuditUncertainDeliveryQuarantine)
}

func TestInvalidSendResponseWithD5AuditFailureLeavesIntentAndNeverResends(t *testing.T) {
	ctx := context.Background()
	cfg := store.Config{DBPath: filepath.Join(t.TempDir(), "dispatch-invalid-d5-failure.db")}
	s, err := store.Open(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	prepareDispatchCoordinatorFixture(t, s, "invalid-d5-failure")
	upstream := &fakeAO{statusResult: &ao.WorkerStatus{ID: "session-invalid-d5-failure", TerminalGeneration: "generation-invalid-d5-failure", Activity: ao.ActivitySnapshot{State: ao.ActivityStateIdle}}, dispatchResult: &ao.DispatchTaskResult{SessionID: "different-session"}}
	c := Coordinator{Store: s, AO: upstream}
	raw, err := sql.Open("sqlite", cfg.DSN())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.ExecContext(ctx, `CREATE TRIGGER reject_unknown_quarantine BEFORE INSERT ON audit_events WHEN NEW.event_type='UNCERTAIN_DELIVERY_QUARANTINE_IMPOSED' BEGIN SELECT RAISE(ABORT,'injected D5 audit failure'); END`); err != nil {
		t.Fatal(err)
	}
	_ = raw.Close()
	report, _ := store.CanonicalExpectedReportPath("task-invalid-d5-failure", "attempt-invalid-d5-failure")
	if err := c.Dispatch(ctx, "task-invalid-d5-failure", "contract-invalid-d5-failure", "attempt-invalid-d5-failure", "dispatch-invalid-d5-failure", "session-invalid-d5-failure", "generation-invalid-d5-failure", report, "contract", "supervisor"); err == nil {
		t.Fatal("invalid response with failed containment unexpectedly succeeded")
	}
	if upstream.sends != 1 {
		t.Fatalf("send call count=%d, want exactly one", upstream.sends)
	}
	op, err := s.GetDispatchOperation(ctx, "dispatch-invalid-d5-failure")
	if err != nil || op.Stage != domain.SendRequested || op.ResolutionState != nil {
		t.Fatalf("failed D5 must leave durable intent for recovery: %+v %v", op, err)
	}
	if err := c.Dispatch(ctx, "task-invalid-d5-failure", "contract-invalid-d5-failure", "attempt-invalid-d5-failure", "dispatch-invalid-d5-failure", "session-invalid-d5-failure", "generation-invalid-d5-failure", report, "contract", "supervisor"); err == nil {
		t.Fatal("retry of existing dispatched intent unexpectedly succeeded")
	}
	if upstream.sends != 1 {
		t.Fatalf("ambiguous intent was resent: %d calls", upstream.sends)
	}
}

func prepareDispatchCoordinatorFixture(t *testing.T, s *store.Store, suffix string) {
	t.Helper()
	ctx := context.Background()
	pairID, taskID, contractID := "pair-"+suffix, "task-"+suffix, "contract-"+suffix
	sessionID, generation := "session-"+suffix, "generation-"+suffix
	if err := s.CreateProject(ctx, domain.Project{ProjectID: "project-" + suffix, Name: suffix, RootPath: "/" + suffix}); err != nil {
		t.Fatal(err)
	}
	if err := s.CreatePair(ctx, domain.Pair{PairID: pairID, ProjectID: "project-" + suffix, CurrentPhaseID: "P03", State: "ACTIVE"}); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateTask(ctx, domain.Task{TaskID: taskID, PhaseID: "P03", PairID: pairID, State: domain.StateDraft}); err != nil {
		t.Fatal(err)
	}
	if err := s.InsertTaskContract(ctx, domain.TaskContract{ContractID: contractID, TaskID: taskID, RevisionNumber: 1, BaseSHA: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", AllowedScope: []string{"internal/domain/**"}}); err != nil {
		t.Fatal(err)
	}
	if err := s.TransitionTask(ctx, taskID, domain.StateDraft, domain.StateReady); err != nil {
		t.Fatal(err)
	}
	provision := domain.PairProvisioningOperation{OperationID: "provision-" + suffix, PairID: pairID, ClientToken: "token-" + suffix}
	if err := s.ReservePairProvisioning(ctx, provision, "supervisor"); err != nil {
		t.Fatal(err)
	}
	if err := s.ConfirmPairProvisioning(ctx, provision.OperationID, domain.WorkerSession{PairID: pairID, SessionID: sessionID, RuntimeType: "agy_tui", WorkerAgentID: "agy", Status: domain.WorkerSessionIdle, TerminalGeneration: generation, QuarantineState: domain.QuarantineClean}, "supervisor", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
}

func assertDispatchAuditTypes(t *testing.T, s *store.Store, operationID string, want ...string) {
	t.Helper()
	events, err := s.ListAuditEvents(context.Background(), 0, store.MaxAuditLimit)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, event := range events {
		if id, _ := event.Event.Details["dispatch_operation_id"].(string); id == operationID {
			got[event.Event.EventType] = true
		}
	}
	for _, eventType := range want {
		if !got[eventType] {
			t.Errorf("audit event_type %q missing for %s; got=%v", eventType, operationID, got)
		}
	}
}

func TestPreSendObservationMatrixKeepsOrClosesExactAttempt(t *testing.T) {
	for _, tc := range []struct {
		name             string
		status           *ao.WorkerStatus
		statusErr        error
		wantTask         domain.TaskState
		wantDisposition  string
		wantAttemptEnded bool
		wantQuarantine   bool
		wantRejectAudit  bool
	}{
		{name: "active", status: &ao.WorkerStatus{ID: "session-pre-send", TerminalGeneration: "generation-pre-send", Activity: ao.ActivitySnapshot{State: ao.ActivityStateActive}}, wantTask: domain.StateDispatched, wantRejectAudit: true},
		{name: "blocked", status: &ao.WorkerStatus{ID: "session-pre-send", TerminalGeneration: "generation-pre-send", Activity: ao.ActivitySnapshot{State: ao.ActivityStateBlocked}}, wantTask: domain.StateDispatched, wantRejectAudit: true},
		{name: "exited", status: &ao.WorkerStatus{ID: "session-pre-send", TerminalGeneration: "generation-pre-send", Activity: ao.ActivitySnapshot{State: ao.ActivityStateExited}}, wantTask: domain.StateDispatched, wantRejectAudit: true},
		{name: "isTerminated", status: &ao.WorkerStatus{ID: "session-pre-send", TerminalGeneration: "generation-pre-send", IsTerminated: true, Activity: ao.ActivitySnapshot{State: ao.ActivityStateIdle}}, wantTask: domain.StateFailed, wantDisposition: "WORKER_TERMINATION_UNKNOWN", wantAttemptEnded: true},
		{name: "generation mismatch", status: &ao.WorkerStatus{ID: "session-pre-send", TerminalGeneration: "generation-new", Activity: ao.ActivitySnapshot{State: ao.ActivityStateIdle}}, wantTask: domain.StateFailed, wantDisposition: "STALE_EXECUTION_GENERATION", wantAttemptEnded: true, wantQuarantine: true},
		{name: "identity mismatch", status: &ao.WorkerStatus{ID: "different-session", TerminalGeneration: "generation-pre-send", Activity: ao.ActivitySnapshot{State: ao.ActivityStateIdle}}, wantTask: domain.StateFailed, wantDisposition: "PRE_SEND_IDENTITY_MISMATCH", wantAttemptEnded: true, wantQuarantine: true},
		{name: "HTTP 404", statusErr: &ao.ProtocolError{StatusCode: 404, Method: "GET", Path: "/api/v1/sessions/session-pre-send"}, wantTask: domain.StateFailed, wantDisposition: "SESSION_ABSENT", wantAttemptEnded: true, wantQuarantine: true},
		{name: "query unavailable", statusErr: errors.New("AO unavailable"), wantTask: domain.StateDispatched, wantDisposition: "RECOVERY_PENDING"},
		{name: "protocol invalid", statusErr: &ao.ProtocolError{StatusCode: 200, Method: "GET", Path: "/api/v1/sessions/session-pre-send", Reason: "invalid response"}, wantTask: domain.StateDispatched, wantDisposition: "PRE_SEND_PROTOCOL_UNVERIFIED"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			s, err := store.Open(ctx, store.Config{DBPath: filepath.Join(t.TempDir(), "pre-send.db")})
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			if err := s.CreateProject(ctx, domain.Project{ProjectID: "project-pre-send", Name: "pre-send", RootPath: "/pre-send"}); err != nil {
				t.Fatal(err)
			}
			if err := s.CreatePair(ctx, domain.Pair{PairID: "pair-pre-send", ProjectID: "project-pre-send", CurrentPhaseID: "P03", State: "ACTIVE"}); err != nil {
				t.Fatal(err)
			}
			if err := s.CreateTask(ctx, domain.Task{TaskID: "task-pre-send", PhaseID: "P03", PairID: "pair-pre-send", State: domain.StateDraft}); err != nil {
				t.Fatal(err)
			}
			if err := s.InsertTaskContract(ctx, domain.TaskContract{ContractID: "contract-pre-send", TaskID: "task-pre-send", RevisionNumber: 1, BaseSHA: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", AllowedScope: []string{"internal/domain/**"}}); err != nil {
				t.Fatal(err)
			}
			if err := s.TransitionTask(ctx, "task-pre-send", domain.StateDraft, domain.StateReady); err != nil {
				t.Fatal(err)
			}
			provision := domain.PairProvisioningOperation{OperationID: "provision-pre-send", PairID: "pair-pre-send", ClientToken: "token-pre-send"}
			if err := s.ReservePairProvisioning(ctx, provision, "supervisor"); err != nil {
				t.Fatal(err)
			}
			if err := s.ConfirmPairProvisioning(ctx, provision.OperationID, domain.WorkerSession{PairID: "pair-pre-send", SessionID: "session-pre-send", RuntimeType: "agy_tui", WorkerAgentID: "agy", Status: domain.WorkerSessionIdle, TerminalGeneration: "generation-pre-send", QuarantineState: domain.QuarantineClean}, "supervisor", time.Now().UTC()); err != nil {
				t.Fatal(err)
			}
			upstream := &fakeAO{statusResult: tc.status, statusErr: tc.statusErr}
			c := Coordinator{Store: s, AO: upstream}
			report, err := store.CanonicalExpectedReportPath("task-pre-send", "attempt-pre-send")
			if err != nil {
				t.Fatal(err)
			}
			if err := c.Dispatch(ctx, "task-pre-send", "contract-pre-send", "attempt-pre-send", "dispatch-pre-send", "session-pre-send", "generation-pre-send", report, "contract text", "supervisor"); err == nil {
				t.Fatal("inadmissible pre-send observation unexpectedly dispatched")
			}
			task, err := s.GetTask(ctx, "task-pre-send")
			if err != nil || task.State != tc.wantTask {
				t.Fatalf("task state=%+v err=%v, want %s", task, err, tc.wantTask)
			}
			attempt, err := s.GetTaskAttempt(ctx, "attempt-pre-send")
			if err != nil || (attempt.EndedAt != nil) != tc.wantAttemptEnded || (attempt.RecoveryDisposition != nil && *attempt.RecoveryDisposition != tc.wantDisposition) {
				t.Fatalf("attempt state=%+v err=%v", attempt, err)
			}
			session, err := s.GetWorkerSessionByPair(ctx, "pair-pre-send")
			if err != nil || (session.QuarantineState == domain.QuarantineQuarantined) != tc.wantQuarantine {
				t.Fatalf("session quarantine=%+v err=%v", session, err)
			}
			if tc.wantRejectAudit {
				events, err := s.ListAuditEvents(ctx, 0, 500)
				if err != nil {
					t.Fatal(err)
				}
				found := false
				for _, event := range events {
					if event.Event.AttemptID == "attempt-pre-send" && event.Event.EventType == domain.AuditPreSendAdmissibilityRejected && event.Event.Details["error_code"] == "PRE_SEND_ADMISSIBILITY_REJECTED" {
						found = true
						break
					}
				}
				if !found {
					t.Fatal("pre-send rejection audit evidence missing")
				}
			}
			if upstream.sends != 0 {
				t.Fatalf("inadmissible observation issued %d sends", upstream.sends)
			}
		})
	}
}
