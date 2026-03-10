from __future__ import annotations

from ..config import AppConfig
from ..deployment_service import runtime_lifecycle


def handle_runtime(config: AppConfig, env: str, action: str) -> dict[str, object]:
    return runtime_lifecycle(config, env, action)
