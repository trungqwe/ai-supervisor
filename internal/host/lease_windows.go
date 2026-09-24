//go:build windows

package host

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"golang.org/x/sys/windows"
)

// ProcessOwnerLease represents the Windows machine-wide single-daemon exclusive lock.
type ProcessOwnerLease struct {
	mu              sync.Mutex
	CanonicalDBPath string
	LockFilePath    string
	MetadataPath    string
	PipeName        string
	InstanceID      string
	PID             int
	lockHandle      windows.Handle
	released        bool
}

// DeriveSidecarPaths computes the deterministic lock file path, metadata file path, and pipe name.
func DeriveSidecarPaths(canonicalDBPath string) (lockPath, metaPath, pipeName string) {
	lockPath = canonicalDBPath + ".owner.lock"
	metaPath = canonicalDBPath + ".owner.json"

	h := sha256.Sum256([]byte(canonicalDBPath))
	pipeName = `\\.\pipe\ai-supervisor-` + hex.EncodeToString(h[:8])
	return lockPath, metaPath, pipeName
}

// AcquireProcessOwnerLease acquires an exclusive, non-inheritable file handle with share mode 0.
// On any contention (e.g. ERROR_SHARING_VIOLATION), it fails closed immediately.
func AcquireProcessOwnerLease(canonicalDBPath, instanceID string) (*ProcessOwnerLease, error) {
	if canonicalDBPath == "" || instanceID == "" {
		return nil, errors.New("host: canonicalDBPath and instanceID must not be empty")
	}

	lockPath, metaPath, pipeName := DeriveSidecarPaths(canonicalDBPath)

	lockPathUTF16, err := windows.UTF16PtrFromString(lockPath)
	if err != nil {
		return nil, fmt.Errorf("host: invalid lock path: %w", err)
	}

	// CreateFileW with non-zero desired access, dwShareMode=0 (exclusive), OPEN_ALWAYS, bInheritHandle=false
	h, err := windows.CreateFile(
		lockPathUTF16,
		windows.GENERIC_READ|windows.GENERIC_WRITE,
		0,   // dwShareMode: 0 = strictly exclusive
		nil, // lpSecurityAttributes: nil = non-inheritable
		windows.OPEN_ALWAYS,
		windows.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		if errors.Is(err, windows.ERROR_SHARING_VIOLATION) || errors.Is(err, windows.ERROR_ACCESS_DENIED) {
			return nil, fmt.Errorf("%w: %v", ErrSharingViolation, err)
		}
		var errno windows.Errno
		if errors.As(err, &errno) && (errno == 32 || errno == 5) {
			return nil, fmt.Errorf("%w: %v", ErrSharingViolation, err)
		}
		return nil, fmt.Errorf("host: failed to acquire process owner lock %q: %w", lockPath, err)
	}

	lease := &ProcessOwnerLease{
		CanonicalDBPath: canonicalDBPath,
		LockFilePath:    lockPath,
		MetadataPath:    metaPath,
		PipeName:        pipeName,
		InstanceID:      instanceID,
		PID:             os.Getpid(),
		lockHandle:      h,
	}

	// Write metadata file atomically
	if err := lease.writeMetadata(); err != nil {
		_ = windows.CloseHandle(h)
		return nil, fmt.Errorf("host: failed to write owner metadata: %w", err)
	}

	return lease, nil
}

func (l *ProcessOwnerLease) writeMetadata() error {
	meta := OwnerMetadata{
		OwnerPID:        l.PID,
		OwnerInstanceID: l.InstanceID,
		PipeName:        l.PipeName,
		StartedAt:       time.Now().UTC(),
		CanonicalDBPath: l.CanonicalDBPath,
	}
	bytes, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(l.MetadataPath, bytes, 0600)
}

// CleanMetadata removes the .owner.json metadata file IF AND ONLY IF its instance ID matches this process.
func (l *ProcessOwnerLease) CleanMetadata() {
	l.mu.Lock()
	defer l.mu.Unlock()

	data, err := os.ReadFile(l.MetadataPath)
	if err != nil {
		return
	}
	var meta OwnerMetadata
	if err := json.Unmarshal(data, &meta); err == nil && meta.OwnerInstanceID == l.InstanceID {
		_ = os.Remove(l.MetadataPath)
	}
}

// CloseLockHandle closes the underlying exclusive Windows lock handle.
// In shutdown drain, this must be called LAST after Store.Close() and PinnedDB.Close().
func (l *ProcessOwnerLease) CloseLockHandle() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.released {
		return nil
	}
	l.released = true
	if l.lockHandle != windows.InvalidHandle && l.lockHandle != 0 {
		return windows.CloseHandle(l.lockHandle)
	}
	return nil
}
