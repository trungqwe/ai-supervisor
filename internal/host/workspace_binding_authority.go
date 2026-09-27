package host

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/trungqwe/ai-supervisor/internal/domain"
)

// DefaultWorkspaceBindingAuthority returns the host's platform-specific WorkspaceBindingAuthority.
func DefaultWorkspaceBindingAuthority() domain.WorkspaceBindingAuthority {
	return NewWindowsWorkspaceBindingAuthority()
}

// MemoryWorkspaceBindingLease is a test/mock lease implementing domain.WorkspaceBindingLease.
type MemoryWorkspaceBindingLease struct {
	snapshot  domain.WorkspaceBindingSnapshot
	closed    bool
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
	Snapshot     *domain.WorkspaceBindingSnapshot
	FailAcquire  bool
}

func NewMemoryWorkspaceBindingAuthority() *MemoryWorkspaceBindingAuthority {
	return &MemoryWorkspaceBindingAuthority{}
}

func NewMemoryWorkspaceBindingAuthorityWithSnapshot(snapshot domain.WorkspaceBindingSnapshot) *MemoryWorkspaceBindingAuthority {
	return &MemoryWorkspaceBindingAuthority{Snapshot: &snapshot}
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
	if !domain.IsValidPinnedAOCommit(candidate.PinnedAOCommit) {
		return nil, fmt.Errorf("host: candidate PinnedAOCommit must be 40 lowercase hex chars, got %q", candidate.PinnedAOCommit)
	}

	var snapshot domain.WorkspaceBindingSnapshot
	if m.SnapshotFunc != nil {
		snapshot = m.SnapshotFunc(candidate)
	} else if m.Snapshot != nil {
		snapshot = *m.Snapshot
		if snapshot.CanonicalWorktreePath == "" {
			snapshot.CanonicalWorktreePath = candidate.CanonicalWorktreePath
		}
		if snapshot.LinkedGitDirPath == "" {
			snapshot.LinkedGitDirPath = candidate.LinkedGitDirPath
		}
		if snapshot.PinnedAOCommit == "" {
			snapshot.PinnedAOCommit = candidate.PinnedAOCommit
		}
	} else {
		return nil, errors.New("host: MemoryWorkspaceBindingAuthority requires explicit Snapshot or SnapshotFunc")
	}

	return NewMemoryWorkspaceBindingLease(snapshot), nil
}
