from __future__ import annotations

from ..config import AppConfig


def handle_serve(config: AppConfig) -> None:
    from ..service import run_tools_service

    run_tools_service(config)
