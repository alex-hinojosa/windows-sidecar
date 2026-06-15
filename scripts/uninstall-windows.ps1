[CmdletBinding()]
param(
    [string]$InstallDir = "",
    [switch]$RemoveFromPath
)

$ErrorActionPreference = "Stop"

if (-not $InstallDir) {
    if (Test-Path (Join-Path $PSScriptRoot "sidecar.exe")) {
        $InstallDir = $PSScriptRoot
    } else {
        $InstallDir = Join-Path $env:LOCALAPPDATA "Programs\WindowsSidecar\bin"
    }
}

$installPath = [System.IO.Path]::GetFullPath($InstallDir)

if ($RemoveFromPath) {
    $currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($currentPath) {
        $parts = $currentPath -split ";" | Where-Object {
            $_ -and ([System.IO.Path]::GetFullPath($_.TrimEnd("\")) -ine $installPath)
        }
        [Environment]::SetEnvironmentVariable("Path", ($parts -join ";"), "User")
        Write-Host "Removed $installPath from the user PATH. Open a new terminal for the change to apply."
    }
}

if (Test-Path $installPath) {
    Remove-Item -Recurse -Force $installPath
    Write-Host "Removed $installPath"
} else {
    Write-Host "$installPath was not present"
}
