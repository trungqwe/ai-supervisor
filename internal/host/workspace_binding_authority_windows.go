//go:build windows

package host

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"unsafe"

	"github.com/trungqwe/ai-supervisor/internal/domain"
	"golang.org/x/sys/windows"
)

const (
	volumeNameDos uint32 = 0x0
)

type winFileIDInfo struct {
	VolumeSerialNumber uint64
	FileID             [16]byte
}

type windowsWorkspaceBindingLease struct {
	mu             sync.Mutex
	closed         bool
	worktreeHandle windows.Handle
	gitdirHandle   windows.Handle
	snapshot       domain.WorkspaceBindingSnapshot
}

func (l *windowsWorkspaceBindingLease) Snapshot() domain.WorkspaceBindingSnapshot {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.snapshot
}

func (l *windowsWorkspaceBindingLease) Revalidate() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return errors.New("host: workspace binding lease is closed")
	}

	wtVol, wtID, wtPath, err := queryHandleIdentity(l.worktreeHandle)
	if err != nil {
		return fmt.Errorf("host: failed to revalidate worktree handle: %w", err)
	}
	if wtVol != l.snapshot.WorktreeVolumeSerialHex || wtID != l.snapshot.WorktreeFileIDHex || wtPath != l.snapshot.CanonicalWorktreePath {
		return fmt.Errorf("host: worktree physical identity mismatch during revalidation")
	}

	gdVol, gdID, gdPath, err := queryHandleIdentity(l.gitdirHandle)
	if err != nil {
		return fmt.Errorf("host: failed to revalidate gitdir handle: %w", err)
	}
	if gdVol != l.snapshot.LinkedGitDirVolumeSerialHex || gdID != l.snapshot.LinkedGitDirFileIDHex || gdPath != l.snapshot.LinkedGitDirPath {
		return fmt.Errorf("host: gitdir physical identity mismatch during revalidation")
	}

	return nil
}

func (l *windowsWorkspaceBindingLease) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return nil
	}
	l.closed = true

	var errs []error
	if l.worktreeHandle != windows.InvalidHandle && l.worktreeHandle != 0 {
		if err := windows.CloseHandle(l.worktreeHandle); err != nil {
			errs = append(errs, err)
		}
		l.worktreeHandle = windows.InvalidHandle
	}
	if l.gitdirHandle != windows.InvalidHandle && l.gitdirHandle != 0 {
		if err := windows.CloseHandle(l.gitdirHandle); err != nil {
			errs = append(errs, err)
		}
		l.gitdirHandle = windows.InvalidHandle
	}

	if len(errs) > 0 {
		return fmt.Errorf("host: error closing workspace binding handles: %v", errs)
	}
	return nil
}

func openDirectoryHandleWithoutDeleteShare(dirPath string) (windows.Handle, error) {
	absPath, err := filepath.Abs(dirPath)
	if err != nil {
		return windows.InvalidHandle, err
	}

	path16, err := windows.UTF16PtrFromString(absPath)
	if err != nil {
		return windows.InvalidHandle, err
	}

	handle, err := windows.CreateFile(
		path16,
		windows.FILE_READ_ATTRIBUTES,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, // omitting FILE_SHARE_DELETE
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_FLAG_BACKUP_SEMANTICS,
		0,
	)
	if err != nil {
		return windows.InvalidHandle, err
	}
	return handle, nil
}

var ErrUnsupportedFilesystem = errors.New("host: unsupported filesystem")

