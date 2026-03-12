from __future__ import annotations

import shutil
from pathlib import Path
from typing import Any

from moltbox_commands.core.components import ComponentSpec, resolve_component
from moltbox_commands.core.config import GatewayConfig
from moltbox_commands.core.errors import ValidationError
from moltbox_commands.core.jsonio import read_json, write_json
from moltbox_docker import engine as docker_engine
from moltbox_repos import adapters as repo_adapters

from .rendering import render_service


def gateway_spec() -> ComponentSpec:
    return resolve_component("gateway")


def _artifact_channel(service_name: str) -> str:
    if service_name.endswith("-dev"):
        return "dev"
    if service_name.endswith("-test"):
        return "test"
    if service_name.endswith("-prod") or service_name == "gateway":
        return "prod"
    return "shared"


def _resolve_artifact(service_name: str, *, version: str | None, commit: str | None) -> dict[str, Any]:
    if version and commit:
        raise ValidationError(
            "deployment overrides are mutually exclusive",
            "pass either --version or --commit, not both",
            service=service_name,
        )
    channel = _artifact_channel(service_name)
    if version:
        return {
            "strategy": "explicit_version",
            "channel": channel,
            "version": version,
            "commit": None,
            "selected_artifact": version,
        }
    if commit:
        return {
            "strategy": "explicit_commit",
            "channel": channel,
            "version": None,
            "commit": commit,
            "selected_artifact": commit,
        }
    selector = {
        "dev": "newest_candidate",
        "test": "latest_approved",
        "prod": "approved_stable",
    }.get(channel, "repository_default")
    return {
        "strategy": "environment_default",
        "channel": channel,
        "version": None,
        "commit": None,
        "selected_artifact": selector,
    }


def _copy_tree(source: Path, destination: Path) -> None:
    if destination.exists():
        shutil.rmtree(destination)
    shutil.copytree(source, destination)


def _safe_service_source(config: GatewayConfig, spec: ComponentSpec) -> dict[str, Any] | None:
    if not config.services_repo_url:
        return None
    try:
        return repo_adapters.service_resource(config, spec.service_name).as_dict()
    except Exception:
        return None


def _safe_runtime_source(config: GatewayConfig, spec: ComponentSpec) -> dict[str, Any] | None:
    if not config.runtime_repo_url:
        return None
    runtime_name = spec.runtime_name or spec.service_name
    if not runtime_name:
        return None
    try:
        return repo_adapters.runtime_resource(config, runtime_name).as_dict()
    except Exception:
        return None


def _state_files(config: GatewayConfig, spec: ComponentSpec) -> dict[str, Path]:
    state_dir = config.layout.service_state_dir(spec.canonical_name)
    state_dir.mkdir(parents=True, exist_ok=True)
    return {
        "state_dir": state_dir,
        "last_success": state_dir / "last-success.json",
        "previous_success": state_dir / "previous-success.json",
        "active_render": state_dir / "active-render",
        "previous_render": state_dir / "previous-render",
        "snapshots_dir": state_dir / "snapshots",
    }


def _snapshot_service(config: GatewayConfig, spec: ComponentSpec) -> dict[str, Any]:
    paths = _state_files(config, spec)
    paths["snapshots_dir"].mkdir(parents=True, exist_ok=True)
    snapshot = docker_engine.inspect_containers([spec.container_name])
    snapshot_path = paths["snapshots_dir"] / f"{spec.canonical_name}.json"
    write_json(snapshot_path, snapshot)
    return {"snapshot_path": str(snapshot_path), "snapshot": snapshot}


def list_services(config: GatewayConfig) -> list[dict[str, Any]]:
    services: list[dict[str, Any]] = []
    for resource in repo_adapters.list_services(config):
        resolved = resolve_component(resource.path.name)
        services.append(
            {
                "name": resource.path.name,
                "component": resolved.as_dict(),
                "supported_by_gateway": True,
                "service_source": resource.as_dict(),
            }
        )
    return services


def status_service(config: GatewayConfig, spec: ComponentSpec) -> dict[str, Any]:
    return {
        "service": spec.canonical_name,
        "container_state": docker_engine.inspect_containers([spec.container_name]),
    }


def inspect_service(config: GatewayConfig, spec: ComponentSpec) -> dict[str, Any]:
    payload = status_service(config, spec)
    payload["deployment_state"] = read_json(_state_files(config, spec)["last_success"], default={}) or {}
    payload["service_source"] = _safe_service_source(config, spec)
    payload["runtime_source"] = _safe_runtime_source(config, spec)
    return payload


def logs_service(config: GatewayConfig, spec: ComponentSpec) -> dict[str, Any]:
    return {
        "service": spec.canonical_name,
        "logs": docker_engine.tail_logs([spec.container_name]),
    }


