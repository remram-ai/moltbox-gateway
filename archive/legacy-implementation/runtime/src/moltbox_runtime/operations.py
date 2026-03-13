from __future__ import annotations

from typing import Any

from moltbox_commands.core.components import ComponentSpec
from moltbox_commands.core.config import GatewayConfig
from moltbox_docker import engine as docker_engine
from moltbox_repos.adapters import runtime_resource


def reload_component(config: GatewayConfig, spec: ComponentSpec) -> dict[str, Any]:
    return {
        "runtime": spec.canonical_name,
        "reload": docker_engine.lifecycle([spec.container_name], "restart"),
    }


def doctor_component(config: GatewayConfig, spec: ComponentSpec) -> dict[str, Any]:
    runtime_source: dict[str, Any] | None = None
    if config.runtime_repo_url:
        try:
            runtime_source = runtime_resource(config, spec.service_name).as_dict()
        except Exception:
            runtime_source = None
    return {
        "runtime": spec.canonical_name,
        "runtime_source": runtime_source,
        "container_state": docker_engine.inspect_containers([spec.container_name]),
        "validation": docker_engine.validate_containers([spec.container_name], timeout_seconds=2, poll_interval_seconds=1),
    }
