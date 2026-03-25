# octo\_demo — OctoManger 插件完整示例

本目录是一个功能完整的 OctoManger 插件演示，覆盖所有 SDK 特性。
配套了一个零依赖的假数据服务器 (`fake_server.py`)，无需真实外部服务即可运行。

***

## 目录结构

```
octo_demo/
├── main.py              # 插件主文件（所有 SDK 特性示例）
├── fake_server.py       # 模拟外部 API 的假数据服务器
├── requirements.txt     # 插件 gRPC 运行所需依赖
└── README.md            # 本文档
```

`account_type.octo_demo.json` 由插件在首次运行时**自动生成**，无需手动创建。

***

## 快速开始

### 第一步：启动假数据服务器

```
cd plugins/modules/octo_demo
python3 fake_server.py
# 输出：
# [fake-server] 启动于 http://127.0.0.1:18080
# [fake-server] 测试账号 → username=testuser  api_key=demo_testkey_12345678
```

自定义端口：

```bash
python3 fake_server.py --port 9000 --host 0.0.0.0
```

### 第二步：配置 OctoManger

在 `.env` 或 Docker Compose 中设置插件目录；`octo_demo` 的 gRPC 地址会在系统首次启动时自动初始化到数据库配置：

```env
PLUGINS_DIR=/path/to/plugins/modules
PLUGIN_SDK_DIR=/path/to/plugins/sdk/python
PYTHON_BIN=python3
```

`worker` 启动时会在入口 goroutine 中拉起 `plugins/modules/octo_demo/main.py`，
自动创建/复用 `plugins/modules/octo_demo/.venv`，执行 `pip install -r requirements.txt`，
并通过数据库中的 `plugins.grpc_services` 配置与其通信。默认会初始化为：

```json
{
  "octo_demo": {
    "address": "127.0.0.1:50051"
  }
}
```

如需覆盖首次初始化值，仍可在首启前设置 `PLUGIN_GRPC_OCTO_DEMO_ADDR`。
账号类型会在插件健康后自动同步。

### 第三步：在 OctoManger 中创建账号

进入「账号」页面 → 新建账号 → 选择 **OctoDemo 演示** 类型，填写：

| 字段      | 示例值                      |
| ------- | ------------------------ |
| 用户名     | `testuser`               |
| API Key | `demo_testkey_12345678`  |
| 服务地址    | `http://127.0.0.1:18080` |

或使用「注册」Tab 自动注册新账号（无需手动填写 API Key）。

***

## 功能说明

### Actions 一览

| Action          | 模式    | 说明                           |
| --------------- | ----- | ---------------------------- |
| `VERIFY`        | sync  | 验证账号凭证是否有效                   |
| `GET_PROFILE`   | sync  | 获取用户资料，自动回写账号 spec           |
| `LIST_TASKS`    | sync  | 分页查询任务（支持状态/优先级过滤）           |
| `CREATE_TASK`   | job   | 创建任务（后台 Job 模式，写入 job\_logs） |
| `COMPLETE_TASK` | sync  | 将任务标记为已完成                    |
| `DELETE_TASK`   | sync  | 删除任务                         |
| `REGISTER`      | sync  | 自动注册新账号（仅创建账号时）              |
| `AGENT_MONITOR` | agent | 定期采集统计数据，发现 critical 任务则告警   |

### 执行模式

- **sync** — 请求后立即执行，前端阻塞等待结果。
- **job** — 提交为后台 Job，可在「Jobs」页查看日志和进度。
- **agent** — 由 Worker 的 AgentSupervisor 循环调用，每轮执行后休眠 `WORKER_AGENT_LOOP_INTERVAL` 秒。

***

## SDK 特性覆盖

### Module 定义

```python
module = octo.Module(
    key="octo_demo",
    name="OctoDemo 演示",
    category="generic",
    account_schema={...},   # 账号凭证 JSON Schema
    settings=[...],         # 插件级全局设置
)
```

### Setting — 插件级设置

```python
octo.Setting(
    key="debug_mode",
    label="调试模式",
    type="string",
    default="false",
    secret=False,           # True = 在 UI 中以密码框显示，值不返回前端
)
```

### ParamSpec — Action 参数声明

```python
octo.ParamSpec(
    name="priority",
    type="string",
    required=False,
    default="medium",
    choices=["low", "medium", "high", "critical"],
    description="优先级",
)
```

### UI 布局

