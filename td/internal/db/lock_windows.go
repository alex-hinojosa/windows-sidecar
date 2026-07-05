//go:build windows

package db

import (
	"golang.org/x/sys/windows"
)

// lockByteOffset is the offset of the 1-byte range used for locking. Windows
// byte-range locks are mandatory: locking byte 0 would block readHolder's
// os.ReadFile (which uses a separate handle) from reading the holder-info
// text at the start of the file. Placing the lock range far beyond any data
// keeps exclusion intact while matching the advisory read semantics of flock
// on Unix. Locking beyond EOF is explicitly allowed by LockFileEx.
const lockByteOffset = 1 << 30

// tryLock attempts to acquire an exclusive lock without blocking.
// Returns nil on success, error if lock is held by another process.
func (l *writeLocker) tryLock() error {
	// LockFileEx with LOCKFILE_EXCLUSIVE_LOCK | LOCKFILE_FAIL_IMMEDIATELY
	// locks 1 byte at lockByteOffset
	ol := &windows.Overlapped{Offset: lockByteOffset}
	return windows.LockFileEx(
		windows.Handle(l.lockFile.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY,
		0, // reserved
		1, // lock 1 byte
		0, // high bits of length
		ol,
	)
}

// unlock releases the exclusive lock.
func (l *writeLocker) unlock() {
	if l.lockFile != nil {
		ol := &windows.Overlapped{Offset: lockByteOffset}
		windows.UnlockFileEx(
			windows.Handle(l.lockFile.Fd()),
			0, // reserved
			1, // unlock 1 byte
			0, // high bits of length
			ol,
		)
	}
}

// isProcessAlive checks if a process with the given PID is still running.
func isProcessAlive(pid int) bool {
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer windows.CloseHandle(handle)

	var exitCode uint32
	err = windows.GetExitCodeProcess(handle, &exitCode)
	if err != nil {
		return false
	}

	// STILL_ACTIVE (259) means process is running
	return exitCode == 259
}
