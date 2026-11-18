# install.ps1 - Global Windows Installer for Serve & Pro
[CmdletBinding()]
param()

$ErrorActionPreference = "Stop"

Write-Host "Installing Serve & Pro developer CLI..." -ForegroundColor Cyan

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Definition
$installDir = Join-Path $env:LOCALAPPDATA "serve\bin"
$localBin = Join-Path $HOME ".local\bin"

# 1. Build or locate binaries
$serveSrc = Join-Path $scriptDir "serve.exe"
$proSrc = Join-Path $scriptDir "pro.exe"

if (-not (Test-Path $serveSrc) -or -not (Test-Path $proSrc)) {
    Write-Host "Building production binaries from source..." -ForegroundColor Yellow
    Push-Location $scriptDir
    try {
        & go build -ldflags="-s -w" -o serve.exe ./cmd/serve
        if ($LASTEXITCODE -ne 0) { throw "Failed to compile serve.exe" }
        & go build -ldflags="-s -w" -o pro.exe ./cmd/pro
        if ($LASTEXITCODE -ne 0) { throw "Failed to compile pro.exe" }
    } finally {
        Pop-Location
    }
}

# 2. Ensure destination directories exist
if (-not (Test-Path $installDir)) {
    New-Item -ItemType Directory -Path $installDir -Force | Out-Null
}

# 3. Install all four command executables into installDir
Write-Host "Deploying executables to $installDir..." -ForegroundColor Gray
Copy-Item -Path $serveSrc -Destination (Join-Path $installDir "serve.exe") -Force
Copy-Item -Path $serveSrc -Destination (Join-Path $installDir "servecc.exe") -Force
Copy-Item -Path $proSrc   -Destination (Join-Path $installDir "pro.exe") -Force
Copy-Item -Path $proSrc   -Destination (Join-Path $installDir "procc.exe") -Force

# Also install into ~/.local/bin if present in user environment for instant availability
if (Test-Path $localBin) {
    Copy-Item -Path $serveSrc -Destination (Join-Path $localBin "serve.exe") -Force
    Copy-Item -Path $serveSrc -Destination (Join-Path $localBin "servecc.exe") -Force
    Copy-Item -Path $proSrc   -Destination (Join-Path $localBin "pro.exe") -Force
    Copy-Item -Path $proSrc   -Destination (Join-Path $localBin "procc.exe") -Force
}

# 4. Configure User PATH permanently
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ([string]::IsNullOrWhiteSpace($userPath)) {
    $userPath = ""
}

$pathEntries = ($userPath -split ';') | Where-Object { -not [string]::IsNullOrWhiteSpace($_) }
$normalizedInstall = (Resolve-Path $installDir).Path.TrimEnd('\')

$alreadyInPath = $false
foreach ($entry in $pathEntries) {
    if ($entry.TrimEnd('\') -ieq $normalizedInstall) {
        $alreadyInPath = $true
        break
    }
}

if (-not $alreadyInPath) {
    Write-Host "Adding $installDir to persistent User PATH..." -ForegroundColor Yellow
    $newUserPath = ($pathEntries + $installDir) -join ';'
    [Environment]::SetEnvironmentVariable("Path", $newUserPath, "User")
} else {
    Write-Host "User PATH already contains $installDir." -ForegroundColor Gray
}

# Also update the current session's PATH
$currentEntries = ($env:Path -split ';') | Where-Object { -not [string]::IsNullOrWhiteSpace($_) }
$inCurrent = $false
foreach ($entry in $currentEntries) {
    if ($entry.TrimEnd('\') -ieq $normalizedInstall) {
        $inCurrent = $true
        break
    }
}
if (-not $inCurrent) {
    $env:Path = "$installDir;$env:Path"
}

# 5. Verify command resolution
Write-Host "`nVerifying command resolution:" -ForegroundColor Cyan
$commands = @("serve", "servecc", "pro", "procc")
$allResolved = $true

foreach ($cmd in $commands) {
    $resolved = Get-Command $cmd -ErrorAction SilentlyContinue
    if ($resolved) {
        Write-Host "  [OK] $cmd -> $($resolved.Source)" -ForegroundColor Green
    } else {
        Write-Host "  [WARN] $cmd could not be resolved in the current session." -ForegroundColor Red
        $allResolved = $false
    }
}

Write-Host "`nServe & Pro successfully installed!" -ForegroundColor Green
Write-Host "Installed Location: $installDir" -ForegroundColor Gray
Write-Host "Available Commands:" -ForegroundColor White
Write-Host "  - serve    : Primary web serving command"
Write-Host "  - servecc  : Alternate web serving command"
Write-Host "  - pro      : Primary project management command"
Write-Host "  - procc    : Alternate project management command"
