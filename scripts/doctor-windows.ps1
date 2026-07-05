[CmdletBinding()]
param(
    [switch]$Fix,
    [switch]$Json,
    [switch]$SkipMuxProbe,
    [switch]$AllowMissingMux,
    [string]$InstallDir = "",
    [string]$BuildDir = ""
)

$ErrorActionPreference = "Stop"

$repoRoot = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot ".."))
if (-not $BuildDir) {
    $BuildDir = Join-Path $repoRoot "bin"
}
if (-not $InstallDir) {
    $InstallDir = Join-Path $env:LOCALAPPDATA "Programs\WindowsSidecar\bin"
}
$sourceCheckout = (Test-Path (Join-Path $repoRoot "td\go.mod")) -and (Test-Path (Join-Path $repoRoot "sidecar\go.mod"))

$script:Checks = New-Object System.Collections.Generic.List[object]

function Add-Check {
    param(
        [string]$Name,
        [ValidateSet("OK", "WARN", "FAIL", "INFO")]
        [string]$Status,
        [string]$Detail,
        [string]$Fix = ""
    )
    $script:Checks.Add([pscustomobject]@{
        name = $Name
        status = $Status
        detail = $Detail
        fix = $Fix
    })
}

function Get-CommandSource {
    param([string]$Name)
    $cmd = Get-Command $Name -ErrorAction SilentlyContinue
    if ($cmd) {
        return $cmd.Source
    }
    return $null
}

function Test-ExeVersion {
    param(
        [string]$Path,
        [string]$Name
    )
    if (-not (Test-Path $Path)) {
        Add-Check "$Name binary" "WARN" "Not found at $Path" "Run .\scripts\build-windows.ps1 -SmokeTest"
        return
    }
    try {
        $out = & $Path --version 2>&1 | Select-Object -First 1
        Add-Check "$Name binary" "OK" "$Path ($out)"
    } catch {
        Add-Check "$Name binary" "FAIL" "Could not run ${Path}: $($_.Exception.Message)" "Rebuild with .\scripts\build-windows.ps1 -SmokeTest"
    }
}

function Ensure-PsmuxShim {
    param([string]$PsmuxPath)

    $cacheDir = $env:LOCALAPPDATA
    if (-not $cacheDir) {
        $cacheDir = [System.IO.Path]::GetTempPath()
    }
    $shimDir = Join-Path $cacheDir "sidecar\psmux-shim"
    $shimPath = Join-Path $shimDir "tmux.cmd"

    New-Item -ItemType Directory -Force -Path $shimDir | Out-Null
    $content = "@echo off`r`n""$PsmuxPath"" %*`r`n"
    if ((-not (Test-Path $shimPath)) -or ((Get-Content -Raw $shimPath) -ne $content)) {
        Set-Content -Path $shimPath -Value $content -Encoding ASCII -NoNewline
    }

    $pathParts = @()
    if ($env:PATH) {
        $pathParts = $env:PATH -split ";"
    }
    if (-not ($pathParts | Where-Object { $_ -and ([System.IO.Path]::GetFullPath($_) -ieq [System.IO.Path]::GetFullPath($shimDir)) })) {
        $env:PATH = "$shimDir;$env:PATH"
    }
    return $shimPath
}

function Invoke-MuxCommand {
    param(
        [string]$MuxCommand,
        [string[]]$Arguments
    )

    # Windows PowerShell 5.1 promotes a native command's stderr (surfaced via 2>&1)
    # to a TERMINATING error when the caller's $ErrorActionPreference is 'Stop';
    # several mux probes (e.g. kill-session against a fresh GUID session) are
    # EXPECTED to write to stderr. Pin Continue locally so an expected failure is
    # captured as output instead of hard-throwing a spurious FAIL (M7).
    $prevEAP = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    try {
        $output = & $MuxCommand @Arguments 2>&1
        $exitCode = $LASTEXITCODE
    } finally {
        $ErrorActionPreference = $prevEAP
    }
    [pscustomobject]@{
        ExitCode = $exitCode
        Lines = @($output | ForEach-Object { "$_" })
    }
}

function Format-MuxOutput {
    param([object[]]$Lines)

    $text = (($Lines | ForEach-Object { "$_".Trim() }) -join " | ").Trim()
    if (-not $text) {
        return "<empty>"
    }
    return $text
}

function Wait-MuxSession {
    param(
        [string]$MuxCommand,
        [string]$Session
    )

    $last = $null
    for ($attempt = 1; $attempt -le 20; $attempt++) {
        $last = Invoke-MuxCommand -MuxCommand $MuxCommand -Arguments @("list-sessions", "-F", "#{session_name}")
        if ($last.ExitCode -eq 0 -and ($last.Lines | Where-Object { $_.Trim() -eq $Session })) {
            return
        }
        Start-Sleep -Milliseconds 100
    }
    throw "list-sessions did not return the disposable session (last exit $($last.ExitCode), output: $(Format-MuxOutput $last.Lines))"
}

