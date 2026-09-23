package recovery

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/trungqwe/ai-supervisor/internal/ao"
	"github.com/trungqwe/ai-supervisor/internal/domain"
	"github.com/trungqwe/ai-supervisor/internal/store"
)

type testScope struct {
	host *testHost
	once sync.Once
}

func (s *testScope) Release() error {
	s.once.Do(func() { s.host.mu.Lock(); s.host.releases++; s.host.owned = false; s.host.mu.Unlock() })
	return nil
}

type testHost struct {
	mu              sync.Mutex
	entered         chan struct{}
	join            chan struct{}
	admissionClosed bool
	owned           bool
	releases        int
}

func (h *testHost) Acquire(ctx context.Context) (ExclusiveScope, error) {
	h.mu.Lock()
	h.admissionClosed = true
	h.mu.Unlock()
	if h.entered != nil {
		close(h.entered)
	}
	if h.join != nil {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-h.join:
		}
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.owned {
		return nil, errors.New("already owned")
	}
	h.owned = true
	return &testScope{host: h}, nil
}
func (h *testHost) Admit() bool { h.mu.Lock(); defer h.mu.Unlock(); return !h.admissionClosed }

type testObserver struct {
	mu     sync.Mutex
	gets   int
	result *ao.WorkerStatus
	err    error
}
type testHandoff struct {
	available bool
	err       error
	calls     int
}

func (h *testHandoff) Available(context.Context, store.RecoveryExecution) (bool, error) {
	h.calls++
	return h.available, h.err
}

func (o *testObserver) GetWorkerStatus(context.Context, string) (*ao.WorkerStatus, error) {
	o.mu.Lock()
	o.gets++
	result, err := o.result, o.err
	o.mu.Unlock()
	if result == nil && err == nil {
		return nil, errors.New("unexpected GET")
	}
	return result, err
}
func newRecoveryStore(t *testing.T) *store.Store {
	t.Helper()
	s, _ := newRecoveryStoreAt(t)
	return s
}
func newRecoveryStoreAt(t *testing.T) (*store.Store, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "recovery.db")
	s, err := store.Open(context.Background(), store.Config{DBPath: path, BusyTimeoutMs: 5000})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s, path
}
func seedProvisionIntent(t *testing.T, s *store.Store) {
	t.Helper()
	ctx := context.Background()
	if err := s.CreateProject(ctx, domain.Project{ProjectID: "p", Name: "p", RootPath: "/p"}); err != nil {
		t.Fatal(err)
	}
	if err := s.CreatePair(ctx, domain.Pair{PairID: "pair", ProjectID: "p", CurrentPhaseID: "P03", State: "ACTIVE"}); err != nil {
		t.Fatal(err)
	}
	if err := s.ReservePairProvisioning(ctx, domain.PairProvisioningOperation{OperationID: "intent", PairID: "pair", ClientToken: "token"}, "caller-A"); err != nil {
		t.Fatal(err)
	}
}

func TestQuiescenceBarrierBeforeIntentClassification(t *testing.T) {
	ctx := context.Background()
	s := newRecoveryStore(t)
	seedProvisionIntent(t, s)
	h := &testHost{entered: make(chan struct{}), join: make(chan struct{})}
	observer := &testObserver{}
	r := &Runner{Store: s, AO: observer, Host: h, ActivityPollInterval: time.Second, ExecutionDeadline: time.Minute, Actor: "supervisor", InvocationID: func() string { return "run-barrier" }}
	result := make(chan error, 1)
	go func() { _, err := r.Run(ctx); result <- err }()
	<-h.entered // Caller A has committed intent but is held before its effect.
	if h.Admit() {
		t.Fatal("caller C admitted while quiescence drain is active")
	}
	op, err := s.GetPairProvisioningOperation(ctx, "intent")
	if err != nil || op.Stage != domain.ProvisionRequested {
		t.Fatalf("classified before A joined: %+v %v", op, err)
	}
	observer.mu.Lock()
	gets := observer.gets
	observer.mu.Unlock()
	if gets != 0 {
		t.Fatal("AO GET before join")
	}
	close(h.join)
	if err = <-result; err != nil {
		t.Fatal(err)
	}
	op, err = s.GetPairProvisioningOperation(ctx, "intent")
	if err != nil || op.Stage != domain.ProvisionFailed {
		t.Fatalf("classification after join: %+v %v", op, err)
	}
	h.mu.Lock()
	releases, closed := h.releases, h.admissionClosed
	h.mu.Unlock()
	if releases != 1 || !closed {
		t.Fatalf("ownership release=%d admissionClosed=%v", releases, closed)
	}
}

func TestQuiescenceMissingAndCancelledDrainFailClosed(t *testing.T) {
	s := newRecoveryStore(t)
	seedProvisionIntent(t, s)
	o := &testObserver{}
	r := &Runner{Store: s, AO: o, ActivityPollInterval: time.Second, ExecutionDeadline: time.Minute, Actor: "supervisor"}
	if report, err := r.Run(context.Background()); err == nil || report.Complete {
		t.Fatal("missing provider accepted")
	}
	h := &testHost{entered: make(chan struct{}), join: make(chan struct{})}
	r.Host = h
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { _, err := r.Run(ctx); done <- err }()
	<-h.entered
	cancel()
	if err := <-done; err == nil {
		t.Fatal("cancelled drain accepted")
	}
	op, _ := s.GetPairProvisioningOperation(context.Background(), "intent")
	if op.Stage != domain.ProvisionRequested {
		t.Fatalf("cancelled drain classified intent: %+v", op)
	}
	if h.Admit() {
		t.Fatal("cancelled drain reopened admission")
	}
}

