[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Assert-ExternalCommand {
    param([Parameter(Mandatory)][string]$Message)

    if ($LASTEXITCODE -ne 0) {
        throw $Message
    }
}

function Assert-Condition {
    param(
        [Parameter(Mandatory)][bool]$Condition,
        [Parameter(Mandatory)][string]$Message
    )

    if (-not $Condition) {
        throw $Message
    }
}

$repositoryRoot = Split-Path -Parent $PSScriptRoot
Push-Location $repositoryRoot

$keycloakAdminConfig = "/tmp/dxos-foundation-kcadm.config"

try {
    & docker compose --profile foundation config --quiet
    Assert-ExternalCommand "Docker Compose configuration is invalid."

    $postgresUser = (& docker compose exec -T postgres printenv POSTGRES_USER).Trim()
    Assert-ExternalCommand "Cannot read the PostgreSQL bootstrap user."

    $databaseNames = @(
        & docker compose exec -T postgres psql `
            --username $postgresUser `
            --dbname postgres `
            --tuples-only `
            --no-align `
            --command "SELECT datname FROM pg_database ORDER BY datname;"
    )
    Assert-ExternalCommand "Cannot query PostgreSQL."

    $expectedDatabases = @("dxos", "keycloak", "nextcloud", "metabase")
    $missingDatabases = @($expectedDatabases | Where-Object { $_ -notin $databaseNames })
    Assert-Condition ($missingDatabases.Count -eq 0) "Missing databases: $($missingDatabases -join ', ')"

    $databaseRoles = @(
        & docker compose exec -T postgres psql `
            --username $postgresUser `
            --dbname postgres `
            --tuples-only `
            --no-align `
            --command "SELECT rolname FROM pg_roles WHERE rolcanlogin ORDER BY rolname;"
    )
    Assert-ExternalCommand "Cannot query PostgreSQL roles."

    $expectedDatabaseRoles = @("dxos_app", "keycloak", "nextcloud", "metabase")
    $missingDatabaseRoles = @($expectedDatabaseRoles | Where-Object { $_ -notin $databaseRoles })
    Assert-Condition ($missingDatabaseRoles.Count -eq 0) "Missing database roles: $($missingDatabaseRoles -join ', ')"

    $discovery = Invoke-RestMethod `
        -Uri "http://localhost:8080/realms/dx-os/.well-known/openid-configuration" `
        -TimeoutSec 10
    Assert-Condition ($discovery.issuer -eq "http://localhost:8080/realms/dx-os") "Unexpected OIDC issuer."
    Assert-Condition ("S256" -in $discovery.code_challenge_methods_supported) "OIDC discovery does not advertise PKCE S256."

    $null = & docker compose exec -T keycloak sh -ec '/opt/keycloak/bin/kcadm.sh config credentials --config /tmp/dxos-foundation-kcadm.config --server http://localhost:8080 --realm master --user "$KC_BOOTSTRAP_ADMIN_USERNAME" --password "$KC_BOOTSTRAP_ADMIN_PASSWORD"'
    Assert-ExternalCommand "Cannot authenticate to the local Keycloak admin API."

    $rolesJson = @(
        & docker compose exec -T keycloak /opt/keycloak/bin/kcadm.sh get roles `
            -r dx-os `
            --config $keycloakAdminConfig `
            --fields name
    )
    Assert-ExternalCommand "Cannot query Keycloak realm roles."
    $realmRoleNames = @((($rolesJson -join "`n") | ConvertFrom-Json) | ForEach-Object name)
    $expectedRealmRoles = @(
        "employee",
        "department_manager",
        "finance",
        "dx_admin",
        "auditor",
        "ai_operator"
    )
    $missingRealmRoles = @($expectedRealmRoles | Where-Object { $_ -notin $realmRoleNames })
    Assert-Condition ($missingRealmRoles.Count -eq 0) "Missing realm roles: $($missingRealmRoles -join ', ')"

    $webClientJson = @(
        & docker compose exec -T keycloak /opt/keycloak/bin/kcadm.sh get clients `
            -r dx-os `
            --config $keycloakAdminConfig `
            -q clientId=dx-web
    )
    Assert-ExternalCommand "Cannot query Keycloak client dx-web."
    $webClient = @(($webClientJson -join "`n") | ConvertFrom-Json)

    $apiClientJson = @(
        & docker compose exec -T keycloak /opt/keycloak/bin/kcadm.sh get clients `
            -r dx-os `
            --config $keycloakAdminConfig `
            -q clientId=dx-api
    )
    Assert-ExternalCommand "Cannot query Keycloak client dx-api."
    $apiClient = @(($apiClientJson -join "`n") | ConvertFrom-Json)

    Assert-Condition ($webClient.Count -eq 1) "Keycloak client dx-web is missing or duplicated."
    Assert-Condition ($apiClient.Count -eq 1) "Keycloak client dx-api is missing or duplicated."
    Assert-Condition ([bool]$webClient[0].publicClient) "dx-web must be a public client."
    Assert-Condition ([bool]$webClient[0].standardFlowEnabled) "dx-web must enable Authorization Code flow."
    Assert-Condition (-not [bool]$webClient[0].directAccessGrantsEnabled) "dx-web must disable Direct Access Grants."
    Assert-Condition ($webClient[0].attributes.'pkce.code.challenge.method' -eq "S256") "dx-web must require PKCE S256."
    Assert-Condition ([bool]$apiClient[0].bearerOnly) "dx-api must be a bearer-only client."

    $audienceMappers = @(
        $webClient[0].protocolMappers |
            Where-Object {
                $_.protocolMapper -eq "oidc-audience-mapper" -and
                $_.config.'included.client.audience' -eq "dx-api"
            }
    )
    Assert-Condition ($audienceMappers.Count -eq 1) "dx-web must map the dx-api audience into access tokens."

    Write-Host "Foundation smoke test passed."
    Write-Host "PostgreSQL: 4 service databases and 4 login roles."
    Write-Host "Keycloak: realm dx-os, 6 business roles, clients dx-web and dx-api."
    Write-Host "OIDC: discovery is reachable and PKCE S256 is advertised."
}
finally {
    & docker compose exec -T keycloak rm -f $keycloakAdminConfig 2>$null
    Pop-Location
}
