//go:build windows

package host_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/trungqwe/ai-supervisor/internal/host"
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

func TestPrepareNewDB(t *testing.T) {
	tempDir := t.TempDir()
	newDB, err := host.PrepareNewDB(tempDir, "new_db.sqlite")
	if err != nil {
		t.Fatalf("PrepareNewDB failed: %v", err)
	}
	defer newDB.Close()

	if !newDB.IsNew {
		t.Fatal("expected IsNew = true")
	}

	fi, err := os.Stat(newDB.StoreDBPath)
	if err != nil {
		t.Fatalf("failed to stat new db: %v", err)
	}
	if fi.Size() != 0 {
		t.Fatalf("expected 0-byte file, got size %d", fi.Size())
	}

	if err := newDB.VerifyPostOpenIdentity(); err != nil {
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
	if takeoverResp.Status != "DRAINED" {
		t.Fatalf("unexpected takeover response: %+v", takeoverResp)
	}
	if !stopCalled {
		t.Fatal("expected stop callback to have been invoked")
	}
}

type orderRecorder struct {
	name   string
	order  *[]string
	target interface{ Close() error }
}

func (c orderRecorder) Close() error {
	*c.order = append(*c.order, c.name)
	if c.target != nil {
		return c.target.Close()
	}
	return nil
}

func TestShutdownDrainSequence(t *testing.T) {
	var callOrder []string

	tempDir := t.TempDir()
	dbFile := filepath.Join(tempDir, "drain_test.sqlite")
	_ = os.WriteFile(dbFile, []byte("d"), 0600)

	pinned, err := host.PrepareExistingDB(dbFile)
	if err != nil {
		t.Fatal(err)
	}
	defer pinned.Close()

	lease, err := host.AcquireProcessOwnerLease(pinned.CanonicalDBPath, "inst-drain")
	if err != nil {
		t.Fatal(err)
	}
	defer lease.CloseLockHandle()

	auth := host.NewAuthority("principal")

	comps := host.ShutdownComponents{
		PipeServer: orderRecorder{name: "pipe", order: &callOrder},
		HTTPServer: orderRecorder{name: "http", order: &callOrder},
		Authority:  auth,
		Store:      orderRecorder{name: "store", order: &callOrder},
		PinnedDB:   orderRecorder{name: "pinnedDB", order: &callOrder, target: pinned},
		OwnerLease: lease,
	}

	err = host.ExecuteShutdownDrain(1*time.Second, comps)
	if err != nil {
		t.Fatalf("ExecuteShutdownDrain failed: %v", err)
	}

	// Verify order: pipe, http, store, pinnedDB
	expected := []string{"pipe", "http", "store", "pinnedDB"}
	if len(callOrder) != len(expected) {
		t.Fatalf("expected callOrder %v, got %v", expected, callOrder)
	}
	for i, v := range expected {
		if callOrder[i] != v {
			t.Fatalf("callOrder[%d] = %s, expected %s", i, callOrder[i], v)
		}
	}

	// Metadata file should be deleted
	if _, err := os.Stat(lease.MetadataPath); !os.IsNotExist(err) {
		t.Fatal("expected metadata file to be cleaned up")
	}
}
