import io
import json
import unittest
from unittest.mock import patch

import account_manager


class AccountManagerHelpersTest(unittest.TestCase):
    def test_require_params_rejects_missing_and_blank(self) -> None:
        ok, details = account_manager.require_params({"invite_code": "  "}, ["invite_code", "email"])

        self.assertFalse(ok)
        self.assertEqual(
            details,
            {
                "invite_code": "invite_code is required",
                "email": "email is required",
            },
        )

    def test_default_graph_config_uses_consumer_tenant_for_outlook(self) -> None:
        config = account_manager.default_graph_config("user@outlook.com", "INBOX")

        self.assertEqual(config["tenant"], "consumers")
        self.assertIn("consumers", config["token_url"])

    def test_handle_bind_email_validates_address(self) -> None:
        result = account_manager.handle_bind_email("demo", {"email": "bad-email"})

        self.assertEqual(result["status"], "error")
        self.assertEqual(result["error_code"], "VALIDATION_FAILED")

    def test_handle_batch_register_email_generates_accounts(self) -> None:
        result = account_manager.handle_batch_register_email(
            {
                "provider": "outlook",
                "domain": "example.com",
                "prefix": "mail",
                "count": 2,
                "start_index": 3,
                "status": 1,
                "options": {
                    "mailbox": "Inbox",
                    "graph_config": {"client_id": "client-123"},
                },
            }
        )

        self.assertEqual(result["status"], "success")
        generated = result["result"]["generated"]
        self.assertEqual(len(generated), 2)
        self.assertEqual(generated[0]["address"], "mail3@example.com")
        self.assertEqual(generated[0]["graph_config"]["client_id"], "client-123")
        self.assertEqual(generated[0]["graph_config"]["mailbox"], "Inbox")


class AccountManagerMainTest(unittest.TestCase):
    def invoke(self, payload: str) -> tuple[int, dict]:
        stdin = io.StringIO(payload)
        stdout = io.StringIO()
        with patch("sys.stdin", stdin), patch("sys.stdout", stdout):
            code = account_manager.main()
        return code, json.loads(stdout.getvalue())

    def test_main_handles_invalid_json(self) -> None:
        code, payload = self.invoke("{")

        self.assertEqual(code, 0)
        self.assertEqual(payload["error_code"], "BAD_INPUT")

    def test_main_handles_batch_register_email(self) -> None:
        code, payload = self.invoke(
            json.dumps(
                {
                    "action": "BATCH_REGISTER_EMAIL",
                    "params": {
                        "provider": "outlook",
                        "domain": "example.com",
                        "count": 1,
                    },
                    "context": {
                        "tenant_id": "tenant-1",
                        "request_id": "req-1",
                    },
                }
            )
        )

        self.assertEqual(code, 0)
        self.assertEqual(payload["status"], "success")
        self.assertEqual(payload["result"]["tenant_id"], "tenant-1")
        self.assertEqual(payload["result"]["request_id"], "req-1")

    def test_main_rejects_missing_identifier(self) -> None:
        code, payload = self.invoke(json.dumps({"action": "VERIFY", "account": {}}))

        self.assertEqual(code, 0)
        self.assertEqual(payload["error_code"], "VALIDATION_FAILED")


if __name__ == "__main__":
    unittest.main()
