[CmdletBinding()]
param(
    [string]$BaseUrl = "http://localhost:3000",
    [string]$AdminEmail = "metabase.admin@dx-os.local",
    [string]$CredentialsPath = "data\runtime\metabase-admin.txt"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repositoryRoot = [IO.Path]::GetFullPath((Split-Path -Parent $PSScriptRoot))
$runtimeRoot = [IO.Path]::GetFullPath((Join-Path $repositoryRoot "data\runtime"))
$resolvedCredentialsPath = if ([IO.Path]::IsPathRooted($CredentialsPath)) {
    [IO.Path]::GetFullPath($CredentialsPath)
}
else {
    [IO.Path]::GetFullPath((Join-Path $repositoryRoot $CredentialsPath))
}
$runtimePrefix = $runtimeRoot.TrimEnd("\") + "\"
if (-not $resolvedCredentialsPath.StartsWith($runtimePrefix, [StringComparison]::OrdinalIgnoreCase)) {
    throw "CredentialsPath must stay inside data\runtime."
}

function New-RandomPassword {
    $bytes = New-Object byte[] 30
    $generator = [Security.Cryptography.RandomNumberGenerator]::Create()
    try {
        $generator.GetBytes($bytes)
    }
    finally {
        $generator.Dispose()
    }
    return ([Convert]::ToBase64String($bytes)).TrimEnd("=").Replace("+", "-").Replace("/", "_") + "!9a"
}

function Get-DotEnvValue {
    param([Parameter(Mandatory)][string]$Name)

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

function Invoke-Metabase {
    param(
        [Parameter(Mandatory)][ValidateSet("GET", "POST", "PUT", "DELETE")][string]$Method,
        [Parameter(Mandatory)][string]$Path,
        [object]$Body,
        [string]$SessionId
    )

    $parameters = @{
        Method = $Method
        Uri = "$BaseUrl$Path"
        TimeoutSec = 30
    }
    if (-not [string]::IsNullOrWhiteSpace($SessionId)) {
        $parameters.Headers = @{ "X-Metabase-Session" = $SessionId }
    }
    if ($null -ne $Body) {
        $parameters.ContentType = "application/json; charset=utf-8"
        $json = $Body | ConvertTo-Json -Depth 30 -Compress
        $parameters.Body = [Text.Encoding]::UTF8.GetBytes($json)
    }
    try {
        return Invoke-RestMethod @parameters
    }
    catch {
        $detail = $_.ErrorDetails.Message
        if ([string]::IsNullOrWhiteSpace($detail)) {
            $detail = $_.Exception.Message
        }
        throw "Metabase $Method $Path failed: $detail"
    }
}

function Get-Items {
    param([object]$Response)

    if ($null -eq $Response) {
        return @()
    }
    if ($Response.PSObject.Properties.Name -contains "data") {
        return @($Response.data)
    }
    return @($Response)
}

function New-TemplateTags {
    return [ordered]@{
        from_date = [ordered]@{
            id = "93b6ad84-20d7-4aad-b1fc-3ef4928121b2"
            name = "from_date"
            "display-name" = "Từ ngày"
            type = "date"
            required = $false
        }
        to_date = [ordered]@{
            id = "12bb9002-c13a-4d33-8db4-2943586fab0a"
            name = "to_date"
            "display-name" = "Đến ngày"
            type = "date"
            required = $false
        }
        currency = [ordered]@{
            id = "d7018b49-c373-4460-afde-7eae8f620554"
            name = "currency"
            "display-name" = "Tiền tệ"
            type = "text"
            required = $false
        }
    }
}

function Ensure-Card {
    param(
        [Parameter(Mandatory)][string]$Name,
        [Parameter(Mandatory)][string]$Description,
        [Parameter(Mandatory)][string]$Query,
        [Parameter(Mandatory)][string]$Display,
        [Parameter(Mandatory)][int]$DatabaseId,
        [Parameter(Mandatory)][int]$CollectionId,
        [Parameter(Mandatory)][string]$SessionId,
        [Parameter(Mandatory)][array]$ExistingCards,
        [System.Collections.IDictionary]$VisualizationSettings = @{}
    )

    $matches = @($ExistingCards | Where-Object {
        $queryMatches = $_.PSObject.Properties.Name -contains "dataset_query" -and
            $null -ne $_.dataset_query -and
            $_.dataset_query.PSObject.Properties.Name -contains "native" -and
            $null -ne $_.dataset_query.native -and
            [string]$_.dataset_query.native.query -eq $Query
        [int]$_.collection_id -eq $CollectionId -and ($_.name -eq $Name -or $queryMatches)
    } | Sort-Object { [int]$_.id })

    $payload = [ordered]@{
        name = $Name
        description = $Description
        collection_id = $CollectionId
        display = $Display
        visualization_settings = $VisualizationSettings
        dataset_query = [ordered]@{
            database = $DatabaseId
            type = "native"
            native = [ordered]@{
                query = $Query
                "template-tags" = New-TemplateTags
            }
        }
    }
    if ($matches.Count -eq 0) {
        return Invoke-Metabase -Method POST -Path "/api/card" -SessionId $SessionId -Body $payload
    }

    $existing = Invoke-Metabase -Method PUT -Path "/api/card/$($matches[0].id)" `
        -SessionId $SessionId -Body $payload
    foreach ($duplicate in @($matches | Select-Object -Skip 1)) {
        $null = Invoke-Metabase -Method PUT -Path "/api/card/$($duplicate.id)" `
            -SessionId $SessionId -Body @{ archived = $true }
    }
    return $existing
}

$health = Invoke-Metabase -Method GET -Path "/api/health"
if ($health.status -ne "ok") {
    throw "Metabase is not healthy."
}

$properties = Invoke-Metabase -Method GET -Path "/api/session/properties"
$adminPassword = $null
$sessionId = $null
if (-not [bool]$properties.'has-user-setup') {
    $adminPassword = New-RandomPassword
    $setup = Invoke-Metabase -Method POST -Path "/api/setup" -Body ([ordered]@{
        token = [string]$properties.'setup-token'
        user = [ordered]@{
            email = $AdminEmail
            first_name = "DX-OS"
            last_name = "Admin"
            password = $adminPassword
        }
        prefs = [ordered]@{
            site_name = "DX-OS Analytics"
            site_locale = "vi"
        }
    })
    $sessionId = [string]$setup.id
    if ([string]::IsNullOrWhiteSpace($sessionId)) {
        throw "Metabase setup did not return a session ID."
    }
    New-Item -ItemType Directory -Path (Split-Path -Parent $resolvedCredentialsPath) -Force | Out-Null
    @(
        "DX-OS local Metabase administrator"
        "email=$AdminEmail"
        "password=$adminPassword"
        "Local development only. Do not share or commit this file."
    ) | Set-Content -LiteralPath $resolvedCredentialsPath -Encoding UTF8
    Write-Host "Metabase administrator was created."
}
else {
    if (-not (Test-Path -LiteralPath $resolvedCredentialsPath)) {
        throw "Metabase is already configured but $resolvedCredentialsPath is missing. Sign in manually or restore the local credential file."
    }
    $AdminEmail = Get-CredentialValue -Path $resolvedCredentialsPath -Name "email"
    $adminPassword = Get-CredentialValue -Path $resolvedCredentialsPath -Name "password"
    $session = Invoke-Metabase -Method POST -Path "/api/session" -Body @{
        username = $AdminEmail
        password = $adminPassword
    }
    $sessionId = [string]$session.id
}

$reportingPassword = Get-DotEnvValue -Name "REPORTING_DB_PASSWORD"
$reportingUser = Get-DotEnvValue -Name "REPORTING_DB_USER"
$dxosDatabase = Get-DotEnvValue -Name "DXOS_DB"
$databases = Get-Items (Invoke-Metabase -Method GET -Path "/api/database" -SessionId $sessionId)
$database = $databases | Where-Object { $_.name -eq "DX-OS Reporting" } | Select-Object -First 1
if (-not $database) {
    $database = Invoke-Metabase -Method POST -Path "/api/database" -SessionId $sessionId -Body ([ordered]@{
        name = "DX-OS Reporting"
        engine = "postgres"
        is_full_sync = $true
        is_on_demand = $false
        details = [ordered]@{
            host = "postgres"
            port = 5432
            dbname = $dxosDatabase
            user = $reportingUser
            password = $reportingPassword
            ssl = $false
            "tunnel-enabled" = $false
            "schema-filters-type" = "inclusion"
            "schema-filters-patterns" = "reporting"
        }
    })
    Write-Host "Read-only DX-OS reporting database was connected."
}
$databaseId = [int]$database.id
$reportingPassword = $null

$null = Invoke-Metabase -Method POST -Path "/api/database/$databaseId/sync_schema" -SessionId $sessionId

$collections = Get-Items (Invoke-Metabase -Method GET -Path "/api/collection" -SessionId $sessionId)
$collection = $collections | Where-Object { $_.name -eq "DX-OS Procurement" } | Select-Object -First 1
if (-not $collection) {
    $collection = Invoke-Metabase -Method POST -Path "/api/collection" -SessionId $sessionId -Body @{
        name = "DX-OS Procurement"
        description = "Báo cáo mua sắm, SLA, tài liệu và ngân sách từ curated views."
    }
}
$collectionId = [int]$collection.id
$collection = Invoke-Metabase -Method PUT -Path "/api/collection/$collectionId" `
    -SessionId $sessionId -Body @{
        name = "DX-OS Procurement"
        description = "Báo cáo mua sắm, SLA, tài liệu và ngân sách từ curated views."
    }

$factFilter = @'
WHERE created_date >= current_date - interval '29 days'
[[AND created_date >= {{from_date}}]]
[[AND created_date <= {{to_date}}]]
[[AND currency = upper({{currency}})]]
'@
$budgetFilter = @'
WHERE period_status = 'ACTIVE'
[[AND period_end >= {{from_date}}]]
[[AND period_start <= {{to_date}}]]
[[AND currency = upper({{currency}})]]
'@

$existingCards = Get-Items (Invoke-Metabase -Method GET -Path "/api/card?f=all" -SessionId $sessionId)
$cardDefinitions = @(
    @{
        Name = "DX-OS - Tổng số phiếu"
        Description = "Số phiếu được tạo trong kỳ báo cáo."
        Display = "scalar"
        Query = "SELECT count(*) AS total_requests FROM reporting.purchase_request_facts`n$factFilter"
        VisualizationSettings = @{
            "scalar.field" = "total_requests"
            column_settings = @{
                '["name","total_requests"]' = @{ decimals = 0; suffix = " phiếu" }
            }
        }
    },
    @{
        Name = "DX-OS - Tỷ lệ phê duyệt"
        Description = "Tỷ lệ phiếu ở trạng thái APPROVED."
        Display = "scalar"
        Query = "SELECT CASE WHEN count(*) = 0 THEN 0 ELSE round(count(*) FILTER (WHERE status = 'APPROVED')::numeric / count(*) * 100, 2) END AS approval_rate_percent FROM reporting.purchase_request_facts`n$factFilter"
        VisualizationSettings = @{
            "scalar.field" = "approval_rate_percent"
            column_settings = @{
                '["name","approval_rate_percent"]' = @{ decimals = 2; suffix = "%" }
            }
        }
    },
    @{
        Name = "DX-OS - Lead time trung bình"
        Description = "Số giờ xử lý trung bình của phiếu đã hoàn tất."
        Display = "scalar"
        Query = "SELECT COALESCE(round(avg(lead_time_hours), 2), 0) AS average_lead_time_hours FROM reporting.purchase_request_facts`n$factFilter"
        VisualizationSettings = @{
            "scalar.field" = "average_lead_time_hours"
            column_settings = @{
                '["name","average_lead_time_hours"]' = @{ decimals = 2; suffix = " giờ" }
            }
        }
    },
    @{
        Name = "DX-OS - Phiếu quá SLA"
        Description = "Số phiếu vượt ngưỡng SLA."
        Display = "scalar"
        Query = "SELECT count(*) FILTER (WHERE sla_breached) AS sla_breached_requests FROM reporting.purchase_request_facts`n$factFilter"
        VisualizationSettings = @{
            "scalar.field" = "sla_breached_requests"
            column_settings = @{
                '["name","sla_breached_requests"]' = @{ decimals = 0; suffix = " phiếu" }
            }
        }
    },
    @{
        Name = "DX-OS - Phân bố trạng thái"
        Description = "Số phiếu theo trạng thái workflow."
        Display = "bar"
        Query = "SELECT status, count(*) AS request_count FROM reporting.purchase_request_facts`n$factFilter`nGROUP BY status ORDER BY request_count DESC"
    },
    @{
        Name = "DX-OS - Xu hướng theo ngày"
        Description = "Số phiếu tạo mới và phê duyệt theo ngày."
        Display = "line"
        Query = "SELECT created_date, count(*) AS request_count, count(*) FILTER (WHERE status = 'APPROVED') AS approved_count FROM reporting.purchase_request_facts`n$factFilter`nGROUP BY created_date ORDER BY created_date"
    },
    @{
        Name = "DX-OS - Giá trị theo phòng ban"
        Description = "Giá trị mua sắm theo phòng ban và tiền tệ."
        Display = "bar"
        Query = "SELECT department_name, currency, sum(total_amount) AS total_amount FROM reporting.purchase_request_facts`n$factFilter`nGROUP BY department_name, currency ORDER BY total_amount DESC"
    },
    @{
        Name = "DX-OS - Sử dụng ngân sách"
        Description = "Allocation, reserved, committed và tỷ lệ sử dụng ngân sách đang hoạt động."
        Display = "table"
        Query = "SELECT period_code, cost_center, currency, allocated_amount, reserved_amount, committed_amount, available_amount, utilization_percent FROM reporting.budget_utilization`n$budgetFilter`nORDER BY utilization_percent DESC, cost_center"
    }
)

