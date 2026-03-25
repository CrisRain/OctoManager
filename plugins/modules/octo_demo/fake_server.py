#!/usr/bin/env python3
"""
OctoDemo 假数据服务器
====================
模拟一个真实的任务管理 API，供 octo_demo 插件连接使用。

用法:
    python3 fake_server.py              # 默认 127.0.0.1:18080
    python3 fake_server.py --port 9000  # 自定义端口
    python3 fake_server.py --host 0.0.0.0 --port 9000

内置测试账号:
    username : testuser
    api_key  : demo_testkey_12345678
    base_url : http://127.0.0.1:18080

API 端点:
    POST /auth/verify              验证账号凭证
    POST /auth/register            注册新账号（无需认证）
    GET  /users/{username}         获取用户资料    (需认证)
    GET  /tasks                    查询任务列表    (需认证)
    POST /tasks                    创建任务        (需认证)
    PATCH /tasks/{id}/complete     完成任务        (需认证)
    DELETE /tasks/{id}             删除任务        (需认证)
    GET  /stats                    任务统计        (需认证)

认证方式: 请求头 X-Api-Key: <api_key>

响应格式: {"code": 0, "data": {...}}  成功
          {"code": <非0>, "message": "..."} 失败
"""
from __future__ import annotations

import argparse
import json
import random
import string
import threading
import time
from http.server import BaseHTTPRequestHandler, HTTPServer
from urllib.parse import parse_qs, urlparse

# ---------------------------------------------------------------------------
# 内存数据存储
# ---------------------------------------------------------------------------

_lock = threading.Lock()
_users: dict[str, dict] = {}       # username -> user dict
_tasks: dict[str, dict] = {}       # task_id  -> task dict
_api_keys: dict[str, str] = {}     # api_key  -> username
_id_counter = 0


def _new_id() -> str:
    global _id_counter
    _id_counter += 1
    return str(_id_counter)


def _rand_str(n: int, chars: str = string.ascii_lowercase) -> str:
    return "".join(random.choices(chars, k=n))


def _now() -> str:
    return time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())


def _past(max_days: int = 30) -> str:
    offset = random.randint(0, 86400 * max_days)
    return time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime(time.time() - offset))


def _fake_task(owner: str) -> dict:
    return {
        "id": _new_id(),
        "title": random.choice([
            "审查代码变更", "更新 API 文档", "修复登录 Bug",
            "优化数据库查询", "添加单元测试", "部署到生产环境",
            "安全审计报告", "性能压测分析", "清理过期日志",
            "迁移旧版数据", "编写接口规范", "重构认证模块",
        ]),
        "status": random.choice(["pending", "in_progress", "completed", "cancelled"]),
        "priority": random.choice(["low", "medium", "high", "critical"]),
        "owner": owner,
        "created_at": _past(),
        "updated_at": _now(),
    }


def _seed_data() -> None:
    """初始化随机假数据（5 个随机用户 + 1 个固定测试账号）。"""
    random.seed(42)
    first_names = ["Alice", "Bob", "Charlie", "Diana", "Eve"]
    last_names = ["Smith", "Jones", "Brown", "Wilson", "Taylor"]

    for i in range(5):
        uname = "user_" + _rand_str(6)
        akey = "demo_" + _rand_str(24, string.ascii_lowercase + string.digits)
        user = {
            "id": _new_id(),
            "username": uname,
            "display_name": f"{first_names[i]} {last_names[i]}",
            "email": f"{uname}@example.com",
            "plan": random.choice(["free", "pro", "enterprise"]),
            "tasks_quota": random.choice([10, 100, 1000]),
            "created_at": _now(),
        }
        _users[uname] = user
        _api_keys[akey] = uname
        for _ in range(random.randint(3, 8)):
            t = _fake_task(uname)
            _tasks[t["id"]] = t

    # 固定测试账号，便于快速上手
    _users["testuser"] = {
        "id": _new_id(),
        "username": "testuser",
        "display_name": "测试用户",
        "email": "testuser@example.com",
        "plan": "pro",
        "tasks_quota": 100,
        "created_at": _now(),
    }
    _api_keys["demo_testkey_12345678"] = "testuser"
    for _ in range(5):
        t = _fake_task("testuser")
        _tasks[t["id"]] = t


