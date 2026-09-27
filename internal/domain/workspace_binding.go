package domain

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// BindingState represents the lifecycle state of a workspace binding.
type BindingState string

const (
	BindingStateActive                  BindingState = "ACTIVE"
	BindingStateRetainedForVerification BindingState = "RETAINED_FOR_VERIFICATION"
	BindingStateReleased                BindingState = "RELEASED"
	BindingStateInvalidated             BindingState = "INVALIDATED"
)

var (
	hex16Regex = regexp.MustCompile(`^[0-9a-f]{16}$`)
	hex32Regex = regexp.MustCompile(`^[0-9a-f]{32}$`)
	hex40Regex = regexp.MustCompile(`^[0-9a-f]{40}$`)
)

// WorkspaceBinding records durable physical binding between a task attempt and host worktree/gitdir.
type WorkspaceBinding struct {
	AttemptID                   string       `json:"attempt_id"`
	TaskID                      string       `json:"task_id"`
	ContractID                  string       `json:"contract_id"`
	SessionID                   string       `json:"session_id"`
	TerminalGeneration          string       `json:"terminal_generation"`
	CanonicalWorktreePath       string       `json:"canonical_worktree_path"`
	VolumeSerialHex             string       `json:"volume_serial_hex"`
	FileIDHex                   string       `json:"file_id_hex"`
	LinkedGitDirPath            string       `json:"linked_gitdir_path"`
	LinkedGitDirVolumeSerialHex string       `json:"linked_gitdir_volume_serial_hex"`
	LinkedGitDirFileIDHex       string       `json:"linked_gitdir_file_id_hex"`
	PinnedAOCommit              string       `json:"pinned_ao_commit"`
	BindingState                BindingState `json:"binding_state"`
	CreatedAtEpochMS            int64        `json:"created_at_epoch_ms"`
	ReleasedAtEpochMS           *int64       `json:"released_at_epoch_ms,omitempty"`
}

// Validate validates workspace binding invariants.
func (b *WorkspaceBinding) Validate() error {
	if strings.TrimSpace(b.AttemptID) == "" {
		return errors.New("attempt_id must not be empty")
	}
	if strings.TrimSpace(b.TaskID) == "" {
		return errors.New("task_id must not be empty")
	}
	if strings.TrimSpace(b.ContractID) == "" {
		return errors.New("contract_id must not be empty")
	}
	if strings.TrimSpace(b.SessionID) == "" {
		return errors.New("session_id must not be empty")
	}
	if strings.TrimSpace(b.TerminalGeneration) == "" {
		return errors.New("terminal_generation must not be empty")
	}
	if strings.TrimSpace(b.CanonicalWorktreePath) == "" {
		return errors.New("canonical_worktree_path must not be empty")
	}
	if !hex16Regex.MatchString(b.VolumeSerialHex) {
		return fmt.Errorf("volume_serial_hex must be 16 lowercase hex chars, got %q", b.VolumeSerialHex)
	}
	if !hex32Regex.MatchString(b.FileIDHex) {
		return fmt.Errorf("file_id_hex must be 32 lowercase hex chars, got %q", b.FileIDHex)
	}
	if strings.TrimSpace(b.LinkedGitDirPath) == "" {
		return errors.New("linked_gitdir_path must not be empty")
	}
	if !hex16Regex.MatchString(b.LinkedGitDirVolumeSerialHex) {
		return fmt.Errorf("linked_gitdir_volume_serial_hex must be 16 lowercase hex chars, got %q", b.LinkedGitDirVolumeSerialHex)
	}
	if !hex32Regex.MatchString(b.LinkedGitDirFileIDHex) {
		return fmt.Errorf("linked_gitdir_file_id_hex must be 32 lowercase hex chars, got %q", b.LinkedGitDirFileIDHex)
	}
	if !hex40Regex.MatchString(b.PinnedAOCommit) {
		return fmt.Errorf("pinned_ao_commit must be 40 lowercase hex chars, got %q", b.PinnedAOCommit)
	}
	switch b.BindingState {
	case BindingStateActive, BindingStateRetainedForVerification:
		if b.ReleasedAtEpochMS != nil {
			return fmt.Errorf("released_at_epoch_ms must be nil for %s state", b.BindingState)
		}
	case BindingStateReleased, BindingStateInvalidated:
		if b.ReleasedAtEpochMS == nil {
			return fmt.Errorf("released_at_epoch_ms must be non-nil for %s state", b.BindingState)
		}
		if *b.ReleasedAtEpochMS < b.CreatedAtEpochMS {
			return errors.New("released_at_epoch_ms must be >= created_at_epoch_ms")
		}
	default:
		return fmt.Errorf("invalid binding_state %q", b.BindingState)
	}
	if b.CreatedAtEpochMS <= 0 {
		return errors.New("created_at_epoch_ms must be positive")
	}
	return nil
}

