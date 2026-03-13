from __future__ import annotations

from ..config import AppConfig
from ..deployment_service import render_assets


def handle_render_assets(config: AppConfig, target: str, profile: str | None) -> dict[str, object]:
    return render_assets(config, target, profile)
