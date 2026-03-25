# 插件开发教程

本文是一份面向 OctoManger 插件作者的完整入门教程，目标是让你从零开始，完成一个可以被平台发现、同步、调用、调试和维护的 Python 插件。

如果你只想先跑通一遍完整链路，建议先看“快速开始”，然后再回来看后面的设计细节。

---

## 1. 先理解插件在系统里的位置

OctoManger 的插件本质上是由 Worker 拉起的独立 Python 进程，平台通过 gRPC 与插件通信。

一条典型链路如下：

1. 你把插件放到 `plugins/modules/<plugin_key>/`
2. Worker 启动时发现插件目录
3. Worker 为插件创建 `.venv` 并安装 `requirements.txt`
4. Worker 启动插件进程，并通过 gRPC 做健康检查
5. 平台读取插件导出的账号类型、动作、设置、UI 描述
6. 平台把这些能力同步到系统里
7. 用户在前端创建账号、执行任务、运行 Agent 时，平台再调用插件动作

当前最完整的参考实现是示例插件 [octo_demo](./plugins/modules/octo_demo)。

---

## 2. 快速开始

### 2.1 环境准备

你至少需要：

- Python 3
- 已能运行的 OctoManger API 和 Worker
- 正确设置插件目录和 SDK 目录

常见环境变量：

```bash
export DATABASE_DSN="postgres://octo:octo@localhost:5432/octomanger?sslmode=disable"
export PLUGINS_DIR="/home/cris/OctoManger/plugins/modules"
export PLUGIN_SDK_DIR="/home/cris/OctoManger/plugins/sdk/python"
export PYTHON_BIN="python3"
```

初始化数据库并启动后端：

```bash
go run ./apps/migrate migrate
go run ./apps/api
go run ./apps/worker
```

---

## 3. 插件目录结构

一个插件目录通常长这样：

```text
plugins/modules/my_plugin/
├── main.py
├── requirements.txt
└── account_type.my_plugin.json
```

不过需要注意：

- 当前推荐写法是让 SDK 在运行时自动生成 `account_type.<key>.json`
- 因此很多新插件并不一定手写这个文件
- 示例插件 `octo_demo` 就主要依赖 `main.py` 中的 `Module(...)` 定义生成账号类型描述

如果你只做最小插件，核心文件通常只有：

- `main.py`
- `requirements.txt`

---

## 4. 推荐写法：使用 Module

当前仓库里最推荐的写法不是旧式 `ActionRouter + run_module`，而是：

- `Module(...)` 定义插件元数据
- `@module.action(...)` 注册动作
- `module.serve(...)` 启动服务

最小示例：

```python
from octo import Module, success

module = Module(
    key="hello_demo",
    name="Hello Demo",
    category="generic",
    account_schema={
        "type": "object",
        "properties": {
            "api_key": {
                "type": "string",
                "title": "API Key"
            }
        },
        "required": ["api_key"]
    },
)

@module.action("PING")
def ping(req):
    return success({
        "message": "pong",
        "account": req.account_spec,
    })

if __name__ == "__main__":
    module.serve()
```

这个例子已经具备以下能力：

- 平台能识别插件 key、name、category
- 平台能生成账号表单
- 平台能调用 `PING` 动作

---

## 5. Request 对象里能拿到什么

插件动作的参数不是随意对象，而是平台构造的请求上下文。

常用字段包括：

- `req.action`：当前动作名
- `req.input`：本次执行输入
- `req.account_spec`：账号的 `spec`
- `req.account_id`：账号 ID
- `req.job_execution_id`：任务执行 ID
- `req.agent_id`：Agent ID
- `req.settings`：插件设置

如果你需要从插件回调平台 API，还可以使用：

- `req.client()`

可参考：

- [Request 定义](./plugins/sdk/python/octo/_request.py)
- [HTTP 客户端](./plugins/sdk/python/octo/_client.py)

---

## 6. 定义账号结构

插件最重要的一部分通常是账号结构，也就是 `account_schema`。

示例：

```python
module = Module(
    key="example_api",
    name="Example API",
    category="generic",
    account_schema={
        "type": "object",
        "properties": {
            "base_url": {
                "type": "string",
                "title": "Base URL"
            },
            "api_key": {
                "type": "string",
                "title": "API Key"
            }
        },
        "required": ["base_url", "api_key"]
    },
)
```

这会影响平台上的：

- 账号创建表单
- 账号编辑表单
- 动作执行时的账号数据结构

如果你的插件依赖 OAuth、Session、Token 或额外业务字段，也都应该放进这里统一定义。

---

## 7. 定义动作

插件动作就是平台可以调用的能力单元。

一个动作通常会做这些事：

1. 读取账号信息
2. 读取输入参数
3. 请求外部服务
4. 输出结构化日志
5. 返回结果

示例：