function Wait-MuxPane {
    param(
        [string]$MuxCommand,
        [string]$Session
    )

    $last = $null
    for ($attempt = 1; $attempt -le 20; $attempt++) {
        $last = Invoke-MuxCommand -MuxCommand $MuxCommand -Arguments @("list-panes", "-t", $Session, "-F", "#{pane_id}")
        if ($last.ExitCode -eq 0) {
            foreach ($line in $last.Lines) {
                $pane = "$line".Trim()
                if ($pane) {
                    return $pane
                }
            }
        }
        Start-Sleep -Milliseconds 100
    }
    throw "list-panes did not return a pane id (last exit $($last.ExitCode), output: $(Format-MuxOutput $last.Lines))"
}

function Wait-MuxPaneSize {
    param(
        [string]$MuxCommand,
        [string]$Target
    )

    $last = $null
    for ($attempt = 1; $attempt -le 20; $attempt++) {
        $last = Invoke-MuxCommand -MuxCommand $MuxCommand -Arguments @("display-message", "-t", $Target, "-p", "#{pane_width},#{pane_height}")
        if ($last.ExitCode -eq 0 -and $last.Lines.Count -gt 0) {
            $size = "$($last.Lines[0])".Trim()
            if ($size -match "^\d+,\d+$") {
                return $size
            }
        }
        Start-Sleep -Milliseconds 100
    }
    throw "display-message did not return pane dimensions (last exit $($last.ExitCode), output: $(Format-MuxOutput $last.Lines))"
}

function Wait-MuxCaptureText {
    param(
        [string]$MuxCommand,
        [string]$Target,
        [string]$ExpectedText
    )

    $last = $null
    for ($attempt = 1; $attempt -le 20; $attempt++) {
        $last = Invoke-MuxCommand -MuxCommand $MuxCommand -Arguments @("capture-pane", "-p", "-e", "-t", $Target)
        if ($last.ExitCode -eq 0 -and (($last.Lines -join "`n") -match [regex]::Escape($ExpectedText))) {
            return
        }
        Start-Sleep -Milliseconds 100
    }
    throw "capture-pane did not include expected probe output (last exit $($last.ExitCode), output: $(Format-MuxOutput $last.Lines))"
}

function Invoke-MuxProbe {
    param([string]$MuxCommand)

    $session = "sidecar-doctor-$([Guid]::NewGuid().ToString('N').Substring(0, 8))"
    $created = $false
    try {
        Invoke-MuxCommand -MuxCommand $MuxCommand -Arguments @("kill-session", "-t", $session) | Out-Null

        $newSession = Invoke-MuxCommand -MuxCommand $MuxCommand -Arguments @("new-session", "-d", "-s", $session, "-c", $repoRoot)
        if ($newSession.ExitCode -ne 0) {
            throw "new-session failed with exit code $($newSession.ExitCode): $(Format-MuxOutput $newSession.Lines)"
        }
        $created = $true

        Wait-MuxSession -MuxCommand $MuxCommand -Session $session
        [void](Wait-MuxPane -MuxCommand $MuxCommand -Session $session)
        $target = $session
        [void](Wait-MuxPaneSize -MuxCommand $MuxCommand -Target $target)

        $resize = Invoke-MuxCommand -MuxCommand $MuxCommand -Arguments @("resize-window", "-t", $target, "-x", "100", "-y", "25")
        if ($resize.ExitCode -ne 0) {
            $resize = Invoke-MuxCommand -MuxCommand $MuxCommand -Arguments @("resize-pane", "-t", $target, "-x", "100", "-y", "25")
            if ($resize.ExitCode -ne 0) {
                throw "resize-window/resize-pane failed with exit code $($resize.ExitCode): $(Format-MuxOutput $resize.Lines)"
            }
        }

        $send = Invoke-MuxCommand -MuxCommand $MuxCommand -Arguments @("send-keys", "-t", $target, "echo sidecar-doctor", "Enter")
        if ($send.ExitCode -ne 0) {
            throw "send-keys failed with exit code $($send.ExitCode): $(Format-MuxOutput $send.Lines)"
        }

        Wait-MuxCaptureText -MuxCommand $MuxCommand -Target $target -ExpectedText "sidecar-doctor"

        Add-Check "mux compatibility" "OK" "Selected mux supports Sidecar's required session, pane, resize, send, and capture commands"
    } catch {
        Add-Check "mux compatibility" "FAIL" $_.Exception.Message "Update psmux/tmux, or run sidecar inside WSL until this probe passes"
    } finally {
        if ($created) {
            Invoke-MuxCommand -MuxCommand $MuxCommand -Arguments @("kill-session", "-t", $session) | Out-Null
        }
    }
}

$isWindowsHost = $IsWindows -or $env:OS -eq "Windows_NT"
Add-Check "Windows" ($(if ($isWindowsHost) { "OK" } else { "FAIL" })) ([System.Environment]::OSVersion.VersionString) ($(if ($isWindowsHost) { "" } else { "Run this doctor from Windows PowerShell" }))
Add-Check "PowerShell" "OK" "$($PSVersionTable.PSEdition) $($PSVersionTable.PSVersion)"

