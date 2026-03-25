from __future__ import annotations

from typing import Any


def success(result: dict[str, Any] | None = None, session: dict[str, Any] | None = None) -> dict[str, Any]:
    data: dict[str, Any] = dict(result or {})
    if session is not None:
        data["session"] = session
    return {
        "type": "result",
        "data": data,
    }


def error(code: str, message: str, details: dict[str, Any] | None = None) -> dict[str, Any]:
    payload: dict[str, Any] = {
        "type": "error",
        "error": str(code).strip() or "UNEXPECTED_ERROR",
        "message": str(message).strip() or "unexpected error",
    }
    if details:
        payload["data"] = {"details": details}
    return payload
