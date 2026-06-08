#!/bin/bash
# =============================================================================
# OPS Platform 迭代部署脚本
# 在本地 Mac 上执行，将编译好的前后端文件传输到线上服务器并重启
#
# 前提条件:
#   - 线上已通过 deploy-init.sh 完成首次部署
#   - 本地已配置 ssh 免密登录到线上服务器
#   - 本地已安装 Go、Node.js 环境
#
# 用法:
#   ./deploy-update.sh            # 同时部署前端和后端
#   ./deploy-update.sh backend    # 只部署后端
#   ./deploy-update.sh frontend   # 只部署前端
#   ./deploy-update.sh nginx      # 只更新 nginx 配置
#   ./deploy-update.sh migrate    # 只同步并执行数据库迁移
#
# 说明:
#   - 不需要把 deploy 目录传到线上，线上首次部署时已存在
#   - 后端采用本地交叉编译，直接替换容器内二进制文件，无需重建镜像
#   - 前端本地 build 后将 dist 文件传到线上 nginx 静态目录
# =============================================================================

set -euo pipefail

# ======================== 配置区 ========================
REMOTE_HOST="${REMOTE_HOST:-}"
REMOTE_USER="${REMOTE_USER:-root}"
REMOTE_DEPLOY_DIR="${REMOTE_DEPLOY_DIR:-/opt/ops-platform/deploy}"
REMOTE_PROJECT_DIR="${REMOTE_PROJECT_DIR:-$(dirname "${REMOTE_DEPLOY_DIR}")}"
REMOTE_NGINX_HTML="${REMOTE_DEPLOY_DIR}/nginx/html"
REMOTE_NGINX_CONFIG_TEMPLATE="${REMOTE_DEPLOY_DIR}/nginx.prod.conf.template"
REMOTE_BACKEND_CONTAINER="${REMOTE_BACKEND_CONTAINER:-ops-backend}"
REMOTE_NGINX_CONTAINER="${REMOTE_NGINX_CONTAINER:-ops-nginx}"
REMOTE_MYSQL_CONTAINER="${REMOTE_MYSQL_CONTAINER:-ops-mysql}"
REMOTE_BACKUP_DIR="${REMOTE_BACKUP_DIR:-${REMOTE_DEPLOY_DIR}/backups}"
BACKUP_BEFORE_MIGRATE="${BACKUP_BEFORE_MIGRATE:-1}"

# 容器内后端二进制路径（与 Dockerfile.backend 中 WORKDIR /app + go build -o server 对应）
REMOTE_BACKEND_BIN="${REMOTE_BACKEND_BIN:-/app/server}"

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
BACKEND_DIR="${PROJECT_DIR}/backend"
FRONTEND_DIR="${PROJECT_DIR}/frontend"
LOCAL_BACKEND_BIN="$(mktemp /tmp/ops-backend-linux.XXXXXX)"
LOCAL_FRONTEND_TAR="$(mktemp /tmp/ops-frontend-dist.XXXXXX.tar.gz)"
LOCAL_RELEASE_ASSETS_TAR="$(mktemp /tmp/ops-release-assets.XXXXXX.tar.gz)"
SSH_TARGET="${REMOTE_USER}@${REMOTE_HOST}"

cleanup() {
    rm -f "${LOCAL_BACKEND_BIN}" "${LOCAL_FRONTEND_TAR}" "${LOCAL_RELEASE_ASSETS_TAR}"
}
trap cleanup EXIT

require_cmd() {
    if ! command -v "$1" >/dev/null 2>&1; then
        echo -e "${RED}错误: 缺少命令 $1${NC}"
        exit 1
    fi
}

