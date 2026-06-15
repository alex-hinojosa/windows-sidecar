# Windows Sidecar

Native Windows package for [marcus/sidecar](https://github.com/marcus/sidecar) and [marcus/td](https://github.com/marcus/td).

This bundle installs:

- `sidecar.exe` - TUI dashboard for AI coding agents
- `td.exe` - local task/session database CLI
- `doctor-windows.ps1` - Windows readiness check
- `sidecar-session.ps1` - Mini-style psmux/tmux launcher for a project

## Install From Zip

1. Extract `windows-sidecar-<version>-windows-<arch>.zip`.
2. Open PowerShell in the extracted folder.
3. Run:

```powershell
powershell -ExecutionPolicy Bypass -File .\install-windows.ps1
```

The installer copies files to:

```text
%LOCALAPPDATA%\Programs\WindowsSidecar\bin
```

It also adds that directory to your user PATH. Open a new terminal afterward.

## Check Install

```powershell
td --version
sidecar --version
sidecar --doctor
```

The only common warning is Windows Terminal not being detected. Sidecar works best in Windows Terminal because terminal rendering and key handling are better there.

## Run Sidecar For A Repo

Direct mode:

```powershell
cd C:\path\to\repo
sidecar -project .
```

Mini-style mux session mode:

```powershell
powershell -ExecutionPolicy Bypass -File "$env:LOCALAPPDATA\Programs\WindowsSidecar\bin\sidecar-session.ps1" -Project C:\path\to\repo
```

That creates a session named `sidecar-<repo-folder>` and attaches to it. Detach without stopping Sidecar:

```text
Ctrl-b, then d
```

Attach later:

```powershell
psmux list-sessions
psmux attach -t sidecar-<repo-folder>
```

`tmux` also works when psmux exposes its tmux-compatible command or Sidecar creates its `tmux.cmd` shim.

## Use Td

Inside a project:

```powershell
td init                  # first time only
td usage --new-session   # start-of-session context for agents
td status
td list
td current
td next
td reviewable
td show <td-id>
td context <td-id>
```

## Uninstall

From the extracted package or installed folder:

```powershell
powershell -ExecutionPolicy Bypass -File .\uninstall-windows.ps1 -RemoveFromPath
```

## Build From Source

From this repo:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\build-windows.ps1 -SmokeTest
```

If Go is not installed, the build script downloads a portable Go toolchain into `.tools/`.

Build outputs:

```text
bin\td.exe
bin\sidecar.exe
```

Install a source build:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\install-windows.ps1
```

Create release zips:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\release-windows.ps1 -Version dev -Arch amd64,arm64
```

## Layout

- `td/` - patched upstream `marcus/td`
- `sidecar/` - patched upstream `marcus/sidecar`, with `github.com/marcus/td` replaced by `../td`
- `scripts/` - Windows build/install/package helpers
- `docs/windows-port.md` - implementation notes and remaining work
- `docs/sharing-upstream.md` - how to publish the bundle and split upstream PRs
