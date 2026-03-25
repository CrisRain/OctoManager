# Contributing to OctoManger

感谢你的贡献！本文档面向**代码贡献者**，描述开发流程、编码规范和 PR 标准。

---

## 目录

- [开发环境搭建](#开发环境搭建)
- [项目结构导览](#项目结构导览)
- [后端开发规范](#后端开发规范)
  - [新增业务域](#新增业务域)
  - [数据库变更](#数据库变更)
  - [测试要求](#测试要求)
- [前端开发规范](#前端开发规范)
- [插件开发规范](#插件开发规范)
- [提交与 PR 规范](#提交与-pr-规范)
- [代码审查标准](#代码审查标准)

---

## 开发环境搭建

### 前置要求

| 工具 | 版本 | 用途 |
|------|------|------|
| Go | 1.25+ | 后端编译运行 |
| Bun | 1.x | 前端包管理与构建 |
| PostgreSQL | 16 | 主数据库 |
| Python | 3.x | 插件执行 |
| Redis | 7（可选）| 缓存，缺失时降级 |

### 本地启动

```bash
# 1. 克隆并进入仓库
git clone <repo-url> && cd OctoManger

# 2. 设置必要环境变量
export DATABASE_DSN="postgres://octo:octo@localhost:5432/octomanger?sslmode=disable"
export ADMIN_KEY="dev-only-key"     # 留空则无鉴权；兼容旧变量 X_ADMIN_KEY

# 3. 执行数据库自动迁移
go run ./apps/migrate migrate

# 4. 启动 API 服务（终端 1）
go run ./apps/api

# 5. 启动 Worker（终端 2）
go run ./apps/worker

# 6. 启动前端开发服务器（终端 3）
cd apps/web && bun install && bun run dev
# 访问 http://localhost:5173（代理 /api/v2 → :8080）
```

### Docker 一键启动

```bash
docker compose up --build
# 访问 http://localhost:8080
```

---

## 项目结构导览

在修改代码前，请先理解以下关键位置：

```
关键文件                                    用途
─────────────────────────────────────────────────────────────────
internal/platform/runtime/runtime.go        依赖注入根节点，注册新服务于此
internal/domains/{domain}/transport/http.go HTTP 路由注册
internal/domains/{domain}/app/service.go    业务逻辑
internal/domains/{domain}/domain/types.go   核心数据类型
internal/domains/{domain}/infra/postgres/   数据库仓库
internal/platform/database/                 GORM 连接与 AutoMigrate 定义
apps/web/src/api/                           前端 API 封装层
apps/web/src/store/                         Pinia 状态
apps/web/src/pages/                         页面组件
plugins/sdk/python/octo/                    插件 Python SDK
```

---

## 后端开发规范

### 代码风格

- 所有 Go 代码须通过 `gofmt` 格式化（使用 Tab 缩进，不要 Space）
- 遵循现有域的命名惯例，不引入新模式
- 业务逻辑限定在 `app/` 层；`transport/` 只做请求解析和响应序列化，不含业务规则
- `domain/` 层的类型零外部依赖（不引 GORM、不引 Hertz）

### 新增业务域

按以下步骤新增一个域（以 `foo` 为例）：

```bash
mkdir -p internal/domains/foo/{transport,app,domain,infra/postgres}
```

1. **`domain/types.go`** — 定义核心实体（纯 Go struct，无 GORM tag）
2. **`infra/postgres/repository.go`** — GORM 实现，定义 Repository 接口
3. **`app/service.go`** — 业务逻辑，持有 Repository 接口
4. **`transport/http.go`** — 注册路由，调用 Service
5. **`internal/platform/runtime/runtime.go`** — 在 `App` struct 中添加字段，在 `Bootstrap()` 中初始化
6. **`apps/api/main.go`** — 调用 `RegisterFooRoutes(app, router)`

### 数据库变更

数据库结构由 GORM AutoMigrate 维护，表结构变更请同步更新 `internal/platform/database/` 下的模型定义与兼容逻辑，然后执行：

```bash
go run ./apps/migrate migrate
```

涉及历史字段兼容或 GORM 无法表达的索引/约束时，请在 AutoMigrate 逻辑中补充显式 SQL。

### 测试要求

```bash
# 运行全部测试
go test ./...

# 针对某个域
go test ./internal/domains/jobs/...

# 运行指定测试函数
go test ./apps/api -run TestJobExecution
```

- 新增域逻辑须包含 `app/` 层的单元测试
- Repository 层的行为变更须有集成测试覆盖
- HTTP handler 的新端点须有端到端测试

---

## 前端开发规范

### 代码风格

| 规则 | 要求 |
|------|------|
| 缩进 | 2 空格 |
| 引号 | 双引号 |
| 分号 | 必须 |
| 组件命名 | PascalCase（`JobEditPage.vue`）|
| Composable | `useXxx`（`useJobs.ts`）|
| Store | 按域分文件（`store/jobs.ts`）|

### 新增页面

1. 在 `apps/web/src/pages/` 创建 Vue SFC
2. 在 `apps/web/src/router/registry.ts` 添加路由
3. 如需 API 调用，在 `apps/web/src/api/{domain}.ts` 补充对应方法
4. 如需状态管理，在 `apps/web/src/store/{domain}.ts` 扩展 Pinia store

### API 层规范

`api/*.ts` 文件是 `shared/api/generated/client.ts` 的薄封装，每个函数只做一件事：

```typescript
// ✅ 正确
export const getJobExecution = (id: number): Promise<JobExecution> =>
  client.getJobExecution({ path: { id } });

// ❌ 错误：不要在 api/ 层写业务逻辑或状态管理
```

### 构建验证

```bash
cd apps/web
bun run build   # 类型检查 + 生产构建，必须无错误
```

> 不要手动编辑 `apps/web/src/shared/api/generated/` 下的自动生成文件。

---

## 插件开发规范

完整教程请优先阅读 [PLUGIN_DEVELOPMENT.md](./PLUGIN_DEVELOPMENT.md)。

```
plugins/modules/your-plugin/
├── main.py                        # 入口
└── requirements.txt               # Python 依赖
```

推荐使用 `Module(...)` 描述插件，账号类型声明文件会由 SDK 自动生成。

**`main.py` 最小示例：**

```python
from octo import Module, success

module = Module(
    key="your_plugin",
    name="Your Plugin",
    category="generic",
    account_schema={
        "type": "object",
        "properties": {
            "api_key": {"type": "string", "title": "API Key"}
        },
        "required": ["api_key"]
    },
)

@module.action("DO_THING")
def do_thing(req):
    api_key = req.account_spec.get("api_key")
    if not api_key:
        raise ValueError("missing api_key")
    return success({"done": True})

if __name__ == "__main__":
    module.serve()
```

运行 `POST /api/v2/plugins/sync` 使新插件生效。

---

## 提交与 PR 规范

### Commit Message

遵循 [Conventional Commits](https://www.conventionalcommits.org/)：

```
<type>(<scope>): <subject>

[可选正文]

[可选 footer]
```

**type 枚举：**

| type | 场景 |
|------|------|
| `feat` | 新功能 |
| `fix` | Bug 修复 |
| `refactor` | 重构（不改变外部行为）|
| `docs` | 仅文档变更 |
| `test` | 测试相关 |
| `chore` | 构建脚本、依赖升级等 |

**示例：**

```
feat(jobs): add manual enqueue endpoint

POST /api/v2/job-definitions/:id/executions now accepts an optional
priority field to bump executions ahead in queue.
```

### PR Checklist

提交 PR 前请确认：

- [ ] `go test ./...` 全部通过
- [ ] `cd apps/web && bun run build` 无错误
- [ ] 若有 Schema 变更：已同步更新 AutoMigrate 模型与兼容逻辑
- [ ] 若有新 API 端点：已更新 `API_REFERENCE.md`
- [ ] 若有 UI 变更：PR 描述中包含截图
- [ ] Commit message 遵循 Conventional Commits

### PR 描述模板

```markdown
## 变更内容
<!-- 用 2-3 条列出主要变更 -->

## 动机
<!-- 为什么要做这个改动 -->

## 验证步骤
<!-- 列出手动验证步骤 -->

## 注意事项
<!-- Schema 变更、配置变更、破坏性变更等 -->
```

---

## 代码审查标准

审查者关注点（按优先级）：

1. **正确性** — 逻辑是否符合预期，边界条件是否处理
2. **安全性** — 无 SQL 注入、无凭证硬编码、输入校验充分
3. **层边界** — 业务逻辑不泄漏到 transport 层，domain 层无外部依赖
4. **迁移安全** — 新迁移是否幂等，是否破坏已有数据
5. **测试覆盖** — 新逻辑是否有对应测试

> 对于纯前端的视觉调整，重点检查 `bun run build` 是否通过以及响应式布局。
