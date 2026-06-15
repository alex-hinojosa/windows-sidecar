//go:build windows

package workspace

import (
	"os"
	"syscall"

	"golang.org/x/sys/windows"
)

func tryAcquireManifestFileLock(f *os.File, exclusive bool) (bool, error) {
	flags := uint32(windows.LOCKFILE_FAIL_IMMEDIATELY)
	if exclusive {
		flags |= windows.LOCKFILE_EXCLUSIVE_LOCK
	}

	ol := new(windows.Overlapped)
	err := windows.LockFileEx(windows.Handle(f.Fd()), flags, 0, 1, 0, ol)
	if err == nil {
		return true, nil
	}

	if errno, ok := err.(syscall.Errno); ok && errno == windows.ERROR_LOCK_VIOLATION {
		return false, nil
	}
	return false, err
}

func releaseManifestFileLock(f *os.File) {
	ol := new(windows.Overlapped)
	_ = windows.UnlockFileEx(windows.Handle(f.Fd()), 0, 1, 0, ol)
}
