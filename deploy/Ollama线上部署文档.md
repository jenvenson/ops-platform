# Ollama 线上部署文档

本文档用于说明如何在 OPS Platform 线上环境中通过 Docker 部署 `Ollama`，并接入本地模型 `qwen2.5:1.5b`。

适用目标：

- 线上 Linux 服务器
- Docker / Docker Compose 已安装
- OPS Platform 后端通过内网或本机访问 Ollama

推荐调用链：

```text
浏览器
-> OPS Platform 前端
-> OPS Platform 后端 /api/assistant/*
-> Ollama http://127.0.0.1:11434
-> qwen2.5:1.5b
```

注意：

- 不建议前端直接访问 Ollama
- 不建议将 `11434` 暴露到公网
- 建议只允许后端服务访问 Ollama

## 1. 部署目标

本方案完成以下内容：

- 通过 Docker 启动 Ollama 服务
- 持久化模型目录
- 拉取 `qwen2.5:1.5b`
- 可选拉取 `qwen3-embedding:4b`
- 配置 OPS Platform 后端环境变量
- 验证助手链路可用

## 2. 前置条件

服务器建议配置：

- CPU：4 核以上
- 内存：16GB 以上更稳妥
- 磁盘：至少预留 20GB

软件要求：

- Docker 20+
- Docker Compose 2+

检查命令：

```bash
docker --version
docker compose version
```

## 3. 目录规划

建议线上目录：

```bash
/opt/ops-platform
/opt/ops-platform/deploy
/opt/ops-platform/ollama
/opt/ops-platform/ollama/data
```

其中：

- `/opt/ops-platform/ollama` 用于存放 Ollama 启动脚本和 compose 文件
- `/opt/ops-platform/ollama/data` 用于持久化 Ollama 模型
- 目录结构统一收口，便于运维管理

## 4. Docker 启动 Ollama

### 4.1 单独启动方式

如果你想先独立验证 Ollama，可直接运行：

```bash
mkdir -p /opt/ops-platform/ollama/data

docker run -d \
  --name ollama \
  --restart unless-stopped \
  -p 127.0.0.1:11434:11434 \
  -v /opt/ops-platform/ollama/data:/root/.ollama \
  ollama/ollama
```

说明：

- `127.0.0.1:11434:11434` 表示只监听本机
- `/opt/ops-platform/ollama/data:/root/.ollama` 用于持久化模型文件

查看状态：

```bash
docker ps | grep ollama
docker logs -f ollama
```

### 4.2 使用 docker-compose

如果希望纳入统一编排，可在 `/opt/ops-platform/ollama` 下放置独立编排文件，例如：

```yaml
services:
  ollama:
    image: ollama/ollama
    container_name: ollama
    restart: unless-stopped
    ports:
      - "127.0.0.1:11434:11434"
    volumes:
      - /opt/ops-platform/ollama/data:/root/.ollama
```

启动命令：

```bash
mkdir -p /opt/ops-platform/ollama/data
cd /opt/ops-platform/ollama
docker compose -f docker-compose.ollama.yml up -d
```

仓库中已提供可直接下发的模板文件：

- `deploy/ollama/docker-compose.ollama.yml`
- `deploy/ollama/run-ollama.sh`

建议线上目录结构：

```bash
/opt/ops-platform/ollama/
├── docker-compose.ollama.yml
├── run-ollama.sh
└── data/
```

## 5. 模型拉取

### 5.1 拉取对话模型

启动容器后执行：

```bash
docker exec -it ollama ollama pull qwen2.5:1.5b
```

### 5.2 拉取向量模型

如果后续要接本地 embedding / RAG，可继续执行：

```bash
docker exec -it ollama ollama pull qwen3-embedding:4b
```

### 5.3 查看已安装模型

```bash
docker exec -it ollama ollama list
```

预期可以看到：

```text
qwen2.5:1.5b
qwen3-embedding:4b
```

## 6. 离线环境模型导入

如果线上机器没有外网，可以在有网机器上先拉模型，再把 `/root/.ollama` 或 `~/.ollama/models` 打包拷贝到线上。

示例：

```bash
tar czf ollama-models.tar.gz ~/.ollama
scp ollama-models.tar.gz root@your-server:/opt/ops-platform/ollama/
```

线上解压：

