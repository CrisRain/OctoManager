#!/usr/bin/env python3
import json
import sys
from datetime import datetime, timedelta, timezone


def utc_now() -> str:
    return datetime.now(timezone.utc).isoformat()


def error(code: str, message: str, details: dict | None = None) -> dict:
    payload = {
        "status": "error",
        "error_code": code,
        "error_message": message,
    }
    if details is not None:
        payload["result"] = {"details": details}
    return payload


def success(result: dict, session: dict | None = None) -> dict:
    payload = {
        "status": "success",
        "result": result,
    }
    if session is not None:
        payload["session"] = session
    return payload


def as_dict(value) -> dict:
    return value if isinstance(value, dict) else {}


def as_int(value, default: int) -> int:
    try:
        return int(value)
    except Exception:
        return default


def require_params(params: dict, required: list[str]) -> tuple[bool, dict]:
    details = {}
    for key in required:
        value = params.get(key)
        if value is None:
            details[key] = f"{key} is required"
            continue
        if isinstance(value, str) and value.strip() == "":
            details[key] = f"{key} is required"
    return (len(details) == 0, details)


def handle_register(identifier: str, params: dict, account_spec: dict) -> dict:
    ok, details = require_params(params, ["invite_code"])
    if not ok:
        return error("VALIDATION_FAILED", "invalid register params", details)

    invite_code = str(params["invite_code"]).strip()
    expires_at = (datetime.now(timezone.utc) + timedelta(hours=12)).isoformat()
    return success(
        {
            "event": "account.registered",
            "identifier": identifier,
            "invite_code": invite_code,
            "source": account_spec.get("source", "custom"),
            "registered_at": utc_now(),
        },
        session={
            "type": "token",
            "payload": {
                "token": f"acct-{identifier}-token",
                "scope": ["account:read", "account:write"],
            },
            "expires_at": expires_at,
        },
    )


def handle_update_profile(identifier: str, params: dict) -> dict:
    profile = as_dict(params.get("profile"))
    if len(profile) == 0:
        return error("VALIDATION_FAILED", "profile is required", {"profile": "profile must be an object"})

    return success(
        {
            "event": "account.profile_updated",
            "identifier": identifier,
            "updated_fields": list(profile.keys()),
            "updated_at": utc_now(),
        }
    )


def handle_lock(identifier: str, params: dict) -> dict:
    reason = str(params.get("reason", "")).strip() or "manual_lock"
    return success(
        {
            "event": "account.locked",
            "identifier": identifier,
            "reason": reason,
            "locked_at": utc_now(),
        }
    )


def handle_unlock(identifier: str, params: dict) -> dict:
    reason = str(params.get("reason", "")).strip() or "manual_unlock"
    return success(
        {
            "event": "account.unlocked",
            "identifier": identifier,
            "reason": reason,
            "unlocked_at": utc_now(),
        }
    )


def handle_bind_email(identifier: str, params: dict) -> dict:
    ok, details = require_params(params, ["email"])
    if not ok:
        return error("VALIDATION_FAILED", "invalid bind email params", details)

    email = str(params["email"]).strip()
    if "@" not in email:
        return error("VALIDATION_FAILED", "invalid bind email params", {"email": "email format is invalid"})

    return success(
        {
            "event": "account.email_bound",
            "identifier": identifier,
            "email": email,
            "bound_at": utc_now(),
        }
    )


def handle_health_check(identifier: str) -> dict:
    return success(
        {
            "event": "account.health_checked",
            "identifier": identifier,
            "healthy": True,
            "checked_at": utc_now(),
        }
    )


def is_outlook_consumer_address(address: str) -> bool:
    parts = address.strip().lower().split("@", 1)
    if len(parts) != 2:
        return False
    return parts[1] in ("outlook.com", "hotmail.com", "live.com", "msn.com")


