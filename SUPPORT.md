# Support

## Documentation

- [README.md](README.md) — Project overview, features, and quick start
- [docs/](docs/) — User manual, design documents, and testing guides
- [deploy/DEPLOY.md](deploy/DEPLOY.md) — Production deployment guide

## Community Support

- **GitHub Issues**: [Report bugs or request features](https://github.com/jenvenson/ops-platform/issues)
- **GitHub Discussions**: [Ask questions and share ideas](https://github.com/jenvenson/ops-platform/discussions)

## Common Issues

### Backend fails to start

Check the logs:
```bash
docker compose logs backend
```

Common causes:
- Database not ready: wait for MySQL to become healthy
- Migration errors: check `schema_migrations` table
- Missing `.env` variables: ensure `DB_PASSWORD`, `REDIS_PASSWORD`, `JWT_SECRET` are set

### Frontend shows blank page

```bash
# Check Nginx error log
docker compose exec nginx cat /var/log/nginx/error.log

# Verify static files exist
docker compose exec nginx ls /usr/share/nginx/html/
```

### AI Assistant not responding

Ensure the assistant model is configured in System Settings → AI Model Settings. The assistant requires an OpenAI-compatible API endpoint (Ollama, DeepSeek, Qwen, etc.).

### Port conflicts

If ports 80, 3306, 6379, or 8080 are in use, use the dev configuration with alternative ports:
```bash
docker compose -f docker-compose.dev.yml up -d
```

## Enterprise Support

For production deployment assistance, see [deploy/DEPLOY.md](deploy/DEPLOY.md).
