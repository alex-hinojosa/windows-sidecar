[CmdletBinding()]
param(
    [string[]]$Path = @(),
    [string]$CertificateThumbprint = "",
    [string]$TimestampServer = "http://timestamp.digicert.com"
)

$ErrorActionPreference = "Stop"

$repoRoot = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot ".."))
if ($Path.Count -eq 0) {
    $Path = @(
        (Join-Path $repoRoot "bin\td.exe"),
        (Join-Path $repoRoot "bin\sidecar.exe")
    )
}

if ($CertificateThumbprint) {
    $cert = Get-ChildItem Cert:\CurrentUser\My, Cert:\LocalMachine\My |
        Where-Object { $_.Thumbprint -replace " ", "" -ieq ($CertificateThumbprint -replace " ", "") } |
        Select-Object -First 1
} else {
    $cert = Get-ChildItem Cert:\CurrentUser\My, Cert:\LocalMachine\My -CodeSigningCert |
        Where-Object { $_.HasPrivateKey } |
        Sort-Object NotAfter -Descending |
        Select-Object -First 1
}

if (-not $cert) {
    throw "No code-signing certificate found. Pass -CertificateThumbprint or import a code-signing certificate with a private key."
}

foreach ($item in $Path) {
    $resolved = [System.IO.Path]::GetFullPath($item)
    if (-not (Test-Path $resolved)) {
        throw "File not found: $resolved"
    }
    $signature = Set-AuthenticodeSignature -FilePath $resolved -Certificate $cert -TimestampServer $TimestampServer
    if ($signature.Status -ne "Valid") {
        throw "Signing failed for ${resolved}: $($signature.Status) $($signature.StatusMessage)"
    }
    Write-Host "Signed $resolved"
}
