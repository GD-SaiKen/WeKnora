#!/bin/bash
# Full-stack Docker dev (Air + Vite HMR). See docker-compose.dev-hot.yml.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_ROOT"

CMD="${1:-up}"
shift || true

COMPOSE=(docker compose -f docker-compose.dev.yml -f docker-compose.dev-hot.yml)

if [ ! -f .env ]; then
  if [ -f .env.example ]; then
    cp .env.example .env
    echo "[INFO] Created .env from .env.example"
  else
    echo "[ERROR] Missing .env" >&2
    exit 1
  fi
fi

case "$CMD" in
  up)
    # Hot-stack host ports: avoid the local SSH tunnel (5432/6379) and other
    # projects' vite on 5173. In-container services still use 5432/6379/5173.
    export DB_PORT="${DB_PORT:-15432}"
    export REDIS_PORT="${REDIS_PORT:-16379}"
    export FRONTEND_DEV_PORT="${FRONTEND_DEV_PORT:-5174}"
    echo "[INFO] Frontend: http://localhost:${FRONTEND_DEV_PORT}"
    echo "[INFO] Backend:  http://localhost:${APP_PORT:-8080}"
    "${COMPOSE[@]}" up -d --build "$@"
    ;;
  build)
    "${COMPOSE[@]}" build "$@"
    ;;
  down)
    "${COMPOSE[@]}" down "$@"
    ;;
  logs)
    if [ $# -eq 0 ]; then
      set -- app-dev frontend-dev docreader-dev
    fi
    "${COMPOSE[@]}" logs -f "$@"
    ;;
  status)
    "${COMPOSE[@]}" ps "$@"
    ;;
  restart)
    "${COMPOSE[@]}" restart app-dev frontend-dev docreader-dev "$@"
    ;;
  *)
    echo "Usage: $0 {up|down|build|logs|status|restart} [compose args...]" >&2
    exit 1
    ;;
esac
