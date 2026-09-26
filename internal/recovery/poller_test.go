package recovery

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"strings"
	"testing"
	"time"

	"github.com/trungqwe/ai-supervisor/internal/ao"
	"github.com/trungqwe/ai-supervisor/internal/domain"
	"github.com/trungqwe/ai-supervisor/internal/store"
)

type barrierObserver struct {
	mu      sync.Mutex
	calls   int
	entered chan struct{}
	release chan struct{}
	once    sync.Once
	status  *ao.WorkerStatus
}

func (o *barrierObserver) GetWorkerStatus(ctx context.Context, _ string) (*ao.WorkerStatus, error) {
	o.mu.Lock()
	o.calls++
	first := o.calls == 1
	o.mu.Unlock()
	if first {
		o.once.Do(func() { close(o.entered) })
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-o.release:
		}
	}
	return o.status, nil
}

func TestPollerNeverClassifiesStartupIntentAndShutsDown(t *testing.T) {
	s := newRecoveryStore(t)
	seedProvisionIntent(t, s)
	o := &testObserver{}
	owner := &Runner{Store: s, AO: o, Host: &testHost{}, ActivityPollInterval: time.Hour, ExecutionDeadline: time.Minute, Actor: "supervisor"}
	p := &Poller{Store: s, AO: o, Owner: owner, Interval: time.Hour, Actor: "supervisor"}
	if err := p.PollOnce(context.Background()); err == nil {
		t.Fatal("direct poller bypassed startup ownership")
	}
	op, err := s.GetPairProvisioningOperation(context.Background(), "intent")
	if err != nil || op.Stage != domain.ProvisionRequested {
		t.Fatalf("poller reclaimed intent: %+v %v", op, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	if err := p.Start(ctx); err == nil {
		t.Fatal("poller started before Run completed")
	}
	if report, err := owner.Run(ctx); err != nil || !report.Complete {
		t.Fatalf("startup: %+v %v", report, err)
	}
	if err := p.Start(ctx); err != nil {
		t.Fatal(err)
	}
	if err := p.Start(ctx); err == nil {
		t.Fatal("duplicate poller owner accepted")
	}
	cancel()
	if err := p.Stop(); err != nil {
		t.Fatal(err)
	}
	if err := p.Stop(); err != nil {
		t.Fatal(err)
	}
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("poller did not join before restart: %v", err)
	}
	if err := p.Stop(); err != nil {
		t.Fatal(err)
	}
}

func TestRepeatedRunJoinsOwnedPollerBeforeAcquire(t *testing.T) {
	s := newRecoveryStore(t)
	o := &testObserver{}
	h := &testHost{}
	r := &Runner{Store: s, AO: o, Host: h, ActivityPollInterval: time.Second, ExecutionDeadline: time.Minute, Actor: "supervisor"}
	if report, err := r.Run(context.Background()); err != nil || !report.Complete {
		t.Fatalf("first Run: %+v %v", report, err)
	}
	p := &Poller{Store: s, AO: o, Owner: r, Interval: time.Hour, Actor: "supervisor"}
	if err := p.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if report, err := r.Run(context.Background()); err != nil || !report.Complete {
		t.Fatalf("repeated Run: %+v %v", report, err)
	}
	p.mu.Lock()
	running := p.running
	p.mu.Unlock()
	if running {
		t.Fatal("Run acquired while poller worker remained active")
	}
	if err := p.Stop(); err != nil {
		t.Fatal(err)
	}
}

func TestCompetingPollerObservationsProduceOneTransition(t *testing.T) {
	ctx := context.Background()
	s := newRecoveryStore(t)
	session, generation, _ := seedBoundExecution(t, s, "pollrace")
	if err := s.RecordSendRequested(ctx, "dispatch-pollrace", session, generation, "idle", false, "fixture", time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordSendConfirmed(ctx, "dispatch-pollrace", "fixture", true, time.Now(), domain.ExecutionBudgetPolicy{Duration: time.Hour, PolicyRef: "fixture-policy"}); err != nil {
		t.Fatal(err)
	}
	o := &barrierObserver{entered: make(chan struct{}), release: make(chan struct{}), status: &ao.WorkerStatus{ID: session, TerminalGeneration: generation, Activity: ao.ActivitySnapshot{State: ao.ActivityStateActive}}}
	owner := &Runner{Store: s, ready: true}
	p := &Poller{Store: s, AO: o, Owner: owner, Interval: time.Second, Actor: "supervisor"}
	results := make(chan error, 2)
	go func() { results <- p.PollOnce(ctx) }()
	<-o.entered
	go func() { results <- p.PollOnce(ctx) }()
	close(o.release)
	if err := <-results; err != nil {
		t.Fatal(err)
	}
	if err := <-results; err != nil {
		t.Fatal(err)
	}
	task, err := s.GetTask(ctx, "pollrace")
	if err != nil || task.State != domain.StateRunning {
		t.Fatalf("poller state: %+v %v", task, err)
	}
	events, err := s.ListAuditEvents(ctx, 0, store.MaxAuditLimit)
	if err != nil {
		t.Fatal(err)
	}
	transitions := 0
	for _, e := range events {
		if e.Event.EventType == "TASK_STATE_TRANSITION" && e.Event.TaskID == "pollrace" {
			transitions++
		}
	}
	if transitions != 1 {
		t.Fatalf("duplicate transition audit: %d", transitions)
	}
}

func TestPollerShutdownJoinsBlockedObservation(t *testing.T) {
	ctx := context.Background()
	s := newRecoveryStore(t)
	session, generation, _ := seedBoundExecution(t, s, "shutdown")
	if err := s.RecordSendRequested(ctx, "dispatch-shutdown", session, generation, "idle", false, "fixture", time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordSendConfirmed(ctx, "dispatch-shutdown", "fixture", true, time.Now(), domain.ExecutionBudgetPolicy{Duration: time.Hour, PolicyRef: "fixture-policy"}); err != nil {
		t.Fatal(err)
	}
	owner := &Runner{Store: s, AO: &testObserver{result: &ao.WorkerStatus{ID: session, TerminalGeneration: generation, Activity: ao.ActivitySnapshot{State: ao.ActivityStateActive}}}, Host: &testHost{}, ActivityPollInterval: time.Second, ExecutionDeadline: time.Minute, Actor: "supervisor"}
	if report, err := owner.Run(ctx); err != nil || !report.Complete {
		t.Fatalf("startup: %+v %v", report, err)
	}
	o := &barrierObserver{entered: make(chan struct{}), release: make(chan struct{}), status: &ao.WorkerStatus{ID: session, TerminalGeneration: generation, Activity: ao.ActivitySnapshot{State: ao.ActivityStateActive}}}
	p := &Poller{Store: s, AO: o, Owner: owner, Interval: time.Millisecond, Actor: "supervisor"}
	if err := p.Start(ctx); err != nil {
		t.Fatal(err)
	}
	<-o.entered
	if err := p.Stop(); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	p.mu.Lock()
	running := p.running
	p.mu.Unlock()
	if running {
		t.Fatal("poller worker retained after Stop")
	}
}

func TestDirectPollOnceOwnershipAndRepeatedRunBarrier(t *testing.T) {
	ctx := context.Background()
	s := newRecoveryStore(t)
	session, generation, _ := seedBoundExecution(t, s, "direct-barrier")
	if err := s.RecordSendRequested(ctx, "dispatch-direct-barrier", session, generation, "idle", false, "fixture", time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordSendConfirmed(ctx, "dispatch-direct-barrier", "fixture", true, time.Now(), domain.ExecutionBudgetPolicy{Duration: time.Hour, PolicyRef: "fixture-policy"}); err != nil {
		t.Fatal(err)
	}
	o := &barrierObserver{entered: make(chan struct{}), release: make(chan struct{}), status: &ao.WorkerStatus{ID: session, TerminalGeneration: generation, Activity: ao.ActivitySnapshot{State: ao.ActivityStateActive}}}
	p := &Poller{Store: s, AO: o, Interval: time.Second, Actor: "supervisor"}
	if err := p.PollOnce(ctx); err == nil {
		t.Fatal("PollOnce without Owner accepted")
	}
	h := &testHost{entered: make(chan struct{})}
	r := &Runner{Store: s, AO: &testObserver{result: o.status}, Host: h, ActivityPollInterval: time.Second, ExecutionDeadline: time.Minute, Actor: "supervisor", ready: true}
	p.Owner = r
	tick := make(chan error, 1)
	go func() { tick <- p.PollOnce(ctx) }()
	<-o.entered
	run := make(chan error, 1)
	go func() { _, err := r.Run(ctx); run <- err }()
	for i := 0; i < 100000; i++ {
		r.gateMu.Lock()
		running := r.running
		r.gateMu.Unlock()
		if running {
			break
		}
		runtime.Gosched()
	}
	r.gateMu.Lock()
	running := r.running
	r.gateMu.Unlock()
	if !running {
		t.Fatal("Run did not request ownership")
	}
	select {
	case <-h.entered:
		t.Fatal("Run acquired host quiescence before granted tick drained")
	default:
	}
	close(o.release)
	if err := <-tick; err != nil {
		t.Fatal(err)
	}
	if err := <-run; err != nil {
		t.Fatal(err)
	}
	<-h.entered
	r.Host = nil
	if _, err := r.Run(ctx); err == nil {
		t.Fatal("missing provider allowed repeated Run")
	}
	if err := p.PollOnce(ctx); err == nil {
		t.Fatal("PollOnce after failed Run accepted")
	}
}

func TestPollerPreSendPendingAORecoveryWithoutRun(t *testing.T) {
	ctx := context.Background()
	s := newRecoveryStore(t)
	session, generation, attempt := seedBoundExecution(t, s, "pre-poll")
	o := &testObserver{err: errors.New("AO unavailable")}
	r := &Runner{Store: s, AO: o, Host: &testHost{}, ActivityPollInterval: time.Second, ExecutionDeadline: time.Minute, Actor: "supervisor"}
	if report, err := r.Run(ctx); err != nil || !report.PendingAO {
		t.Fatalf("startup pending: %+v %v", report, err)
	}
	a, _ := s.GetTaskAttempt(ctx, attempt)
	if a.RecoveryDisposition == nil || *a.RecoveryDisposition != "RECOVERY_PENDING" {
		t.Fatalf("missing pending hold: %+v", a)
	}
	o.err = nil
	o.result = &ao.WorkerStatus{ID: session, TerminalGeneration: generation, Activity: ao.ActivitySnapshot{State: ao.ActivityStateIdle}}
	p := &Poller{Store: s, AO: o, Owner: r, Interval: time.Second, Actor: "supervisor"}
	if err := p.PollOnce(ctx); err != nil {
		t.Fatal(err)
	}
	a, _ = s.GetTaskAttempt(ctx, attempt)
	op, _ := s.GetDispatchOperation(ctx, "dispatch-pre-poll")
	if a.RecoveryDisposition != nil || op.Stage != domain.DispatchBound {
		t.Fatalf("poller did not resolve only hold: attempt=%+v op=%+v", a, op)
	}
	if err := s.RecordPreSendHold(ctx, op.OperationID, "PRE_SEND_PROTOCOL_UNVERIFIED", "supervisor", time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := p.PollOnce(ctx); err != nil {
		t.Fatal(err)
	}
	a, _ = s.GetTaskAttempt(ctx, attempt)
	if a.RecoveryDisposition == nil || *a.RecoveryDisposition != "PRE_SEND_PROTOCOL_UNVERIFIED" {
		t.Fatalf("successful GET cleared protocol hold: %+v", a)
	}
}

func TestPollerStopOwnedOutcomes(t *testing.T) {
	for _, tc := range []struct {
		name        string
		status      *ao.WorkerStatus
		getErr      error
		deadline    time.Duration
		resolution  domain.StopResolutionState
		disposition string
		quarantine  domain.QuarantineState
	}{
		{name: "terminated-before-deadline", deadline: time.Minute, resolution: domain.StopResolutionTerminationConfirmed, disposition: "WORKER_STOPPED", quarantine: domain.QuarantineClean},
		{name: "timeout", deadline: -time.Second, resolution: domain.StopResolutionConfirmationTimeout, disposition: "STOP_CONFIRMATION_TIMEOUT", quarantine: domain.QuarantineQuarantined},
		{name: "absent", getErr: ao.ErrNotFound, deadline: time.Minute, resolution: domain.StopResolutionTargetAbsent, disposition: "SESSION_ABSENT", quarantine: domain.QuarantineQuarantined},
		{name: "generation-mismatch", deadline: time.Minute, resolution: domain.StopResolutionGenerationMismatch, disposition: "STOP_GENERATION_MISMATCH", quarantine: domain.QuarantineQuarantined},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			s := newRecoveryStore(t)
			stop, session, generation := seedLiveStopIntent(t, s, "stop-poll")
			call := time.Now().UTC().Add(-2 * time.Second)
			deadline := time.Now().UTC().Add(tc.deadline)
			if err := s.CommitStopCallAccepted(ctx, stop.OperationID, call, deadline); err != nil {
				t.Fatal(err)
			}
			status := &ao.WorkerStatus{ID: session, TerminalGeneration: generation, IsTerminated: true}
			if tc.name == "timeout" {
				status.IsTerminated = false
			}
			if tc.name == "generation-mismatch" {
				status.TerminalGeneration = generation + "-new"
			}
			o := &testObserver{result: status, err: tc.getErr}
			owner := &Runner{Store: s, ready: true}
			p := &Poller{Store: s, AO: o, Owner: owner, Interval: time.Second, Actor: "supervisor"}
			if err := p.PollOnce(ctx); err != nil {
				t.Fatal(err)
			}
			after, _ := s.GetStopOperation(ctx, stop.OperationID)
			task, _ := s.GetTask(ctx, *stop.TaskID)
			attempt, _ := s.GetTaskAttempt(ctx, *stop.AttemptID)
			ws, _ := s.GetWorkerSessionByPair(ctx, stop.PairID)
			if after.ResolutionState != tc.resolution || task.State != domain.StateFailed || attempt.EndedAt == nil || attempt.RecoveryDisposition == nil || *attempt.RecoveryDisposition != tc.disposition || attempt.QuarantineState != tc.quarantine || ws.QuarantineState != tc.quarantine {
				t.Fatalf("stop lifecycle lost: stop=%+v task=%+v attempt=%+v", after, task, attempt)
			}
			events, err := s.ListAuditEvents(ctx, 0, store.MaxAuditLimit)
			if err != nil {
				t.Fatal(err)
			}
			seenStop, seenTask := false, false
			for _, event := range events {
				if event.Event.EventType == "TASK_STATE_TRANSITION" && event.Event.TaskID == *stop.TaskID {
					seenTask = true
				}
				if event.Event.EventType == "STOP_OPERATION_CONFIRMED" || event.Event.EventType == "STOP_OPERATION_TARGET_ABSENT" || event.Event.EventType == "STOP_CONFIRMATION_TIMEOUT" || event.Event.EventType == "STOP_OPERATION_RESOLVED" {
					seenStop = true
				}
			}
			if !seenStop || !seenTask {
				t.Fatalf("missing atomic audit: stop=%v task=%v", seenStop, seenTask)
			}
		})
	}
}

func TestStopDrainsPublicTickAndCancellation(t *testing.T) {
	ctx := context.Background()
	s := newRecoveryStore(t)
	session, generation, _ := seedBoundExecution(t, s, "public-stop")
	if err := s.RecordSendRequested(ctx, "dispatch-public-stop", session, generation, "idle", false, "fixture", time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordSendConfirmed(ctx, "dispatch-public-stop", "fixture", true, time.Now(), domain.ExecutionBudgetPolicy{Duration: time.Hour, PolicyRef: "fixture-policy"}); err != nil {
		t.Fatal(err)
	}
	o := &barrierObserver{entered: make(chan struct{}), release: make(chan struct{}), status: &ao.WorkerStatus{ID: session, TerminalGeneration: generation, Activity: ao.ActivitySnapshot{State: ao.ActivityStateActive}}}
	owner := &Runner{Store: s, ready: true}
	p := &Poller{Store: s, AO: o, Owner: owner, Interval: time.Second, Actor: "supervisor"}
	tick := make(chan error, 1)
	go func() { tick <- p.PollOnce(ctx) }()
	<-o.entered
	stopped := make(chan error, 1)
	go func() { stopped <- p.Stop() }()
	for i := 0; i < 100000; i++ {
		p.mu.Lock()
		stopping := p.stopping
		p.mu.Unlock()
		if stopping {
			break
		}
		runtime.Gosched()
	}
	p.mu.Lock()
	stopping := p.stopping
	p.mu.Unlock()
	if !stopping {
		t.Fatal("Stop did not close admission")
	}
	select {
	case <-stopped:
		t.Fatal("Stop returned while a public tick still held effect observation")
	default:
	}
	close(o.release)
	if err := <-tick; err != nil {
		t.Fatal(err)
	}
	if err := <-stopped; err != nil {
		t.Fatal(err)
	}
	if err := p.PollOnce(ctx); err == nil {
		t.Fatal("PollOnce after shutdown accepted")
	}
	if err := p.PollOnce(func() context.Context { c, cancel := context.WithCancel(ctx); cancel(); return c }()); err == nil {
		t.Fatal("cancelled PollOnce accepted")
	}
}
func TestPollerDoneAndErrLifecycle_RealFailure(t *testing.T) {
	ctx := context.Background()
	s := newRecoveryStore(t)
	stop, _, generation := seedLiveStopIntent(t, s, "stop-fail-lifecycle")
	call := time.Now().UTC().Add(-2 * time.Second)
	deadline := time.Now().UTC().Add(time.Minute)
	if err := s.CommitStopCallAccepted(ctx, stop.OperationID, call, deadline); err != nil {
		t.Fatal(err)
	}

	// Return a mismatched observation ID which triggers a fatal non-ignorable protocol failure in classifyStop
	o := &testObserver{result: &ao.WorkerStatus{ID: "mismatched-session-id", TerminalGeneration: generation}}
	owner := &Runner{Store: s, ready: true}
	p := &Poller{Store: s, AO: o, Owner: owner, Interval: 5 * time.Millisecond, Actor: "test-supervisor"}

	if err := p.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	doneChan := p.Done()
	if doneChan == nil {
		t.Fatal("Done() returned nil channel")
	}

	select {
	case <-doneChan:
		err := p.Err()
		if err == nil {
			t.Fatal("expected non-nil Err() on real PollOnce failure, got nil")
		}
		if !strings.Contains(err.Error(), "stop observation identity invalid") {
			t.Fatalf("unexpected Err message: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for Done() channel on poller failure")
	}

	// Calling Stop() after failure should return the recorded error cleanly without blocking
	stopErr := p.Stop()
	if stopErr == nil {
		t.Fatal("expected Stop() to return recorded failure error, got nil")
	}
}

func TestPollerDoneAndErrLifecycle_CleanStop(t *testing.T) {
	ctx := context.Background()
	s := newRecoveryStore(t)
	o := &testObserver{}
	owner := &Runner{Store: s, ready: true}
	p := &Poller{Store: s, AO: o, Owner: owner, Interval: 50 * time.Millisecond, Actor: "test-supervisor"}

	if err := p.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	doneChan := p.Done()
	if err := p.Stop(); err != nil {
		t.Fatalf("clean Stop failed: %v", err)
	}

	select {
	case <-doneChan:
		if err := p.Err(); err != nil {
			t.Fatalf("expected nil Err() on clean Stop, got: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for Done() on clean Stop")
	}
}

func TestPollerDoneAndErrLifecycle_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	s := newRecoveryStore(t)
	o := &testObserver{}
	owner := &Runner{Store: s, ready: true}
	p := &Poller{Store: s, AO: o, Owner: owner, Interval: 50 * time.Millisecond, Actor: "test-supervisor"}

	if err := p.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	doneChan := p.Done()
	cancel()

	select {
	case <-doneChan:
		if err := p.Err(); err != nil {
			t.Fatalf("expected nil Err() on context cancel, got: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for Done() on context cancel")
	}
}

func TestPollerDoneAndErrLifecycle_Restart(t *testing.T) {
	ctx := context.Background()
	s := newRecoveryStore(t)
	o := &testObserver{}
	owner := &Runner{Store: s, ready: true}
	p := &Poller{Store: s, AO: o, Owner: owner, Interval: 50 * time.Millisecond, Actor: "test-supervisor"}

	// First cycle
	if err := p.Start(ctx); err != nil {
		t.Fatalf("Cycle 1 Start failed: %v", err)
	}
	done1 := p.Done()
	if err := p.Stop(); err != nil {
		t.Fatalf("Cycle 1 Stop failed: %v", err)
	}
	<-done1
	if err := p.Err(); err != nil {
		t.Fatalf("Cycle 1 Err() not nil: %v", err)
	}

	// Second cycle (restart)
	if err := p.Start(ctx); err != nil {
		t.Fatalf("Cycle 2 Start (restart) failed: %v", err)
	}
	done2 := p.Done()
	if done1 == done2 {
		t.Fatal("expected fresh Done channel on restart, got same channel")
	}
	if err := p.Stop(); err != nil {
		t.Fatalf("Cycle 2 Stop failed: %v", err)
	}
	<-done2
	if err := p.Err(); err != nil {
		t.Fatalf("Cycle 2 Err() not nil: %v", err)
	}
}

func TestPollerDoneAndErrLifecycle_Race(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s := newRecoveryStore(t)
	o := &testObserver{}
	owner := &Runner{Store: s, ready: true}
	p := &Poller{Store: s, AO: o, Owner: owner, Interval: 10 * time.Millisecond, Actor: "test-supervisor"}

	if err := p.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				_ = p.Done()
				_ = p.Err()
				time.Sleep(time.Millisecond)
			}
		}()
	}

	time.Sleep(30 * time.Millisecond)
	_ = p.Stop()
	wg.Wait()
}
func TestPollerDoneAndErrLifecycle_BarrierErrorAndRestart(t *testing.T) {
	ctx := context.Background()
	s := newRecoveryStore(t)
	o := &testObserver{}
	owner := &Runner{Store: s, ready: true}
	p := &Poller{Store: s, AO: o, Owner: owner, Interval: 5 * time.Millisecond, Actor: "test-supervisor"}

	// Barrier coordination for Cycle 1
	hook1Triggered := make(chan struct{}, 1)
	hook1Release := make(chan struct{})
	p.TestFailHook = func() error {
		select {
		case hook1Triggered <- struct{}{}:
		default:
		}
		<-hook1Release
		return errors.New("deterministic barrier error in cycle 1")
	}

	// Start Cycle 1
	if err := p.Start(ctx); err != nil {
		t.Fatalf("Cycle 1 Start failed: %v", err)
	}
	done1 := p.Done()

	// Wait deterministically for PollOnce to enter the hook
	select {
	case <-hook1Triggered:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for Cycle 1 hook trigger")
	}

	// Release the hook with failure
	close(hook1Release)

	// Wait deterministically for cycle 1 to close done1
	select {
	case <-done1:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for done1 to close")
	}

	// Stop cycle 1 (must join cleanly)
	if err := p.Stop(); err == nil {
		t.Fatal("expected Stop to return cycle 1 error, got nil")
	}

	// NOW: Before reading cycle 1 error, immediately start Cycle 2!
	// This proves that starting Cycle 2 does NOT mask or delete Cycle 1 error for its watcher.
	hook2Triggered := make(chan struct{}, 1)
	hook2Release := make(chan struct{})
	p.TestFailHook = func() error {
		select {
		case hook2Triggered <- struct{}{}:
		default:
		}
		<-hook2Release
		return nil
	}

	if err := p.Start(ctx); err != nil {
		t.Fatalf("Cycle 2 Start failed: %v", err)
	}
	done2 := p.Done()
	if done1 == done2 {
		t.Fatal("expected fresh done channel for Cycle 2")
	}

	// Wait for Cycle 2 to actively run
	select {
	case <-hook2Triggered:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for Cycle 2 hook trigger")
	}

	// While Cycle 2 is actively running, verify Cycle 1 watcher can still read Cycle 1 error!
	err1Bound := p.ErrFor(done1)
	if err1Bound == nil || !strings.Contains(err1Bound.Error(), "deterministic barrier error in cycle 1") {
		t.Fatalf("Cycle 1 error was masked/lost via ErrFor: %v", err1Bound)
	}

	err1Legacy := p.Err()
	if err1Legacy == nil || !strings.Contains(err1Legacy.Error(), "deterministic barrier error in cycle 1") {
		t.Fatalf("Cycle 1 error was masked/lost via Err: %v", err1Legacy)
	}

	// Now release Cycle 2 and stop cleanly
	close(hook2Release)
	if err := p.Stop(); err != nil {
		t.Fatalf("Cycle 2 clean stop failed: %v", err)
	}
	<-done2
	if err := p.ErrFor(done2); err != nil {
		t.Fatalf("expected nil error for clean Cycle 2, got: %v", err)
	}
}

func TestPollerDoneAndErrLifecycle_StopAlwaysJoinsGoroutine(t *testing.T) {
	ctx := context.Background()
	s := newRecoveryStore(t)
	o := &testObserver{}
	owner := &Runner{Store: s, ready: true}
	p := &Poller{Store: s, AO: o, Owner: owner, Interval: 5 * time.Millisecond, Actor: "test-supervisor"}

	hookEntered := make(chan struct{})
	hookProceed := make(chan struct{})
	p.TestFailHook = func() error {
		close(hookEntered)
		<-hookProceed
		return errors.New("fatal hook error during stop")
	}

	if err := p.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	done := p.Done()

	// Wait until goroutine is inside PollOnce
	<-hookEntered

	// Concurrently call Stop while PollOnce is blocked
	stopResult := make(chan error, 1)
	go func() {
		stopResult <- p.Stop()
	}()

	// Let PollOnce proceed with error
	close(hookProceed)

	// Stop must join the goroutine and return the error
	select {
	case err := <-stopResult:
		if err == nil || !strings.Contains(err.Error(), "fatal hook error during stop") {
			t.Fatalf("expected fatal hook error from Stop, got: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for Stop to join")
	}

	// Verify done is closed
	select {
	case <-done:
	default:
		t.Fatal("expected done channel to be closed after Stop joined")
	}
}
