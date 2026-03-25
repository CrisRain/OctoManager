#!/usr/bin/env python3
"""
OctoDemo 插件 — 演示所有 Octo Plugin SDK 功能
=============================================

本文件是一个完整的插件示例，覆盖：
  • Module / Setting / ParamSpec / UITab / UISection / UIButton / UIField
  • 同步 action (mode="sync") 与 Job action (mode="job")
  • Agent action：emit_daemon_init_ok / emit_daemon_event / emit_daemon_done / emit_daemon_error
  • emit_log / log_context
  • OctoClient：回调 OctoManger API（写回 spec）
  • success / error 响应
  • REGISTER action + session 字段（自动填充新账号凭证）

配合使用：先启动 fake_server.py 提供模拟数据，再将本插件挂载到 OctoManger。
"""
from __future__ import annotations

import time
import os
from typing import Any

import octo

# ============================================================================
# 1. 声明模块
# ============================================================================

module = octo.Module(
    key="octo_demo",
    name="OctoDemo 演示",
    category="generic",
    # ── 账号凭证 JSON Schema ─────────────────────────────────────────────────
    # 用户在 OctoManger 中创建账号时填写的字段
    account_schema={
        "type": "object",
        "properties": {
            "username": {
                "type": "string",
                "title": "用户名",
                "description": "fake_server 中的用户名",
            },
            "api_key": {
                "type": "string",
                "title": "API Key",
                "description": "fake_server 返回的 API 密钥",
            },
            "base_url": {
                "type": "string",
                "title": "服务地址",
                "description": "fake_server 监听地址，例如 http://127.0.0.1:18080",
                "default": "http://127.0.0.1:18080",
            },
        },
        "required": ["username", "api_key"],
    },
    # ── 插件级设置（所有账号共享，管理员在插件页配置）────────────────────────
    settings=[
        octo.Setting(
            key="default_page_size",
            label="默认分页大小",
            type="string",
            default="20",
            description="列表查询时每页返回的条目数 (1–50)",
        ),
        octo.Setting(
            key="debug_mode",
            label="调试模式",
            type="string",
            default="false",
            description="设为 true 时输出更详细的日志",
        ),
        octo.Setting(
            key="agent_loop_hint",
            label="Agent 循环提示 (秒)",
            type="string",
            default="60",
            description="仅用于文档说明；实际间隔由 Worker 的 WORKER_AGENT_LOOP_INTERVAL 控制",
        ),
    ],
)

# ============================================================================
# 2. UI 布局
# ============================================================================
# set_ui 定义在 OctoManger 前端如何渲染账号详情页的按钮和表单

