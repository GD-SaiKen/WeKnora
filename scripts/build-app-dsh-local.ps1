# Build a local weknora-app image that includes dsh-work integration code.
#
# Apt mirror: deb.debian.org CDN often returns 502 from China. By default this
# script uses APK_MIRROR_ARG from .env / .env.local (typically mirrors.tencent.com).
# Force official Debian only:
#   $env:WEKNORA_DOCKER_APT_MIRROR = ""
#   pnpm run build:app:dsh-local

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

$tag = if ($env:WEKNORA_DEV_APP_IMAGE) { $env:WEKNORA_DEV_APP_IMAGE } else { "weknora-app-dsh-local:latest" }

$aptMirror = $env:WEKNORA_DOCKER_APT_MIRROR
if ($null -eq $aptMirror) {
    $aptMirror = if ($env:APK_MIRROR_ARG) { $env:APK_MIRROR_ARG } else { "mirrors.tencent.com" }
}

Write-Host "[INFO] Building $tag (APK_MIRROR_ARG=$aptMirror) ..."
docker build -f docker/Dockerfile.app --build-arg "APK_MIRROR_ARG=$aptMirror" -t $tag .
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
Write-Host "[INFO] Done. Start with: `$env:WEKNORA_DEV_APP_IMAGE='$tag'; pnpm run dev:app:docker"
