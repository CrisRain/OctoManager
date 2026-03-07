# 鉴权与 API Key

---

## 基本规则

系统有两种 API Key 角色：

| 角色 | 用途 | 可访问的路由 |
| --- | --- | --- |
| `admin` | 管理操作（控制台使用） | `/api/v1/*` 全部接口 |
| `webhook` | 外部系统触发 Webhook | `/webhooks/:slug` |

**引导模式**：数据库里还没有任何 Admin Key 时，`/api/v1/*` 接口全部放行，不需要鉴权。这是为了让初始化向导能顺利完成。一旦创建了第一个 Admin Key，引导模式立即关闭。

---

## 初始化流程

首次访问控制台会自动跳转到 `/setup`。

### 步骤 1 — 运行数据库迁移

系统尚未初始化时（数据表不存在），需要先执行迁移。

点击"运行迁移"，系统调用 `POST /api/v1/system/migrate`，执行 GORM AutoMigrate 并写入默认系统配置（应用名称、任务超时、最大并发数等）。迁移是幂等操作，对已有数据库安全，可以重复执行。

### 步骤 2 — 创建第一个 Admin Key

调用 `POST /api/v1/system/setup`（此接口不需要鉴权），创建第一个角色为 `admin` 的 API Key。

响应包含完整的原始密钥（`raw_key`），**只返回这一次**，之后无法再查看。密钥前 8 位会作为 `key_prefix` 存储，用于在列表中辨认密钥身份。

### 步骤 3 — 保存密钥

请立即复制并妥善保存。控制台会把密钥自动写入浏览器 localStorage，下次访问时无需重新输入。

---

## Admin Key 使用方式

### 控制台

控制台自动从 localStorage 读取并在每个请求中附加 `X-Api-Key` 请求头，无需手动操作。

登出：点击"设置 → 退出登录"，会清除 localStorage 中保存的密钥并跳转到登录页。

### curl / 程序调用

```bash
curl -H "X-Api-Key: octo_xxxxxxxxxxxxxxxx" \
  http://localhost:8080/api/v1/account-types/
```

---

## 管理 API Key

打开控制台 **API Keys** 页面，可以：

- **创建**：指定名称、角色（admin / webhook）和 Webhook 范围
- **启用 / 禁用**：通过开关控制密钥是否有效，禁用后立即生效
- **删除**：删除后立即失效，不可恢复

> 创建密钥只有一次机会看到原始密钥，请在创建对话框关闭前复制。

---

## Webhook Key（角色 = webhook）

Webhook Key 专门用于外部系统触发 Trigger，不能访问其他管理接口。

创建时可以设置 **Webhook 范围**：
- `*`（默认）：允许触发所有 Trigger
- 具体的 slug（如 `partner-a-register`）：只允许触发该 Trigger

### 使用方式

外部系统在请求头中携带 Webhook Key：

```bash
curl -X POST http://localhost:8080/webhooks/partner-a-register \
  -H "X-Api-Key: octo_webhookkey..."
```

也可以继续使用旧版 Bearer Token 方式（Trigger 创建时会生成一个专属 token）：

```bash
curl -X POST http://localhost:8080/webhooks/partner-a-register \
  -H "Authorization: Bearer <trigger_token>"
```

---

## 登录 / 重新登录

如果密钥失效（被删除或禁用），控制台会在下次页面访问时自动检测并跳转到 `/auth` 页面，要求重新输入有效的 Admin Key。

验证逻辑：输入密钥后，控制台调用 `GET /api/v1/api-keys/` 进行服务端验证。返回 `code: 401` 表示密钥无效；成功则跳转回原目标页。

---

## 不需要鉴权的接口

以下接口无论是否已初始化都可以直接访问：

| 接口 | 说明 |
| --- | --- |
| `GET /api/v1/system/status` | 查询系统是否已初始化 |
| `POST /api/v1/system/setup` | 创建第一个 Admin Key（已有 Admin Key 时返回错误） |
| `GET /health` | 健康检查 |

`POST /api/v1/system/migrate` 迁移接口需要鉴权（引导模式下放行）。

---

## 密钥存储安全说明

- 密钥原文只在创建时展示一次，之后只存哈希值（bcrypt）
- `key_prefix` 存储密钥前 8 位明文，用于在列表中辨认密钥
- 密钥格式：`octo_` 前缀 + 随机字符串
- 控制台将密钥存在浏览器 localStorage，请勿在公共设备上长期保存
