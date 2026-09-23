package recovery

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/trungqwe/ai-supervisor/internal/ao"
	"github.com/trungqwe/ai-supervisor/internal/domain"
	"github.com/trungqwe/ai-supervisor/internal/stop"
	"github.com/trungqwe/ai-supervisor/internal/store"
)

type timeoutTestHost struct {
	mu               sync.Mutex
	grants, releases int
}
type timeoutTestPermit struct{ host *timeoutTestHost }

func (p timeoutTestPermit) Release() error {
	p.host.mu.Lock()
	p.host.releases++
	p.host.mu.Unlock()
	return nil
}
func (h *timeoutTestHost) Acquire(context.Context) (ExclusiveScope, error) {
	return timeoutTestPermit{h}, nil
}
func (h *timeoutTestHost) AcquireEffect(context.Context, string, string) (stop.TimeoutPermit, error) {
	h.mu.Lock()
	h.grants++
	h.mu.Unlock()
	return timeoutTestPermit{h}, nil
}

type timeoutTestAO struct {
	mu                  sync.Mutex
	session, generation string
	gets, kills         int
	initialState        ao.ActivityState
	secondState         ao.ActivityState
	terminated          bool
	preflightEntered    chan struct{}
	preflightRelease    chan struct{}
}

func (a *timeoutTestAO) GetWorkerStatus(context.Context, string) (*ao.WorkerStatus, error) {
	a.mu.Lock()
	a.gets++
	call := a.gets
	if call == 2 && a.preflightEntered != nil {
		close(a.preflightEntered)
	}
	state := ao.ActivityStateActive
	if a.initialState != "" {
		state = a.initialState
	}
	if a.gets >= 2 && a.secondState != "" {
		state = a.secondState
	}
	result := &ao.WorkerStatus{ID: a.session, TerminalGeneration: a.generation, IsTerminated: a.terminated || a.kills > 0, Activity: ao.ActivitySnapshot{State: state}}
	a.mu.Unlock()
	if call <= 2 && a.preflightRelease != nil {
		<-a.preflightRelease
	}
	return result, nil
}
func (a *timeoutTestAO) StopWorker(context.Context, string) (*ao.StopWorkerResult, error) {
	a.mu.Lock()
	a.kills++
	a.mu.Unlock()
	return &ao.StopWorkerResult{SessionID: a.session, Freed: true}, nil
}

func TestTimeoutMonitorBeforeEqualAfterAndNoReplay(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name     string
		offset   time.Duration
		wantKill int
	}{{"before", -time.Nanosecond, 0}, {"equal", 0, 1}, {"after", time.Nanosecond, 1}} {
		t.Run(tc.name, func(t *testing.T) {
			s := newRecoveryStore(t)
			session, generation, attempt := seedBoundExecution(t, s, "budget-"+tc.name)
			origin := time.Date(2026, 1, 1, 0, 0, 0, 123456789, time.UTC)
			if err := s.RecordSendRequested(ctx, "dispatch-budget-"+tc.name, session, generation, "idle", false, "fixture", origin); err != nil {
				t.Fatal(err)
			}
			if err := s.RecordSendConfirmed(ctx, "dispatch-budget-"+tc.name, "fixture", true, origin, domain.ExecutionBudgetPolicy{Duration: time.Hour, PolicyRef: "historical-policy"}); err != nil {
				t.Fatal(err)
			}
			snap, err := s.ListRecoverySnapshot(ctx)
			if err != nil {
				t.Fatal(err)
			}
			var targetExec store.RecoveryExecution
			for _, ex := range snap.Executions {
				if ex.AttemptID == attempt {
					targetExec = ex
					break
				}
			}
			if err := s.ObservePostSend(ctx, targetExec, store.PostSendObservation{SessionID: session, Generation: generation, Activity: "waiting_input"}, "fixture", "obs-"+tc.name, origin); err != nil {
				t.Fatal(err)
			}
			h := &timeoutTestHost{}
			a := &timeoutTestAO{session: session, generation: generation, initialState: ao.ActivityStateWaitingInput}
			r := &Runner{Store: s, Host: h, ready: true}
			c := &stop.Coordinator{Store: s, AO: a, KillTimeout: time.Minute}
			m := &TimeoutMonitor{Owner: r, Stop: c, Interval: time.Second, Actor: "fixture", Now: func() time.Time { return origin.Add(time.Hour).Add(tc.offset) }}
			if err := m.Tick(ctx); err != nil {
				t.Fatal(err)
			}
			a.mu.Lock()
			kills := a.kills
			a.mu.Unlock()
			if kills != tc.wantKill {
				t.Fatalf("kill calls=%d want %d", kills, tc.wantKill)
			}
			if err := m.Tick(ctx); err != nil {
				t.Fatal(err)
			}
			a.mu.Lock()
			replayKills := a.kills
			a.mu.Unlock()
			if replayKills != tc.wantKill {
				t.Fatalf("replay kill=%d", replayKills)
			}
			if tc.wantKill == 1 {
				task, err := s.GetTask(ctx, "budget-"+tc.name)
				if err != nil || task.State != domain.StateFailed {
					t.Fatalf("task=%+v err=%v", task, err)
				}
				att, err := s.GetTaskAttempt(ctx, attempt)
				if err != nil || att.EndedAt == nil {
					t.Fatalf("closure=%+v err=%v", att, err)
				}
			}
		})
	}
}

