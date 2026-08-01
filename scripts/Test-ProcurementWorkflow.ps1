[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repositoryRoot = [IO.Path]::GetFullPath((Split-Path -Parent $PSScriptRoot))
$runtimeRoot = Join-Path $repositoryRoot "data\runtime"
$employeeCredentials = "data\runtime\workflow-employee.txt"
$managerCredentials = "data\runtime\workflow-manager.txt"
$financeCredentials = "data\runtime\workflow-finance.txt"
$outsiderCredentials = "data\runtime\workflow-outsider.txt"
$employeeTokenPath = Join-Path $runtimeRoot "workflow-employee.token"
$managerTokenPath = Join-Path $runtimeRoot "workflow-manager.token"
$financeTokenPath = Join-Path $runtimeRoot "workflow-finance.token"
$outsiderTokenPath = Join-Path $runtimeRoot "workflow-outsider.token"

function Invoke-DxApi {
    param(
        [Parameter(Mandatory)][ValidateSet("GET", "POST", "PATCH")][string]$Method,
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

function Add-TestQuotation {
    param(
        [Parameter(Mandatory)][string]$RequestId,
        [Parameter(Mandatory)][string]$Token,
        [Parameter(Mandatory)][string]$Name
    )

    $filePath = Join-Path $runtimeRoot "$Name.pdf"
    $responsePath = Join-Path $runtimeRoot "$Name-upload.json"
    [IO.File]::WriteAllBytes(
        $filePath,
        [Text.Encoding]::ASCII.GetBytes("%PDF-1.4`nDX-OS attachment smoke test`n%%EOF")
    )
    try {
        $status = & curl.exe --silent --show-error `
            --output $responsePath `
            --write-out "%{http_code}" `
            --header "Authorization: Bearer $Token" `
            --form "documentType=QUOTATION" `
            --form "file=@$filePath;type=application/pdf" `
            "http://localhost:8081/api/v1/purchase-requests/$RequestId/attachments"
        if ([int]$status -ne 201) {
            $detail = if (Test-Path -LiteralPath $responsePath) {
                [IO.File]::ReadAllText($responsePath)
            } else {
                ""
            }
            throw "Quotation upload returned HTTP $status. $detail"
        }
        return [IO.File]::ReadAllText($responsePath) | ConvertFrom-Json
    }
    finally {
        foreach ($path in @($filePath, $responsePath)) {
            if (Test-Path -LiteralPath $path) {
                Remove-Item -LiteralPath $path -Force
            }
        }
    }
}

New-Item -ItemType Directory -Path $runtimeRoot -Force | Out-Null

Push-Location $repositoryRoot
try {
    & "$PSScriptRoot\Initialize-DevUser.ps1" `
        -Username "workflow.employee" `
        -Role "employee" `
        -CredentialsPath $employeeCredentials
    & "$PSScriptRoot\Initialize-DevUser.ps1" `
        -Username "workflow.manager" `
        -Role "department_manager" `
        -CredentialsPath $managerCredentials
    & "$PSScriptRoot\Initialize-DevUser.ps1" `
        -Username "workflow.finance" `
        -Role "finance" `
        -CredentialsPath $financeCredentials
    & "$PSScriptRoot\Initialize-DevUser.ps1" `
        -Username "workflow.outsider" `
        -Role "employee" `
        -CredentialsPath $outsiderCredentials

    & "$PSScriptRoot\Test-OIDCLogin.ps1" `
        -CredentialsPath $employeeCredentials `
        -ExpectedRole "employee" `
        -AccessTokenOutputPath "data\runtime\workflow-employee.token"
    & "$PSScriptRoot\Test-OIDCLogin.ps1" `
        -CredentialsPath $managerCredentials `
        -ExpectedRole "department_manager" `
        -AccessTokenOutputPath "data\runtime\workflow-manager.token"
    & "$PSScriptRoot\Test-OIDCLogin.ps1" `
        -CredentialsPath $financeCredentials `
        -ExpectedRole "finance" `
        -AccessTokenOutputPath "data\runtime\workflow-finance.token"
    & "$PSScriptRoot\Test-OIDCLogin.ps1" `
        -CredentialsPath $outsiderCredentials `
        -ExpectedRole "employee" `
        -AccessTokenOutputPath "data\runtime\workflow-outsider.token"

    $employeeToken = [IO.File]::ReadAllText($employeeTokenPath)
    $managerToken = [IO.File]::ReadAllText($managerTokenPath)
    $financeToken = [IO.File]::ReadAllText($financeTokenPath)
    $outsiderToken = [IO.File]::ReadAllText($outsiderTokenPath)

    $budgetBefore = Invoke-DxApi -Method GET `
        -Path "/api/v1/budgets/summary?costCenter=CC-GENERAL&currency=VND" `
        -Token $financeToken
    $reservedBefore = [decimal]$budgetBefore.reservedAmount
    $committedBefore = [decimal]$budgetBefore.committedAmount

    $draftBody = @{
        title      = "Workflow smoke test $([DateTimeOffset]::UtcNow.ToUnixTimeSeconds())"
        reason     = "End-to-end verification of update and two-stage approval."
        currency   = "VND"
        costCenter = "CC-GENERAL"
        items      = @(
            @{
                description = "Development workstation"
                quantity    = "1"
                unit        = "unit"
                unitPrice   = "25000000"
            }
        )
    }
    $draft = Invoke-DxApi -Method POST -Path "/api/v1/purchase-requests" `
        -Token $employeeToken -Body $draftBody
    Assert-Equal $draft.status "DRAFT" "Create must produce a draft."
    Assert-Equal ([int64]$draft.version) 1 "Create must start at version 1."

    $updateBody = @{
        title           = $draftBody.title
        reason          = "Updated end-to-end verification before submitting for approval."
        currency        = "VND"
        costCenter      = "CC-GENERAL"
        expectedVersion = 1
        items           = @(
            @{
                description = "Development workstation"
                quantity    = "1"
                unit        = "unit"
                unitPrice   = "27000000"
            }
        )
    }
    $updated = Invoke-DxApi -Method PATCH -Path "/api/v1/purchase-requests/$($draft.id)" `
        -Token $employeeToken -Body $updateBody
    Assert-Equal ([int64]$updated.version) 2 "Update must increment the version."
    Assert-Equal $updated.totalAmount "27000000.0000" "Update must recalculate the total."

    try {
        $null = Invoke-DxApi -Method PATCH -Path "/api/v1/purchase-requests/$($draft.id)" `
            -Token $employeeToken -Body $updateBody
        throw "A stale update unexpectedly succeeded."
    }
    catch {
        $statusCode = [int]$_.Exception.Response.StatusCode
        Assert-Equal $statusCode 409 "A stale expectedVersion must return HTTP 409."
    }

    $null = Add-TestQuotation `
        -RequestId $draft.id `
        -Token $employeeToken `
        -Name "workflow-main-quotation"

    $submitKey = "submit-$([Guid]::NewGuid().ToString('N'))"
    $submitted = Invoke-DxApi -Method POST `
        -Path "/api/v1/purchase-requests/$($draft.id)/transitions" `
        -Token $employeeToken `
        -IdempotencyKey $submitKey `
        -Body @{ action = "SUBMIT"; expectedVersion = 2; comment = "" }
    Assert-Equal $submitted.status "SUBMITTED" "Employee submit must enter manager review."
    Assert-Equal ([int64]$submitted.version) 3 "Submit must increment the version."

    $replayed = Invoke-DxApi -Method POST `
        -Path "/api/v1/purchase-requests/$($draft.id)/transitions" `
        -Token $employeeToken `
        -IdempotencyKey $submitKey `
        -Body @{ action = "SUBMIT"; expectedVersion = 2; comment = "" }
    Assert-Equal ([int64]$replayed.version) 3 "Idempotent replay must not increment the version."

    $managerApproved = Invoke-DxApi -Method POST `
        -Path "/api/v1/purchase-requests/$($draft.id)/transitions" `
        -Token $managerToken `
        -IdempotencyKey "manager-$([Guid]::NewGuid().ToString('N'))" `
        -Body @{ action = "APPROVE"; expectedVersion = 3; comment = "Department approval." }
    Assert-Equal $managerApproved.status "MANAGER_APPROVED" "Manager approval must enter finance review."
    Assert-Equal ([int64]$managerApproved.version) 4 "Manager approval must increment the version."

    $reservedCheck = Invoke-DxApi -Method GET `
        -Path "/api/v1/purchase-requests/$($draft.id)/budget-check" `
        -Token $managerToken
    Assert-Equal $reservedCheck.result "RESERVED" "Manager approval must reserve budget."
    Assert-Equal $reservedCheck.reservationState "RESERVED" "Reservation state must be RESERVED."
    Assert-Equal `
        ([decimal]$reservedCheck.summary.reservedAmount) `
        ($reservedBefore + [decimal]27000000) `
        "Reserved total must increase by the request amount."

    $approved = Invoke-DxApi -Method POST `
        -Path "/api/v1/purchase-requests/$($draft.id)/transitions" `
        -Token $financeToken `
        -IdempotencyKey "finance-$([Guid]::NewGuid().ToString('N'))" `
        -Body @{ action = "APPROVE"; expectedVersion = 4; comment = "Budget approved." }
    Assert-Equal $approved.status "APPROVED" "Finance approval must complete the workflow."
    Assert-Equal ([int64]$approved.version) 5 "Finance approval must increment the version."

    $committedCheck = Invoke-DxApi -Method GET `
        -Path "/api/v1/purchase-requests/$($draft.id)/budget-check" `
        -Token $financeToken
    Assert-Equal $committedCheck.result "COMMITTED" "Finance approval must commit budget."
    Assert-Equal $committedCheck.reservationState "COMMITTED" "Reservation state must be COMMITTED."
    Assert-Equal `
        ([decimal]$committedCheck.summary.reservedAmount) `
        $reservedBefore `
        "Committed reservation must leave the reserved bucket."
    Assert-Equal `
        ([decimal]$committedCheck.summary.committedAmount) `
        ($committedBefore + [decimal]27000000) `
        "Committed total must increase by the request amount."

    $timeline = Invoke-DxApi -Method GET `
        -Path "/api/v1/purchase-requests/$($draft.id)/timeline?page=1&pageSize=20" `
        -Token $employeeToken
    Assert-Equal ([int64]$timeline.total) 7 "Timeline must include workflow and budget events."
    $eventTypes = @($timeline.items | ForEach-Object { $_.eventType })
    foreach ($expectedEvent in @(
        "DRAFT_CREATED",
        "DRAFT_UPDATED",
        "SUBMITTED",
        "MANAGER_APPROVED",
        "BUDGET_RESERVED",
        "FINANCE_APPROVED"
        "BUDGET_COMMITTED"
    )) {
        if ($expectedEvent -notin $eventTypes) {
            throw "Timeline is missing event $expectedEvent."
        }
    }
    $managerEvent = @($timeline.items | Where-Object { $_.eventType -eq "MANAGER_APPROVED" })[0]
    $financeEvent = @($timeline.items | Where-Object { $_.eventType -eq "FINANCE_APPROVED" })[0]
    Assert-Equal $managerEvent.comment "Department approval." "Timeline must expose the manager comment."
    Assert-Equal $financeEvent.comment "Budget approved." "Timeline must expose the finance comment."
    foreach ($event in @($timeline.items)) {
        if (
            "metadata" -in $event.PSObject.Properties.Name -or
            "idempotencyKey" -in $event.PSObject.Properties.Name
        ) {
            throw "Timeline exposed internal metadata or an idempotency key."
        }
    }

    $releaseDraft = Invoke-DxApi -Method POST -Path "/api/v1/purchase-requests" `
        -Token $employeeToken `
        -Body @{
            title      = "Budget release smoke test $([DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds())"
            reason     = "Verify that requesting changes releases the active budget reservation."
            currency   = "VND"
            costCenter = "CC-GENERAL"
            items      = @(
                @{
                    description = "Release test item"
                    quantity    = "1"
                    unit        = "unit"
                    unitPrice   = "1000000"
                }
            )
        }
    $releaseSubmitted = Invoke-DxApi -Method POST `
        -Path "/api/v1/purchase-requests/$($releaseDraft.id)/transitions" `
        -Token $employeeToken `
        -IdempotencyKey "release-submit-$([Guid]::NewGuid().ToString('N'))" `
        -Body @{ action = "SUBMIT"; expectedVersion = 1; comment = "" }
    $releaseApproved = Invoke-DxApi -Method POST `
        -Path "/api/v1/purchase-requests/$($releaseDraft.id)/transitions" `
        -Token $managerToken `
        -IdempotencyKey "release-manager-$([Guid]::NewGuid().ToString('N'))" `
        -Body @{ action = "APPROVE"; expectedVersion = $releaseSubmitted.version; comment = "" }
    $releaseReservedCheck = Invoke-DxApi -Method GET `
        -Path "/api/v1/purchase-requests/$($releaseDraft.id)/budget-check" `
        -Token $financeToken
    Assert-Equal $releaseReservedCheck.result "RESERVED" "The second request must reserve budget."
    $releasedRequest = Invoke-DxApi -Method POST `
        -Path "/api/v1/purchase-requests/$($releaseDraft.id)/transitions" `
        -Token $financeToken `
        -IdempotencyKey "release-finance-$([Guid]::NewGuid().ToString('N'))" `
        -Body @{
            action          = "REQUEST_CHANGES"
            expectedVersion = $releaseApproved.version
            comment         = "Adjust the specification."
        }
    Assert-Equal $releasedRequest.status "CHANGES_REQUESTED" "Finance must return the request for changes."
    $releasedCheck = Invoke-DxApi -Method GET `
        -Path "/api/v1/purchase-requests/$($releaseDraft.id)/budget-check" `
        -Token $employeeToken
    Assert-Equal $releasedCheck.result "AVAILABLE" "Requesting changes must release the reservation."
    Assert-Equal `
        ([decimal]$releasedCheck.summary.reservedAmount) `
        $reservedBefore `
        "Released budget must return to the available bucket."

    $budgetAfterRelease = Invoke-DxApi -Method GET `
        -Path "/api/v1/budgets/summary?costCenter=CC-GENERAL&currency=VND" `
        -Token $financeToken
    $insufficientAmount = [decimal]$budgetAfterRelease.availableAmount + [decimal]1
    $insufficientAmountText = $insufficientAmount.ToString(
        "0",
        [Globalization.CultureInfo]::InvariantCulture
    )
    $insufficientDraft = Invoke-DxApi -Method POST -Path "/api/v1/purchase-requests" `
        -Token $employeeToken `
        -Body @{
            title      = "Insufficient budget smoke test $([DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds())"
            reason     = "Verify that a manager cannot approve beyond the available allocation."
            currency   = "VND"
            costCenter = "CC-GENERAL"
            items      = @(
                @{
                    description = "Over-budget test item"
                    quantity    = "1"
                    unit        = "unit"
                    unitPrice   = $insufficientAmountText
                }
            )
        }
    $null = Add-TestQuotation `
        -RequestId $insufficientDraft.id `
        -Token $employeeToken `
        -Name "workflow-insufficient-quotation"
    $insufficientSubmitted = Invoke-DxApi -Method POST `
        -Path "/api/v1/purchase-requests/$($insufficientDraft.id)/transitions" `
        -Token $employeeToken `
        -IdempotencyKey "insufficient-submit-$([Guid]::NewGuid().ToString('N'))" `
        -Body @{ action = "SUBMIT"; expectedVersion = 1; comment = "" }
    $insufficientCheck = Invoke-DxApi -Method GET `
        -Path "/api/v1/purchase-requests/$($insufficientDraft.id)/budget-check" `
        -Token $managerToken
    Assert-Equal $insufficientCheck.result "INSUFFICIENT" "Budget check must flag an over-budget request."
    try {
        $null = Invoke-DxApi -Method POST `
            -Path "/api/v1/purchase-requests/$($insufficientDraft.id)/transitions" `
            -Token $managerToken `
            -IdempotencyKey "insufficient-manager-$([Guid]::NewGuid().ToString('N'))" `
            -Body @{ action = "APPROVE"; expectedVersion = $insufficientSubmitted.version; comment = "" }
        throw "An over-budget manager approval unexpectedly succeeded."
    }
    catch {
        $statusCode = [int]$_.Exception.Response.StatusCode
        Assert-Equal $statusCode 409 "Over-budget manager approval must return HTTP 409."
    }
    $null = Invoke-DxApi -Method POST `
        -Path "/api/v1/purchase-requests/$($insufficientDraft.id)/transitions" `
        -Token $managerToken `
        -IdempotencyKey "insufficient-reject-$([Guid]::NewGuid().ToString('N'))" `
        -Body @{
            action          = "REJECT"
            expectedVersion = $insufficientSubmitted.version
            comment         = "Insufficient budget test cleanup."
        }

    try {
        $null = Invoke-DxApi -Method GET `
            -Path "/api/v1/purchase-requests/$($draft.id)/timeline?page=1&pageSize=20" `
            -Token $outsiderToken
        throw "An employee outside the request ownership unexpectedly read the timeline."
    }
    catch {
        $statusCode = [int]$_.Exception.Response.StatusCode
        Assert-Equal $statusCode 404 "Timeline outside the employee ownership scope must return HTTP 404."
    }

    Write-Host "Procurement workflow smoke test passed."
    Write-Host "Reservation, commitment, release, insufficient-budget blocking, timeline, and scope checks passed."
}
finally {
    $employeeToken = $null
    $managerToken = $null
    $financeToken = $null
    $outsiderToken = $null
    foreach ($tokenPath in @(
        $employeeTokenPath,
        $managerTokenPath,
        $financeTokenPath,
        $outsiderTokenPath
    )) {
        if (Test-Path -LiteralPath $tokenPath) {
            Remove-Item -LiteralPath $tokenPath -Force
        }
    }
    Pop-Location
}
