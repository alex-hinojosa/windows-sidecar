[CmdletBinding()]
param(
    [ValidateSet("amd64", "arm64")]
    [string]$Arch = "amd64",
    [string]$Version = "dev",
    [string]$OutDir = "",
    [switch]$SkipBuild
)

$ErrorActionPreference = "Stop"

$repoRoot = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot ".."))
if (-not $OutDir) {
    $OutDir = Join-Path $repoRoot "dist"
}
$distPath = [System.IO.Path]::GetFullPath($OutDir)
$buildDir = Join-Path (Join-Path $repoRoot "bin") $Arch
$stageDir = Join-Path $distPath "windows-sidecar-$Version-windows-$Arch"
$zipPath = Join-Path $distPath "windows-sidecar-$Version-windows-$Arch.zip"
$checksumPath = Join-Path $distPath "windows-sidecar-$Version-windows-$Arch.sha256"

if (-not $SkipBuild) {
    & (Join-Path $PSScriptRoot "build-windows.ps1") -Arch $Arch -Version $Version -OutDir $buildDir -SmokeTest
}

if (Test-Path $stageDir) {
    Remove-Item -Recurse -Force $stageDir
}
New-Item -ItemType Directory -Force -Path $stageDir | Out-Null

Copy-Item -Force (Join-Path $buildDir "td.exe") (Join-Path $stageDir "td.exe")
Copy-Item -Force (Join-Path $buildDir "sidecar.exe") (Join-Path $stageDir "sidecar.exe")
Copy-Item -Force (Join-Path $repoRoot "README.md") (Join-Path $stageDir "README.md")
Copy-Item -Force (Join-Path $repoRoot "docs\windows-port.md") (Join-Path $stageDir "windows-port.md")
Copy-Item -Force (Join-Path $PSScriptRoot "doctor-windows.ps1") (Join-Path $stageDir "doctor-windows.ps1")
Copy-Item -Force (Join-Path $PSScriptRoot "install-package-windows.ps1") (Join-Path $stageDir "install-windows.ps1")
Copy-Item -Force (Join-Path $PSScriptRoot "sidecar-session.ps1") (Join-Path $stageDir "sidecar-session.ps1")
Copy-Item -Force (Join-Path $PSScriptRoot "uninstall-windows.ps1") (Join-Path $stageDir "uninstall-windows.ps1")

$packageInfo = [ordered]@{
    name = "windows-sidecar"
    version = $Version
    arch = $Arch
    builtAt = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
}
$packageInfo | ConvertTo-Json -Depth 3 | Set-Content -Path (Join-Path $stageDir "package-info.json") -Encoding UTF8

if (Test-Path $zipPath) {
    Remove-Item -Force $zipPath
}
Compress-Archive -Path (Join-Path $stageDir "*") -DestinationPath $zipPath -Force

$hash = Get-FileHash -Algorithm SHA256 $zipPath
Set-Content -Path $checksumPath -Encoding ASCII -Value "$($hash.Hash.ToLowerInvariant())  $(Split-Path -Leaf $zipPath)"

Write-Host "Packaged $zipPath"
Write-Host "Checksum $checksumPath"