print_help() {
    cat <<EOF
用法:
  ./deploy-update.sh [all|backend|frontend|nginx|migrate]

说明:
  all       同时部署后端和前端（默认）
  backend   只部署后端
  frontend  只部署前端
  nginx     只更新 Nginx 生产配置
  migrate   只同步并执行数据库迁移

可选环境变量:
  REMOTE_HOST               远端主机，默认 ${REMOTE_HOST}
  REMOTE_USER               远端用户，默认 ${REMOTE_USER}
  REMOTE_DEPLOY_DIR         远端 deploy 目录，默认 ${REMOTE_DEPLOY_DIR}
  REMOTE_BACKEND_BIN        容器内后端二进制路径，默认 ${REMOTE_BACKEND_BIN}
  REMOTE_BACKEND_CONTAINER  远端后端容器名，默认 ${REMOTE_BACKEND_CONTAINER}
  REMOTE_NGINX_CONTAINER    远端 Nginx 容器名，默认 ${REMOTE_NGINX_CONTAINER}
  REMOTE_MYSQL_CONTAINER    远端 MySQL 容器名，默认 ${REMOTE_MYSQL_CONTAINER}
  REMOTE_BACKUP_DIR         远端数据库备份目录，默认 ${REMOTE_BACKUP_DIR}
  BACKUP_BEFORE_MIGRATE     迁移前是否自动备份数据库，默认 ${BACKUP_BEFORE_MIGRATE}

示例:
  ./deploy-update.sh
  ./deploy-update.sh backend
  REMOTE_HOST=<your-server-ip> REMOTE_USER=deploy ./deploy-update.sh frontend
  REMOTE_HOST=<your-server-ip> REMOTE_USER=deploy ./deploy-update.sh migrate
EOF
}

warn_if_remote_env_matches() {
    local key="$1"
    local risky_value="$2"
    local message="$3"
    local remote_value

    remote_value=$(remote_env_value "${key}")
    if [ "${remote_value}" = "${risky_value}" ]; then
        echo -e "${YELLOW}警告: ${message}${NC}"
    fi
}

remote_env_value() {
    local key="$1"
    ssh "${SSH_TARGET}" "cd '${REMOTE_DEPLOY_DIR}' && awk -F= '/^${key}=/{sub(/^[^=]*=/, \"\"); print; exit}' .env" 2>/dev/null || true
}

remote_backend_health_url() {
    local backend_port
    backend_port="$(remote_env_value "BACKEND_PORT")"
    backend_port="${backend_port:-8080}"
    printf 'http://localhost:%s/health' "${backend_port}"
}

should_backup_before_migrate() {
    case "${DEPLOY_TARGET}" in
        migrate|backend|all)
            [ "${BACKUP_BEFORE_MIGRATE}" != "0" ]
            ;;
        *)
            return 1
            ;;
    esac
}

