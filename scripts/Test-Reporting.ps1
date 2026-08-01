[CmdletBinding()]
param(
    [switch]$SkipMetabase
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repositoryRoot = [IO.Path]::GetFullPath((Split-Path -Parent $PSScriptRoot))
$runtimeRoot = Join-Path $repositoryRoot "data\runtime"
$roles = @("finance", "auditor", "employee")
$tokens = @{}

function Invoke-DxApi {
    param(
        [Parameter(Mandatory)][ValidateSet("GET", "POST")][string]$Method,
        [Parameter(Mandatory)][string]$Path,
        [Parameter(Mandatory)][string]$Token,
        [object]$Body
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

function Get-DotEnvValue {
    param(
        [Parameter(Mandatory)][string]$Name
    )

    $line = Get-Content -LiteralPath (Join-Path $repositoryRoot ".env") |
        Where-Object { $_ -match "^\s*$([Regex]::Escape($Name))=" } |
        Select-Object -Last 1
    if ([string]::IsNullOrWhiteSpace($line)) {
        throw "Missing $Name in .env."
    }
    return (($line -split "=", 2)[1]).Trim().Trim('"').Trim("'")
}

function Get-CredentialValue {
    param(
        [Parameter(Mandatory)][string]$Path,
        [Parameter(Mandatory)][string]$Name
    )

    $line = Get-Content -LiteralPath $Path |
        Where-Object { $_ -match "^$([Regex]::Escape($Name))=" } |
        Select-Object -Last 1
    if ([string]::IsNullOrWhiteSpace($line)) {
        throw "Missing $Name in $Path."
    }
    return ($line -split "=", 2)[1]
}

function Get-ResponseItems {
    param([object]$Response)

    if ($null -eq $Response) {
        return @()
    }
    if ($Response.PSObject.Properties.Name -contains "data") {
        return @($Response.data)
    }
    return @($Response)
}

New-Item -ItemType Directory -Path $runtimeRoot -Force | Out-Null

Push-Location $repositoryRoot
try {
    foreach ($role in $roles) {
        $username = "reporting.$role"
        $credentialPath = "data\runtime\reporting-$role.txt"
        $tokenPath = "data\runtime\reporting-$role.token"
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

    $fixture = Invoke-DxApi -Method POST -Path "/api/v1/purchase-requests" `
        -Token $tokens["employee"] `
        -Body @{
            title      = "Reporting reconciliation $([DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds())"
            reason     = "Create a deterministic record for the reporting reconciliation smoke test."
            currency   = "VND"
            costCenter = "CC-GENERAL"
            items      = @(@{
                description = "Reporting verification fixture"
                quantity    = "1"
                unit        = "unit"
                unitPrice   = "1000"
            })
        }
    if ([string]::IsNullOrWhiteSpace([string]$fixture.id)) {
        throw "Reporting fixture was not created."
    }

    $from = [DateTime]::UtcNow.Date.AddDays(-1).ToString("yyyy-MM-dd")
    $to = [DateTime]::UtcNow.Date.AddDays(1).ToString("yyyy-MM-dd")
    $reportPath = "/api/v1/reports/procurement?from=$from&to=$to&currency=VND"

    $financeReport = Invoke-DxApi -Method GET -Path $reportPath -Token $tokens["finance"]
    $auditorReport = Invoke-DxApi -Method GET -Path $reportPath -Token $tokens["auditor"]

    Assert-Equal $financeReport.filters.from $from "Finance report must echo the from filter."
    Assert-Equal $financeReport.filters.to $to "Finance report must echo the to filter."
    Assert-Equal $financeReport.filters.currency "VND" "Finance report must echo the currency filter."
    if ([int64]$financeReport.summary.totalRequests -lt 1) {
        throw "Finance report did not include the reconciliation fixture."
    }
    if ([int64]$auditorReport.summary.totalRequests -lt 1) {
        throw "Auditor report did not include the reconciliation fixture."
    }

    try {
        $null = Invoke-DxApi -Method GET -Path $reportPath -Token $tokens["employee"]
        throw "Employee unexpectedly accessed the reporting endpoint."
    }
    catch {
        Assert-Equal ([int]$_.Exception.Response.StatusCode) 403 `
            "Employee reporting access must return HTTP 403."
    }

    $reportingUser = Get-DotEnvValue -Name "REPORTING_DB_USER"
    $reportingPassword = Get-DotEnvValue -Name "REPORTING_DB_PASSWORD"
    $dxosDatabase = Get-DotEnvValue -Name "DXOS_DB"
    $countSql = "SELECT count(*) FROM reporting.purchase_request_facts WHERE created_date BETWEEN DATE '$from' AND DATE '$to' AND currency = 'VND';"
    $countOutput = & docker compose --profile foundation --profile application --profile reporting `
        run --rm --no-deps -e "PGPASSWORD=$reportingPassword" `
        --entrypoint psql reporting-bootstrap -qAt -h postgres -U $reportingUser `
        -d $dxosDatabase -v ON_ERROR_STOP=1 -c $countSql
    if ($LASTEXITCODE -ne 0) {
        throw "Read-only reporting role could not query the curated view."
    }
    $curatedCount = [int64](($countOutput | Select-Object -Last 1).Trim())
    Assert-Equal ([int64]$auditorReport.summary.totalRequests) $curatedCount `
        "Auditor API total must match the curated view."

    $permissionSql = "SELECT has_schema_privilege(current_user, 'reporting', 'CREATE')::text || '|' || current_setting('default_transaction_read_only');"
    $permissionOutput = & docker compose --profile foundation --profile application --profile reporting `
        run --rm --no-deps -e "PGPASSWORD=$reportingPassword" `
        --entrypoint psql reporting-bootstrap -qAt -h postgres -U $reportingUser `
        -d $dxosDatabase -v ON_ERROR_STOP=1 -c $permissionSql
    if ($LASTEXITCODE -ne 0) {
        throw "Could not inspect the read-only reporting role."
    }
    Assert-Equal (($permissionOutput | Select-Object -Last 1).Trim()) "false|on" `
        "Reporting role must have no CREATE privilege and must default to read-only transactions."
    $reportingPassword = $null

    if (-not $SkipMetabase) {
        $metabaseHealthy = $false
        for ($attempt = 1; $attempt -le 24; $attempt++) {
            try {
                $health = Invoke-RestMethod -Method GET -Uri "http://localhost:3000/api/health" `
                    -TimeoutSec 5
                if ($health.status -eq "ok") {
                    $metabaseHealthy = $true
                    break
                }
            }
            catch {
                if ($attempt -eq 24) {
                    throw
                }
            }
            Start-Sleep -Seconds 5
        }
        if (-not $metabaseHealthy) {
            throw "Metabase health endpoint did not become healthy."
        }

        $metabaseCredentials = Join-Path $runtimeRoot "metabase-admin.txt"
        if (-not (Test-Path -LiteralPath $metabaseCredentials)) {
            throw "Metabase is healthy but has not been provisioned. Run scripts\Initialize-Metabase.ps1."
        }
        $metabaseEmail = Get-CredentialValue -Path $metabaseCredentials -Name "email"
        $metabasePassword = Get-CredentialValue -Path $metabaseCredentials -Name "password"
        $metabaseSession = Invoke-RestMethod -Method POST -Uri "http://localhost:3000/api/session" `
            -ContentType "application/json" -Body (@{
                username = $metabaseEmail
                password = $metabasePassword
            } | ConvertTo-Json -Compress)
        $metabaseHeaders = @{ "X-Metabase-Session" = [string]$metabaseSession.id }

        $databaseResponse = Invoke-RestMethod -Headers $metabaseHeaders `
            -Uri "http://localhost:3000/api/database"
        $metabaseDatabase = Get-ResponseItems $databaseResponse |
            Where-Object { $_.name -eq "DX-OS Reporting" } |
            Select-Object -First 1
        if (-not $metabaseDatabase) {
            throw "Metabase does not contain the DX-OS Reporting database."
        }
        $syncableSchemasResponse = Invoke-RestMethod -Headers $metabaseHeaders `
            -Uri "http://localhost:3000/api/database/$($metabaseDatabase.id)/syncable_schemas"
        $syncableSchemas = @($syncableSchemasResponse | ForEach-Object { [string]$_ })
        Assert-Equal ($syncableSchemas -join ",") "reporting" `
            "Metabase data source must expose only the reporting schema."

        $dashboardResponse = Invoke-RestMethod -Headers $metabaseHeaders `
            -Uri "http://localhost:3000/api/dashboard?f=all"
        $metabaseDashboard = Get-ResponseItems $dashboardResponse |
            Where-Object { $_.name -eq "DX-OS - Procurement Overview" } |
            Select-Object -First 1
        if (-not $metabaseDashboard) {
            throw "Metabase Procurement dashboard was not provisioned."
        }
        $dashboardDetail = Invoke-RestMethod -Headers $metabaseHeaders `
            -Uri "http://localhost:3000/api/dashboard/$($metabaseDashboard.id)"
        Assert-Equal @($dashboardDetail.dashcards).Count 8 `
            "Metabase Procurement dashboard must contain eight cards."
        Assert-Equal @($dashboardDetail.parameters).Count 3 `
            "Metabase Procurement dashboard must contain three filters."
        foreach ($dashcard in @($dashboardDetail.dashcards)) {
            $queryResult = Invoke-RestMethod -Method POST -Headers $metabaseHeaders `
                -Uri "http://localhost:3000/api/card/$($dashcard.card_id)/query" `
                -ContentType "application/json" -Body "{}"
            Assert-Equal $queryResult.status "completed" `
                "Metabase card $($dashcard.card_id) must execute successfully."
        }
        $metabasePassword = $null
        $metabaseSession = $null
    }

    Write-Host "Reporting smoke test passed."
    if ($SkipMetabase) {
        Write-Host "Finance/auditor access, employee denial, filters, curated-view reconciliation, and read-only role checks passed. Metabase health was skipped."
    }
    else {
        Write-Host "Finance/auditor access, employee denial, filters, curated-view reconciliation, read-only role, Metabase schema isolation, dashboard, filters, cards, and health checks passed."
    }
}
finally {
    foreach ($role in $roles) {
        $tokens[$role] = $null
        $tokenPath = Join-Path $runtimeRoot "reporting-$role.token"
        if (Test-Path -LiteralPath $tokenPath) {
            Remove-Item -LiteralPath $tokenPath -Force
        }
    }
    $tokens.Clear()
    Pop-Location
}