module.set_ui(
    tabs=[
        # ── Tab 1: 概览（账号详情页默认显示）────────────────────────────────
        octo.UITab(
            key="overview",
            label="概览",
            context="",             # 空字符串 = 在账号详情页显示
            sections=[
                octo.UISection(
                    title="账号操作",
                    buttons=[
                        octo.UIButton(
                            action="VERIFY",
                            label="验证账号",
                            mode="sync",            # sync = 立即执行，阻塞等待结果
                            variant="outline",
                        ),
                        octo.UIButton(
                            action="GET_PROFILE",
                            label="获取并同步资料",
                            mode="sync",
                            variant="outline",
                        ),
                    ],
                ),
            ],
        ),

        # ── Tab 2: 任务管理──────────────────────────────────────────────────
        octo.UITab(
            key="tasks",
            label="任务",
            context="",
            sections=[
                # 查询任务（带可选过滤参数）
                octo.UISection(
                    title="查询任务",
                    buttons=[
                        octo.UIButton(
                            action="LIST_TASKS",
                            label="查询任务列表",
                            mode="sync",
                            form=[
                                octo.ParamSpec(
                                    name="status",
                                    type="string",
                                    required=False,
                                    choices=["", "pending", "in_progress", "completed", "cancelled"],
                                    description="按状态过滤（留空=全部）",
                                ),
                                octo.ParamSpec(
                                    name="priority",
                                    type="string",
                                    required=False,
                                    choices=["", "low", "medium", "high", "critical"],
                                    description="按优先级过滤（留空=全部）",
                                ),
                                octo.ParamSpec(
                                    name="page",
                                    type="string",
                                    default="1",
                                    required=False,
                                    description="页码",
                                ),
                            ],
                        ),
                    ],
                ),
                # 新建任务（使用 mode="job" 异步执行，写入 job_logs）
                octo.UISection(
                    title="新建任务",
                    buttons=[
                        octo.UIButton(
                            action="CREATE_TASK",
                            label="创建任务",
                            mode="job",             # job = 提交后台 Job，可在 Jobs 页查看日志
                            form=[
                                octo.ParamSpec(
                                    name="title",
                                    type="string",
                                    required=True,
                                    description="任务标题",
                                ),
                                octo.ParamSpec(
                                    name="priority",
                                    type="string",
                                    required=False,
                                    default="medium",
                                    choices=["low", "medium", "high", "critical"],
                                    description="优先级",
                                ),
                            ],
                        ),
                    ],
                ),
                # 操作已有任务
                octo.UISection(
                    title="操作任务",
                    buttons=[
                        octo.UIButton(
                            action="COMPLETE_TASK",
                            label="标记为完成",
                            mode="sync",
                            form=[
                                octo.ParamSpec(
                                    name="task_id",
                                    type="string",
                                    required=True,
                                    description="任务 ID（从查询结果中复制）",
                                ),
                            ],
                        ),
                        octo.UIButton(
                            action="DELETE_TASK",
                            label="删除任务",
                            mode="sync",
                            variant="outline",
                            form=[
                                octo.ParamSpec(
                                    name="task_id",
                                    type="string",
                                    required=True,
                                    description="任务 ID",
                                ),
                            ],
                        ),
                    ],
                ),
            ],
        ),

        # ── Tab 3: 注册（仅在「创建账号」页面显示）──────────────────────────
        octo.UITab(
            key="register",
            label="注册",
            context="create",       # "create" = 仅在创建账号时显示
            sections=[
                octo.UISection(
                    title="自动注册新账号",
                    buttons=[
                        octo.UIButton(
                            action="REGISTER",
                            label="自动注册",
                            mode="sync",
                            form=[
                                octo.ParamSpec(
                                    name="display_name",
                                    type="string",
                                    required=False,
                                    description="显示名称（可选，不填则使用用户名）",
                                ),
                            ],
                        ),
                    ],
                ),
            ],
        ),
    ],
    # ── 列表视图快捷操作（账号列表页每行右侧显示）───────────────────────────
    list_actions=[
        octo.UIButton(
            action="VERIFY",
            label="验证",
            mode="sync",
            variant="outline",
        ),
    ],
)

# ============================================================================
# 3. 工具函数
# ============================================================================

def _api_client(req: octo.ModuleRequest) -> octo.OctoClient:
    """用账号 spec 中的 base_url / api_key 构造指向 fake_server 的客户端。"""
    base_url = str(req.spec.get("base_url", "http://127.0.0.1:18080")).rstrip("/")
    api_key = str(req.spec.get("api_key", ""))
    return octo.OctoClient(base_url, api_key)


def _debug(req: octo.ModuleRequest) -> bool:
    """读取 debug_mode 插件设置（通过 params 注入）。"""
    return str(req.params.get("debug_mode", "false")).lower() == "true"


# ============================================================================
# 4. Action 处理器
# ============================================================================

# ── VERIFY ──────────────────────────────────────────────────────────────────

@module.action("VERIFY", description="验证账号凭证是否有效")
def handle_verify(req: octo.ModuleRequest) -> dict[str, Any]:
    """
    同步验证。演示：
      - emit_log 基本用法
      - OctoAPIError 捕获
      - success / error 返回
    """
    username = str(req.spec.get("username", "")).strip()
    api_key = str(req.spec.get("api_key", "")).strip()
    base_url = str(req.spec.get("base_url", "http://127.0.0.1:18080")).rstrip("/")

    if not username or not api_key:
        return octo.error("MISSING_CREDENTIALS", "spec.username 和 spec.api_key 均不能为空")

    octo.emit_log("开始验证账号", level="info", username=username)

    client = octo.OctoClient(base_url, api_key)
    try:
        data = client.post("/auth/verify", body={"username": username, "api_key": api_key})
    except octo.OctoAPIError as exc:
        octo.emit_log("验证请求异常", level="error", error=str(exc))
        return octo.error("API_ERROR", f"请求失败: {exc}")

    valid: bool = bool(data and data.get("valid"))
    user: dict = data.get("user") or {} if data else {}

    if not valid:
        octo.emit_log("凭证无效", level="warning", username=username)
        return octo.error("INVALID_CREDENTIALS", "用户名或 API Key 不匹配")

    octo.emit_log("验证通过", level="info", username=username, plan=user.get("plan"))
    return octo.success({"valid": True, "user": user})


