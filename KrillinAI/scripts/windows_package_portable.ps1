$ErrorActionPreference = 'Stop'

Write-Host "== KrillinAI Windows Portable Package =="

$root = Get-Location
$buildDir = Join-Path $root "build"
$exe = Join-Path $buildDir "KrillinAI.exe"
if (-not (Test-Path $exe)) {
  throw "Missing $exe. Run scripts/windows_build_desktop.ps1 first."
}

$pkg = Join-Path $buildDir "KrillinAI_Windows_portable"
if (Test-Path $pkg) { Remove-Item -Recurse -Force $pkg }
New-Item -ItemType Directory -Force -Path $pkg | Out-Null

Copy-Item $exe (Join-Path $pkg "KrillinAI.exe")

# Copy portable data skeleton
$dataSrc = Join-Path $root "portable\\data"
$dataDst = Join-Path $pkg "data"
Copy-Item -Recurse -Force $dataSrc $dataDst

# Zip
$zipPath = Join-Path $buildDir "KrillinAI_Windows_portable.zip"
if (Test-Path $zipPath) { Remove-Item -Force $zipPath }
Compress-Archive -Path (Join-Path $pkg "*") -DestinationPath $zipPath

Write-Host "Packaged: $zipPath"
