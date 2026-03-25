# proto/plugin/v1/plugin.proto — Python gRPC service stubs
#
# Hand-written placeholder; replace with `make proto-gen` output when
# grpcio-tools is available.  Requires:  pip install grpcio
#
# Wire format: JSON bytes (matches Go's grpccodec.jsonCodec).
# The Go worker sends/receives `application/grpc+json` — the Python server
# registers a matching "json" codec so grpcio accepts the connection.

from __future__ import annotations

import base64
import json as _json

import grpc

from . import plugin_pb2 as _pb2


# ── Codec registration ────────────────────────────────────────────────────────
# Register a "json" codec so that the Python gRPC server accepts connections
# from the Go worker which uses grpc.ForceCodec(grpccodec.Codec()).
# This is called once at module import time.

class _JsonCodec:
    """JSON codec for grpcio that matches the Go grpccodec.jsonCodec."""

    def name(self) -> str:
        return "json"

    def encode(self, message: object) -> bytes:
        # message is already a bytes-like JSON payload produced by a serialiser
        if isinstance(message, (bytes, bytearray, memoryview)):
            return bytes(message)
        return _json.dumps(message, ensure_ascii=False).encode()

    def decode(self, data: bytes, type_url: str) -> bytes:
        # Return raw bytes; the per-method deserialiser does the parsing.
        return data


try:
    # grpcio >= 1.46 exposes an experimental codec API.
    grpc.experimental.codec.register_codec(_JsonCodec())  # type: ignore[attr-defined]
except AttributeError:
    # Older grpcio versions: fall back to the per-method serialiser approach.
    # The content-type negotiation may fail if the Go client forces "json".
    # Upgrade to grpcio >= 1.54 to fix this.
    pass


# ── Serialisers / deserialisers ───────────────────────────────────────────────
# grpcio calls request_deserializer(raw_bytes) on incoming payloads and
# response_serializer(message) → bytes for outgoing payloads.
# The framing (5-byte prefix) is handled by grpcio internally.

def _ser_execute_request(req: _pb2.ExecuteRequest) -> bytes:
    return _json.dumps({
        "mode":    req.mode,
        "action":  req.action,
        "input":   _encode_bytes(req.input),
        "context": _encode_bytes(req.context),
    }, ensure_ascii=False).encode()


def _des_execute_request(data: bytes) -> _pb2.ExecuteRequest:
    d = _json.loads(data)
    return _pb2.ExecuteRequest(
        mode=d.get("mode", ""),
        action=d.get("action", ""),
        input=_decode_bytes(d.get("input", "")),
        context=_decode_bytes(d.get("context", "")),
    )


def _ser_execute_event(ev: _pb2.ExecuteEvent) -> bytes:
    return _json.dumps({
        "type":     ev.type,
        "message":  ev.message,
        "progress": ev.progress,
        "data":     _encode_bytes(ev.data),
        "error":    ev.error,
    }, ensure_ascii=False).encode()


def _des_execute_event(data: bytes) -> _pb2.ExecuteEvent:
    d = _json.loads(data)
    return _pb2.ExecuteEvent(
        type=d.get("type", ""),
        message=d.get("message", ""),
        progress=int(d.get("progress", 0)),
        data=_decode_bytes(d.get("data", "")),
        error=d.get("error", ""),
    )


def _ser_get_manifest_request(_: _pb2.GetManifestRequest) -> bytes:
    return b"{}"


def _des_get_manifest_request(_data: bytes) -> _pb2.GetManifestRequest:
    return _pb2.GetManifestRequest()


def _ser_get_manifest_response(resp: _pb2.GetManifestResponse) -> bytes:
    return _json.dumps({"manifest": _encode_bytes(resp.manifest)}, ensure_ascii=False).encode()


def _des_get_manifest_response(data: bytes) -> _pb2.GetManifestResponse:
    d = _json.loads(data)
    return _pb2.GetManifestResponse(manifest=_decode_bytes(d.get("manifest", "")))


def _ser_health_check_request(_: _pb2.HealthCheckRequest) -> bytes:
    return b"{}"


def _des_health_check_request(_data: bytes) -> _pb2.HealthCheckRequest:
    return _pb2.HealthCheckRequest()


def _ser_health_check_response(resp: _pb2.HealthCheckResponse) -> bytes:
    return _json.dumps({"healthy": resp.healthy, "version": resp.version},
                       ensure_ascii=False).encode()


def _des_health_check_response(data: bytes) -> _pb2.HealthCheckResponse:
    d = _json.loads(data)
    return _pb2.HealthCheckResponse(
        healthy=bool(d.get("healthy", False)),
        version=str(d.get("version", "")),
    )


def _encode_bytes(value: object) -> str:
    if isinstance(value, (bytes, bytearray, memoryview)):
        raw = bytes(value)
    elif value is None:
        raw = b""
    elif isinstance(value, str):
        raw = value.encode()
    else:
        raw = _json.dumps(value, ensure_ascii=False).encode()
    return base64.b64encode(raw).decode("ascii")


def _decode_bytes(value: object) -> bytes:
    if isinstance(value, (bytes, bytearray, memoryview)):
        value = bytes(value).decode()
    if not isinstance(value, str) or value == "":
        return b""
    try:
        return base64.b64decode(value, validate=True)
    except Exception:
        # Backward compatibility with older payloads that wrote raw strings.
        return value.encode()


# ── Servicer base class ───────────────────────────────────────────────────────

class PluginServiceServicer:
    """Base class for plugin gRPC servers. Subclass and override all methods."""

    def Execute(
        self,
        request: _pb2.ExecuteRequest,
        context: grpc.ServicerContext,
    ):
        """Server-streaming RPC: yield ExecuteEvent objects until done."""
        context.set_code(grpc.StatusCode.UNIMPLEMENTED)
        context.set_details("Execute not implemented")
        raise NotImplementedError

    def GetManifest(
        self,
        request: _pb2.GetManifestRequest,
        context: grpc.ServicerContext,
    ) -> _pb2.GetManifestResponse:
        context.set_code(grpc.StatusCode.UNIMPLEMENTED)
        context.set_details("GetManifest not implemented")
        raise NotImplementedError

    def HealthCheck(
        self,
        request: _pb2.HealthCheckRequest,
        context: grpc.ServicerContext,
    ) -> _pb2.HealthCheckResponse:
        context.set_code(grpc.StatusCode.UNIMPLEMENTED)
        context.set_details("HealthCheck not implemented")
        raise NotImplementedError


def add_PluginServiceServicer_to_server(
    servicer: PluginServiceServicer,
    server: grpc.Server,
) -> None:
    """Register *servicer* with a grpc.Server instance."""
    rpc_method_handlers = {
        "Execute": grpc.unary_stream_rpc_method_handler(
            servicer.Execute,
            request_deserializer=_des_execute_request,
            response_serializer=_ser_execute_event,
        ),
        "GetManifest": grpc.unary_unary_rpc_method_handler(
            servicer.GetManifest,
            request_deserializer=_des_get_manifest_request,
            response_serializer=_ser_get_manifest_response,
        ),
        "HealthCheck": grpc.unary_unary_rpc_method_handler(
            servicer.HealthCheck,
            request_deserializer=_des_health_check_request,
            response_serializer=_ser_health_check_response,
        ),
    }
    generic_handler = grpc.method_handlers_generic_handler(
        "plugin.v1.PluginService", rpc_method_handlers
    )
    server.add_generic_rpc_handlers((generic_handler,))
