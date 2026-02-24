$ErrorActionPreference = 'Stop'

Write-Host "== KrillinAI Windows Desktop Build =="

# Preconditions:
# - Go toolchain installed (go env shows windows/amd64)
# - MSYS2/MinGW-w64 installed and available on PATH (gcc)
# - (Optional) set CGO_ENABLED=1

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
  throw "Go is not installed or not on PATH"
}

$env:CGO_ENABLED = "1"

Write-Host "Go version:"; go version
Write-Host "Go env:"; go env GOOS GOARCH CGO_ENABLED

Write-Host "Running unit tests..."
go test ./...

Write-Host "Building desktop exe..."
$outDir = Join-Path (Get-Location) "build"
New-Item -ItemType Directory -Force -Path $outDir | Out-Null

# Build tags: desktop ensures main.go is used even if not on windows.
# On Windows, cmd/desktop/main.go is already enabled by //go:build desktop || windows.
go build -tags "desktop" -o (Join-Path $outDir "KrillinAI.exe") ./cmd/desktop

Write-Host "Built: $outDir\\KrillinAI.exe"
