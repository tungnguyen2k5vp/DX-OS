[CmdletBinding()]
param(
    [string]$CredentialsPath = "data\runtime\dev-user.txt",
    [string]$ExpectedRole = "employee",
    [string]$AccessTokenOutputPath,
    [switch]$TestProcurement
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function New-RandomBase64Url {
    param([int]$Length = 48)

    $bytes = New-Object byte[] $Length
    $generator = [System.Security.Cryptography.RandomNumberGenerator]::Create()
    try {
        $generator.GetBytes($bytes)
    }
    finally {
        $generator.Dispose()
    }

    return ([Convert]::ToBase64String($bytes)).TrimEnd("=").Replace("+", "-").Replace("/", "_")
}

function ConvertTo-Base64Url {
    param([byte[]]$Bytes)

    return ([Convert]::ToBase64String($Bytes)).TrimEnd("=").Replace("+", "-").Replace("/", "_")
}

function ConvertFrom-QueryString {
    param([string]$Query)

    $values = @{}
    foreach ($part in $Query.TrimStart("?").Split("&", [System.StringSplitOptions]::RemoveEmptyEntries)) {
        $pair = $part.Split("=", 2)
        $name = [Uri]::UnescapeDataString($pair[0])
        $value = if ($pair.Count -gt 1) { [Uri]::UnescapeDataString($pair[1]) } else { "" }
        $values[$name] = $value
    }
    return $values
}

function ConvertTo-CurlConfigValue {
    param([Parameter(Mandatory)][string]$Value)

    if ($Value.Contains("`r") -or $Value.Contains("`n")) {
        throw "curl config values cannot contain line breaks."
    }
    return $Value.Replace("\", "\\").Replace('"', '\"')
}

function Write-CurlConfig {
    param(
        [Parameter(Mandatory)][string]$Path,
        [Parameter(Mandatory)][string[]]$Lines
    )

    [IO.File]::WriteAllLines($Path, $Lines, [Text.UTF8Encoding]::new($false))
}

function Invoke-CurlConfig {
    param([Parameter(Mandatory)][string]$Path)

    $output = @(& curl.exe --config $Path)
    if ($LASTEXITCODE -ne 0) {
        throw "curl failed while executing the OIDC smoke test."
    }
    return ($output -join "").Trim()
}

$repositoryRoot = [IO.Path]::GetFullPath((Split-Path -Parent $PSScriptRoot))
$resolvedCredentialsPath = if ([IO.Path]::IsPathRooted($CredentialsPath)) {
    [IO.Path]::GetFullPath($CredentialsPath)
}
else {
    [IO.Path]::GetFullPath((Join-Path $repositoryRoot $CredentialsPath))
}

$repositoryPrefix = $repositoryRoot.TrimEnd("\") + "\"
if (-not $resolvedCredentialsPath.StartsWith($repositoryPrefix, [StringComparison]::OrdinalIgnoreCase)) {
    throw "CredentialsPath must stay inside the repository."
}
if (-not (Test-Path -LiteralPath $resolvedCredentialsPath)) {
    throw "Local development credentials are missing. Run scripts\Initialize-DevUser.ps1 first."
}

$credentials = @{}
foreach ($line in Get-Content -LiteralPath $resolvedCredentialsPath) {
    if ($line -match "^[^#=]+=") {
        $pair = $line.Split("=", 2)
        $credentials[$pair[0].Trim()] = $pair[1]
    }
}
if (-not $credentials.ContainsKey("username") -or -not $credentials.ContainsKey("password")) {
    throw "The local credential file is invalid. Regenerate it with scripts\Initialize-DevUser.ps1."
}

$username = [string]$credentials["username"]
$password = [string]$credentials["password"]
$redirectUri = "http://localhost:4200/dashboard"
$state = New-RandomBase64Url 24
$nonce = New-RandomBase64Url 24
$verifier = New-RandomBase64Url 64

$sha256 = [System.Security.Cryptography.SHA256]::Create()
try {
    $challenge = ConvertTo-Base64Url $sha256.ComputeHash([Text.Encoding]::ASCII.GetBytes($verifier))
}
finally {
    $sha256.Dispose()
}

$authorizationParameters = @{
    client_id             = "dx-web"
    redirect_uri          = $redirectUri
    response_type         = "code"
    response_mode         = "query"
    scope                 = "openid profile email"
    state                 = $state
    nonce                 = $nonce
    code_challenge        = $challenge
    code_challenge_method = "S256"
}.GetEnumerator() |
    ForEach-Object { "{0}={1}" -f [Uri]::EscapeDataString($_.Key), [Uri]::EscapeDataString($_.Value) }
$authorizationUri = "http://localhost:8080/realms/dx-os/protocol/openid-connect/auth?" + (
    $authorizationParameters -join "&"
)

$runtimeRoot = Join-Path $repositoryRoot "data\runtime"
New-Item -ItemType Directory -Path $runtimeRoot -Force | Out-Null
$resolvedAccessTokenOutputPath = $null
if (-not [string]::IsNullOrWhiteSpace($AccessTokenOutputPath)) {
    $resolvedAccessTokenOutputPath = if ([IO.Path]::IsPathRooted($AccessTokenOutputPath)) {
        [IO.Path]::GetFullPath($AccessTokenOutputPath)
    }
    else {
        [IO.Path]::GetFullPath((Join-Path $repositoryRoot $AccessTokenOutputPath))
    }
    $runtimeOutputPrefix = [IO.Path]::GetFullPath($runtimeRoot).TrimEnd("\") + "\"
    if (-not $resolvedAccessTokenOutputPath.StartsWith($runtimeOutputPrefix, [StringComparison]::OrdinalIgnoreCase)) {
        throw "AccessTokenOutputPath must stay inside data\runtime."
    }
}
$temporaryDirectory = [IO.Path]::GetFullPath(
    (Join-Path $runtimeRoot (".oidc-smoke-" + [Guid]::NewGuid().ToString("N")))
)
$runtimePrefix = [IO.Path]::GetFullPath($runtimeRoot).TrimEnd("\") + "\"
if (-not $temporaryDirectory.StartsWith($runtimePrefix, [StringComparison]::OrdinalIgnoreCase)) {
    throw "OIDC temporary directory resolved outside data\runtime."
}
New-Item -ItemType Directory -Path $temporaryDirectory | Out-Null

$cookiePath = Join-Path $temporaryDirectory "cookies.txt"
$loginPagePath = Join-Path $temporaryDirectory "login.html"
$loginHeadersPath = Join-Path $temporaryDirectory "login-headers.txt"
$loginResultPath = Join-Path $temporaryDirectory "login-result.html"
$tokenPath = Join-Path $temporaryDirectory "token.json"
$principalPath = Join-Path $temporaryDirectory "principal.json"
$purchaseRequestBodyPath = Join-Path $temporaryDirectory "purchase-request.json"
$purchaseRequestResultPath = Join-Path $temporaryDirectory "purchase-request-result.json"
$purchaseRequestListPath = Join-Path $temporaryDirectory "purchase-request-list.json"
$purchaseRequestDetailPath = Join-Path $temporaryDirectory "purchase-request-detail.json"

try {
    $authorizeConfig = Join-Path $temporaryDirectory "authorize.curl"
    Write-CurlConfig $authorizeConfig @(
        'silent'
        'show-error'
        ('url = "' + (ConvertTo-CurlConfigValue $authorizationUri) + '"')
        ('cookie-jar = "' + (ConvertTo-CurlConfigValue $cookiePath) + '"')
        ('output = "' + (ConvertTo-CurlConfigValue $loginPagePath) + '"')
        'write-out = "%{http_code}"'
    )
    $authorizeStatus = Invoke-CurlConfig $authorizeConfig
    if ($authorizeStatus -ne "200") {
        throw "OIDC authorization endpoint did not return the login page."
    }

    $loginHtml = Get-Content -Raw -LiteralPath $loginPagePath
    $formTag = [regex]::Match($loginHtml, '(?is)<form\b[^>]*\bid=["'']kc-form-login["''][^>]*>')
    if (-not $formTag.Success) {
        throw "Cannot find the Keycloak login form."
    }
    $actionMatch = [regex]::Match($formTag.Value, '(?is)\baction=["'']([^"'']+)["'']')
    if (-not $actionMatch.Success) {
        throw "Cannot find the Keycloak login action."
    }
    $loginAction = [System.Net.WebUtility]::HtmlDecode($actionMatch.Groups[1].Value)

    $loginConfig = Join-Path $temporaryDirectory "login.curl"
    Write-CurlConfig $loginConfig @(
        'silent'
        'show-error'
        ('url = "' + (ConvertTo-CurlConfigValue $loginAction) + '"')
        ('cookie = "' + (ConvertTo-CurlConfigValue $cookiePath) + '"')
        ('data-urlencode = "username=' + (ConvertTo-CurlConfigValue $username) + '"')
        ('data-urlencode = "password=' + (ConvertTo-CurlConfigValue $password) + '"')
        'data-urlencode = "credentialId="'
        ('dump-header = "' + (ConvertTo-CurlConfigValue $loginHeadersPath) + '"')
        ('output = "' + (ConvertTo-CurlConfigValue $loginResultPath) + '"')
        'write-out = "%{http_code}"'
    )
    $loginStatus = Invoke-CurlConfig $loginConfig
    if ($loginStatus -notin @("302", "303")) {
        throw "Keycloak rejected the local development login."
    }

    $loginHeaders = Get-Content -Raw -LiteralPath $loginHeadersPath
    $locationMatch = [regex]::Match($loginHeaders, '(?im)^location:\s*(\S+)\s*$')
    if (-not $locationMatch.Success) {
        throw "Keycloak login response does not contain a callback location."
    }
    $callbackUri = [Uri]$locationMatch.Groups[1].Value
    if (-not $callbackUri.AbsoluteUri.StartsWith($redirectUri, [StringComparison]::OrdinalIgnoreCase)) {
        $safeCallback = "{0}://{1}{2}" -f $callbackUri.Scheme, $callbackUri.Authority, $callbackUri.AbsolutePath
        throw "Keycloak callback URI is unexpected: $safeCallback"
    }

    $callback = ConvertFrom-QueryString $callbackUri.Query
    if ($callback["state"] -ne $state -or -not $callback.ContainsKey("code")) {
        throw "OIDC callback state/code validation failed."
    }

    $tokenConfig = Join-Path $temporaryDirectory "token.curl"
    Write-CurlConfig $tokenConfig @(
        'silent'
        'show-error'
        'url = "http://localhost:8080/realms/dx-os/protocol/openid-connect/token"'
        'data-urlencode = "grant_type=authorization_code"'
        'data-urlencode = "client_id=dx-web"'
        ('data-urlencode = "redirect_uri=' + (ConvertTo-CurlConfigValue $redirectUri) + '"')
        ('data-urlencode = "code=' + (ConvertTo-CurlConfigValue ([string]$callback["code"])) + '"')
        ('data-urlencode = "code_verifier=' + (ConvertTo-CurlConfigValue $verifier) + '"')
        ('output = "' + (ConvertTo-CurlConfigValue $tokenPath) + '"')
        'write-out = "%{http_code}"'
    )
    $tokenStatus = Invoke-CurlConfig $tokenConfig
    if ($tokenStatus -ne "200") {
        throw "Keycloak rejected the authorization code exchange."
    }
    $tokenBody = Get-Content -Raw -LiteralPath $tokenPath | ConvertFrom-Json
    if (-not $tokenBody.access_token) {
        throw "Keycloak response does not contain an access token."
    }

    $principalConfig = Join-Path $temporaryDirectory "principal.curl"
    Write-CurlConfig $principalConfig @(
        'silent'
        'show-error'
        'url = "http://localhost:8081/api/v1/me"'
        ('header = "Authorization: Bearer ' + (ConvertTo-CurlConfigValue ([string]$tokenBody.access_token)) + '"')
        ('output = "' + (ConvertTo-CurlConfigValue $principalPath) + '"')
        'write-out = "%{http_code}"'
    )
    $principalStatus = Invoke-CurlConfig $principalConfig
    if ($principalStatus -ne "200") {
        throw "Go API rejected the Keycloak access token."
    }
    $principal = Get-Content -Raw -LiteralPath $principalPath | ConvertFrom-Json
    if ($principal.username -ne $username -or $ExpectedRole -notin @($principal.roles)) {
        throw "The verified principal does not contain the expected username/$ExpectedRole role."
    }

    Write-Host "OIDC login smoke test passed."
    Write-Host "Authorization Code + PKCE succeeded; Go API accepted audience dx-api and role $ExpectedRole."

    if ($resolvedAccessTokenOutputPath) {
        New-Item -ItemType Directory -Path (Split-Path -Parent $resolvedAccessTokenOutputPath) -Force | Out-Null
        [IO.File]::WriteAllText(
            $resolvedAccessTokenOutputPath,
            [string]$tokenBody.access_token,
            [Text.UTF8Encoding]::new($false)
        )
        Write-Host "Short-lived access token was written to an ignored runtime file."
    }

    if ($TestProcurement) {
        $purchaseRequestJson = @{
            title      = "Procurement smoke test $([DateTimeOffset]::UtcNow.ToUnixTimeSeconds())"
            reason     = "End-to-end verification of the Procurement MVP create endpoint."
            currency   = "VND"
            costCenter = "CC-GENERAL"
            items      = @(
                @{
                    description = "Development workstation"
                    quantity    = "2"
                    unit        = "unit"
                    unitPrice   = "25000000"
                }
            )
        } | ConvertTo-Json -Depth 5
        [IO.File]::WriteAllText(
            $purchaseRequestBodyPath,
            $purchaseRequestJson,
            (New-Object Text.UTF8Encoding($false))
        )

        $createConfig = Join-Path $temporaryDirectory "purchase-request-create.curl"
        Write-CurlConfig $createConfig @(
            'silent'
            'show-error'
            'url = "http://localhost:8081/api/v1/purchase-requests"'
            'request = "POST"'
            'header = "Content-Type: application/json"'
            ('header = "Authorization: Bearer ' + (ConvertTo-CurlConfigValue ([string]$tokenBody.access_token)) + '"')
            ('data = "@' + (ConvertTo-CurlConfigValue $purchaseRequestBodyPath) + '"')
            ('output = "' + (ConvertTo-CurlConfigValue $purchaseRequestResultPath) + '"')
            'write-out = "%{http_code}"'
        )
        $createStatus = Invoke-CurlConfig $createConfig
        if ($createStatus -ne "201") {
            $problemCode = "unknown"
            try {
                $problem = Get-Content -Raw -LiteralPath $purchaseRequestResultPath | ConvertFrom-Json
                if ($problem.code) {
                    $problemCode = [string]$problem.code
                }
            }
            catch {
                $problemCode = "invalid-response"
            }
            throw "Procurement API rejected the create request (HTTP $createStatus, code $problemCode)."
        }
        $createdRequest = Get-Content -Raw -LiteralPath $purchaseRequestResultPath | ConvertFrom-Json
        if (
            -not $createdRequest.id -or
            $createdRequest.status -ne "DRAFT" -or
            $createdRequest.totalAmount -ne "50000000.0000" -or
            @($createdRequest.items).Count -ne 1
        ) {
            throw "Created purchase request does not contain the expected draft, total, and item."
        }

        $listConfig = Join-Path $temporaryDirectory "purchase-request-list.curl"
        Write-CurlConfig $listConfig @(
            'silent'
            'show-error'
            'url = "http://localhost:8081/api/v1/purchase-requests?page=1&pageSize=20&status=DRAFT"'
            ('header = "Authorization: Bearer ' + (ConvertTo-CurlConfigValue ([string]$tokenBody.access_token)) + '"')
            ('output = "' + (ConvertTo-CurlConfigValue $purchaseRequestListPath) + '"')
            'write-out = "%{http_code}"'
        )
        $listStatus = Invoke-CurlConfig $listConfig
        if ($listStatus -ne "200") {
            throw "Procurement API rejected the list request."
        }
        $requestList = Get-Content -Raw -LiteralPath $purchaseRequestListPath | ConvertFrom-Json
        if ($createdRequest.id -notin @($requestList.items | ForEach-Object { $_.id })) {
            throw "Created purchase request is missing from the employee-scoped list."
        }

        $detailConfig = Join-Path $temporaryDirectory "purchase-request-detail.curl"
        Write-CurlConfig $detailConfig @(
            'silent'
            'show-error'
            ('url = "http://localhost:8081/api/v1/purchase-requests/' + (ConvertTo-CurlConfigValue ([string]$createdRequest.id)) + '"')
            ('header = "Authorization: Bearer ' + (ConvertTo-CurlConfigValue ([string]$tokenBody.access_token)) + '"')
            ('output = "' + (ConvertTo-CurlConfigValue $purchaseRequestDetailPath) + '"')
            'write-out = "%{http_code}"'
        )
        $detailStatus = Invoke-CurlConfig $detailConfig
        if ($detailStatus -ne "200") {
            throw "Procurement API rejected the detail request."
        }
        $requestDetail = Get-Content -Raw -LiteralPath $purchaseRequestDetailPath | ConvertFrom-Json
        if ($requestDetail.id -ne $createdRequest.id -or @($requestDetail.items).Count -ne 1) {
            throw "Purchase request detail does not match the created request."
        }

        Write-Host "Procurement MVP smoke test passed."
        Write-Host "Create, employee-scoped list, and detail endpoints returned the expected DRAFT request."
    }
}
finally {
    $password = $null
    $credentials.Clear()
    if (Test-Path -LiteralPath $temporaryDirectory) {
        Get-ChildItem -LiteralPath $temporaryDirectory -File |
            ForEach-Object { Remove-Item -LiteralPath $_.FullName -Force }
        Remove-Item -LiteralPath $temporaryDirectory -Force
    }
}