func seedBoundExecution(t *testing.T, s *store.Store, taskID string) (string, string, string) {
	t.Helper()
	ctx := context.Background()
	pairID := "pair-" + taskID
	contractID := "contract-" + taskID
	attemptID := "attempt-" + taskID
	sessionID := "session-" + taskID
	generation := "gen-" + taskID
	if err := s.CreateProject(ctx, domain.Project{ProjectID: "project-" + taskID, Name: "project", RootPath: "/project"}); err != nil {
		t.Fatal(err)
	}
	if err := s.CreatePair(ctx, domain.Pair{PairID: pairID, ProjectID: "project-" + taskID, CurrentPhaseID: "P03", State: "ACTIVE"}); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateTask(ctx, domain.Task{TaskID: taskID, PhaseID: "P03", PairID: pairID, State: domain.StateDraft}); err != nil {
		t.Fatal(err)
	}
	if err := s.InsertTaskContract(ctx, domain.TaskContract{ContractID: contractID, TaskID: taskID, RevisionNumber: 1, BaseSHA: "583e700eb125a08cc6bd7d63b6b27a6f3d4cc527", AllowedScope: []string{"internal/recovery/**"}}); err != nil {
		t.Fatal(err)
	}
	if err := s.TransitionTask(ctx, taskID, domain.StateDraft, domain.StateReady); err != nil {
		t.Fatal(err)
	}
	if err := s.ReservePairProvisioning(ctx, domain.PairProvisioningOperation{OperationID: "provision-" + taskID, PairID: pairID, ClientToken: "client-" + taskID}, "fixture"); err != nil {
		t.Fatal(err)
	}
	if err := s.ConfirmPairProvisioning(ctx, "provision-"+taskID, domain.WorkerSession{PairID: pairID, SessionID: sessionID, RuntimeType: "agy_tui", WorkerAgentID: "agy", Status: domain.WorkerSessionIdle, TerminalGeneration: generation}, "fixture", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	path, err := store.CanonicalExpectedReportPath(taskID, attemptID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.PrepareBoundDispatch(ctx, taskID, contractID, attemptID, path, time.Now().UTC(), store.DispatchBinding{OperationID: "dispatch-" + taskID, SessionID: sessionID, TerminalGeneration: generation}); err != nil {
		t.Fatal(err)
	}
	return sessionID, generation, attemptID
}

func TestRunnerUnknownDeliveryAtomicAndZeroEffectReplay(t *testing.T) {
	ctx := context.Background()
	s := newRecoveryStore(t)
	session, generation, attempt := seedBoundExecution(t, s, "unknown")
	if err := s.RecordSendRequested(ctx, "dispatch-unknown", session, generation, "idle", false, "fixture", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	h := &testHost{}
	o := &testObserver{}
	r := &Runner{Store: s, AO: o, Host: h, ActivityPollInterval: time.Second, ExecutionDeadline: time.Minute, Actor: "supervisor"}
	for i := 0; i < 2; i++ {
		report, err := r.Run(ctx)
		if err != nil || !report.Complete {
			t.Fatalf("run %d: %+v %v", i, report, err)
		}
	}
	task, err := s.GetTask(ctx, "unknown")
	if err != nil || task.State != domain.StateHumanRequired {
		t.Fatalf("D5 state: %+v %v", task, err)
	}
	a, err := s.GetTaskAttempt(ctx, attempt)
	if err != nil || a.EndedAt == nil || a.RecoveryDisposition == nil || *a.RecoveryDisposition != "UNCERTAIN_DELIVERY_CRASH" || a.QuarantineState != domain.QuarantineQuarantined {
		t.Fatalf("D5 attempt: %+v %v", a, err)
	}
	op, err := s.GetDispatchOperation(ctx, "dispatch-unknown")
	if err != nil || op.ResolutionState == nil || *op.ResolutionState != "DELIVERY_OUTCOME_UNKNOWN" {
		t.Fatalf("D5 operation: %+v %v", op, err)
	}
	o.mu.Lock()
	gets := o.gets
	o.mu.Unlock()
	if gets != 0 {
		t.Fatalf("intent replay made %d AO GET calls", gets)
	}
	events, err := s.ListAuditEvents(ctx, 0, store.MaxAuditLimit)
	if err != nil {
		t.Fatal(err)
	}
	transitions := 0
	quarantine := 0
	for _, e := range events {
		if e.Event.EventType == "TASK_STATE_TRANSITION" && e.Event.TaskID == "unknown" {
			transitions++
		}
		if e.Event.EventType == "UNCERTAIN_DELIVERY_QUARANTINE_IMPOSED" && e.Event.TaskID == "unknown" {
			quarantine++
		}
	}
	if transitions != 2 || quarantine != 1 {
		t.Fatalf("D5 audit transitions=%d quarantine=%d", transitions, quarantine)
	}
}

func TestRunnerPostSendOutageAndRecoveryOnDispatchedAndRunning(t *testing.T) {
	ctx := context.Background()
	s := newRecoveryStore(t)
	session, generation, attempt := seedBoundExecution(t, s, "observed")
	if err := s.RecordSendRequested(ctx, "dispatch-observed", session, generation, "idle", false, "fixture", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordSendConfirmed(ctx, "dispatch-observed", "fixture", true, time.Now().UTC(), domain.ExecutionBudgetPolicy{Duration: time.Hour, PolicyRef: "fixture-policy"}); err != nil {
		t.Fatal(err)
	}
	o := &testObserver{err: errors.New("AO unavailable")}
	r := &Runner{Store: s, AO: o, Host: &testHost{}, ActivityPollInterval: time.Second, ExecutionDeadline: time.Minute, Actor: "supervisor"}
	report, err := r.Run(ctx)
	if err != nil || !report.Complete || !report.PendingAO {
		t.Fatalf("outage: %+v %v", report, err)
	}
	a, _ := s.GetTaskAttempt(ctx, attempt)
	if a.RecoveryDisposition == nil || *a.RecoveryDisposition != "RECOVERY_PENDING" {
		t.Fatalf("DISPATCHED hold: %+v", a)
	}
	o.mu.Lock()
	o.err = nil
	o.result = &ao.WorkerStatus{ID: session, TerminalGeneration: generation, Activity: ao.ActivitySnapshot{State: ao.ActivityStateActive}}
	o.mu.Unlock()
	report, err = r.Run(ctx)
	if err != nil || !report.Complete {
		t.Fatalf("recovery: %+v %v", report, err)
	}
	task, _ := s.GetTask(ctx, "observed")
	a, _ = s.GetTaskAttempt(ctx, attempt)
	if task.State != domain.StateRunning || a.RecoveryDisposition != nil {
		t.Fatalf("RUNNING recovery: %+v %+v", task, a)
	}
	o.mu.Lock()
	o.err = errors.New("AO unavailable")
	o.result = nil
	o.mu.Unlock()
	report, err = r.Run(ctx)
	if err != nil || !report.PendingAO {
		t.Fatalf("RUNNING outage: %+v %v", report, err)
	}
	o.mu.Lock()
	o.err = nil
	o.result = &ao.WorkerStatus{ID: session, TerminalGeneration: generation, Activity: ao.ActivitySnapshot{State: ao.ActivityStateWaitingInput}}
	o.mu.Unlock()
	report, err = r.Run(ctx)
	if err != nil || !report.Complete {
		t.Fatalf("RUNNING recovery: %+v %v", report, err)
	}
	task, _ = s.GetTask(ctx, "observed")
	a, _ = s.GetTaskAttempt(ctx, attempt)
	if task.State != domain.StateRunning || a.RecoveryDisposition == nil || *a.RecoveryDisposition != "AO_WAITING_INPUT_OBSERVED" {
		t.Fatalf("RUNNING waiting: %+v %+v", task, a)
	}
}

func TestMissedActiveWindowHandoffAvailability(t *testing.T) {
	for _, tc := range []struct {
		name      string
		handoff   *testHandoff
		wantState domain.TaskState
		wantErr   bool
	}{
		{name: "missing", wantState: domain.StateDispatched, wantErr: true},
		{name: "unavailable", handoff: &testHandoff{}, wantState: domain.StateFailed},
		{name: "available", handoff: &testHandoff{available: true}, wantState: domain.StateRunning},
		{name: "error", handoff: &testHandoff{err: errors.New("handoff unavailable")}, wantState: domain.StateDispatched, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			s := newRecoveryStore(t)
			session, generation, attempt := seedBoundExecution(t, s, "handoff")
			if err := s.RecordSendRequested(ctx, "dispatch-handoff", session, generation, "idle", false, "fixture", time.Now()); err != nil {
				t.Fatal(err)
			}
			if err := s.RecordSendConfirmed(ctx, "dispatch-handoff", "fixture", true, time.Now(), domain.ExecutionBudgetPolicy{Duration: time.Hour, PolicyRef: "fixture-policy"}); err != nil {
				t.Fatal(err)
			}
			o := &testObserver{result: &ao.WorkerStatus{ID: session, TerminalGeneration: generation, Activity: ao.ActivitySnapshot{State: ao.ActivityStateIdle}}}
			var handoff HandoffAvailability
			if tc.handoff != nil {
				handoff = tc.handoff
			}
			r := &Runner{Store: s, AO: o, Handoff: handoff, Host: &testHost{}, ActivityPollInterval: time.Second, ExecutionDeadline: time.Minute, Actor: "supervisor"}
			report, err := r.Run(ctx)
			if (err != nil) != tc.wantErr || report.Complete == tc.wantErr {
				t.Fatalf("handoff outcome: %+v %v", report, err)
			}
			task, _ := s.GetTask(ctx, "handoff")
			a, _ := s.GetTaskAttempt(ctx, attempt)
			if task.State != tc.wantState {
				t.Fatalf("state=%s want=%s", task.State, tc.wantState)
			}
			if !tc.wantErr && (a.RecoveryDisposition == nil || *a.RecoveryDisposition != "MISSED_ACTIVE_WINDOW") {
				t.Fatalf("missing ambiguity disposition: %+v", a)
			}
			if tc.handoff != nil && tc.handoff.calls != 1 {
				t.Fatalf("handoff checks=%d", tc.handoff.calls)
			}
		})
	}
}

func seedLiveStopIntent(t *testing.T, s *store.Store, taskID string) (domain.StopOperation, string, string) {
	t.Helper()
	ctx := context.Background()
	session, generation, attempt := seedBoundExecution(t, s, taskID)
	if err := s.RecordSendRequested(ctx, "dispatch-"+taskID, session, generation, "idle", false, "fixture", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordSendConfirmed(ctx, "dispatch-"+taskID, "fixture", true, time.Now().UTC(), domain.ExecutionBudgetPolicy{Duration: time.Hour, PolicyRef: "fixture-policy"}); err != nil {
		t.Fatal(err)
	}
	if err := s.TransitionTask(ctx, taskID, domain.StateDispatched, domain.StateRunning); err != nil {
		t.Fatal(err)
	}
	contract := "contract-" + taskID
	op := domain.StopOperation{OperationID: "stop-" + taskID, Purpose: domain.RunningAttemptStop, PairID: "pair-" + taskID, TaskID: &taskID, ContractID: &contract, AttemptID: &attempt, SessionID: session, TerminalGeneration: generation, Actor: "supervisor"}
	if err := s.ReserveStopOperation(ctx, op); err != nil {
		t.Fatal(err)
	}
	return op, session, generation
}

func TestRunnerStopRestartAliveAndUnknownDoNotReissue(t *testing.T) {
	ctx := context.Background()
	s := newRecoveryStore(t)
	stop, session, generation := seedLiveStopIntent(t, s, "stopalive")
	o := &testObserver{err: errors.New("AO unavailable")}
	r := &Runner{Store: s, AO: o, Host: &testHost{}, ActivityPollInterval: time.Second, ExecutionDeadline: time.Minute, Actor: "supervisor"}
	report, err := r.Run(ctx)
	if err != nil || !report.PendingAO {
		t.Fatalf("unknown observation: %+v %v", report, err)
	}
	op, err := s.GetStopOperation(ctx, stop.OperationID)
	if err != nil || op.ResolutionState != domain.StopResolutionInFlight {
		t.Fatalf("unknown changed stop: %+v %v", op, err)
	}
	o.mu.Lock()
	o.err = nil
	o.result = &ao.WorkerStatus{ID: session, TerminalGeneration: generation, Activity: ao.ActivitySnapshot{State: ao.ActivityStateActive}}
	o.mu.Unlock()
	report, err = r.Run(ctx)
	if err != nil || !report.Complete {
		t.Fatalf("alive classification: %+v %v", report, err)
	}
	op, err = s.GetStopOperation(ctx, stop.OperationID)
	if err != nil || op.Stage != domain.StopRequested || op.ResolutionState != domain.StopResolutionReissueRequiresHuman {
		t.Fatalf("alive stop: %+v %v", op, err)
	}
	task, _ := s.GetTask(ctx, *stop.TaskID)
	if task.State != domain.StateRunning {
		t.Fatalf("live task changed: %s", task.State)
	}
	report, err = r.Run(ctx)
	if err != nil || !report.Complete {
		t.Fatalf("replay: %+v %v", report, err)
	}
}

func TestRunnerStopConfirmationStrictDeadline(t *testing.T) {
	for _, tc := range []struct {
		name      string
		delta     time.Duration
		confirmed bool
	}{{"before_1ns", -time.Nanosecond, true}, {"equal", 0, false}, {"after_1ns", time.Nanosecond, false}} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			s := newRecoveryStore(t)
			stop, session, generation := seedLiveStopIntent(t, s, "deadline")
			call := time.Now().UTC().Add(-time.Second)
			deadline := call.Add(time.Second)
			if err := s.CommitStopCallAccepted(ctx, stop.OperationID, call, deadline); err != nil {
				t.Fatal(err)
			}
			o := &testObserver{result: &ao.WorkerStatus{ID: session, TerminalGeneration: generation, IsTerminated: true, Activity: ao.ActivitySnapshot{State: ao.ActivityStateExited}}}
			r := &Runner{Store: s, AO: o, Host: &testHost{}, ActivityPollInterval: time.Second, ExecutionDeadline: time.Minute, Actor: "supervisor", Now: func() time.Time { return deadline.Add(tc.delta) }}
			report, err := r.Run(ctx)
			if err != nil || !report.Complete {
				t.Fatalf("run: %+v %v", report, err)
			}
			got, err := s.GetStopOperation(ctx, stop.OperationID)
			if err != nil {
				t.Fatal(err)
			}
			if tc.confirmed {
				if got.Stage != domain.StopTerminationConfirmed || got.ResolutionState != domain.StopResolutionTerminationConfirmed {
					t.Fatalf("before deadline: %+v", got)
				}
			} else {
				if got.Stage != domain.StopCallSucceeded || got.ResolutionState != domain.StopResolutionConfirmationTimeout {
					t.Fatalf("at/after deadline: %+v", got)
				}
			}
		})
	}
}

type serialHost struct {
	token    chan struct{}
	join     chan struct{}
	acquired chan struct{}
	mu       sync.Mutex
	releases int
}
type serialScope struct {
	host *serialHost
	once sync.Once
}

func (s *serialScope) Release() error {
	s.once.Do(func() { s.host.mu.Lock(); s.host.releases++; s.host.mu.Unlock(); s.host.token <- struct{}{} })
	return nil
}
func (h *serialHost) Acquire(ctx context.Context) (ExclusiveScope, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-h.token:
	}
	h.acquired <- struct{}{}
	select {
	case <-ctx.Done():
		h.token <- struct{}{}
		return nil, ctx.Err()
	case <-h.join:
	}
	return &serialScope{host: h}, nil
}

