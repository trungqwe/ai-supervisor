package host

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/trungqwe/ai-supervisor/internal/recovery"
	"github.com/trungqwe/ai-supervisor/internal/stop"
)

type scopePermit struct {
	releaseFunc func() error
	released    bool
	mu          sync.Mutex
}

func (s *scopePermit) Release() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.released {
		return nil
	}
	s.released = true
	return s.releaseFunc()
}

// Authority serves as the single trusted host admission, exclusivity, and quiescence controller.
type Authority struct {
	mu               sync.Mutex
	pairLocks        map[string]string // pairID -> owner purpose
	globalLocked     bool
	globalOwner      string
	activeOps        sync.WaitGroup
	draining         bool
	principal        string
	automaticRestore bool // immutable false
}

// NewAuthority initializes a trusted Authority.
func NewAuthority(principal string) *Authority {
	return &Authority{
		pairLocks:        make(map[string]string),
		principal:        principal,
		automaticRestore: false, // Invariant: AUTOMATIC_RESTORE = DISABLED
	}
}

// Principal returns the verified operator principal string, if any.
func (a *Authority) Principal() string {
	return a.principal
}

// AutomaticRestoreEnabled returns whether automatic restore is permitted.
// Invariant: always false.
func (a *Authority) AutomaticRestoreEnabled() bool {
	return false
}

// Available reports whether the host is accepting new operations (not draining).
func (a *Authority) Available() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return !a.draining
}

// Acquire satisfies recovery.HostQuiescence
func (a *Authority) Acquire(ctx context.Context) (recovery.ExclusiveScope, error) {
	return a.AcquireGlobalExclusiveScope(ctx, "RECOVERY_SCAN")
}

// AcquireMaintenance satisfies recovery.LegacyMaintenanceHost
func (a *Authority) AcquireMaintenance(ctx context.Context, pairID, purpose string) (recovery.ExclusiveScope, error) {
	return a.AcquireMaintenanceScope(ctx, pairID, purpose)
}

// VerifiedRestorePrincipal satisfies recovery.LegacyOperatorBoundary and stop.OperatorBoundary.
// Invariant (R1-003): Unauthenticated CLI flag strings or caller parameters do not prove authentication.
// Verified operator principal remains an OPEN dependency at the trusted boundary until real cryptographic
// or authenticated token verification is integrated.
// Always returns ("", false, nil) fail-closed.
func (a *Authority) VerifiedRestorePrincipal(ctx context.Context, taskID, contractID, pairID, caller string) (string, bool, error) {
	return "", false, nil
}

// AcquireExclusiveScope locks a specific Pair for a dedicated purpose.
func (a *Authority) AcquireExclusiveScope(ctx context.Context, pairID, purpose string) (recovery.ExclusiveScope, error) {
	if pairID == "" {
		return nil, errors.New("host: pairID must not be empty")
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if a.draining {
		return nil, errors.New("host: authority is shutting down (draining)")
	}
	if a.globalLocked {
		return nil, fmt.Errorf("host: global exclusive scope is currently held by %q", a.globalOwner)
	}
	if existing, held := a.pairLocks[pairID]; held {
		return nil, fmt.Errorf("host: pair %q already locked by %q", pairID, existing)
	}

	a.pairLocks[pairID] = purpose
	a.activeOps.Add(1)

	return &scopePermit{
		releaseFunc: func() error {
			a.mu.Lock()
			defer a.mu.Unlock()
			delete(a.pairLocks, pairID)
			a.activeOps.Done()
			return nil
		},
	}, nil
}

// AcquireGlobalExclusiveScope locks all pairs for system-wide operations (e.g. startup scan).
func (a *Authority) AcquireGlobalExclusiveScope(ctx context.Context, purpose string) (recovery.ExclusiveScope, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.draining {
		return nil, errors.New("host: authority is shutting down (draining)")
	}
	if a.globalLocked {
		return nil, fmt.Errorf("host: global exclusive scope already held by %q", a.globalOwner)
	}
	if len(a.pairLocks) > 0 {
		return nil, errors.New("host: cannot acquire global scope while pair scopes are active")
	}

	a.globalLocked = true
	a.globalOwner = purpose
	a.activeOps.Add(1)

	return &scopePermit{
		releaseFunc: func() error {
			a.mu.Lock()
			defer a.mu.Unlock()
			a.globalLocked = false
			a.globalOwner = ""
			a.activeOps.Done()
			return nil
		},
	}, nil
}

// AcquireMaintenanceScope provides trusted maintenance scope for legacy budget / stop resolution.
func (a *Authority) AcquireMaintenanceScope(ctx context.Context, pairID, purpose string) (recovery.ExclusiveScope, error) {
	return a.AcquireExclusiveScope(ctx, pairID, "MAINTENANCE:"+purpose)
}

// AcquireEffect provides a one-use timeout effect permit for stop coordinator.
func (a *Authority) AcquireEffect(ctx context.Context, pairID, caller string) (stop.TimeoutPermit, error) {
	scope, err := a.AcquireExclusiveScope(ctx, pairID, "EFFECT:"+caller)
	if err != nil {
		return nil, err
	}
	return scope, nil
}

// Drain blocks until all in-flight scopes have been released or timeout occurs.
func (a *Authority) Drain(timeout time.Duration) error {
	a.mu.Lock()
	a.draining = true
	a.mu.Unlock()

	done := make(chan struct{})
	go func() {
		a.activeOps.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return ErrDrainTimeout
	}
}
