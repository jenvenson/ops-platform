#!/bin/bash
# =============================================================================
# OPS Platform 首次全新部署脚本
# 适用于服务器是空的，没有任何容器和数据的情况
# =============================================================================

set -Eeuo pipefail

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
FRONTEND_DIR="${PROJECT_DIR}/frontend"
ENV_FILE="${ENV_FILE:-${SCRIPT_DIR}/.env}"
NGINX_HTML_DIR="${NGINX_HTML_DIR:-${SCRIPT_DIR}/nginx/html}"
FRONTEND_BUILD_IMAGE="${FRONTEND_BUILD_IMAGE:-node:20-alpine}"
DB_READY_TIMEOUT="${DB_READY_TIMEOUT:-30}"
APP_READY_TIMEOUT="${APP_READY_TIMEOUT:-60}"
FORCE_INIT="${FORCE_INIT:-0}"
CURRENT_STEP="初始化"

read -r -d '' HELP_TEXT <<EOF || true
用法:
  ./deploy-init.sh

说明:
  首次部署脚本会清理当前 compose 项目容器并重新初始化数据库。
  如果检测到已有运行中的容器，默认会直接退出，避免误覆盖现有环境。

可选环境变量:
  ENV_FILE              环境变量文件路径，默认 ${ENV_FILE}
  NGINX_HTML_DIR        前端静态文件发布目录，默认 ${NGINX_HTML_DIR}
  FRONTEND_BUILD_IMAGE  前端构建镜像，默认 ${FRONTEND_BUILD_IMAGE}
  DB_READY_TIMEOUT      MySQL 就绪等待秒数，默认 ${DB_READY_TIMEOUT}
  APP_READY_TIMEOUT     全部服务健康检查等待秒数，默认 ${APP_READY_TIMEOUT}
  FORCE_INIT            设为 1 时允许在检测到现有容器后继续执行

示例:
  ./deploy-init.sh
  ENV_FILE=/opt/ops-platform/deploy/.env ./deploy-init.sh
  FORCE_INIT=1 ./deploy-init.sh
EOF

print_help() {
    printf '%s\n' "${HELP_TEXT}"
}

require_cmd() {
    if ! command -v "$1" >/dev/null 2>&1; then
        echo -e "${RED}错误: 缺少命令 $1${NC}"
        exit 1
    fi
}

on_error() {
    local exit_code=$?
    local failed_command="$1"
    local line_no="$2"

    echo ""
    echo -e "${RED}错误: 步骤失败${NC}"
    echo "阶段: ${CURRENT_STEP}"
    echo "行号: ${line_no}"
    echo "命令: ${failed_command}"
    echo ""

    if command -v docker >/dev/null 2>&1; then
        echo -e "${YELLOW}当前容器状态:${NC}"
        compose ps || true
        echo ""
        echo -e "${YELLOW}最近日志（如容器已启动）:${NC}"
        compose logs --tail 30 mysql redis backend nginx 2>/dev/null || true
        echo ""
    fi

    echo -e "${YELLOW}排查建议:${NC}"
    echo "  1. 检查 ${ENV_FILE} 中 DB_PASSWORD、REDIS_PASSWORD、JWT_SECRET 是否已配置"
    echo "  2. 检查 Docker 服务和镜像拉取网络"
    echo "  3. 手动执行: cd ${SCRIPT_DIR} && ${COMPOSE_CMD:-docker compose} logs -f"

    exit "${exit_code}"
}

trap 'on_error "$BASH_COMMAND" "$LINENO"' ERR

case "${1:-}" in
    -h|--help|help)
        print_help
        exit 0
        ;;
esac

require_cmd docker
require_cmd curl
require_cmd cp
require_cmd grep
require_cmd cut
require_cmd awk

if docker compose version >/dev/null 2>&1; then
    COMPOSE_CMD="docker compose"
elif command -v docker-compose >/dev/null 2>&1; then
    COMPOSE_CMD="docker-compose"
else
    echo "错误: 未找到 docker compose / docker-compose"
    exit 1
fi

compose() {
    ${COMPOSE_CMD} "$@"
}

read_env_value() {
    local key="$1"
    grep "^${key}=" "${ENV_FILE}" | cut -d'=' -f2- || true
}

cd "${SCRIPT_DIR}"

echo -e "${GREEN}=== OPS Platform 首次部署 ===${NC}"
echo "时间: $(date '+%Y-%m-%d %H:%M:%S')"
echo "目录: ${SCRIPT_DIR}"
echo "Compose: ${CYAN}${COMPOSE_CMD}${NC}"
echo ""

# -----------------------------------------------------------------------------
# 配置检查
# -----------------------------------------------------------------------------
echo -e "${YELLOW}[1/6] 检查配置...${NC}"

CURRENT_STEP="检查配置"

if [ ! -f "${ENV_FILE}" ]; then
    echo -e "${YELLOW}未找到 .env 文件，正在创建...${NC}"
    cp "${SCRIPT_DIR}/.env.example" "${ENV_FILE}"
    echo -e "${RED}请先编辑 .env 文件配置密码和密钥！${NC}"
    echo "执行: vim ${ENV_FILE}"
    exit 1
fi

# 读取配置
DB_PASSWORD="$(read_env_value DB_PASSWORD)"
REDIS_PASSWORD="$(read_env_value REDIS_PASSWORD)"
JWT_SECRET="$(read_env_value JWT_SECRET)"

if [ -z "${DB_PASSWORD}" ] || [ -z "${REDIS_PASSWORD}" ] || [ -z "${JWT_SECRET}" ]; then
    echo -e "${RED}错误: ${ENV_FILE} 中必须配置 DB_PASSWORD、REDIS_PASSWORD、JWT_SECRET${NC}"
    exit 1
