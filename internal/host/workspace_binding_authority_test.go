//go:build windows

package host

import (
	"context"
	"os"
	"os/exec"
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
	pinnedCommit := strings.Repeat("a", 40)

	leaseA, err := auth.Acquire(ctx, domain.WorkspaceBindingCandidate{
		CanonicalWorktreePath: dirA,
		LinkedGitDirPath:      gitdirA,
		PinnedAOCommit:        pinnedCommit,
	})
	if err != nil {
		t.Fatalf("acquire dirA failed: %v", err)
	}
	defer leaseA.Close()

	leaseB, err := auth.Acquire(ctx, domain.WorkspaceBindingCandidate{
		CanonicalWorktreePath: dirB,
		LinkedGitDirPath:      gitdirB,
		PinnedAOCommit:        pinnedCommit,
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

// AC-P04A-07 Probe 1: Directory Junction resolves to target physical identity and detects substitution
func TestWindowsWorkspaceBindingAuthority_AC07_JunctionProbe(t *testing.T) {
	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, "target_repo")
	targetGit := filepath.Join(targetDir, ".git")
	junctionDir := filepath.Join(tmpDir, "junction_repo")
	otherDir := filepath.Join(tmpDir, "other_repo")
	otherGit := filepath.Join(otherDir, ".git")

	if err := os.MkdirAll(targetGit, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(otherGit, 0755); err != nil {
		t.Fatal(err)
	}

	// Create directory junction using mklink /j
	cmd := exec.Command("cmd", "/c", "mklink", "/j", junctionDir, targetDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to create directory junction: %v, out=%s", err, string(out))
	}
	defer func() {
		_ = exec.Command("cmd", "/c", "rmdir", junctionDir).Run()
	}()

	auth := NewWindowsWorkspaceBindingAuthority()
	ctx := context.Background()
	pinnedCommit := strings.Repeat("c", 40)

	// Acquire on junction path
	leaseJunction, err := auth.Acquire(ctx, domain.WorkspaceBindingCandidate{
		CanonicalWorktreePath: junctionDir,
		LinkedGitDirPath:      filepath.Join(junctionDir, ".git"),
		PinnedAOCommit:        pinnedCommit,
	})
	if err != nil {
		t.Fatalf("failed to acquire lease on junction: %v", err)
	}
	defer leaseJunction.Close()

	// Acquire directly on target
	leaseTarget, err := auth.Acquire(ctx, domain.WorkspaceBindingCandidate{
		CanonicalWorktreePath: targetDir,
		LinkedGitDirPath:      targetGit,
		PinnedAOCommit:        pinnedCommit,
	})
	if err != nil {
		t.Fatalf("failed to acquire lease on target: %v", err)
	}
	defer leaseTarget.Close()

	snapJunction := leaseJunction.Snapshot()
	snapTarget := leaseTarget.Snapshot()

	// Junction must resolve to identical physical identity (FileID & VolumeSerial) as target
	if snapJunction.WorktreeFileIDHex != snapTarget.WorktreeFileIDHex {
		t.Fatalf("junction file_id %q != target file_id %q", snapJunction.WorktreeFileIDHex, snapTarget.WorktreeFileIDHex)
	}
	if snapJunction.WorktreeVolumeSerialHex != snapTarget.WorktreeVolumeSerialHex {
		t.Fatalf("junction volume %q != target volume %q", snapJunction.WorktreeVolumeSerialHex, snapTarget.WorktreeVolumeSerialHex)
	}

	// Substitution detection: binding for target compared with otherDir snapshot must fail
	leaseOther, err := auth.Acquire(ctx, domain.WorkspaceBindingCandidate{
		CanonicalWorktreePath: otherDir,
		LinkedGitDirPath:      otherGit,
		PinnedAOCommit:        pinnedCommit,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer leaseOther.Close()
	snapOther := leaseOther.Snapshot()

	bindingTarget := domain.WorkspaceBinding{
		CanonicalWorktreePath:       snapTarget.CanonicalWorktreePath,
		VolumeSerialHex:             snapTarget.WorktreeVolumeSerialHex,
		FileIDHex:                   snapTarget.WorktreeFileIDHex,
		LinkedGitDirPath:            snapTarget.LinkedGitDirPath,
		LinkedGitDirVolumeSerialHex: snapTarget.LinkedGitDirVolumeSerialHex,
		LinkedGitDirFileIDHex:       snapTarget.LinkedGitDirFileIDHex,
		PinnedAOCommit:              snapTarget.PinnedAOCommit,
	}
	if snapOther.MatchesBinding(&bindingTarget) {
		t.Fatal("substituted other directory unexpectedly matched target binding")
	}
}

// AC-P04A-07 Probe 2: subst drive alias resolves to physical identity
func TestWindowsWorkspaceBindingAuthority_AC07_SubstProbe(t *testing.T) {
	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, "subst_target")
	targetGit := filepath.Join(targetDir, ".git")
	if err := os.MkdirAll(targetGit, 0755); err != nil {
		t.Fatal(err)
	}

	// Find an available drive letter (e.g. Y, X, W)
	driveLetter := "Y"
	for _, l := range []string{"Y", "X", "W", "V"} {
		if _, err := os.Stat(l + ":\\"); err != nil {
			driveLetter = l
			break
		}
	}

	// Map drive alias via subst
	substCmd := exec.Command("cmd", "/c", "subst", driveLetter+":", targetDir)
	if out, err := substCmd.CombinedOutput(); err != nil {
		t.Skipf("skipping subst probe (subst unavailable or denied): %v, out=%s", err, string(out))
	}
	defer func() {
		_ = exec.Command("cmd", "/c", "subst", driveLetter+":", "/d").Run()
	}()

	auth := NewWindowsWorkspaceBindingAuthority()
	ctx := context.Background()
	pinnedCommit := strings.Repeat("d", 40)

	substWorktree := driveLetter + ":\\"
	substGit := driveLetter + ":\\.git"

	leaseSubst, err := auth.Acquire(ctx, domain.WorkspaceBindingCandidate{
		CanonicalWorktreePath: substWorktree,
		LinkedGitDirPath:      substGit,
		PinnedAOCommit:        pinnedCommit,
	})
	if err != nil {
		t.Fatalf("failed to acquire lease on subst drive: %v", err)
	}
	defer leaseSubst.Close()

	leaseTarget, err := auth.Acquire(ctx, domain.WorkspaceBindingCandidate{
		CanonicalWorktreePath: targetDir,
		LinkedGitDirPath:      targetGit,
		PinnedAOCommit:        pinnedCommit,
	})
	if err != nil {
		t.Fatalf("failed to acquire lease on physical target: %v", err)
	}
	defer leaseTarget.Close()

	snapSubst := leaseSubst.Snapshot()
	snapTarget := leaseTarget.Snapshot()

	if snapSubst.WorktreeFileIDHex != snapTarget.WorktreeFileIDHex {
		t.Fatalf("subst file_id %q != physical target file_id %q", snapSubst.WorktreeFileIDHex, snapTarget.WorktreeFileIDHex)
	}
	if snapSubst.WorktreeVolumeSerialHex != snapTarget.WorktreeVolumeSerialHex {
		t.Fatalf("subst volume %q != physical target volume %q", snapSubst.WorktreeVolumeSerialHex, snapTarget.WorktreeVolumeSerialHex)
	}
}

// AC-P04A-07 Probe 3: Symlink probe
func TestWindowsWorkspaceBindingAuthority_AC07_SymlinkProbe(t *testing.T) {
	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, "symlink_target")
	targetGit := filepath.Join(targetDir, ".git")
	symlinkDir := filepath.Join(tmpDir, "symlink_link")

	if err := os.MkdirAll(targetGit, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.Symlink(targetDir, symlinkDir); err != nil {
		t.Logf("skipping symlink probe (unprivileged account without symlink evaluation): %v", err)
		return
	}

	auth := NewWindowsWorkspaceBindingAuthority()
	ctx := context.Background()
	pinnedCommit := strings.Repeat("e", 40)

	leaseSymlink, err := auth.Acquire(ctx, domain.WorkspaceBindingCandidate{
		CanonicalWorktreePath: symlinkDir,
		LinkedGitDirPath:      filepath.Join(symlinkDir, ".git"),
		PinnedAOCommit:        pinnedCommit,
	})
	if err != nil {
		t.Fatalf("failed to acquire lease on symlink: %v", err)
	}
	defer leaseSymlink.Close()

	leaseTarget, err := auth.Acquire(ctx, domain.WorkspaceBindingCandidate{
		CanonicalWorktreePath: targetDir,
		LinkedGitDirPath:      targetGit,
		PinnedAOCommit:        pinnedCommit,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer leaseTarget.Close()

	if leaseSymlink.Snapshot().WorktreeFileIDHex != leaseTarget.Snapshot().WorktreeFileIDHex {
		t.Fatalf("symlink file_id does not match physical target")
	}
}

// AC-P04A-07 Probe 4: Candidate PinnedAOCommit strict validation
func TestWindowsWorkspaceBindingAuthority_CandidateAOCommitValidation(t *testing.T) {
	auth := NewWindowsWorkspaceBindingAuthority()
	ctx := context.Background()

	// Empty commit fails closed
	_, err := auth.Acquire(ctx, domain.WorkspaceBindingCandidate{
		CanonicalWorktreePath: "C:\\Windows",
		LinkedGitDirPath:      "C:\\Windows",
		PinnedAOCommit:        "",
	})
	if err == nil {
		t.Fatal("expected error on empty pinned AO commit")
	}

	// Short/bad commit fails closed
	_, err = auth.Acquire(ctx, domain.WorkspaceBindingCandidate{
		CanonicalWorktreePath: "C:\\Windows",
		LinkedGitDirPath:      "C:\\Windows",
		PinnedAOCommit:        "bad_short_commit",
	})
	if err == nil {
		t.Fatal("expected error on invalid pinned AO commit")
	}
}

func TestMemoryWorkspaceBindingAuthority(t *testing.T) {
	pinnedCommit := strings.Repeat("b", 40)
	auth := NewMemoryWorkspaceBindingAuthorityWithSnapshot(domain.WorkspaceBindingSnapshot{
		CanonicalWorktreePath:       "/test/worktree",
		WorktreeVolumeSerialHex:     "0000000012345678",
		WorktreeFileIDHex:           "000000000000000012345678abcdef01",
		LinkedGitDirPath:            "/test/worktree/.git",
		LinkedGitDirVolumeSerialHex: "0000000012345678",
		LinkedGitDirFileIDHex:       "000000000000000012345678abcdef02",
		PinnedAOCommit:              pinnedCommit,
	})
	ctx := context.Background()

	candidate := domain.WorkspaceBindingCandidate{
		CanonicalWorktreePath: "/test/worktree",
		LinkedGitDirPath:      "/test/worktree/.git",
		PinnedAOCommit:        pinnedCommit,
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
