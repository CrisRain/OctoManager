# proto/plugin/v1/plugin.proto — Python message classes
#
# This file is a hand-written placeholder committed to the repo so that
# plugin code can be imported without running protoc.
#
# ⚠️  Run `make proto-gen` to replace with properly generated stubs once
#     grpcio-tools is available in your development environment:
#
#       pip install grpcio grpcio-tools
#       make proto-gen
#
# The hand-written classes are 100% compatible with the gRPC JSON codec used
# by the Go worker (internal/platform/grpccodec/json_codec.go).  Wire format:
# plain UTF-8 JSON bytes — no binary protobuf encoding required.

from __future__ import annotations


class ExecuteRequest:
    """Sent by the Go worker to the plugin's Execute RPC."""
    __slots__ = ("mode", "action", "input", "context")

    def __init__(
        self,
        mode: str = "",
        action: str = "",
        input: bytes = b"",   # JSON-encoded map[string]any
        context: bytes = b"", # JSON-encoded map[string]any
    ) -> None:
        self.mode = mode
        self.action = action
        self.input = input
        self.context = context


class ExecuteEvent:
    """Single event streamed from the plugin back to the Go worker."""
    __slots__ = ("type", "message", "progress", "data", "error")

    def __init__(
        self,
        type: str = "",
        message: str = "",
        progress: int = 0,
        data: bytes = b"",  # JSON-encoded map[string]any
        error: str = "",
    ) -> None:
        self.type = type
        self.message = message
        self.progress = progress
        self.data = data
        self.error = error


class GetManifestRequest:
    __slots__ = ()

    def __init__(self) -> None:
        pass


class GetManifestResponse:
    __slots__ = ("manifest",)

    def __init__(self, manifest: bytes = b"") -> None:
        self.manifest = manifest  # JSON-encoded AccountTypeSpec


class HealthCheckRequest:
    __slots__ = ()

    def __init__(self) -> None:
        pass


class HealthCheckResponse:
    __slots__ = ("healthy", "version")

    def __init__(self, healthy: bool = False, version: str = "") -> None:
        self.healthy = healthy
        self.version = version