if ($env:WT_SESSION) {
    Add-Check "terminal" "OK" "Windows Terminal detected"
} else {
    Add-Check "terminal" "WARN" "Windows Terminal was not detected" "Use Windows Terminal for best rendering and keyboard behavior"
}

$git = Get-CommandSource "git"
if ($git) {
    Add-Check "git" "OK" "$git ($((git --version 2>$null) -join ' '))"
} else {
    Add-Check "git" "FAIL" "git was not found on PATH" "Install Git for Windows: winget install Git.Git"
}

$go = Get-CommandSource "go"
if ($go) {
	Add-Check "go" "OK" "$go ($((go version 2>$null) -join ' '))"
} else {
	$localGoExe = Get-ChildItem -Path (Join-Path $repoRoot ".tools") -Filter "go.exe" -Recurse -ErrorAction SilentlyContinue | Select-Object -First 1
	if ($localGoExe) {
		Add-Check "go" "OK" "Using portable Go at $($localGoExe.FullName)"
	} elseif (-not $sourceCheckout) {
		Add-Check "go" "INFO" "Go is not required for this installed binary package"
	} else {
		$localGo = Join-Path $repoRoot ".tools"
		Add-Check "go" "WARN" "go was not found on PATH; build scripts can use a portable toolchain under $localGo" "Run .\scripts\Ensure-Go.ps1"
	}
}

Test-ExeVersion (Join-Path $BuildDir "td.exe") "td"
Test-ExeVersion (Join-Path $BuildDir "sidecar.exe") "sidecar"

$installPath = [System.IO.Path]::GetFullPath($InstallDir)
if (Test-Path (Join-Path $installPath "sidecar.exe")) {
    Add-Check "install dir" "OK" "$installPath contains sidecar.exe"
} else {
    Add-Check "install dir" "INFO" "$installPath does not contain sidecar.exe" "Run .\scripts\install-windows.ps1"
}

$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -and (($userPath -split ";") | Where-Object { $_ -and ([System.IO.Path]::GetFullPath($_.TrimEnd("\")) -ieq $installPath) })) {
    Add-Check "user PATH" "OK" "$installPath is on the user PATH"
} else {
	Add-Check "user PATH" "INFO" "$installPath is not on the user PATH" "Add $installPath to your user PATH, then open a new terminal"
}

$tmux = Get-CommandSource "tmux"
$psmux = Get-CommandSource "psmux"
if ($tmux) {
    $help = (& tmux --help 2>&1 | Select-Object -First 1) -join " "
    Add-Check "mux command" "OK" "$tmux ($help)"
} elseif ($psmux) {
    if ($Fix) {
        $shim = Ensure-PsmuxShim -PsmuxPath $psmux
        Add-Check "mux command" "OK" "Created psmux shim: $shim"
        $tmux = Get-CommandSource "tmux"
    } else {
        Add-Check "mux command" "WARN" "psmux exists at $psmux, but the tmux-compatible command was not found" "Run .\scripts\doctor-windows.ps1 -Fix, or start Sidecar once to create its runtime shim"
    }
} else {
	$status = if ($AllowMissingMux) { "WARN" } else { "FAIL" }
	Add-Check "mux command" $status "Neither psmux nor tmux was found on PATH" "Install psmux, for example: winget search psmux"
}

if (-not $SkipMuxProbe) {
    if ($tmux) {
        Invoke-MuxProbe -MuxCommand "tmux"
	} elseif ($psmux) {
		Invoke-MuxProbe -MuxCommand "psmux"
	} else {
		$status = if ($AllowMissingMux) { "WARN" } else { "FAIL" }
		Add-Check "mux compatibility" $status "Skipped because no mux command is available" "Install psmux or tmux-compatible mux"
	}
}

$failCount = @($script:Checks | Where-Object { $_.status -eq "FAIL" }).Count
$warnCount = @($script:Checks | Where-Object { $_.status -eq "WARN" }).Count

if ($Json) {
    [pscustomobject]@{
        ok = $failCount -eq 0
        failures = $failCount
        warnings = $warnCount
        checks = $script:Checks
    } | ConvertTo-Json -Depth 5
} else {
    Write-Host "Windows Sidecar Doctor"
    Write-Host ""
    foreach ($check in $script:Checks) {
        $prefix = switch ($check.status) {
            "OK" { "[OK]  " }
            "WARN" { "[WARN]" }
            "FAIL" { "[FAIL]" }
            default { "[INFO]" }
        }
        Write-Host "$prefix $($check.name): $($check.detail)"
        if ($check.fix) {
            Write-Host "       Fix: $($check.fix)"
        }
    }
    Write-Host ""
    Write-Host "Summary: $failCount failure(s), $warnCount warning(s)"
}

if ($failCount -gt 0) {
    exit 1
}
