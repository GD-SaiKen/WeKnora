#!/bin/bash
set -euo pipefail

cd /app
export PATH="/app/.venv/bin:${PATH}"

POLL="${WATCHFILES_FORCE_POLLING:-${CHOKIDAR_USEPOLLING:-false}}"
WATCHFILES_ARGS=""
if [ "$POLL" = "true" ] || [ "$POLL" = "1" ]; then
  export WATCHFILES_FORCE_POLLING=1
fi

exec /app/.venv/bin/python -m docreader.container_entrypoint \
  /app/.venv/bin/watchfiles \
  --filter python \
  '/app/.venv/bin/python -m docreader.main' \
  docreader
