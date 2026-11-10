#!/bin/sh
set -e

cd /app/frontend

if [ ! -f node_modules/.package-lock.json ] && [ ! -d node_modules/vite ]; then
  echo "[frontend-dev] Installing npm dependencies..."
  npm ci
fi

exec npm run dev -- --host 0.0.0.0
