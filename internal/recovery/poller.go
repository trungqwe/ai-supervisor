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
}

func (p *Poller) PollOnce(ctx context.Context) error {
	if p == nil || p.Store == nil || p.AO == nil || p.Interval <= 0 || p.Actor == "" {
		return errors.New("recovery: poller dependencies and injected cadence required")
	}
	p.tickMu.Lock()
	defer p.tickMu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	snap, err := p.Store.ListRecoverySnapshot(ctx)
	if err != nil {
		return err
	}
	r := Runner{Store: p.Store, AO: p.AO, Actor: p.Actor}
	for _, x := range snap.Executions {
		if err = ctx.Err(); err != nil {
			return err
		}
		if x.DispatchStage != "SEND_CONFIRMED" {
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
	if p.running {
		return errors.New("recovery: poller already running")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	runCtx, cancel := context.WithCancel(ctx)
	p.cancel = cancel
	p.done = make(chan struct{})
	p.running = true
	p.lastErr = nil
	p.Owner.activePoller = p
	go func() {
		defer close(p.done)
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
					p.mu.Unlock()
					return
				}
			}
		}
	}()
	return nil
}

func (p *Poller) Stop() error {
	if p == nil {
		return nil
	}
	p.mu.Lock()
	if !p.running {
		err := p.lastErr
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
	return err
}
