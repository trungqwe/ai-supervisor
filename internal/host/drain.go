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
// 1. Close listeners (Named Pipe, HTTP probe) so no new requests enter.
// 2. Wait/drain in-flight operations with drainTimeout.
//    CRITICAL (R1-002): If drain times out or fails, fail-closed immediately!
//    DO NOT close Store, DO NOT close PinnedDB, and DO NOT release .owner.lock.
// 3. Stop background schedulers (poller, TimeoutMonitor) and join (R1-004).
// 4. Clean .owner.json metadata file IF AND ONLY IF instance ID matches.
// 5. Store.Close() to flush WAL and close SQLite connection.
// 6. Close pinned DB OS handle.
// 7. Close exclusive .owner.lock handle LAST.
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
			// CRITICAL (R1-002): If drain times out or fails because active permits are still held,
			// fail-closed immediately. DO NOT close Store, DO NOT close PinnedDB, and DO NOT release .owner.lock.
			return fmt.Errorf("host: shutdown drain timeout (active operations still held): %w", err)
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
