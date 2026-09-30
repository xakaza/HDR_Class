param(
    [Parameter(Mandatory=$true)][string]$PfxPath,
    [Parameter(Mandatory=$true)][string]$Password,
    [string]$TimestampUrl = "http://timestamp.digicert.com"
)

$ErrorActionPreference = "Stop"
$signtool = Get-ChildItem "${env:ProgramFiles(x86)}\Windows Kits\10\bin\*\x64\signtool.exe" |
    Sort-Object FullName -Descending |
    Select-Object -First 1 -ExpandProperty FullName
if (-not $signtool) { throw "SignTool.exe was not found. Install the Windows SDK." }

foreach ($file in @("dist\HDRStudio_x64.exe", "dist\HDRStudio_x86.exe")) {
    if (-not (Test-Path $file)) { throw "Missing file: $file" }
    & $signtool sign /fd SHA256 /f $PfxPath /p $Password /tr $TimestampUrl /td SHA256 /d "HDR Studio" $file
    if ($LASTEXITCODE -ne 0) { throw "Signing failed: $file" }
    & $signtool verify /pa /v $file
    if ($LASTEXITCODE -ne 0) { throw "Verification failed: $file" }
}

$items = Get-ChildItem dist\HDRStudio_*.exe | Sort-Object Name
$lines = foreach ($item in $items) {
    $hash = (Get-FileHash $item.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
    "$hash  $($item.Name)"
}
Set-Content -Path dist\SHA256SUMS.txt -Value $lines -Encoding ascii
Write-Host "Signed release binaries verified successfully."