```python
from octo import Module, ParamSpec, emit_log, success
import requests

module = Module(
    key="task_demo",
    name="Task Demo",
    category="generic",
    account_schema={
        "type": "object",
        "properties": {
            "base_url": {"type": "string"},
            "api_key": {"type": "string"}
        },
        "required": ["base_url", "api_key"]
    },
)

@module.action(
    "LIST_TASKS",
    params=[
        ParamSpec(
            key="page",
            type="string",
            required=False,
            default="1",
            description="页码"
        )
    ],
)
def list_tasks(req):
    base_url = req.account_spec["base_url"]
    api_key = req.account_spec["api_key"]
    page = req.input.get("page", "1")

    emit_log("info", "开始获取任务列表", {"page": page})

    response = requests.get(
        f"{base_url}/tasks",
        headers={"Authorization": f"Bearer {api_key}"},
        params={"page": page},
        timeout=15,
    )
    response.raise_for_status()

    data = response.json()

    emit_log("info", "任务列表获取成功", {"count": len(data.get('items', []))})

    return success(data)

if __name__ == "__main__":
    module.serve()
```

建议：

- action 名统一使用大写风格
- 参数通过 `ParamSpec` 定义，便于平台展示和校验
- 外部请求要设置超时
- 返回值尽量结构化

---

## 8. 三种常见执行模式

OctoManger 中插件动作通常会跑在三种场景里。

### 8.1 同步模式

适合：

- 测试连接
- 获取账号资料
- 小规模查询

特点：

- 用户发起后立即等待返回
- 适合短操作

示例动作：

- `VERIFY`
- `GET_PROFILE`

可参考 [octo_demo/main.py](./plugins/modules/octo_demo/main.py)

### 8.2 Job 模式

适合：

- 创建任务
- 批量同步
- 耗时调用

特点：

- 由 Job 执行器异步运行
- 日志会进入 `job_logs`
- 可在执行记录页面查看

### 8.3 Agent 模式

适合：

- 轮询监控
- 长期运行
- 周期事件上报

特点：

- 由 Worker 周期调度
- 日志会进入 `agent_logs`
- 适合实现持续监听或后台守护逻辑

---

## 9. 日志怎么写

插件内建议始终写结构化日志。

常用方式：

```python
from octo import emit_log

emit_log("info", "开始执行", {"step": "start"})
emit_log("error", "调用失败", {"reason": "timeout"})
```

日志的意义有三层：

- 给开发者调试
- 给用户看执行进度
- 给平台记录执行历史

当前系统对日志做了保留控制：

- 每个 Job 执行流只保留最近 200 条
- 每个 Agent 日志流只保留最近 200 条

所以日志要写关键信息，不要无意义刷屏。

---

## 10. 如何回调平台 API

有些插件不仅要“从外部拉数据”，还要“把结果写回平台”。

这时可以使用：

```python
client = req.client()
```

常见用途：

- 更新账号资料
- 写入补充字段
- 触发平台内部动作

示例思路：

```python
@module.action("REFRESH_PROFILE")
def refresh_profile(req):
    profile = {
        "nickname": "demo-user",
        "avatar": "https://example.com/avatar.png"
    }

    client = req.client()
    client.patch_account(
        req.account_id,
        spec=profile
    )

    return success(profile)
```

实际可用方法请以 SDK 中的客户端实现为准：

- [OctoClient](./plugins/sdk/python/octo/_client.py)

---

## 11. UI 配置怎么做

如果你希望平台前端为插件渲染更友好的交互界面，可以在 `Module` 里定义 UI。

平台支持的方向包括：

- Tab
- Section
- Button
- Field

完整参考最好直接看示例插件的 UI 定义：

- [octo_demo/main.py](./plugins/modules/octo_demo/main.py)

建议理解方式：

- `account_schema` 决定账号数据结构
- `settings` 决定插件设置项
- `ui` 决定页面如何呈现这些能力

如果你刚开始写插件，可以先不写 UI，先把动作跑通。

---

## 12. Worker 是如何启动插件的

这部分很重要，因为它能解释很多“为什么插件没起来”的问题。

Worker 会：

1. 扫描 `PLUGINS_DIR`
2. 查找每个插件目录中的 `main.py`
3. 自动创建 `.venv`
4. 执行 `pip install -r requirements.txt`
5. 以 gRPC 服务方式启动插件
6. 做健康检查
7. 同步插件能力到平台

相关实现：

- [grpclauncher manager](./internal/domains/plugins/grpclauncher/manager.go)

这意味着：

- `requirements.txt` 必须可安装
- `main.py` 必须能独立启动
- 插件启动时不要依赖交互输入

---

## 13. 本地调试完整闭环

最推荐的本地调试对象就是 `octo_demo`。

### 13.1 先启动假服务

```bash
python3 plugins/modules/octo_demo/fake_server.py
```

