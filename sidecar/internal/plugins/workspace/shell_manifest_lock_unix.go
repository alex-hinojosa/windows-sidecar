//go:build !windows

package workspace

import (
	"os"
	"syscall"
)

func tryAcquireManifestFileLock(f *os.File, exclusive bool) (bool, error) {
	lockType := syscall.LOCK_SH | syscall.LOCK_NB
	if exclusive {
		lockType = syscall.LOCK_EX | syscall.LOCK_NB
	}

	err := syscall.Flock(int(f.Fd()), lockType)
	if err == nil {
		return true, nil
	}
	if err == syscall.EWOULDBLOCK || err == syscall.EAGAIN {
		return false, nil
	}
	return false, err
}

func releaseManifestFileLock(f *os.File) {
	_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
}
