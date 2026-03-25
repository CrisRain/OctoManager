from __future__ import annotations

import json
import sys
from dataclasses import asdict, dataclass, field
from pathlib import Path
from typing import Any

from ._router import make_router
from ._io import run_module


@dataclass
class Setting:
    key: str
    label: str
    type: str = "string"
    secret: bool = False
    default: Any = None
    description: str = ""
    required: bool = False


@dataclass
class ParamSpec:
    name: str
    type: str = "string"
    default: Any = None
    required: bool = False
    choices: list[str] | None = None
    description: str = ""


@dataclass
class UIField:
    key: str
    label: str
    type: str = "string"


@dataclass
class UIButton:
    action: str
    label: str
    variant: str = "outline"
    mode: str = "sync"
    params: dict[str, Any] = field(default_factory=dict)
    form: list[ParamSpec] = field(default_factory=list)


@dataclass
class UISection:
    title: str = ""
    fields: list[UIField] = field(default_factory=list)
    buttons: list[UIButton] = field(default_factory=list)


@dataclass
class UITab:
    key: str
    label: str
    sections: list[UISection] = field(default_factory=list)
    # context hints where this tab should appear:
    #   ""         — default, shown in account detail
    #   "create"   — only relevant when creating / registering a new account
    #   "list"     — shown in account list view
    context: str = ""


@dataclass
class _ActionDefinition:
    key: str
    handler: Any
    description: str = ""
    params: list[ParamSpec] = field(default_factory=list)


class Module:
    def __init__(
        self,
        *,
        key: str,
        name: str,
        category: str = "generic",
        account_schema: dict[str, Any] | None = None,
        settings: list[Setting] | None = None,
        script_config: dict[str, Any] | None = None,
    ) -> None:
        self.key = str(key).strip()
        self.name = str(name).strip()
        self.category = str(category).strip() or "generic"
        self.account_schema = account_schema or {}
        self.settings = list(settings or [])
        self.script_config = script_config or {}
        self.tabs: list[UITab] = []
        self.list_actions: list[UIButton] = []
        self._actions: dict[str, _ActionDefinition] = {}

    def action(
        self,
        action: str,
        *,
        description: str = "",
        params: list[ParamSpec] | None = None,
    ):
        normalized = str(action).strip().upper()
        if not normalized:
            raise ValueError("action is required")

        def decorator(handler):
            self._actions[normalized] = _ActionDefinition(
                key=normalized,
                handler=handler,
                description=str(description).strip(),
                params=list(params or []),
            )
            return handler

        return decorator

    def set_ui(self, tabs: list[UITab], *, list_actions: list[UIButton] | None = None) -> "Module":
        self.tabs = list(tabs)
        self.list_actions = list(list_actions or [])
        return self

    def run(self) -> int:
        """Start in subprocess / stdin-stdout IPC mode (default)."""
        self._write_manifest_if_needed()
        router = make_router()
        for action in self._actions.values():
            router.add(action.key, action.handler)
        return run_module(router.dispatch)

    def serve(self, address: str = "[::]:50051", *, max_workers: int = 10) -> int:
        """Start as a persistent gRPC microservice.

        This replaces ``module.run()`` when the plugin is launched with
        ``--grpc``.  The plugin listens on *address* and handles concurrent
        requests from the Go worker via the PluginService gRPC API.

        Requires:  pip install grpcio

        Args:
            address:     gRPC listen address, e.g. ``"[::]:50051"`` or
                         ``"127.0.0.1:50051"``.
            max_workers: Size of the thread pool used to handle concurrent RPCs.

        Returns:
            Exit code (0 = clean shutdown after SIGINT/SIGTERM).
        """
        from ._grpc_server import PluginServicer, serve as _serve

        self._write_manifest_if_needed()
        router = make_router()
        for action in self._actions.values():
            router.add(action.key, action.handler)

        servicer = PluginServicer(router, self.manifest)
        _serve(servicer, address=address, max_workers=max_workers)
        return 0

    def manifest(self) -> dict[str, Any]:
        self._validate()
        return {
            "key": self.key,
            "name": self.name,
            "category": self.category,
            "schema": self.account_schema,
            "capabilities": {
                "actions": [self._action_manifest(item) for item in self._actions.values()],
            },
            "settings": [self._clean_dict(asdict(item)) for item in self.settings],
            "ui": self._ui_manifest(),
            "script_config": self.script_config,
        }

    def _write_manifest_if_needed(self) -> None:
        manifest_path = self._manifest_path()
        payload = json.dumps(self.manifest(), ensure_ascii=False, indent=2) + "\n"
        if manifest_path.exists():
            current = manifest_path.read_text(encoding="utf-8")
            if current == payload:
                return
        manifest_path.write_text(payload, encoding="utf-8")

    def _manifest_path(self) -> Path:
        main_module = sys.modules.get("__main__")
        main_file = getattr(main_module, "__file__", "") or sys.argv[0]
        if not main_file:
            raise RuntimeError("cannot resolve module entry file")
        module_dir = Path(main_file).resolve().parent
        return module_dir / f"account_type.{self.key}.json"

    def _ui_manifest(self) -> dict[str, Any]:
        return {
            "tabs": [self._clean_dict(asdict(item)) for item in self.tabs],
            "list_actions": [self._clean_dict(asdict(item)) for item in self.list_actions],
        }

    def _action_manifest(self, action: _ActionDefinition) -> dict[str, Any]:
        payload: dict[str, Any] = {"key": action.key}
        if action.description:
            payload["description"] = action.description
        if action.params:
            payload["params"] = [self._clean_dict(asdict(item)) for item in action.params]
        return payload

    def _validate(self) -> None:
        if not self.key:
            raise ValueError("module key is required")
        if not self.name:
            raise ValueError("module name is required")
        if not self._actions:
            raise ValueError("module must declare at least one action")
        action_keys = set(self._actions.keys())
        self._validate_settings()
        self._validate_buttons(self.list_actions, action_keys, "ui.list_actions")
        seen_tabs: set[str] = set()
        for tab in self.tabs:
            tab_key = str(tab.key).strip()
            if not tab_key:
                raise ValueError("ui tab key is required")
            if tab_key in seen_tabs:
                raise ValueError(f"duplicate ui tab key: {tab_key}")
            seen_tabs.add(tab_key)
            for index, section in enumerate(tab.sections):
                self._validate_buttons(section.buttons, action_keys, f"ui.tabs[{tab_key}].sections[{index}]")

    def _validate_settings(self) -> None:
        seen: set[str] = set()
        for item in self.settings:
            key = str(item.key).strip()
            if not key:
                raise ValueError("setting key is required")
            if key in seen:
                raise ValueError(f"duplicate setting key: {key}")
            seen.add(key)

    def _validate_buttons(self, buttons: list[UIButton], action_keys: set[str], scope: str) -> None:
        for button in buttons:
            action = str(button.action).strip().upper()
            if not action:
                raise ValueError(f"{scope}: button action is required")
            if action not in action_keys:
                raise ValueError(f"{scope}: unknown button action {action}")
            if button.mode not in ("sync", "job"):
                raise ValueError(f"{scope}: unsupported button mode {button.mode}")

    def _clean_dict(self, value: Any) -> Any:
        if isinstance(value, list):
            return [self._clean_dict(item) for item in value]
        if isinstance(value, dict):
            cleaned: dict[str, Any] = {}
            for key, item in value.items():
                if item is None:
                    continue
                cleaned[str(key)] = self._clean_dict(item)
            return cleaned
        return value