### 13.2 启动主系统

```bash
go run ./apps/migrate migrate
go run ./apps/api
go run ./apps/worker
```

### 13.3 确认插件被发现

你可以：

- 打开插件管理页面
- 或调用插件同步接口

```bash
curl -X POST http://localhost:8080/api/v2/plugins/sync \
  -H "X-Admin-Key: your-admin-key"
```

### 13.4 创建账号并测试动作

可以先走这些顺序：

1. 创建账号
2. 执行 `VERIFY`
3. 执行 `GET_PROFILE`
4. 创建 Job 调用 `CREATE_TASK`
5. 创建 Agent 调用 `AGENT_MONITOR`

这样可以把同步、异步、长运行三类场景都走一遍。

---

## 14. 常见问题

### 14.1 插件没有被发现

优先检查：

- 目录是否位于 `PLUGINS_DIR`
- 是否存在 `main.py`
- Worker 是否启动
- 插件 gRPC 地址配置是否正确

### 14.2 依赖安装失败

优先检查：

- `requirements.txt` 是否可安装
- Python 版本是否兼容
- 网络是否可访问 PyPI 或镜像源

### 14.3 插件启动了但平台看不到动作

优先检查：

- `module.key` 是否稳定且唯一
- 是否执行过插件同步
- `module.serve()` 是否真正执行

### 14.4 动作能运行但没有日志

优先检查：

- 是否调用了 `emit_log`
- 是否有异常在函数外被提前抛出
- 当前日志是否已被 200 条保留策略裁剪

---

## 15. 开发建议

### 15.1 先小后大

建议开发顺序：

1. 先写最小 `PING`
2. 再写 `VERIFY`
3. 再接真实外部 API
4. 再补 settings、UI、复杂动作
5. 最后再做 Agent

### 15.2 保持输入输出稳定

插件最容易出问题的地方是：

- `account_schema` 改了，但历史账号数据没兼容
- action 输入字段改了，但前端仍传旧结构

所以你要尽量保持：

- 账号字段名稳定
- action 输入结构稳定
- 返回结果结构稳定

### 15.3 日志写关键节点

建议日志最少覆盖：

- 开始执行
- 请求外部服务前
- 成功返回
- 异常分支

---

## 16. 一个推荐的最小实战模板

```python
from octo import Module, ParamSpec, emit_log, success
import requests

module = Module(
    key="simple_demo",
    name="Simple Demo",
    category="generic",
    account_schema={
        "type": "object",
        "properties": {
            "base_url": {"type": "string", "title": "Base URL"},
            "api_key": {"type": "string", "title": "API Key"}
        },
        "required": ["base_url", "api_key"]
    },
)

@module.action(
    "VERIFY",
    params=[],
)
def verify(req):
    emit_log("info", "开始验证账号", {})

    response = requests.get(
        f"{req.account_spec['base_url']}/profile",
        headers={"Authorization": f"Bearer {req.account_spec['api_key']}"},
        timeout=10,
    )
    response.raise_for_status()

    emit_log("info", "验证成功", {})
    return success(response.json())

@module.action(
    "LIST_TASKS",
    params=[
        ParamSpec(key="page", type="string", required=False, default="1")
    ],
)
def list_tasks(req):
    page = req.input.get("page", "1")
    emit_log("info", "开始查询任务", {"page": page})

    response = requests.get(
        f"{req.account_spec['base_url']}/tasks",
        headers={"Authorization": f"Bearer {req.account_spec['api_key']}"},
        params={"page": page},
        timeout=10,
    )
    response.raise_for_status()

    return success(response.json())

if __name__ == "__main__":
    module.serve()
```

配套的 `requirements.txt`：

```text
requests>=2.32.0
```

---

## 17. 建议阅读顺序

如果你准备正式开始写插件，建议按这个顺序读代码：

1. [本教程](./PLUGIN_DEVELOPMENT.md)
2. [README 中的插件说明](./README.md)
3. [octo_demo README](./plugins/modules/octo_demo/README.md)
4. [octo_demo main.py](./plugins/modules/octo_demo/main.py)
5. [SDK 导出入口](./plugins/sdk/python/octo/__init__.py)
6. [Worker 插件启动器](./internal/domains/plugins/grpclauncher/manager.go)

---

## 18. 最后 checklist

在你提交一个新插件前，至少确认这些事情：

- 插件目录位于 `plugins/modules/<plugin_key>/`
- `main.py` 可独立运行
- `requirements.txt` 可安装
- `Module.key` 稳定且唯一
- `account_schema` 已覆盖必要字段
- 至少有一个可调用 action
- 关键步骤有结构化日志
- Worker 启动后插件能通过健康检查
- 平台执行 `plugins/sync` 后能看到插件能力

如果以上都通过，这个插件通常就已经具备可用性了。
