# WeKnora full-stack Docker dev (Air + Vite HMR + docreader watchfiles).
# Usage:
#   pnpm run dev:docker
#   pnpm run dev:docker -- down
#   pnpm run dev:docker -- logs -f app-dev

param(
    [Parameter(Position = 0)]
    [ValidateSet("up", "down", "logs", "status", "build", "restart")]
    [string]$Command = "up",
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$Extra
)

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

function Get-DockerComposeArgs {
    param([string[]]$SubCommand)
    $fileArgs = @(
        "-f", (Join-Path $ProjectRoot "docker-compose.dev.yml"),
        "-f", (Join-Path $ProjectRoot "docker-compose.dev-hot.yml")
    )
    return @("compose") + $fileArgs + $SubCommand
}

function Invoke-DockerCompose {
    param([string[]]$SubCommand)
    $dockerArgs = Get-DockerComposeArgs -SubCommand $SubCommand
    & docker @dockerArgs
    if ($LASTEXITCODE -ne 0) {
        exit $LASTEXITCODE
    }
}

if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
    Write-Error "Docker is not installed or not on PATH."
}

docker compose version 2>$null | Out-Null
if ($LASTEXITCODE -ne 0) {
    Write-Error "Docker Compose v2 (docker compose) is required."
}

$envPath = Join-Path $ProjectRoot ".env"
$envExamplePath = Join-Path $ProjectRoot ".env.example"
if (-not (Test-Path -LiteralPath $envPath)) {
    if (Test-Path -LiteralPath $envExamplePath) {
        Copy-Item -LiteralPath $envExamplePath -Destination $envPath
        Write-Host "[INFO] Created .env from .env.example"
    } else {
        Write-Error "Missing .env - copy from .env.example first."
    }
}

Import-DotEnvFile (Join-Path $ProjectRoot ".env")
Import-DotEnvFile (Join-Path $ProjectRoot ".env.local")

# 热更新栈的宿主机端口固定覆盖：避开本机 SSH 隧道占用的 5432/6379，以及
# 本机其他项目 vite 占用的 5173（容器内互连仍是 5432/6379/5173，见
# docker-compose.dev-hot.yml 中 app-dev 的钉死环境变量）。需要 Langfuse 时
# 传参：pnpm run dev:docker -- --profile langfuse
$env:DB_PORT = "15432"
$env:REDIS_PORT = "16379"
if (-not $env:FRONTEND_DEV_PORT) { $env:FRONTEND_DEV_PORT = "5174" }

$devServices = @("app-dev", "frontend-dev", "docreader-dev")

switch ($Command) {
    "up" {
        $frontendPort = if ($env:FRONTEND_DEV_PORT) { $env:FRONTEND_DEV_PORT } else { "5173" }
        $appPort = if ($env:APP_PORT) { $env:APP_PORT } else { "8080" }
        Write-Host "[INFO] Starting Docker dev stack (infra + app + frontend + docreader)..."
        Write-Host "[INFO] Frontend: http://localhost:$frontendPort"
        Write-Host "[INFO] Backend:  http://localhost:$appPort"
        Invoke-DockerCompose -SubCommand (@("up", "-d", "--build") + $Extra)
        Write-Host "[INFO] Logs: pnpm run dev:docker -- logs -f"
    }
    "build" {
        Invoke-DockerCompose -SubCommand (@("build") + $Extra)
    }
    "down" {
        Invoke-DockerCompose -SubCommand (@("down") + $Extra)
    }
    "logs" {
        $sub = @("logs") + $Extra
        if ($Extra.Count -eq 0) {
            $sub = @("logs") + $devServices
        }
        Invoke-DockerCompose -SubCommand $sub
    }
    "status" {
        Invoke-DockerCompose -SubCommand @("ps")
    }
    "restart" {
        Invoke-DockerCompose -SubCommand (@("restart") + $devServices + $Extra)
    }
}
