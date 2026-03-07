# OctoModule 开发指南

目标：让你尽快写出一个可运行的模块，并通过 Job 或 Trigger 跑起来。

想先跑通再看细节？直接跳到 [第三节：10 分钟跑通一个模块](#三10-分钟跑通一个模块)。

---

## 一、核心概念

| 概念 | 说明 |
| --- | --- |
| **OctoModule** | 一段 Python 脚本，负责执行账号的某个动作（REGISTER、VERIFY、LOCK…） |
| **Account Type** | 类型定义，`category=generic` 的类型才走 OctoModule 链路 |
| **Account** | 具体账号实例，执行时把 `identifier` 和 `spec` 传给模块 |
| **Job** | 一次异步任务，Worker 分发给对应模块批量执行 |
| **Dry Run** | 在控制台直接测试模块，不经过 Job 队列，不写入真实数据库 |

只有 `category=generic` 的 Account Type 才走 OctoModule，`email` 类型不走。

---

## 二、文件结构

每个模块的默认入口：

```text
scripts/python/modules/<type_key>/main.py
```

创建 `generic` 类型后，系统自动在该目录生成脚手架。建议目录结构：

```text
scripts/python/modules/demo_shop/
├── main.py            ← 默认入口（自动生成）
├── client.py          ← 可拆出的辅助模块
├── requirements.txt   ← 依赖声明
└── .venv/             ← 独立虚拟环境，Worker 优先使用
```

不同模块的依赖完全隔离。Worker 执行时工作目录切到脚本所在目录，`import` 相对路径以此为基准。

### 自定义入口

在 Account Type 的 `script_config` 里指定 `entry` 可以覆盖默认路径：

```json
{ "entry": "demo_shop/app/entry.py" }
```

路径相对 `OCTO_MODULE_DIR`，不能包含 `../` 和绝对路径。

---

## 三、10 分钟跑通一个模块

### 步骤 1：创建 generic 类型

打开控制台 **Account Types**，新建：

- `key`: `demo_shop`，`category`: `generic`

`schema`（账号字段定义）：

```json
{
  "type": "object",
  "properties": {
    "username": { "type": "string", "title": "用户名" },
    "password": { "type": "string", "title": "密码" }
  },
  "required": ["username", "password"]
}
```

`capabilities`（支持的动作）：

```json
{
  "actions": [
    { "key": "REGISTER" },
    { "key": "VERIFY" }
  ]
}
```

保存后，系统自动生成 `scripts/python/modules/demo_shop/main.py` 脚手架。

### 步骤 2：创建一个测试账号

打开 **Accounts**，类型选 `demo_shop`：

- `identifier`: `demo-user-01`，`username`: `alice`，`password`: `pass-123`

### 步骤 3：填写业务逻辑

打开 **Octo Modules → demo_shop**，脚手架已经预置了 dispatch 结构，**只需要填写 handler 函数体**：

```python
def handle_register(identifier: str, spec: dict, params: dict) -> dict:
    invite_code = str(params.get("invite_code", "")).strip()
    if not invite_code:
        return error("VALIDATION_FAILED", "invite_code is required")
    return success({
        "event": "registered",
        "identifier": identifier,
        "username": spec.get("username", ""),
        "invite_code": invite_code,
        "handled_at": now_utc(),
    })

def handle_verify(identifier: str, spec: dict, params: dict) -> dict:
    return success({"event": "verified", "identifier": identifier, "handled_at": now_utc()})
```

**添加新动作只需两步：**
1. 写 `handle_xxx(identifier, spec, params)` 函数
2. 在 `ACTIONS` dict 里加一行 `"XXX": handle_xxx`

不需要修改 `main()` 函数，dispatch 逻辑已经内置。

### 步骤 4：Dry Run 验证

在模块管理页点 **Dry Run**：

```
action:     REGISTER
identifier: demo-user-01
spec:       {"username": "alice", "password": "pass-123"}
params:     {"invite_code": "ABC123"}
```

Dry Run 不打真实请求，专门用于调试。确认输出符合预期再继续。

### 步骤 5：创建 Job

打开 **Jobs**，新建：

- `type_key`: `demo_shop`，`action_key`: `REGISTER`
- 勾选账号 `demo-user-01`
- `params`: `{"invite_code": "ABC123"}`

结果在 Jobs 详情和 **Octo Modules → demo_shop → 运行历史** 中查看。

---

## 四、模块输入/输出规范

### 模块接收的输入

Worker 通过 stdin 向模块传入 JSON：

```json
{
  "action": "REGISTER",
  "account": {
    "identifier": "demo-user-01",
    "spec": { "username": "alice", "password": "pass-123" }
  },
  "params": { "invite_code": "ABC123" },
  "context": { "request_id": "42:7" }
}
```

| 字段 | 说明 |
| --- | --- |
| `action` | 当前要执行的动作（全大写） |
| `account.identifier` | 账号的业务唯一标识 |
| `account.spec` | 账号详情，来自 Account 的 `spec` 字段 |
| `params` | 本次任务的额外参数，来自 Job 的 `params` 字段 |
| `context.request_id` | 格式为 `job_id:account_id` 的追踪 ID |

### 模块必须返回什么

Python 脚本向 stdout 打印**恰好一行** JSON，然后退出（exit code 0）。

**成功：**

```json
{ "status": "success", "result": { "event": "registered", "handled_at": "..." } }
```

**失败：**

```json
{ "status": "error", "error_code": "VALIDATION_FAILED", "error_message": "invite_code is required" }
```

**带 Session（可选）：**

```json
{
  "status": "success",
  "result": { "event": "login_ok" },
  "session": {
    "type": "token",
    "payload": { "token": "abc123" },
    "expires_at": "2026-12-31T00:00:00Z"
  }
}
```

返回的 `session` 会存储到 `account_sessions` 表，但目前不会自动注入到下一次调用的输入中。

---

## 五、依赖管理

### 安装依赖

在控制台 **Octo Modules → demo_shop → 依赖管理** 中：

- 从 `requirements.txt` 批量安装
- 或者手动输入包名单独安装

系统在模块目录创建 `.venv`，不同模块的依赖完全隔离。

Worker 执行模块时，优先使用 `.venv/bin/python`（Linux/Mac）或 `.venv/Scripts/python.exe`（Windows），找不到时退回到系统 Python。

### 相关 API

```
GET  /api/v1/octo-modules/{typeKey}/venv          查看依赖状态
POST /api/v1/octo-modules/{typeKey}/venv/install   安装依赖
```

---

## 六、Daemon 模式（后台持续运行）

普通模块是"一次请求、一次执行、退出"。**Daemon 模式**让模块以持久子进程的形式后台运行，适合需要保持连接、持续监听事件的场景（如消息推送、账号状态监控、WebSocket 监听）。

### 6.1 协议差异

| | 普通模块 | Daemon 模块 |
|---|---|---|
| 生命周期 | 一次执行后退出 | 永久运行直到被终止 |
| stdout | 一行 JSON 输出 | NDJSON（逐行，`flush=True`） |
| 触发方式 | Job / Trigger | 启动时自动运行 |
| 运行实例 | 每次 Job 一个 | 每个活跃账号一个长期进程 |

Daemon 模块的事件输出格式：

| `status` 值 | 含义 | 数据字段 |
|---|---|---|
| `init_ok` | 初始化完成 | — |
| `event` | 收到事件 | `result: {...}` |
| `error` | 模块报错 | `error_code`, `error_message` |
| `done` | 主动退出（Go 侧会重启） | — |

### 6.2 开启 Daemon 模式

在 Account Type 的 `capabilities` 中加入 `daemon` 字段：

```json
{
  "actions": [{"key": "REGISTER"}, {"key": "VERIFY"}],
  "daemon": {
    "action": "WATCH"
  }
}
```

`action` 是传给模块的 `input.action` 值，默认为 `"WATCH"`。

### 6.3 编写 Daemon 模块

Daemon 模块复用同一个 `main.py` 入口，但针对 `WATCH` 动作实现持续循环：

```python
#!/usr/bin/env python3
import json, sys, time

def handle_register(identifier, spec, params):
    return {"status": "success", "result": {"event": "registered"}}

def handle_watch(identifier, spec, params):
    """Daemon 动作：初始化后进入事件循环，永远不 return"""
    # 第一阶段：初始化（登录、建立连接等）
    token = login(spec["username"], spec["password"])
    print(json.dumps({"status": "init_ok"}), flush=True)

    # 第二阶段：事件循环
    while True:
        messages = poll_messages(token)   # 阻塞或轮询
        for msg in messages:
            print(json.dumps({
                "status": "event",
                "result": {
                    "type": "message",
                    "from": msg["sender"],
                    "text": msg["body"],
                    "identifier": identifier,
                }
            }), flush=True)
        time.sleep(5)

ACTIONS = {
    "REGISTER": handle_register,
    "WATCH":    handle_watch,    # ← daemon 动作
}

def main():
    request = json.loads(sys.stdin.read())
    action = request.get("action", "").upper()
    account = request.get("account", {})
    identifier = account.get("identifier", "")
    spec = account.get("spec", {})
    params = request.get("params", {})

    handler = ACTIONS.get(action)
    if handler is None:
        print(json.dumps({"status": "error", "error_code": "UNSUPPORTED_ACTION",
                          "error_message": f"unsupported: {action}"}))
        return

    result = handler(identifier, spec, params)
    # 普通动作：打印单行 JSON 然后退出
    # Daemon 动作：handler 内部已经持续输出，永不 return
    if result is not None:
        print(json.dumps(result), flush=True)

if __name__ == "__main__":
    main()
```

> **调试提示**：调试信息写 `stderr`，不要写 `stdout`：
> ```python
> print("connecting...", file=sys.stderr)
> ```

### 6.4 启动 Daemon Manager

```bash
cd backend

# 随统一入口启动（推荐）
go run ./cmd/octomanger

# 或单独启动
go run ./cmd/daemon
```

Daemon Manager 启动时：
1. 扫描所有 `capabilities.daemon` 不为空且 `category=generic` 的账号类型
2. 查找每个类型下所有 `status=1`（启用）的账号
3. 为每个账号启动一个持久 Python 子进程
4. 收到 `event` 行时，写入 `job_runs` 表（可在 Jobs 运行历史中查看）
5. 进程崩溃后自动以指数退避（5s → 10s → … → 5min）重启

---

## 七、常见坑

**stdout 只能打一行 JSON**

```python
print("debug info", file=sys.stderr)   # 正确：调试信息走 stderr
print("starting...")                     # 错误：破坏 JSON 输出
```

**动作名大小写**

模块中的 `ACTIONS` dict 的 key（如 `"REGISTER"`）必须和 Account Type 的 `capabilities.actions[].key` 完全一致，包括大小写。

**先 Dry Run，再 Job**

Dry Run 里报的错更容易定位，参数问题在这一步修掉，不要直接上 Job。

**工作目录**

Worker 执行脚本时，工作目录是脚本所在目录（即 `<type_key>/`），`open("data.json")` 这类相对路径以此为基准。

---

## 八、排查问题

**模块没有触发**
1. `Account Type.category` 是不是 `generic`？
2. `capabilities.actions` 里有没有你要的动作 key？
3. 模块入口文件是否存在？

**模块返回错误**
1. Dry Run 能通吗？先用 Dry Run 隔离问题
2. Python 有没有往 stdout 打了多余内容？
3. 依赖是不是装到了模块目录的 `.venv` 里？

**查看运行历史**

```
GET /api/v1/octo-modules/{typeKey}/runs?limit=20&offset=0
```

控制台 **Octo Modules → 运行历史** 标签也可以查看。

**Job 选错了账号**

检查 Job 的 `selector` 配置，通过 `identifiers` 精确锁定到单个账号再调试。

---

## 九、生产建议

- **动作拆小**：`LOGIN / REGISTER / VERIFY / LOCK`，不要搞 `DO_EVERYTHING`
- **让动作幂等**：Job 可能因网络问题重试，同一请求跑两次不能把状态搞乱
- **result 里放排查字段**：`event`、`identifier`、`request_id`、`handled_at`、`upstream_status`
- **开发顺序**：写死返回成功 → 读 `spec` → 读 `params` → 加校验 → 接真实 API，每步 Dry Run 验证
- **Trigger 场景**：`params` 里带 `source`、`event_id`，方便关联上游事件

---

## 十、Account Type 配置完整参考

### schema（更多字段类型）

```json
{
  "type": "object",
  "properties": {
    "username":  { "type": "string", "title": "用户名" },
    "password":  { "type": "string", "title": "密码" },
    "region":    { "type": "string", "title": "区域", "enum": ["us", "eu", "ap"] },
    "age":       { "type": "integer", "title": "年龄", "minimum": 0 },
    "active":    { "type": "boolean", "title": "是否激活" },
    "proxy_url": { "type": "string", "title": "代理地址" }
  },
  "required": ["username", "password", "region"]
}
```

### capabilities

```json
{
  "actions": [
    { "key": "REGISTER" },
    { "key": "VERIFY" },
    { "key": "LOCK" },
    { "key": "UNLOCK" }
  ]
}
```

如需开启 Daemon 模式，加入 `daemon` 字段：

```json
{
  "actions": [
    { "key": "REGISTER" },
    { "key": "VERIFY" }
  ],
  "daemon": {
    "action": "WATCH"
  }
}
```

### script_config（仅需自定义入口时填写）

```json
{ "entry": "demo_shop/app/entry.py" }
```
