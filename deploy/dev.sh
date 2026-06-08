#!/bin/bash

# 开发环境启动脚本
# 修改代码后只需重启对应服务即可生效

cd "$(dirname "$0")"

COMPOSE_FILE="-f docker-compose.dev.yml"

case "$1" in
  start)
    echo "启动开发环境..."
    docker-compose -f docker-compose.dev.yml up -d
    ;;
  stop)
    echo "停止服务..."
    docker-compose -f docker-compose.dev.yml down
    ;;
  restart)
    echo "重启服务: $2"
    if [ -z "$2" ]; then
      docker-compose -f docker-compose.dev.yml restart
    else
      docker-compose -f docker-compose.dev.yml restart "$2"
    fi
    ;;
  logs)
    if [ -z "$2" ]; then
      docker-compose -f docker-compose.dev.yml logs -f
    else
      docker-compose -f docker-compose.dev.yml logs -f "$2"
    fi
    ;;
  rebuild)
    echo "重建并启动: $2"
    docker-compose -f docker-compose.dev.yml rm -sf "$2"
    docker-compose -f docker-compose.dev.yml up -d "$2"
    ;;
  backend)
    echo "重启后端服务（重新编译）..."
    docker-compose -f docker-compose.dev.yml rm -sf backend && docker-compose -f docker-compose.dev.yml up -d backend
    ;;
  frontend)
    echo "重启前端服务..."
    docker-compose -f docker-compose.dev.yml restart frontend
    ;;
  *)
    echo "用法: $0 {start|stop|restart|logs|rebuild|backend|frontend} [服务名]"
    echo ""
    echo "开发流程:"
    echo "  1. 修改后端代码后: $0 backend  (或 restart backend)"
    echo "  2. 修改前端代码后: $0 frontend  (Vite 会热更新)"
    echo ""
    echo "服务:"
    echo "  backend  - Go 后端服务 (http://localhost:8080)"
    echo "  frontend - React 前端开发服务器 (http://localhost:5173)"
    echo "  nginx    - 生产 nginx 代理"
    ;;
esac
