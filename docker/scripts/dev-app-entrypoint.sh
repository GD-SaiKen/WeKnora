#!/bin/bash
set -euo pipefail

cd /app

mkdir -p "${LOCAL_STORAGE_BASE_DIR:-/data/files}"

export GIN_MODE="${GIN_MODE:-debug}"
export CGO_CFLAGS="${CGO_CFLAGS:--Wno-deprecated-declarations -Wno-gnu-folding-constant}"

# In compose, service DNS names override .env values meant for host-local dev.
export DB_HOST="${DB_HOST:-postgres}"
export REDIS_ADDR="${REDIS_ADDR:-redis:6379}"
export DOCREADER_ADDR="${DOCREADER_ADDR:-docreader:50051}"
export DOCREADER_TRANSPORT="${DOCREADER_TRANSPORT:-grpc}"

go mod download

exec "$@"
