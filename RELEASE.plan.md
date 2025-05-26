# Treb Release Plan

This document outlines the binary build and release workflow needed to support the `trebup` installer.

## Overview

The `trebup` installer expects to download pre-built binaries from GitHub releases in a specific format. This plan describes the necessary build and release automation to make this work seamlessly.

## Release Artifact Structure

### Binary Naming Convention
Releases should include binaries with the following naming pattern:
```
treb_{version}_{platform}_{architecture}.{extension}
```

Where:
- `version`: The release tag (e.g., `v0.2.0`, `stable`, `nightly`)
- `platform`: `linux`, `darwin`, or `windows`
- `architecture`: `amd64` or `arm64`
- `extension`: `tar.gz` for Unix-like systems, `zip` for Windows

### Example Release Artifacts
For a release tagged `v0.2.0`, the following artifacts should be created:
```
treb_v0.2.0_linux_amd64.tar.gz
treb_v0.2.0_linux_arm64.tar.gz
treb_v0.2.0_darwin_amd64.tar.gz
treb_v0.2.0_darwin_arm64.tar.gz
treb_v0.2.0_windows_amd64.zip
treb_v0.2.0_windows_arm64.zip
```

Each archive should contain the `treb` binary at the root level.

## GitHub Actions Workflow

### 1. Build Matrix
Create `.github/workflows/release.yml` with a build matrix covering:
- OS: Ubuntu (linux), macOS (darwin), Windows
- Architecture: amd64 (x86_64), arm64 (aarch64)
- Go version: Use latest stable Go

### 2. Build Process
For each platform/architecture combination:
1. Set up Go environment
2. Set appropriate `GOOS` and `GOARCH` environment variables
3. Build with: `go build -o treb ./cli`
4. Create archive with correct naming convention
5. Upload as release artifact

### 3. Release Triggers
The workflow should trigger on:
- Push to tags matching `v*` (e.g., `v0.2.0`)
- Manual dispatch for nightly builds
- Scheduled nightly builds (optional)

### 4. Special Releases
- **Stable**: Point to the latest versioned release
- **Nightly**: Built from main branch daily

## Implementation Tasks

### Phase 1: Basic Release Workflow
1. **Create GitHub Actions workflow** (`release.yml`)
   - Set up build matrix for all platforms/architectures
   - Implement cross-compilation for Go
   - Create properly named archives
   - Upload to GitHub releases

2. **Update Makefile**
   - Add `make release-build` target for local testing
   - Support cross-compilation targets
   - Generate archives locally

3. **Test release process**
   - Create a test release
   - Verify trebup can download and install it
   - Test on all supported platforms

### Phase 2: Stable/Nightly Channels
1. **Implement stable channel**
   - Create a "stable" release that points to latest version
   - Update stable release on each new version
   - Ensure trebup handles this correctly

2. **Implement nightly channel**
   - Set up scheduled GitHub Action (daily at midnight UTC)
   - Build from main branch
   - Create/update "nightly" release
   - Include commit SHA in binary version

### Phase 3: Enhanced Features
1. **Binary signing** (optional but recommended)
   - Sign macOS binaries for Gatekeeper
   - Sign Windows binaries
   - Generate and publish checksums

2. **Homebrew integration** (for macOS)
   - Create Homebrew formula
   - Auto-update on releases

3. **Version embedding**
   - Embed git commit and version in binary
   - Update `treb version` to show this info

## Example GitHub Actions Workflow

```yaml
name: Release

on:
  push:
    tags:
      - 'v*'
  workflow_dispatch:

jobs:
  build:
    strategy:
      matrix:
        include:
          - os: ubuntu-latest
            goos: linux
            goarch: amd64
          - os: ubuntu-latest
            goos: linux
            goarch: arm64
          - os: macos-latest
            goos: darwin
            goarch: amd64
          - os: macos-latest
            goos: darwin
            goarch: arm64
          - os: windows-latest
            goos: windows
            goarch: amd64
            extension: .exe

    runs-on: ${{ matrix.os }}
    
    steps:
      - uses: actions/checkout@v4
      
      - uses: actions/setup-go@v5
        with:
          go-version: '1.21'
      
      - name: Build
        env:
          GOOS: ${{ matrix.goos }}
          GOARCH: ${{ matrix.goarch }}
        run: |
          go build -o treb${{ matrix.extension }} ./cli
      
      - name: Create archive
        run: |
          if [ "${{ matrix.goos }}" = "windows" ]; then
            7z a treb_${{ github.ref_name }}_${{ matrix.goos }}_${{ matrix.goarch }}.zip treb.exe
          else
            tar czf treb_${{ github.ref_name }}_${{ matrix.goos }}_${{ matrix.goarch }}.tar.gz treb
          fi
      
      - name: Upload release artifact
        uses: actions/upload-artifact@v4
        with:
          name: treb_${{ github.ref_name }}_${{ matrix.goos }}_${{ matrix.goarch }}
          path: treb_*.{tar.gz,zip}

  release:
    needs: build
    runs-on: ubuntu-latest
    
    steps:
      - name: Download artifacts
        uses: actions/download-artifact@v4
      
      - name: Create release
        uses: softprops/action-gh-release@v1
        with:
          files: treb_*/*.{tar.gz,zip}
          draft: false
          prerelease: false
```

## Verification Checklist

- [ ] Binary builds successfully for all platforms/architectures
- [ ] Release artifacts follow correct naming convention
- [ ] trebup can download and install from releases
- [ ] Version information is correctly embedded in binary
- [ ] Stable channel points to latest release
- [ ] Nightly builds work (if implemented)
- [ ] Installation works on fresh systems without Go installed

## Future Enhancements

1. **APT/YUM repositories** for Linux distributions
2. **MSI installer** for Windows
3. **Docker images** with treb pre-installed
4. **Release notes automation** from commit messages
5. **Binary size optimization** with build flags