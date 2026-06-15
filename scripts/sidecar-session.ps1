[CmdletBinding()]
param(
    [string]$Project = ".",
    [string]$Name = "",
    [switch]$NoAttach,
    [switch]$Restart
)

$ErrorActionPreference = "Stop"

function Get-CommandPath {
    param([string]$Name)
    $cmd = Get-Command $Name -ErrorAction SilentlyContinue
    if ($cmd) {
        return $cmd.Source
    }
    return $null
}

function ConvertTo-SessionName {
    param([string]$Value)
    $name = $Value -replace '[\\/:.\s]+', '-'
    $name = $name -replace '[^A-Za-z0-9_-]', '-'
    $name = $name.Trim("-")
    if (-not $name) {
        return "project"
    }
    return $name.ToLowerInvariant()
}

$projectPath = [System.IO.Path]::GetFullPath($Project)
if (-not (Test-Path $projectPath -PathType Container)) {
    throw "Project directory not found: $projectPath"
}

if (-not $Name) {
    $Name = "sidecar-$(ConvertTo-SessionName (Split-Path -Leaf $projectPath))"
} else {
    $Name = ConvertTo-SessionName $Name
}

$mux = Get-CommandPath "tmux"
if (-not $mux) {
    $mux = Get-CommandPath "psmux"
}
if (-not $mux) {
    throw "Neither psmux nor tmux was found on PATH. Install psmux, then run sidecar --doctor."
}

$sidecar = Get-CommandPath "sidecar"
if (-not $sidecar) {
    $candidate = Join-Path $env:LOCALAPPDATA "Programs\WindowsSidecar\bin\sidecar.exe"
    if (Test-Path $candidate) {
        $sidecar = $candidate
    }
}
if (-not $sidecar) {
    $candidate = Join-Path $PSScriptRoot "sidecar.exe"
    if (Test-Path $candidate) {
        $sidecar = $candidate
    }
}
if (-not $sidecar) {
    throw "sidecar.exe was not found. Run install-windows.ps1 first."
}

$sessionExists = $false
& $mux has-session -t $Name *> $null
if ($LASTEXITCODE -eq 0) {
    $sessionExists = $true
}

if ($Restart -and $sessionExists) {
    & $mux kill-session -t $Name *> $null
    $sessionExists = $false
}

if (-not $sessionExists) {
    $command = "`"$sidecar`" -project `"$projectPath`""
    & $mux new-session -d -s $Name -c $projectPath $command
    if ($LASTEXITCODE -ne 0) {
        throw "Failed to create mux session: $Name"
    }
    Write-Host "Started Sidecar session '$Name' for $projectPath"
} else {
    Write-Host "Using existing Sidecar session '$Name'"
}

if (-not $NoAttach) {
    Write-Host "Detach without stopping Sidecar: Ctrl-b, then d"
    & $mux attach -t $Name
}
