[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repositoryRoot = [IO.Path]::GetFullPath((Split-Path -Parent $PSScriptRoot))
$runtimeRoot = Join-Path $repositoryRoot "data\runtime"
$employeeCredentials = "data\runtime\attachments-employee.txt"
$outsiderCredentials = "data\runtime\attachments-outsider.txt"
$employeeTokenPath = Join-Path $runtimeRoot "attachments-employee.token"
$outsiderTokenPath = Join-Path $runtimeRoot "attachments-outsider.token"
$sourcePath = Join-Path $runtimeRoot "attachment-source.pdf"
$downloadPath = Join-Path $runtimeRoot "attachment-download.pdf"
$uploadResponsePath = Join-Path $runtimeRoot "attachment-upload.json"

function Invoke-DxApi {
    param(
        [Parameter(Mandatory)][ValidateSet("GET", "POST", "DELETE")][string]$Method,
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
    if ($IdempotencyKey) {
        $parameters.Headers["Idempotency-Key"] = $IdempotencyKey
    }
    return Invoke-RestMethod @parameters
}

function Assert-Equal {
    param($Actual, $Expected, [string]$Message)
    if ($Actual -ne $Expected) {
        throw "$Message Expected '$Expected', received '$Actual'."
    }
}

function Upload-Quotation {
    param([string]$RequestId, [string]$Token)
    $status = & curl.exe --silent --show-error `
        --output $uploadResponsePath `
        --write-out "%{http_code}" `
        --header "Authorization: Bearer $Token" `
        --form "documentType=QUOTATION" `
        --form "file=@$sourcePath;type=application/pdf" `
        "http://localhost:8081/api/v1/purchase-requests/$RequestId/attachments"
    Assert-Equal ([int]$status) 201 "Quotation upload must return HTTP 201."
    return [IO.File]::ReadAllText($uploadResponsePath) | ConvertFrom-Json
}

New-Item -ItemType Directory -Path $runtimeRoot -Force | Out-Null
[IO.File]::WriteAllBytes(
    $sourcePath,
    [Text.Encoding]::ASCII.GetBytes("%PDF-1.4`nDX-OS attachment acceptance test`n%%EOF")
)

Push-Location $repositoryRoot
try {
    & "$PSScriptRoot\Initialize-DevUser.ps1" `
        -Username "attachments.employee" -Role "employee" `
        -CredentialsPath $employeeCredentials
    & "$PSScriptRoot\Initialize-DevUser.ps1" `
        -Username "attachments.outsider" -Role "employee" `
        -CredentialsPath $outsiderCredentials
    & "$PSScriptRoot\Test-OIDCLogin.ps1" `
        -CredentialsPath $employeeCredentials -ExpectedRole "employee" `
        -AccessTokenOutputPath "data\runtime\attachments-employee.token"
    & "$PSScriptRoot\Test-OIDCLogin.ps1" `
        -CredentialsPath $outsiderCredentials -ExpectedRole "employee" `
        -AccessTokenOutputPath "data\runtime\attachments-outsider.token"

    $employeeToken = [IO.File]::ReadAllText($employeeTokenPath)
    $outsiderToken = [IO.File]::ReadAllText($outsiderTokenPath)
    $draft = Invoke-DxApi -Method POST -Path "/api/v1/purchase-requests" `
        -Token $employeeToken `
        -Body @{
            title      = "Attachment smoke test $([DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds())"
            reason     = "Verify quotation policy, WebDAV content integrity, scope and lifecycle."
            currency   = "VND"
            costCenter = "CC-GENERAL"
            items      = @(@{
                description = "Attachment policy test item"
                quantity    = "1"
                unit        = "unit"
                unitPrice   = "25000000"
            })
        }

    $rules = Invoke-DxApi -Method GET `
        -Path "/api/v1/purchase-requests/$($draft.id)/attachments" `
        -Token $employeeToken
    Assert-Equal $rules.required $true "A 25M VND request must require a quotation."
    Assert-Equal $rules.requirementMet $false "The initial requirement must not be met."

    try {
        $null = Invoke-DxApi -Method POST `
            -Path "/api/v1/purchase-requests/$($draft.id)/transitions" `
            -Token $employeeToken `
            -IdempotencyKey "attachment-missing-$([Guid]::NewGuid().ToString('N'))" `
            -Body @{ action = "SUBMIT"; expectedVersion = 1; comment = "" }
        throw "Submit without a quotation unexpectedly succeeded."
    }
    catch {
        Assert-Equal ([int]$_.Exception.Response.StatusCode) 422 `
            "Submit without quotation must return HTTP 422."
    }

    $attachment = Upload-Quotation -RequestId $draft.id -Token $employeeToken
    $rules = Invoke-DxApi -Method GET `
        -Path "/api/v1/purchase-requests/$($draft.id)/attachments" `
        -Token $employeeToken
    Assert-Equal $rules.requirementMet $true "Uploaded quotation must satisfy the rule."
    Assert-Equal ([int]$rules.items.Count) 1 "Attachment list must contain the uploaded file."

    $downloadStatus = & curl.exe --silent --show-error `
        --output $downloadPath `
        --write-out "%{http_code}" `
        --header "Authorization: Bearer $employeeToken" `
        "http://localhost:8081/api/v1/purchase-requests/$($draft.id)/attachments/$($attachment.id)/content"
    Assert-Equal ([int]$downloadStatus) 200 "Attachment download must return HTTP 200."
    Assert-Equal `
        (Get-FileHash -Algorithm SHA256 -LiteralPath $downloadPath).Hash `
        (Get-FileHash -Algorithm SHA256 -LiteralPath $sourcePath).Hash `
        "Downloaded content checksum must match the source."

    try {
        $null = Invoke-DxApi -Method GET `
            -Path "/api/v1/purchase-requests/$($draft.id)/attachments" `
            -Token $outsiderToken
        throw "Out-of-scope user unexpectedly listed attachments."
    }
    catch {
        Assert-Equal ([int]$_.Exception.Response.StatusCode) 404 `
            "Out-of-scope attachment list must return HTTP 404."
    }

    $null = Invoke-DxApi -Method DELETE `
        -Path "/api/v1/purchase-requests/$($draft.id)/attachments/$($attachment.id)" `
        -Token $employeeToken
    $rules = Invoke-DxApi -Method GET `
        -Path "/api/v1/purchase-requests/$($draft.id)/attachments" `
        -Token $employeeToken
    Assert-Equal ([int]$rules.items.Count) 0 "Deleted attachment must disappear from the list."

    $attachment = Upload-Quotation -RequestId $draft.id -Token $employeeToken
    $submitted = Invoke-DxApi -Method POST `
        -Path "/api/v1/purchase-requests/$($draft.id)/transitions" `
        -Token $employeeToken `
        -IdempotencyKey "attachment-submit-$([Guid]::NewGuid().ToString('N'))" `
        -Body @{ action = "SUBMIT"; expectedVersion = 1; comment = "" }
    Assert-Equal $submitted.status "SUBMITTED" "Submit with quotation must succeed."
    try {
        $null = Invoke-DxApi -Method DELETE `
            -Path "/api/v1/purchase-requests/$($draft.id)/attachments/$($attachment.id)" `
            -Token $employeeToken
        throw "Deleting an attachment after submit unexpectedly succeeded."
    }
    catch {
        Assert-Equal ([int]$_.Exception.Response.StatusCode) 403 `
            "Delete after submit must return HTTP 403."
    }

    Write-Host "Attachment acceptance test passed."
    Write-Host "Quotation rule, upload, download checksum, delete lifecycle and scope checks passed."
}
finally {
    $employeeToken = $null
    $outsiderToken = $null
    foreach ($path in @(
        $employeeTokenPath,
        $outsiderTokenPath,
        $sourcePath,
        $downloadPath,
        $uploadResponsePath
    )) {
        if (Test-Path -LiteralPath $path) {
            Remove-Item -LiteralPath $path -Force
        }
    }
    Pop-Location
}
