# Windows Release Build (Desktop UI, Portable Default)

This doc describes how to build a **Windows desktop UI** release artifact that a normal user can download, unzip, and run.

## Goal

- Build `KrillinAI.exe` (desktop UI)
- Package a portable layout zip:
  - `KrillinAI.exe`
  - `data/` skeleton (created on first run)

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

## Verify (Smoke)

Follow:
- `docs/windows/smoke_test.md`

At minimum:

- unzip `KrillinAI_Windows_portable.zip` into an empty folder
- double click `KrillinAI.exe`
- confirm `data\config`, `data\logs`, `data\output`, `data\cache` created
- run `./KrillinAI.exe --diagnose` and confirm printed paths + deps summary
