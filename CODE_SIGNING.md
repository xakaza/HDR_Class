# Windows code signing

Windows may show a SmartScreen or Unknown Publisher warning for unsigned Internet-downloaded executables. File metadata and a clean build help identify the application, but publisher trust comes from a valid Authenticode signature.

## GitHub Actions setup

The included workflow supports an exportable `.pfx` code-signing certificate.

Create these repository secrets:

- `WINDOWS_CERTIFICATE_BASE64` — base64 representation of the `.pfx`
- `WINDOWS_CERTIFICATE_PASSWORD` — password for the `.pfx`

Optional repository variable:

- `WINDOWS_TIMESTAMP_URL` — RFC 3161 timestamp endpoint. If omitted, the workflow uses `http://timestamp.digicert.com`.

PowerShell example for creating the base64 secret value locally:

```powershell
[Convert]::ToBase64String([IO.File]::ReadAllBytes("code-signing.pfx")) | Set-Clipboard
```

Never commit the `.pfx` file or its password to the repository.

The workflow performs these operations in order:

1. Build the binaries.
2. Embed icon, product/version metadata and the Windows manifest.
3. Sign the final `.exe` files with SHA-256.
4. Add an RFC 3161 SHA-256 timestamp.
5. Verify the Authenticode signature.
6. Generate SHA-256 file checksums.

If your signing identity is provided through a cloud/HSM service instead of an exportable `.pfx`, replace only the signing step with the action supplied by that provider. Keep signing after all resource changes and before checksum generation.

## Local SignTool example

```powershell
signtool sign /fd SHA256 /f code-signing.pfx /p "PASSWORD" /tr "http://timestamp.digicert.com" /td SHA256 /d "HDR Studio" dist\HDRStudio_x64.exe
signtool verify /pa /v dist\HDRStudio_x64.exe
```

A self-signed test certificate is useful only on machines where that certificate is explicitly trusted; it does not provide normal Internet publisher reputation.
