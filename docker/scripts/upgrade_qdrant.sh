#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPOSE_DIR="$(dirname "$SCRIPT_DIR")"
SERVICE="vector-db-upgrade"

cd "$COMPOSE_DIR"

echo "[upgrade_qdrant] ensure qdrant up"
docker compose -f docker-compose-db.yaml up -d qdrant

echo "[upgrade_qdrant] build $SERVICE"
docker compose build "$SERVICE"

echo "[upgrade_qdrant] run $SERVICE (one-shot)"
docker compose run --rm "$SERVICE"

echo "[upgrade_qdrant] done"
