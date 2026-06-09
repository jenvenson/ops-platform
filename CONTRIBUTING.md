# Contributing to OPS Platform

Thank you for your interest in contributing!

## Development Setup

### Prerequisites

- Docker 20.10+ and Docker Compose 2.0+
- Go 1.25+ (for backend development outside Docker)
- Node.js 20+ (for frontend development outside Docker)
- pnpm (preferred) or npm

### Quick Start

```bash
git clone git@github.com:jenvenson/ops-platform.git
cd ops-platform/deploy
cp .env.example .env
# Edit .env: set DB_PASSWORD, REDIS_PASSWORD, JWT_SECRET
docker compose -f docker-compose.dev.yml up -d
```

- **Frontend**: http://localhost:8890
- **Backend API**: http://localhost:8080
- **Default login**: admin / admin123 (change on first login)

### Local Development (without Docker for frontend/backend)

```bash
# Backend
cd backend
go run ./cmd/server/main.go

# Frontend (separate terminal)
cd frontend
pnpm install && pnpm dev
```

The frontend dev server proxies `/api` and `/auth` to the backend. See `vite.config.ts` for proxy settings.

## Project Architecture

```
backend/           Go API server (Gin + GORM)
  cmd/server/      Entry point
  internal/        Business modules (auth, cmdb, monitor, security, assistant, etc.)
  pkg/             Shared packages (config, logger)
  migrations/      Database migrations
frontend/          React + TypeScript + Vite + Ant Design
  src/pages/       Page components by module
  src/api/         API client modules
  src/components/  Shared components
deploy/            Docker Compose + Nginx deployment configs
```

See [README.md](README.md) for the complete project structure.

## Commit Convention

- One logical change per commit
- Write clear, imperative commit messages: "Add X", "Fix Y", "Update Z"
- Reference issues when applicable

## Code Style

- **Go**: Follow standard Go conventions. Run `gofmt` before committing. Always handle errors with context.
- **TypeScript/React**: Prefer ESM. Avoid `any` (except at API boundaries). Use existing Ant Design components. Run `npx tsc --noEmit` before committing.
- **SQL**: Use the migration system (`backend/migrations/`). Never modify existing migrations — always create a new one.

## Testing

```bash
# Go tests
cd backend && go test ./...

# Frontend type check
cd frontend && npx tsc --noEmit

# CI runs both on every push (see .github/workflows/ci.yml)
```

## Pull Requests

1. Fork the repository
2. Create a feature branch from `main`
3. Make your changes (one logical change per PR)
4. Ensure CI passes: `cd backend && go vet ./... && go test ./...`
5. Submit a PR with a clear description of what and why

## Reporting Issues

- **Bug reports**: Include steps to reproduce, expected vs actual behavior, relevant logs, and environment details
- **Feature requests**: Describe the use case, expected behavior, and any alternatives considered
- **Security issues**: See [SECURITY.md](SECURITY.md) — do not file public issues for vulnerabilities
