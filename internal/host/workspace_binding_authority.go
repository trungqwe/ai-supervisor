package host

import (
	"context"
	"errors"
	"strings"

	"github.com/trungqwe/ai-supervisor/internal/domain"
)

// DefaultWorkspaceBindingAuthority returns the host's platform-specific WorkspaceBindingAuthority.
func DefaultWorkspaceBindingAuthority() domain.WorkspaceBindingAuthority {
	return NewWindowsWorkspaceBindingAuthority()
}

// MemoryWorkspaceBindingLease is a test/mock lease implementing domain.WorkspaceBindingLease.
type MemoryWorkspaceBindingLease struct {
	snapshot domain.WorkspaceBindingSnapshot
	closed   bool
	failReval bool
}

func NewMemoryWorkspaceBindingLease(snapshot domain.WorkspaceBindingSnapshot) *MemoryWorkspaceBindingLease {
	return &MemoryWorkspaceBindingLease{
		snapshot: snapshot,
	}
}

func (m *MemoryWorkspaceBindingLease) Snapshot() domain.WorkspaceBindingSnapshot {
	return m.snapshot
}

func (m *MemoryWorkspaceBindingLease) Revalidate() error {
	if m.closed {
		return errors.New("host: lease is closed")
	}
	if m.failReval {
		return errors.New("host: simulated revalidation failure")
	}
	return nil
}

func (m *MemoryWorkspaceBindingLease) Close() error {
	m.closed = true
	return nil
}

func (m *MemoryWorkspaceBindingLease) SetFailRevalidation(fail bool) {
	m.failReval = fail
}

// MemoryWorkspaceBindingAuthority is an in-memory implementation for testing without real disk handles.
type MemoryWorkspaceBindingAuthority struct {
	SnapshotFunc func(candidate domain.WorkspaceBindingCandidate) domain.WorkspaceBindingSnapshot
	FailAcquire  bool
}

func NewMemoryWorkspaceBindingAuthority() *MemoryWorkspaceBindingAuthority {
	return &MemoryWorkspaceBindingAuthority{}
}

func (m *MemoryWorkspaceBindingAuthority) Acquire(ctx context.Context, candidate domain.WorkspaceBindingCandidate) (domain.WorkspaceBindingLease, error) {
	if m.FailAcquire {
		return nil, errors.New("host: simulated acquire failure")
	}
	if strings.TrimSpace(candidate.CanonicalWorktreePath) == "" {
		return nil, errors.New("host: candidate CanonicalWorktreePath must not be empty")
	}
	if strings.TrimSpace(candidate.LinkedGitDirPath) == "" {
		return nil, errors.New("host: candidate LinkedGitDirPath must not be empty")
	}

	var snapshot domain.WorkspaceBindingSnapshot
	if m.SnapshotFunc != nil {
		snapshot = m.SnapshotFunc(candidate)
	} else {
		pinned := candidate.PinnedAOCommit
		if pinned == "" {
			pinned = strings.Repeat("0", 40)
		}
		snapshot = domain.WorkspaceBindingSnapshot{
			CanonicalWorktreePath:       candidate.CanonicalWorktreePath,
			WorktreeVolumeSerialHex:     "0000000012345678",
			WorktreeFileIDHex:           "000000000000000012345678abcdef01",
			LinkedGitDirPath:            candidate.LinkedGitDirPath,
			LinkedGitDirVolumeSerialHex: "0000000012345678",
			LinkedGitDirFileIDHex:       "000000000000000012345678abcdef02",
			PinnedAOCommit:              pinned,
		}
	}

	return NewMemoryWorkspaceBindingLease(snapshot), nil
}
