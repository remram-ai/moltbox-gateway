from __future__ import annotations

from typing import Any

from moltbox_commands.core.components import ComponentSpec, resolve_component
from moltbox_commands.core.errors import ValidationError
from moltbox_runtime import config_sync, operations as runtime_operations
from moltbox_services import pipeline as service_pipeline

from .shared import success_payload


def _component_payload(spec: ComponentSpec, component_name: str) -> dict[str, Any]:
    return {
        "requested_component": component_name,
        "resolved_component": spec.canonical_name,
        "component": spec.as_dict(),
    }


def execute_component(config: Any, component_name: str, verb: str) -> dict[str, Any]:
    spec = resolve_component(component_name)
    if spec.kind == "gateway":
        raise ValidationError(
            "gateway commands must use the gateway namespace",
            "run `moltbox gateway --help` for supported commands",
            component=component_name,
        )

    command = f"moltbox {component_name} {verb}"
    if verb == "status":
        payload = service_pipeline.status_service(config, spec)
        return success_payload(command, **_component_payload(spec, component_name), **payload)
    if verb == "inspect":
        payload = service_pipeline.inspect_service(config, spec)
        return success_payload(command, **_component_payload(spec, component_name), **payload)
    if verb == "logs":
        payload = service_pipeline.logs_service(config, spec)
        return success_payload(command, **_component_payload(spec, component_name), **payload)
    if verb in {"start", "stop", "restart"}:
        payload = service_pipeline.lifecycle_service(config, spec, verb)
        return success_payload(command, **_component_payload(spec, component_name), **payload)
    if verb in {"doctor", "monitor"}:
        if spec.supports_runtime:
            payload = runtime_operations.doctor_component(config, spec)
        else:
            payload = service_pipeline.doctor_service(config, spec)
        return success_payload(command, **_component_payload(spec, component_name), **payload)
    if verb == "reload":
        if not spec.supports_runtime:
            raise ValidationError(
                f"component '{component_name}' does not support reload",
                "use a runtime component such as openclaw-dev, openclaw-test, or openclaw",
                component=component_name,
            )
        payload = runtime_operations.reload_component(config, spec)
        return success_payload(command, **_component_payload(spec, component_name), **payload)
    raise ValidationError(
        f"unsupported component command '{verb}'",
        "run `moltbox --help` for supported commands",
        component=component_name,
        command=verb,
    )


def sync_component_config(config: Any, component_name: str) -> dict[str, Any]:
    spec = resolve_component(component_name)
    if not spec.supports_runtime:
        raise ValidationError(
            f"component '{component_name}' does not support config sync",
            "use a runtime component such as openclaw-dev, openclaw-test, or openclaw",
            component=component_name,
        )
    payload = config_sync.sync_component_config(config, spec)
    return success_payload(
        f"moltbox {component_name} config sync",
        **_component_payload(spec, component_name),
        **payload,
    )
