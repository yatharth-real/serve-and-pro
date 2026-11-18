$ErrorActionPreference = "Stop"

$targets = @(
    @{ os = "windows"; arch = "amd64"; ext = ".exe"; nodeOS = "win32";  nodeArch = "x64" },
    @{ os = "windows"; arch = "arm64"; ext = ".exe"; nodeOS = "win32";  nodeArch = "arm64" },
    @{ os = "linux";   arch = "amd64"; ext = "";     nodeOS = "linux";  nodeArch = "x64" },
    @{ os = "linux";   arch = "arm64"; ext = "";     nodeOS = "linux";  nodeArch = "arm64" },
    @{ os = "darwin";  arch = "amd64"; ext = "";     nodeOS = "darwin"; nodeArch = "x64" },
    @{ os = "darwin";  arch = "arm64"; ext = "";     nodeOS = "darwin"; nodeArch = "arm64" }
)

$outDir = Join-Path $PSScriptRoot "..\dist-npm\native"
if (-not (Test-Path $outDir)) {
    New-Item -ItemType Directory -Force -Path $outDir | Out-Null
}

Write-Host "Building cross-platform native binaries for npm package..."

foreach ($t in $targets) {
    $serveTarget = Join-Path $outDir ("serve-" + $t.nodeOS + "-" + $t.nodeArch + $t.ext)
    $proTarget = Join-Path $outDir ("pro-" + $t.nodeOS + "-" + $t.nodeArch + $t.ext)

    Write-Host "Compiling $($t.os)/$($t.arch) -> $serveTarget"
    $env:GOOS = $t.os
    $env:GOARCH = $t.arch
    $env:CGO_ENABLED = "0"

    go build -ldflags="-s -w" -o $serveTarget ./cmd/serve
    go build -ldflags="-s -w" -o $proTarget ./cmd/pro
}

Write-Host "All targets compiled successfully."