func TestCompetingRunsSerializeAndClassifyProvisionOnce(t *testing.T) {
	ctx := context.Background()
	s := newRecoveryStore(t)
	seedProvisionIntent(t, s)
	h := &serialHost{token: make(chan struct{}, 1), join: make(chan struct{}), acquired: make(chan struct{}, 2)}
	h.token <- struct{}{}
	r1 := &Runner{Store: s, AO: &testObserver{}, Host: h, ActivityPollInterval: time.Second, ExecutionDeadline: time.Minute, Actor: "supervisor"}
	r2 := &Runner{Store: s, AO: &testObserver{}, Host: h, ActivityPollInterval: time.Second, ExecutionDeadline: time.Minute, Actor: "supervisor"}
	results := make(chan error, 2)
	go func() { _, e := r1.Run(ctx); results <- e }()
	go func() { _, e := r2.Run(ctx); results <- e }()
	<-h.acquired
	op, _ := s.GetPairProvisioningOperation(ctx, "intent")
	if op.Stage != domain.ProvisionRequested {
		t.Fatalf("classified before drain: %+v", op)
	}
	close(h.join)
	if e := <-results; e != nil {
		t.Fatal(e)
	}
	if e := <-results; e != nil {
		t.Fatal(e)
	}
	op, _ = s.GetPairProvisioningOperation(ctx, "intent")
	if op.Stage != domain.ProvisionFailed {
		t.Fatalf("classification: %+v", op)
	}
	h.mu.Lock()
	releases := h.releases
	h.mu.Unlock()
	if releases != 2 {
		t.Fatalf("release count=%d", releases)
	}
	events, e := s.ListAuditEvents(ctx, 0, store.MaxAuditLimit)
	if e != nil {
		t.Fatal(e)
	}
	failed := 0
	completed := 0
	for _, x := range events {
		if x.Event.EventType == domain.AuditPairSessionProvisionFailed {
			failed++
		}
		if x.Event.EventType == "STARTUP_RECOVERY_SWEEP_COMPLETED" {
			completed++
		}
	}
	if failed != 1 || completed != 2 {
		t.Fatalf("failed=%d completed=%d", failed, completed)
	}
}

