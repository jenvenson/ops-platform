#!/bin/bash

set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
ENV_FILE="${ENV_FILE:-${SCRIPT_DIR}/.env}"
DB_NAME="${DB_NAME:-ops_platform}"
BOOTSTRAP_TABLE_THRESHOLD="${BOOTSTRAP_TABLE_THRESHOLD:-3}"
MODE="compose"

if [ "${1:-}" = "--direct" ]; then
    MODE="direct"
fi

mysql_client_bin() {
    if command -v mariadb >/dev/null 2>&1; then
        printf '%s\n' "mariadb"
        return 0
    fi
    printf '%s\n' "mysql"
}

load_env_file() {
    local key="$1"
    [ -f "${ENV_FILE}" ] || return 0
    grep "^${key}=" "${ENV_FILE}" | cut -d'=' -f2- || true
}

if [ -z "${DB_PASSWORD:-}" ]; then
    DB_PASSWORD="$(load_env_file DB_PASSWORD)"
fi

if [ "${MODE}" = "direct" ]; then
    DB_HOST="${DB_HOST:-mysql}"
    DB_PORT="${DB_PORT:-3306}"
    DB_USER="${DB_USER:-root}"
    MYSQL_DIRECT_SSL_FLAG="${MYSQL_DIRECT_SSL_FLAG:---skip-ssl}"

    mysql_exec() {
        MYSQL_PWD="${DB_PASSWORD}" "$(mysql_client_bin)" ${MYSQL_DIRECT_SSL_FLAG} --protocol=TCP -h "${DB_HOST}" -P "${DB_PORT}" -u "${DB_USER}" "$@"
    }

    mysql_query() {
        MYSQL_PWD="${DB_PASSWORD}" "$(mysql_client_bin)" ${MYSQL_DIRECT_SSL_FLAG} --protocol=TCP -N -B -h "${DB_HOST}" -P "${DB_PORT}" -u "${DB_USER}" "$@"
    }
else
    cd "${SCRIPT_DIR}"

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

    compose_mysql_client() {
        compose exec -T -e MYSQL_PWD="${DB_PASSWORD}" mysql sh -lc '
            if command -v mariadb >/dev/null 2>&1; then
                exec mariadb "$@"
            fi
            exec mysql "$@"
        ' sh "$@"
    }

    mysql_exec() {
        compose_mysql_client -uroot "$@"
    }

    mysql_query() {
        compose_mysql_client -N -B -uroot "$@"
    }
fi

require_db_password() {
    if [ -z "${DB_PASSWORD:-}" ]; then
        echo "错误: 缺少 DB_PASSWORD"
        exit 1
    fi
}

checksum_file() {
    local file="$1"
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "${file}" | awk '{print $1}'
    elif command -v shasum >/dev/null 2>&1; then
        LC_ALL=C shasum -a 256 "${file}" | awk '{print $1}'
    else
        echo "错误: 缺少 sha256sum/shasum"
        exit 1
    fi
}

sql_escape() {
    printf "%s" "$1" | sed "s/'/''/g"
}

ensure_database() {
    mysql_exec -e "CREATE DATABASE IF NOT EXISTS \`${DB_NAME}\` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
}

ensure_migration_table() {
    mysql_exec "${DB_NAME}" -e "
CREATE TABLE IF NOT EXISTS schema_migrations (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    path VARCHAR(255) NOT NULL UNIQUE,
    checksum CHAR(64) NOT NULL,
    mode VARCHAR(20) NOT NULL DEFAULT 'apply' COMMENT 'apply/baseline',
    applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='数据库迁移记录';
"
}

list_migrations() {
    local dir=""

    for dir in \
        "${PROJECT_DIR}/backend/migrations" \
        "${SCRIPT_DIR}/migrations" \
        "${PROJECT_DIR}/migrations"
    do
        [ -d "${dir}" ] || continue
        find "${dir}" -maxdepth 1 -type f -name '*.sql' | sort
    done
}

relative_path() {
    local file="$1"
    file="${file#${PROJECT_DIR}/}"
    printf "%s" "${file}"
}

baseline_existing_schema_if_needed() {
    local migration_count table_count file rel checksum rel_escaped checksum_escaped

    migration_count="$(mysql_query "${DB_NAME}" -e "SELECT COUNT(*) FROM schema_migrations;" < /dev/null | tr -d '\r')"
    table_count="$(mysql_query "${DB_NAME}" -e "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = '${DB_NAME}' AND table_type = 'BASE TABLE';" < /dev/null | tr -d '\r')"

    if [ "${migration_count}" != "0" ]; then
        return 0
    fi

    if [ "${table_count}" -le "${BOOTSTRAP_TABLE_THRESHOLD}" ]; then
        return 0
    fi

    echo "检测到已有数据库结构且迁移记录为空，正在基线化现有迁移记录..."

    while IFS= read -r file; do
        [ -n "${file}" ] || continue
        rel="$(relative_path "${file}")"
        checksum="$(checksum_file "${file}")"
        rel_escaped="$(sql_escape "${rel}")"
        checksum_escaped="$(sql_escape "${checksum}")"

        mysql_exec "${DB_NAME}" -e "
INSERT INTO schema_migrations (path, checksum, mode)
VALUES ('${rel_escaped}', '${checksum_escaped}', 'baseline')
ON DUPLICATE KEY UPDATE checksum = VALUES(checksum), mode = 'baseline';
" < /dev/null
        echo "  baseline ${rel}"
    done < <(list_migrations)
}

apply_pending_migrations() {
    local file rel checksum rel_escaped checksum_escaped existing_checksum

    while IFS= read -r file; do
        [ -n "${file}" ] || continue

        rel="$(relative_path "${file}")"
        checksum="$(checksum_file "${file}")"
        rel_escaped="$(sql_escape "${rel}")"
        checksum_escaped="$(sql_escape "${checksum}")"
        existing_checksum="$(mysql_query "${DB_NAME}" -e "SELECT checksum FROM schema_migrations WHERE path = '${rel_escaped}' LIMIT 1;" < /dev/null | tr -d '\r')"

        if [ -n "${existing_checksum}" ]; then
            if [ "${existing_checksum}" != "${checksum}" ]; then
                echo "错误: 迁移文件校验和不一致: ${rel}"
                echo "数据库记录: ${existing_checksum}"
                echo "当前文件:   ${checksum}"
                exit 1
            fi
            echo "skip ${rel}"
            continue
        fi

        echo "apply ${rel}"
        mysql_exec "${DB_NAME}" < "${file}"
        mysql_exec "${DB_NAME}" -e "
INSERT INTO schema_migrations (path, checksum, mode)
VALUES ('${rel_escaped}', '${checksum_escaped}', 'apply');
" < /dev/null
    done < <(list_migrations)
}

require_db_password
ensure_database
ensure_migration_table
baseline_existing_schema_if_needed
apply_pending_migrations

echo "数据库迁移已完成"
