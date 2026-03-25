# Changelog

本文件遵循 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/) 规范，版本号遵循 [Semantic Versioning](https://semver.org/lang/zh-CN/)。

---

## [Unreleased]

### Added
- 完整文档套件：README.md、ARCHITECTURE.md、API_REFERENCE.md、CONTRIBUTING.md

---

## [0.1.0] — 2026-03-21

> V2 全面重构版本。相较 V1（`backend/`），本版本为完全重写，不兼容旧版 API。

### Added

**后端**
- 基于 `cloudwego/hertz` 的高性能 HTTP 服务器，全量 `/api/v2/` 路由前缀
- 8 个业务域，各含严格四层架构（transport → app → domain → infra）：
  - `account-types`：插件声明的账号凭证类型管理
  - `accounts`：多平台账号凭证 CRUD 与 ad-hoc 执行
  - `agents`：长驻后台 Python 进程管理（启动/停止/心跳/SSE 日志）
  - `email`：IMAP 与 Outlook OAuth 邮件账号管理、消息预览
  - `jobs`：任务定义、Cron 调度（数据库级分布式锁）、手动入队、SSE 执行日志
  - `plugins`：Python 插件目录扫描、账号类型自动同步、插件设置
  - `system`：健康检查、看板统计、`system_configs` 键值配置
  - `triggers`：Webhook 触发器（Token bcrypt 鉴权、同步/异步两种模式）
- Worker 进程：Cron 调度器 + 任务队列消费者 + Agent 状态机
- Python 子进程执行模型，通过 stdout JSON 行传递日志与事件
- `internal/platform/runtime/runtime.go`：统一依赖注入容器（App struct）
- API Key 认证中间件（`X-Admin-Key` / `Authorization: Bearer`），支持开发无鉴权模式
- 数据库迁移运行器（`apps/migrate`），追踪已应用迁移，支持 `import-legacy` 子命令

**前端**
- Vue 3 + TypeScript + Tailwind CSS 4 + Pinia 单页应用
- 30 个页面，覆盖全部 8 个域的 CRUD 操作
- 命令面板（Command Palette）
- SSE 实时日志查看器（`vue-virtual-scroller` 虚拟渲染）
- 类型安全 API 客户端（OpenAPI 代码生成）
- OAuth 回调页面（Outlook 授权流程）

**基础设施**
- PostgreSQL 16 + GORM ORM，JSONB 灵活数据存储
- Redis 7（可选），降级时不影响核心功能
- Docker 多阶段构建（web + Go 二进制 + runtime 三阶段）
- `docker-compose.yml` 一键启动全栈开发环境

**插件 SDK（Python）**
- `octo_runtime` Python 包，提供 `ActionRouter`、`OctoClient`、`emit_log`、`emit_daemon_*` 等核心 API
- GitHub 插件参考实现（`plugins/modules/github/`）

### Changed

- 项目结构从 `backend/` 单目录重构为 `apps/` + `internal/` Go workspace
- 前端从 React（V1）迁移到 Vue 3
- API 路径前缀从 `/api/v1/` 升级为 `/api/v2/`
- HTTP 框架从 `net/http` 迁移到 `cloudwego/hertz`
- 账号系统新增 `account_type_id` 关联，支持多插件类型
- 日志系统从标准库迁移到 `go.uber.org/zap` + `lumberjack`

### Fixed

- `system_configs` 表旧列名 `value` 重命名为 `value_json`（迁移 `0003`），修复 V2 配置接口 500 错误

### Removed

- V1 `backend/` 目录（旧版 monolith 代码）
- V1 `web/` 目录（React 前端）
- 旧版配置文件 `configs/config.yaml`

---

## V1 历史摘要（已归档）

> V1 代码存档于 Git 历史（commit `539d55c`），不再维护。

- `2026-03-07` — 首次提交，基于 `net/http` + React 的 V1 实现
- `2026-03-xx` — 新增 Daemon 模式（持久化后台模块执行）
- `2026-03-xx` — 新增 TLS 证书管理与 HTTP 重定向服务器
- `2026-03-xx` — 新增任务运行详情组件与日志页面

---

[Unreleased]: https://github.com/CrisRain/OctoManger/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/CrisRain/OctoManger/releases/tag/v0.1.0