func TestCompletionAuditFailureDoesNotReportSuccessOrReopenAdmission(t *testing.T) {
	s, path := newRecoveryStoreAt(t)
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec(`CREATE TRIGGER reject_sweep_completion BEFORE INSERT ON audit_events WHEN NEW.event_type='STARTUP_RECOVERY_SWEEP_COMPLETED' BEGIN SELECT RAISE(ABORT,'injected completion audit failure'); END`); err != nil {
		t.Fatal(err)
	}
	h := &testHost{}
	r := &Runner{Store: s, AO: &testObserver{}, Host: h, ActivityPollInterval: time.Second, ExecutionDeadline: time.Minute, Actor: "supervisor"}
	report, err := r.Run(context.Background())
	if err == nil || report.Complete {
		t.Fatalf("completion audit failure: %+v %v", report, err)
	}
	h.mu.Lock()
	released, closed := h.releases, h.admissionClosed
	h.mu.Unlock()
	if released != 1 || !closed {
		t.Fatalf("release=%d closed=%v", released, closed)
	}
}

func TestPreSendProtocolHoldSurvivesFreshGET(t *testing.T) {
	ctx := context.Background()
	s := newRecoveryStore(t)
	session, generation, attempt := seedBoundExecution(t, s, "protocol")
	o := &testObserver{err: &ao.ProtocolError{StatusCode: 200, Method: "GET", Path: "/sessions"}}
	r := &Runner{Store: s, AO: o, Host: &testHost{}, ActivityPollInterval: time.Second, ExecutionDeadline: time.Minute, Actor: "supervisor"}
	if report, err := r.Run(ctx); err != nil || !report.Complete || !report.PendingAO {
		t.Fatalf("protocol classification: %+v %v", report, err)
	}
	a, _ := s.GetTaskAttempt(ctx, attempt)
	if a.RecoveryDisposition == nil || *a.RecoveryDisposition != "PRE_SEND_PROTOCOL_UNVERIFIED" {
		t.Fatalf("protocol hold absent: %+v", a)
	}
	o.mu.Lock()
	o.err = nil
	o.result = &ao.WorkerStatus{ID: session, TerminalGeneration: generation, Activity: ao.ActivitySnapshot{State: ao.ActivityStateIdle}}
	o.mu.Unlock()
	if report, err := r.Run(ctx); err != nil || !report.Complete {
		t.Fatalf("fresh GET classification: %+v %v", report, err)
	}
	a, _ = s.GetTaskAttempt(ctx, attempt)
	if a.RecoveryDisposition == nil || *a.RecoveryDisposition != "PRE_SEND_PROTOCOL_UNVERIFIED" {
		t.Fatalf("protocol hold downgraded: %+v", a)
	}
	if err := s.RecordSendRequested(ctx, "dispatch-protocol", session, generation, "idle", false, "supervisor", time.Now()); err == nil {
		t.Fatal("send bypassed protocol hold")
	}
}

