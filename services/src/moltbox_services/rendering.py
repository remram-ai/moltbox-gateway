from __future__ import annotations

import shutil
from dataclasses import dataclass
from pathlib import Path
from typing import Any

import yaml

from moltbox_commands.core.components import ComponentSpec
from moltbox_commands.core.config import GatewayConfig
from moltbox_commands.core.errors import ValidationError
from moltbox_commands.core.jsonio import write_json
from moltbox_repos.adapters import RepoResource


@dataclass(frozen=True)
class RenderedService:
    service_name: str
    compose_project: str
    container_names: list[str]
    output_dir: Path
    compose_file: Path
    render_manifest_path: Path
    service_source: dict[str, str]
    artifact: dict[str, Any]
    metadata: dict[str, Any]


def _read_service_metadata(service_dir: Path) -> dict[str, Any]:
    for name in ("service.yaml", "service.yml"):
        path = service_dir / name
        if not path.exists():
            continue
        loaded = yaml.safe_load(path.read_text(encoding="utf-8"))
        if loaded is None:
            return {}
        if not isinstance(loaded, dict):
            raise ValidationError(
                f"service metadata '{path}' must contain a mapping",
                "rewrite the service metadata as a YAML object and rerun the command",
                service_metadata_path=str(path),
            )
        return loaded
    return {}


def _find_compose_source(service_dir: Path, metadata: dict[str, Any]) -> Path:
    configured = metadata.get("compose_file")
    if isinstance(configured, str) and configured.strip():
        path = service_dir / configured.strip()
        if path.exists():
            return path
        raise ValidationError(
            f"configured compose file '{configured}' was not found",
            "fix the compose_file setting or add the compose file to the service definition",
            compose_file=str(path),
        )
    for name in ("compose.yml.template", "compose.yaml.template", "compose.yml", "compose.yaml"):
        path = service_dir / name
        if path.exists():
            return path
    raise ValidationError(
        f"service definition '{service_dir.name}' does not contain a compose file",
        "add compose.yml.template, compose.yaml.template, compose.yml, or compose.yaml to the service directory",
        service_dir=str(service_dir),
    )


def _string_dict(raw: Any) -> dict[str, str]:
    if raw is None:
        return {}
    if not isinstance(raw, dict):
        raise ValidationError(
            "template_context must be a mapping of strings",
            "rewrite template_context in service metadata and rerun the command",
            configured_value=raw,
        )
    values: dict[str, str] = {}
    for key, value in raw.items():
        if not isinstance(key, str):
            raise ValidationError(
                "template_context keys must be strings",
                "rewrite template_context with string keys and rerun the command",
                configured_value=raw,
            )
        values[str(key)] = "" if value is None else str(value)
    return values


def _replace_tokens(text: str, context: dict[str, str]) -> str:
    rendered = text
    for key in sorted(context):
        rendered = rendered.replace(f"{{{{ {key} }}}}", context[key])
        rendered = rendered.replace(f"{{{{{key}}}}}", context[key])
    return rendered


def _copy_or_render(source: Path, destination: Path, context: dict[str, str]) -> None:
    destination.parent.mkdir(parents=True, exist_ok=True)
    if source.suffix == ".template":
        destination.with_suffix("").write_text(
            _replace_tokens(source.read_text(encoding="utf-8"), context),
            encoding="utf-8",
        )
        return
    destination.write_bytes(source.read_bytes())


def _copy_tree(service_dir: Path, output_dir: Path, context: dict[str, str]) -> list[str]:
    source_paths: list[str] = []
    for source in sorted(path for path in service_dir.rglob("*") if path.is_file()):
        if source.name in {"service.yaml", "service.yml"}:
            continue
        destination = output_dir / source.relative_to(service_dir)
        _copy_or_render(source, destination, context)
        source_paths.append(str(source))
    return source_paths


def render_service(
    config: GatewayConfig,
    spec: ComponentSpec,
    source: RepoResource,
    artifact: dict[str, Any],
) -> RenderedService:
    metadata = _read_service_metadata(source.path)
    compose_source = _find_compose_source(source.path, metadata)
    compose_project = str(metadata.get("compose_project") or spec.compose_project)
    raw_container_names = metadata.get("container_names")
    if raw_container_names is None:
        container_names = [spec.container_name]
    elif isinstance(raw_container_names, list) and raw_container_names and all(isinstance(item, str) and item.strip() for item in raw_container_names):
        container_names = [item.strip() for item in raw_container_names]
    else:
        raise ValidationError(
            "container_names must be a non-empty list of strings",
            "rewrite container_names in service metadata and rerun the command",
            configured_value=raw_container_names,
        )

    output_dir = config.layout.rendered_service_dir(spec.canonical_name)
    if output_dir.exists():
        shutil.rmtree(output_dir)
    output_dir.mkdir(parents=True, exist_ok=True)

    context = {
        "service_name": spec.canonical_name,
        "component_name": spec.canonical_name,
        "container_name": container_names[0],
        "compose_project": compose_project,
        "state_root": str(config.state_root),
        "service_state_dir": str(config.layout.service_state_dir(spec.canonical_name)),
        "runtime_root": str(config.runtime_artifacts_root),
        "runtime_component_dir": str(config.layout.runtime_component_dir(spec.canonical_name)),
        "internal_host": config.internal_host,
        "internal_port": str(config.internal_port),
        "selected_artifact": str(artifact.get("selected_artifact") or ""),
        "version": str(artifact.get("version") or ""),
        "commit": str(artifact.get("commit") or ""),
        "artifact_channel": str(artifact.get("channel") or ""),
        "artifact_strategy": str(artifact.get("strategy") or ""),
        "service_source_path": str(source.path),
        **_string_dict(metadata.get("template_context")),
    }
    source_paths = _copy_tree(source.path, output_dir, context)
    compose_relative = compose_source.relative_to(source.path)
    compose_file = output_dir / compose_relative
    if compose_file.suffix == ".template":
        compose_file = compose_file.with_suffix("")
    render_manifest_path = output_dir / "render-manifest.json"
    write_json(
        render_manifest_path,
        {
            "service": spec.canonical_name,
            "compose_project": compose_project,
            "container_names": container_names,
            "artifact": artifact,
            "source_paths": source_paths,
        },
    )
    return RenderedService(
        service_name=spec.canonical_name,
        compose_project=compose_project,
        container_names=container_names,
        output_dir=output_dir,
        compose_file=compose_file,
        render_manifest_path=render_manifest_path,
        service_source=source.as_dict(),
        artifact=artifact,
        metadata=metadata,
    )
