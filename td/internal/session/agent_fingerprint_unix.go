//go:build unix

package session

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// getProcessInfo returns process name and parent PID for a given PID
func getProcessInfo(pid int) (name string, ppid int, err error) {
	// Use ps command for cross-platform compatibility
	out, err := exec.Command("ps", "-o", "ppid=,comm=", "-p", strconv.Itoa(pid)).Output()
	if err != nil {
		return "", 0, err
	}

	line := strings.TrimSpace(string(out))
	if line == "" {
		return "", 0, fmt.Errorf("process not found: %d", pid)
	}

	parts := strings.Fields(line)
	if len(parts) < 2 {
		return "", 0, fmt.Errorf("unexpected ps output: %s", line)
	}

	ppid, err = strconv.Atoi(parts[0])
	if err != nil {
		return "", 0, err
	}

	// Join remaining parts as command name (may contain spaces)
	name = strings.Join(parts[1:], " ")
	return name, ppid, nil
}