func queryHandleIdentity(h windows.Handle) (volSerialHex string, fileIDHex string, canonicalPath string, err error) {
	var fsNameBuf [260]uint16
	err = windows.GetVolumeInformationByHandle(
		h,
		nil, 0,
		nil,
		nil,
		nil,
		&fsNameBuf[0], uint32(len(fsNameBuf)),
	)
	if err != nil {
		return "", "", "", fmt.Errorf("GetVolumeInformationByHandle failed: %w", err)
	}
	fsName := windows.UTF16ToString(fsNameBuf[:])
	if fsName != "NTFS" && fsName != "ReFS" {
		return "", "", "", fmt.Errorf("%w: unsupported filesystem %q (must be NTFS or ReFS)", ErrUnsupportedFilesystem, fsName)
	}

	var info winFileIDInfo
	err = windows.GetFileInformationByHandleEx(
		h,
		windows.FileIdInfo,
		(*byte)(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info)),
	)
	if err != nil {
		return "", "", "", fmt.Errorf("GetFileInformationByHandleEx(FileIdInfo) failed: %w", err)
	}

	volSerialHex = fmt.Sprintf("%016x", info.VolumeSerialNumber)
	fileIDHex = hex.EncodeToString(info.FileID[:])

	buf := make([]uint16, windows.MAX_PATH+1)
	n, err := windows.GetFinalPathNameByHandle(h, &buf[0], uint32(len(buf)), volumeNameDos)
	if err != nil {
		return "", "", "", fmt.Errorf("GetFinalPathNameByHandle failed: %w", err)
	}
	if n > uint32(len(buf)) {
		buf = make([]uint16, n)
		_, err = windows.GetFinalPathNameByHandle(h, &buf[0], uint32(len(buf)), volumeNameDos)
		if err != nil {
			return "", "", "", fmt.Errorf("GetFinalPathNameByHandle with resized buffer failed: %w", err)
		}
	}

	finalPath := windows.UTF16ToString(buf)
	canonicalPath = normalizeDosDevicePath(finalPath)
	return volSerialHex, fileIDHex, canonicalPath, nil
}

func normalizeDosDevicePath(p string) string {
	if strings.HasPrefix(p, `\\?\UNC\`) {
		return `\\` + filepath.Clean(strings.TrimPrefix(p, `\\?\UNC\`))
	}
	p = strings.TrimPrefix(p, `\\?\`)
	return filepath.Clean(p)
}

type WindowsWorkspaceBindingAuthority struct{}

func NewWindowsWorkspaceBindingAuthority() *WindowsWorkspaceBindingAuthority {
	return &WindowsWorkspaceBindingAuthority{}
}

func (a *WindowsWorkspaceBindingAuthority) Acquire(ctx context.Context, candidate domain.WorkspaceBindingCandidate) (domain.WorkspaceBindingLease, error) {
	if strings.TrimSpace(candidate.CanonicalWorktreePath) == "" {
		return nil, errors.New("host: candidate CanonicalWorktreePath must not be empty")
	}
	if strings.TrimSpace(candidate.LinkedGitDirPath) == "" {
		return nil, errors.New("host: candidate LinkedGitDirPath must not be empty")
	}
	if !domain.IsValidPinnedAOCommit(candidate.PinnedAOCommit) {
		return nil, fmt.Errorf("host: candidate PinnedAOCommit must be exactly 40 lowercase hex chars, got %q", candidate.PinnedAOCommit)
	}

	wtHandle, err := openDirectoryHandleWithoutDeleteShare(candidate.CanonicalWorktreePath)
	if err != nil {
		return nil, fmt.Errorf("host: failed to open worktree directory handle: %w", err)
	}

	gdHandle, err := openDirectoryHandleWithoutDeleteShare(candidate.LinkedGitDirPath)
	if err != nil {
		_ = windows.CloseHandle(wtHandle)
		return nil, fmt.Errorf("host: failed to open linked gitdir directory handle: %w", err)
	}

	wtVol, wtID, wtCanonical, err := queryHandleIdentity(wtHandle)
	if err != nil {
		_ = windows.CloseHandle(wtHandle)
		_ = windows.CloseHandle(gdHandle)
		return nil, fmt.Errorf("host: failed to query worktree physical identity: %w", err)
	}

	gdVol, gdID, gdCanonical, err := queryHandleIdentity(gdHandle)
	if err != nil {
		_ = windows.CloseHandle(wtHandle)
		_ = windows.CloseHandle(gdHandle)
		return nil, fmt.Errorf("host: failed to query linked gitdir physical identity: %w", err)
	}

	snapshot := domain.WorkspaceBindingSnapshot{
		CanonicalWorktreePath:       wtCanonical,
		WorktreeVolumeSerialHex:     wtVol,
		WorktreeFileIDHex:           wtID,
		LinkedGitDirPath:            gdCanonical,
		LinkedGitDirVolumeSerialHex: gdVol,
		LinkedGitDirFileIDHex:       gdID,
		PinnedAOCommit:              candidate.PinnedAOCommit,
	}

	return &windowsWorkspaceBindingLease{
		worktreeHandle: wtHandle,
		gitdirHandle:   gdHandle,
		snapshot:       snapshot,
	}, nil
}
