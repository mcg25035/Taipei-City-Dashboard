#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DE_COMPOSE_DIR="$(cd "$SCRIPT_DIR/../../Taipei-City-Dashboard-DE/docker/develop" && pwd)"

# Airflow 需要 AIRFLOW_UID, 預設 50000
export AIRFLOW_UID="${AIRFLOW_UID:-50000}"

cd "$DE_COMPOSE_DIR"

# 首次啟動需要 init (init 完會自動退出, 已 init 過會 no-op)
echo "[restart_de] airflow-init"
docker compose up airflow-init

echo "[restart_de] build all DE services"
docker compose build

echo "[restart_de] recreate DE stack"
docker compose up -d --force-recreate --remove-orphans

echo "[restart_de] status"
docker compose ps
