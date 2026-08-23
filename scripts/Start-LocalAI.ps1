[CmdletBinding()]
param(
    [switch]$CpuOnly
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repositoryRoot = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot ".."))
Push-Location $repositoryRoot
try {
    $composeArguments = @("compose", "-f", "docker-compose.yml")
    if (-not $CpuOnly) {
        $composeArguments += @("-f", "docker-compose.gpu.yml")
    }
    $profileArguments = @(
        "--profile", "foundation",
        "--profile", "application"
    )

    & docker @composeArguments @profileArguments up -d ollama
    if ($LASTEXITCODE -ne 0) {
        throw "Docker could not start the Ollama service."
    }

    & docker @composeArguments @profileArguments run --rm ollama-init
    if ($LASTEXITCODE -ne 0) {
        throw "Docker could not download or verify the configured model."
    }

    & docker compose exec -T ollama ollama list
    if ($LASTEXITCODE -ne 0) {
        throw "Ollama is running but the model list could not be read."
    }

    Write-Host "Ollama and the configured model are ready inside Docker."
}
finally {
    Pop-Location
}
