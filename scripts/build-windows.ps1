[CmdletBinding()]
param(
    [ValidateSet("amd64", "arm64")]
    [string]$Arch = "amd64",
    [string]$Version = "dev",
    [string]$OutDir = "",
    [switch]$UseSystemGo,
    [switch]$SmokeTest
)

$ErrorActionPreference = "Stop"

$repoRoot = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot ".."))
if (-not $OutDir) {
    $OutDir = Join-Path $repoRoot "bin"
}
$outPath = [System.IO.Path]::GetFullPath($OutDir)
New-Item -ItemType Directory -Force -Path $outPath | Out-Null

if ($UseSystemGo) {
    $goCommand = Get-Command go -ErrorAction Stop
    $goExe = $goCommand.Source
} else {
    $systemGo = Get-Command go -ErrorAction SilentlyContinue
    if ($systemGo) {
        $goExe = $systemGo.Source
    } else {
        $hostArch = switch ([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString()) {
            "Arm64" { "arm64" }
            default { "amd64" }
        }
        $goExe = & (Join-Path $PSScriptRoot "Ensure-Go.ps1") -Arch $hostArch
    }
}

$env:CGO_ENABLED = "0"
$env:GOOS = "windows"
$env:GOARCH = $Arch

$ldflags = "-s -w -X main.Version=$Version"

Push-Location (Join-Path $repoRoot "td")
try {
    & $goExe build -trimpath -ldflags $ldflags -o (Join-Path $outPath "td.exe") .
} finally {
    Pop-Location
}

Push-Location (Join-Path $repoRoot "sidecar")
try {
    & $goExe build -trimpath -ldflags $ldflags -o (Join-Path $outPath "sidecar.exe") .\cmd\sidecar
} finally {
    Pop-Location
}

Write-Host "Built:"
Get-Item (Join-Path $outPath "td.exe"), (Join-Path $outPath "sidecar.exe") |
    Select-Object FullName, Length |
    Format-Table -AutoSize

function Invoke-SmokeCommand {
    param(
        [string]$Path,
        [string[]]$Arguments,
        [int]$First = 0
    )

    $process = New-Object System.Diagnostics.Process
    $process.StartInfo.FileName = $Path
    $process.StartInfo.Arguments = ($Arguments -join " ")
    $process.StartInfo.UseShellExecute = $false
    $process.StartInfo.RedirectStandardOutput = $true
    $process.StartInfo.RedirectStandardError = $true
    $exitCode = 1
    try {
        $null = $process.Start()
        $stdout = $process.StandardOutput.ReadToEnd()
        $stderr = $process.StandardError.ReadToEnd()
        $process.WaitForExit()
        $exitCode = $process.ExitCode
    } finally {
        $process.Dispose()
    }
    $output = @()
    if ($stdout) {
        $output += $stdout -split "\r?\n"
    }
    if ($stderr) {
        $output += $stderr -split "\r?\n"
    }
    if ($exitCode -ne 0) {
        $text = (($output | ForEach-Object { "$_" }) -join "`n").Trim()
        throw "$Path $($Arguments -join ' ') failed with exit code ${exitCode}: $text"
    }
    if ($First -gt 0) {
        $output | Select-Object -First $First
    } else {
        $output
    }
}

if ($SmokeTest) {
    $hostArch = switch ([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString()) {
        "Arm64" { "arm64" }
        default { "amd64" }
    }
    $canRunTarget = ($Arch -eq $hostArch) -or ($hostArch -eq "arm64" -and $Arch -eq "amd64")
    if (-not $canRunTarget) {
        Write-Host "Skipping smoke test: built $Arch binaries cannot run on this $hostArch host."
    } else {
        Invoke-SmokeCommand -Path (Join-Path $outPath "td.exe") -Arguments @("--version")
        Invoke-SmokeCommand -Path (Join-Path $outPath "td.exe") -Arguments @("--help") -First 8
        Invoke-SmokeCommand -Path (Join-Path $outPath "sidecar.exe") -Arguments @("--version")
        Invoke-SmokeCommand -Path (Join-Path $outPath "sidecar.exe") -Arguments @("--help") -First 8
    }
}
