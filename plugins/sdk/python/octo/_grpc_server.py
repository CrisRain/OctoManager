"""
_grpc_server.py — gRPC servicer that wraps an ActionRouter.

This module bridges the existing Octo SDK (ActionRouter / emit_* functions)
into the gRPC plugin microservice model:

  - Execute RPC  → calls the router, captures events via _sink.use_sink(),
                   and yields them as ExecuteEvent stream messages.
  - GetManifest  → calls the Module's manifest() method.
  - HealthCheck  → always returns healthy=True while the process is alive.

All emit_* functions (emit_log, emit_daemon_event, …) continue to work exactly
as before — they just write to the per-request queue instead of stdout.

Usage (called by Module.serve()):

    import grpc
    from concurrent import futures
    from octo._grpc_server import PluginServicer

    servicer = PluginServicer(router, manifest_fn)
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    add_PluginServiceServicer_to_server(servicer, server)
    server.add_insecure_port("[::]:50051")
    server.start()
    server.wait_for_termination()
"""
from __future__ import annotations

import json
import queue
import threading
from typing import Any, Callable

import grpc

from ._router import ActionRouter
from ._sink import use_sink
from . import plugin_pb2 as _pb2
from .plugin_pb2_grpc import (
    PluginServiceServicer,
    add_PluginServiceServicer_to_server,
)

# Sentinel object placed on the queue by the handler thread when it finishes.
_DONE = object()

VERSION = "1.0"


class PluginServicer(PluginServiceServicer):
    """gRPC servicer that routes requests through an Octo ActionRouter."""

    def __init__(
        self,
        router: ActionRouter,
        manifest_fn: Callable[[], dict[str, Any]],
    ) -> None:
        self._router = router
        self._manifest_fn = manifest_fn

    # ── Execute ───────────────────────────────────────────────────────────────

    def Execute(
        self,
        request: _pb2.ExecuteRequest,
        context: grpc.ServicerContext,
    ):
        """
        Server-streaming RPC.

        Reconstructs the ExecutionRequest dict from the proto fields, runs the
        action handler in a background thread (so emit_* calls can yield events
        back to this generator), and streams ExecuteEvent messages until the
        handler returns or the context is cancelled.
        """
        # Decode JSON bytes from the proto fields.
        try:
            input_data = json.loads(request.input or b"{}")
        except Exception:
            input_data = {}
        try:
            ctx_data = json.loads(request.context or b"{}")
        except Exception:
            ctx_data = {}

        payload: dict[str, Any] = {
            "mode":    request.mode,
            "action":  request.action,
            "input":   input_data,
            "context": ctx_data,
        }

        # Each request gets its own event queue.
        evt_queue: queue.Queue[dict[str, Any] | object] = queue.Queue()

        def _sink(event: dict[str, Any]) -> None:
            """Called by every emit_* function inside the handler thread."""
            evt_queue.put(event)

        def _run_handler() -> None:
            """Execute the action and signal completion."""
            try:
                with use_sink(_sink):
                    result = self._router.dispatch(payload)
                # The final result dict is itself an event (type=result or type=error).
                evt_queue.put(result)
            except Exception as exc:
                evt_queue.put({"type": "error", "error": "UNEXPECTED_ERROR", "message": str(exc)})
            finally:
                evt_queue.put(_DONE)

        worker = threading.Thread(target=_run_handler, daemon=True)
        worker.start()

        # Yield events from the queue until the handler thread signals completion
        # or the gRPC context is cancelled by the caller.
        while True:
            if not context.is_active():
                break
            try:
                item = evt_queue.get(timeout=0.1)
            except queue.Empty:
                continue

            if item is _DONE:
                break

            event = item  # type: ignore[assignment]
            yield _dict_to_execute_event(event)

        worker.join(timeout=5)

    # ── GetManifest ───────────────────────────────────────────────────────────

    def GetManifest(
        self,
        request: _pb2.GetManifestRequest,
        context: grpc.ServicerContext,
    ) -> _pb2.GetManifestResponse:
        try:
            manifest = self._manifest_fn()
            raw = json.dumps(manifest, ensure_ascii=False).encode()
        except Exception as exc:
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(f"manifest generation failed: {exc}")
            return _pb2.GetManifestResponse()
        return _pb2.GetManifestResponse(manifest=raw)

    # ── HealthCheck ───────────────────────────────────────────────────────────

    def HealthCheck(
        self,
        request: _pb2.HealthCheckRequest,
        context: grpc.ServicerContext,
    ) -> _pb2.HealthCheckResponse:
        return _pb2.HealthCheckResponse(healthy=True, version=VERSION)


# ── helpers ───────────────────────────────────────────────────────────────────

def _dict_to_execute_event(event: dict[str, Any]) -> _pb2.ExecuteEvent:
    """Convert an emitted event dict to an ExecuteEvent proto message."""
    data = event.get("data")
    data_bytes = b"{}"
    if data is not None:
        try:
            data_bytes = json.dumps(data, ensure_ascii=False).encode()
        except Exception:
            data_bytes = b"{}"
    return _pb2.ExecuteEvent(
        type=str(event.get("type", "log")),
        message=str(event.get("message", "")),
        progress=int(event.get("progress", 0)),
        data=data_bytes,
        error=str(event.get("error", "")),
    )


def serve(
    servicer: PluginServicer,
    address: str = "[::]:50051",
    max_workers: int = 10,
) -> None:
    """Start a blocking gRPC server at *address*."""
    from concurrent.futures import ThreadPoolExecutor

    server = grpc.server(ThreadPoolExecutor(max_workers=max_workers))
    add_PluginServiceServicer_to_server(servicer, server)
    server.add_insecure_port(address)
    server.start()
    server.wait_for_termination()
