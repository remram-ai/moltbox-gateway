from __future__ import annotations

from ..config import AppConfig
from ..deployment_service import deploy_target


def handle_deploy(config: AppConfig, target: str) -> dict[str, object]:
    return deploy_target(config, target)