def lifecycle_service(config: GatewayConfig, spec: ComponentSpec, action: str) -> dict[str, Any]:
    return {
        "service": spec.canonical_name,
        "lifecycle": docker_engine.lifecycle([spec.container_name], action),
    }


def doctor_service(config: GatewayConfig, spec: ComponentSpec) -> dict[str, Any]:
    return {
        "service": spec.canonical_name,
        "service_source": _safe_service_source(config, spec),
        "runtime_source": _safe_runtime_source(config, spec),
        "status": docker_engine.inspect_containers([spec.container_name]),
        "validation": docker_engine.validate_containers([spec.container_name], timeout_seconds=2, poll_interval_seconds=1),
    }


def deploy_service(config: GatewayConfig, spec: ComponentSpec, *, version: str | None = None, commit: str | None = None) -> dict[str, Any]:
    source = repo_adapters.service_resource(config, spec.service_name)
    artifact = _resolve_artifact(spec.canonical_name, version=version, commit=commit)
    rendered = render_service(config, spec, source, artifact)
    snapshot = _snapshot_service(config, spec)
    paths = _state_files(config, spec)

    if paths["last_success"].exists():
        write_json(paths["previous_success"], read_json(paths["last_success"], default={}) or {})
    if paths["active_render"].exists():
        _copy_tree(paths["active_render"], paths["previous_render"])

    pull_result = docker_engine.pull_stack(rendered.output_dir, rendered.compose_project)
    deploy_result = docker_engine.deploy_stack(
        render_dir=rendered.output_dir,
        compose_project=rendered.compose_project,
        container_names=rendered.container_names,
        replace_existing_containers=True,
    )
    validation_result = docker_engine.validate_containers(rendered.container_names)

    rollback_result: dict[str, Any] | None = None
    if not pull_result.get("ok") or not deploy_result.get("ok") or not validation_result.get("ok"):
        if paths["previous_render"].exists():
            previous_success = read_json(paths["previous_success"], default={}) or {}
            compose_project = str(previous_success.get("compose_project") or spec.compose_project)
            raw_container_names = previous_success.get("container_names")
            if not isinstance(raw_container_names, list) or not raw_container_names:
                raw_container_names = [spec.container_name]
            rollback_result = docker_engine.deploy_stack(
                render_dir=paths["previous_render"],
                compose_project=compose_project,
                container_names=[str(item) for item in raw_container_names],
                replace_existing_containers=True,
            )
        return {
            "ok": False,
            "status": "failure",
            "service": spec.canonical_name,
            "artifact": artifact,
            "service_source": source.as_dict(),
            "runtime_source": rendered.runtime_source,
            "render": {
                "output_dir": str(rendered.output_dir),
                "compose_file": str(rendered.compose_file),
                "render_manifest_path": str(rendered.render_manifest_path),
            },
            **snapshot,
            "pull": pull_result,
            "deploy": deploy_result,
            "validation": validation_result,
            "rollback": rollback_result,
        }

    _copy_tree(rendered.output_dir, paths["active_render"])
    write_json(
        paths["last_success"],
        {
            "service": spec.canonical_name,
            "compose_project": rendered.compose_project,
            "container_names": rendered.container_names,
            "artifact": artifact,
            "render_dir": str(paths["active_render"]),
            "render_manifest_path": str(rendered.render_manifest_path),
        },
    )
    return {
        "service": spec.canonical_name,
        "artifact": artifact,
        "service_source": source.as_dict(),
        "runtime_source": rendered.runtime_source,
        "render": {
            "output_dir": str(rendered.output_dir),
            "compose_file": str(rendered.compose_file),
            "render_manifest_path": str(rendered.render_manifest_path),
        },
        **snapshot,
        "pull": pull_result,
        "deploy": deploy_result,
        "validation": validation_result,
    }


def rollback_service(config: GatewayConfig, spec: ComponentSpec) -> dict[str, Any]:
    paths = _state_files(config, spec)
    if not paths["previous_render"].exists() or not paths["previous_success"].exists():
        raise ValidationError(
            f"no rollback data is available for service '{spec.canonical_name}'",
            "deploy the service at least twice before attempting rollback",
            service=spec.canonical_name,
        )
    previous_success = read_json(paths["previous_success"], default={}) or {}
    compose_project = str(previous_success.get("compose_project") or spec.compose_project)
    raw_container_names = previous_success.get("container_names")
    if not isinstance(raw_container_names, list) or not raw_container_names:
        raw_container_names = [spec.container_name]
    result = docker_engine.deploy_stack(
        render_dir=paths["previous_render"],
        compose_project=compose_project,
        container_names=[str(item) for item in raw_container_names],
        replace_existing_containers=True,
    )
    return {
        "service": spec.canonical_name,
        "rollback": result,
        "previous_success": previous_success,
    }
