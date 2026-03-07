import io
import json
import unittest
from unittest.mock import patch

import main as demo_main


class MainScriptTest(unittest.TestCase):
    def invoke(self, payload: str) -> tuple[int, dict]:
        stdin = io.StringIO(payload)
        stdout = io.StringIO()
        with patch("sys.stdin", stdin), patch("sys.stdout", stdout):
            code = demo_main.main()
        return code, json.loads(stdout.getvalue())

    def test_main_handles_invalid_json(self) -> None:
        code, payload = self.invoke("not json")

        self.assertEqual(code, 0)
        self.assertEqual(payload["error_code"], "BAD_INPUT")

    def test_register_requires_invite_code(self) -> None:
        code, payload = self.invoke(
            json.dumps(
                {
                    "action": "REGISTER",
                    "account": {"identifier": "alpha@example.com"},
                    "params": {},
                }
            )
        )

        self.assertEqual(code, 0)
        self.assertEqual(payload["error_code"], "VALIDATION_FAILED")

    def test_register_returns_session(self) -> None:
        code, payload = self.invoke(
            json.dumps(
                {
                    "action": "REGISTER",
                    "account": {"identifier": "alpha@example.com"},
                    "params": {"invite_code": "demo"},
                }
            )
        )

        self.assertEqual(code, 0)
        self.assertEqual(payload["status"], "success")
        self.assertTrue(payload["result"]["registered"])
        self.assertEqual(payload["session"]["type"], "token")

    def test_verify_returns_success(self) -> None:
        code, payload = self.invoke(
            json.dumps(
                {
                    "action": "VERIFY",
                    "account": {"identifier": "alpha@example.com"},
                }
            )
        )

        self.assertEqual(code, 0)
        self.assertEqual(payload["status"], "success")
        self.assertTrue(payload["result"]["ok"])

    def test_unsupported_action_returns_error(self) -> None:
        code, payload = self.invoke(
            json.dumps(
                {
                    "action": "DELETE",
                    "account": {"identifier": "alpha@example.com"},
                }
            )
        )

        self.assertEqual(code, 0)
        self.assertEqual(payload["error_code"], "UNSUPPORTED_ACTION")


if __name__ == "__main__":
    unittest.main()
