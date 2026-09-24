//go:build windows

package host

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modkernel32                      = windows.NewLazySystemDLL("kernel32.dll")
	procGetFinalPathNameByHandleW    = modkernel32.NewProc("GetFinalPathNameByHandleW")
	procGetFileInformationByHandleEx = modkernel32.NewProc("GetFileInformationByHandleEx")
)

const (
	VOLUME_NAME_DOS      = 0x0
	FILE_NAME_NORMALIZED = 0x0
	FileIdInfo           = 0x12 // FILE_INFO_BY_HANDLE_CLASS: FileIdInfo
)

type FILE_ID_128 struct {
	Identifier [16]byte
}

type FILE_ID_INFO struct {
	VolumeSerialNumber uint64
	FileId             FILE_ID_128
}

// PinnedDB holds an open OS handle to the database file omitting FILE_SHARE_DELETE,
// preventing concurrent file deletion, replacement, or directory unlinking while Store is open.
type PinnedDB struct {
	Handle          windows.Handle
	CanonicalDBPath string
	StoreDBPath     string
	PreIDInfo       FILE_ID_INFO
	ParentVolume    uint64
	IsNew           bool
	closed          bool
}

// GetFinalCanonicalPath derives the normalized canonical Win32 DOS path from an open OS handle.
func GetFinalCanonicalPath(h windows.Handle) (string, error) {
	buf := make([]uint16, 1024)
	r0, _, err := procGetFinalPathNameByHandleW.Call(
		uintptr(h),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
		uintptr(VOLUME_NAME_DOS|FILE_NAME_NORMALIZED),
	)
	if r0 == 0 {
		return "", fmt.Errorf("GetFinalPathNameByHandleW: %w", err)
	}
	return syscall.UTF16ToString(buf[:r0]), nil
}

// GetPhysicalFileIDInfo retrieves the 128-bit FileId and VolumeSerialNumber from an open OS handle.
func GetPhysicalFileIDInfo(h windows.Handle) (*FILE_ID_INFO, error) {
	var info FILE_ID_INFO
	r0, _, err := procGetFileInformationByHandleEx.Call(
		uintptr(h),
		uintptr(FileIdInfo),
		uintptr(unsafe.Pointer(&info)),
		uintptr(unsafe.Sizeof(info)),
	)
	if r0 == 0 {
		return nil, fmt.Errorf("GetFileInformationByHandleEx(FileIdInfo): %w", err)
	}
	return &info, nil
}

