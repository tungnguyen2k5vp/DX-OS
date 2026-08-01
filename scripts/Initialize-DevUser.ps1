[CmdletBinding()]
param(
    [string]$Username = "employee.demo",
    [string]$Email,
    [ValidateSet("employee", "department_manager", "finance", "auditor", "ai_operator", "dx_admin")]
    [string]$Role = "employee",
    [string]$CredentialsPath = "data\runtime\dev-user.txt"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Assert-ExternalCommand {
    param([Parameter(Mandatory)][string]$Message)

    if ($LASTEXITCODE -ne 0) {
        throw $Message
    }
}

function New-RandomPassword {
    $bytes = New-Object byte[] 24
    $generator = [System.Security.Cryptography.RandomNumberGenerator]::Create()
    try {
        $generator.GetBytes($bytes)
    }
    finally {
        $generator.Dispose()
    }

    return ([Convert]::ToBase64String($bytes)).TrimEnd("=").Replace("+", "-").Replace("/", "_") + "!9a"
}

$repositoryRoot = Split-Path -Parent $PSScriptRoot
$credentialsDirectory = Join-Path $repositoryRoot "data\runtime"
$resolvedCredentialsPath = if ([IO.Path]::IsPathRooted($CredentialsPath)) {
    [IO.Path]::GetFullPath($CredentialsPath)
}
else {
    [IO.Path]::GetFullPath((Join-Path $repositoryRoot $CredentialsPath))
}
$runtimePrefix = [IO.Path]::GetFullPath($credentialsDirectory).TrimEnd("\") + "\"
if (-not $resolvedCredentialsPath.StartsWith($runtimePrefix, [StringComparison]::OrdinalIgnoreCase)) {
    throw "CredentialsPath must stay inside data\runtime."
}
$adminConfig = "/tmp/dxos-dev-user-kcadm.config"
if ([string]::IsNullOrWhiteSpace($Email)) {
    $Email = "$Username@dx-os.local"
}

Push-Location $repositoryRoot
try {
    $null = & docker compose exec -T keycloak sh -ec '/opt/keycloak/bin/kcadm.sh config credentials --config /tmp/dxos-dev-user-kcadm.config --server http://localhost:8080 --realm master --user "$KC_BOOTSTRAP_ADMIN_USERNAME" --password "$KC_BOOTSTRAP_ADMIN_PASSWORD"'
    Assert-ExternalCommand "Cannot authenticate to the local Keycloak admin API."

    $usersJson = @(
        & docker compose exec -T keycloak /opt/keycloak/bin/kcadm.sh get users `
            -r dx-os `
            --config $adminConfig `
            -q "username=$Username" `
            -q exact=true
    )
    Assert-ExternalCommand "Cannot query the DX-OS realm."
    $users = @(
        (($usersJson -join "`n") | ConvertFrom-Json) |
            Where-Object { $null -ne $_ }
    )

    if ($users.Count -eq 0) {
        $null = & docker compose exec -T keycloak /opt/keycloak/bin/kcadm.sh create users `
            -r dx-os `
             --config $adminConfig `
             -s "username=$Username" `
             -s "email=$Email" `
             -s enabled=true `
             -s emailVerified=true `
             -s firstName=Demo `
            -s lastName=Employee
        Assert-ExternalCommand "Cannot create the development user."
    }

    $createdUsersJson = @(
        & docker compose exec -T keycloak /opt/keycloak/bin/kcadm.sh get users `
            -r dx-os `
            --config $adminConfig `
            -q "username=$Username" `
            -q exact=true
    )
    Assert-ExternalCommand "Cannot verify the development user."
    $createdUsers = @(
        (($createdUsersJson -join "`n") | ConvertFrom-Json) |
            Where-Object { $null -ne $_ }
    )
    if ($createdUsers.Count -ne 1) {
        throw "Development user was not created exactly once."
    }

    $userId = [string]$createdUsers[0].id
    $null = & docker compose exec -T keycloak /opt/keycloak/bin/kcadm.sh update "users/$userId" `
        -r dx-os `
        --config $adminConfig `
        -s "email=$Email" `
        -s emailVerified=true `
        -s firstName=Demo `
        -s lastName=Employee `
        -s 'requiredActions=[]'
    Assert-ExternalCommand "Cannot normalize the development user profile."

    $password = New-RandomPassword
    $null = & docker compose exec -T keycloak /opt/keycloak/bin/kcadm.sh set-password `
        -r dx-os `
        --config $adminConfig `
        --username $Username `
        --new-password $password
    Assert-ExternalCommand "Cannot set the development password."

    $null = & docker compose exec -T keycloak /opt/keycloak/bin/kcadm.sh add-roles `
        -r dx-os `
        --config $adminConfig `
        --uusername $Username `
        --rolename $Role
    Assert-ExternalCommand "Cannot assign the $Role role."

    New-Item -ItemType Directory -Path (Split-Path -Parent $resolvedCredentialsPath) -Force | Out-Null
    @(
        "DX-OS local development user"
        "username=$Username"
        "password=$password"
        "role=$Role"
        "Local development only. Regenerate with scripts\Initialize-DevUser.ps1."
    ) | Set-Content -LiteralPath $resolvedCredentialsPath -Encoding UTF8

    Write-Host "Development user is ready: $Username"
    Write-Host "Assigned realm role: $Role"
    Write-Host "Local credentials were written to: $resolvedCredentialsPath"
    Write-Host "This file is ignored by Git. Do not share or commit it."
}
finally {
    & docker compose exec -T keycloak rm -f $adminConfig 2>$null
    Pop-Location
}
