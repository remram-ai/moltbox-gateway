from __future__ import annotations

from dataclasses import dataclass
from typing import Any

from . import component as component_commands
from . import gateway as gateway_commands
from . import service as service_commands
from . import skill as skill_commands


@dataclass(frozen=True)
class GatewayCommand:
    verb: str
    version: str | None = None
    commit: str | None = None
    repo_name: str | None = None
    bundle_path: str | None = None


@dataclass(frozen=True)
class ServiceCommand:
    verb: str
    service_name: str | None = None
    version: str | None = None
    commit: str | None = None


@dataclass(frozen=True)
class ComponentCommand:
    component_name: str
    verb: str


@dataclass(frozen=True)
class SkillCommand:
    verb: str
    skill_name: str
    runtime_name: str | None = None


CommandRequest = GatewayCommand | ServiceCommand | ComponentCommand | SkillCommand


def dispatch(config: Any, request: CommandRequest) -> dict[str, Any]:
    if isinstance(request, GatewayCommand):
        if request.verb == "health":
            return gateway_commands.health_gateway(config)
        if request.verb == "status":
            return gateway_commands.status_gateway(config)
        if request.verb == "inspect":
            return gateway_commands.inspect_gateway(config)
        if request.verb == "logs":
            return gateway_commands.logs_gateway(config)
        if request.verb == "update":
            return gateway_commands.update_gateway(config, version=request.version, commit=request.commit)
        if request.verb == "repo_refresh":
            return gateway_commands.refresh_gateway_repo(config, request.repo_name)
        if request.verb == "repo_seed":
            return gateway_commands.seed_gateway_repo(config, str(request.repo_name), str(request.bundle_path))
        raise ValueError(f"unsupported gateway verb: {request.verb}")

    if isinstance(request, ServiceCommand):
        if request.verb == "list":
            return service_commands.list_services(config)
        if request.verb == "inspect":
            return service_commands.inspect_service(config, str(request.service_name))
        if request.verb == "status":
            return service_commands.status_service(config, str(request.service_name))
        if request.verb == "logs":
            return service_commands.logs_service(config, str(request.service_name))
        if request.verb in {"start", "stop", "restart"}:
            return service_commands.lifecycle_service(config, str(request.service_name), request.verb)
        if request.verb == "deploy":
            return service_commands.deploy_service(config, str(request.service_name), version=request.version, commit=request.commit)
        if request.verb == "rollback":
            return service_commands.rollback_service(config, str(request.service_name))
        if request.verb == "doctor":
            return service_commands.doctor_service(config, str(request.service_name))
        raise ValueError(f"unsupported service verb: {request.verb}")

    if isinstance(request, ComponentCommand):
        if request.verb == "config_sync":
            return component_commands.sync_component_config(config, request.component_name)
        return component_commands.execute_component(config, request.component_name, request.verb)

    if isinstance(request, SkillCommand):
        return skill_commands.deploy_skill(config, request.skill_name, runtime_name=request.runtime_name)

    raise TypeError(f"unsupported command request: {type(request)!r}")
