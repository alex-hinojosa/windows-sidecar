//go:build windows

package session

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

// getProcessInfo returns process name and parent PID for a given PID using a
// Toolhelp32 snapshot. This mirrors the contract of the Unix ps-based
// implementation so detectAgentAncestor can walk the process tree on Windows.
func getProcessInfo(pid int) (name string, ppid int, err error) {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return "", 0, fmt.Errorf("create process snapshot: %w", err)
	}
	defer windows.CloseHandle(snapshot)

	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))

	if err := windows.Process32First(snapshot, &entry); err != nil {
		return "", 0, fmt.Errorf("walk process snapshot: %w", err)
	}
	for {
		if int(entry.ProcessID) == pid {
			return windows.UTF16ToString(entry.ExeFile[:]), int(entry.ParentProcessID), nil
		}
		if err := windows.Process32Next(snapshot, &entry); err != nil {
			break
		}
	}
	return "", 0, fmt.Errorf("process not found: %d", pid)
}
