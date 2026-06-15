package main

import (
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestEnsurePsmuxTmuxShimCreatesTmuxCommand(t *testing.T) {
	binDir := t.TempDir()
	fakePsmux := filepath.Join(binDir, "psmux.cmd")
	if err := os.WriteFile(fakePsmux, []byte("@echo off\r\necho psmux %*\r\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cacheDir := t.TempDir()
	t.Setenv("PATH", binDir)
	t.Setenv("LOCALAPPDATA", cacheDir)

	ensurePsmuxTmuxShim(slog.New(slog.NewTextHandler(os.Stderr, nil)))

	tmuxPath, err := exec.LookPath("tmux")
	if err != nil {
		t.Fatalf("tmux shim not found on PATH: %v", err)
	}
	if filepath.Base(tmuxPath) != "tmux.cmd" {
		t.Fatalf("LookPath(tmux) = %q, want tmux.cmd shim", tmuxPath)
	}

	out, err := exec.Command("tmux", "list-sessions").CombinedOutput()
	if err != nil {
		t.Fatalf("tmux shim command failed: %v\n%s", err, out)
	}
	if string(out) != "psmux list-sessions\r\n" {
		t.Fatalf("tmux shim output = %q", out)
	}
}
