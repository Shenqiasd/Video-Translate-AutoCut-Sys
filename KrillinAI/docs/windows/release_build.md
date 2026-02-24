# Windows Release Build (Desktop UI, Portable Default)

This doc describes how to build a **Windows desktop UI** release artifact that a normal user can download, unzip, and run.

## Goal

- Build `KrillinAI.exe` (desktop UI)
- Package a portable layout zip:
  - `KrillinAI.exe`
  - `data/` skeleton (created on first run)
- Target architecture: `windows/amd64` only (no arm64 portable zip in current release flow)

## Preconditions (Windows machine)

- Windows 10/11 x64
- Go installed (match project requirement)
- Git installed
- **MSYS2 + MinGW-w64** toolchain installed (required by Fyne/GLFW)
  - Install MSYS2: https://www.msys2.org/
  - In MSYS2 shell:
    - `pacman -Syu`
    - `pacman -S --needed mingw-w64-x86_64-toolchain`
  - Ensure MinGW `bin` is on PATH for PowerShell builds, typically:
    - `C:\msys64\mingw64\bin`

## Build (PowerShell)

From repo `KrillinAI/` directory:

```powershell
# Build desktop exe
./scripts/windows_build_desktop.ps1

# Package portable zip (exe + data/ skeleton)
./scripts/windows_package_portable.ps1
```

Outputs:

- `build\KrillinAI.exe`
- `build\KrillinAI_Windows_portable.zip`

## CI Release (GitHub Tags)

The effective GitHub Actions workflow is at the repository root:

- `/.github/workflows/release.yml`

> Note: `KrillinAI/.github/workflows/release.yml` is kept for module-local reference, but GitHub does not discover workflows from that nested path.

Pushing a tag like `v1.2.3` now does:

- build Windows desktop UI on `windows-latest` (`amd64` only)
- install MSYS2 in CI using `msys2/setup-msys2@v2` and install `mingw-w64-x86_64-toolchain`
- add `mingw64\bin` to PATH in PowerShell steps before building, so `gcc` is available for cgo/Fyne
- run in `KrillinAI/` working directory:
  - `scripts/windows_build_desktop.ps1`
  - `scripts/windows_package_portable.ps1`
- upload `KrillinAI/build/KrillinAI_Windows_portable.zip` as a workflow artifact
- download that zip in the GoReleaser job to `KrillinAI/build/`, where `release.extra_files` attaches it to the GitHub Release as a downloadable asset

CI constraint notes:

- Windows portable zip artifact is produced only for `amd64`.
- CI assumes MSYS2 MINGW64 layout (`<msys2-root>\mingw64\bin`) for gcc toolchain wiring.

## Troubleshooting

### `go` is not found

- Install Go (version from `go.mod`), then reopen PowerShell.
- Verify:
  - `go version`
  - `go env GOOS GOARCH CGO_ENABLED CC CXX`

### `gcc` is not found (required for desktop cgo build)

- Install MSYS2: https://www.msys2.org/
- In MSYS2 shell:
  - `pacman -Syu`
  - `pacman -S --needed mingw-w64-x86_64-toolchain`
- Add to Windows PATH:
  - `C:\msys64\mingw64\bin`
- Reopen PowerShell and verify:
  - `gcc --version`

### `go build ./cmd/desktop` fails

The build script prints the exact failed command. Most common causes:

- `C:\msys64\mingw64\bin` is not on PATH
- cgo is disabled (set `CGO_ENABLED=1`)
- MinGW-w64 toolchain is missing or incomplete

PowerShell quick fix for current shell:

- `$env:CGO_ENABLED = "1"`

### Portable packaging fails

- If `build\KrillinAI.exe` is missing, run:
  - `./scripts/windows_build_desktop.ps1`
- If `portable\data` is missing, restore it from repo before packaging.
- After packaging, confirm:
  - `build\KrillinAI_Windows_portable.zip` exists and is non-empty
  - script output shows artifact paths for exe, package dir, and zip

## Verify (Smoke)

Follow:
- `docs/windows/smoke_test.md`

At minimum:

- unzip `KrillinAI_Windows_portable.zip` into an empty folder
- double click `KrillinAI.exe`
- confirm `data\config`, `data\logs`, `data\output`, `data\cache` created
- run `./KrillinAI.exe --diagnose` and confirm printed paths + deps summary
