from __future__ import annotations

import logging
import os
from contextlib import asynccontextmanager
from datetime import UTC, datetime

from fastapi import FastAPI
from starlette.routing import Mount

from .config import AppConfig
from .health import build_local_health_payload
from .logging_setup import log_event
from .mcp_adapter import create_mcp_server
from .runtime_state import clear_pid, write_runtime_state
from .versioning import resolve_version_info


def create_http_app(config: AppConfig, logger: logging.Logger) -> FastAPI:
    version = resolve_version_info().version
    mcp_app = create_mcp_server(config).streamable_http_app()

    @asynccontextmanager
    async def lifespan(_: FastAPI):
        started_at = datetime.now(tz=UTC).isoformat()
        async with mcp_app.router.lifespan_context(mcp_app):
            write_runtime_state(
                config.layout.runtime_state_file,
                {
                    "serve_state": "ready",
                    "status": "ready",
                    "started_at": started_at,
                    "host": config.internal_host,
                    "port": config.internal_port,
                    "pid": os.getpid(),
                    "version": version,
                },
            )
            log_event(logger, logging.INFO, "ready", "control-plane", "control-plane service is ready")
            try:
                yield
            finally:
                write_runtime_state(
                    config.layout.runtime_state_file,
                    {
                        "serve_state": "stopped",
                        "status": "stopped",
                        "started_at": started_at,
                        "stopped_at": datetime.now(tz=UTC).isoformat(),
                        "host": config.internal_host,
                        "port": config.internal_port,
                        "version": version,
                    },
                )
                clear_pid(config.layout.pid_file)
                log_event(logger, logging.INFO, "shutdown", "control-plane", "control-plane service stopped")

    app = FastAPI(title="Remram Control Plane", lifespan=lifespan)

    @app.get("/health")
    async def health() -> dict:
        return build_local_health_payload(config, version)

    app.router.routes.append(Mount("/mcp", app=mcp_app))
    return app