def default_graph_config(username: str, mailbox: str) -> dict:
    tenant = "consumers" if is_outlook_consumer_address(username) else "common"
    return {
        "auth_method": "graph_oauth2",
        "username": username,
        "tenant": tenant,
        "scope": ["https://graph.microsoft.com/.default"],
        "token_url": f"https://login.microsoftonline.com/{tenant}/oauth2/v2.0/token",
        "graph_base_url": "https://graph.microsoft.com/v1.0",
        "mailbox": mailbox,
    }


def merge_dict(base: dict, extra: dict) -> dict:
    result = dict(base)
    for key, value in extra.items():
        result[key] = value
    return result


def handle_batch_register_email(params: dict) -> dict:
    provider = str(params.get("provider", "custom")).strip().lower() or "custom"
    domain = str(params.get("domain", "")).strip().lower()
    prefix = str(params.get("prefix", "mail")).strip().lower() or "mail"
    count = as_int(params.get("count", 0), 0)
    start_index = as_int(params.get("start_index", 1), 1)
    status = as_int(params.get("status", 0), 0)

    options = as_dict(params.get("options"))
    mailbox = str(options.get("mailbox", "INBOX")).strip() or "INBOX"
    graph_override = as_dict(options.get("graph_config"))

    details = {}
    if domain == "":
        details["domain"] = "domain is required"
    elif "." not in domain:
        details["domain"] = "domain format is invalid"

    if count <= 0:
        details["count"] = "count must be > 0"
    elif count > 200:
        details["count"] = "count must be <= 200"

    if start_index <= 0:
        details["start_index"] = "start_index must be > 0"

    if status not in (0, 1):
        details["status"] = "status must be 0 (pending) or 1 (verified)"

    if details:
        return error("VALIDATION_FAILED", "invalid batch register params", details)

    generated = []
    for i in range(count):
        index = start_index + i
        local = f"{prefix}{index}"
        address = f"{local}@{domain}"

        graph_default = default_graph_config(address, mailbox)
        graph_config = merge_dict(graph_default, graph_override)

        generated.append(
            {
                "address": address,
                "provider": provider,
                "status": status,
                "graph_config": graph_config,
            }
        )

    return success(
        {
            "event": "email.batch_registered",
            "provider": provider,
            "requested": count,
            "generated": generated,
            "generated_at": utc_now(),
        }
    )


def main() -> int:
    try:
        request = json.loads(sys.stdin.read())
    except Exception:
        print(json.dumps(error("BAD_INPUT", "invalid json input")))
        return 0

    action = str(request.get("action", "")).strip().upper()
    account = as_dict(request.get("account"))
    params = as_dict(request.get("params"))
    context = as_dict(request.get("context"))

    account_spec = as_dict(account.get("spec"))
    tenant_id = str(context.get("tenant_id", "")).strip()
    request_id = str(context.get("request_id", "")).strip()

    if action == "BATCH_REGISTER_EMAIL":
        result = handle_batch_register_email(params)
        if result.get("status") == "success":
            response_result = as_dict(result.get("result"))
            response_result["tenant_id"] = tenant_id
            response_result["request_id"] = request_id
            result["result"] = response_result
        print(json.dumps(result))
        return 0

    identifier = str(account.get("identifier", "")).strip()
    if identifier == "":
        print(json.dumps(error("VALIDATION_FAILED", "account.identifier is required")))
        return 0

    handlers = {
        "REGISTER": lambda: handle_register(identifier, params, account_spec),
        "UPDATE_PROFILE": lambda: handle_update_profile(identifier, params),
        "LOCK": lambda: handle_lock(identifier, params),
        "UNLOCK": lambda: handle_unlock(identifier, params),
        "BIND_EMAIL": lambda: handle_bind_email(identifier, params),
        "HEALTH_CHECK": lambda: handle_health_check(identifier),
        "VERIFY": lambda: handle_health_check(identifier),
    }

    if action not in handlers:
        print(json.dumps(error("UNSUPPORTED_ACTION", f"unsupported action: {action}")))
        return 0

    result = handlers[action]()
    if result.get("status") == "success":
        response_result = as_dict(result.get("result"))
        response_result["tenant_id"] = tenant_id
        response_result["request_id"] = request_id
        result["result"] = response_result

    print(json.dumps(result))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