# ── GET_PROFILE ─────────────────────────────────────────────────────────────

@module.action("GET_PROFILE", description="获取用户资料，并将关键字段写回账号 spec")
def handle_get_profile(req: octo.ModuleRequest) -> dict[str, Any]:
    """
    演示：
      - log_context 上下文绑定
      - OctoClient.patch_account_spec() 回写账号字段
      - req.client() 获取 OctoManger 内部 API 客户端
    """
    username = str(req.spec.get("username", "")).strip()
    if not username:
        return octo.error("MISSING_USERNAME", "spec.username 不能为空")

    # log_context 会将字段自动附加到该作用域内所有 emit_log 调用
    with octo.log_context(username=username, action="GET_PROFILE"):
        octo.emit_log("正在获取用户资料")

        client = _api_client(req)
        try:
            user = client.get(f"/users/{username}")
        except octo.OctoAPIError as exc:
            octo.emit_log("获取资料失败", level="error", error=str(exc))
            return octo.error("API_ERROR", str(exc))

        octo.emit_log(
            "资料获取成功",
            plan=user.get("plan"),
            tasks_quota=user.get("tasks_quota"),
        )

        # 通过 OctoManger 内部 API 将新字段写回账号 spec
        octo_client = req.client()
        if octo_client and req.account_id:
            try:
                octo_client.patch_account_spec(req.account_id, {
                    "display_name": user.get("display_name"),
                    "plan": user.get("plan"),
                    "tasks_quota": user.get("tasks_quota"),
                })
                octo.emit_log("已将资料同步到账号 spec")
            except Exception as exc:
                # 非致命：记录警告，不中断主流程
                octo.emit_log("同步 spec 失败（非致命）", level="warning", error=str(exc))

    return octo.success({"user": user})


# ── LIST_TASKS ───────────────────────────────────────────────────────────────

@module.action(
    "LIST_TASKS",
    description="分页查询账号下的任务",
    params=[
        octo.ParamSpec(name="status",    type="string", required=False, description="状态过滤"),
        octo.ParamSpec(name="priority",  type="string", required=False, description="优先级过滤"),
        octo.ParamSpec(name="page",      type="string", default="1",    required=False, description="页码"),
        octo.ParamSpec(name="page_size", type="string", default="20",   required=False, description="每页条数"),
    ],
)
def handle_list_tasks(req: octo.ModuleRequest) -> dict[str, Any]:
    """演示：ParamSpec 可选参数读取 + 分页查询。"""
    query: dict[str, Any] = {}
    status = str(req.params.get("status", "")).strip()
    priority = str(req.params.get("priority", "")).strip()

    if status:
        query["status"] = status
    if priority:
        query["priority"] = priority

    try:
        query["page"] = str(max(1, int(req.params.get("page", 1) or 1)))
        query["page_size"] = str(min(50, max(1, int(req.params.get("page_size", 20) or 20))))
    except (ValueError, TypeError):
        query["page"] = "1"
        query["page_size"] = "20"

    client = _api_client(req)
    try:
        data = client.get("/tasks", query=query)
    except octo.OctoAPIError as exc:
        return octo.error("API_ERROR", str(exc))

    items = (data or {}).get("items", [])
    octo.emit_log(f"查询到 {len(items)} 条任务（共 {(data or {}).get('total', 0)} 条）")
    return octo.success(data)


# ── CREATE_TASK ──────────────────────────────────────────────────────────────

