[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repositoryRoot = Split-Path -Parent $PSScriptRoot
Push-Location $repositoryRoot
try {
    $live = Invoke-RestMethod -Uri "http://localhost:8081/health/live" -TimeoutSec 10
    if ($live.status -ne "ok") {
        throw "Unexpected API liveness response."
    }

    $ready = Invoke-RestMethod -Uri "http://localhost:8081/health/ready" -TimeoutSec 10
    if ($ready.status -ne "ready") {
        throw "Unexpected API readiness response."
    }

    try {
        Invoke-WebRequest -UseBasicParsing -Uri "http://localhost:8081/api/v1/me" -TimeoutSec 10 | Out-Null
        throw "Protected endpoint unexpectedly accepted a request without a token."
    }
    catch {
        if ($_.Exception.Response.StatusCode.value__ -ne 401) {
            throw
        }
    }

    $web = Invoke-WebRequest -UseBasicParsing -Uri "http://localhost:4200/config.json" -TimeoutSec 10
    if ($web.StatusCode -ne 200) {
        throw "Angular runtime config is unavailable."
    }

    $index = Invoke-WebRequest -UseBasicParsing -Uri "http://localhost:4200/" -TimeoutSec 10
    if ($index.Content -match 'media="print"') {
        throw "Angular stylesheet is deferred through an inline onload handler that the CSP blocks."
    }
    $stylesheetMatch = [regex]::Match(
        $index.Content,
        '<link rel="stylesheet" href="([^"]+\.css)"'
    )
    if (-not $stylesheetMatch.Success) {
        throw "Angular index does not contain a normal stylesheet link."
    }
    $stylesheetUri = "http://localhost:4200/" + $stylesheetMatch.Groups[1].Value.TrimStart("/")
    $stylesheet = Invoke-WebRequest -UseBasicParsing -Uri $stylesheetUri -TimeoutSec 10
    if (
        $stylesheet.StatusCode -ne 200 -or
        $stylesheet.Content -notmatch '\.grid\{display:grid\}' -or
        $stylesheet.Content -notmatch '\.rounded-xl\{'
    ) {
        throw "Angular stylesheet does not contain the expected Tailwind utilities."
    }
    $contentSecurityPolicy = [string]$index.Headers["Content-Security-Policy"]
    if ($contentSecurityPolicy -notmatch "script-src 'self'") {
        throw "Angular response is missing the strict script-src CSP."
    }

    Write-Host "Application smoke test passed."
    Write-Host "Go API: live and ready; /api/v1/me rejects missing tokens."
    Write-Host "Angular: runtime config, CSP-compatible stylesheet and Tailwind utilities are reachable."
}
finally {
    Pop-Location
}