// ConvertWin32PathToStorePath validates that the canonical Win32 path is a local DOS volume (\\?\C:\...)
// and strips the \\?\ prefix so standard SQLite URI parsers accept it cleanly without syntax errors.
func ConvertWin32PathToStorePath(win32Path string) (string, error) {
	// Must match \\?\<Drive>:\... where Drive is a single ASCII letter
	if !strings.HasPrefix(win32Path, `\\?\`) {
		return "", fmt.Errorf("%w: path does not have Win32 extended prefix: %s", ErrInvalidPath, win32Path)
	}
	stripped := strings.TrimPrefix(win32Path, `\\?\`)
	if len(stripped) < 3 || stripped[1] != ':' || (stripped[2] != '\\' && stripped[2] != '/') {
		return "", fmt.Errorf("%w: path is not a local DOS drive (UNC or device paths forbidden): %s", ErrInvalidPath, win32Path)
	}
	driveLetter := stripped[0]
	if !((driveLetter >= 'A' && driveLetter <= 'Z') || (driveLetter >= 'a' && driveLetter <= 'z')) {
		return "", fmt.Errorf("%w: invalid drive letter in path: %s", ErrInvalidPath, win32Path)
	}
	return stripped, nil
}

// CheckHardlinks ensures the file does not have multiple hard links.
func CheckHardlinks(h windows.Handle) error {
	var byHandleInfo windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(h, &byHandleInfo); err != nil {
		return fmt.Errorf("GetFileInformationByHandle: %w", err)
	}
	if byHandleInfo.NumberOfLinks > 1 {
		return fmt.Errorf("%w: NumberOfLinks=%d", ErrHardlinkUnsupported, byHandleInfo.NumberOfLinks)
	}
	return nil
}

// PrepareExistingDB opens and pins an existing DB file, derives its canonical and Store paths,
// verifies nNumberOfLinks == 1, and records pre-open FILE_ID_INFO.
func PrepareExistingDB(rawPath string) (*PinnedDB, error) {
	pathUTF16, err := windows.UTF16PtrFromString(rawPath)
	if err != nil {
		return nil, fmt.Errorf("host: invalid DB path %q: %w", rawPath, err)
	}

	// Open handle omitting FILE_SHARE_DELETE: allows read and write by Store, blocks file removal/rename
	h, err := windows.CreateFile(
		pathUTF16,
		windows.GENERIC_READ,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		return nil, fmt.Errorf("host: failed to open existing DB file %q: %w", rawPath, err)
	}

	if err := CheckHardlinks(h); err != nil {
		_ = windows.CloseHandle(h)
		return nil, err
	}

	canonPath, err := GetFinalCanonicalPath(h)
	if err != nil {
		_ = windows.CloseHandle(h)
		return nil, fmt.Errorf("host: failed to get canonical path: %w", err)
	}

	storePath, err := ConvertWin32PathToStorePath(canonPath)
	if err != nil {
		_ = windows.CloseHandle(h)
		return nil, err
	}

	idInfo, err := GetPhysicalFileIDInfo(h)
	if err != nil {
		_ = windows.CloseHandle(h)
		return nil, fmt.Errorf("host: failed to get physical file ID info: %w", err)
	}

	return &PinnedDB{
		Handle:          h,
		CanonicalDBPath: canonPath,
		StoreDBPath:     storePath,
		PreIDInfo:       *idInfo,
		IsNew:           false,
	}, nil
}

// PrepareNewDB creates a 0-byte file using CREATE_NEW, pins its handle without FILE_SHARE_DELETE,
// and validates parent directory volume identity.
func PrepareNewDB(parentDir, fileName string) (*PinnedDB, error) {
	absParent, err := filepath.Abs(filepath.Clean(parentDir))
	if err != nil {
		return nil, fmt.Errorf("host: invalid parent dir %q: %w", parentDir, err)
	}
	parentUTF16, err := windows.UTF16PtrFromString(absParent)
	if err != nil {
		return nil, fmt.Errorf("host: invalid parent dir utf16: %w", err)
	}

	// Open parent directory to derive canonical path and volume serial number
	hParent, err := windows.CreateFile(
		parentUTF16,
		windows.GENERIC_READ,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_FLAG_BACKUP_SEMANTICS, // required to open directory
		0,
	)
	if err != nil {
		return nil, fmt.Errorf("host: failed to open parent directory %q: %w", absParent, err)
	}
	parentCanon, err := GetFinalCanonicalPath(hParent)
	parentID, errID := GetPhysicalFileIDInfo(hParent)
	_ = windows.CloseHandle(hParent)
	if err != nil {
		return nil, fmt.Errorf("host: failed to get canonical parent path: %w", err)
	}
	if errID != nil {
		return nil, fmt.Errorf("host: failed to get parent directory volume info: %w", errID)
	}

	parentStore, err := ConvertWin32PathToStorePath(parentCanon)
	if err != nil {
		return nil, err
	}

	targetCanon := filepath.Join(parentCanon, fileName)
	targetStore := filepath.Join(parentStore, fileName)

	targetUTF16, err := windows.UTF16PtrFromString(targetCanon)
	if err != nil {
		return nil, err
	}

	// Create 0-byte file exclusively via CREATE_NEW and pin with no FILE_SHARE_DELETE
	h, err := windows.CreateFile(
		targetUTF16,
		windows.GENERIC_READ|windows.GENERIC_WRITE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
		nil,
		windows.CREATE_NEW,
		windows.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		return nil, fmt.Errorf("host: failed to exclusive-create new DB file %q: %w", targetCanon, err)
	}

	idInfo, err := GetPhysicalFileIDInfo(h)
	if err != nil {
		_ = windows.CloseHandle(h)
		_ = os.Remove(targetStore)
		return nil, fmt.Errorf("host: failed to get new DB file ID info: %w", err)
	}

	return &PinnedDB{
		Handle:          h,
		CanonicalDBPath: targetCanon,
		StoreDBPath:     targetStore,
		PreIDInfo:       *idInfo,
		ParentVolume:    parentID.VolumeSerialNumber,
		IsNew:           true,
	}, nil
}

// VerifyPostOpenIdentity opens a validation handle to the database path after Store.Open()
// and compares VolumeSerialNumber and 128-bit FileId directly against the pinned OS handle identity.
func (p *PinnedDB) VerifyPostOpenIdentity() error {
	pathUTF16, err := windows.UTF16PtrFromString(p.StoreDBPath)
	if err != nil {
		return fmt.Errorf("host: verify identity path utf16 error: %w", err)
	}

	hCheck, err := windows.CreateFile(
		pathUTF16,
		windows.GENERIC_READ,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		return fmt.Errorf("host: post-open probe cannot open DB %q: %w", p.StoreDBPath, err)
	}
	defer windows.CloseHandle(hCheck)

	postInfo, err := GetPhysicalFileIDInfo(hCheck)
	if err != nil {
		return fmt.Errorf("host: post-open probe cannot query file ID: %w", err)
	}

	// Physical identity must match exactly between OS handles
	if postInfo.VolumeSerialNumber != p.PreIDInfo.VolumeSerialNumber ||
		!bytes.Equal(postInfo.FileId.Identifier[:], p.PreIDInfo.FileId.Identifier[:]) {
		return fmt.Errorf("%w: opened DB identity (vol=%x, id=%x) does not match pre-open pinned identity (vol=%x, id=%x)",
			ErrIdentityMismatch, postInfo.VolumeSerialNumber, postInfo.FileId.Identifier,
			p.PreIDInfo.VolumeSerialNumber, p.PreIDInfo.FileId.Identifier)
	}

	// For newly created DB, also verify volume matches parent directory volume
	if p.IsNew && p.ParentVolume != 0 && postInfo.VolumeSerialNumber != p.ParentVolume {
		return fmt.Errorf("%w: new DB volume %x does not match parent directory volume %x",
			ErrIdentityMismatch, postInfo.VolumeSerialNumber, p.ParentVolume)
	}

	return nil
}

// Close releases the pinned DB handle.
// In shutdown drain, this must be called AFTER Store.Close() and BEFORE ProcessOwnerLease.CloseLockHandle().
func (p *PinnedDB) Close() error {
	if p == nil || p.closed {
		return nil
	}
	p.closed = true
	if p.Handle != windows.InvalidHandle && p.Handle != 0 {
		return windows.CloseHandle(p.Handle)
	}
	return nil
}
