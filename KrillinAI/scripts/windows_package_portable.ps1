$ErrorActionPreference = 'Stop'

function Fail([string]$Message) {
  Write-Host "[error] $Message" -ForegroundColor Red
  exit 1
}

Write-Host "== KrillinAI Windows Portable Package =="

$root = (Resolve-Path ".").Path
$buildDir = Join-Path $root "build"
$exe = Join-Path $buildDir "KrillinAI.exe"
$dataSrc = Join-Path $root "portable\\data"

if (-not (Test-Path $exe -PathType Leaf)) {
  Fail "Missing build artifact: $exe`nRun: .\\scripts\\windows_build_desktop.ps1"
}
if (-not (Test-Path $dataSrc -PathType Container)) {
  Fail "Missing portable data skeleton: $dataSrc"
}

$pkg = Join-Path $buildDir "KrillinAI_Windows_portable"
if (Test-Path $pkg) {
  Remove-Item -Recurse -Force $pkg
}
New-Item -ItemType Directory -Force -Path $pkg | Out-Null

Copy-Item -Path $exe -Destination (Join-Path $pkg "KrillinAI.exe")
Copy-Item -Path $dataSrc -Destination (Join-Path $pkg "data") -Recurse -Force

$zipPath = Join-Path $buildDir "KrillinAI_Windows_portable.zip"
if (Test-Path $zipPath -PathType Leaf) {
  Remove-Item -Force $zipPath
}
Compress-Archive -Path (Join-Path $pkg "*") -DestinationPath $zipPath

if (-not (Test-Path $zipPath -PathType Leaf)) {
  Fail "Zip was not created: $zipPath"
}
if ((Get-Item $zipPath).Length -le 0) {
  Fail "Zip was created but is empty: $zipPath"
}

Write-Host "[artifact] exe: $exe"
Write-Host "[artifact] package dir: $pkg"
Write-Host "[artifact] zip: $zipPath"