validate_remote_env() {
    local required_keys=()
    local optional_warn_keys=()

    case "${DEPLOY_TARGET}" in
        migrate)
            required_keys=(DB_PASSWORD REDIS_PASSWORD JWT_SECRET)
            ;;
        backend)
            required_keys=(DB_PASSWORD REDIS_PASSWORD JWT_SECRET)
            optional_warn_keys=(ASSISTANT_API_KEY GRAFANA_URL GRAFANA_USERNAME GRAFANA_PASSWORD JENKINS_URL JENKINS_USERNAME JENKINS_TOKEN)
            ;;
        nginx)
            required_keys=(GRAFANA_UPSTREAM GRAFANA_HOST_HEADER GRAFANA_APP_URL)
            optional_warn_keys=(GRAFANA_BASIC_AUTH)
            ;;
        frontend)
            ;;
        all)
            required_keys=(DB_PASSWORD REDIS_PASSWORD JWT_SECRET GRAFANA_UPSTREAM GRAFANA_HOST_HEADER GRAFANA_APP_URL)
            optional_warn_keys=(ASSISTANT_API_KEY GRAFANA_URL GRAFANA_USERNAME GRAFANA_PASSWORD GRAFANA_BASIC_AUTH JENKINS_URL JENKINS_USERNAME JENKINS_TOKEN)
            ;;
    esac

    if [ "${#required_keys[@]}" -eq 0 ] && [ "${#optional_warn_keys[@]}" -eq 0 ]; then
        return 0
    fi

    echo -e "${YELLOW}[检查] 校验线上 .env 配置...${NC}"

    local env_check_result
    env_check_result=$(ssh "${SSH_TARGET}" "cd '${REMOTE_DEPLOY_DIR}' && python3 - <<'PY' .env ${required_keys[*]:-} -- ${optional_warn_keys[*]:-}
import sys
from pathlib import Path

env_path = Path(sys.argv[1])
args = sys.argv[2:]
sep = args.index('--') if '--' in args else len(args)
required = [arg for arg in args[:sep] if arg]
optional = [arg for arg in args[sep + 1:] if arg]

if not env_path.exists():
    print('ERROR:.env 文件不存在')
    raise SystemExit(1)

values = {}
for line in env_path.read_text(encoding='utf-8').splitlines():
    line = line.strip()
    if not line or line.startswith('#') or '=' not in line:
        continue
    key, value = line.split('=', 1)
    values[key.strip()] = value.strip()

missing = [key for key in required if not values.get(key)]
if missing:
    print('ERROR:缺少必填配置: ' + ', '.join(missing))
    raise SystemExit(1)

empty_optional = [key for key in optional if key in values and not values.get(key)]
missing_optional = [key for key in optional if key not in values]

if required:
    print('OK:必填配置已就绪: ' + ', '.join(required))
if missing_optional:
    print('WARN:建议补充配置: ' + ', '.join(missing_optional))
if empty_optional:
    print('WARN:配置存在但为空: ' + ', '.join(empty_optional))
PY" 2>&1) || {
        echo "${env_check_result}"
        echo -e "${RED}错误: 线上 .env 校验失败，请先修正配置再发布${NC}"
        exit 1
    }

    echo "${env_check_result}"
    warn_if_remote_env_matches "DB_PASSWORD" "change_me_in_production" "DB_PASSWORD 仍为占位值，建议尽快更换"
    warn_if_remote_env_matches "REDIS_PASSWORD" "redis_secure_password_change_in_production" "REDIS_PASSWORD 仍为占位风格值，建议尽快轮换"
    echo ""
}

sync_remote_release_assets() {
    echo -e "${YELLOW}[同步] 上传部署与迁移文件...${NC}"
    cd "${PROJECT_DIR}"
    tar czf "${LOCAL_RELEASE_ASSETS_TAR}" \
        backend/scripts/init.sql \
        backend/migrations \
        docs \
        deploy/.env.example \
        deploy/apply-migrations.sh \
        deploy/deploy-init.sh \
        deploy/deploy-update.sh \
        deploy/docker-compose.yml \
        deploy/migrations \
        deploy/nginx.prod.conf.template \
        migrations

    scp "${LOCAL_RELEASE_ASSETS_TAR}" "${SSH_TARGET}:/tmp/ops-release-assets.tar.gz"

    ssh "${SSH_TARGET}" bash -s << SYNC_EOF
        set -euo pipefail
        mkdir -p '${REMOTE_PROJECT_DIR}'
        tar xzf /tmp/ops-release-assets.tar.gz -C '${REMOTE_PROJECT_DIR}'
        rm -f /tmp/ops-release-assets.tar.gz
        echo "部署与迁移文件已同步"
SYNC_EOF

    echo ""
}

apply_remote_migrations() {
    echo -e "${YELLOW}[迁移] 应用数据库迁移...${NC}"

    ssh "${SSH_TARGET}" bash -s << MIGRATION_EOF
        $(compose_remote_prelude)
        cd '${REMOTE_DEPLOY_DIR}'
        compose up -d mysql redis >/dev/null
        ENV_FILE='${REMOTE_DEPLOY_DIR}/.env' bash '${REMOTE_DEPLOY_DIR}/apply-migrations.sh'
MIGRATION_EOF

    echo -e "${GREEN}[迁移] 完成 ✓${NC}"
    echo ""
}

