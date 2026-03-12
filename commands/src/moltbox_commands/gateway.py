from __future__ import annotations

from typing import Any

from moltbox_docker import engine as docker_engine
from moltbox_services import pipeline as service_pipeline

from .service import deploy_service
from .shared import success_payload


def health_gateway(config: Any) -> dict[str, Any]:
    return success_payload(
        "moltbox gateway health",
        gateway={
            "docker_available": docker_engine.docker_available(),
            "services_repo_configured": bool(config.services_repo_url),
            "runtime_repo_configured": bool(config.runtime_repo_url),
            "skills_repo_configured": bool(config.skills_repo_url),
        },
    )


def status_gateway(config: Any) -> dict[str, Any]:
    payload = service_pipeline.status_service(config, service_pipeline.gateway_spec())
    return success_payload("moltbox gateway status", **payload)


def inspect_gateway(config: Any) -> dict[str, Any]:
    payload = service_pipeline.inspect_service(config, service_pipeline.gateway_spec())
    payload["configured_repositories"] = {
        "moltbox-services": config.services_repo_url,
        "moltbox-runtime": config.runtime_repo_url,
        "remram-skills": config.skills_repo_url,
    }
    return success_payload("moltbox gateway inspect", **payload)


def logs_gateway(config: Any) -> dict[str, Any]:
    payload = service_pipeline.logs_service(config, service_pipeline.gateway_spec())
    return success_payload("moltbox gateway logs", **payload)


def update_gateway(config: Any, *, version: str | None = None, commit: str | None = None) -> dict[str, Any]:
    payload = deploy_service(config, "gateway", version=version, commit=commit)
    payload["command"] = "moltbox gateway update"
    return payload
