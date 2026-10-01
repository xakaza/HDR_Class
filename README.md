<div align="center">

<img src="assets/hdrstudio_256.png" alt="HDR Studio" width="128">

# HDR Studio

**Knight Online `.hdr` / `.src` Archive Pack, Unpack & Batch Tool**

[![Platform](https://img.shields.io/badge/Platform-Windows-0078D6?style=flat-square&logo=windows&logoColor=white)](https://github.com/akazaio/HDR_Class)
[![Language](https://img.shields.io/badge/Built%20with-Go-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev)
[![Architecture](https://img.shields.io/badge/Arch-x64%20%7C%20x86-orange?style=flat-square)]()

---

</div>

## Overview

HDR Studio is a native Windows GUI utility for `.hdr` + `.src` resource archives. It supports single-archive workflows, selective `.dxt` extraction, and recursive batch processing for a complete Knight Online client tree.

The UI supports **Türkçe / English** and long operations run on a worker thread so the window remains responsive.

## Features

- **Pack** a source folder into `.hdr` + `.src`
- **Unpack** one archive pair
- **DXT Only** — extract only `.dxt` entries while preserving their internal folders
- **Batch Unpack** — recursively discover every `.hdr` + matching `.src` under a selected root
- **Batch Pack** — recursively find `.hdr` files and rebuild them from their extracted working folders
- Default batch path: `C:\NTTGame\KnightOnlineEn`
- Optional separate batch output / working root
- Recursive directory layout is preserved when a separate root is selected
- Optional overwrite and backup behavior
- Drag & drop support
- Turkish / English interface
- Responsive background worker and cancellable operations
- Archive bounds validation and path-traversal protection
- Custom dark controls, bright owner-drawn checkbox labels, and double-buffered progress display
- Windows x64 + x86 builds

## Operation Modes

### Pack

Choose a source folder. HDR Studio creates:

```text
<folder-name>.hdr
<folder-name>.src
```

If an archive output folder is selected, the pair is written there. Otherwise it is written into the source folder.

### Unpack

Choose a `.hdr` file. Its matching `.src` is loaded automatically.

Enable **Extract .DXT files only** to skip every non-DXT entry. DXT-only extraction is intentionally non-destructive and never deletes the original HDR/SRC pair.

When no output path is selected:

```text
archive.hdr -> archive\...
```

DXT-only mode uses:

```text
archive.hdr -> archive_DXT\...
```

### Batch Unpack

Select a root folder. The default is:

```text
C:\NTTGame\KnightOnlineEn
```

HDR Studio scans every subfolder recursively and processes each valid `.hdr` + `.src` pair.

With no separate output root:

```text
C:\NTTGame\KnightOnlineEn\Data\UI.hdr
-> C:\NTTGame\KnightOnlineEn\Data\UI\...
```

With DXT-only enabled:

```text
-> C:\NTTGame\KnightOnlineEn\Data\UI_DXT\...
```

If a separate output root such as `D:\KO_Extracted` is selected, the original relative directory structure is retained:

```text
Data\UI.hdr
-> D:\KO_Extracted\Data\UI\...
```

Batch Unpack always keeps the source HDR/SRC archives so the operation remains reversible.

### Batch Pack

Select the client/archive root. HDR Studio recursively locates `.hdr` files and looks for the corresponding extracted directory.

Default sibling workflow:

```text
Data\UI.hdr
Data\UI.src
Data\UI\...          <- working folder
```

`Data\UI\` is packed back into the original `UI.hdr` / `UI.src` pair.

If files were Batch Unpacked to a separate working root, select that same root in **Extracted files root**. Relative folder mapping is preserved automatically.

Enabling backup creates `.bak` copies of existing archive files before replacement.

## Quick Start

1. Run `HDRStudio_x64.exe` on a modern 64-bit Windows system.
2. Choose **Türkçe** or **English**.
3. Select one of: Pack, Unpack, Batch Unpack, Batch Pack.
4. For batch operations, the client path is prefilled with `C:\NTTGame\KnightOnlineEn`.
5. Adjust output / working root if desired.
6. For texture-only extraction, enable **Yalnızca .DXT dosyalarını çıkar / Extract .DXT files only**.
7. Start the operation and follow progress in the activity log.

## Repository Layout

```text
.github/workflows/build.yml   Windows x64/x86 CI build and release
assets/                       Application artwork
source/
  main.go                     Complete application source
  hdrstudio.ico               Embedded application icon
tools/                        PE resource / icon build tools
dist/                         Prebuilt binaries and checksums
build.ps1                     Windows build script
build.sh                      Linux / WSL cross-build script
sign_release.ps1              Optional Authenticode helper
CODE_SIGNING.md               Signing setup
```

## Build

### Windows

Requirements: Go 1.23+, Python 3 and the PE resource tool dependencies used by the build script.

```powershell
./build.ps1
```

### Linux / WSL

```bash
./build.sh
```

### GitHub Actions

Pushing to `main` / `master` builds x64 and x86 artifacts automatically. Version `1.1.7.0` is used by the included workflow unless overridden.

## Notes

- Batch Pack intentionally uses normal full extraction folders, not `_DXT` folders. A DXT-only folder is a selective export and does not contain enough data to rebuild the whole archive.
- Unsigned executables can trigger Windows SmartScreen reputation warnings. Production signing support is documented in `CODE_SIGNING.md`.

---

<div align="center">

**HDR Studio**

</div>