backup_remote_database() {
    echo -e "${YELLOW}[备份] 迁移前备份数据库...${NC}"

    ssh "${SSH_TARGET}" bash -s << BACKUP_EOF
        $(compose_remote_prelude)
        cd '${REMOTE_DEPLOY_DIR}'

        DB_PASSWORD=\$(awk -F= '/^DB_PASSWORD=/{sub(/^[^=]*=/, ""); print; exit}' .env)
        if [ -z "\${DB_PASSWORD}" ]; then
            echo "DB_PASSWORD 未配置，无法执行数据库备份"
            exit 1
        fi

        compose up -d mysql >/dev/null
        mkdir -p '${REMOTE_BACKUP_DIR}'

        echo -n "等待 MySQL 就绪"
        for i in \$(seq 1 30); do
            if docker exec -e MYSQL_PWD="\${DB_PASSWORD}" ${REMOTE_MYSQL_CONTAINER} \
                sh -lc 'exec mysqladmin ping -uroot -h 127.0.0.1 --silent' >/dev/null 2>&1; then
                echo ""
                echo "MySQL 已就绪"
                break
            fi
            sleep 1
            echo -n "."
            if [ "\${i}" -eq 30 ]; then
                echo ""
                echo "MySQL 启动超时，无法执行数据库备份"
                exit 1
            fi
        done

        BACKUP_FILE='${REMOTE_BACKUP_DIR}/ops_platform_'"\$(date +%Y%m%d_%H%M%S)"'.sql'
        docker exec -e MYSQL_PWD="\${DB_PASSWORD}" ${REMOTE_MYSQL_CONTAINER} \
            sh -lc 'exec mysqldump -uroot --single-transaction --quick --routines --events ops_platform' \
            > "\${BACKUP_FILE}"
        gzip -f "\${BACKUP_FILE}"
        echo "数据库备份已保存: \${BACKUP_FILE}.gz"
BACKUP_EOF

    echo -e "${GREEN}[备份] 完成 ✓${NC}"
    echo ""
}

compose_remote_prelude() {
    cat <<'EOF'
set -euo pipefail
if docker compose version >/dev/null 2>&1; then
    compose() { docker compose "$@"; }
elif command -v docker-compose >/dev/null 2>&1; then
    compose() { docker-compose "$@"; }
else
    echo "remote compose command not found"
    exit 1
fi
EOF
}

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

# 部署目标（默认全部）
DEPLOY_TARGET="${1:-all}"

case "${DEPLOY_TARGET}" in
    -h|--help|help)
        print_help
        exit 0
        ;;
esac

echo -e "${GREEN}=======================================${NC}"
echo -e "${GREEN}  OPS Platform 迭代部署${NC}"
echo -e "${GREEN}=======================================${NC}"
echo -e "时间: $(date '+%Y-%m-%d %H:%M:%S')"
echo -e "目标: ${CYAN}${DEPLOY_TARGET}${NC}"
echo -e "服务器: ${CYAN}${REMOTE_USER}@${REMOTE_HOST}${NC}"
echo ""

require_cmd ssh
require_cmd scp
require_cmd tar
require_cmd go
require_cmd npm

# ======================== 前置检查 ========================
echo -e "${YELLOW}[检查] 验证线上服务状态...${NC}"

REMOTE_STATUS=$(ssh "${SSH_TARGET}" "$(compose_remote_prelude)
cd '${REMOTE_DEPLOY_DIR}'
compose ps --format 'table {{.Name}}\t{{.Status}}' 2>/dev/null || docker ps --format '{{.Names}}\t{{.Status}}' | grep -E '${REMOTE_BACKEND_CONTAINER}|${REMOTE_NGINX_CONTAINER}|${REMOTE_MYSQL_CONTAINER}'
" 2>/dev/null) || {
    echo -e "${RED}错误: 无法连接线上服务器或服务未运行${NC}"
    echo -e "${RED}请先在线上执行首次部署 (deploy-init.sh)${NC}"
    exit 1
}
echo "${REMOTE_STATUS}"
echo -e "${GREEN}线上服务运行正常${NC}"
echo ""

