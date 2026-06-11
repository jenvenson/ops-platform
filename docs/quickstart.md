# 快速开始

五分钟在本机跑起 OPS Platform，完成第一次登录。

## 前置条件

| 依赖 | 版本要求 | 说明 |
|------|----------|------|
| Docker | 20.10+ | 必须 |
| Docker Compose | v2（`docker compose`） | 必须 |
| 可用内存 | ≥ 4 GB | MySQL + Redis + 后端 + 前端同时运行 |
| 可用磁盘 | ≥ 5 GB | 镜像 + 数据卷 |

> **验证方式**：运行 `docker compose version`，输出版本号即表示满足要求。

---

## 第一步：获取代码

```bash
git clone https://github.com/jenvenson/ops-platform.git
cd ops-platform
```

---

## 第二步：配置环境变量

```bash
cp deploy/.env.example deploy/.env
```

用任意编辑器打开 `deploy/.env`，**必须修改以下三项**（其余保持默认即可）：

```ini
DB_PASSWORD=请改为一个安全的数据库密码
REDIS_PASSWORD=请改为一个安全的缓存密码
JWT_SECRET=请改为一个随机字符串（至少32位）
```

> **国内用户**：如果依赖下载慢，在 `.env` 末尾追加：
> ```ini
> GOPROXY=https://goproxy.cn,direct
> ```

---

## 第三步：启动服务

```bash
docker compose -f deploy/docker-compose.dev.yml -p ops-dev up -d
```

**首次启动**会从 Docker Hub 拉取基础镜像（golang、mysql、redis、nginx）并下载 Go 依赖，约需 **3–5 分钟**（取决于网速），请耐心等待。

查看启动进度：

```bash
docker logs -f ops-backend-dev
```

看到类似以下输出即表示后端就绪：

```
[GIN-debug] Listening and serving HTTP on :8080
```

按 `Ctrl+C` 退出日志跟踪（服务仍在后台运行）。

> **端口说明**：容器内部使用标准端口（如后端 8080），宿主机映射到不同端口以避免冲突：
>
> | 服务 | 宿主机端口 | 容器端口 |
> |------|-----------|---------|
> | 前端 | 18890 | 80 |
> | 后端 API | 28080 | 8080 |
> | MySQL | 23306 | 3306 |
> | Redis | 16379 | 6379 |

---

## 第四步：验证并登录

**验证后端健康状态：**

```bash
curl http://localhost:28080/health
# 正常返回：{"status":"ok","checks":{"database":"ok"}}
```

**打开浏览器访问：**

```
http://localhost:18890
```

使用默认账号登录：

| 字段 | 值 |
|------|----|
| 用户名 | `admin` |
| 密码 | `admin123` |

> 首次登录后，请在右上角「个人中心」修改密码。

---

## 常见问题

<details>
<summary><b>启动后访问 18890 显示空白或无法连接</b></summary>

前端容器可能仍在安装依赖，等待 1–2 分钟后刷新。也可查看前端日志：

```bash
docker logs -f ops-frontend-dev
```

看到 `VITE ready` 字样即表示前端就绪。
</details>

<details>
<summary><b>后端日志报 "dial tcp: connection refused"</b></summary>

MySQL 或 Redis 尚未完成初始化。稍等片刻，后端会自动重连。若持续报错，检查 `deploy/.env` 中密码是否已正确填写。
</details>

<details>
<summary><b>端口冲突（18890 / 28080 / 23306 已被占用）</b></summary>

修改 `deploy/.env` 中对应的端口变量，或直接编辑 `deploy/docker-compose.dev.yml` 的 `ports` 映射，然后重新启动：

```bash
docker compose -f deploy/docker-compose.dev.yml -p ops-dev down
docker compose -f deploy/docker-compose.dev.yml -p ops-dev up -d
```
</details>

<details>
<summary><b>MySQL 认证插件报错</b></summary>

```bash
docker exec ops-mysql mysql -uroot -p"${DB_PASSWORD}" -e \
  "ALTER USER 'root'@'%' IDENTIFIED WITH mysql_native_password BY '${DB_PASSWORD}'; FLUSH PRIVILEGES;"
```
</details>

---

## 停止服务

```bash
docker compose -f deploy/docker-compose.dev.yml -p ops-dev down
```

数据卷不会被删除，下次 `up -d` 后数据仍在。如需彻底清除数据：

```bash
docker compose -f deploy/docker-compose.dev.yml -p ops-dev down -v
```

---

## 下一步

- [功能模块说明](../README.md#功能概览)
- [接入 AI 运维小助手](../README.md#运维小助手)
- [生产环境部署](../deploy/DEPLOY.md)
- [用户手册](user_manual.md)
