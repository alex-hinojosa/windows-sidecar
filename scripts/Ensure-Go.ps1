[CmdletBinding()]
param(
    [string]$Version = "1.25.8",
    [ValidateSet("amd64", "arm64")]
    [string]$Arch = "amd64",
    [string]$ToolsDir = ""
)

$ErrorActionPreference = "Stop"

if (-not $ToolsDir) {
    $ToolsDir = Join-Path $PSScriptRoot "..\.tools"
}

$toolsPath = [System.IO.Path]::GetFullPath($ToolsDir)
$goRoot = Join-Path $toolsPath "go-$Version-windows-$Arch"
$goExe = Join-Path $goRoot "bin\go.exe"

if (Test-Path $goExe) {
    Write-Output $goExe
    exit 0
}

New-Item -ItemType Directory -Force -Path $toolsPath | Out-Null

$zipName = "go$Version.windows-$Arch.zip"
$zipPath = Join-Path $toolsPath $zipName
$url = "https://go.dev/dl/$zipName"

if (-not (Test-Path $zipPath)) {
    Write-Host "Downloading $url"
    Invoke-WebRequest -Uri $url -OutFile $zipPath -UseBasicParsing
}

$extractDir = Join-Path $toolsPath "extract-$Version-windows-$Arch"
if (Test-Path $extractDir) {
    Remove-Item -Recurse -Force $extractDir
}
New-Item -ItemType Directory -Force -Path $extractDir | Out-Null

tar -xf $zipPath -C $extractDir

if (Test-Path $goRoot) {
    Remove-Item -Recurse -Force $goRoot
}
Move-Item -Path (Join-Path $extractDir "go") -Destination $goRoot
Remove-Item -Recurse -Force $extractDir

if (-not (Test-Path $goExe)) {
    throw "Go executable was not found after extraction: $goExe"
}

Write-Output $goExe
