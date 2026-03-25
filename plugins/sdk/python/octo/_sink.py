"""
_sink.py — pluggable event sink for the Octo SDK.

By default all emit_* functions write JSON to stdout (subprocess/IPC mode).
In gRPC server mode the sink is replaced per-request with a thread-safe queue,
so emitted events are yielded back to the gRPC stream instead of printed.

Usage
-----
Default (stdout):
    emit_event({"type": "log", "message": "hello"})   # → prints to stdout

gRPC per-request override:
    with grpc_sink(queue) as _:
        emit_event({"type": "log", "message": "hello"})  # → queue.put(event)
"""
from __future__ import annotations

import contextlib
import contextvars
import json
import sys
from collections.abc import Callable
from typing import Any

# The current active sink for this execution context.
# None  → use the stdout fallback.
_CURRENT_SINK: contextvars.ContextVar[
    Callable[[dict[str, Any]], None] | None
] = contextvars.ContextVar("octo_event_sink", default=None)


def emit_event(payload: dict[str, Any]) -> None:
    """Emit one event dict through the active sink (stdout or gRPC queue)."""
    sink = _CURRENT_SINK.get()
    if sink is None:
        # Default: serialize to stdout, mirroring the original IPC protocol.
        print(json.dumps(payload, ensure_ascii=False), flush=True)
    else:
        sink(payload)


@contextlib.contextmanager
def use_sink(sink: Callable[[dict[str, Any]], None]):
    """Context manager that routes emit_event() calls to *sink* for the duration."""
    token = _CURRENT_SINK.set(sink)
    try:
        yield
    finally:
        _CURRENT_SINK.reset(token)
