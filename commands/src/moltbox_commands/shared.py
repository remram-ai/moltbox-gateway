from __future__ import annotations

from datetime import UTC, datetime
from typing import Any


def utc_now_iso() -> str:
    return datetime.now(UTC).isoformat().replace("+00:00", "Z")


def success_payload(command: str, **fields: Any) -> dict[str, Any]:
    return {
        "ok": True,
        "status": "success",
        "command": command,
        "timestamp": utc_now_iso(),
        **fields,
    }
