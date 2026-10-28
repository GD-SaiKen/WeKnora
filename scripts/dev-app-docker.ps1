# Run WeKnora API in Docker while DB/Redis use the host (ssh tunnel on 127.0.0.1).
# Requires: docker pull wechatopenai/weknora-app:latest, ssh -N tencent-db, docreader/qdrant as needed.

$ErrorActionPreference = "Stop"
$ProjectRoot = Split-Path -Parent $PSScriptRoot
Set-Location $ProjectRoot

function Import-DotEnvFile {
    param([string]$Path)
    if (-not (Test-Path -LiteralPath $Path)) { return }
    foreach ($rawLine in Get-Content -LiteralPath $Path -Encoding UTF8) {
        $line = $rawLine.Trim()
        if ($line.Length -eq 0 -or $line.StartsWith("#")) { continue }
        $eq = $line.IndexOf("=")
        if ($eq -lt 1) { continue }
        $name = $line.Substring(0, $eq).Trim()
        $value = $line.Substring($eq + 1).Trim()
        if (
            ($value.StartsWith('"') -and $value.EndsWith('"')) -or
            ($value.StartsWith("'") -and $value.EndsWith("'"))
        ) {
            $value = $value.Substring(1, $value.Length - 2)
        }
        Set-Item -Path "Env:$name" -Value $value
    }
}

Import-DotEnvFile (Join-Path $ProjectRoot ".env")
Import-DotEnvFile (Join-Path $ProjectRoot ".env.local")

$hostDb = if ($env:DB_HOST -eq "127.0.0.1" -or $env:DB_HOST -eq "localhost") { "host.docker.internal" } else { $env:DB_HOST }
$redis = $env:REDIS_ADDR
if ($redis -match "^127\.0\.0\.1:") { $redis = $redis -replace "^127\.0\.0\.1:", "host.docker.internal:" }
$docreader = $env:DOCREADER_ADDR
if ($docreader -match "^127\.0\.0\.1:") { $docreader = $docreader -replace "^127\.0\.0\.1", "host.docker.internal" }
$qdrant = $env:QDRANT_HOST
if ($qdrant -eq "127.0.0.1" -or $qdrant -eq "localhost") { $qdrant = "host.docker.internal" }

$name = "WeKnora-app-dev"
docker rm -f $name 2>$null | Out-Null

$args = @(
    "run", "-d", "--name", $name, "-p", "8080:8080",
    "--env-file", (Join-Path $ProjectRoot ".env"),
    "-e", "DB_HOST=$hostDb",
    "-e", "DB_USER=$($env:DB_USER)",
    "-e", "DB_PASSWORD=$($env:DB_PASSWORD)",
    "-e", "DB_NAME=$($env:DB_NAME)",
    "-e", "REDIS_ADDR=$redis",
    "-e", "REDIS_PASSWORD=$($env:REDIS_PASSWORD)",
    "-e", "RETRIEVE_DRIVER=$($env:RETRIEVE_DRIVER)",
    "-e", "QDRANT_HOST=$qdrant",
    "-e", "DOCREADER_ADDR=$docreader",
    "-e", "LANGFUSE_ENABLED=false",
    "-v", "$(Join-Path $ProjectRoot 'config\config.yaml'):/app/config/config.yaml:ro",
    "wechatopenai/weknora-app:latest"
)

Write-Host "[INFO] Starting container $name (API http://localhost:8080) ..."
& docker @args
Write-Host "[INFO] Logs: docker logs -f $name"
