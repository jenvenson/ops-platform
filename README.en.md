# OPS Platform

<div align="center">

**All-in-one Ops Management Platform — CMDB, CI/CD, Security Scanning, Alerting & Monitoring**

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![CI](https://github.com/jenvenson/ops-platform/actions/workflows/ci.yml/badge.svg)](https://github.com/jenvenson/ops-platform/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go)](https://go.dev/)
[![React](https://img.shields.io/badge/React-18-61DAFB?logo=react)](https://react.dev/)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.x-3178C6?logo=typescript)](https://www.typescriptlang.org/)
[![Docker](https://img.shields.io/badge/Docker-20.10+-2496ED?logo=docker)](https://www.docker.com/)

[中文](README.md) | English

</div>

---

## Features

| Module | Description |
|--------|-------------|
| **CMDB** | Project, environment, server, and application asset management |
| **CI/CD** | Jenkins integration with application deployment and archive release |
| **Aggregate Packaging** | Multi-application aggregate packaging driven by Consul configuration |
| **Consul Management** | Configuration management, batch copy, pipeline replacement |
| **Alert Center** | Rule management, contacts, channels, templates, webhooks |
| **Monitoring** | Grafana dashboard proxy, health checks, Prometheus integration |
| **Security Scanning** | Host scanning, web scanning, vulnerability management, FIM file integrity monitoring |
| **AI Assistant** | AI chat assistant supporting local Ollama models and mainstream third-party models; replies follow the UI language |
| **Internationalization** | One-click Chinese/English UI switching; browser title and site name translate accordingly (custom site names stay verbatim) |
| **Access Control** | JWT authentication, RBAC role permissions, menu-level control |
| **Audit Log** | Operation auditing and platform event streaming |

## Screenshots

| Dashboard | CMDB Projects |
|-----------|---------------|
| ![Dashboard](docs/screenshots/dashboard.png) | ![CMDB](docs/screenshots/cmdb-projects.png) |

| Dashboard (English) | Security Scanning |
|---------------------|-------------------|
| ![Dashboard EN](docs/screenshots/dashboard-en.png) | ![Security](docs/screenshots/security-tasks.png) |

| Deploy Release | Notification Templates (English) | AI Assistant |
|----------------|----------------------------------|--------------|
| ![Deploy](docs/screenshots/deploy-release.png) | ![Templates EN](docs/screenshots/alert-templates-en.png) | ![AI Assistant](docs/screenshots/ai-chatbot.png) |

## Tech Stack

**Backend**: Go 1.25 · Gin · GORM · MySQL 8.0 · Redis 7.4 · Zap

**Frontend**: React 18 · TypeScript · Vite · Ant Design 5 · Axios

**Infrastructure**: Docker Compose · Nginx · Ollama (optional)

## Quick Start

### 1. Clone the repository

```bash
git clone git@github.com:jenvenson/ops-platform.git
cd ops-platform
```

### 2. Configure environment variables

```bash
cp deploy/.env.example deploy/.env
```

Edit `deploy/.env` and **change at least these three values**:

```ini
DB_PASSWORD=your_secure_mysql_password
REDIS_PASSWORD=your_secure_redis_password
JWT_SECRET=your_jwt_secret_key_change_in_production
```

### 3. Start the services

```bash
docker compose -f deploy/docker-compose.dev.yml -p ops-dev up -d
```

The first startup pulls images and installs dependencies, which takes about 3–5 minutes.

### 4. Verify

```bash
curl http://localhost:28080/health
# {"status":"ok","checks":{"database":"ok"}}
```

Open **http://localhost:18890** in your browser and log in with the default account.

### Service endpoints

| Service | Address | Description |
|---------|---------|-------------|
| Frontend | http://localhost:18890 | Web management UI |
| Backend API | http://localhost:28080 | REST API |
| MySQL | localhost:23306 | Direct database access |
| Redis | localhost:16379 | Direct cache access |

### Default account

| Field | Value |
|-------|-------|
| Username | `admin` |
| Password | `admin123` |

> Please change the password in "Profile" after the first login.

### Development mode (all services run in Docker)

`docker-compose.dev.yml` starts everything: MySQL, Redis, backend (`go run` inside a golang container), frontend (Vite inside a node container), and Nginx. Source directories are mounted into the containers, so no local Go / Node installation is needed:

```bash
# Frontend: Vite hot-reload — code changes apply automatically
# Backend: no hot-reload — restart the container after code changes
docker restart ops-backend-dev

# Tail logs
docker logs -f ops-backend-dev
docker logs -f ops-frontend-dev
```

### Troubleshooting

<details>
<summary><b>MySQL connection failure / authentication plugin error</b></summary>

Some clients do not support MySQL 8.0's `caching_sha2_password`:

```bash
docker exec ops-mysql mysql -uroot -p -e \
  "ALTER USER 'root'@'%' IDENTIFIED WITH mysql_native_password BY 'your_password'; FLUSH PRIVILEGES;"
```
</details>

<details>
<summary><b>Frontend cannot reach the backend</b></summary>

Make sure `JWT_SECRET` is set in `deploy/.env` and the backend container `ops-backend-dev` is running:

```bash
docker logs ops-backend-dev --tail 20
```
</details>

<details>
<summary><b>Port conflicts</b></summary>

Change the port variables in `deploy/.env` or edit the `ports` mappings in `deploy/docker-compose.dev.yml`.
</details>

<details>
<summary><b>Slow network in mainland China / dependency download failures</b></summary>

Set the Go module proxy in `deploy/.env`:

```ini
GOPROXY=https://goproxy.cn,direct
```
</details>

## Project Structure

```
├── backend/                # Go backend
│   ├── cmd/server/         # Entry point
│   ├── internal/           # Business modules (cmdb/security/alert/assistant/...)
│   ├── pkg/                # Shared packages (config/logger/jenkins/consul)
│   ├── configs/            # Configuration files
│   ├── migrations/         # Database migrations
│   └── scripts/            # Init scripts
├── frontend/               # React frontend
│   └── src/
│       ├── api/            # API clients
│       ├── components/     # Shared components (MainLayout, AIChatbot)
│       ├── pages/          # Pages (cmdb/security/alarm/deploy/...)
│       └── styles/         # Theme styles
├── deploy/                 # Deployment configs
│   ├── docker-compose.yml        # Production
│   ├── docker-compose.dev.yml    # Development
│   ├── Dockerfile.backend        # Backend image
│   ├── Dockerfile.frontend       # Frontend image
│   ├── nginx.conf.template       # Nginx template
│   ├── .env.example              # Env var example
│   ├── deploy-init.sh            # First deployment
│   └── deploy-update.sh          # Iterative updates
├── docs/                   # Docs and screenshots
└── migrations/             # Legacy supplementary migrations
```

## Architecture

```
Browser → Nginx (:80)
            ├── /api/*        → Backend (:8080) → MySQL + Redis
            ├── /grafana-proxy → Grafana
            └── /*            → Frontend static files
```

Backend module dependencies:

```
auth → cmdb / security / alert / consul / cicd / assistant
                   ↓
            platformevent / platformobject
                   ↓
              database (MySQL + Redis)
```

## AI Assistant

A built-in AI chat assistant that can run local models via Ollama or connect to third-party model APIs:

```bash
# Local Ollama
ASSISTANT_PROVIDER=ollama
OLLAMA_BASE_URL=http://localhost:11434
OLLAMA_CHAT_MODEL=qwen3:8b

# Third-party models (DeepSeek / Qwen / GLM / Kimi / MiniMax / Doubao, etc.)
ASSISTANT_PROVIDER=deepseek
ASSISTANT_API_KEY=sk-your-api-key
```

Supported tool calls: CMDB queries, alert management, deployment operations, security scanning.

## Documentation

- [Quick Start](docs/quickstart.en.md)
- [Deployment Guide](deploy/DEPLOY.md)
- [User Manual](docs/user_manual.md)
- [Testing Guide](docs/testing.md)
- [Contributing Guide](CONTRIBUTING.md)

## Community

- [Submit an Issue](https://github.com/jenvenson/ops-platform/issues)
- [Report a Security Vulnerability](SECURITY.md)

## License

This project is licensed under the [MIT License](LICENSE) — free to use, modify, and distribute.
