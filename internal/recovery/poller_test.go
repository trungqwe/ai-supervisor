package recovery

import (
	"context"
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
	if err := p.PollOnce(context.Background()); err != nil {
		t.Fatal(err)
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
	p := &Poller{Store: s, AO: o, Interval: time.Second, Actor: "supervisor"}
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
