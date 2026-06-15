//go:build !windows

package main

import "log/slog"

func ensurePsmuxTmuxShim(_ *slog.Logger) {}
