from __future__ import annotations

from ..config import AppConfig
from ..deployment_service import runtime_chat, runtime_lifecycle


def handle_runtime(config: AppConfig, env: str, action: str) -> dict[str, object]:
    return runtime_lifecycle(config, env, action)


def handle_runtime_chat(
    config: AppConfig,
    env: str,
    message: str | None,
    timeout_seconds: int,
) -> dict[str, object]:
    return runtime_chat(config, env, message, timeout_seconds)