func TestPreSendOutageFreshExactIdleResolvesOnlyRecoveryPending(t *testing.T) {
	ctx := context.Background()
	s := newRecoveryStore(t)
	session, generation, attempt := seedBoundExecution(t, s, "presend")
	o := &testObserver{err: errors.New("transport outage")}
	r := &Runner{Store: s, AO: o, Host: &testHost{}, ActivityPollInterval: time.Second, ExecutionDeadline: time.Minute, Actor: "supervisor"}
	if report, err := r.Run(ctx); err != nil || !report.PendingAO {
		t.Fatalf("outage: %+v %v", report, err)
	}
	a, _ := s.GetTaskAttempt(ctx, attempt)
	if a.RecoveryDisposition == nil || *a.RecoveryDisposition != "RECOVERY_PENDING" {
		t.Fatalf("missing pre-send hold: %+v", a)
	}
	o.mu.Lock()
	o.err = nil
	o.result = &ao.WorkerStatus{ID: session, TerminalGeneration: generation, Activity: ao.ActivitySnapshot{State: ao.ActivityStateIdle}}
	o.mu.Unlock()
	if report, err := r.Run(ctx); err != nil || !report.Complete {
		t.Fatalf("fresh GET: %+v %v", report, err)
	}
	a, _ = s.GetTaskAttempt(ctx, attempt)
	if a.RecoveryDisposition != nil || a.EndedAt != nil {
		t.Fatalf("pre-send hold not resolved: %+v", a)
	}
}

