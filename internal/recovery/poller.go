package recovery

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/trungqwe/ai-supervisor/internal/store"
)

// Poller performs observation only. It never classifies persisted effect
// intents, and each Store mutation rechecks the exact durable tuple.
type Poller struct {
	Store    *store.Store
	AO       Observer
	Owner    *Runner
	Interval time.Duration
	Actor    string
	mu       sync.Mutex
	running  bool
	lastErr  error
	cancel   context.CancelFunc
	done     chan struct{}
	tickMu   sync.Mutex
	stopMu   sync.Mutex
	shutdown bool
	stopping bool
}

// Done returns a receive-only channel that is closed when the poller stops or encounters an unrecoverable failure.
// Each call to Start allocates a fresh channel. Access is thread-safe.
func (p *Poller) Done() <-chan struct{} {
	if p == nil {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.done
}

// Err returns the last recorded error that caused the poller loop to terminate.
// Clean cancellation or intentional Stop leaves Err returning nil. Access is thread-safe.
func (p *Poller) Err() error {
	if p == nil {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
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
		active := p.Owner.activePoller.running
		p.Owner.activePoller.mu.Unlock()
		if active {
			return errors.New("recovery: another poller owns this runner")
		}
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.running || p.stopping {
		return errors.New("recovery: poller already running")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	runCtx, cancel := context.WithCancel(ctx)
	p.cancel = cancel
	doneChan := make(chan struct{})
	p.done = doneChan
	p.running = true
	p.shutdown = false
	p.lastErr = nil
	p.Owner.activePoller = p
	go func(done chan struct{}) {
		defer close(done)
		t := time.NewTicker(p.Interval)
		defer t.Stop()
		for {
			select {
			case <-runCtx.Done():
				return
			case <-t.C:
				if err := p.PollOnce(runCtx); err != nil {
					if runCtx.Err() != nil {
						return
					}
					p.mu.Lock()
					p.lastErr = err
					p.running = false
					p.mu.Unlock()
					return
				}
			}
		}
	}(doneChan)
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
	if !p.running {
		err := p.lastErr
		p.mu.Unlock()
		p.tickMu.Lock()
		p.tickMu.Unlock()
		p.mu.Lock()
		p.stopping = false
		p.mu.Unlock()
		return err
	}
	cancel, done := p.cancel, p.done
	p.mu.Unlock()
	cancel()
	<-done
	p.mu.Lock()
	p.running = false
	err := p.lastErr
	p.mu.Unlock()
	p.tickMu.Lock()
	p.tickMu.Unlock()
	p.mu.Lock()
	p.stopping = false
	p.mu.Unlock()
	return err
}
