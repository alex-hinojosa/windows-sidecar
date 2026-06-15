//go:build !windows

package config

import (
	"os"
	"syscall"
)

func acquireConfigLock(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_EX)
}

func releaseConfigLock(f *os.File) {
	_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
}
