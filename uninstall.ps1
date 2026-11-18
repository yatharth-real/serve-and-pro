# uninstall.ps1 - Global Windows Uninstaller for Serve & Pro
[CmdletBinding()]
param()

$ErrorActionPreference = "Stop"

Write-Host "Uninstalling Serve & Pro developer CLI..." -ForegroundColor Cyan

$installDir = Join-Path $env:LOCALAPPDATA "serve\bin"
$localBin = Join-Path $HOME ".local\bin"
$executables = @("serve.exe", "servecc.exe", "pro.exe", "procc.exe")

# 1. Remove executables from installDir
if (Test-Path $installDir) {
    foreach ($exe in $executables) {
        $exePath = Join-Path $installDir $exe
        if (Test-Path $exePath) {
            Remove-Item -Path $exePath -Force
            Write-Host "Removed $exe from $installDir" -ForegroundColor Gray
        }
    }

    $remaining = Get-ChildItem -Path $installDir -ErrorAction SilentlyContinue
    if (-not $remaining) {
        Remove-Item -Path $installDir -Force -Recurse -ErrorAction SilentlyContinue
    }
}

# 2. Remove executables from ~/.local/bin if present
if (Test-Path $localBin) {
    foreach ($exe in $executables) {
        $exePath = Join-Path $localBin $exe
        if (Test-Path $exePath) {
            Remove-Item -Path $exePath -Force
            Write-Host "Removed $exe from $localBin" -ForegroundColor Gray
        }
    }
}

# 3. Remove from User PATH
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if (-not [string]::IsNullOrWhiteSpace($userPath)) {
    $normalizedInstall = $installDir.TrimEnd('\')
    $pathEntries = ($userPath -split ';') | Where-Object {
        $trimmed = $_.Trim().TrimEnd('\')
        (-not [string]::IsNullOrWhiteSpace($trimmed)) -and ($trimmed -ine $normalizedInstall)
    }
    $newUserPath = $pathEntries -join ';'
    [Environment]::SetEnvironmentVariable("Path", $newUserPath, "User")
    Write-Host "Removed $installDir from persistent User PATH." -ForegroundColor Gray
}

# 4. Update current session PATH
$currentEntries = ($env:Path -split ';') | Where-Object {
    $trimmed = $_.Trim().TrimEnd('\')
    (-not [string]::IsNullOrWhiteSpace($trimmed)) -and ($trimmed -ine $installDir.TrimEnd('\'))
}
$env:Path = $currentEntries -join ';'

Write-Host "`nServe & Pro successfully uninstalled!" -ForegroundColor Green
Write-Host "Note: User projects and registry data (%APPDATA%\serve) were preserved." -ForegroundColor Gray