func TestStopIntentCallerBarrierPreventsEarlyGETAndClassification(t *testing.T) {
	ctx := context.Background()
	s := newRecoveryStore(t)
	stop, session, generation := seedLiveStopIntent(t, s, "barrierstop")
	h := &testHost{entered: make(chan struct{}), join: make(chan struct{})}
	o := &testObserver{result: &ao.WorkerStatus{ID: session, TerminalGeneration: generation, Activity: ao.ActivitySnapshot{State: ao.ActivityStateActive}}}
	r := &Runner{Store: s, AO: o, Host: h, ActivityPollInterval: time.Second, ExecutionDeadline: time.Minute, Actor: "supervisor"}
	done := make(chan error, 1)
	go func() { _, e := r.Run(ctx); done <- e }()
	<-h.entered
	op, _ := s.GetStopOperation(ctx, stop.OperationID)
	if op.ResolutionState != domain.StopResolutionInFlight {
		t.Fatalf("classified before caller joined: %+v", op)
	}
	o.mu.Lock()
	gets := o.gets
	o.mu.Unlock()
	if gets != 0 {
		t.Fatalf("GET before drain: %d", gets)
	}
	if h.Admit() {
		t.Fatal("new effect caller admitted during drain")
	}
	close(h.join)
	if e := <-done; e != nil {
		t.Fatal(e)
	}
	op, _ = s.GetStopOperation(ctx, stop.OperationID)
	if op.ResolutionState != domain.StopResolutionReissueRequiresHuman {
		t.Fatalf("post-drain resolution: %+v", op)
	}
}

func TestRunnerBlockedOpenAttemptClosesWithoutAOEffect(t *testing.T) {
	ctx := context.Background()
	s := newRecoveryStore(t)
	session, generation, attempt := seedBoundExecution(t, s, "blocked")
	if err := s.RecordSendRequested(ctx, "dispatch-blocked", session, generation, "idle", false, "fixture", time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordSendConfirmed(ctx, "dispatch-blocked", "fixture", true, time.Now(), domain.ExecutionBudgetPolicy{Duration: time.Hour, PolicyRef: "fixture-policy"}); err != nil {
		t.Fatal(err)
	}
	if err := s.TransitionTask(ctx, "blocked", domain.StateDispatched, domain.StateRunning); err != nil {
		t.Fatal(err)
	}
	if err := s.TransitionTask(ctx, "blocked", domain.StateRunning, domain.StateBlocked); err != nil {
		t.Fatal(err)
	}
	o := &testObserver{}
	r := &Runner{Store: s, AO: o, Host: &testHost{}, ActivityPollInterval: time.Second, ExecutionDeadline: time.Minute, Actor: "supervisor"}
	if report, err := r.Run(ctx); err != nil || !report.Complete {
		t.Fatalf("blocked closure: %+v %v", report, err)
	}
	task, _ := s.GetTask(ctx, "blocked")
	a, _ := s.GetTaskAttempt(ctx, attempt)
	if task.State != domain.StateHumanRequired || a.EndedAt == nil || a.RecoveryDisposition == nil || *a.RecoveryDisposition != "AO_BLOCKED_ESCALATED" {
		t.Fatalf("blocked outcome: %+v %+v", task, a)
	}
	o.mu.Lock()
	gets := o.gets
	o.mu.Unlock()
	if gets != 0 {
		t.Fatalf("blocked closure made AO calls: %d", gets)
	}
}

func TestRunnerD5AuditFailureRollsBackAndKeepsAdmissionClosed(t *testing.T) {
	ctx := context.Background()
	s, path := newRecoveryStoreAt(t)
	session, generation, attempt := seedBoundExecution(t, s, "d5rollback")
	if err := s.RecordSendRequested(ctx, "dispatch-d5rollback", session, generation, "idle", false, "fixture", time.Now()); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec(`CREATE TRIGGER reject_d5_audit BEFORE INSERT ON audit_events WHEN NEW.event_type='UNCERTAIN_DELIVERY_QUARANTINE_IMPOSED' BEGIN SELECT RAISE(ABORT,'injected D5 audit failure'); END`); err != nil {
		t.Fatal(err)
	}
	h := &testHost{}
	r := &Runner{Store: s, AO: &testObserver{}, Host: h, ActivityPollInterval: time.Second, ExecutionDeadline: time.Minute, Actor: "supervisor"}
	report, err := r.Run(ctx)
	if err == nil || report.Complete {
		t.Fatalf("D5 audit failure accepted: %+v %v", report, err)
	}
	op, _ := s.GetDispatchOperation(ctx, "dispatch-d5rollback")
	task, _ := s.GetTask(ctx, "d5rollback")
	a, _ := s.GetTaskAttempt(ctx, attempt)
	if op.ResolutionState != nil || task.State != domain.StateDispatched || a.EndedAt != nil {
		t.Fatalf("D5 partial state: %+v %+v %+v", op, task, a)
	}
	if h.Admit() {
		t.Fatal("fatal Run reopened admission")
	}
}

func TestStopOutageTimeoutOnlyAtStrictDeadline(t *testing.T) {
	ctx := context.Background()
	s := newRecoveryStore(t)
	stop, _, _ := seedLiveStopIntent(t, s, "stopoutage")
	call := time.Now().UTC().Add(-time.Second)
	deadline := call.Add(time.Second)
	if err := s.CommitStopCallAccepted(ctx, stop.OperationID, call, deadline); err != nil {
		t.Fatal(err)
	}
	when := deadline.Add(-time.Nanosecond)
	r := &Runner{Store: s, AO: &testObserver{err: errors.New("AO unavailable")}, Host: &testHost{}, ActivityPollInterval: time.Second, ExecutionDeadline: time.Minute, Actor: "supervisor", Now: func() time.Time { return when }}
	report, err := r.Run(ctx)
	if err != nil || !report.Complete || !report.PendingAO {
		t.Fatalf("before deadline: %+v %v", report, err)
	}
	op, _ := s.GetStopOperation(ctx, stop.OperationID)
	if op.ResolutionState != domain.StopResolutionInFlight {
		t.Fatalf("early timeout: %+v", op)
	}
	when = deadline
	report, err = r.Run(ctx)
	if err != nil || !report.Complete {
		t.Fatalf("at deadline: %+v %v", report, err)
	}
	op, _ = s.GetStopOperation(ctx, stop.OperationID)
	if op.ResolutionState != domain.StopResolutionConfirmationTimeout {
		t.Fatalf("missing timeout: %+v", op)
	}
}

func TestRunnerPreSendObservationMatrix(t *testing.T) {
	for _, tc := range []struct {
		name            string
		activity        ao.ActivityState
		terminated      bool
		wrongIdentity   bool
		wrongGeneration bool
		absent          bool
		want            domain.TaskState
		disposition     string
	}{
		{"idle", ao.ActivityStateIdle, false, false, false, false, domain.StateDispatched, ""},
		{"waiting", ao.ActivityStateWaitingInput, false, false, false, false, domain.StateDispatched, ""},
		{"active", ao.ActivityStateActive, false, false, false, false, domain.StateDispatched, ""},
		{"blocked", ao.ActivityStateBlocked, false, false, false, false, domain.StateDispatched, ""},
		{"exited", ao.ActivityStateExited, false, false, false, false, domain.StateDispatched, ""},
		{"terminated", ao.ActivityStateExited, true, false, false, false, domain.StateFailed, "WORKER_TERMINATION_UNKNOWN"},
		{"identity", ao.ActivityStateIdle, false, true, false, false, domain.StateFailed, "PRE_SEND_IDENTITY_MISMATCH"},
		{"generation", ao.ActivityStateIdle, false, false, true, false, domain.StateFailed, "STALE_EXECUTION_GENERATION"},
		{"404", ao.ActivityStateIdle, false, false, false, true, domain.StateFailed, "SESSION_ABSENT"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			s := newRecoveryStore(t)
			session, generation, attempt := seedBoundExecution(t, s, "presendmatrix")
			o := &testObserver{}
			if tc.absent {
				o.err = &ao.APIError{StatusCode: 404}
			} else {
				o.result = &ao.WorkerStatus{ID: session, TerminalGeneration: generation, IsTerminated: tc.terminated, Activity: ao.ActivitySnapshot{State: tc.activity}}
				if tc.wrongIdentity {
					o.result.ID = "foreign"
				}
				if tc.wrongGeneration {
					o.result.TerminalGeneration = "foreign"
				}
			}
			r := &Runner{Store: s, AO: o, Host: &testHost{}, ActivityPollInterval: time.Second, ExecutionDeadline: time.Minute, Actor: "supervisor"}
			if report, err := r.Run(ctx); err != nil || !report.Complete {
				t.Fatalf("matrix Run: %+v %v", report, err)
			}
			task, _ := s.GetTask(ctx, "presendmatrix")
			a, _ := s.GetTaskAttempt(ctx, attempt)
			d := ""
			if a.RecoveryDisposition != nil {
				d = *a.RecoveryDisposition
			}
			if task.State != tc.want || d != tc.disposition {
				t.Fatalf("state=%s disposition=%q want %s/%q", task.State, d, tc.want, tc.disposition)
			}
		})
	}
}

