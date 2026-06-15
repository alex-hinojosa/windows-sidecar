[CmdletBinding()]
param(
    [ValidateSet("amd64", "arm64")]
    [string]$Arch = "amd64",
    [string]$Version = "dev",
    [string]$InstallDir = "",
    [switch]$NoPathUpdate,
    [switch]$SkipDoctor
)

$ErrorActionPreference = "Stop"

$repoRoot = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot ".."))
$buildDir = Join-Path $repoRoot "bin"
if (-not $InstallDir) {
    $InstallDir = Join-Path $env:LOCALAPPDATA "Programs\WindowsSidecar\bin"
}

& (Join-Path $PSScriptRoot "build-windows.ps1") -Arch $Arch -Version $Version -OutDir $buildDir -SmokeTest

$installPath = [System.IO.Path]::GetFullPath($InstallDir)
New-Item -ItemType Directory -Force -Path $installPath | Out-Null
Copy-Item -Force (Join-Path $buildDir "td.exe") (Join-Path $installPath "td.exe")
Copy-Item -Force (Join-Path $buildDir "sidecar.exe") (Join-Path $installPath "sidecar.exe")
Copy-Item -Force (Join-Path $PSScriptRoot "doctor-windows.ps1") (Join-Path $installPath "doctor-windows.ps1")
Copy-Item -Force (Join-Path $PSScriptRoot "sidecar-session.ps1") (Join-Path $installPath "sidecar-session.ps1")
Copy-Item -Force (Join-Path $PSScriptRoot "uninstall-windows.ps1") (Join-Path $installPath "uninstall-windows.ps1")

if (-not $NoPathUpdate) {
    $currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
    $parts = @()
    if ($currentPath) {
        $parts = $currentPath -split ";"
    }
    $alreadyPresent = $parts | Where-Object {
        $_ -and ([System.IO.Path]::GetFullPath($_.TrimEnd("\")) -ieq $installPath)
    }
    if (-not $alreadyPresent) {
        $newPath = if ($currentPath) { "$currentPath;$installPath" } else { $installPath }
        [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
        Write-Host "Added to user PATH. Open a new terminal before running td or sidecar by name."
    }
}

Write-Host "Installed td.exe, sidecar.exe, and helper scripts to $installPath"

if (-not $SkipDoctor) {
    Write-Host ""
    powershell -ExecutionPolicy Bypass -File (Join-Path $PSScriptRoot "doctor-windows.ps1") -Fix -InstallDir $installPath -BuildDir $buildDir
    if ($LASTEXITCODE -ne 0) {
        Write-Warning "Install completed, but the doctor reported issues. Re-run sidecar --doctor after fixing them."
    }
}