func TestTimeoutMonitorMissingSharedAdmissionFailsClosed(t *testing.T) {
	s := newRecoveryStore(t)
	r := &Runner{Store: s, Host: &testHost{}, ready: true}
	m := &TimeoutMonitor{Owner: r, Stop: &stop.Coordinator{Store: s, AO: &timeoutTestAO{}}, Interval: time.Second, Actor: "fixture"}
	if err := m.Tick(context.Background()); err == nil {
		t.Fatal("missing shared admission accepted")
	}
}

func TestTimeoutMonitorChangedObservationRetainsUnresolvedIntent(t *testing.T) {
	ctx := context.Background()
	s := newRecoveryStore(t)
	session, generation, attempt := seedBoundExecution(t, s, "changed-observation")
	origin := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := s.RecordSendRequested(ctx, "dispatch-changed-observation", session, generation, "idle", false, "fixture", origin); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordSendConfirmed(ctx, "dispatch-changed-observation", "fixture", true, origin, domain.ExecutionBudgetPolicy{Duration: time.Hour, PolicyRef: "p1"}); err != nil {
		t.Fatal(err)
	}
	if err := s.TransitionTask(ctx, "changed-observation", domain.StateDispatched, domain.StateRunning); err != nil {
		t.Fatal(err)
	}
	h := &timeoutTestHost{}
	a := &timeoutTestAO{session: session, generation: generation, secondState: ao.ActivityStateBlocked}
	r := &Runner{Store: s, Host: h, ready: true}
	m := &TimeoutMonitor{Owner: r, Stop: &stop.Coordinator{Store: s, AO: a, KillTimeout: time.Minute}, Interval: time.Second, Actor: "fixture", Now: func() time.Time { return origin.Add(time.Hour) }}
	if err := m.Tick(ctx); err == nil {
		t.Fatal("changed observation silently accepted")
	}
	a.mu.Lock()
	kills := a.kills
	gets := a.gets
	a.mu.Unlock()
	if gets != 2 || kills != 0 {
		t.Fatalf("gets=%d kills=%d", gets, kills)
	}
	snap, err := s.ListRecoverySnapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.StopOwnedAttemptIDs) != 1 || snap.StopOwnedAttemptIDs[0] != attempt {
		t.Fatalf("intent not retained: %+v", snap.StopOwnedAttemptIDs)
	}
	task, err := s.GetTask(ctx, "changed-observation")
	if err != nil || task.State != domain.StateRunning {
		t.Fatalf("task=%+v err=%v", task, err)
	}
	att, err := s.GetTaskAttempt(ctx, attempt)
	if err != nil || att.EndedAt != nil || att.QuarantineState != domain.QuarantineQuarantined {
		t.Fatalf("attempt=%+v err=%v", att, err)
	}
	if err := m.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	a.mu.Lock()
	replayKills := a.kills
	a.mu.Unlock()
	if replayKills != 0 {
		t.Fatalf("replay kill=%d", replayKills)
	}
}

