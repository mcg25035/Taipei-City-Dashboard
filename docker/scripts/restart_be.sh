#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPOSE_DIR="$(dirname "$SCRIPT_DIR")"
SERVICE="dashboard-be"

cd "$COMPOSE_DIR"

echo "[restart_be] build $SERVICE"
docker compose build "$SERVICE"

echo "[restart_be] recreate $SERVICE"
docker compose up -d --force-recreate --no-deps "$SERVICE"

echo "[restart_be] status"
docker compose ps "$SERVICE"
