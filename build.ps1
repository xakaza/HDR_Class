$ErrorActionPreference = "Stop"
$Version = if ($env:HDRSTUDIO_VERSION) { $env:HDRSTUDIO_VERSION } else { "1.1.7.0" }
New-Item -ItemType Directory -Force -Path "dist" | Out-Null

$env:GOOS = "windows"
$env:GOARCH = "amd64"
go build -trimpath -ldflags="-H=windowsgui -s -w" -o "dist/HDRStudio_x64.raw.exe" ./source

$env:GOARCH = "386"
go build -trimpath -ldflags="-H=windowsgui -s -w" -o "dist/HDRStudio_x86.raw.exe" ./source

python tools/embed_pe_resources.py dist/HDRStudio_x64.raw.exe source/hdrstudio.ico dist/HDRStudio_x64.exe --version $Version
python tools/embed_pe_resources.py dist/HDRStudio_x86.raw.exe source/hdrstudio.ico dist/HDRStudio_x86.exe --version $Version
Remove-Item dist/HDRStudio_x64.raw.exe, dist/HDRStudio_x86.raw.exe -Force

Get-FileHash dist/HDRStudio_x64.exe -Algorithm SHA256
Get-FileHash dist/HDRStudio_x86.exe -Algorithm SHA256
Write-Host "Built HDR Studio $Version"
