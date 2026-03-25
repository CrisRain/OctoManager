# OctoManger

> **多账号自动化任务管理平台** — 调度、执行、监控，一站式搞定。

![Go Version](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go)
![Vue Version](https://img.shields.io/badge/Vue-3.x-4FC08D?logo=vue.js)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-336791?logo=postgresql)
![License](https://img.shields.io/badge/License-待确认-lightgrey)
![Version](https://img.shields.io/badge/version-0.1.0-blue)

OctoManger 是一个以 **Plugin-first** 为核心理念的自动化平台。它通过 Python 插件描述对外部系统（GitHub、邮件、自定义 API 等）的操作，再由平台负责调度、执行、日志记录与错误重试。

---

## 目录

- [功能特性](#功能特性)
- [快速开始](#快速开始)
  - [Docker Compose（推荐）](#docker-compose推荐)
  - [本地开发](#本地开发)
- [项目结构](#项目结构)
- [环境变量](#环境变量)
- [插件开发](#插件开发)
- [文档索引](#文档索引)
- [贡献指南](#贡献指南)

---

## 功能特性

| 功能 | 说明 |
|------|------|
| **账号管理** | 统一管理多平台账号凭证，类型由插件声明 |
| **定时任务** | Cron 表达式调度，支持时区，数据库级分布式锁 |
| **Webhook 触发** | 公开 Webhook 端点，支持 Token 鉴权与同步/异步两种模式 |
| **后台 Agent** | 长驻后台进程，具备心跳监控与自动重启 |
| **邮件账号** | 内置 IMAP/Outlook OAuth 邮件账号管理与消息预览 |
| **实时日志** | 任务与 Agent 执行日志通过 SSE 实时流式推送 |
| **插件系统** | Python 插件子进程执行，声明式 JSON Schema 账号类型 |
| **Web UI** | Vue 3 单页应用，完整的 CRUD 界面与命令面板 |

---

## 快速开始

### Docker Compose（推荐）

```bash
# 1. 克隆仓库
git clone <repo-url> && cd OctoManger

# 2. 配置环境（可选：修改 compose 中的默认值）
# 默认: DB=octomanger, user=octo, password=octo, Redis无密码

# 3. 启动所有服务（PostgreSQL + Redis + API + Worker + 前端）
docker compose up --build

# 4. 访问 Web UI
open http://localhost:8080
```

> `start.sh` 会自动运行数据库迁移，无需手动执行。

### 本地开发

**前置要求**：Go 1.25+、Bun 1.x、PostgreSQL 16、Python 3.x（可选 Redis）

```bash
# ── 数据库 ──────────────────────────────────────────────
export DATABASE_DSN="postgres://octo:octo@localhost:5432/octomanger?sslmode=disable"

# 执行 GORM 自动迁移
go run ./apps/migrate migrate

# ── 后端 ────────────────────────────────────────────────
# 启动 API 服务器（监听 :8080）
go run ./apps/api

# 在另一终端启动 Worker（调度 + 执行）
go run ./apps/worker

# ── 前端 ────────────────────────────────────────────────
cd apps/web
bun install
bun run dev   # Vite 开发服务器，自动代理 /api/v2 → localhost:8080
```

---

## 项目结构

```
OctoManger/
├── apps/
│   ├── api/            # HTTP API 服务器入口
│   ├── worker/         # Cron 调度器 + 插件执行器入口
│   ├── migrate/        # 数据库 AutoMigrate 与旧库导入入口
│   └── web/            # Vue 3 前端（TypeScript + Tailwind CSS）
├── internal/
│   ├── domains/        # 8 个业务域（各含 transport/app/domain/infra 四层）
│   │   ├── account-types/
│   │   ├── accounts/
│   │   ├── agents/
│   │   ├── email/
│   │   ├── jobs/
│   │   ├── plugins/
│   │   ├── system/
│   │   └── triggers/
│   └── platform/       # 跨域基础设施（auth、DB、日志、运行时 DI）
├── plugins/
│   ├── modules/        # Python 插件模块目录
│   └── sdk/python/     # octo_runtime Python SDK
├── docker-compose.yml
├── Dockerfile
└── go.mod
```

详细架构说明见 [ARCHITECTURE.md](./ARCHITECTURE.md)。

---

## 环境变量

| 变量 | 默认值 | 必填 | 说明 |
|------|--------|:----:|------|
| `DATABASE_DSN` | — | ✅ | PostgreSQL 连接字符串 |
| `REDIS_ADDR` | — | ❌ | Redis 地址，缺失时降级运行 |
| `API_ADDR` | `:8080` | ❌ | HTTP 监听地址 |
| `WEB_DIST_DIR` | `/app/web-dist` | ❌ | 构建后的前端资源目录 |
| `PLUGINS_DIR` | `/app/plugins/modules` | ❌ | Python 插件模块目录 |
| `PLUGIN_SDK_DIR` | `/app/plugins/sdk/python` | ❌ | Python SDK 路径 |
| `PYTHON_BIN` | `python3` | ❌ | Python 解释器路径 |
| `PLUGINS_TIMEOUT_ACCOUNT` | `60s` | ❌ | 账号页面直连插件执行超时 |
| `PLUGINS_TIMEOUT_JOB` | `10m` | ❌ | Job 模式插件执行超时 |
| `PLUGINS_TIMEOUT_AGENT` | `0s` | ❌ | Agent 模式插件执行超时（`0s`=禁用） |
| `X_ADMIN_KEY` | — | ❌ | API 管理员密钥（空则无鉴权） |
| `LOG_LEVEL` | `info` | ❌ | 日志级别（debug/info/warn/error）|
| `WORKER_POLL_INTERVAL` | — | ❌ | Worker 轮询间隔 |

完整 Worker 变量见 [ARCHITECTURE.md § Worker](./ARCHITECTURE.md#worker)。

---

## 插件开发

插件是位于 `PLUGINS_DIR` 下的 Python 目录，包含：

```
plugins/modules/my-plugin/
├── main.py                       # 插件入口
└── requirements.txt              # Python 依赖
```

推荐写法是使用 `Module(...)` 定义插件能力，账号类型描述会由 SDK 自动生成。

**最简 `main.py`：**

```python
from octo import Module, success

module = Module(
    key="hello_demo",
    name="Hello Demo",
    category="generic",
    account_schema={
        "type": "object",
        "properties": {},
    },
)

@module.action("HELLO")
def hello(req):
    name = req.input.get("name", "world")
    return success({"message": f"Hello, {name}!"})

if __name__ == "__main__":
    module.serve()
```

SDK 文档与完整示例见 `plugins/sdk/python/`。

如果你要从零开始写一个新插件，直接阅读 [PLUGIN_DEVELOPMENT.md](./PLUGIN_DEVELOPMENT.md)。

---

## 文档索引

| 文档 | 内容 |
|------|------|
| [ARCHITECTURE.md](./ARCHITECTURE.md) | 系统架构、数据流、模块说明 |
| [API_REFERENCE.md](./API_REFERENCE.md) | 全量 REST API 端点参考 |
| [CONTRIBUTING.md](./CONTRIBUTING.md) | 贡献流程、编码规范、PR 指南 |
| [PLUGIN_DEVELOPMENT.md](./PLUGIN_DEVELOPMENT.md) | 插件开发完整教程 |
| [CHANGELOG.md](./CHANGELOG.md) | 版本变更历史 |

---

## 贡献指南

欢迎任何形式的贡献！请先阅读 [CONTRIBUTING.md](./CONTRIBUTING.md)。

提交 issue 或 PR 前，请确认：
- [ ] 后端：`go test ./...` 全部通过
- [ ] 前端：`bun run build` 无类型错误
- [ ] 数据库变更：更新 AutoMigrate 模型与兼容逻辑，并验证启动迁移通过
- [ ] Commit 遵循 Conventional Commits 格式（`feat:` / `fix:` / `docs:`）
