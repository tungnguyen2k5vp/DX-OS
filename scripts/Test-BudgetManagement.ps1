[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repositoryRoot = [IO.Path]::GetFullPath((Split-Path -Parent $PSScriptRoot))
$runtimeRoot = Join-Path $repositoryRoot "data\runtime"
$roles = @("finance", "auditor", "employee")
$tokens = @{}

function Invoke-DxApi {
    param(
        [Parameter(Mandatory)][ValidateSet("GET", "PATCH")][string]$Method,
        [Parameter(Mandatory)][string]$Path,
        [Parameter(Mandatory)][string]$Token,
        [object]$Body,
        [string]$IdempotencyKey
    )

    $parameters = @{
        Method  = $Method
        Uri     = "http://localhost:8081$Path"
        Headers = @{ Authorization = "Bearer $Token" }
    }
    if ($null -ne $Body) {
        $parameters.ContentType = "application/json"
        $parameters.Body = $Body | ConvertTo-Json -Depth 8 -Compress
    }
    if (-not [string]::IsNullOrWhiteSpace($IdempotencyKey)) {
        $parameters.Headers["Idempotency-Key"] = $IdempotencyKey
    }
    return Invoke-RestMethod @parameters
}

function Assert-Equal {
    param(
        [Parameter(Mandatory)]$Actual,
        [Parameter(Mandatory)]$Expected,
        [Parameter(Mandatory)][string]$Message
    )
    if ($Actual -ne $Expected) {
        throw "$Message Expected '$Expected', received '$Actual'."
    }
}

New-Item -ItemType Directory -Path $runtimeRoot -Force | Out-Null

Push-Location $repositoryRoot
try {
    foreach ($role in $roles) {
        $username = "budget.$role"
        $credentialPath = "data\runtime\budget-$role.txt"
        $tokenPath = "data\runtime\budget-$role.token"
        & "$PSScriptRoot\Initialize-DevUser.ps1" `
            -Username $username `
            -Role $role `
            -CredentialsPath $credentialPath
        & "$PSScriptRoot\Test-OIDCLogin.ps1" `
            -CredentialsPath $credentialPath `
            -ExpectedRole $role `
            -AccessTokenOutputPath $tokenPath
        $tokens[$role] = [IO.File]::ReadAllText((Join-Path $repositoryRoot $tokenPath))
    }

    $financeDashboard = Invoke-DxApi -Method GET -Path "/api/v1/budgets/dashboard" `
        -Token $tokens["finance"]
    Assert-Equal $financeDashboard.canManage $true "Finance must receive write capability."
    if (@($financeDashboard.allocations).Count -lt 1) {
        throw "Finance dashboard did not return the seeded allocation."
    }

    $auditorDashboard = Invoke-DxApi -Method GET -Path "/api/v1/budgets/dashboard" `
        -Token $tokens["auditor"]
    Assert-Equal $auditorDashboard.canManage $false "Auditor must be read-only."

    try {
        $null = Invoke-DxApi -Method GET -Path "/api/v1/budgets/dashboard" `
            -Token $tokens["employee"]
        throw "Employee unexpectedly accessed the budget dashboard."
    }
    catch {
        Assert-Equal ([int]$_.Exception.Response.StatusCode) 403 `
            "Employee budget dashboard access must return HTTP 403."
    }

    $allocation = @($financeDashboard.allocations)[0]
    $originalAmount = [decimal]$allocation.allocatedAmount
    $raisedAmount = $originalAmount + [decimal]1000
    $raisedText = $raisedAmount.ToString("0.0000", [Globalization.CultureInfo]::InvariantCulture)
    $adjustmentKey = "budget-adjust-$([Guid]::NewGuid().ToString('N'))"
    $adjustmentBody = @{
        allocatedAmount = $raisedText
        expectedVersion = [int64]$allocation.version
        reason          = "Automated finance adjustment verification."
    }
    $adjusted = Invoke-DxApi -Method PATCH `
        -Path "/api/v1/budgets/allocations/$($allocation.id)" `
        -Token $tokens["finance"] `
        -IdempotencyKey $adjustmentKey `
        -Body $adjustmentBody
    Assert-Equal ([decimal]$adjusted.allocatedAmount) $raisedAmount `
        "Finance adjustment must update the allocation."
    Assert-Equal ([int64]$adjusted.version) ([int64]$allocation.version + 1) `
        "Finance adjustment must increment the version."

    $replayed = Invoke-DxApi -Method PATCH `
        -Path "/api/v1/budgets/allocations/$($allocation.id)" `
        -Token $tokens["finance"] `
        -IdempotencyKey $adjustmentKey `
        -Body $adjustmentBody
    Assert-Equal ([int64]$replayed.version) ([int64]$adjusted.version) `
        "Idempotent replay must not increment the allocation version."

    try {
        $null = Invoke-DxApi -Method PATCH `
            -Path "/api/v1/budgets/allocations/$($allocation.id)" `
            -Token $tokens["auditor"] `
            -IdempotencyKey "auditor-adjust-$([Guid]::NewGuid().ToString('N'))" `
            -Body @{
                allocatedAmount = $raisedText
                expectedVersion = [int64]$adjusted.version
                reason          = "Auditor must not change allocations."
            }
        throw "Auditor unexpectedly adjusted a budget."
    }
    catch {
        Assert-Equal ([int]$_.Exception.Response.StatusCode) 403 `
            "Auditor budget adjustment must return HTTP 403."
    }

    try {
        $null = Invoke-DxApi -Method PATCH `
            -Path "/api/v1/budgets/allocations/$($allocation.id)" `
            -Token $tokens["finance"] `
            -IdempotencyKey "stale-adjust-$([Guid]::NewGuid().ToString('N'))" `
            -Body @{
                allocatedAmount = $raisedText
                expectedVersion = [int64]$allocation.version
                reason          = "Stale version must be rejected safely."
            }
        throw "A stale budget adjustment unexpectedly succeeded."
    }
    catch {
        Assert-Equal ([int]$_.Exception.Response.StatusCode) 409 `
            "Stale budget adjustment must return HTTP 409."
    }

    $restored = Invoke-DxApi -Method PATCH `
        -Path "/api/v1/budgets/allocations/$($allocation.id)" `
        -Token $tokens["finance"] `
        -IdempotencyKey "restore-adjust-$([Guid]::NewGuid().ToString('N'))" `
        -Body @{
            allocatedAmount = $originalAmount.ToString(
                "0.0000",
                [Globalization.CultureInfo]::InvariantCulture
            )
            expectedVersion = [int64]$adjusted.version
            reason          = "Restore allocation after automated verification."
        }
    Assert-Equal ([decimal]$restored.allocatedAmount) $originalAmount `
        "Smoke test must restore the original allocation."

    $finalDashboard = Invoke-DxApi -Method GET -Path "/api/v1/budgets/dashboard" `
        -Token $tokens["finance"]
    if (@($finalDashboard.adjustments).Count -lt 2) {
        throw "Budget dashboard did not return adjustment history."
    }

    Write-Host "Budget management smoke test passed."
    Write-Host "Finance write, auditor read-only, employee denial, idempotency, version conflict, history, and restore checks passed."
}
finally {
    foreach ($role in $roles) {
        $tokens[$role] = $null
        $tokenPath = Join-Path $runtimeRoot "budget-$role.token"
        if (Test-Path -LiteralPath $tokenPath) {
            Remove-Item -LiteralPath $tokenPath -Force
        }
    }
    $tokens.Clear()
    Pop-Location
}
