from __future__ import annotations

from typing import Any

from moltbox_commands.core.components import resolve_component
from moltbox_services import pipeline as service_pipeline

from .shared import success_payload


def list_services(config: Any) -> dict[str, Any]:
    services = service_pipeline.list_services(config)
    return success_payload("moltbox service list", services=services)


def inspect_service(config: Any, service_name: str) -> dict[str, Any]:
    spec = resolve_component(service_name)
    return success_payload(
        f"moltbox service inspect {service_name}",
        requested_service=service_name,
        resolved_service=spec.canonical_name,
        component=spec.as_dict(),
        **service_pipeline.inspect_service(config, spec),
    )


def status_service(config: Any, service_name: str) -> dict[str, Any]:
    spec = resolve_component(service_name)
    return success_payload(
        f"moltbox service status {service_name}",
        requested_service=service_name,
        resolved_service=spec.canonical_name,
        component=spec.as_dict(),
        **service_pipeline.status_service(config, spec),
    )


def logs_service(config: Any, service_name: str) -> dict[str, Any]:
    spec = resolve_component(service_name)
    return success_payload(
        f"moltbox service logs {service_name}",
        requested_service=service_name,
        resolved_service=spec.canonical_name,
        component=spec.as_dict(),
        **service_pipeline.logs_service(config, spec),
    )


def lifecycle_service(config: Any, service_name: str, action: str) -> dict[str, Any]:
    spec = resolve_component(service_name)
    return success_payload(
        f"moltbox service {action} {service_name}",
        requested_service=service_name,
        resolved_service=spec.canonical_name,
        component=spec.as_dict(),
        **service_pipeline.lifecycle_service(config, spec, action),
    )


def deploy_service(config: Any, service_name: str, *, version: str | None = None, commit: str | None = None) -> dict[str, Any]:
    spec = resolve_component(service_name)
    payload = service_pipeline.deploy_service(config, spec, version=version, commit=commit)
    command = f"moltbox service deploy {service_name}"
    if not payload.get("ok", True):
        payload["command"] = command
        payload["requested_service"] = service_name
        payload["resolved_service"] = spec.canonical_name
        payload["component"] = spec.as_dict()
        return payload
    return success_payload(
        command,
        requested_service=service_name,
        resolved_service=spec.canonical_name,
        component=spec.as_dict(),
        **payload,
    )


def rollback_service(config: Any, service_name: str) -> dict[str, Any]:
    spec = resolve_component(service_name)
    return success_payload(
        f"moltbox service rollback {service_name}",
        requested_service=service_name,
        resolved_service=spec.canonical_name,
        component=spec.as_dict(),
        **service_pipeline.rollback_service(config, spec),
    )


def doctor_service(config: Any, service_name: str) -> dict[str, Any]:
    spec = resolve_component(service_name)
    return success_payload(
        f"moltbox service doctor {service_name}",
        requested_service=service_name,
        resolved_service=spec.canonical_name,
        component=spec.as_dict(),
        **service_pipeline.doctor_service(config, spec),
    )
