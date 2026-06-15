[CmdletBinding()]
param(
    [string]$Version = "dev",
    [string[]]$Arch = @("amd64"),
    [switch]$SkipTests
)

$ErrorActionPreference = "Stop"

$Arch = @(
    foreach ($item in $Arch) {
        foreach ($part in ($item -split ",")) {
            $normalized = $part.Trim().ToLowerInvariant()
            if ($normalized) {
                if ($normalized -notin @("amd64", "arm64")) {
                    throw "Unsupported architecture: $normalized. Use amd64 or arm64."
                }
                $normalized
            }
        }
    }
) | Select-Object -Unique
if ($Arch.Count -eq 0) {
    throw "Pass at least one architecture: amd64 or arm64."
}

$repoRoot = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot ".."))
$hostArch = switch ([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString()) {
    "Arm64" { "arm64" }
    default { "amd64" }
}
$goExe = & (Join-Path $PSScriptRoot "Ensure-Go.ps1") -Arch $hostArch
$env:PATH = "$(Split-Path -Parent $goExe);$env:PATH"
$tmpDir = Join-Path $repoRoot ".tmp-go"
New-Item -ItemType Directory -Force -Path $tmpDir | Out-Null
$env:GOTMPDIR = $tmpDir

if (-not $SkipTests) {
    Push-Location (Join-Path $repoRoot "td")
    try {
        go test ./internal/config
    } finally {
        Pop-Location
    }

    Push-Location (Join-Path $repoRoot "sidecar")
    try {
        go test ./internal/tty ./internal/plugins/workspace ./cmd/sidecar
    } finally {
        Pop-Location
    }
}

foreach ($targetArch in $Arch) {
    & (Join-Path $PSScriptRoot "package-windows.ps1") -Version $Version -Arch $targetArch
}

if ($Arch -contains "amd64") {
    & (Join-Path $PSScriptRoot "generate-scoop-manifest.ps1") -Version $Version
}

Write-Host ""
Write-Host "Release artifacts:"
Get-ChildItem (Join-Path $repoRoot "dist") -Filter "windows-sidecar-$Version-windows-*" |
    Select-Object FullName, Length |
    Format-Table -AutoSize
