$ErrorActionPreference = 'Stop'

function Fail([string]$Message) {
  Write-Host "[error] $Message" -ForegroundColor Red
  exit 1
}

Write-Host "== KrillinAI Windows Desktop Build =="

$root = (Resolve-Path ".").Path
$outDir = Join-Path $root "build"
$exePath = Join-Path $outDir "KrillinAI.exe"
$buildCommand = 'go build -tags "desktop" -o "build\KrillinAI.exe" ./cmd/desktop'

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
  Fail "go was not found on PATH. Install Go and reopen PowerShell."
}

Write-Host "[preflight] go version"
& go version
if ($LASTEXITCODE -ne 0) {
  Fail "go version failed."
}

Write-Host "[preflight] go env GOOS GOARCH CGO_ENABLED CC CXX"
& go env GOOS GOARCH CGO_ENABLED CC CXX
if ($LASTEXITCODE -ne 0) {
  Fail "go env failed."
}

$gcc = Get-Command gcc -ErrorAction SilentlyContinue
if (-not $gcc) {
  Write-Host "[error] gcc was not found on PATH (required for Fyne/GLFW cgo builds)." -ForegroundColor Red
  Write-Host "Install MSYS2 + MinGW-w64:"
  Write-Host "  1) https://www.msys2.org/"
  Write-Host "  2) In MSYS2 shell:"
  Write-Host "     pacman -Syu"
  Write-Host "     pacman -S --needed mingw-w64-x86_64-toolchain"
  Write-Host "  3) Add this to Windows PATH:"
  Write-Host "     C:\\msys64\\mingw64\\bin"
  Write-Host "  4) Restart PowerShell and verify:"
  Write-Host "     gcc --version"
  exit 1
}

Write-Host "[preflight] gcc --version"
& gcc --version
if ($LASTEXITCODE -ne 0) {
  Fail "gcc was found but could not run."
}

$env:CGO_ENABLED = "1"

New-Item -ItemType Directory -Force -Path $outDir | Out-Null

Write-Host "[build] $buildCommand"
& go build -tags "desktop" -o $exePath ./cmd/desktop
if ($LASTEXITCODE -ne 0) {
  Write-Host "[error] Desktop build failed." -ForegroundColor Red
  Write-Host "Command:"
  Write-Host "  $buildCommand"
  Write-Host "Likely causes:"
  Write-Host "  - C:\\msys64\\mingw64\\bin is not on PATH"
  Write-Host '  - CGO is disabled (run: $env:CGO_ENABLED = "1")'
  Write-Host "  - MinGW-w64 toolchain is not installed in MSYS2"
  exit $LASTEXITCODE
}

if (-not (Test-Path $exePath -PathType Leaf)) {
  Fail "Build completed without output file: $exePath"
}

Write-Host "[ok] built: $exePath"
