//go:build windows

package host_test

import (
	"fmt"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/trungqwe/ai-supervisor/internal/host"
	"golang.org/x/sys/windows"
)

func TestPolicyValidation(t *testing.T) {
	// Missing policy fails closed
	p := host.Policies{}
	if err := p.Validate(); !errors.Is(err, host.ErrMissingPolicy) {
		t.Fatalf("expected ErrMissingPolicy, got %v", err)
	}

	// Injected valid policies pass
	valid := host.Policies{
		SupervisorHTTPTimeout:          10 * time.Second,
		SupervisorHealthProbeTimeout:   5 * time.Second,
		SupervisorSpawnTimeout:         30 * time.Second,
		SupervisorSendTimeout:          15 * time.Second,
		SupervisorActivityPollInterval: 1 * time.Second,
		SupervisorExecutionDeadline:    60 * time.Second,
		SupervisorKillStopTimeout:      10 * time.Second,
		SupervisorWorkspaceReadTimeout: 10 * time.Second,
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func TestDeriveSidecarPaths(t *testing.T) {
	dbPath := `\\?\C:\data\db.sqlite`
	lockPath, metaPath, pipeName := host.DeriveSidecarPaths(dbPath)

	if lockPath != `\\?\C:\data\db.sqlite.owner.lock` {
		t.Fatalf("unexpected lockPath: %s", lockPath)
	}
	if metaPath != `\\?\C:\data\db.sqlite.owner.json` {
		t.Fatalf("unexpected metaPath: %s", metaPath)
	}
	if len(pipeName) == 0 || pipeName[:9] != `\\.\pipe\` {
		t.Fatalf("unexpected pipeName: %s", pipeName)
	}
}

func TestAuthorityExclusiveScope(t *testing.T) {
	auth := host.NewAuthority("test-operator-principal")

	if auth.AutomaticRestoreEnabled() {
		t.Fatal("AUTOMATIC_RESTORE must be false")
	}

	ctx := context.Background()
	scope1, err := auth.AcquireExclusiveScope(ctx, "pair-1", "UNIT_TEST")
	if err != nil {
		t.Fatalf("AcquireExclusiveScope failed: %v", err)
	}

	// Contention on same pair must fail
	_, err = auth.AcquireExclusiveScope(ctx, "pair-1", "CONTENDER")
	if err == nil {
		t.Fatal("expected contention error for same pair, got nil")
	}

	// Different pair succeeds
	scope2, err := auth.AcquireExclusiveScope(ctx, "pair-2", "UNIT_TEST")
	if err != nil {
		t.Fatalf("second pair failed: %v", err)
	}

	// Release scope1
	if err := scope1.Release(); err != nil {
		t.Fatalf("Release scope1 failed: %v", err)
	}

	// Re-acquire pair-1 now succeeds
	scope1Again, err := auth.AcquireExclusiveScope(ctx, "pair-1", "RETRY")
	if err != nil {
		t.Fatalf("re-acquire pair-1 failed: %v", err)
	}

	_ = scope2.Release()
	_ = scope1Again.Release()

	// Drain succeeds when all released
	if err := auth.Drain(1 * time.Second); err != nil {
		t.Fatalf("Drain failed: %v", err)
	}
}

func TestVerifiedRestorePrincipalFailClosed(t *testing.T) {
	auth := host.NewAuthority("any-token")
	ctx := context.Background()

	// Invariant (R1-003): Verified operator principal must fail-closed (return "", false, nil)
	p, ok, err := auth.VerifiedRestorePrincipal(ctx, "task-1", "contract-1", "pair-1", "caller")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok || p != "" {
		t.Fatalf("expected fail-closed (false, empty), got ok=%v, p=%q", ok, p)
	}
}

func TestProcessOwnerLeaseAndTwoProcessContention(t *testing.T) {
	tempDir := t.TempDir()
	dbFile := filepath.Join(tempDir, "test.sqlite")
	if err := os.WriteFile(dbFile, []byte("sqlite header"), 0600); err != nil {
		t.Fatal(err)
	}

	pinned, err := host.PrepareExistingDB(dbFile)
	if err != nil {
		t.Fatalf("PrepareExistingDB failed: %v", err)
	}
	defer pinned.Close()

	lease1, err := host.AcquireProcessOwnerLease(pinned.CanonicalDBPath, "instance-1")
	if err != nil {
		t.Fatalf("AcquireProcessOwnerLease failed: %v", err)
	}
	defer lease1.CloseLockHandle()

	// Contender trying to acquire same lock path must receive ErrSharingViolation
	_, err = host.AcquireProcessOwnerLease(pinned.CanonicalDBPath, "instance-2")
	if !errors.Is(err, host.ErrSharingViolation) {
		t.Fatalf("expected ErrSharingViolation, got %v", err)
	}

	// Close first lease
	if err := lease1.CloseLockHandle(); err != nil {
		t.Fatalf("CloseLockHandle failed: %v", err)
	}

	// Now contender succeeds
	lease2, err := host.AcquireProcessOwnerLease(pinned.CanonicalDBPath, "instance-2")
	if err != nil {
		t.Fatalf("second lease acquire failed: %v", err)
	}
	lease2.CleanMetadata()
	_ = lease2.CloseLockHandle()
}

func TestPinnedDBPreventsFileDeletion(t *testing.T) {
	tempDir := t.TempDir()
	dbFile := filepath.Join(tempDir, "pinned_test.sqlite")
	if err := os.WriteFile(dbFile, []byte("data"), 0600); err != nil {
		t.Fatal(err)
	}

	pinned, err := host.PrepareExistingDB(dbFile)
	if err != nil {
		t.Fatalf("PrepareExistingDB failed: %v", err)
	}
	defer pinned.Close()

	// While pinned handle is open, attempting to delete the file must fail with Windows sharing violation
	err = os.Remove(pinned.StoreDBPath)
	if err == nil {
		t.Fatal("expected file deletion to fail due to pinned handle, but it succeeded")
	}

	// Verify post-open identity check against itself
	if err := pinned.VerifyPostOpenIdentity(); err != nil {
		t.Fatalf("VerifyPostOpenIdentity failed: %v", err)
	}
}

func TestPrepareNewDBOrderAndVolume(t *testing.T) {
	tempDir := t.TempDir()
	prep, err := host.ResolveNewDBPaths(tempDir, "order_test.sqlite")
	if err != nil {
		t.Fatalf("ResolveNewDBPaths failed: %v", err)
	}

	// Invariant (R1-001): Before CreateAndPinDB, target file does not exist
	if _, err := os.Stat(prep.StoreDBPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected target file not to exist before CreateAndPinDB, got err=%v", err)
	}

	// Acquire owner lease FIRST
	lease, err := host.AcquireProcessOwnerLease(prep.CanonicalDBPath, "inst-order-1")
	if err != nil {
		t.Fatalf("AcquireProcessOwnerLease failed: %v", err)
	}
	defer lease.CloseLockHandle()

	// Now create and pin DB
	pinned, err := prep.CreateAndPinDB()
	if err != nil {
		t.Fatalf("CreateAndPinDB failed: %v", err)
	}
	defer pinned.Close()

	if !pinned.IsNew {
		t.Fatal("expected IsNew = true")
	}

	fi, err := os.Stat(pinned.StoreDBPath)
	if err != nil {
		t.Fatalf("failed to stat new db: %v", err)
	}
	if fi.Size() != 0 {
		t.Fatalf("expected 0-byte file, got size %d", fi.Size())
	}

	if err := pinned.VerifyPostOpenIdentity(); err != nil {
		t.Fatalf("VerifyPostOpenIdentity for new DB failed: %v", err)
	}
}

func TestNamedPipeTakeoverAndStatus(t *testing.T) {
	tempDir := t.TempDir()
	dbFile := filepath.Join(tempDir, "pipe_test.sqlite")
	_ = os.WriteFile(dbFile, []byte("d"), 0600)

	pinned, err := host.PrepareExistingDB(dbFile)
	if err != nil {
		t.Fatal(err)
	}
	defer pinned.Close()

	_, _, pipeName := host.DeriveSidecarPaths(pinned.CanonicalDBPath)

	stopCalled := false
	server, err := host.StartNamedPipeServer(pipeName, "inst-server", func() error {
		stopCalled = true
		return nil
	})
	if err != nil {
		t.Fatalf("StartNamedPipeServer failed: %v", err)
	}
	defer server.Close()

	// Query status from client
	statusResp, err := host.RequestPipeStatus(pipeName, 2*time.Second)
	if err != nil {
		t.Fatalf("RequestPipeStatus failed: %v", err)
	}
	if statusResp.Status != "RUNNING" || statusResp.InstanceID != "inst-server" {
		t.Fatalf("unexpected status response: %+v", statusResp)
	}

	// Request takeover
	takeoverResp, err := host.RequestPipeTakeover(pipeName, "inst-client", 2*time.Second)
	if err != nil {
		t.Fatalf("RequestPipeTakeover failed: %v", err)
	}
	if takeoverResp.Status != "STOP_ACKNOWLEDGED" {
		t.Fatalf("unexpected takeover response: %+v", takeoverResp)
	}
	if !stopCalled {
		t.Fatal("expected stop callback to have been invoked")
	}
}

func TestRequestPipeTakeoverNegativeCases(t *testing.T) {
	tempDir := t.TempDir()
	dbFile := filepath.Join(tempDir, "negative_pipe_test.sqlite")
	_ = os.WriteFile(dbFile, []byte("d"), 0600)

	pinned, err := host.PrepareExistingDB(dbFile)
	if err != nil {
		t.Fatal(err)
	}
	defer pinned.Close()

	_, _, pipeName := host.DeriveSidecarPaths(pinned.CanonicalDBPath)

	t.Run("server error returns non-nil error", func(t *testing.T) {
		server, err := host.StartNamedPipeServer(pipeName, "inst-server-err", func() error {
			return errors.New("simulated internal drain hook failure")
		})
		if err != nil {
			t.Fatalf("StartNamedPipeServer failed: %v", err)
		}
		defer server.Close()

		resp, err := host.RequestPipeTakeover(pipeName, "inst-client", 2*time.Second)
		if err == nil {
			t.Fatalf("expected error from RequestPipeTakeover on server ERROR, got nil (resp=%+v)", resp)
		}
		if !strings.Contains(err.Error(), "simulated internal drain hook failure") {
			t.Fatalf("expected error to contain hook failure message, got: %v", err)
		}
	})

	t.Run("unauthorized status returns ErrUnauthorizedCaller", func(t *testing.T) {
		unauthPipe := "\\\\.\\pipe\\test-unauth-pipe-" + filepath.Base(tempDir)
		pipeUTF16, err := windows.UTF16PtrFromString(unauthPipe)
		if err != nil {
			t.Fatal(err)
		}
		h, err := windows.CreateNamedPipe(
			pipeUTF16,
			windows.PIPE_ACCESS_DUPLEX,
			windows.PIPE_TYPE_MESSAGE|windows.PIPE_READMODE_MESSAGE|windows.PIPE_WAIT,
			1,
			4096,
			4096,
			0,
			nil,
		)
		if err != nil {
			t.Fatalf("CreateNamedPipe failed: %v", err)
		}
		defer windows.CloseHandle(h)

		go func() {
			if err := windows.ConnectNamedPipe(h, nil); err != nil && !errors.Is(err, windows.ERROR_PIPE_CONNECTED) {
				return
			}
			defer windows.DisconnectNamedPipe(h)

			buf := make([]byte, 4096)
			var bytesRead uint32
			_ = windows.ReadFile(h, buf, &bytesRead, nil)

			respData, _ := json.Marshal(host.PipeMessageResponse{
				Status: "UNAUTHORIZED",
				Error:  "Caller SID mismatch; access denied fail-closed",
			})
			var bytesWritten uint32
			_ = windows.WriteFile(h, respData, &bytesWritten, nil)
			_ = windows.FlushFileBuffers(h)
		}()

		resp, err := host.RequestPipeTakeover(unauthPipe, "inst-client", 2*time.Second)
		if err == nil {
			t.Fatalf("expected error on UNAUTHORIZED status, got resp=%+v", resp)
		}
		if !errors.Is(err, host.ErrUnauthorizedCaller) {
			t.Fatalf("expected ErrUnauthorizedCaller, got: %v", err)
		}
	})

	t.Run("unexpected status returns non-nil error", func(t *testing.T) {
		unexpectedPipe := "\\\\.\\pipe\\test-unexpected-pipe-" + filepath.Base(tempDir)
		pipeUTF16, err := windows.UTF16PtrFromString(unexpectedPipe)
		if err != nil {
			t.Fatal(err)
		}
		h, err := windows.CreateNamedPipe(
			pipeUTF16,
			windows.PIPE_ACCESS_DUPLEX,
			windows.PIPE_TYPE_MESSAGE|windows.PIPE_READMODE_MESSAGE|windows.PIPE_WAIT,
			1,
			4096,
			4096,
			0,
			nil,
		)
		if err != nil {
			t.Fatalf("CreateNamedPipe failed: %v", err)
		}
		defer windows.CloseHandle(h)

		go func() {
			if err := windows.ConnectNamedPipe(h, nil); err != nil && !errors.Is(err, windows.ERROR_PIPE_CONNECTED) {
				return
			}
			defer windows.DisconnectNamedPipe(h)

			buf := make([]byte, 4096)
			var bytesRead uint32
			_ = windows.ReadFile(h, buf, &bytesRead, nil)

			respData, _ := json.Marshal(host.PipeMessageResponse{
				Status:     "DRAINED",
				InstanceID: "rogue-daemon",
			})
			var bytesWritten uint32
			_ = windows.WriteFile(h, respData, &bytesWritten, nil)
			_ = windows.FlushFileBuffers(h)
		}()

		resp, err := host.RequestPipeTakeover(unexpectedPipe, "inst-client", 2*time.Second)
		if err == nil {
			t.Fatalf("expected error on unexpected status DRAINED, got resp=%+v", resp)
		}
		if !strings.Contains(err.Error(), "unexpected takeover response status") {
			t.Fatalf("expected unexpected status error message, got: %v", err)
		}
	})
}

type trackingCloser struct {
	closed bool
}

func (t *trackingCloser) Close() error {
	t.closed = true
	return nil
}

func TestDrainTimeoutPreservesLockAndStore(t *testing.T) {
	tempDir := t.TempDir()
	dbFile := filepath.Join(tempDir, "drain_timeout.sqlite")
	_ = os.WriteFile(dbFile, []byte("d"), 0600)

	pinned, err := host.PrepareExistingDB(dbFile)
	if err != nil {
		t.Fatal(err)
	}
	defer pinned.Close()

	lease, err := host.AcquireProcessOwnerLease(pinned.CanonicalDBPath, "inst-drain-to")
	if err != nil {
		t.Fatal(err)
	}
	defer lease.CloseLockHandle()

	auth := host.NewAuthority("principal")
	ctx := context.Background()

	// Acquire in-flight permit that will be held for 100ms
	permit, err := auth.AcquireExclusiveScope(ctx, "active-pair", "IN_FLIGHT_CALLER")
	if err != nil {
		t.Fatal(err)
	}

	storeCloser := &trackingCloser{}
	pinnedCloser := &trackingCloser{}
	schedCloser := &trackingCloser{}

	comps := host.ShutdownComponents{
		Authority:  auth,
		Store:      storeCloser,
		PinnedDB:   pinnedCloser,
		Schedulers: []io.Closer{schedCloser},
		OwnerLease: lease,
	}

	// Release permit after 100ms in a goroutine
	go func() {
		time.Sleep(100 * time.Millisecond)
		_ = permit.Release()
	}()

	// Probe contender lock contention during the timeout period (at 40ms, while drain timeout is 20ms)
	contenderChecked := make(chan struct{})
	go func() {
		time.Sleep(40 * time.Millisecond)
		// Contender trying to acquire .owner.lock must be rejected with ErrSharingViolation
		// because ExecuteShutdownDrain has NOT released the lock!
		_, contenderErr := host.AcquireProcessOwnerLease(pinned.CanonicalDBPath, "contender-inst")
		if !errors.Is(contenderErr, host.ErrSharingViolation) {
			t.Errorf("expected contender to fail with ErrSharingViolation while lock preserved at 40ms, got %v", contenderErr)
		}
		close(contenderChecked)
	}()

	// Execute drain with 20ms timeout.
	// Drain times out at 20ms, but ExecuteShutdownDrain blocks until 100ms when permit joins (R1-002).
	start := time.Now()
	err = host.ExecuteShutdownDrain(20*time.Millisecond, comps)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected ExecuteShutdownDrain to report drain completed with timeout, got nil")
	}
	if !strings.Contains(err.Error(), "drain completed after timeout") && !strings.Contains(err.Error(), "drain timeout") {
		t.Fatalf("expected drain timeout error, got %v", err)
	}

	<-contenderChecked

	// Must have waited at least ~80ms (proving it blocked until permit holder joined)
	if elapsed < 80*time.Millisecond {
		t.Fatalf("ExecuteShutdownDrain returned too early (%v), did not wait for permit holder to join (R1-002)", elapsed)
	}

	// Invariant (R1-002): After all permit holders joined, normal teardown completed:
	if !storeCloser.closed {
		t.Fatal("Store was not closed after permit holder joined")
	}
	if !pinnedCloser.closed {
		t.Fatal("PinnedDB was not closed after permit holder joined")
	}
	if !schedCloser.closed {
		t.Fatal("Schedulers were not closed after permit holder joined")
	}

	// Subsequent contender can now acquire .owner.lock because normal teardown closed the handle
	contenderLease, err := host.AcquireProcessOwnerLease(pinned.CanonicalDBPath, "contender-after-clean")
	if err != nil {
		t.Fatalf("expected contender to succeed after clean teardown, got %v", err)
	}
	_ = contenderLease.CloseLockHandle()
}

func TestAuthorityDrainTimeoutClosesAdmissionAndBlocks(t *testing.T) {
	auth := host.NewAuthority("principal")
	ctx := context.Background()

	// Acquire in-flight permit
	permit, err := auth.AcquireExclusiveScope(ctx, "pair-adm", "WORKER_1")
	if err != nil {
		t.Fatal(err)
	}

	// Drain times out with 20ms
	err = auth.Drain(20 * time.Millisecond)
	if !errors.Is(err, host.ErrDrainTimeout) {
		t.Fatalf("expected ErrDrainTimeout, got %v", err)
	}

	// Admission is closed: new permits are rejected
	_, err = auth.AcquireExclusiveScope(ctx, "pair-adm-2", "WORKER_2")
	if err == nil || !strings.Contains(err.Error(), "draining") {
		t.Fatalf("expected admission closed error, got %v", err)
	}

	// Background release after 60ms
	go func() {
		time.Sleep(60 * time.Millisecond)
		_ = permit.Release()
	}()

	start := time.Now()
	auth.WaitAllReleased()
	elapsed := time.Since(start)

	if elapsed < 50*time.Millisecond {
		t.Fatalf("WaitAllReleased returned too early (%v), expected >= 50ms", elapsed)
	}
}

type barrierStoreCloser struct {
	enterClose chan struct{}
	unblock    chan struct{}
	closed     bool
}

func (b *barrierStoreCloser) Close() error {
	close(b.enterClose)
	<-b.unblock
	b.closed = true
	return nil
}

func TestShutdownDrainStoreCloseBarrierAndOrder(t *testing.T) {
	tempDir := t.TempDir()
	dbFile := filepath.Join(tempDir, "barrier_order.sqlite")
	if err := os.WriteFile(dbFile, []byte("test-data"), 0600); err != nil {
		t.Fatal(err)
	}

	pinned, err := host.PrepareExistingDB(dbFile)
	if err != nil {
		t.Fatal(err)
	}
	defer pinned.Close()

	instID := "inst-barrier-order"
	lease, err := host.AcquireProcessOwnerLease(pinned.CanonicalDBPath, instID)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.CloseLockHandle()

	// Verify metadata file exists before drain
	metaBytes, err := os.ReadFile(lease.MetadataPath)
	if err != nil || !strings.Contains(string(metaBytes), instID) {
		t.Fatalf("expected metadata file to exist with instance ID, err=%v", err)
	}

	storeCloser := &barrierStoreCloser{
		enterClose: make(chan struct{}),
		unblock:    make(chan struct{}),
	}
	schedCloser := &trackingCloser{}

	comps := host.ShutdownComponents{
		Store:      storeCloser,
		PinnedDB:   pinned,
		Schedulers: []io.Closer{schedCloser},
		OwnerLease: lease,
	}

	drainDone := make(chan error, 1)
	go func() {
		drainDone <- host.ExecuteShutdownDrain(5*time.Second, comps)
	}()

	// Wait until Store.Close() is reached and entered
	select {
	case <-storeCloser.enterClose:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for Store.Close barrier entry")
	}

	// INVARIANT 1: Schedulers were stopped before Store.Close
	if !schedCloser.closed {
		t.Fatal("expected schedulers to be closed before Store.Close")
	}

	// INVARIANT 2: While Store.Close() is blocked, metadata file STILL exists (R4-001)
	if _, err := os.Stat(lease.MetadataPath); os.IsNotExist(err) {
		t.Fatal("metadata file was removed BEFORE Store.Close completed (violates ADR-017 §4.2 and AC-004-04)")
	}

	// INVARIANT 3: While Store.Close() is blocked, owner lock is STILL held by this process
	_, contenderErr := host.AcquireProcessOwnerLease(pinned.CanonicalDBPath, "contender-during-store-close")
	if !errors.Is(contenderErr, host.ErrSharingViolation) {
		t.Fatalf("expected contender lock acquisition to fail with ErrSharingViolation while Store.Close is blocked, got %v", contenderErr)
	}

	// Unblock Store.Close()
	close(storeCloser.unblock)

	// Wait for ExecuteShutdownDrain to complete
	select {
	case drainErr := <-drainDone:
		if drainErr != nil {
			t.Fatalf("ExecuteShutdownDrain failed: %v", drainErr)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for ExecuteShutdownDrain to complete after unblocking Store.Close")
	}

	// INVARIANT 4: Store was closed
	if !storeCloser.closed {
		t.Fatal("Store was not marked closed")
	}

	// INVARIANT 5: Metadata file removed AFTER Store.Close completed
	if _, err := os.Stat(lease.MetadataPath); !os.IsNotExist(err) {
		t.Fatalf("expected metadata file to be cleaned up after Store.Close, err=%v", err)
	}

	// INVARIANT 6: Lock was released LAST - contender can now acquire lock
	contenderLease, err := host.AcquireProcessOwnerLease(pinned.CanonicalDBPath, "contender-after-drain")
	if err != nil {
		t.Fatalf("expected contender to acquire owner lock after drain completed, got %v", err)
	}
	_ = contenderLease.CloseLockHandle()
}

func TestCleanMetadataOnlyRemovesMatchingInstanceID(t *testing.T) {
	tempDir := t.TempDir()
	dbFile := filepath.Join(tempDir, "mismatch_meta.sqlite")
	if err := os.WriteFile(dbFile, []byte("test-data"), 0600); err != nil {
		t.Fatal(err)
	}

	pinned, err := host.PrepareExistingDB(dbFile)
	if err != nil {
		t.Fatal(err)
	}
	defer pinned.Close()

	instID := "inst-owner-matching"
	lease, err := host.AcquireProcessOwnerLease(pinned.CanonicalDBPath, instID)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.CloseLockHandle()

	// Simulate metadata belonging to a foreign instance ID
	foreignMeta := `{"owner_instance_id":"foreign-instance-999","process_id":99999}`
	if err := os.WriteFile(lease.MetadataPath, []byte(foreignMeta), 0600); err != nil {
		t.Fatal(err)
	}

	// Calling CleanMetadata on lease with instID != "foreign-instance-999" must NOT remove the file
	lease.CleanMetadata()

	if _, err := os.Stat(lease.MetadataPath); os.IsNotExist(err) {
		t.Fatal("CleanMetadata removed metadata file whose instance ID did not match!")
	}

	// Now restore matching metadata and verify CleanMetadata removes it
	matchingMeta := fmt.Sprintf(`{"owner_instance_id":"%s","process_id":%d}`, instID, os.Getpid())
	if err := os.WriteFile(lease.MetadataPath, []byte(matchingMeta), 0600); err != nil {
		t.Fatal(err)
	}

	lease.CleanMetadata()

	if _, err := os.Stat(lease.MetadataPath); !os.IsNotExist(err) {
		t.Fatal("CleanMetadata failed to remove metadata file when instance ID matched!")
	}
}
