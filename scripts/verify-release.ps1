#!/usr/bin/env pwsh
# PRO 3.0.0 Release Verification
# Run from workspace root: .\scripts\verify-release.ps1

$ErrorActionPreference = 'Stop'
$failures = @()

function Check($label, $result) {
    if ($result) {
        Write-Host "  [OK]  $label" -ForegroundColor Green
    } else {
        Write-Host "  [FAIL] $label" -ForegroundColor Red
        $script:failures += $label
    }
}

Write-Host "`nPRO 3.0.0 Release Verification`n" -ForegroundColor Cyan

# 1. Go tests
Write-Host "Running Go tests..." -ForegroundColor Yellow
try {
    $testOut = go test ./... 2>&1
    $testFailed = $testOut | Where-Object { $_ -match "^FAIL" }
    Check "Go test suite passes" ($testFailed.Count -eq 0)
} catch {
    Check "Go test suite passes" $false
}

# 2. Build
try {
    go build ./... 2>&1 | Out-Null
    Check "Go build succeeds" $true
} catch {
    Check "Go build succeeds" $false
}

# 3. Version checks
$proVer = & go run ./cmd/pro --version --quiet 2>&1
Check "pro --version contains 3.0.0" ($proVer -match '3\.0\.0')

$proGitVer = & go run ./cmd/pro --git-version 2>&1
Check "pro --git-version returns 3.0.0" ($proGitVer -eq '3.0.0')

$serveVer = & go run ./cmd/serve --version --quiet 2>&1
Check "serve --version contains 3.0.0" ($serveVer -match '3\.0\.0')

# 4. Functional checks
try {
    & go run ./cmd/pro ls 2>&1 | Out-Null
    Check "pro ls runs" $true
} catch {
    Check "pro ls runs" $false
}

try {
    & go run ./cmd/pro dashboard --quiet 2>&1 | Out-Null
    Check "pro dashboard runs" $true
} catch {
    Check "pro dashboard runs" $false
}

try {
    $initOut = & go run ./cmd/pro init powershell 2>&1
    Check "pro init powershell produces output" ($initOut.Length -gt 5)
} catch {
    Check "pro init powershell produces output" $false
}

try {
    & go run ./cmd/pro git 2>&1 | Out-Null
    Check "pro git runs" $true
} catch {
    Check "pro git runs" $false
}

try {
    & go run ./cmd/pro report 2>&1 | Out-Null
    Check "pro report runs" $true
} catch {
    Check "pro report runs" $false
}

try {
    & go run ./cmd/pro gc --dry-run 2>&1 | Out-Null
    Check "pro gc --dry-run runs" $true
} catch {
    Check "pro gc --dry-run runs" $false
}

try {
    & go run ./cmd/pro workspace ls 2>&1 | Out-Null
    Check "pro workspace ls runs" $true
} catch {
    Check "pro workspace ls runs" $false
}

# Summary
Write-Host ""
if ($failures.Count -eq 0) {
    Write-Host "All checks passed. PRO 3.0.0 is ready." -ForegroundColor Green
} else {
    Write-Host "FAILED checks ($($failures.Count)):" -ForegroundColor Red
    $failures | ForEach-Object { Write-Host "  - $_" -ForegroundColor Red }
    exit 1
}
