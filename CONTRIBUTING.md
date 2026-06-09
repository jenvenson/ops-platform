# Contributing to OPS Platform

Thank you for your interest in contributing!

## Development Setup

```bash
git clone git@github.com:jenvenson/ops-platform.git
cd ops-platform/deploy
cp .env.example .env
# Edit .env with your DB_PASSWORD, REDIS_PASSWORD, JWT_SECRET
docker compose -f docker-compose.dev.yml up -d
```

- **Frontend**: http://localhost:8890
- **Backend API**: http://localhost:8080

## Commit Convention

- One logical change per commit
- Write clear commit messages summarizing what and why

## Code Style

- **Go**: Follow standard Go conventions. Always handle errors.
- **TypeScript**: Prefer ESM. Avoid `any` (except at boundaries).
- **Frontend**: Follow Ant Design conventions. Use existing components when possible.

## Reporting Issues

- Bug reports should include reproduction steps and relevant logs
- Feature requests should describe the use case and expected behavior

## Pull Requests

1. Fork the repository
2. Create a feature branch
3. Ensure local tests pass: `cd backend && go test ./...`
4. Submit a PR with a description of the changes

## Project Structure

See [README.md](README.md) for the full project structure.