func TestRestoreClassificationPrecedesProvisioningSweep(t *testing.T) {
	ctx := context.Background()
	s, path := newRecoveryStoreAt(t)
	seedProvisionIntent(t, s)
	if err := s.CreateProject(ctx, domain.Project{ProjectID: "restore-project", Name: "restore", RootPath: "/restore"}); err != nil {
		t.Fatal(err)
	}
	if err := s.CreatePair(ctx, domain.Pair{PairID: "restore-pair", ProjectID: "restore-project", CurrentPhaseID: "P03", State: "ACTIVE"}); err != nil {
		t.Fatal(err)
	}
	if err := s.ReservePairProvisioning(ctx, domain.PairProvisioningOperation{OperationID: "restore-provision", PairID: "restore-pair", ClientToken: "restore-token"}, "fixture"); err != nil {
		t.Fatal(err)
	}
	if err := s.ConfirmPairProvisioning(ctx, "restore-provision", domain.WorkerSession{PairID: "restore-pair", SessionID: "restore-session", RuntimeType: "agy_tui", WorkerAgentID: "agy", Status: domain.WorkerSessionTerminated, TerminalGeneration: "restore-generation"}, "fixture", time.Now()); err != nil {
		t.Fatal(err)
	}
	req := domain.RestoreReservation{AuthorizationID: "restore-auth", OperationID: "restore-op", PairID: "restore-pair", SessionID: "restore-session", ExpectedGeneration: "restore-generation", RiskScope: "POSSIBLE_PROMPT_REPLAY", AuthorizedPrincipal: "verified-subject", Actor: "supervisor"}
	if err := s.ReservePairRestore(ctx, req); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec(`CREATE TRIGGER reject_restore_startup BEFORE INSERT ON audit_events WHEN NEW.event_type='PAIR_RESTORE_OUTCOME_UNKNOWN' BEGIN SELECT RAISE(ABORT,'injected restore startup failure'); END`); err != nil {
		t.Fatal(err)
	}
	r := &Runner{Store: s, AO: &testObserver{}, Host: &testHost{}, ActivityPollInterval: time.Second, ExecutionDeadline: time.Minute, Actor: "supervisor"}
	if report, err := r.Run(ctx); err == nil || report.Complete {
		t.Fatalf("restore audit failure: %+v %v", report, err)
	}
	provision, _ := s.GetPairProvisioningOperation(ctx, "intent")
	if provision.Stage != domain.ProvisionRequested {
		t.Fatalf("provisioning ran before restore: %+v", provision)
	}
	if _, err = db.Exec(`DROP TRIGGER reject_restore_startup`); err != nil {
		t.Fatal(err)
	}
	if report, err := r.Run(ctx); err != nil || !report.Complete {
		t.Fatalf("repeated Run: %+v %v", report, err)
	}
	restore, _ := s.GetPairRestoreOperation(ctx, "restore-op")
	provision, _ = s.GetPairProvisioningOperation(ctx, "intent")
	if restore.ResolutionState != domain.RestoreOutcomeUnknown || provision.Stage != domain.ProvisionFailed {
		t.Fatalf("recovery ordering: %+v %+v", restore, provision)
	}
}

