# Changelog

## [Unreleased]

### Added
- Internationalization (i18n): full Chinese/English UI switching with 13 translation namespaces
- AI Assistant replies follow the current UI language
- Browser title and site name translate with UI language (custom site names stay verbatim)
- DB-backed AI model configuration with hot-reload
- Multi-provider AI model support (DeepSeek, Qwen, Zhipu, Kimi, MiniMax, Doubao, Baichuan)
- Custom provider support via `ASSISTANT_BASE_URL`

### Changed
- AI model config migrated from env vars to database with admin UI
- Provider abstraction refactored for third-party model support

### Security
- Removed all hardcoded credentials and internal paths
- Replaced company-specific identifiers for open-source release

## [0.1.0] - 2026-06-08

### Added
- Initial open-source release
- CMDB: project, environment, server, and application management
- CI/CD: Jenkins integration with deploy and archive workflows
- Security: host scanning, web scanning, vulnerability management, FIM monitoring
- Alerting: rule management, contacts, channels, templates, webhook
- Consul: configuration management and batch operations
- Monitoring: Grafana dashboard proxy, health checks, Prometheus integration
- AI Assistant: Ollama-powered chat with tool-calling support
- RBAC: JWT authentication with role-based menu access
- Audit: operation logging and platform event streaming
