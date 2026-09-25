package host

import (
	"fmt"
	"io"
	"time"
)

// ShutdownComponents groups the active resources required for safe, ordered teardown.
type ShutdownComponents struct {
	PipeServer io.Closer
	HTTPServer io.Closer
	Authority  *Authority
	Store      io.Closer
	PinnedDB   io.Closer
	OwnerLease *ProcessOwnerLease
	Schedulers []io.Closer // poller and timeout monitor schedulers stopped before Store.Close
}

// ExecuteShutdownDrain enforces the immutable teardown protocol:
//  1. Close listeners (Named Pipe, HTTP probe) so no new requests enter.
//  2. Wait/drain in-flight operations with drainTimeout.
//     CRITICAL (R1-002): If drain times out because an effect caller still holds a permit,
//     keep admission closed and ownership held. Block until ALL permits are released
//     (join the caller), then proceed with normal teardown. NEVER return to main/os.Exit
//     while a permit holder is still active, because process exit releases the OS lock.
//  3. Stop background schedulers (poller, TimeoutMonitor) and join (R1-004).
//  4. Clean .owner.json metadata file IF AND ONLY IF instance ID matches.
//  5. Store.Close() to flush WAL and close SQLite connection.
//  6. Close pinned DB OS handle.
//  7. Close exclusive .owner.lock handle LAST.
func ExecuteShutdownDrain(timeout time.Duration, c ShutdownComponents) error {
	var firstErr error
	recordErr := func(err error) {
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}

	// Step 1: Close listeners
	if c.PipeServer != nil {
		recordErr(c.PipeServer.Close())
	}
	if c.HTTPServer != nil {
		recordErr(c.HTTPServer.Close())
	}

	// Step 2: Drain active callers/operations
	if c.Authority != nil {
		if err := c.Authority.Drain(timeout); err != nil {
			// CRITICAL (R1-002): Drain timed out - an effect caller still holds a permit.
			// Admission is already closed (Authority.draining = true from Drain()).
			// We MUST NOT return to main/os.Exit because process exit would release the
			// OS lock handle, allowing a contender to acquire it while the permit holder
			// is still active. Instead, wait indefinitely for ALL permits to join.
			//
			// This is the ADR-permitted fail-closed mechanism: keep admission closed,
			// keep lock ownership, and block until the active caller finishes.
			fmt.Printf("host: drain timeout exceeded, blocking until all active permits are released (R1-002)...\n")
			c.Authority.WaitAllReleased()
			// After all permits joined, record the original timeout error but proceed
			// with normal teardown (schedulers, Store, PinnedDB, lock).
			recordErr(fmt.Errorf("host: drain completed after timeout (active callers joined late): %w", err))
		}
	}

	// Step 3: Stop background schedulers (poller, TimeoutMonitor) before Store.Close (R1-004)
	for _, sched := range c.Schedulers {
		if sched != nil {
			recordErr(sched.Close())
		}
	}

	// Step 4: Remove .owner.json metadata file IF instance matches
	if c.OwnerLease != nil {
		c.OwnerLease.CleanMetadata()
	}

	// Step 5: Close Store
	if c.Store != nil {
		recordErr(c.Store.Close())
	}

	// Step 6: Close pinned DB handle
	if c.PinnedDB != nil {
		recordErr(c.PinnedDB.Close())
	}

	// Step 7: Close .owner.lock handle LAST
	if c.OwnerLease != nil {
		recordErr(c.OwnerLease.CloseLockHandle())
	}

	if firstErr != nil {
		return fmt.Errorf("host: error during shutdown drain: %w", firstErr)
	}
	return nil
}
