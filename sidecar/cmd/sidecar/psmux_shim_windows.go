package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
)

func ensurePsmuxTmuxShim(logger *slog.Logger) {
	if os.Getenv("SIDECAR_DISABLE_PSMUX_SHIM") == "1" {
		return
	}
	if _, err := exec.LookPath("tmux"); err == nil {
		return
	}

	psmuxPath, err := exec.LookPath("psmux")
	if err != nil {
		return
	}

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		cacheDir = os.TempDir()
	}
	shimDir := filepath.Join(cacheDir, "sidecar", "psmux-shim")
	if err := os.MkdirAll(shimDir, 0755); err != nil {
		logger.Debug("psmux shim: mkdir failed", "err", err)
		return
	}

	shimPath := filepath.Join(shimDir, "tmux.cmd")
	content := fmt.Sprintf("@echo off\r\n\"%s\" %%*\r\n", psmuxPath)
	if existing, err := os.ReadFile(shimPath); err != nil || string(existing) != content {
		if err := os.WriteFile(shimPath, []byte(content), 0644); err != nil {
			logger.Debug("psmux shim: write failed", "path", shimPath, "err", err)
			return
		}
	}

	path := os.Getenv("PATH")
	os.Setenv("PATH", shimDir+string(os.PathListSeparator)+path)
	logger.Debug("psmux shim: using psmux as tmux", "shim", shimPath, "psmux", psmuxPath)
}
