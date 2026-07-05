//go:build windows

package session

import (
	"os"
	"testing"
)

// TestGetProcessInfoCurrentProcess verifies the Toolhelp32-based process
// lookup returns a real image name and parent PID for the current process.
func TestGetProcessInfoCurrentProcess(t *testing.T) {
	name, ppid, err := getProcessInfo(os.Getpid())
	if err != nil {
		t.Fatalf("getProcessInfo(%d) error: %v", os.Getpid(), err)
	}
	if name == "" {
		t.Error("name should not be empty for the current process")
	}
	if ppid <= 0 {
		t.Errorf("ppid = %d, want > 0 for the current process", ppid)
	}
}

// TestGetProcessInfoWalkToParent verifies the ppid returned for the current
// process can itself be resolved, i.e. the ancestry walk can take at least
// one step (the core of detectAgentAncestor).
func TestGetProcessInfoWalkToParent(t *testing.T) {
	_, ppid, err := getProcessInfo(os.Getpid())
	if err != nil {
		t.Fatalf("getProcessInfo(current) error: %v", err)
	}
	name, _, err := getProcessInfo(ppid)
	if err != nil {
		// The parent may legitimately have exited; only fail when it is alive.
		t.Skipf("parent process %d not resolvable (may have exited): %v", ppid, err)
	}
	if name == "" {
		t.Errorf("parent process %d resolved with empty name", ppid)
	}
}

// TestGetProcessInfoNotFound verifies a PID that cannot exist yields an error.
// Windows PIDs are multiples of 4, so an odd PID is never valid.
func TestGetProcessInfoNotFound(t *testing.T) {
	if _, _, err := getProcessInfo(0x7FFFFFFD); err == nil {
		t.Error("expected error for nonexistent PID, got nil")
	}
}