@module.action(
    "CREATE_TASK",
    description="创建新任务（mode=job，写入后台日志）",
    params=[
        octo.ParamSpec(name="title",    type="string", required=True,  description="任务标题"),
        octo.ParamSpec(name="priority", type="string", required=False, default="medium",
                       choices=["low", "medium", "high", "critical"], description="优先级"),
    ],
)
def handle_create_task(req: octo.ModuleRequest) -> dict[str, Any]:
    """
    演示：
      - 必填参数校验（raise ValueError → 自动转为 VALIDATION_FAILED 错误）
      - mode="job"：由后台 Worker 执行，结果写入 job_logs
    """
    title = str(req.params.get("title", "")).strip()
    if not title:
        raise ValueError("title 不能为空")         # ActionRouter 会捕获并返回 VALIDATION_FAILED

    priority = str(req.params.get("priority", "medium")).strip() or "medium"
    if priority not in ("low", "medium", "high", "critical"):
        raise ValueError(f"无效的优先级: {priority!r}，可选值: low/medium/high/critical")

    octo.emit_log("正在创建任务", title=title, priority=priority)

    client = _api_client(req)
    try:
        task = client.post("/tasks", body={"title": title, "priority": priority})
    except octo.OctoAPIError as exc:
        return octo.error("API_ERROR", str(exc))

    octo.emit_log("任务创建成功", task_id=(task or {}).get("id"))
    return octo.success({"task": task})


# ── COMPLETE_TASK ────────────────────────────────────────────────────────────

@module.action(
    "COMPLETE_TASK",
    description="将指定任务标记为已完成",
    params=[
        octo.ParamSpec(name="task_id", type="string", required=True, description="任务 ID"),
    ],
)
def handle_complete_task(req: octo.ModuleRequest) -> dict[str, Any]:
    task_id = str(req.params.get("task_id", "")).strip()
    if not task_id:
        raise ValueError("task_id 不能为空")

    client = _api_client(req)
    try:
        task = client.patch(f"/tasks/{task_id}/complete")
    except octo.OctoAPIError as exc:
        return octo.error("API_ERROR", str(exc))

    octo.emit_log("任务已完成", task_id=task_id)
    return octo.success({"task": task})


# ── DELETE_TASK ──────────────────────────────────────────────────────────────

@module.action(
    "DELETE_TASK",
    description="删除指定任务",
    params=[
        octo.ParamSpec(name="task_id", type="string", required=True, description="任务 ID"),
    ],
)
def handle_delete_task(req: octo.ModuleRequest) -> dict[str, Any]:
    task_id = str(req.params.get("task_id", "")).strip()
    if not task_id:
        raise ValueError("task_id 不能为空")

    client = _api_client(req)
    try:
        result = client.delete(f"/tasks/{task_id}")
    except octo.OctoAPIError as exc:
        return octo.error("API_ERROR", str(exc))

    octo.emit_log("任务已删除", task_id=task_id)
    return octo.success(result)


# ── REGISTER ─────────────────────────────────────────────────────────────────

@module.action(
    "REGISTER",
    description="自动注册新账号并填充凭证（仅在创建账号时使用）",
    params=[
        octo.ParamSpec(name="display_name", type="string", required=False, description="显示名称"),
    ],
)
def handle_register(req: octo.ModuleRequest) -> dict[str, Any]:
    """
    演示：
      - success(result, session=...) — session 字段由 OctoManger 自动合并到账号 spec
      - 在 context="create" 的 Tab 中执行，不需要预先填写 api_key
    """
    username = str(req.spec.get("username", "")).strip()
    if not username:
        raise ValueError("spec.username 是必填项，请先填写用户名再点注册")

    display_name = str(req.params.get("display_name", "") or username).strip()
    base_url = str(req.spec.get("base_url", "http://127.0.0.1:18080")).rstrip("/")

    # 注册端点无需认证，api_token 传空串
    register_client = octo.OctoClient(base_url, "")
    octo.emit_log("正在注册账号", username=username)

    try:
        data = register_client.post("/auth/register", body={
            "username": username,
            "display_name": display_name,
        })
    except octo.OctoAPIError as exc:
        return octo.error("REGISTER_FAILED", str(exc))

    new_api_key = (data or {}).get("api_key", "")
    user = (data or {}).get("user", {})

    octo.emit_log("注册成功", username=username, plan=(user or {}).get("plan"))

    # session 字段会被 OctoManger 合并到账号 spec —— 这里把 api_key 自动写入
    return octo.success(
        {"user": user},
        session={"api_key": new_api_key},
    )


