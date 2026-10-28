# WeKnora backend - local dev on Windows (no Bash required).
# Loads .env + .env.local, applies the same host overrides as scripts/dev.sh app, then go run.

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

function Ensure-Mingw64Toolchain {
    $candidates = @(
        $env:WEKNORA_MINGW_BIN,
        "G:\app\dev\gcc\mingw64\bin",
        "C:\msys64\ucrt64\bin",
        "C:\msys64\mingw64\bin"
    ) | Where-Object { $_ -and (Test-Path (Join-Path $_ "gcc.exe")) }

    if (@($candidates).Count -eq 0) {
        if ($env:PATH -match 'w64devkit') {
            Write-Warning 'CGO: remove w64devkit from PATH; use MSYS2 ucrt64 gcc + sqlite dev package.'
        }
        return
    }

    $bin = @($candidates | Select-Object -First 1)
    $pathParts = $env:PATH -split ';' | Where-Object { $_ -and ($_ -notmatch 'w64devkit') }
    $env:PATH = ($bin + ';' + ($pathParts -join ';')).TrimEnd(';')
    $gcc = Join-Path $bin 'gcc.exe'
    $gpp = Join-Path $bin 'g++.exe'
    if (-not (Test-Path -LiteralPath $gcc)) {
        Write-Error "gcc.exe not found under $bin"
    }
    if (-not (Test-Path -LiteralPath $gpp)) {
        Write-Error @"
g++.exe not found under $bin (required by gojieba).
Install MSYS2 UCRT64: pacman -S mingw-w64-ucrt-x86_64-gcc mingw-w64-ucrt-x86_64-sqlite
Set WEKNORA_MINGW_BIN=C:\msys64\ucrt64\bin, then: pnpm run setup:cgo
"@
    }
    $env:CC = $gcc
    $env:CXX = $gpp
    Write-Host "[INFO] CGO toolchain: $bin"

    $machine = & $gcc -dumpmachine 2>$null
    if ($machine -notmatch 'x86_64') {
        Write-Error "gcc -dumpmachine=$machine (need x86_64 MinGW). Check WEKNORA_MINGW_BIN / PATH."
    }

    $includeRoot = Split-Path -Parent $bin
    $sqliteHeader = Join-Path $includeRoot 'include\sqlite3.h'
    $libSqlite = Join-Path $includeRoot 'lib\libsqlite3.a'
    if (-not (Test-Path -LiteralPath $sqliteHeader)) {
        Write-Error "Missing sqlite3.h. Run once: pnpm run setup:cgo (or MSYS2: pacman -S mingw-w64-ucrt-x86_64-sqlite)"
    }
    $libDir = Join-Path $includeRoot 'lib'
    if (Test-Path -LiteralPath $libSqlite) {
        $libDirFwd = $libDir -replace '\\', '/'
        $env:CGO_LDFLAGS = "-L$libDirFwd -lsqlite3"
    }
    $includeFwd = (Join-Path $includeRoot 'include') -replace '\\', '/'
    if ($env:CGO_CFLAGS) {
        $env:CGO_CFLAGS = "$env:CGO_CFLAGS -I$includeFwd"
    } else {
        $env:CGO_CFLAGS = "-I$includeFwd"
    }
}

if (-not (Test-Path -LiteralPath (Join-Path $ProjectRoot ".env"))) {
    Write-Error "Missing .env in project root. Copy from .env.example: cp .env.example .env"
}

Import-DotEnvFile (Join-Path $ProjectRoot ".env")
Import-DotEnvFile (Join-Path $ProjectRoot ".env.local")

if ($env:DEV_REMOTE_HOST) {
    Write-Host "[INFO] Remote dev: DEV_REMOTE_HOST=$($env:DEV_REMOTE_HOST) (keeping DB/Redis from env files)"
} else {
    $env:DB_HOST = "127.0.0.1"
    $env:DOCREADER_ADDR = "127.0.0.1:50051"
    $env:MINIO_ENDPOINT = "127.0.0.1:9000"
    $env:REDIS_ADDR = "127.0.0.1:6379"
    $env:MILVUS_ADDRESS = "127.0.0.1:19530"
    $env:NEO4J_URI = "bolt://127.0.0.1:7687"
    $env:QDRANT_HOST = "127.0.0.1"
}

if (-not $env:DOCREADER_TRANSPORT) { $env:DOCREADER_TRANSPORT = "grpc" }

if (-not $env:LOCAL_STORAGE_BASE_DIR -or $env:LOCAL_STORAGE_BASE_DIR -eq "/data/files") {
    $env:LOCAL_STORAGE_BASE_DIR = Join-Path $ProjectRoot ".local-data\files"
}
New-Item -ItemType Directory -Force -Path $env:LOCAL_STORAGE_BASE_DIR | Out-Null

if (-not $env:DB_DRIVER) {
    Write-Error "DB_DRIVER is not set. Check .env"
}

$versionPath = Join-Path $ProjectRoot "VERSION"
$version = "unknown"
if (Test-Path -LiteralPath $versionPath) {
    $version = (Get-Content -LiteralPath $versionPath -Raw).Trim()
}
$edition = if ($env:EDITION) { $env:EDITION } else { "standard" }
$commit = "unknown"
try {
    $commit = (git -C $ProjectRoot rev-parse --short HEAD 2>$null)
    if (-not $commit) { $commit = "unknown" }
} catch { }
$buildTime = (Get-Date).ToUniversalTime().ToString("yyyy-MM-dd HH:mm:ss") + " UTC"
$goVersion = "unknown"
try { $goVersion = (go version 2>$null) } catch { }

$ldflags = @(
    "-X 'github.com/Tencent/WeKnora/internal/handler.Version=$version'"
    "-X 'github.com/Tencent/WeKnora/internal/handler.Edition=$edition'"
    "-X 'github.com/Tencent/WeKnora/internal/handler.CommitID=$commit'"
    "-X 'github.com/Tencent/WeKnora/internal/handler.BuildTime=$buildTime'"
    "-X 'github.com/Tencent/WeKnora/internal/handler.GoVersion=$goVersion'"
    "-X 'google.golang.org/protobuf/reflect/protoregistry.conflictPolicy=warn'"
) -join " "

$appPort = if ($env:APP_PORT) { $env:APP_PORT } else { "8080" }
$dbPort = if ($env:DB_PORT) { $env:DB_PORT } else { "5432" }
Write-Host "[INFO] Starting WeKnora API on http://localhost:$appPort"
Write-Host "[INFO] Database: $($env:DB_HOST):$dbPort / $($env:DB_NAME)"

Ensure-Mingw64Toolchain

$goArgs = @("run", "-ldflags=$ldflags", "./cmd/server")
if ($env:GO_BUILD_TAGS) {
    $goArgs = @("run", "-tags", $env:GO_BUILD_TAGS, "-ldflags=$ldflags", "./cmd/server")
}

& go @goArgs
exit $LASTEXITCODE