// WorkspaceBindingCandidate specifies candidate target paths for acquiring a binding lease.
type WorkspaceBindingCandidate struct {
	CanonicalWorktreePath string `json:"canonical_worktree_path"`
	LinkedGitDirPath      string `json:"linked_gitdir_path"`
	PinnedAOCommit        string `json:"pinned_ao_commit,omitempty"`
}

// WorkspaceBindingSnapshot represents an in-memory verified physical identity snapshot.
type WorkspaceBindingSnapshot struct {
	CanonicalWorktreePath       string `json:"canonical_worktree_path"`
	WorktreeVolumeSerialHex     string `json:"worktree_volume_serial_hex"`
	WorktreeFileIDHex           string `json:"worktree_file_id_hex"`
	LinkedGitDirPath            string `json:"linked_gitdir_path"`
	LinkedGitDirVolumeSerialHex string `json:"linked_gitdir_volume_serial_hex"`
	LinkedGitDirFileIDHex       string `json:"linked_gitdir_file_id_hex"`
	PinnedAOCommit              string `json:"pinned_ao_commit"`
}

// Equals checks whether two snapshots are physically identical.
func (s WorkspaceBindingSnapshot) Equals(other WorkspaceBindingSnapshot) bool {
	return s.CanonicalWorktreePath == other.CanonicalWorktreePath &&
		s.WorktreeVolumeSerialHex == other.WorktreeVolumeSerialHex &&
		s.WorktreeFileIDHex == other.WorktreeFileIDHex &&
		s.LinkedGitDirPath == other.LinkedGitDirPath &&
		s.LinkedGitDirVolumeSerialHex == other.LinkedGitDirVolumeSerialHex &&
		s.LinkedGitDirFileIDHex == other.LinkedGitDirFileIDHex &&
		s.PinnedAOCommit == other.PinnedAOCommit
}

// MatchesBinding checks whether the snapshot matches the stored binding row fields.
func (s WorkspaceBindingSnapshot) MatchesBinding(b *WorkspaceBinding) bool {
	if b == nil {
		return false
	}
	return s.CanonicalWorktreePath == b.CanonicalWorktreePath &&
		s.WorktreeVolumeSerialHex == b.VolumeSerialHex &&
		s.WorktreeFileIDHex == b.FileIDHex &&
		s.LinkedGitDirPath == b.LinkedGitDirPath &&
		s.LinkedGitDirVolumeSerialHex == b.LinkedGitDirVolumeSerialHex &&
		s.LinkedGitDirFileIDHex == b.LinkedGitDirFileIDHex &&
		s.PinnedAOCommit == b.PinnedAOCommit
}

// WorkspaceBindingAuthority acquires exclusive physical directory capability leases.
type WorkspaceBindingAuthority interface {
	Acquire(ctx context.Context, candidate WorkspaceBindingCandidate) (WorkspaceBindingLease, error)
}

// WorkspaceBindingLease maintains live, anti-rename open directory handles.
type WorkspaceBindingLease interface {
	Snapshot() WorkspaceBindingSnapshot
	Revalidate() error
	Close() error
}
