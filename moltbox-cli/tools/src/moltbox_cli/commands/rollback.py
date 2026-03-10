from __future__ import annotations

from ..config import AppConfig
from ..deployment_service import rollback_target


def handle_rollback(config: AppConfig, target: str) -> dict[str, object]:
    return rollback_target(config, target)
