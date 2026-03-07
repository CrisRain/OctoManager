#!/usr/bin/env python3
import json
import sys
from datetime import datetime, timedelta, timezone


def error(code: str, message: str) -> dict:
    return {
        "status": "error",
        "error_code": code,
        "error_message": message,
    }


def success(result: dict, session: dict | None = None) -> dict:
    payload = {
        "status": "success",
        "result": result,
    }
    if session is not None:
        payload["session"] = session
    return payload


def main() -> int:
    try:
        raw = sys.stdin.read()
        request = json.loads(raw)
    except Exception:
        print(json.dumps(error("BAD_INPUT", "invalid json input")))
        return 0

    action = str(request.get("action", "")).upper()
    account = request.get("account", {})
    params = request.get("params", {})

    if not isinstance(account, dict):
        print(json.dumps(error("VALIDATION_FAILED", "account must be an object")))
        return 0

    identifier = str(account.get("identifier", "")).strip()
    if not identifier:
        print(json.dumps(error("VALIDATION_FAILED", "account.identifier is required")))
        return 0

    if action == "REGISTER":
        invite_code = str(params.get("invite_code", "")).strip()
        if not invite_code:
            print(json.dumps(error("VALIDATION_FAILED", "invite_code is required")))
            return 0
        expires_at = (datetime.now(timezone.utc) + timedelta(hours=12)).isoformat()
        print(
            json.dumps(
                success(
                    {
                        "registered": True,
                        "identifier": identifier,
                    },
                    session={
                        "type": "token",
                        "payload": {"token": f"token-{identifier}"},
                        "expires_at": expires_at,
                    },
                )
            )
        )
        return 0

    if action in {"VERIFY", "HEALTH_CHECK"}:
        print(json.dumps(success({"ok": True, "identifier": identifier})))
        return 0

    print(json.dumps(error("UNSUPPORTED_ACTION", f"unsupported action: {action}")))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
