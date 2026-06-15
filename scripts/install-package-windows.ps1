[CmdletBinding()]
param(
    [string]$InstallDir = "",
    [switch]$NoPathUpdate,
    [switch]$SkipDoctor
)

$ErrorActionPreference = "Stop"

if (-not $InstallDir) {
    $InstallDir = Join-Path $env:LOCALAPPDATA "Programs\WindowsSidecar\bin"
}

$packageDir = [System.IO.Path]::GetFullPath($PSScriptRoot)
$installPath = [System.IO.Path]::GetFullPath($InstallDir)

foreach ($file in @("td.exe", "sidecar.exe", "doctor-windows.ps1")) {
    $source = Join-Path $packageDir $file
    if (-not (Test-Path $source)) {
        throw "Package is missing $file"
    }
}

$packageInfoPath = Join-Path $packageDir "package-info.json"
if (Test-Path $packageInfoPath) {
    $packageInfo = Get-Content -Raw $packageInfoPath | ConvertFrom-Json
    $hostArch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString()
    if ($packageInfo.arch -eq "arm64" -and $hostArch -ne "Arm64") {
        throw "This is an arm64 package, but this Windows host is $hostArch. Use the amd64 package."
    }
}

New-Item -ItemType Directory -Force -Path $installPath | Out-Null
Copy-Item -Force (Join-Path $packageDir "td.exe") (Join-Path $installPath "td.exe")
Copy-Item -Force (Join-Path $packageDir "sidecar.exe") (Join-Path $installPath "sidecar.exe")
Copy-Item -Force (Join-Path $packageDir "doctor-windows.ps1") (Join-Path $installPath "doctor-windows.ps1")
foreach ($helper in @("sidecar-session.ps1", "uninstall-windows.ps1", "package-info.json")) {
    $source = Join-Path $packageDir $helper
    if (Test-Path $source) {
        Copy-Item -Force $source (Join-Path $installPath $helper)
    }
}

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
Write-Host ""
Write-Host "Open a new terminal, then try:"
Write-Host "  td --version"
Write-Host "  sidecar --doctor"
Write-Host "  powershell -ExecutionPolicy Bypass -File `"$installPath\sidecar-session.ps1`" -Project C:\path\to\repo"

if (-not $SkipDoctor) {
    Write-Host ""
    & (Join-Path $installPath "sidecar.exe") --doctor
    if ($LASTEXITCODE -ne 0) {
        Write-Warning "Install completed, but the doctor reported issues. Re-run sidecar --doctor after fixing them."
    }
}
