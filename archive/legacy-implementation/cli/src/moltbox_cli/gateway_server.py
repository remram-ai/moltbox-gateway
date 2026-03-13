from __future__ import annotations

import json
from http import HTTPStatus
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from typing import Any
from urllib.parse import urlparse

from moltbox_commands.shared import utc_now_iso
from moltbox_commands.core.config import GatewayConfig
from moltbox_commands.core.errors import ValidationError
from moltbox_commands.core.versioning import resolve_version_info


class _GatewayRequestHandler(BaseHTTPRequestHandler):
    config: GatewayConfig

    def log_message(self, format: str, *args: object) -> None:  # noqa: A003
        return

    def _send_json(self, status_code: int, payload: dict[str, Any]) -> None:
        body = json.dumps(payload, indent=2).encode("utf-8")
        self.send_response(status_code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def do_GET(self) -> None:  # noqa: N802
        path = urlparse(self.path).path
        if path == "/health":
            version = resolve_version_info().as_dict()
            self._send_json(
                HTTPStatus.OK,
                {
                    "ok": True,
                    "status": "healthy",
                    "service": "gateway",
                    "timestamp": utc_now_iso(),
                    **version,
                },
            )
            return
        if path == "/version":
            self._send_json(HTTPStatus.OK, {"ok": True, **resolve_version_info().as_dict()})
            return
        self._send_json(
            HTTPStatus.NOT_FOUND,
            {
                "ok": False,
                "error_type": "not_found",
                "error_message": f"unsupported gateway path '{path}'",
            },
        )

    def do_POST(self) -> None:  # noqa: N802
        path = urlparse(self.path).path
        if path != "/run":
            self._send_json(
                HTTPStatus.NOT_FOUND,
                {
                    "ok": False,
                    "error_type": "not_found",
                    "error_message": f"unsupported gateway path '{path}'",
                },
            )
            return
        content_length = int(self.headers.get("Content-Length", "0") or "0")
        raw_payload = self.rfile.read(content_length) if content_length > 0 else b"{}"
        try:
            payload = json.loads(raw_payload.decode("utf-8"))
        except json.JSONDecodeError:
            self._send_json(
                HTTPStatus.BAD_REQUEST,
                ValidationError(
                    "gateway run payload must be valid JSON",
                    "send a JSON body with an argv list and retry",
                ).to_payload(),
            )
            return
        argv = payload.get("argv")
        if not isinstance(argv, list) or not all(isinstance(item, str) and item.strip() for item in argv):
            self._send_json(
                HTTPStatus.BAD_REQUEST,
                ValidationError(
                    "gateway run payload must provide argv as a list of non-empty strings",
                    "send a JSON body such as {\"argv\": [\"gateway\", \"health\"]} and retry",
                ).to_payload(),
            )
            return
        if argv[:2] == ["gateway", "serve"]:
            self._send_json(
                HTTPStatus.BAD_REQUEST,
                ValidationError(
                    "gateway serve cannot be invoked through the gateway server",
                    "call a concrete gateway or service command instead",
                ).to_payload(),
            )
            return
        from .cli import execute

        result = execute(argv)
        status_code = HTTPStatus.OK if result.get("ok", True) else HTTPStatus.BAD_REQUEST
        self._send_json(status_code, result)


def serve(config: GatewayConfig) -> int:
    handler = type("GatewayRequestHandler", (_GatewayRequestHandler,), {"config": config})
    server = ThreadingHTTPServer((config.internal_host, config.internal_port), handler)
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        pass
    finally:
        server.server_close()
    return 0
