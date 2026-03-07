# Trigger 触发器

Trigger 把外部 Webhook 请求接入 OctoModule 动作，支持同步和异步两种模式。

```text
外部系统 → POST /webhooks/:slug → Trigger → Job → Python 模块 → 结果
```

---

## 适用场景

- 外部事件（用户注册、订单创建）触发批量账号操作
- 合作方通过 Webhook 调用平台能力
- 定时任务触发器（配合 cron 服务定期推送请求）

**注意**：Trigger 只支持 `generic` 类型的账号，不支持 `email` 类型。

---

## 同步 vs 异步模式

| 模式 | 行为 | 适合场景 |
| --- | --- | --- |
| `async` | 入 Asynq 队列，立即返回 `{queued: true}`，后台执行 | 不关心实时结果，批量处理 |
| `sync` | 在 API 进程直接执行，等待完成后返回结果 | 需要实时结果，账号数量少 |

---

## 创建 Trigger

打开控制台 **Triggers**，点击"新建 Trigger"：

| 字段 | 说明 |
| --- | --- |
| `name` | 显示名称 |
| `slug` | URL 中的唯一标识，创建后不可修改。如 `partner-a-register` |
| `type_key` | 目标账号类型（`generic` 类型） |
| `action_key` | 要执行的动作，如 `REGISTER` |
| `mode` | `async` 或 `sync` |
| `default_selector` | 默认账号选择范围（JSON），可被请求覆盖 |
| `default_params` | 默认参数（JSON），可被请求中的 `extra_params` 覆盖 |

创建成功后，对话框会显示：
- **Webhook 地址**：`/webhooks/<slug>`
- **Trigger Token**：只显示一次，请立即保存

---

## 触发 Webhook

### 鉴权方式（二选一）

**方式一：Webhook Key（推荐）**

创建一个角色为 `webhook` 的 API Key，在请求头中携带：

```bash
curl -X POST http://localhost:8080/webhooks/partner-a-register \
  -H "X-Api-Key: octo_webhookkey..."
```

**方式二：Trigger Token（Bearer）**

使用创建 Trigger 时生成的专属 token：

```bash
curl -X POST http://localhost:8080/webhooks/partner-a-register \
  -H "Authorization: Bearer <trigger_token>" \
  -H "Content-Type: application/json"
```

### 请求体

```json
{
  "mode": "sync",
  "selector": {
    "account_ids": [1, 2]
  },
  "extra_params": {
    "invite_code": "ABC123"
  }
}
```

| 字段 | 说明 |
| --- | --- |
| `mode` | `async` 或 `sync`，覆盖 Trigger 的默认模式（可选） |
| `selector` | 覆盖 Trigger 的 `default_selector`（可选） |
| `extra_params` | 与 `default_params` 合并，同名字段以 `extra_params` 优先（可选） |

**params 合并逻辑**：`final_params = default_params + extra_params`（extra 优先），系统还会注入 `_trigger` 元数据字段（包含 trigger_id、slug 等）。

---

## 响应格式

### async 模式

```json
{
  "code": 0,
  "data": {
    "queued": true,
    "job": {
      "id": 42,
      "status": "queued"
    }
  }
}
```

用 `job.id` 轮询 `GET /api/v1/jobs/{id}` 获取最终结果。

### sync 模式

```json
{
  "code": 0,
  "data": {
    "queued": false,
    "job": { "id": 43, "status": "done" },
    "output": {
      "results": [
        {
          "account_id": 7,
          "identifier": "demo-user-01",
          "status": "success",
          "result": { "event": "registered" }
        }
      ]
    }
  }
}
```

---

## Webhook Key 与 Trigger Token 的区别

| | Webhook Key | Trigger Token |
| --- | --- | --- |
| 创建位置 | API Keys 页面 | 创建 Trigger 时自动生成 |
| 请求头 | `X-Api-Key` | `Authorization: Bearer` |
| 范围控制 | 可限制到单个 slug，也可允许所有 | 只能用于创建它的那个 Trigger |
| 可管理 | 可以禁用、删除 | 无法查看原文，只能删除 Trigger |

---

## Trigger 管理

| 操作 | 说明 |
| --- | --- |
| **启用 / 禁用** | 禁用后，对该 Trigger 的请求返回 403 |
| **编辑** | 可修改 name、mode、default_selector、default_params、启用状态；slug 不可改 |
| **删除** | 删除后该 slug 不再可用，Trigger Token 同时失效 |

---

## 完整示例

### 场景：合作方 A 每次有新用户注册，通过 Webhook 触发批量注册

**1. 创建 Trigger**

```json
{
  "name": "合作方A注册",
  "slug": "partner-a-register",
  "type_key": "demo_shop",
  "action_key": "REGISTER",
  "mode": "async",
  "default_selector": { "identifier_contains": "partner-a", "limit": 50 },
  "default_params": { "source": "partner-a" }
}
```

**2. 保存 Trigger Token**（创建响应里的 `raw_token`）

**3. 外部系统触发**

```bash
curl -X POST http://localhost:8080/webhooks/partner-a-register \
  -H "Authorization: Bearer <trigger_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "extra_params": { "invite_code": "PARTNER2024" }
  }'
```

**4. 查询任务结果**

```bash
curl -H "X-Api-Key: <admin_key>" http://localhost:8080/api/v1/jobs/42
```