# ---------------------------------------------------------------------------
# HTTP 处理器
# ---------------------------------------------------------------------------

class Handler(BaseHTTPRequestHandler):

    def log_message(self, fmt, *args):  # type: ignore[override]
        print(f"[fake-server] {self.address_string()} {fmt % args}")

    # ── helpers ────────────────────────────────────────────────────────────

    def _authed(self) -> str | None:
        """返回当前请求对应的 username，未认证返回 None。"""
        key = self.headers.get("X-Api-Key", "").strip()
        with _lock:
            return _api_keys.get(key)

    def _send_json(self, data: object, status: int = 200) -> None:
        body = json.dumps({"code": 0, "data": data}, ensure_ascii=False).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json; charset=utf-8")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def _send_err(self, code: int, msg: str, status: int = 400) -> None:
        body = json.dumps({"code": code, "message": msg}, ensure_ascii=False).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json; charset=utf-8")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def _read_body(self) -> dict:
        try:
            length = int(self.headers.get("Content-Length", 0))
            return json.loads(self.rfile.read(length)) if length > 0 else {}
        except Exception:
            return {}

    # ── GET ────────────────────────────────────────────────────────────────

    def do_GET(self):
        parsed = urlparse(self.path)
        path = parsed.path.rstrip("/")
        qs = parse_qs(parsed.query)

        def q(key: str, default: str = "") -> str:
            vals = qs.get(key)
            return vals[0].strip() if vals else default

        parts = [p for p in path.split("/") if p]

        # GET /users/{username}
        if len(parts) == 2 and parts[0] == "users":
            username = self._authed()
            if not username:
                return self._send_err(4010, "unauthorized", 401)
            with _lock:
                user = _users.get(parts[1])
            if not user:
                return self._send_err(4040, "user not found", 404)
            return self._send_json(user)

        # GET /tasks  (支持 status/priority/page/page_size 查询参数)
        if len(parts) == 1 and parts[0] == "tasks":
            username = self._authed()
            if not username:
                return self._send_err(4010, "unauthorized", 401)
            status_f = q("status")
            priority_f = q("priority")
            try:
                page = max(1, int(q("page", "1") or "1"))
                page_size = min(50, max(1, int(q("page_size", "20") or "20")))
            except ValueError:
                page, page_size = 1, 20
            with _lock:
                items = [t for t in _tasks.values() if t["owner"] == username]
            if status_f:
                items = [t for t in items if t["status"] == status_f]
            if priority_f:
                items = [t for t in items if t["priority"] == priority_f]
            total = len(items)
            start = (page - 1) * page_size
            return self._send_json({
                "items": items[start: start + page_size],
                "total": total,
                "page": page,
                "page_size": page_size,
                "has_more": start + page_size < total,
            })

        # GET /stats
        if len(parts) == 1 and parts[0] == "stats":
            username = self._authed()
            if not username:
                return self._send_err(4010, "unauthorized", 401)
            with _lock:
                my_tasks = [t for t in _tasks.values() if t["owner"] == username]
            by_status: dict[str, int] = {}
            by_priority: dict[str, int] = {}
            for t in my_tasks:
                by_status[t["status"]] = by_status.get(t["status"], 0) + 1
                by_priority[t["priority"]] = by_priority.get(t["priority"], 0) + 1
            return self._send_json({
                "total": len(my_tasks),
                "by_status": by_status,
                "by_priority": by_priority,
            })

        self._send_err(4040, "not found", 404)

    # ── POST ───────────────────────────────────────────────────────────────

    def do_POST(self):
        parts = [p for p in urlparse(self.path).path.rstrip("/").split("/") if p]
        body = self._read_body()

        # POST /auth/verify
        if parts == ["auth", "verify"]:
            username = str(body.get("username", "")).strip()
            api_key = str(body.get("api_key", "")).strip()
            with _lock:
                stored = _api_keys.get(api_key)
                user = _users.get(username) if stored == username else None
            if user:
                return self._send_json({"valid": True, "user": user})
            return self._send_json({"valid": False, "user": None})

        # POST /auth/register  (无需认证)
        if parts == ["auth", "register"]:
            username = str(body.get("username", "")).strip()
            if not username:
                return self._send_err(1001, "username is required")
            with _lock:
                if username in _users:
                    return self._send_err(1002, "username already exists", 409)
                akey = "demo_" + _rand_str(24, string.ascii_lowercase + string.digits)
                user = {
                    "id": _new_id(),
                    "username": username,
                    "display_name": str(body.get("display_name", username)).strip() or username,
                    "email": f"{username}@example.com",
                    "plan": "free",
                    "tasks_quota": 10,
                    "created_at": _now(),
                }
                _users[username] = user
                _api_keys[akey] = username
            return self._send_json({"user": user, "api_key": akey})

        # POST /tasks
        if parts == ["tasks"]:
            username = self._authed()
            if not username:
                return self._send_err(4010, "unauthorized", 401)
            title = str(body.get("title", "")).strip()
            if not title:
                return self._send_err(1001, "title is required")
            priority = str(body.get("priority", "medium")).strip()
            if priority not in ("low", "medium", "high", "critical"):
                return self._send_err(1002, "invalid priority; must be low/medium/high/critical")
            with _lock:
                task = {
                    "id": _new_id(),
                    "title": title,
                    "status": "pending",
                    "priority": priority,
                    "owner": username,
                    "created_at": _now(),
                    "updated_at": _now(),
                }
                _tasks[task["id"]] = task
            return self._send_json(task)

        self._send_err(4040, "not found", 404)

    # ── PATCH ──────────────────────────────────────────────────────────────

    def do_PATCH(self):
        parts = [p for p in urlparse(self.path).path.rstrip("/").split("/") if p]

        # PATCH /tasks/{id}/complete
        if len(parts) == 3 and parts[0] == "tasks" and parts[2] == "complete":
            username = self._authed()
            if not username:
                return self._send_err(4010, "unauthorized", 401)
            task_id = parts[1]
            with _lock:
                task = _tasks.get(task_id)
                if not task:
                    return self._send_err(4040, "task not found", 404)
                if task["owner"] != username:
                    return self._send_err(4030, "forbidden", 403)
                task["status"] = "completed"
                task["updated_at"] = _now()
            return self._send_json(task)

        self._send_err(4040, "not found", 404)

    # ── DELETE ─────────────────────────────────────────────────────────────

    def do_DELETE(self):
        parts = [p for p in urlparse(self.path).path.rstrip("/").split("/") if p]

        # DELETE /tasks/{id}
        if len(parts) == 2 and parts[0] == "tasks":
            username = self._authed()
            if not username:
                return self._send_err(4010, "unauthorized", 401)
            task_id = parts[1]
            with _lock:
                task = _tasks.get(task_id)
                if not task:
                    return self._send_err(4040, "task not found", 404)
                if task["owner"] != username:
                    return self._send_err(4030, "forbidden", 403)
                del _tasks[task_id]
            return self._send_json({"deleted": True, "id": task_id})

        self._send_err(4040, "not found", 404)


# ---------------------------------------------------------------------------
# 入口
# ---------------------------------------------------------------------------

def main() -> None:
    _seed_data()

    parser = argparse.ArgumentParser(description="OctoDemo 假数据服务器")
    parser.add_argument("--host", default="127.0.0.1", help="监听地址 (默认: 127.0.0.1)")
    parser.add_argument("--port", type=int, default=18080, help="监听端口 (默认: 18080)")
    args = parser.parse_args()

    print(f"[fake-server] 启动于 http://{args.host}:{args.port}")
    print(f"[fake-server] 测试账号 → username=testuser  api_key=demo_testkey_12345678")
    print(f"[fake-server] 已初始化 {len(_users)} 个用户，{len(_tasks)} 条任务")
    print(f"[fake-server] 按 Ctrl+C 停止\n")

    server = HTTPServer((args.host, args.port), Handler)
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print("\n[fake-server] 已停止")


if __name__ == "__main__":
    main()