func TestTimeoutMonitorCompetingCallersOneEffect(t *testing.T) {
	ctx := context.Background()
	s := newRecoveryStore(t)
	session, generation, _ := seedBoundExecution(t, s, "timeout-race")
	origin := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := s.RecordSendRequested(ctx, "dispatch-timeout-race", session, generation, "idle", false, "fixture", origin); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordSendConfirmed(ctx, "dispatch-timeout-race", "fixture", true, origin, domain.ExecutionBudgetPolicy{Duration: time.Hour, PolicyRef: "p1"}); err != nil {
		t.Fatal(err)
	}
	if err := s.TransitionTask(ctx, "timeout-race", domain.StateDispatched, domain.StateRunning); err != nil {
		t.Fatal(err)
	}
	h := &timeoutTestHost{}
	a := &timeoutTestAO{session: session, generation: generation, preflightEntered: make(chan struct{}), preflightRelease: make(chan struct{})}
	r := &Runner{Store: s, Host: h, ready: true}
	m := &TimeoutMonitor{Owner: r, Stop: &stop.Coordinator{Store: s, AO: a, KillTimeout: time.Minute}, Interval: time.Second, Actor: "fixture", Now: func() time.Time { return origin.Add(time.Hour) }}
	results := make(chan error, 2)
	go func() { results <- m.Tick(ctx) }()
	go func() { results <- m.Tick(ctx) }()
	<-a.preflightEntered
	close(a.preflightRelease)
	first, second := <-results, <-results
	if first != nil || second != nil {
		t.Fatalf("competing results: %v %v", first, second)
	}
	a.mu.Lock()
	kills := a.kills
	a.mu.Unlock()
	if kills != 1 {
		t.Fatalf("competing /kill count=%d", kills)
	}
	h.mu.Lock()
	grants, releases := h.grants, h.releases
	h.mu.Unlock()
	if grants != 2 || releases != 2 {
		t.Fatalf("permits grants=%d releases=%d", grants, releases)
	}
}

func TestTimeoutMonitorPreflightTerminationUsesLifecycleWithoutStop(t *testing.T) {
	ctx := context.Background()
	s := newRecoveryStore(t)
	session, generation, attempt := seedBoundExecution(t, s, "timeout-preflight-terminated")
	origin := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := s.RecordSendRequested(ctx, "dispatch-timeout-preflight-terminated", session, generation, "idle", false, "fixture", origin); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordSendConfirmed(ctx, "dispatch-timeout-preflight-terminated", "fixture", true, origin, domain.ExecutionBudgetPolicy{Duration: time.Hour, PolicyRef: "p1"}); err != nil {
		t.Fatal(err)
	}
	if err := s.TransitionTask(ctx, "timeout-preflight-terminated", domain.StateDispatched, domain.StateRunning); err != nil {
		t.Fatal(err)
	}
	h := &timeoutTestHost{}
	a := &timeoutTestAO{session: session, generation: generation, terminated: true}
	r := &Runner{Store: s, Host: h, ready: true}
	m := &TimeoutMonitor{Owner: r, Stop: &stop.Coordinator{Store: s, AO: a, KillTimeout: time.Minute}, Interval: time.Second, Actor: "fixture", Now: func() time.Time { return origin.Add(time.Hour) }}
	if err := m.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	a.mu.Lock()
	kills := a.kills
	a.mu.Unlock()
	if kills != 0 {
		t.Fatalf("preflight termination emitted %d kills", kills)
	}
	task, err := s.GetTask(ctx, "timeout-preflight-terminated")
	if err != nil || task.State != domain.StateFailed {
		t.Fatalf("task=%+v err=%v", task, err)
	}
	att, err := s.GetTaskAttempt(ctx, attempt)
	if err != nil || att.EndedAt == nil {
		t.Fatalf("attempt=%+v err=%v", att, err)
	}
}


