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
$auditorCredentials = "data\runtime\workflow-auditor.txt"
$adminCredentials = "data\runtime\workflow-admin.txt"
$employeeTokenPath = Join-Path $runtimeRoot "workflow-employee.token"
$managerTokenPath = Join-Path $runtimeRoot "workflow-manager.token"
$financeTokenPath = Join-Path $runtimeRoot "workflow-finance.token"
$outsiderTokenPath = Join-Path $runtimeRoot "workflow-outsider.token"
$auditorTokenPath = Join-Path $runtimeRoot "workflow-auditor.token"
$adminTokenPath = Join-Path $runtimeRoot "workflow-admin.token"

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

function Wait-DxNotification {
    param(
        [Parameter(Mandatory)][string]$Token,
        [Parameter(Mandatory)][string]$EventType,
        [Parameter(Mandatory)][string]$ResourceId
    )

    for ($attempt = 0; $attempt -lt 12; $attempt++) {
        $notifications = Invoke-DxApi -Method GET `
            -Path "/api/v1/me/notifications?page=1&pageSize=50&unreadOnly=false" `
            -Token $Token
        $match = @(
            $notifications.items |
                Where-Object { $_.eventType -eq $EventType -and $_.resourceId -eq $ResourceId }
        ) | Select-Object -First 1
        if ($null -ne $match) {
            return $match
        }
        Start-Sleep -Milliseconds 500
    }
    throw "Notification $EventType for resource $ResourceId was not materialized in time."
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
$supplier = $null
$financeToken = $null
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
    & "$PSScriptRoot\Initialize-DevUser.ps1" `
        -Username "workflow.auditor" `
        -Role "auditor" `
        -CredentialsPath $auditorCredentials
    & "$PSScriptRoot\Initialize-DevUser.ps1" `
        -Username "workflow.admin" `
        -Role "dx_admin" `
        -CredentialsPath $adminCredentials

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
    & "$PSScriptRoot\Test-OIDCLogin.ps1" `
        -CredentialsPath $auditorCredentials `
        -ExpectedRole "auditor" `
        -AccessTokenOutputPath "data\runtime\workflow-auditor.token"
    & "$PSScriptRoot\Test-OIDCLogin.ps1" `
        -CredentialsPath $adminCredentials `
        -ExpectedRole "dx_admin" `
        -AccessTokenOutputPath "data\runtime\workflow-admin.token"

    $employeeToken = [IO.File]::ReadAllText($employeeTokenPath)
    $managerToken = [IO.File]::ReadAllText($managerTokenPath)
    $financeToken = [IO.File]::ReadAllText($financeTokenPath)
    $outsiderToken = [IO.File]::ReadAllText($outsiderTokenPath)
    $auditorToken = [IO.File]::ReadAllText($auditorTokenPath)
    $adminToken = [IO.File]::ReadAllText($adminTokenPath)

    $adminPolicies = Invoke-DxApi -Method GET -Path "/api/v1/admin/policies" -Token $adminToken
    Assert-Equal $adminPolicies.canManage $true "DX admin must manage operating policies."
    $approvalPolicy = @(
        $adminPolicies.slaPolicies | Where-Object { $_.processName -eq "PURCHASE_REQUEST_APPROVAL" }
    )[0]
    $temporaryHours = if ([int]$approvalPolicy.targetHours -eq 71) { 72 } else { 71 }
    $updatedPolicy = Invoke-DxApi -Method PATCH `
        -Path "/api/v1/admin/policies/sla/PURCHASE_REQUEST_APPROVAL" `
        -Token $adminToken `
        -Body @{
            targetHours     = $temporaryHours
            active          = $approvalPolicy.active
            expectedVersion = $approvalPolicy.version
        }
    Assert-Equal ([int]$updatedPolicy.targetHours) $temporaryHours "DX admin must update the SLA policy."
    Assert-Equal ([int64]$updatedPolicy.version) ([int64]$approvalPolicy.version + 1) "Policy updates must increment version."
    try {
        $null = Invoke-DxApi -Method PATCH `
            -Path "/api/v1/admin/policies/sla/PURCHASE_REQUEST_APPROVAL" `
            -Token $adminToken `
            -Body @{
                targetHours     = $temporaryHours
                active          = $approvalPolicy.active
                expectedVersion = $approvalPolicy.version
            }
        throw "A stale policy update unexpectedly succeeded."
    }
    catch {
        $statusCode = [int]$_.Exception.Response.StatusCode
        Assert-Equal $statusCode 409 "Optimistic locking must reject stale policy updates."
    }
    $restoredPolicy = Invoke-DxApi -Method PATCH `
        -Path "/api/v1/admin/policies/sla/PURCHASE_REQUEST_APPROVAL" `
        -Token $adminToken `
        -Body @{
            targetHours     = $approvalPolicy.targetHours
            active          = $approvalPolicy.active
            expectedVersion = $updatedPolicy.version
        }
    Assert-Equal ([int]$restoredPolicy.targetHours) ([int]$approvalPolicy.targetHours) "The smoke test must restore the SLA value."

    $auditorPolicies = Invoke-DxApi -Method GET -Path "/api/v1/admin/policies" -Token $auditorToken
    Assert-Equal $auditorPolicies.canManage $false "Auditors must have read-only policy access."
    try {
        $null = Invoke-DxApi -Method PATCH `
            -Path "/api/v1/admin/policies/sla/PURCHASE_REQUEST_APPROVAL" `
            -Token $auditorToken `
            -Body @{ targetHours = 48; active = $true; expectedVersion = $restoredPolicy.version }
        throw "An auditor unexpectedly changed an operating policy."
    }
    catch {
        $statusCode = [int]$_.Exception.Response.StatusCode
        Assert-Equal $statusCode 403 "Auditors must not update operating policies."
    }
    try {
        $null = Invoke-DxApi -Method GET -Path "/api/v1/admin/policies" -Token $employeeToken
        throw "An employee unexpectedly read operating policies."
    }
    catch {
        $statusCode = [int]$_.Exception.Response.StatusCode
        Assert-Equal $statusCode 403 "Employees must not read operating policies."
    }

    $budgetBefore = Invoke-DxApi -Method GET `
        -Path "/api/v1/budgets/summary?costCenter=CC-GENERAL&currency=VND" `
        -Token $financeToken
    $reservedBefore = [decimal]$budgetBefore.reservedAmount
    $committedBefore = [decimal]$budgetBefore.committedAmount

    $draftBody = @{
        title      = "[KIỂM THỬ TỰ ĐỘNG] Luồng mua sắm $([DateTimeOffset]::UtcNow.ToUnixTimeSeconds())"
        reason     = "Kiểm tra tự động luồng cập nhật và phê duyệt hai cấp."
        currency   = "VND"
        costCenter = "CC-GENERAL"
        items      = @(
            @{
                description = "Máy trạm phục vụ nhóm phát triển"
                quantity    = "1"
                unit        = "chiếc"
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
        reason          = "Hồ sơ kiểm thử tự động đã cập nhật trước khi trình phê duyệt."
        currency        = "VND"
        costCenter      = "CC-GENERAL"
        expectedVersion = 1
        items           = @(
            @{
                description = "Máy trạm phục vụ nhóm phát triển"
                quantity    = "1"
                unit        = "chiếc"
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

    $managerNotification = Wait-DxNotification `
        -Token $managerToken `
        -EventType "SUBMITTED" `
        -ResourceId $draft.id
    $null = Invoke-DxApi -Method POST `
        -Path "/api/v1/me/notifications/$($managerNotification.id)/read" `
        -Token $managerToken
    $managerNotificationsAfterRead = Invoke-DxApi -Method GET `
        -Path "/api/v1/me/notifications?page=1&pageSize=50&unreadOnly=false" `
        -Token $managerToken
    $readManagerNotification = @(
        $managerNotificationsAfterRead.items | Where-Object { $_.id -eq $managerNotification.id }
    )[0]
    if ($null -eq $readManagerNotification.readAt) {
        throw "A notification marked as read still has no readAt timestamp."
    }

    $replayed = Invoke-DxApi -Method POST `
        -Path "/api/v1/purchase-requests/$($draft.id)/transitions" `
        -Token $employeeToken `
        -IdempotencyKey $submitKey `
        -Body @{ action = "SUBMIT"; expectedVersion = 2; comment = "" }
    Assert-Equal ([int64]$replayed.version) 3 "Idempotent replay must not increment the version."

    $managerTasks = Invoke-DxApi -Method GET -Path "/api/v1/me/tasks-summary" -Token $managerToken
    if ($draft.id -notin @($managerTasks.items | ForEach-Object { $_.purchaseRequestId })) {
        throw "Manager work center is missing the submitted request."
    }

    $createdComment = Invoke-DxApi -Method POST `
        -Path "/api/v1/purchase-requests/$($draft.id)/comments" `
        -Token $employeeToken `
        -Body @{ body = "Vui lòng xác nhận ngày giao dự kiến trước khi phê duyệt." }
    Assert-Equal $createdComment.body "Vui lòng xác nhận ngày giao dự kiến trước khi phê duyệt." `
        "Independent comment must preserve its normalized body."

    $managerComments = Invoke-DxApi -Method GET `
        -Path "/api/v1/purchase-requests/$($draft.id)/comments" `
        -Token $managerToken
    Assert-Equal ([int64]$managerComments.total) 1 "Manager must read scoped comments."

    $auditorComments = Invoke-DxApi -Method GET `
        -Path "/api/v1/purchase-requests/$($draft.id)/comments" `
        -Token $auditorToken
    Assert-Equal ([int64]$auditorComments.total) 1 "Auditor must read comments."
    try {
        $null = Invoke-DxApi -Method POST `
            -Path "/api/v1/purchase-requests/$($draft.id)/comments" `
            -Token $auditorToken `
            -Body @{ body = "Auditor must not be able to post." }
        throw "An auditor unexpectedly posted a comment."
    }
    catch {
        $statusCode = [int]$_.Exception.Response.StatusCode
        Assert-Equal $statusCode 403 "Auditor comment creation must return HTTP 403."
    }

    $managerApproved = Invoke-DxApi -Method POST `
        -Path "/api/v1/purchase-requests/$($draft.id)/transitions" `
        -Token $managerToken `
        -IdempotencyKey "manager-$([Guid]::NewGuid().ToString('N'))" `
        -Body @{ action = "APPROVE"; expectedVersion = 3; comment = "Trưởng bộ phận đã phê duyệt nhu cầu." }
    Assert-Equal $managerApproved.status "MANAGER_APPROVED" "Manager approval must enter finance review."
    Assert-Equal ([int64]$managerApproved.version) 4 "Manager approval must increment the version."

    $financeTasks = Invoke-DxApi -Method GET -Path "/api/v1/me/tasks-summary" -Token $financeToken
    if ($draft.id -notin @($financeTasks.items | ForEach-Object { $_.purchaseRequestId })) {
        throw "Finance work center is missing the manager-approved request."
    }

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
        -Body @{ action = "APPROVE"; expectedVersion = 4; comment = "Bộ phận tài chính đã xác nhận ngân sách." }
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
    Assert-Equal ([int64]$timeline.total) 8 "Timeline must include workflow, comment, and budget events."
    $eventTypes = @($timeline.items | ForEach-Object { $_.eventType })
    foreach ($expectedEvent in @(
        "DRAFT_CREATED",
        "DRAFT_UPDATED",
        "SUBMITTED",
        "COMMENT_ADDED",
        "MANAGER_APPROVED",
        "BUDGET_RESERVED",
        "FINANCE_APPROVED",
        "BUDGET_COMMITTED"
    )) {
        if ($expectedEvent -notin $eventTypes) {
            throw "Timeline is missing event $expectedEvent."
        }
    }
    $managerEvent = @($timeline.items | Where-Object { $_.eventType -eq "MANAGER_APPROVED" })[0]
    $financeEvent = @($timeline.items | Where-Object { $_.eventType -eq "FINANCE_APPROVED" })[0]
    Assert-Equal $managerEvent.comment "Trưởng bộ phận đã phê duyệt nhu cầu." "Timeline must expose the manager comment."
    Assert-Equal $financeEvent.comment "Bộ phận tài chính đã xác nhận ngân sách." "Timeline must expose the finance comment."
    foreach ($event in @($timeline.items)) {
        if (
            "metadata" -in $event.PSObject.Properties.Name -or
            "idempotencyKey" -in $event.PSObject.Properties.Name
        ) {
            throw "Timeline exposed internal metadata or an idempotency key."
        }
    }

    $supplierSuffix = [DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds()
    $supplier = Invoke-DxApi -Method POST `
        -Path "/api/v1/suppliers" `
        -Token $financeToken `
        -Body @{
            code        = "WF-$supplierSuffix"
            name        = "[KIỂM THỬ TỰ ĐỘNG] Nhà cung cấp tạm $supplierSuffix"
            taxCode     = "TAX-$supplierSuffix"
            contactName = "Tài khoản kiểm thử"
            email       = "workflow-$supplierSuffix@example.test"
            phone       = "0900000000"
            status      = "ACTIVE"
            riskLevel   = "LOW"
        }
    Assert-Equal $supplier.status "ACTIVE" "Finance must be able to create an active supplier."
    Assert-Equal ([int64]$supplier.version) 1 "A new supplier must start at version 1."

    $financeSuppliers = Invoke-DxApi -Method GET -Path "/api/v1/suppliers" -Token $financeToken
    Assert-Equal $financeSuppliers.canManage $true "Finance must be allowed to manage suppliers."
    if ($supplier.id -notin @($financeSuppliers.items | ForEach-Object { $_.id })) {
        throw "The finance supplier directory is missing the newly created supplier."
    }

    $financeOperations = Invoke-DxApi -Method GET `
        -Path "/api/v1/procurement-operations" `
        -Token $financeToken
    $awaitingOrder = @($financeOperations.items | Where-Object { $_.purchaseRequestId -eq $draft.id })[0]
    Assert-Equal $awaitingOrder.status "AWAITING_ORDER" "An approved request must wait for order placement."
    Assert-Equal $awaitingOrder.canPlaceOrder $true "Finance must be able to place an order."

    $order = Invoke-DxApi -Method POST `
        -Path "/api/v1/procurement-operations/orders" `
        -Token $financeToken `
        -IdempotencyKey "order-$([Guid]::NewGuid().ToString('N'))" `
        -Body @{
            purchaseRequestId  = $draft.id
            supplierId         = $supplier.id
            externalReference  = "DEMO-$supplierSuffix"
            expectedDeliveryOn = [DateTime]::UtcNow.Date.AddDays(1).ToString("yyyy-MM-dd")
            note               = "Đơn hàng do kiểm thử tự động tạo; nhà cung cấp sẽ được lưu trữ sau khi hoàn tất."
        }
    Assert-Equal $order.status "ORDERED" "Finance order placement must enter ORDERED status."
    Assert-Equal ([int64]$order.version) 1 "A new order must start at version 1."

    $invoice = Invoke-DxApi -Method POST `
        -Path "/api/v1/invoices" `
        -Token $financeToken `
        -IdempotencyKey "invoice-$([Guid]::NewGuid().ToString('N'))" `
        -Body @{
            purchaseOrderId = $order.id
            invoiceNumber   = "INV-WF-$supplierSuffix"
            issuedOn        = [DateTime]::UtcNow.Date.ToString("yyyy-MM-dd")
            dueOn           = [DateTime]::UtcNow.Date.AddDays(15).ToString("yyyy-MM-dd")
            amount          = $order.totalAmount
            currency        = $order.currency
            note            = "Hóa đơn kiểm thử được ghi nhận trước khi nhận hàng để kiểm tra đối chiếu ba bên."
        }
    Assert-Equal $invoice.invoiceStatus "RECORDED" "A new invoice must be recorded."
    Assert-Equal $invoice.matchStatus "WAITING_RECEIPT" "An invoice must wait for a goods receipt."
    Assert-Equal ([int64]$invoice.version) 1 "A new invoice must start at version 1."

    try {
        $null = Invoke-DxApi -Method POST `
            -Path "/api/v1/invoices/$($invoice.invoiceId)/transitions" `
            -Token $financeToken `
            -IdempotencyKey "invoice-premature-$([Guid]::NewGuid().ToString('N'))" `
            -Body @{ action = "VERIFY"; expectedVersion = $invoice.version }
        throw "An invoice was unexpectedly verified before goods receipt."
    }
    catch {
        $statusCode = [int]$_.Exception.Response.StatusCode
        Assert-Equal $statusCode 409 "Three-way matching must block verification before receipt."
    }

    try {
        $null = Invoke-DxApi -Method POST `
            -Path "/api/v1/procurement-operations/orders/$($draft.id)/receipt" `
            -Token $financeToken `
            -Body @{
                expectedVersion  = $order.version
                actualDeliveryOn = [DateTime]::UtcNow.Date.ToString("yyyy-MM-dd")
            }
        throw "Finance unexpectedly confirmed receipt for its own purchase order."
    }
    catch {
        $statusCode = [int]$_.Exception.Response.StatusCode
        Assert-Equal $statusCode 403 "Separation of duties must prevent finance from confirming receipt."
    }

    $receivedOrder = Invoke-DxApi -Method POST `
        -Path "/api/v1/procurement-operations/orders/$($draft.id)/receipt" `
        -Token $employeeToken `
        -Body @{
            expectedVersion  = $order.version
            actualDeliveryOn = [DateTime]::UtcNow.Date.ToString("yyyy-MM-dd")
        }
    Assert-Equal $receivedOrder.status "RECEIVED" "The requester must be able to confirm delivery."
    Assert-Equal ([int64]$receivedOrder.version) 2 "Receipt confirmation must increment the order version."

    $financeInvoices = Invoke-DxApi -Method GET -Path "/api/v1/invoices" -Token $financeToken
    $matchedInvoice = @(
        $financeInvoices.items | Where-Object { $_.invoiceId -eq $invoice.invoiceId }
    )[0]
    Assert-Equal $matchedInvoice.matchStatus "MATCHED" "Receipt must complete the three-way match."
    Assert-Equal $financeInvoices.canManage $true "Finance must manage invoices."

    $verifiedInvoice = Invoke-DxApi -Method POST `
        -Path "/api/v1/invoices/$($invoice.invoiceId)/transitions" `
        -Token $financeToken `
        -IdempotencyKey "invoice-verify-$([Guid]::NewGuid().ToString('N'))" `
        -Body @{ action = "VERIFY"; expectedVersion = $matchedInvoice.version }
    Assert-Equal $verifiedInvoice.invoiceStatus "VERIFIED" "A matched invoice must be verifiable."

    $paidInvoice = Invoke-DxApi -Method POST `
        -Path "/api/v1/invoices/$($invoice.invoiceId)/transitions" `
        -Token $financeToken `
        -IdempotencyKey "invoice-payment-$([Guid]::NewGuid().ToString('N'))" `
        -Body @{
            action           = "MARK_PAID"
            expectedVersion  = $verifiedInvoice.version
            paymentReference = "BANK-$supplierSuffix"
            paidOn           = [DateTime]::UtcNow.Date.ToString("yyyy-MM-dd")
        }
    Assert-Equal $paidInvoice.invoiceStatus "PAID" "A verified invoice must be payable."
    Assert-Equal $paidInvoice.paymentReference "BANK-$supplierSuffix" "Payment evidence must be retained."

    $auditorInvoices = Invoke-DxApi -Method GET -Path "/api/v1/invoices" -Token $auditorToken
    Assert-Equal $auditorInvoices.canManage $false "Auditors must have read-only invoice access."
    if ($invoice.invoiceId -notin @($auditorInvoices.items | ForEach-Object { $_.invoiceId })) {
        throw "Auditor invoice board is missing the paid invoice."
    }
    try {
        $null = Invoke-DxApi -Method POST `
            -Path "/api/v1/invoices/$($invoice.invoiceId)/transitions" `
            -Token $auditorToken `
            -IdempotencyKey "auditor-payment-$([Guid]::NewGuid().ToString('N'))" `
            -Body @{ action = "MARK_PAID"; expectedVersion = $paidInvoice.version; paymentReference = "FORBIDDEN"; paidOn = [DateTime]::UtcNow.Date.ToString("yyyy-MM-dd") }
        throw "An auditor unexpectedly changed an invoice."
    }
    catch {
        $statusCode = [int]$_.Exception.Response.StatusCode
        Assert-Equal $statusCode 403 "Auditors must not change invoices."
    }
    try {
        $null = Invoke-DxApi -Method GET -Path "/api/v1/invoices" -Token $employeeToken
        throw "An employee unexpectedly read the invoice board."
    }
    catch {
        $statusCode = [int]$_.Exception.Response.StatusCode
        Assert-Equal $statusCode 403 "Employees must not read the invoice board."
    }

    $null = Wait-DxNotification -Token $employeeToken -EventType "FINANCE_APPROVED" -ResourceId $draft.id
    $null = Wait-DxNotification -Token $employeeToken -EventType "ORDER_PLACED" -ResourceId $draft.id
    $null = Wait-DxNotification -Token $financeToken -EventType "DELIVERY_RECEIVED" -ResourceId $draft.id
    $null = Wait-DxNotification -Token $employeeToken -EventType "INVOICE_PAID" -ResourceId $draft.id
    $outsiderNotifications = Invoke-DxApi -Method GET `
        -Path "/api/v1/me/notifications?page=1&pageSize=50&unreadOnly=false" `
        -Token $outsiderToken
    if ($draft.id -in @($outsiderNotifications.items | ForEach-Object { $_.resourceId })) {
        throw "A notification leaked outside its user/role/department scope."
    }

    $auditorOperations = Invoke-DxApi -Method GET `
        -Path "/api/v1/procurement-operations" `
        -Token $auditorToken
    $auditedOrder = @($auditorOperations.items | Where-Object { $_.purchaseRequestId -eq $draft.id })[0]
    Assert-Equal $auditedOrder.status "RECEIVED" "Auditors must see the completed fulfillment state."
    Assert-Equal $auditedOrder.canPlaceOrder $false "Auditors must have read-only access to operations."
    Assert-Equal $auditedOrder.canConfirmReceipt $false "Auditors must not confirm receipt."

    $auditorSuppliers = Invoke-DxApi -Method GET -Path "/api/v1/suppliers" -Token $auditorToken
    Assert-Equal $auditorSuppliers.canManage $false "Auditor supplier access must be read-only."
    try {
        $null = Invoke-DxApi -Method POST `
            -Path "/api/v1/suppliers" `
            -Token $auditorToken `
            -Body @{
                code = "AUD-$supplierSuffix"; name = "Forbidden Auditor Supplier"
                taxCode = ""; contactName = ""; email = ""; phone = ""
                status = "ACTIVE"; riskLevel = "LOW"
            }
        throw "Auditor unexpectedly created a supplier."
    }
    catch {
        $statusCode = [int]$_.Exception.Response.StatusCode
        Assert-Equal $statusCode 403 "Auditors must not create suppliers."
    }

    try {
        $null = Invoke-DxApi -Method GET -Path "/api/v1/suppliers" -Token $employeeToken
        throw "Employee unexpectedly read the supplier directory."
    }
    catch {
        $statusCode = [int]$_.Exception.Response.StatusCode
        Assert-Equal $statusCode 403 "Employees must not read the supplier directory."
    }

    $auditCenter = Invoke-DxApi -Method GET `
        -Path "/api/v1/audit/events?resourceType=purchase_order&page=1&pageSize=50" `
        -Token $auditorToken
    $orderAuditActions = @(
        $auditCenter.items |
            Where-Object { $_.resourceId -eq $order.id } |
            ForEach-Object { $_.action }
    )
    foreach ($expectedAction in @("PURCHASE_ORDER_CREATED", "DELIVERY_RECEIVED")) {
        if ($expectedAction -notin $orderAuditActions) {
            throw "Audit center is missing purchase-order action $expectedAction."
        }
    }

    $invoiceAuditCenter = Invoke-DxApi -Method GET `
        -Path "/api/v1/audit/events?resourceType=purchase_invoice&page=1&pageSize=50" `
        -Token $auditorToken
    $invoiceAuditActions = @(
        $invoiceAuditCenter.items |
            Where-Object { $_.resourceId -eq $invoice.invoiceId } |
            ForEach-Object { $_.action }
    )
    foreach ($expectedAction in @("INVOICE_RECORDED", "INVOICE_VERIFIED", "INVOICE_PAID")) {
        if ($expectedAction -notin $invoiceAuditActions) {
            throw "Audit center is missing invoice action $expectedAction."
        }
    }

    $releaseDraft = Invoke-DxApi -Method POST -Path "/api/v1/purchase-requests" `
        -Token $employeeToken `
        -Body @{
            title      = "[KIỂM THỬ TỰ ĐỘNG] Kiểm tra hoàn ngân sách $([DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds())"
            reason     = "Kiểm tra việc hoàn lại khoản ngân sách đang giữ khi phiếu bị yêu cầu chỉnh sửa."
            currency   = "VND"
            costCenter = "CC-GENERAL"
            items      = @(
                @{
                    description = "Thiết bị kiểm tra hoàn ngân sách"
                    quantity    = "1"
                    unit        = "chiếc"
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
            comment         = "Cần điều chỉnh thông số kỹ thuật và làm rõ nhu cầu."
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
            title      = "[KIỂM THỬ TỰ ĐỘNG] Kiểm tra vượt hạn mức $([DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds())"
            reason     = "Kiểm tra việc ngăn phê duyệt khi giá trị phiếu vượt ngân sách khả dụng."
            currency   = "VND"
            costCenter = "CC-GENERAL"
            items      = @(
                @{
                    description = "Thiết bị kiểm tra vượt hạn mức"
                    quantity    = "1"
                    unit        = "chiếc"
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
            comment         = "Dừng xử lý do ngân sách khả dụng chưa đáp ứng."
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
    Write-Host "Work center, notifications/outbox, comments, SLA/policy administration, rate-safe RBAC, budget, suppliers, orders, receipt, invoice matching/payment, audit, timeline, and scope checks passed."
}
finally {
    if ($null -ne $supplier -and -not [string]::IsNullOrWhiteSpace($financeToken)) {
        try {
            $null = Invoke-DxApi -Method PATCH `
                -Path "/api/v1/suppliers/$($supplier.id)" `
                -Token $financeToken `
                -Body @{
                    code                = $supplier.code
                    name                = "[KIỂM THỬ TỰ ĐỘNG] Nhà cung cấp đã lưu trữ"
                    taxCode             = $supplier.taxCode
                    contactName         = $supplier.contactName
                    email               = $supplier.email
                    phone               = $supplier.phone
                    address             = $supplier.address
                    bankName            = $supplier.bankName
                    bankAccountNumber   = $supplier.bankAccountNumber
                    contractReference   = $supplier.contractReference
                    contractExpiresOn   = $supplier.contractExpiresOn
                    complianceStatus    = $supplier.complianceStatus
                    performanceScore    = $supplier.performanceScore
                    businessNote        = "Bản ghi tạm của kiểm thử tự động; được lưu trữ để không xuất hiện trong danh sách nghiệp vụ."
                    status              = "INACTIVE"
                    riskLevel           = $supplier.riskLevel
                    expectedVersion     = $supplier.version
                }
        }
        catch {
            Write-Warning "Không thể lưu trữ nhà cung cấp kiểm thử tự động: $($_.Exception.Message)"
        }
    }
    $employeeToken = $null
    $managerToken = $null
    $financeToken = $null
    $outsiderToken = $null
    $auditorToken = $null
    $adminToken = $null
    foreach ($tokenPath in @(
        $employeeTokenPath,
        $managerTokenPath,
        $financeTokenPath,
        $outsiderTokenPath,
        $auditorTokenPath,
        $adminTokenPath
    )) {
        if (Test-Path -LiteralPath $tokenPath) {
            Remove-Item -LiteralPath $tokenPath -Force
        }
    }
    Pop-Location
}
