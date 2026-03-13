from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any


@dataclass
class GatewayError(Exception):
    error_type: str
    error_message: str
    recovery_message: str
    exit_code: int = 1
    details: dict[str, Any] = field(default_factory=dict)

    def to_payload(self) -> dict[str, Any]:
        return {
            "ok": False,
            "error_type": self.error_type,
            "error_message": self.error_message,
            "recovery_message": self.recovery_message,
            **self.details,
        }


class ConfigError(GatewayError):
    def __init__(self, error_message: str, recovery_message: str, **details: Any) -> None:
        super().__init__("config_error", error_message, recovery_message, exit_code=2, details=details)


class ValidationError(GatewayError):
    def __init__(self, error_message: str, recovery_message: str, **details: Any) -> None:
        super().__init__("validation_failure", error_message, recovery_message, exit_code=3, details=details)