func TestTimeoutMonitorWaitingInputFullLifecycleAndRollback(t *testing.T) {
	ctx := context.Background()
	s := newRecoveryStore(t)
	taskID := "waiting-input-lifecycle"
	session, generation, attempt := seedBoundExecution(t, s, taskID)
	origin := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	duration := time.Hour
	deadline := origin.Add(duration)

	if err := s.RecordSendRequested(ctx, "dispatch-"+taskID, session, generation, "idle", false, "fixture", origin); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordSendConfirmed(ctx, "dispatch-"+taskID, "fixture", true, origin, domain.ExecutionBudgetPolicy{Duration: duration, PolicyRef: "historical-policy"}); err != nil {
		t.Fatal(err)
	}

	// 1. Genuine lifecycle: SEND_CONFIRMED -> ObservePostSend(waiting_input)
	snap, err := s.ListRecoverySnapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var exec store.RecoveryExecution
	for _, ex := range snap.Executions {
		if ex.AttemptID == attempt {
			exec = ex
			break
		}
	}
	if exec.AttemptID == "" {
		t.Fatal("execution attempt not found in snapshot")
	}
	if err := s.ObservePostSend(ctx, exec, store.PostSendObservation{SessionID: session, Generation: generation, Activity: "waiting_input"}, "fixture", "obs-waiting", origin); err != nil {
		t.Fatal(err)
	}

	task, err := s.GetTask(ctx, taskID)
	if err != nil || task.State != domain.StateRunning {
		t.Fatalf("task state=%+v err=%v, want RUNNING", task, err)
	}
	att, err := s.GetTaskAttempt(ctx, attempt)
	if err != nil || att.RecoveryDisposition == nil || *att.RecoveryDisposition != "AO_WAITING_INPUT_OBSERVED" {
		t.Fatalf("attempt disposition=%+v err=%v, want AO_WAITING_INPUT_OBSERVED", att, err)
	}

	// 2. Budget is preserved across restart while attempt is open
	budget, err := s.GetExecutionBudget(ctx, attempt)
	if err != nil {
		t.Fatalf("GetExecutionBudget err=%v", err)
	}
	if !budget.OriginAt.Equal(origin) || !budget.DeadlineAt.Equal(deadline) || budget.Duration != duration || budget.PolicyRef != "historical-policy" {
		t.Fatalf("budget mismatch on restart: %+v", budget)
	}

	h := &timeoutTestHost{}
	a := &timeoutTestAO{session: session, generation: generation, initialState: ao.ActivityStateWaitingInput}
	r := &Runner{Store: s, Host: h, ready: true}
	c := &stop.Coordinator{Store: s, AO: a, KillTimeout: time.Minute}
	m := &TimeoutMonitor{Owner: r, Stop: c, Interval: time.Second, Actor: "fixture", Now: func() time.Time { return deadline.Add(-10 * time.Second) }}

	// 2. Before deadline: monitor tick does NOT kill
	if err := m.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	a.mu.Lock()
	kills := a.kills
	a.mu.Unlock()
	if kills != 0 {
		t.Fatalf("early tick killed worker: %d kills", kills)
	}

	// 3. At deadline: monitor tick issues exactly one kill
	m.Now = func() time.Time { return deadline }
	if err := m.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	a.mu.Lock()
	kills = a.kills
	a.mu.Unlock()
	if kills != 1 {
		t.Fatalf("on-time tick kills=%d, want 1", kills)
	}

	// 4. After deadline: repeated tick emits zero replay effects
	m.Now = func() time.Time { return deadline.Add(10 * time.Second) }
	if err := m.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	a.mu.Lock()
	kills = a.kills
	a.mu.Unlock()
	if kills != 1 {
		t.Fatalf("replay tick kills=%d, want 1", kills)
	}

	// 5. Check D11 outcome
	task, err = s.GetTask(ctx, taskID)
	if err != nil || task.State != domain.StateFailed {
		t.Fatalf("task state=%+v err=%v, want FAILED", task, err)
	}
	att, err = s.GetTaskAttempt(ctx, attempt)
	if err != nil || att.EndedAt == nil || att.QuarantineState != domain.QuarantineClean || att.RecoveryDisposition == nil || *att.RecoveryDisposition != "WORKER_STOPPED" {
		t.Fatalf("attempt closure state=%+v err=%v", att, err)
	}

	// 6. Check D11 audit: failure_reason=TIMEOUT and recovery_disposition=WORKER_STOPPED
	events, err := s.ListAuditEvents(ctx, 0, 500)
	if err != nil {
		t.Fatal(err)
	}
	var foundClosureAudit bool
	var stopOpID string
	for _, ev := range events {
		if ev.Event.AttemptID == attempt && ev.Event.EventType == domain.AuditTaskStateTransition {
			if ev.Event.Details["failure_reason"] == "TIMEOUT" && ev.Event.Details["recovery_disposition"] == "WORKER_STOPPED" {
				foundClosureAudit = true
				if id, ok := ev.Event.Details["stop_operation_id"].(string); ok {
					stopOpID = id
				}
			}
		}
	}
	if !foundClosureAudit {
		t.Fatal("D11 closure audit with failure_reason=TIMEOUT not found")
	}
	if stopOpID == "" {
		t.Fatal("stop_operation_id missing from transition audit")
	}

	// 7. Check durable timeout cause on stop operation
	stopOp, err := s.GetStopOperation(ctx, stopOpID)
	if err != nil || stopOp.InitiatingFailureReason == nil || *stopOp.InitiatingFailureReason != "TIMEOUT" {
		t.Fatalf("stop operation initiating_failure_reason=%+v err=%v, want TIMEOUT", stopOp, err)
	}

	// 8. Stale writer is rejected; task cannot transition backwards from terminal state
	if err := s.TransitionTask(ctx, taskID, domain.StateRunning, domain.StateFailed); err == nil {
		t.Fatal("stale task transition accepted")
	}
}