$cards = @()
foreach ($definition in $cardDefinitions) {
    $visualizationSettings = if ($definition.ContainsKey("VisualizationSettings")) {
        $definition.VisualizationSettings
    }
    else {
        @{}
    }
    $cards += Ensure-Card -Name $definition.Name -Description $definition.Description `
        -Query $definition.Query -Display $definition.Display -DatabaseId $databaseId `
        -CollectionId $collectionId -SessionId $sessionId -ExistingCards $existingCards `
        -VisualizationSettings $visualizationSettings
}

$dashboards = Get-Items (Invoke-Metabase -Method GET -Path "/api/dashboard?f=all" -SessionId $sessionId)
$dashboard = $dashboards | Where-Object { $_.name -eq "DX-OS - Procurement Overview" } | Select-Object -First 1
if (-not $dashboard) {
    $dashboard = Invoke-Metabase -Method POST -Path "/api/dashboard" -SessionId $sessionId -Body @{
        name = "DX-OS - Procurement Overview"
        description = "Dashboard vận hành mua sắm từ schema reporting chỉ đọc."
        collection_id = $collectionId
        parameters = @()
    }
}
$dashboardId = [int]$dashboard.id
$dashboardDetail = Invoke-Metabase -Method GET -Path "/api/dashboard/$dashboardId" -SessionId $sessionId
$dashboardParameters = @(
    @{ id = "from_date"; name = "Từ ngày"; slug = "from_date"; type = "date/single"; sectionId = "date" },
    @{ id = "to_date"; name = "Đến ngày"; slug = "to_date"; type = "date/single"; sectionId = "date" },
    @{ id = "currency"; name = "Tiền tệ"; slug = "currency"; type = "string/="; sectionId = "string" }
)
$expectedCardIds = @($cards | ForEach-Object { [int]$_.id })
$currentCardIds = @($dashboardDetail.dashcards | ForEach-Object { [int]$_.card_id })
$needsCardRefresh = $currentCardIds.Count -ne $expectedCardIds.Count -or
    @(Compare-Object $currentCardIds $expectedCardIds).Count -gt 0
if ($needsCardRefresh) {
    $staleCardIds = @($currentCardIds | Where-Object { $_ -notin $expectedCardIds })
    $dashcards = @()
    for ($index = 0; $index -lt $cards.Count; $index++) {
        $card = $cards[$index]
        $isKpi = $index -lt 4
        $row = if ($isKpi) { 0 } else { 4 + [math]::Floor(($index - 4) / 2) * 6 }
        $col = if ($isKpi) { $index * 6 } else { (($index - 4) % 2) * 12 }
        $sizeX = if ($isKpi) { 6 } else { 12 }
        $sizeY = if ($isKpi) { 4 } else { 6 }
        $mappings = @(
            @{ parameter_id = "from_date"; card_id = [int]$card.id; target = @("variable", @("template-tag", "from_date")) },
            @{ parameter_id = "to_date"; card_id = [int]$card.id; target = @("variable", @("template-tag", "to_date")) },
            @{ parameter_id = "currency"; card_id = [int]$card.id; target = @("variable", @("template-tag", "currency")) }
        )
        $dashcards += [ordered]@{
            id = -($index + 1)
            card_id = [int]$card.id
            row = [int]$row
            col = [int]$col
            size_x = $sizeX
            size_y = $sizeY
            parameter_mappings = $mappings
            visualization_settings = @{}
            series = @()
        }
    }
    $null = Invoke-Metabase -Method PUT -Path "/api/dashboard/$dashboardId" `
        -SessionId $sessionId -Body @{
            name = "DX-OS - Procurement Overview"
            description = "Dashboard vận hành mua sắm từ schema reporting chỉ đọc."
            collection_id = $collectionId
            parameters = $dashboardParameters
            dashcards = $dashcards
            width = "full"
        }
    foreach ($staleCardId in $staleCardIds) {
        $staleCard = $existingCards | Where-Object {
            [int]$_.id -eq $staleCardId -and
            [int]$_.collection_id -eq $collectionId -and
            [string]$_.name -like "DX-OS -*"
        } | Select-Object -First 1
        if ($staleCard) {
            $null = Invoke-Metabase -Method PUT -Path "/api/card/$staleCardId" `
                -SessionId $sessionId -Body @{ archived = $true }
        }
    }
}
else {
    $null = Invoke-Metabase -Method PUT -Path "/api/dashboard/$dashboardId" `
        -SessionId $sessionId -Body @{
            name = "DX-OS - Procurement Overview"
            description = "Dashboard vận hành mua sắm từ schema reporting chỉ đọc."
            collection_id = $collectionId
            parameters = $dashboardParameters
            width = "full"
        }
}

$finalDashboard = Invoke-Metabase -Method GET -Path "/api/dashboard/$dashboardId" -SessionId $sessionId
if (@($finalDashboard.dashcards).Count -ne $cards.Count) {
    throw "Metabase dashboard card count does not match the expected fixture."
}

$adminPassword = $null
$sessionId = $null
Write-Host "Metabase provisioning passed."
Write-Host "Database: DX-OS Reporting (read-only reporting schema)"
Write-Host "Dashboard: DX-OS - Procurement Overview ($($cards.Count) cards)"
Write-Host "Local administrator credentials: $resolvedCredentialsPath"