validate_remote_env

if should_backup_before_migrate; then
    backup_remote_database
fi

# ======================== 部署后端 ========================
deploy_backend() {
    echo -e "${YELLOW}[后端 1/3] 交叉编译 Linux 二进制...${NC}"
    cd "${BACKEND_DIR}"

    # 交叉编译：Mac 编译出 Linux amd64 二进制
    GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o "${LOCAL_BACKEND_BIN}" ./cmd/server/

    FILE_SIZE=$(ls -lh "${LOCAL_BACKEND_BIN}" | awk '{print $5}')
    echo -e "编译完成，文件大小: ${CYAN}${FILE_SIZE}${NC}"

    echo -e "${YELLOW}[后端 2/3] 传输到线上服务器...${NC}"
    scp "${LOCAL_BACKEND_BIN}" "${SSH_TARGET}:/tmp/ops-backend-linux"
    echo "传输完成"

    echo -e "${YELLOW}[后端 3/3] 替换二进制并重启容器...${NC}"
    local backend_health_url
    backend_health_url="$(remote_backend_health_url)"

    ssh "${SSH_TARGET}" bash -s << BACKEND_EOF
        $(compose_remote_prelude)
        cd '${REMOTE_DEPLOY_DIR}'
        # 先重建 backend，使新的卷挂载（如 docs）生效
        compose up -d --force-recreate backend >/dev/null

        # 替换容器内的二进制文件（线上通过 docker-compose 启动，容器名固定为 ops-backend）
        docker cp /tmp/ops-backend-linux ${REMOTE_BACKEND_CONTAINER}:${REMOTE_BACKEND_BIN}

        # 使用 docker-compose restart 保持与编排一致
        compose restart backend
        rm -f /tmp/ops-backend-linux
        echo "后端容器已重启"

        # 等待后端启动并检查健康状态
        echo -n "等待启动"
        for i in \$(seq 1 15); do
            sleep 1
            echo -n "."
            if curl -sf '${backend_health_url}' >/dev/null 2>&1; then
                echo ""
                echo "健康检查通过"
                compose ps backend
                exit 0
            fi
        done
        echo ""
        echo "警告: 健康检查超时，请检查日志:"
        echo "  cd ${REMOTE_DEPLOY_DIR} && compose logs --tail 30 backend"
BACKEND_EOF

    echo -e "${GREEN}[后端] 部署完成 ✓${NC}"
    echo ""
}