type legacyHost struct {
	mu                             sync.Mutex
	acquire, maintenance, releases int
	entered, drain                 chan struct{}
}
type legacyScope struct {
	host *legacyHost
	once sync.Once
}

func (s *legacyScope) Release() error {
	s.once.Do(func() { s.host.mu.Lock(); s.host.releases++; s.host.mu.Unlock() })
	return nil
}
func (h *legacyHost) Acquire(context.Context) (ExclusiveScope, error) {
	h.mu.Lock()
	h.acquire++
	h.mu.Unlock()
	return &legacyScope{host: h}, nil
}
func (h *legacyHost) AcquireMaintenance(ctx context.Context, _, _ string) (ExclusiveScope, error) {
	h.mu.Lock()
	h.maintenance++
	h.mu.Unlock()
	if h.entered != nil {
		close(h.entered)
	}
	if h.drain != nil {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-h.drain:
		}
	}
	return &legacyScope{host: h}, nil
}

type legacyOperator struct {
	principal string
	allowed   bool
}

func (o legacyOperator) VerifiedRestorePrincipal(_ context.Context, _, _, _, scope string) (string, bool, error) {
	return o.principal, o.allowed && scope == "LEGACY_EXECUTION_BUDGET_RECONCILIATION", nil
}

func TestLegacyMaintenanceKeepsAdmissionClosedUntilCompleteRun(t *testing.T) {
	ctx := context.Background()
	s, path := newRecoveryStoreAt(t)
	session, generation, attempt := seedBoundExecution(t, s, "legacy-maintenance")
	if err := s.RecordSendRequested(ctx, "dispatch-legacy-maintenance", session, generation, "idle", false, "fixture", time.Now()); err != nil {
		t.Fatal(err)
	}
	cfg := store.Config{DBPath: path, BusyTimeoutMs: 5000}
	raw, err := sql.Open("sqlite", cfg.DSN())
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	if _, err = raw.Exec(`DROP TRIGGER trg_dispatch_requires_execution_budget`); err != nil {
		t.Fatal(err)
	}
	origin := time.Date(2026, 1, 2, 3, 4, 5, 123456789, time.UTC)
	if _, err = raw.Exec(`UPDATE dispatch_operations SET stage='SEND_CONFIRMED',confirmed_at=? WHERE operation_id='dispatch-legacy-maintenance'`, origin.Format(time.RFC3339Nano)); err != nil {
		t.Fatal(err)
	}
	h := &legacyHost{entered: make(chan struct{}), drain: make(chan struct{})}
	o := &testObserver{result: &ao.WorkerStatus{ID: session, TerminalGeneration: generation, Activity: ao.ActivitySnapshot{State: ao.ActivityStateActive}}}
	r := &Runner{Store: s, AO: o, Host: h, ActivityPollInterval: time.Second, ExecutionDeadline: time.Hour, Actor: "fixture", Operator: legacyOperator{principal: "operator", allowed: true}}
	result, err := r.Run(ctx)
	if err == nil || result.Complete || !strings.Contains(err.Error(), "immutable execution budget") {
		t.Fatalf("legacy Run=%+v err=%v", result, err)
	}
	if r.ready {
		t.Fatal("incomplete Run opened readiness")
	}
	finished := make(chan error, 1)
	go func() {
		finished <- r.BindLegacyBudget(ctx, attempt, "historical-policy-record", domain.ExecutionBudgetPolicy{Duration: 2 * time.Hour, PolicyRef: "historical-p1"})
	}()
	<-h.entered
	select {
	case e := <-finished:
		t.Fatalf("maintenance skipped drain: %v", e)
	default:
	}
	if r.ready {
		t.Fatal("maintenance opened normal readiness")
	}
	close(h.drain)
	if err = <-finished; err != nil {
		t.Fatal(err)
	}
	if r.ready {
		t.Fatal("maintenance release opened readiness")
	}
	b, err := s.GetExecutionBudget(ctx, attempt)
	if err != nil || !b.OriginAt.Equal(origin) || !b.DeadlineAt.Equal(origin.Add(2*time.Hour)) {
		t.Fatalf("budget=%+v err=%v", b, err)
	}
	result, err = r.Run(ctx)
	if err != nil || !result.Complete || !r.ready {
		t.Fatalf("subsequent Run=%+v err=%v ready=%v", result, err, r.ready)
	}
	h.mu.Lock()
	acquire, maintenance, releases := h.acquire, h.maintenance, h.releases
	h.mu.Unlock()
	if acquire != 2 || maintenance != 1 || releases != 3 {
		t.Fatalf("scopes acquire=%d maintenance=%d releases=%d", acquire, maintenance, releases)
	}
}

func TestLegacyMaintenanceMissingTrustedBoundaryFailsClosed(t *testing.T) {
	s := newRecoveryStore(t)
	_, _, attempt := seedBoundExecution(t, s, "no-maintenance-host")
	r := &Runner{Store: s, Host: &testHost{}, Operator: legacyOperator{principal: "operator", allowed: true}}
	if err := r.BindLegacyBudget(context.Background(), attempt, "evidence", domain.ExecutionBudgetPolicy{Duration: time.Hour, PolicyRef: "p"}); err == nil || !strings.Contains(err.Error(), "maintenance scope unavailable") {
		t.Fatalf("missing trusted host=%v", err)
	}
}
