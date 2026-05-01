#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPOSE_DIR="$(dirname "$SCRIPT_DIR")"
FE_DIR="$(cd "$COMPOSE_DIR/../Taipei-City-Dashboard-FE" && pwd)"
SERVICE="dashboard-fe"
NODE_IMAGE="node:21.6.0-alpine3.18"

cd "$COMPOSE_DIR"

# 偵測 deps 是否需要重裝 (package.json 改 or node_modules 不在)
NEED_INSTALL=0
if [ ! -d "$FE_DIR/node_modules" ]; then
  NEED_INSTALL=1
elif [ "$FE_DIR/package.json" -nt "$FE_DIR/node_modules" ] || \
     [ "$FE_DIR/package-lock.json" -nt "$FE_DIR/node_modules" ]; then
  NEED_INSTALL=1
fi

if [ "$NEED_INSTALL" -eq 1 ]; then
  echo "[restart_fe] npm install (deps changed or missing)"
  docker run --rm -v "$FE_DIR:/app" -w /app "$NODE_IMAGE" npm install
fi

echo "[restart_fe] recreate $SERVICE"
docker compose up -d --force-recreate --no-deps "$SERVICE"

echo "[restart_fe] status"
docker compose ps "$SERVICE"
