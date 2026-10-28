# One-time (or repeat) setup: SQLite headers/libs for CGO on Windows MinGW.
# Full g++ is still required (gojieba). Install MSYS2 UCRT64:
#   pacman -S mingw-w64-ucrt-x86_64-gcc mingw-w64-ucrt-x86_64-sqlite
# Then set WEKNORA_MINGW_BIN=C:\msys64\ucrt64\bin

$ErrorActionPreference = "Stop"
$ProjectRoot = Split-Path -Parent $PSScriptRoot
$DepsDir = Join-Path $ProjectRoot ".local-deps\sqlite-amalgamation"

$mingwBin = $env:WEKNORA_MINGW_BIN
if (-not $mingwBin) { $mingwBin = "G:\app\dev\gcc\mingw64\bin" }
if (-not (Test-Path (Join-Path $mingwBin "gcc.exe"))) {
    Write-Error "gcc not found in $mingwBin. Set WEKNORA_MINGW_BIN to MSYS2 ucrt64\bin."
}

$gpp = Join-Path $mingwBin "g++.exe"
if (-not (Test-Path -LiteralPath $gpp)) {
    Write-Error @"
g++.exe not found in $mingwBin (gojieba needs C++).
Your MinGW install looks incomplete. Recommended:
  1. Install MSYS2 from https://www.msys2.org/
  2. In UCRT64 shell: pacman -S mingw-w64-ucrt-x86_64-gcc mingw-w64-ucrt-x86_64-sqlite
  3. setx WEKNORA_MINGW_BIN C:\msys64\ucrt64\bin
  4. Re-open terminal and run: pnpm run setup:cgo
"@
}

$mingwRoot = Split-Path -Parent $mingwBin
$includeDir = Join-Path $mingwRoot "include"
$libDir = Join-Path $mingwRoot "lib"
New-Item -ItemType Directory -Force -Path $includeDir, $libDir, $DepsDir | Out-Null

if (Test-Path (Join-Path $includeDir "sqlite3.h")) {
    Write-Host "[OK] sqlite3.h already in $includeDir"
} else {
    $zipUrl = "https://www.sqlite.org/2024/sqlite-amalgamation-3460100.zip"
    $zipPath = Join-Path $DepsDir "sqlite-amalgamation.zip"
    Write-Host "[INFO] Downloading SQLite amalgamation..."
    Invoke-WebRequest -Uri $zipUrl -OutFile $zipPath -UseBasicParsing
    Expand-Archive -Path $zipPath -DestinationPath $DepsDir -Force
    $srcDir = Get-ChildItem -Path $DepsDir -Directory | Where-Object { $_.Name -like "sqlite-amalgamation-*" } | Select-Object -First 1
    if (-not $srcDir) { Write-Error "Unexpected zip layout under $DepsDir" }
    Copy-Item -Path (Join-Path $srcDir.FullName "sqlite3.h") -Destination $includeDir -Force
    Copy-Item -Path (Join-Path $srcDir.FullName "sqlite3ext.h") -Destination $includeDir -Force -ErrorAction SilentlyContinue
    $sqliteC = Join-Path $srcDir.FullName "sqlite3.c"
    Write-Host "[INFO] Building libsqlite3.a (may take a minute)..."
    $env:PATH = "$mingwBin;$env:PATH"
    Push-Location $srcDir.FullName
    & gcc -c sqlite3.c -o sqlite3.o
    & ar rcs (Join-Path $libDir "libsqlite3.a") sqlite3.o
    Pop-Location
    Write-Host "[OK] Installed sqlite3.h and libsqlite3.a under $mingwRoot"
}

Write-Host "[OK] CGO deps ready. Run: pnpm run dev:app"
