# 部署文档说明

`docs/deploy.md` 不再维护完整部署流程。

当前仅保留两份最新部署文档：

- [README.md](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/README.md)：项目入口与部署概览
- [deploy/DEPLOY.md](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/deploy/DEPLOY.md)：生产部署与本地执行发布清单

推荐阅读顺序：

1. 先看 `README.md` 了解整体结构和部署入口
2. 再看 `deploy/DEPLOY.md` 执行首次部署、迁移发布和上线检查

当前生产发布统一以 `deploy/deploy-init.sh` 和 `deploy/deploy-update.sh` 为准。
