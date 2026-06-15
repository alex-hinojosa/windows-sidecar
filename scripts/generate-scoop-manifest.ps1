[CmdletBinding()]
param(
    [string]$Version = "dev",
    [string]$Url = "",
    [string]$Sha256 = "",
    [string]$OutFile = ""
)

$ErrorActionPreference = "Stop"

$repoRoot = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot ".."))
$distDir = Join-Path $repoRoot "dist"
$zipName = "windows-sidecar-$Version-windows-amd64.zip"
$zipPath = Join-Path $distDir $zipName

if (-not $Url) {
    if (-not (Test-Path $zipPath)) {
        throw "Package not found: $zipPath. Run .\scripts\package-windows.ps1 -Version $Version first, or pass -Url and -Sha256."
    }
    $Url = $zipPath
}

if (-not $Sha256) {
    if (Test-Path $zipPath) {
        $Sha256 = (Get-FileHash -Algorithm SHA256 $zipPath).Hash.ToLowerInvariant()
    } else {
        throw "Pass -Sha256 when -Url points to a remote package not present locally."
    }
}

if (-not $OutFile) {
    $outDir = Join-Path $distDir "scoop"
    New-Item -ItemType Directory -Force -Path $outDir | Out-Null
    $OutFile = Join-Path $outDir "windows-sidecar.json"
}

$manifest = [ordered]@{
    version = $Version
    description = "Native Windows build of Sidecar and td for AI coding workflows"
    homepage = "https://github.com/marcus/sidecar"
    license = "MIT"
    architecture = [ordered]@{
        "64bit" = [ordered]@{
            url = $Url
            hash = $Sha256
        }
    }
    bin = @("sidecar.exe", "td.exe")
    checkver = [ordered]@{
        github = "https://github.com/marcus/sidecar"
    }
    notes = "Run 'sidecar --doctor' after install. Install psmux if workspace shell panes report mux issues."
}

$json = $manifest | ConvertTo-Json -Depth 6
Set-Content -Path $OutFile -Value $json -Encoding UTF8
Write-Host "Wrote Scoop manifest to $OutFile"
