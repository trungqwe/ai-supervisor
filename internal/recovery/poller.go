package recovery

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/trungqwe/ai-supervisor/internal/store"
)

type pollerCycle struct {
	id      uint64
	done    chan struct{}
	cancel  context.CancelFunc
	err     error
	errRead bool
	stopped bool
	running bool
}

// Poller performs observation only. It never classifies persisted effect
// intents, and each Store mutation rechecks the exact durable tuple.
type Poller struct {
	Store        *store.Store
	AO           Observer
	Owner        *Runner
	Interval     time.Duration
	Actor        string
	TestFailHook func() error // Test seam: hook to trigger real PollOnce error during testing

	mu          sync.Mutex
	activeCycle *pollerCycle
	prevCycle   *pollerCycle
	cycleByDone map[<-chan struct{}]*pollerCycle
	cycleSeq    uint64

	running  bool
	lastErr  error
	cancel   context.CancelFunc
	done     chan struct{}
	tickMu   sync.Mutex
	stopMu   sync.Mutex
	shutdown bool
	stopping bool
}

// Done returns a receive-only channel that is closed when the current cycle stops or encounters an unrecoverable failure.
// Each call to Start allocates a fresh channel. Access is thread-safe.
func (p *Poller) Done() <-chan struct{} {
	if p == nil {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.activeCycle != nil {
		return p.activeCycle.done
	}
	if p.prevCycle != nil {
		return p.prevCycle.done
	}
	return p.done
}

// Err returns the error that caused the poller cycle to terminate.
// Clean cancellation or intentional Stop leaves Err returning nil.
// If a new Start cycle has been created, Err preserves the error of the completed cycle
// until read by the watcher, preventing new cycles from masking previous failures. Access is thread-safe.
func (p *Poller) Err() error {
	if p == nil {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.activeCycle != nil && p.activeCycle.err != nil {
		p.activeCycle.errRead = true
		return p.activeCycle.err
	}
	if p.prevCycle != nil && p.prevCycle.err != nil && !p.prevCycle.errRead {
		p.prevCycle.errRead = true
		return p.prevCycle.err
	}
	if p.prevCycle != nil && p.prevCycle.err != nil {
		return p.prevCycle.err
	}
	return p.lastErr
}

// ErrFor returns the error bound to a specific cycle's Done channel.
// This guarantees that a caller holding a captured Done channel always receives the exact error
// of that execution cycle, regardless of subsequent Start or Stop invocations.
func (p *Poller) ErrFor(done <-chan struct{}) error {
	if p == nil || done == nil {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if c, ok := p.cycleByDone[done]; ok {
		c.errRead = true
		return c.err
	}
	if p.prevCycle != nil && p.prevCycle.done == done {
		p.prevCycle.errRead = true
		return p.prevCycle.err
	}
	return p.lastErr
}

func (p *Poller) PollOnce(ctx context.Context) error {
	if p == nil || p.Store == nil || p.AO == nil || p.Owner == nil || p.Interval <= 0 || p.Actor == "" {
		return errors.New("recovery: poller dependencies and injected cadence required")
	}
	if p.Owner.Store != p.Store {
		return errors.New("recovery: poller Store differs from completed runner")
	}
	p.Owner.pollAccess.RLock()
	defer p.Owner.pollAccess.RUnlock()
	p.tickMu.Lock()
	defer p.tickMu.Unlock()
	p.mu.Lock()
	shutdown := p.shutdown
	p.mu.Unlock()
	p.Owner.gateMu.Lock()
	ready := p.Owner.ready && !p.Owner.running
	p.Owner.gateMu.Unlock()
	if shutdown || !ready {
		return errors.New("recovery: poll tick has no active ownership boundary")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	// Test seam: trigger deterministic real error if hook is configured
	if p.TestFailHook != nil {
		if err := p.TestFailHook(); err != nil {
			return err
		}
	}

	snap, err := p.Store.ListRecoverySnapshot(ctx)
	if err != nil {
		return err
	}
	r := Runner{Store: p.Store, AO: p.AO, Handoff: p.Owner.Handoff, Actor: p.Actor}
	stopOwnedAttempts := make(map[string]bool, len(snap.StopOwnedAttemptIDs))
	for _, id := range snap.StopOwnedAttemptIDs {
		stopOwnedAttempts[id] = true
	}
	for _, x := range snap.StopOperations {
		if x.Stage != "STOP_CALL_SUCCEEDED" {
			continue // STOP_REQUESTED has no proved abandoned effect owner.
		}
		if err = ctx.Err(); err != nil {
			return err
		}
		var report Report
		if err = r.classifyStop(ctx, x, "poller", &report); err != nil {
			if errors.Is(err, store.ErrStateConflict) {
				err = r.stopWinner(ctx, x, err)
			}
			if err != nil {
				return err
			}
		}
	}
	for _, x := range snap.Executions {
		if stopOwnedAttempts[x.AttemptID] {
			continue
		}
		if err = ctx.Err(); err != nil {
			return err
		}
		if x.DispatchStage != "SEND_CONFIRMED" && x.DispatchStage != "DISPATCH_BOUND" {
			continue
		}
		var report Report
		if err = r.classifyExecution(ctx, x, "poller", &report); err != nil {
			if errors.Is(err, store.ErrStateConflict) {
				err = r.executionWinner(ctx, x, err, &report)
			}
			if err == nil {
				continue
			}
			return err
		}
	}
	return nil
}

func (p *Poller) Start(ctx context.Context) error {
	if p == nil || p.Store == nil || p.AO == nil || p.Owner == nil || p.Interval <= 0 || p.Actor == "" {
		return errors.New("recovery: poller dependencies and injected cadence required")
	}
	p.Owner.gateMu.Lock()
	defer p.Owner.gateMu.Unlock()
	if p.Owner.running || !p.Owner.ready {
		return errors.New("recovery: startup classification has not completed")
	}
	if p.Owner.Store != p.Store {
		return errors.New("recovery: poller Store differs from completed runner")
	}
	if p.Owner.activePoller != nil && p.Owner.activePoller != p {
		p.Owner.activePoller.mu.Lock()
		active := (p.Owner.activePoller.activeCycle != nil && p.Owner.activePoller.activeCycle.running) || p.Owner.activePoller.running
		p.Owner.activePoller.mu.Unlock()
		if active {
			return errors.New("recovery: another poller owns this runner")
		}
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if (p.activeCycle != nil && p.activeCycle.running) || p.running || p.stopping {
		return errors.New("recovery: poller already running")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	p.cycleSeq++
	runCtx, cancel := context.WithCancel(ctx)
	cycle := &pollerCycle{
		id:      p.cycleSeq,
		done:    make(chan struct{}),
		cancel:  cancel,
		running: true,
	}
	if p.cycleByDone == nil {
		p.cycleByDone = make(map[<-chan struct{}]*pollerCycle)
	}
	p.cycleByDone[cycle.done] = cycle

	p.activeCycle = cycle
	p.cancel = cancel
	p.done = cycle.done
	p.running = true
	p.shutdown = false
	p.lastErr = nil
	p.Owner.activePoller = p

	go func(c *pollerCycle, cCtx context.Context) {
		defer close(c.done)
		t := time.NewTicker(p.Interval)
		defer t.Stop()
		for {
			select {
			case <-cCtx.Done():
				return
			case <-t.C:
				if err := p.PollOnce(cCtx); err != nil {
					if cCtx.Err() != nil && (errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)) {
						return
					}
					p.mu.Lock()
					c.err = err
					c.running = false
					p.lastErr = err
					p.running = false
					p.mu.Unlock()
					return
				}
			}
		}
	}(cycle, runCtx)
	return nil
}

func (p *Poller) Stop() error {
	if p == nil {
		return nil
	}
	p.stopMu.Lock()
	defer p.stopMu.Unlock()

	p.mu.Lock()
	p.shutdown = true
	p.stopping = true

	cycle := p.activeCycle
	if cycle == nil {
		var err error
		if p.prevCycle != nil {
			err = p.prevCycle.err
		} else {
			err = p.lastErr
		}
		p.mu.Unlock()

		p.tickMu.Lock()
		p.tickMu.Unlock()

		p.mu.Lock()
		p.stopping = false
		p.mu.Unlock()
		return err
	}

	cancel := cycle.cancel
	done := cycle.done
	p.mu.Unlock()

	if cancel != nil {
		cancel()
	}

	// ALWAYS join the corresponding Start cycle, even if PollOnce has already recorded an error and running=false
	if done != nil {
		<-done
	}

	p.mu.Lock()
	cycle.running = false
	cycle.stopped = true
	p.running = false
	err := cycle.err
	p.lastErr = err
	p.prevCycle = cycle
	p.activeCycle = nil
	p.mu.Unlock()

	p.tickMu.Lock()
	p.tickMu.Unlock()

	p.mu.Lock()
	p.stopping = false
	p.mu.Unlock()
	return err
}
