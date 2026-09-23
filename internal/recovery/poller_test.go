package recovery

import (
	"context"
	"errors"
	"runtime"
	"sync"
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
	if err := s.RecordSendConfirmed(ctx, "dispatch-pollrace", "fixture", true, time.Now()); err != nil {
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
	if err := s.RecordSendConfirmed(ctx, "dispatch-shutdown", "fixture", true, time.Now()); err != nil {
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
	if err := s.RecordSendConfirmed(ctx, "dispatch-direct-barrier", "fixture", true, time.Now()); err != nil {
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
	if err := s.RecordSendConfirmed(ctx, "dispatch-public-stop", "fixture", true, time.Now()); err != nil {
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
