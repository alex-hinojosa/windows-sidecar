# Sharing This Windows Work

This workspace is a Windows bundle repo. It is useful for producing installable
Windows artifacts that include both `td.exe` and `sidecar.exe`.

For upstream contribution, split the work by original project:

## Binary Bundle

Use this repo to publish easy Windows downloads:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\release-windows.ps1 -Version <version> -Arch amd64,arm64
```

Publish these files from `dist/`:

- `windows-sidecar-<version>-windows-amd64.zip`
- `windows-sidecar-<version>-windows-amd64.sha256`
- `windows-sidecar-<version>-windows-arm64.zip`
- `windows-sidecar-<version>-windows-arm64.sha256`

Users extract a zip and run:

```powershell
powershell -ExecutionPolicy Bypass -File .\install-windows.ps1
```

## Upstream Pull Requests

Open separate PRs:

1. `marcus/td`
   - Windows file locking for config writes.
   - Windows-safe tests.

2. `marcus/sidecar`
   - Windows file locking for shell manifests.
   - psmux/tmux compatibility fixes.
   - Windows doctor and helper scripts.
   - `.worktree-setup.ps1` support.
   - Windows CI updates.

Keep PRs small where possible. The bundle repo can remain the place where both
projects are built and packaged together.
