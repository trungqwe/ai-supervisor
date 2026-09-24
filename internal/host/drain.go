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
}

// ExecuteShutdownDrain enforces the immutable 6-step teardown protocol:
// 1. Close listeners (Named Pipe, HTTP probe) so no new requests enter.
// 2. Wait/drain in-flight operations with drainTimeout.
// 3. Clean .owner.json metadata file IF AND ONLY IF instance ID matches.
// 4. Store.Close() to flush WAL and close SQLite connection.
// 5. Close pinned DB OS handle.
// 6. Close exclusive .owner.lock handle LAST.
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
		recordErr(c.Authority.Drain(timeout))
	}

	// Step 3: Remove .owner.json metadata file IF instance matches
	if c.OwnerLease != nil {
		c.OwnerLease.CleanMetadata()
	}

	// Step 4: Close Store
	if c.Store != nil {
		recordErr(c.Store.Close())
	}

	// Step 5: Close pinned DB handle
	if c.PinnedDB != nil {
		recordErr(c.PinnedDB.Close())
	}

	// Step 6: Close .owner.lock handle LAST
	if c.OwnerLease != nil {
		recordErr(c.OwnerLease.CloseLockHandle())
	}

	if firstErr != nil {
		return fmt.Errorf("host: error during shutdown drain: %w", firstErr)
	}
	return nil
}