# ======================== 部署前端 ========================
deploy_frontend() {
    echo -e "${YELLOW}[前端 1/3] 编译前端项目...${NC}"
    cd "${FRONTEND_DIR}"
    npm run build

    if [ ! -f "dist/index.html" ]; then
        echo -e "${RED}错误: 前端编译失败，dist/index.html 不存在${NC}"
        exit 1
    fi

    echo -e "${YELLOW}[前端 2/3] 打包并传输到线上...${NC}"
    # 用 tar 打包传输更可靠，避免 scp -r 大量小文件的问题
    cd dist
    tar czf "${LOCAL_FRONTEND_TAR}" .
    scp "${LOCAL_FRONTEND_TAR}" "${SSH_TARGET}:/tmp/ops-frontend-dist.tar.gz"
    echo "传输完成"

    echo -e "${YELLOW}[前端 3/3] 解压并刷新 Nginx...${NC}"
    local backend_health_url
    backend_health_url="$(remote_backend_health_url)"

    ssh "${SSH_TARGET}" bash -s << FRONTEND_EOF
        $(compose_remote_prelude)
        mkdir -p '${REMOTE_NGINX_HTML}'
        # 清空旧文件，解压新文件
        rm -rf ${REMOTE_NGINX_HTML}/*
        tar xzf /tmp/ops-frontend-dist.tar.gz -C ${REMOTE_NGINX_HTML}/
        rm -f /tmp/ops-frontend-dist.tar.gz

        # 验证 index.html 存在
        if [ -f "${REMOTE_NGINX_HTML}/index.html" ]; then
            echo "前端文件部署成功"
        else
            echo "错误: index.html 未找到"
            exit 1
        fi

        # 刷新 Nginx（docker-compose 启动的容器，直接 exec 即可）
        docker exec ${REMOTE_NGINX_CONTAINER} nginx -t >/dev/null
        docker exec ${REMOTE_NGINX_CONTAINER} nginx -s reload

        # 校验首页和后端健康检查
        curl -sf http://localhost/ >/dev/null
        curl -sf '${backend_health_url}' >/dev/null
        echo "Nginx 已刷新"
        echo "前端页面与后端健康检查通过"
FRONTEND_EOF

    echo -e "${GREEN}[前端] 部署完成 ✓${NC}"
    echo ""
}

# ======================== 更新 Nginx 配置 ========================
deploy_nginx() {
    echo -e "${YELLOW}[Nginx 1/2] 传输配置文件...${NC}"
    scp "${SCRIPT_DIR}/nginx.prod.conf.template" "${SSH_TARGET}:${REMOTE_NGINX_CONFIG_TEMPLATE}"
    echo "传输完成"

    echo -e "${YELLOW}[Nginx 2/2] 测试并重新加载配置...${NC}"
    local backend_health_url
    backend_health_url="$(remote_backend_health_url)"

    ssh "${SSH_TARGET}" bash -s << NGINX_EOF
        $(compose_remote_prelude)
        cd '${REMOTE_DEPLOY_DIR}'
        # 强制重建容器，确保旧版直接挂载 nginx.conf 的容器切换到模板化配置
        compose up -d --force-recreate nginx
        sleep 2
        docker exec ${REMOTE_NGINX_CONTAINER} nginx -t
        curl -sf http://localhost/ >/dev/null
        curl -sf '${backend_health_url}' >/dev/null
        echo "Nginx 配置已更新"
        echo "页面与后端健康检查通过"
NGINX_EOF

    echo -e "${GREEN}[Nginx] 配置更新完成 ✓${NC}"
    echo ""
}

# ======================== 执行部署 ========================
START_TIME=$(date +%s)

case "${DEPLOY_TARGET}" in
    backend)
        sync_remote_release_assets
        apply_remote_migrations
        deploy_backend
        ;;
    frontend)
        deploy_frontend
        ;;
    nginx)
        sync_remote_release_assets
        deploy_nginx
        ;;
    migrate)
        sync_remote_release_assets
        apply_remote_migrations
        ;;
    all)
        sync_remote_release_assets
        apply_remote_migrations
        deploy_backend
        deploy_frontend
        ;;
    *)
        echo -e "${RED}未知目标: ${DEPLOY_TARGET}${NC}"
        echo ""
        print_help
        exit 1
        ;;
esac

END_TIME=$(date +%s)
ELAPSED=$((END_TIME - START_TIME))

echo -e "${GREEN}=======================================${NC}"
echo -e "${GREEN}  部署完成！耗时: ${ELAPSED} 秒${NC}"
echo -e "${GREEN}=======================================${NC}"
echo ""
echo -e "访问地址: ${CYAN}http://${REMOTE_HOST}${NC}"
echo -e "后端日志: ${CYAN}ssh ${REMOTE_USER}@${REMOTE_HOST} 'cd ${REMOTE_DEPLOY_DIR} && docker compose logs -f --tail 50 backend || docker-compose logs -f --tail 50 backend'${NC}"