```python
module.set_ui(
    tabs=[
        octo.UITab(
            key="tasks",
            label="任务",
            context="",         # "" = 账号详情页 | "create" = 创建页 | "list" = 列表页
            sections=[
                octo.UISection(
                    title="操作",
                    fields=[octo.UIField(key="x", label="X")],
                    buttons=[
                        octo.UIButton(
                            action="LIST_TASKS",
                            label="查询",
                            mode="sync",        # "sync" 或 "job"
                            variant="outline",  # 按钮样式
                            form=[...],         # 弹出表单参数
                            params={...},       # 固定参数（不弹表单）
                        ),
                    ],
                ),
            ],
        ),
    ],
    list_actions=[...],     # 账号列表页每行的快捷操作
)
```

### emit\_log — 日志输出

```python
octo.emit_log("消息文本", level="info")          # level: info/warning/error/debug
octo.emit_log("附带字段", level="warning", count=5, task_id="42")

# log_context：为代码块内所有日志自动附加字段
with octo.log_context(username="alice", action="VERIFY"):
    octo.emit_log("开始处理")   # 自动包含 username 和 action 字段
```

### Agent 事件流

```python
# 1. 初始化完成（Agent 状态变为 running）
octo.emit_daemon_init_ok("初始化成功", agent_id=agent_id)

# 2. 上报业务数据（可调用多次）
octo.emit_daemon_event({"type": "snapshot", "data": {...}}, message="采集完成")

# 3. 本轮执行结束（Agent 进入 idle，等待下次调用）
octo.emit_daemon_done("执行完毕", total=100)

# 4. 发生错误（Agent 状态变为 error）
octo.emit_daemon_error("FETCH_FAILED", "无法连接服务器", details={"url": "..."})
```

### OctoClient — 回调 OctoManger API

```python
# 从执行上下文中获取客户端（仅 job/agent 模式下有效）
octo_client = req.client()

if octo_client and req.account_id:
    # 将新字段合并写入账号 spec
    octo_client.patch_account_spec(req.account_id, {"plan": "pro"})

    # 通用 HTTP 方法
    data = octo_client.get("/api/v2/accounts")
    result = octo_client.post("/api/v2/something", body={...})
```

### success / error 响应

```python
# 成功响应
return octo.success({"key": "value"})

# 成功 + session（合并写入账号 spec，常用于注册）
return octo.success({"user": user}, session={"api_key": new_key})

# 错误响应（ActionRouter 同时捕获 raise ValueError）
return octo.error("ERROR_CODE", "人类可读的错误信息")
return octo.error("DETAIL_ERROR", "错误", details={"field": "value"})
```

***

## 假数据服务器 API 参考

所有需要认证的端点通过 `X-Api-Key` 请求头传递 API Key。

响应格式：

- 成功：`{"code": 0, "data": {...}}`
- 失败：`{"code": <非0>, "message": "..."}`

### 认证

```http
POST /auth/verify
Content-Type: application/json

{"username": "testuser", "api_key": "demo_testkey_12345678"}
```

```http
POST /auth/register
Content-Type: application/json

{"username": "newuser", "display_name": "新用户"}
```

### 用户

```http
GET /users/{username}
X-Api-Key: demo_testkey_12345678
```

### 任务

```http
# 查询（支持 ?status=pending&priority=critical&page=1&page_size=20）
GET /tasks
X-Api-Key: demo_testkey_12345678

# 创建
POST /tasks
X-Api-Key: demo_testkey_12345678
{"title": "新任务", "priority": "high"}

# 完成
PATCH /tasks/{id}/complete
X-Api-Key: demo_testkey_12345678

# 删除
DELETE /tasks/{id}
X-Api-Key: demo_testkey_12345678
```

### 统计

```http
GET /stats
X-Api-Key: demo_testkey_12345678
# 返回: {"total": 5, "by_status": {"pending": 2, ...}, "by_priority": {...}}
```

***

## Agent 配置示例

在 OctoManger 的「Agents」页面创建一个新 Agent：

| 字段     | 值               |
| ------ | --------------- |
| 名称     | `任务监控`          |
| 插件     | `octo_demo`     |
| Action | `AGENT_MONITOR` |
| 账号 ID  | `<你创建的账号 ID>`   |

启动 Agent 后，可在 Agent 详情页的事件流中看到：

- 每轮采集的任务统计快照
- critical 任务告警事件

Worker 循环间隔由环境变量 `WORKER_AGENT_LOOP_INTERVAL`（默认 `5m`）控制。

***

## 本地调试（gRPC）

可以直接把插件作为 gRPC 服务启动，再由 Worker 或自定义客户端调用：

```bash
export PYTHONPATH=/path/to/plugins/sdk/python
python3 main.py --address 127.0.0.1:50051
```

启动后 stderr 会看到类似输出：

```text
[octo_demo] 以 gRPC 微服务模式启动，监听 127.0.0.1:50051
```

