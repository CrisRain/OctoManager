# OctoManger

多账户管理平台。支持通过 OctoModule（Python 脚本）处理任意类型账号的批量注册、验证等自定义动作；内置 Outlook 邮箱账号管理、Webhook 触发器、异步任务队列。

运行依赖：PostgreSQL、Redis、Python 3

---

## 文档导航

| 文档 | 内容 |
| --- | --- |
| [01 · 快速开始](docs/01-快速开始.md) | 5 分钟内跑起来 |
| [02 · 架构说明](docs/02-架构说明.md) | 各组件职责与数据流向 |
| [03 · 部署与配置](docs/03-部署与配置.md) | Docker / 本地开发 / 全部环境变量 |
| [04 · 鉴权与 API Key](docs/04-鉴权与API-Key.md) | 初始化向导、Admin Key、Webhook Key |
| [05 · 账号类型与账号](docs/05-账号类型与账号.md) | Account Type 定义、账号管理、批量操作 |
| [06 · OctoModule 开发指南](docs/06-OctoModule开发指南.md) | 写模块、Dry Run、依赖管理、接入 Trigger |
| [07 · 任务系统](docs/07-任务系统.md) | Job 生命周期、Worker、Asynq |
| [08 · Trigger 触发器](docs/08-Trigger触发器.md) | Webhook 接入、同步/异步模式 |
| [09 · 邮箱账号](docs/09-邮箱账号.md) | Outlook 账号、OAuth 配置、批量导入与注册 |

---

## 项目结构

```text
.
├── backend/            Go API 服务 + Asynq Worker
├── web/                React 控制台
├── scripts/python/     Python 执行桥 + OctoModule 脚本目录
├── configs/            默认配置文件
├── docs/               文档
├── docker-compose.yml
└── Dockerfile
```

---

## 一分钟跑起来

```bash
docker compose up -d --build
```

打开 `http://localhost:8080`，按初始化向导完成数据库迁移和首个 Admin Key 创建。

详细步骤见 [01 · 快速开始](docs/01-快速开始.md)。