fi

EXISTING_SERVICES="$(compose ps --services 2>/dev/null || true)"
if [ -n "${EXISTING_SERVICES}" ] && [ "${FORCE_INIT}" != "1" ]; then
    echo -e "${RED}错误: 检测到当前 compose 项目已有服务，本脚本默认拒绝继续执行${NC}"
    echo "${EXISTING_SERVICES}"
    echo ""
    echo "如果确认要重置当前 compose 项目，请显式执行:"
    echo "  FORCE_INIT=1 ./deploy-init.sh"
    exit 1
fi

echo "配置检查通过！"
echo ""

# -----------------------------------------------------------------------------
# 启动数据库
# -----------------------------------------------------------------------------
echo -e "${YELLOW}[2/6] 启动数据库服务...${NC}"
CURRENT_STEP="启动数据库服务"

# 停止可能存在的旧容器
compose down 2>/dev/null || true

# 只启动数据库服务
compose up -d mysql redis

# 等待数据库启动
echo "等待数据库启动..."
for ((i=1; i<=DB_READY_TIMEOUT; i++)); do
    if compose exec -e MYSQL_PWD="${DB_PASSWORD}" mysql mysqladmin ping -uroot --silent &>/dev/null; then
        echo "MySQL 已就绪！"
        break
    fi
    if [ "${i}" -eq "${DB_READY_TIMEOUT}" ]; then
        echo -e "${RED}错误: MySQL 启动超时${NC}"
        exit 1
    fi
    sleep 1
done

echo ""

# -----------------------------------------------------------------------------
# 初始化数据库
# -----------------------------------------------------------------------------
echo -e "${YELLOW}[3/6] 初始化数据库...${NC}"
CURRENT_STEP="初始化数据库"

echo "执行初始化脚本..."
compose exec -T -e MYSQL_PWD="${DB_PASSWORD}" mysql mysql -uroot < "${PROJECT_DIR}/backend/scripts/init.sql"

echo "执行迁移文件..."
ENV_FILE="${ENV_FILE}" bash "${SCRIPT_DIR}/apply-migrations.sh"

echo "初始化角色数据..."
compose exec -T -e MYSQL_PWD="${DB_PASSWORD}" mysql mysql -uroot ops_platform -e "
INSERT IGNORE INTO roles (name, code, description, status) VALUES
('超级管理员', 'admin', '拥有所有权限', 1),
('运维人员', 'ops', '负责系统运维工作', 1),
('开发人员', 'dev', '负责应用开发', 1),
('普通用户', 'user', '普通用户角色', 1);
" 2>/dev/null || true

echo ""

# -----------------------------------------------------------------------------
# 构建后端
# -----------------------------------------------------------------------------
echo -e "${YELLOW}[4/6] 构建后端镜像（包含 Nmap、RustScan、Nuclei）...${NC}"
CURRENT_STEP="构建后端镜像"

compose build backend

echo ""

# -----------------------------------------------------------------------------
# 部署前端静态文件
# -----------------------------------------------------------------------------
echo -e "${YELLOW}[5/6] 部署前端静态页面...${NC}"
CURRENT_STEP="构建并发布前端"

if [ ! -f "${FRONTEND_DIR}/package.json" ]; then
    echo -e "${RED}错误: 前端源码目录不存在: ${FRONTEND_DIR}${NC}"
    exit 1
fi

echo "使用 Node 容器构建前端静态资源..."
docker run --rm \
    -v "${FRONTEND_DIR}:/app" \
    -w /app \
    "${FRONTEND_BUILD_IMAGE}" \
    sh -lc "set -e; corepack enable pnpm >/dev/null 2>&1; pnpm install --frozen-lockfile; pnpm build"

if [ ! -f "${FRONTEND_DIR}/dist/index.html" ]; then
    echo -e "${RED}错误: 前端编译失败，dist/index.html 未生成${NC}"
    exit 1
fi

mkdir -p "${NGINX_HTML_DIR}"
rm -rf "${NGINX_HTML_DIR:?}"/*
echo "复制前端静态文件到 ${NGINX_HTML_DIR}..."
cp -R "${FRONTEND_DIR}/dist/." "${NGINX_HTML_DIR}/"

if [ -f "${NGINX_HTML_DIR}/index.html" ]; then
    echo "前端静态文件部署成功！"
else
    echo -e "${RED}错误: index.html 未找到，部署失败${NC}"
    exit 1
fi

echo ""

# -----------------------------------------------------------------------------
# 启动所有服务
# -----------------------------------------------------------------------------
echo -e "${YELLOW}[6/6] 启动所有服务...${NC}"
CURRENT_STEP="启动全部服务"

compose up -d

echo "等待后端和 Nginx 就绪..."
for ((i=1; i<=APP_READY_TIMEOUT; i++)); do
    if curl -sf http://localhost:8080/health >/dev/null 2>&1 && \
       curl -sf http://localhost/api/health >/dev/null 2>&1 && \
       curl -sf http://localhost/ >/dev/null 2>&1; then
        echo "健康检查通过"
        break
    fi
    if [ "${i}" -eq "${APP_READY_TIMEOUT}" ]; then
        echo -e "${YELLOW}警告: 健康检查超时，请手动检查日志${NC}"
    fi
    sleep 1
done

echo ""
echo -e "${GREEN}=== 部署完成 ===${NC}"
echo ""
echo "服务状态:"
compose ps
echo ""
echo -e "${GREEN}访问地址: http://$(hostname -I | awk '{print $1}')/${NC}"
echo ""
echo "默认账号:"
echo "  用户名: admin"
echo "  密码:   admin123"
echo ""
echo -e "${YELLOW}建议立即修改默认密码！${NC}"
