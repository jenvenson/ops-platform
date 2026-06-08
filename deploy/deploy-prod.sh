#!/bin/bash
# =============================================================================
# 兼容入口：统一转发到 deploy-update.sh
#
# 旧版 deploy-prod.sh 会绕过 docker compose，且内置过期容器参数与敏感配置。
# 现在保留该文件仅用于兼容历史调用，实际部署统一走 deploy-update.sh。
# =============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

echo "deploy-prod.sh 已兼容转发到 deploy-update.sh"
exec "${SCRIPT_DIR}/deploy-update.sh" "${@:-all}"
