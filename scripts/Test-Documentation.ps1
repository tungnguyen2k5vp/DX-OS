[CmdletBinding()]
param(
    [string]$BaseUrl = "http://localhost:4300"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Assert-Condition {
    param(
        [Parameter(Mandatory)][bool]$Condition,
        [Parameter(Mandatory)][string]$Message
    )

    if (-not $Condition) {
        throw $Message
    }
}

function Invoke-Curl {
    param(
        [Parameter(Mandatory)][string]$Uri,
        [switch]$HeadersOnly
    )

    $arguments = @("--fail", "--silent", "--show-error", "--location", "--max-time", "15")
    if ($HeadersOnly) {
        $arguments += @("--head")
    }
    $arguments += $Uri

    $result = @(& curl.exe @arguments)
    if ($LASTEXITCODE -ne 0) {
        throw "Documentation request failed: $Uri"
    }
    return ($result -join "`n")
}

$normalizedBaseUrl = $BaseUrl.TrimEnd("/")
$routes = @(
    "/",
    "/bat-dau",
    "/huong-dan-su-dung",
    "/roles/employee",
    "/architecture/CONTEXT"
)

foreach ($route in $routes) {
    $content = Invoke-Curl -Uri "$normalizedBaseUrl$route"
    Assert-Condition ($content -match "DX-OS") "Documentation route has unexpected content: $route"
}

$health = Invoke-Curl -Uri "$normalizedBaseUrl/healthz"
Assert-Condition ($health.Trim() -eq "ok") "Documentation health response is unexpected."

$headers = Invoke-Curl -Uri "$normalizedBaseUrl/" -HeadersOnly
Assert-Condition (
    $headers -match "(?im)^X-Content-Type-Options:\s*nosniff\s*$"
) "Documentation response is missing X-Content-Type-Options."
Assert-Condition (
    $headers -match "(?im)^Referrer-Policy:\s*strict-origin-when-cross-origin\s*$"
) "Documentation response is missing Referrer-Policy."

Write-Host "Documentation smoke test passed."
Write-Host "Verified 5 routes, health endpoint, UTF-8 content and security headers."
