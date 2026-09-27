package domain_test

import (
	"testing"

	"github.com/trungqwe/ai-supervisor/internal/domain"
)

func validTestBinding() domain.WorkspaceBinding {
	return domain.WorkspaceBinding{
		AttemptID:                   "attempt-001",
		TaskID:                      "task-001",
		ContractID:                  "contract-001",
		SessionID:                   "session-001",
		TerminalGeneration:          "gen-001",
		CanonicalWorktreePath:       "C:\\repo\\worktree",
		VolumeSerialHex:             "0000000012345678",
		FileIDHex:                   "00000000000000000000000012345678",
		LinkedGitDirPath:            "C:\\repo\\.git",
		LinkedGitDirVolumeSerialHex: "0000000012345678",
		LinkedGitDirFileIDHex:       "00000000000000000000000087654321",
		PinnedAOCommit:              "0123456789abcdef0123456789abcdef01234567",
		BindingState:                domain.BindingStateActive,
		CreatedAtEpochMS:            1000,
	}
}

func TestWorkspaceBinding_Validate(t *testing.T) {
	b := validTestBinding()
	if err := b.Validate(); err != nil {
		t.Fatalf("expected valid binding, got: %v", err)
	}

	// Invalid hex length
	bBadHex := b
	bBadHex.VolumeSerialHex = "1234"
	if err := bBadHex.Validate(); err == nil {
		t.Fatal("expected error on short volume_serial_hex")
	}

	// Uppercase hex
	bUpperHex := b
	bUpperHex.VolumeSerialHex = "000000001234567A"
	if err := bUpperHex.Validate(); err == nil {
		t.Fatal("expected error on uppercase volume_serial_hex")
	}

	// Released state without released_at
	bRel := b
	bRel.BindingState = domain.BindingStateReleased
	if err := bRel.Validate(); err == nil {
		t.Fatal("expected error on released state with nil released_at_epoch_ms")
	}

	now := int64(1500)
	bRel.ReleasedAtEpochMS = &now
	if err := bRel.Validate(); err != nil {
		t.Fatalf("expected valid released binding, got: %v", err)
	}

	// Released at before created at
	past := int64(500)
	bRel.ReleasedAtEpochMS = &past
	if err := bRel.Validate(); err == nil {
		t.Fatal("expected error on released_at < created_at")
	}
}

func TestWorkspaceBindingSnapshot_EqualsAndMatches(t *testing.T) {
	b := validTestBinding()
	snap := domain.WorkspaceBindingSnapshot{
		CanonicalWorktreePath:       b.CanonicalWorktreePath,
		WorktreeVolumeSerialHex:     b.VolumeSerialHex,
		WorktreeFileIDHex:           b.FileIDHex,
		LinkedGitDirPath:            b.LinkedGitDirPath,
		LinkedGitDirVolumeSerialHex: b.LinkedGitDirVolumeSerialHex,
		LinkedGitDirFileIDHex:       b.LinkedGitDirFileIDHex,
		PinnedAOCommit:              b.PinnedAOCommit,
	}

	if !snap.Equals(snap) {
		t.Fatal("expected snapshot to equal itself")
	}

	if !snap.MatchesBinding(&b) {
		t.Fatal("expected snapshot to match binding")
	}

	snap2 := snap
	snap2.WorktreeFileIDHex = "00000000000000000000000099999999"
	if snap.Equals(snap2) {
		t.Fatal("expected snapshot with different file ID to differ")
	}
	if snap2.MatchesBinding(&b) {
		t.Fatal("expected modified snapshot to not match binding")
	}
}