# ── AGENT_MONITOR ─────────────────────────────────────────────────────────────

@module.action("AGENT_MONITOR", description="Agent 模式：定期采集任务统计并上报异常")
def handle_agent_monitor(req: octo.ModuleRequest) -> dict[str, Any]:
    """
    Agent 专用 action，展示完整的 daemon 事件序列：

      emit_daemon_init_ok  → 初始化完成，Agent 服务标记为 running
      emit_daemon_event    → 每次采集到数据后上报（可多次）
      emit_daemon_done     → 本轮执行结束（Agent 睡眠后下次再调用）
      emit_daemon_error    → 发生不可恢复错误时调用

    Worker 的 RunSupervisor 每隔 WORKER_AGENT_LOOP_INTERVAL 调用一次此 action。
    """
    agent_id = str(req.context.get("agent_id", "unknown"))
    username = str(req.spec.get("username", "")).strip()
    api_key = str(req.spec.get("api_key", "")).strip()
    base_url = str(req.spec.get("base_url", "http://127.0.0.1:18080")).rstrip("/")

    if not username or not api_key:
        octo.emit_daemon_error("MISSING_CREDENTIALS", "username 和 api_key 均不能为空")
        return octo.error("MISSING_CREDENTIALS", "缺少账号凭证")

    # 1. 通知 Worker：初始化完成
    octo.emit_daemon_init_ok(
        "任务监控 Agent 已就绪",
        agent_id=agent_id,
        username=username,
    )

    client = octo.OctoClient(base_url, api_key)

    with octo.log_context(username=username, agent_id=agent_id, mode="agent"):

        # 2. 采集任务统计
        octo.emit_log("开始采集统计数据")
        try:
            stats = client.get("/stats")
        except octo.OctoAPIError as exc:
            octo.emit_log("采集失败", level="error", error=str(exc))
            octo.emit_daemon_error("STATS_FETCH_FAILED", f"获取统计数据失败: {exc}")
            return octo.error("STATS_FETCH_FAILED", str(exc))

        # 3. 上报统计摘要事件
        octo.emit_daemon_event(
            {
                "type": "stats_snapshot",
                "stats": stats,
                "collected_at": time.strftime("%Y-%m-%dT%H:%M:%SZ"),
            },
            message=f"采集完成：共 {(stats or {}).get('total', 0)} 条任务",
        )

        # 4. 检查高优先级待处理任务并发出告警事件
        try:
            pending_critical = client.get("/tasks", query={"status": "pending", "priority": "critical"})
        except octo.OctoAPIError:
            pending_critical = None

        critical_count = (pending_critical or {}).get("total", 0)
        if critical_count > 0:
            octo.emit_log(
                f"发现 {critical_count} 条 critical 待处理任务",
                level="warning",
                count=critical_count,
            )
            octo.emit_daemon_event(
                {
                    "type": "alert",
                    "alert": "critical_tasks_pending",
                    "count": critical_count,
                    "items": (pending_critical or {}).get("items", []),
                },
                message=f"警告：有 {critical_count} 条 critical 任务待处理",
            )

        # 5. 通知 Worker：本轮执行结束
        octo.emit_daemon_done(
            "本轮监控完成",
            total_tasks=(stats or {}).get("total", 0),
            critical_pending=critical_count,
        )

    return octo.success({
        "stats": stats,
        "critical_pending": critical_count,
        "timestamp": time.strftime("%Y-%m-%dT%H:%M:%SZ"),
    })


# ============================================================================
# 5. 入口点
# ============================================================================
# 仅保留 gRPC 微服务模式：
#
#   python3 main.py [--address 127.0.0.1:50051]
#
# Worker 会在启动时拉起该进程，并通过 gRPC 长连接与插件通信。
# 依赖: pip install grpcio

if __name__ == "__main__":
    import sys

    address = os.environ.get("OCTO_PLUGIN_ADDR", "127.0.0.1:50051")
    try:
        idx = sys.argv.index("--address")
        address = sys.argv[idx + 1]
    except (ValueError, IndexError):
        pass

    print(f"[octo_demo] 以 gRPC 微服务模式启动，监听 {address}", file=sys.stderr)
    raise SystemExit(module.serve(address=address))
