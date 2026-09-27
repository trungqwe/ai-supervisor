//go:build windows

package host

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/trungqwe/ai-supervisor/internal/domain"
)

var (
	hex16Regex = regexp.MustCompile(`^[0-9a-f]{16}$`)
	hex32Regex = regexp.MustCompile(`^[0-9a-f]{32}$`)
)

func TestWindowsWorkspaceBindingAuthority_LockAndSharingViolation(t *testing.T) {
	tmpDir := t.TempDir()
	worktreePath := filepath.Join(tmpDir, "worktree")
	gitdirPath := filepath.Join(tmpDir, "worktree", ".git")
	if err := os.MkdirAll(gitdirPath, 0755); err != nil {
		t.Fatalf("failed to create test directories: %v", err)
	}

	auth := NewWindowsWorkspaceBindingAuthority()
	ctx := context.Background()

	pinnedCommit := strings.Repeat("a", 40)
	candidate := domain.WorkspaceBindingCandidate{
		CanonicalWorktreePath: worktreePath,
		LinkedGitDirPath:      gitdirPath,
		PinnedAOCommit:        pinnedCommit,
	}

	lease, err := auth.Acquire(ctx, candidate)
	if err != nil {
		t.Fatalf("auth.Acquire failed: %v", err)
	}
	defer lease.Close()

	snapshot := lease.Snapshot()
	if !hex16Regex.MatchString(snapshot.WorktreeVolumeSerialHex) {
		t.Fatalf("invalid worktree volume serial: %q", snapshot.WorktreeVolumeSerialHex)
	}
	if !hex32Regex.MatchString(snapshot.WorktreeFileIDHex) {
		t.Fatalf("invalid worktree file id: %q", snapshot.WorktreeFileIDHex)
	}
	if !hex16Regex.MatchString(snapshot.LinkedGitDirVolumeSerialHex) {
		t.Fatalf("invalid gitdir volume serial: %q", snapshot.LinkedGitDirVolumeSerialHex)
	}
	if !hex32Regex.MatchString(snapshot.LinkedGitDirFileIDHex) {
		t.Fatalf("invalid gitdir file id: %q", snapshot.LinkedGitDirFileIDHex)
	}
	if snapshot.PinnedAOCommit != pinnedCommit {
		t.Fatalf("expected pinned commit %q, got %q", pinnedCommit, snapshot.PinnedAOCommit)
	}

	// AC-P04A-06: Attempt concurrent rename or deletion while lease is active
	// Windows kernel must reject rename with sharing violation / access denied.
	renamedPath := filepath.Join(tmpDir, "worktree_renamed")
	if err := os.Rename(worktreePath, renamedPath); err == nil {
		t.Fatalf("os.Rename succeeded while lease was held! Expected sharing violation")
	}

	// AC-P04A-08: Revalidate detects valid handle
	if err := lease.Revalidate(); err != nil {
		t.Fatalf("lease.Revalidate failed unexpectedly: %v", err)
	}

	// Close lease and verify idempotency
	if err := lease.Close(); err != nil {
		t.Fatalf("lease.Close failed: %v", err)
	}
	if err := lease.Close(); err != nil {
		t.Fatalf("second lease.Close failed (must be idempotent): %v", err)
	}

	// Revalidate after Close must fail
	if err := lease.Revalidate(); err == nil {
		t.Fatalf("lease.Revalidate succeeded on closed lease!")
	}

	// Now that lease is closed, rename must succeed
	if err := os.Rename(worktreePath, renamedPath); err != nil {
		t.Fatalf("os.Rename failed after lease was closed: %v", err)
	}
}

func TestWindowsWorkspaceBindingAuthority_PhysicalIdentityCaptureAndSubstitution(t *testing.T) {
	tmpDir := t.TempDir()
	dirA := filepath.Join(tmpDir, "dirA")
	dirB := filepath.Join(tmpDir, "dirB")
	gitdirA := filepath.Join(dirA, ".git")
	gitdirB := filepath.Join(dirB, ".git")

	_ = os.MkdirAll(gitdirA, 0755)
	_ = os.MkdirAll(gitdirB, 0755)

	auth := NewWindowsWorkspaceBindingAuthority()
	ctx := context.Background()

	leaseA, err := auth.Acquire(ctx, domain.WorkspaceBindingCandidate{
		CanonicalWorktreePath: dirA,
		LinkedGitDirPath:      gitdirA,
	})
	if err != nil {
		t.Fatalf("acquire dirA failed: %v", err)
	}
	defer leaseA.Close()

	leaseB, err := auth.Acquire(ctx, domain.WorkspaceBindingCandidate{
		CanonicalWorktreePath: dirB,
		LinkedGitDirPath:      gitdirB,
	})
	if err != nil {
		t.Fatalf("acquire dirB failed: %v", err)
	}
	defer leaseB.Close()

	snapA := leaseA.Snapshot()
	snapB := leaseB.Snapshot()

	// Different directories must have different 128-bit FileIDs
	if snapA.WorktreeFileIDHex == snapB.WorktreeFileIDHex {
		t.Fatalf("two distinct directories produced identical FileIDHex: %s", snapA.WorktreeFileIDHex)
	}
	if snapA.LinkedGitDirFileIDHex == snapB.LinkedGitDirFileIDHex {
		t.Fatalf("two distinct gitdirs produced identical FileIDHex: %s", snapA.LinkedGitDirFileIDHex)
	}

	// Test Equals
	if snapA.Equals(snapB) {
		t.Fatalf("snapA.Equals(snapB) returned true for distinct snapshots")
	}
}

func TestMemoryWorkspaceBindingAuthority(t *testing.T) {
	auth := NewMemoryWorkspaceBindingAuthority()
	ctx := context.Background()

	candidate := domain.WorkspaceBindingCandidate{
		CanonicalWorktreePath: "/test/worktree",
		LinkedGitDirPath:      "/test/worktree/.git",
		PinnedAOCommit:        strings.Repeat("b", 40),
	}

	lease, err := auth.Acquire(ctx, candidate)
	if err != nil {
		t.Fatalf("Acquire failed: %v", err)
	}
	defer lease.Close()

	snap := lease.Snapshot()
	if snap.CanonicalWorktreePath != candidate.CanonicalWorktreePath {
		t.Fatalf("expected worktree %q, got %q", candidate.CanonicalWorktreePath, snap.CanonicalWorktreePath)
	}
	if err := lease.Revalidate(); err != nil {
		t.Fatalf("Revalidate failed: %v", err)
	}

	memLease := lease.(*MemoryWorkspaceBindingLease)
	memLease.SetFailRevalidation(true)
	if err := lease.Revalidate(); err == nil {
		t.Fatalf("expected simulated revalidation failure")
	}

	if err := lease.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}
