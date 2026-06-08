#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

mkdir -p /opt/ops-platform/ollama/data

if command -v docker-compose >/dev/null 2>&1; then
  docker-compose -f "${SCRIPT_DIR}/docker-compose.ollama.yml" up -d
else
  docker compose -f "${SCRIPT_DIR}/docker-compose.ollama.yml" up -d
fi
