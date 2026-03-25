# API Reference

> **Base URL**: `http://<host>:8080`
> **API 前缀**: `/api/v2/`
> **认证**: 管理员端点需携带 `X-Admin-Key: {key}` 或 `Authorization: Bearer {key}`。
> 标记 🔒 的端点需要认证；标记 🌐 的端点公开无需认证。

---

## 目录

- [系统 (System)](#系统-system)
- [账号类型 (Account Types)](#账号类型-account-types)
- [账号 (Accounts)](#账号-accounts)
- [任务定义与执行 (Jobs)](#任务定义与执行-jobs)
- [触发器 (Triggers)](#触发器-triggers)
- [Webhook（公开）](#webhook公开)
- [Agent](#agent)
- [插件 (Plugins)](#插件-plugins)
- [邮件账号 (Email Accounts)](#邮件账号-email-accounts)
- [错误格式](#错误格式)

---

## 系统 (System)

### `GET /healthz` 🌐

健康检查，返回 200 表示服务正常。

**响应 200**
```json
"ok"
```

---

### `GET /api/v2/system/status` 🌐

系统初始化状态。

**响应 200**
```json
{
  "initialized": true,
  "needs_setup": false
}
```

---

### `GET /api/v2/dashboard` 🌐

看板统计数据。

**响应 200** — 返回各域的计数汇总（具体字段以实际响应为准）。

---

### `GET /api/v2/config/{key}` 🌐

读取全局配置项。

**路径参数**

| 参数 | 类型 | 说明 |
|------|------|------|
| `key` | string | 配置键，如 `app.name` |

**已知配置键**

| 键 | 类型 | 说明 |
|----|------|------|
| `app.name` | string | 应用名称 |
| `job.default_timeout_minutes` | number | 默认任务超时（分钟）|
| `job.max_concurrency` | number | 最大并发执行数 |

**响应 200**
```json
{
  "key": "app.name",
  "value": "OctoManger"
}
```

**响应 200（键不存在）**
```json
{
  "key": "unknown.key",
  "value": null
}
```

---

### `PUT /api/v2/config/{key}` 🔒

更新全局配置项，值须为合法 JSON。

**路径参数**

| 参数 | 类型 | 说明 |
|------|------|------|
| `key` | string | 配置键 |

**请求体** — 任意合法 JSON 值
```json
"MyApp"
```
或
```json
30
```

**响应 200**
```json
{ "saved": true }
```

---

## 账号类型 (Account Types)

账号类型由插件声明，定义凭证的 JSON Schema 与能力集。

### `GET /api/v2/account-types` 🌐

列出所有账号类型。

**响应 200**
```json
{
  "items": [
    {
      "key": "github",
      "name": "GitHub",
      "category": "vcs",
      "schema_json": { "type": "object", "properties": { "token": { "type": "string" } } },
      "capabilities_json": { "actions": ["star_repo", "list_repos"] },
      "created_at": "2026-01-01T00:00:00Z",
      "updated_at": "2026-01-01T00:00:00Z"
    }
  ]
}
```

---

### `GET /api/v2/account-types/{key}` 🌐

获取单个账号类型。

**响应 200** — 同上单条记录。
**响应 404** — 账号类型不存在。

---

### `POST /api/v2/account-types` 🔒

创建账号类型（通常由插件自动同步，手动创建适用于调试）。

**请求体**
```json
{
  "key": "my-service",
  "name": "My Service",
  "category": "generic",
  "schema_json": {},
  "capabilities_json": {}
}
```

**响应 201** — 返回创建后的账号类型对象。

---

### `PATCH /api/v2/account-types/{key}` 🔒

部分更新账号类型。请求体为可选字段子集（同 POST）。

**响应 200** — 返回更新后的对象。

---

### `DELETE /api/v2/account-types/{key}` 🔒

删除账号类型。

**响应 204** — 无响应体。

---

## 账号 (Accounts)

账号为外部系统的凭证实例，关联到某个账号类型。

### `GET /api/v2/accounts` 🌐

列出所有账号。

**响应 200**
```json
{
  "items": [
    {
      "id": 1,
      "account_type_id": 1,
      "identifier": "user@example.com",
      "spec_json": { "token": "ghp_..." },
      "status": "active",
      "tags_json": ["prod"],
      "created_at": "...",
      "updated_at": "..."
    }
  ]
}
```

---

### `GET /api/v2/accounts/{id}` 🌐

获取单个账号。

---

### `POST /api/v2/accounts` 🔒

创建账号。

**请求体**
```json
{
  "account_type_id": 1,
  "identifier": "user@example.com",
  "spec_json": { "token": "ghp_..." },
  "status": "active",
  "tags_json": ["prod"]
}
```

**响应 201** — 返回创建后的账号对象。

---

### `PATCH /api/v2/accounts/{id}` 🔒

部分更新账号（字段同 POST）。

---

### `DELETE /api/v2/accounts/{id}` 🔒

删除账号。**响应 204**。

---

### `POST /api/v2/accounts/{id}/execute` 🔒

对账号同步执行一次插件 action（调试用途）。

**请求体**
```json
{
  "plugin_key": "github",
  "action": "list_repos",
  "input_json": { "per_page": 10 }
}
```

**响应 200**
```json
{
  "result_json": { "repos": ["octo/one", "octo/two"] }
}
```

---

## 任务定义与执行 (Jobs)

### `GET /api/v2/job-definitions` 🌐

列出所有任务定义。

**响应 200**
```json
{
  "items": [
    {
      "id": 1,
      "key": "daily-github-sync",
      "name": "Daily GitHub Sync",
      "plugin_key": "github",
      "action": "sync_stars",
      "input_json": {},
      "enabled": true,
      "schedule": {
        "cron_expression": "0 2 * * *",
        "timezone": "Asia/Shanghai",
        "next_run_at": "2026-03-22T02:00:00+08:00"
      }
    }
  ]
}
```

---

### `POST /api/v2/job-definitions` 🔒

创建任务定义。

**请求体**
```json
{
  "key": "daily-github-sync",
  "name": "Daily GitHub Sync",
  "plugin_key": "github",
  "action": "sync_stars",
  "input_json": { "account_id": 1 },
  "enabled": true,
  "schedule": {
    "cron_expression": "0 2 * * *",
    "timezone": "Asia/Shanghai"
  }
}
```

**响应 201** — 返回创建后的任务定义对象。

---

### `PATCH /api/v2/job-definitions/{id}` 🔒

部分更新任务定义。

---

### `POST /api/v2/job-definitions/{id}/executions` 🔒

手动入队一次执行（立即触发）。

**响应 201**
```json
{
  "id": 42,
  "status": "pending",
  "source": "manual",
  "created_at": "..."
}
```

---

### `GET /api/v2/job-executions` 🌐

列出执行记录（最新在前）。

**响应 200**
```json
{
  "items": [
    {
      "id": 42,
      "job_definition_id": 1,
      "status": "completed",
      "source": "schedule",
      "worker_id": "worker-01",
      "summary": "Synced 12 repos",
      "result_json": {},
      "error_message": null,
      "started_at": "...",
      "finished_at": "...",
      "created_at": "..."
    }
  ]
}
```

**执行状态值**

| 状态 | 说明 |
|------|------|
| `pending` | 等待 Worker 拾取 |
| `running` | 正在执行 |
| `completed` | 成功完成 |
| `failed` | 执行失败 |

---

### `GET /api/v2/job-executions/{id}` 🌐

获取单条执行记录详情。

---

### `GET /api/v2/job-executions/{id}/events` 🌐 (SSE)

以 Server-Sent Events 流式获取执行日志。

**响应** — `text/event-stream`

```
data: {"stream":"stdout","event_type":"log","message":"Fetching repos...","payload_json":{}}

data: {"stream":"stdout","event_type":"progress","message":"12/12 done","payload_json":{"count":12}}

data: {"event_type":"done"}
```

---

## 触发器 (Triggers)

触发器将 Webhook 请求路由到任务定义。

### `GET /api/v2/triggers` 🌐

列出所有触发器。

**响应 200**
```json
{
  "items": [
    {
      "id": 1,
      "key": "ci-notify",
      "name": "CI Notification",
      "job_definition_id": 1,
      "mode": "async",
      "default_input_json": {},
      "token_prefix": "oct_",
      "enabled": true
    }
  ]
}
```

> 注意：`token_hash` 不会在列表中返回，Token 明文仅在创建时返回一次。

---

### `POST /api/v2/triggers` 🔒

创建触发器。

**请求体**
```json
{
  "key": "ci-notify",
  "name": "CI Notification",
  "job_definition_id": 1,
  "mode": "async",
  "default_input_json": {},
  "enabled": true
}
```

**响应 201**
```json
{
  "id": 1,
  "token": "oct_xxxxxxxxxxxxxxxxxxxxxxxx"
}
```

> ⚠️ `token` 只在创建响应中出现一次，之后不可再查询，请妥善保管。

---

### `PATCH /api/v2/triggers/{id}` 🔒

部分更新触发器。

---

### `DELETE /api/v2/triggers/{id}` 🔒

删除触发器。**响应 204**。

---

### `POST /api/v2/triggers/{id}/fire` 🔒

通过内部 ID 直接触发（无需 Token，用于测试）。

**请求体**（可选）
```json
{ "extra_input": "value" }
```

---

## Webhook（公开）

### `POST /api/v2/webhooks/{key}` 🌐

通过触发器 key 调用 Webhook。

**路径参数**

| 参数 | 类型 | 说明 |
|------|------|------|
| `key` | string | 触发器的 key |

**请求头**（二选一）

| 请求头 | 格式 |
|--------|------|
| `X-Trigger-Token` | `{token}` |
| `Authorization` | `Bearer {token}` |

**请求体**（可选）— 任意 JSON，与 `default_input_json` 合并后传入任务。

**响应 — async 模式（202）**
```json
{
  "execution_id": 42,
  "status": "pending"
}
```

**响应 — sync 模式（200）**
```json
{
  "result_json": { ... }
}
```

**响应 401** — Token 无效或缺失。
**响应 404** — 触发器不存在或已禁用。

---

## Agent

### `GET /api/v2/agents` 🌐

列出所有 Agent。

**响应 200**
```json
{
  "items": [
    {
      "id": 1,
      "name": "GitHub Watcher",
      "plugin_key": "github",
      "action": "watch_notifications",
      "input_json": {},
      "desired_state": "running",
      "runtime_state": "running",
      "last_error": null,
      "last_heartbeat_at": "2026-03-21T10:00:00Z"
    }
  ]
}
```

**运行时状态值**

| 状态 | 说明 |
|------|------|
| `idle` | 未启动 |
| `starting` | 正在启动中 |
| `running` | 正常运行，心跳活跃 |
| `stopping` | 正在停止 |
| `error` | 异常退出 |

---

### `POST /api/v2/agents` 🔒

创建 Agent。

**请求体**
```json
{
  "name": "GitHub Watcher",
  "plugin_key": "github",
  "action": "watch_notifications",
  "input_json": { "account_id": 1 }
}
```

---

### `PATCH /api/v2/agents/{id}` 🔒

更新 Agent 配置（不影响运行状态）。

---

### `GET /api/v2/agents/{id}/status` 🌐

获取 Agent 实时运行状态。

---

### `POST /api/v2/agents/{id}/start` 🔒

启动 Agent（将 `desired_state` 设为 `running`）。**响应 204**。

---

### `POST /api/v2/agents/{id}/stop` 🔒

停止 Agent（将 `desired_state` 设为 `stopped`）。**响应 204**。

---

### `GET /api/v2/agents/{id}/events` 🌐 (SSE)

以 Server-Sent Events 流式获取 Agent 事件日志，格式同 Job 执行日志。

---

## 插件 (Plugins)

### `GET /api/v2/plugins` 🌐

列出所有已加载的插件。

**响应 200**
```json
{
  "items": [
    {
      "key": "github",
      "name": "GitHub Plugin",
      "version": "1.0.0",
      "account_type_keys": ["github"],
      "actions": ["sync_stars", "list_repos"]
    }
  ]
}
```

---

### `GET /api/v2/plugins/{key}` 🌐

获取插件详情（包括 actions 列表与参数 Schema）。

---

### `POST /api/v2/plugins/sync` 🔒

重新扫描 `PLUGINS_DIR`，将插件声明的账号类型同步到数据库。

**响应 200**
```json
{
  "synced": ["github", "email-imap"],
  "errors": []
}
```

---

### `GET /api/v2/plugins/{key}/settings` 🔒

获取插件当前设置（存储于 `system_configs`）。

**响应 200** — 返回键值对象。

---

### `PUT /api/v2/plugins/{key}/settings` 🔒

全量更新插件设置。

**请求体** — 任意 JSON 对象
```json
{ "api_timeout": 30, "max_retries": 3 }
```

**响应 200**
```json
{ "saved": true }
```

---

## 邮件账号 (Email Accounts)

> 所有邮件端点均需 🔒 管理员认证。

### `GET /api/v2/email/accounts`

列出邮件账号。

**响应 200**
```json
{
  "items": [
    {
      "id": 1,
      "provider": "imap",
      "address": "user@example.com",
      "status": "active",
      "config_json": {}
    }
  ]
}
```

---

### `POST /api/v2/email/accounts/bulk-import`

批量导入邮件账号（CSV 格式行）。

**请求体**
```json
{
  "lines": ["imap|user1@example.com|password1", "imap|user2@example.com|password2"]
}
```

**响应 200**
```json
{
  "imported": 2,
  "errors": []
}
```

---

### `POST /api/v2/email/accounts`

创建邮件账号。

**请求体**
```json
{
  "provider": "imap",
  "address": "user@example.com",
  "config_json": {
    "host": "imap.example.com",
    "port": 993,
    "password": "secret"
  }
}
```

---

### `PATCH /api/v2/email/accounts/{id}`

部分更新邮件账号配置。

---

### `DELETE /api/v2/email/accounts/{id}`

删除邮件账号。**响应 204**。

---

### `POST /api/v2/email/accounts/{id}/outlook/authorize-url`

生成 Outlook OAuth 授权 URL。

**响应 200**
```json
{ "url": "https://login.microsoftonline.com/..." }
```

---

### `POST /api/v2/email/accounts/{id}/outlook/exchange-code`

用 OAuth 回调 code 换取访问令牌。

**请求体**
```json
{ "code": "M.xxxxx", "redirect_uri": "http://localhost:8080/oauth/callback" }
```

**响应 200** — 返回更新后的邮件账号。

---

### `GET /api/v2/email/accounts/{id}/mailboxes`

列出邮箱文件夹。

**查询参数**

| 参数 | 类型 | 说明 |
|------|------|------|
| `pattern` | string | 可选，文件夹名通配模式 |

**响应 200**
```json
{ "mailboxes": ["INBOX", "Sent", "Trash"] }
```

---

### `GET /api/v2/email/accounts/{id}/messages`

分页列出邮件。

**查询参数**

| 参数 | 类型 | 默认 | 说明 |
|------|------|------|------|
| `mailbox` | string | `INBOX` | 文件夹名 |
| `limit` | number | 20 | 每页条数 |
| `offset` | number | 0 | 偏移量 |

---

### `GET /api/v2/email/accounts/{id}/messages/latest`

获取最新一封邮件。

**查询参数**

| 参数 | 类型 | 说明 |
|------|------|------|
| `mailbox` | string | 可选，指定文件夹 |

---

### `GET /api/v2/email/accounts/{id}/messages/{message_id}`

获取指定邮件详情（含正文）。

---

### `POST /api/v2/email/preview/mailboxes`

预览邮箱文件夹，无需先创建账号记录。

**请求体**
```json
{
  "provider": "imap",
  "config_json": { "host": "imap.example.com", "port": 993, "address": "x@y.com", "password": "..." }
}
```

---

### `POST /api/v2/email/preview/messages/latest`

预览最新邮件，无需先创建账号记录（同上请求体格式）。

---

## 错误格式

所有错误响应遵循 [RFC 7807 Problem Details](https://tools.ietf.org/html/rfc7807) 格式：

```json
{
  "type": "https://docs.octomanger.dev/problems/404",
  "title": "not_found",
  "status": 404,
  "detail": "account type 'unknown-key' not found"
}
```

| HTTP 状态 | title | 说明 |
|-----------|-------|------|
| 400 | `bad_request` | 请求参数校验失败 |
| 401 | `unauthorized` | 缺少或无效的管理员密钥 |
| 404 | `not_found` | 资源不存在 |
| 409 | `conflict` | 唯一键冲突 |
| 500 | `internal_server_error` | 服务端错误 |
