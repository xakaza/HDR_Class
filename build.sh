#!/usr/bin/env bash
set -euo pipefail
VERSION="${HDRSTUDIO_VERSION:-1.1.7.0}"
mkdir -p dist
GOOS=windows GOARCH=amd64 go build -trimpath -ldflags='-H=windowsgui -s -w' -o dist/HDRStudio_x64.raw.exe ./source
GOOS=windows GOARCH=386 go build -trimpath -ldflags='-H=windowsgui -s -w' -o dist/HDRStudio_x86.raw.exe ./source
python3 tools/embed_pe_resources.py dist/HDRStudio_x64.raw.exe source/hdrstudio.ico dist/HDRStudio_x64.exe --version "$VERSION"
python3 tools/embed_pe_resources.py dist/HDRStudio_x86.raw.exe source/hdrstudio.ico dist/HDRStudio_x86.exe --version "$VERSION"
rm -f dist/*.raw.exe
sha256sum dist/HDRStudio_x64.exe dist/HDRStudio_x86.exe > dist/SHA256SUMS.txt
printf 'Built HDR Studio %s\n' "$VERSION"
