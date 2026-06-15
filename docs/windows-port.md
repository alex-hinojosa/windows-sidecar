# Windows Port Notes

## What Changed

- Added Windows `LockFileEx` config locking in `td/internal/config`.
- Added Windows `LockFileEx` shell manifest locking in `sidecar/internal/plugins/workspace`.
- Added Windows to both GoReleaser target matrices.
- Added zip archive output for releases.
- Added `.worktree-setup.ps1` support for Sidecar worktree setup on Windows.
- Added `scripts/build-windows.ps1` and `scripts/install-windows.ps1`.
- Added `scripts/doctor-windows.ps1` for one-command Windows readiness checks.
- Added `scripts/package-windows.ps1` for shareable zip artifacts and checksums.
- Added `scripts/install-package-windows.ps1` so release zips install without Go or source files.
- Added `scripts/sidecar-session.ps1` for Mini-style psmux/tmux launch-and-attach sessions on Windows.
- Added `scripts/uninstall-windows.ps1` for local cleanup and optional PATH removal.
- Added `scripts/release-windows.ps1` to run tests and produce one or more Windows release zips.
- Added optional `scripts/sign-windows.ps1` and `scripts/generate-scoop-manifest.ps1` release helpers.
- Added root Windows CI workflow for tests, build, doctor, and package artifacts.
- Added a root `go.work` and a local `sidecar/go.mod` replace to build Sidecar against this repo's patched `td`.
- Hardened the Windows mux probe for psmux readiness races: it now waits for session, pane, dimensions, resize, send, and capture behavior instead of sampling once.
- Hardened Sidecar's runtime pane lookup and pane-size query so freshly-created psmux sessions do not get cached as missing panes.
- In psmux mode, Sidecar now falls back to session targets instead of pane-id targets because psmux can report pane ids like `%1` that are not globally unique across sessions.
- Changed Windows smoke tests to capture native process output before truncating help text, avoiding PowerShell pipe hangs with long CLI help output.
- Added package metadata and copied helper scripts into release zips so extracted packages are self-contained.

## Verified

On Windows amd64 with Go 1.25.8:

```powershell
.\scripts\build-windows.ps1 -SmokeTest
powershell -ExecutionPolicy Bypass -File .\scripts\doctor-windows.ps1 -Json
powershell -ExecutionPolicy Bypass -File .\scripts\release-windows.ps1 -Version dev -Arch amd64
```

The smoke test checks:

- `td.exe --version`
- `td.exe --help`
- `sidecar.exe --version`
- `sidecar.exe --help`

Additional local verification:

- `go test ./internal/tty ./internal/plugins/workspace ./cmd/sidecar` from `sidecar/`
- `go test ./internal/config` from `td/`
- Installed `sidecar.exe --doctor` passed 10 consecutive runs against psmux v3.3.4 with 0 failures.
- Package install smoke from extracted zip installs without source or Go.

## Known Gaps

- Sidecar workspace shells and agent panes are still tmux-command based. On Windows, Sidecar uses psmux through its tmux-compatible command surface and creates a `tmux.cmd` shim when real `tmux` is absent. The doctor now probes psmux successfully, but full parity still needs long-running interactive agent testing, and a ConPTY backend may still be cleaner long term.
- `.worktree-setup.sh` still requires `bash` on PATH. On Windows, prefer `.worktree-setup.ps1`.
- Release signing requires a real code-signing certificate. The signing script is present, but unsigned local builds remain the default.

## Suggested Next Work

1. Exercise real Sidecar workspace sessions in Windows Terminal for long-running agent output, resize, copy/paste, and attach/detach behavior.
2. Consider a native shell backend for Sidecar workspace panes using ConPTY.
3. Add a real code-signing certificate to the release environment.
4. Add a Winget manifest after a stable public release URL exists.
5. Expand smoke tests to create a temporary td project and launch Sidecar against it.