```bash
mkdir -p /opt/ops-platform/ollama/import
tar xzf /opt/ops-platform/ollama/ollama-models.tar.gz -C /opt/ops-platform/ollama/import
cp -R /opt/ops-platform/ollama/import/.ollama/* /opt/ops-platform/ollama/data/
```

然后重启容器：

```bash
docker restart ollama
```

再次验证：

```bash
docker exec -it ollama ollama list
```

## 7. Ollama 服务验证

### 7.1 查看标签列表

```bash
curl http://127.0.0.1:11434/api/tags
```

### 7.2 测试模型生成

```bash
curl http://127.0.0.1:11434/api/generate -d '{
  "model": "qwen2.5:1.5b",
  "prompt": "你好，请用一句话介绍你自己",
  "stream": false
}'
```

如果返回正常 JSON，说明模型可用。

## 8. OPS Platform 后端配置

在线上 `deploy/.env` 或后端环境变量中加入：

```bash
ASSISTANT_PROVIDER=ollama
OLLAMA_BASE_URL=http://127.0.0.1:11434
OLLAMA_CHAT_MODEL=qwen2.5:1.5b
OLLAMA_EMBED_MODEL=qwen3-embedding:4b
ASSISTANT_REQUEST_TIMEOUT_SEC=20
ASSISTANT_RATE_LIMIT_PER_MINUTE=30
ASSISTANT_MAX_MESSAGE_RUNES=1000
```

如果后端和 Ollama 不在同一台机器，可改成内网地址：

```bash
OLLAMA_BASE_URL=http://10.x.x.x:11434
```

但仍然建议只开放内网，不开放公网。

## 9. 重启 OPS Platform 后端

如果线上是当前项目这套 Docker Compose 部署，修改配置后重启后端：

```bash
cd /opt/ops-platform/deploy
docker-compose restart backend
```

或使用现有迭代脚本：

```bash
./deploy-update.sh backend
```

## 10. 联调验证

### 10.1 验证 assistant 会话接口

登录平台后验证：

```bash
curl -H "Authorization: Bearer <token>" \
  http://127.0.0.1:8080/api/assistant/sessions
```

### 10.2 验证 assistant 消息接口

```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  http://127.0.0.1:8080/api/assistant/messages \
  -d '{
    "sessionId": "your-session-id",
    "message": "最近有哪些未恢复告警"
  }'
```

如果响应中的 `model` 为：

```text
qwen2.5:1.5b
```

说明当前已经通过 Ollama 调用了本地模型。  
如果返回：

```text
ops-assistant
```

说明当前走的是后端 fallback。

## 11. 常用运维命令

查看 Ollama 日志：

```bash
docker logs -f ollama
```

查看模型列表：

```bash
docker exec -it ollama ollama list
```

进入容器：

```bash
docker exec -it ollama sh
```

重启 Ollama：

```bash
docker restart ollama
```

停止 Ollama：

```bash
docker stop ollama
```

## 12. 常见问题

### 12.1 模型拉取很慢

原因：

- 模型文件较大
- 服务器网络不稳定

建议：

- 在可联网机器先拉取再离线导入
- 模型目录放在大盘

### 12.2 assistant 仍然显示 fallback

排查顺序：

1. `docker exec -it ollama ollama list`
2. `curl http://127.0.0.1:11434/api/tags`
3. 检查 `OLLAMA_BASE_URL`
4. 检查 `OLLAMA_CHAT_MODEL=qwen2.5:1.5b`
5. 查看后端日志

### 12.3 服务器内存不足

建议：

- 优先只部署 `qwen2.5:1.5b`
- 不要同时加载过多模型
- 内存不足时先退到更小模型验证链路

## 13. 推荐最终配置

对当前 OPS Platform 项目，推荐线上最小可用组合：

```text
Ollama 容器
+ qwen2.5:1.5b
+ qwen3-embedding:4b
+ OPS Platform assistant 后端
```

推荐方式：

- Ollama 本机监听 `127.0.0.1:11434`
- 后端通过 `OLLAMA_BASE_URL=http://127.0.0.1:11434` 调用
- 不对公网暴露 Ollama 端口

## 14. 当前落地约定

当前线上部署按以下约定执行：

- Ollama 部署目录：`/opt/ops-platform/ollama`
- 启动脚本放在：`/opt/ops-platform/ollama/run-ollama.sh`
- 持久化目录：`/opt/ops-platform/ollama/data`
- 模型拉取由运维手动执行，不写入启动脚本
