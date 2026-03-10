from __future__ import annotations

import os
from pathlib import Path
from typing import Any

from .config import AppConfig
from .errors import ValidationError
from .jsonio import write_json_file
from .layout import build_repo_layout
from .operation_ids import utc_now_iso
from .registry import get_target
from .target_resolution import canonical_cli_command
from .versioning import resolve_version_info


def deployment_assets_root() -> Path:
    return build_repo_layout().containers_dir


def asset_path_for_target(asset_path: str) -> Path:
    return deployment_assets_root() / asset_path


def config_path_for_target(target_id: str, target_class: str) -> Path | None:
    config_dir = build_repo_layout().config_dir
    if target_class == "runtime":
        return config_dir / "openclaw"
    if target_id == "opensearch":
        return config_dir / "opensearch.yml"
    return None


def rendered_output_dir(config: AppConfig, target: str, profile: str | None) -> Path:
    bucket = profile if profile else "shared"
    return config.layout.deploy_dir / "rendered" / bucket / target


def _existing_owner(path: Path) -> tuple[str, str]:
    override_uid = os.environ.get("MOLTBOX_CONTAINER_UID")
    override_gid = os.environ.get("MOLTBOX_CONTAINER_GID")
    if override_uid and override_gid:
        return override_uid, override_gid
    for candidate in (path, *path.parents):
        if not candidate.exists():
            continue
        try:
            stat_result = candidate.stat()
        except OSError:
            continue
        return str(stat_result.st_uid), str(stat_result.st_gid)
    return str(getattr(os, "getuid", lambda: 1000)()), str(getattr(os, "getgid", lambda: 1000)())


def _docker_socket_gid(default_gid: str) -> str:
    override_gid = os.environ.get("MOLTBOX_DOCKER_SOCKET_GID")
    if override_gid:
        return override_gid
    socket_path = Path("/var/run/docker.sock")
    try:
        return str(socket_path.stat().st_gid)
    except OSError:
        return default_gid


def render_context(config: AppConfig, target: str) -> dict[str, str]:
    record = get_target(config, target)
    runtime_root = record.runtime_root or ""
    shared_root = str(config.layout.shared_dir / target) if record.target_class == "shared_service" else ""
    container_uid, container_gid = _existing_owner(config.state_root)
    data_volume_name = {
        "ollama": "moltbox_ollama_data",
        "opensearch": "moltbox_opensearch_data",
    }.get(record.id, "")
    gateway_port = {
        "tools": "7474",
        "dev": "18789",
        "test": "28789",
        "prod": "38789",
    }.get(record.id, "")
    return {
        "target": record.id,
        "profile": record.profile or "",
        "compose_project": record.compose_project,
        "container_name": record.container_names[0] if record.container_names else record.id,
        "runtime_root": runtime_root,
        "shared_root": shared_root,
        "data_volume_name": data_volume_name,
        "internal_network_name": "moltbox_moltbox_internal" if record.target_class == "shared_service" else "",
        "state_root": str(config.state_root),
        "runtime_artifacts_root": str(config.runtime_artifacts_root),
        "gateway_port": gateway_port,
        "container_uid": container_uid,
        "container_gid": container_gid,
        "docker_socket_gid": _docker_socket_gid(container_gid),
    }


def _replace_tokens(text: str, context: dict[str, str]) -> str:
    rendered = text
    for key in sorted(context):
        rendered = rendered.replace(f"{{{{ {key} }}}}", context[key])
        rendered = rendered.replace(f"{{{{{key}}}}}", context[key])
    return rendered


def _render_file(source: Path, destination: Path, context: dict[str, str]) -> None:
    destination.parent.mkdir(parents=True, exist_ok=True)
    if source.name.endswith(".template"):
        output_name = source.name[: -len(".template")]
        target_path = destination.parent / output_name
        text = source.read_text(encoding="utf-8")
        target_path.write_text(_replace_tokens(text, context), encoding="utf-8")
        return
    destination.write_bytes(source.read_bytes())


def _render_tree(source_root: Path, output_root: Path, context: dict[str, str]) -> list[str]:
    source_paths: list[str] = []
    for source in sorted(path for path in source_root.rglob("*") if path.is_file()):
        relative = source.relative_to(source_root)
        _render_file(source, output_root / relative, context)
        source_paths.append(str(source))
    return source_paths


def _render_config_source(source: Path, output_root: Path, context: dict[str, str]) -> tuple[list[str], Path]:
    if source.is_dir():
        rendered_root = output_root / source.name
        return _render_tree(source, rendered_root, context), rendered_root
    rendered_path = output_root / source.name
    _render_file(source, rendered_path, context)
    return [str(source)], rendered_path


def render_target(config: AppConfig, target: str, profile: str | None = None) -> dict[str, Any]:
    record = get_target(config, target)
    render_profile = profile or record.profile
    if record.profile and render_profile != record.profile:
        raise ValidationError(
            f"target '{record.id}' requires profile '{record.profile}'",
            f"rerun `{canonical_cli_command(record.id, 'deploy')}` using the required profile",
            target=record.id,
            profile=render_profile,
        )
    asset_dir = asset_path_for_target(record.asset_path)
    if not asset_dir.exists():
        raise ValidationError(
            f"deployment assets for target '{record.id}' were not found",
            "create the canonical deployment asset directory and rerun the command",
            target=record.id,
            asset_path=str(asset_dir),
        )
    config_source = config_path_for_target(record.id, record.target_class)
    if record.target_class == "runtime" and (config_source is None or not config_source.exists()):
        raise ValidationError(
            f"deployment config for target '{record.id}' was not found",
            "create the canonical runtime config directory under `moltbox/config/` and rerun the command",
            target=record.id,
            config_path=str(config_source) if config_source is not None else "",
        )
    output_dir = rendered_output_dir(config, record.id, render_profile)
    if output_dir.exists():
        for child in sorted(output_dir.rglob("*"), reverse=True):
            if child.is_file():
                child.unlink()
            elif child.is_dir():
                child.rmdir()
    output_dir.mkdir(parents=True, exist_ok=True)

    context = render_context(config, record.id)
    source_paths = _render_tree(asset_dir, output_dir, context)
    config_source_paths: list[str] = []
    rendered_config_path: Path | None = None
    if config_source is not None:
        config_source_paths, rendered_config_path = _render_config_source(config_source, output_dir / "config", context)

    manifest = {
        "target": record.id,
        "profile": render_profile,
        "render_timestamp": utc_now_iso(),
        "render_version": resolve_version_info().version,
        "render_outcome": "success",
        "source_asset_paths": source_paths,
        "source_config_paths": config_source_paths,
    }
    write_json_file(output_dir / "render-manifest.json", manifest)
    payload = {
        "target": record.id,
        "profile": render_profile,
        "output_dir": str(output_dir),
        "render_manifest_path": str(output_dir / "render-manifest.json"),
        "asset_path": str(asset_dir),
    }
    if config_source is not None and rendered_config_path is not None:
        payload["config_path"] = str(config_source)
        if config_source.is_dir():
            payload["rendered_config_dir"] = str(rendered_config_path)
        else:
            payload["rendered_config_path"] = str(rendered_config_path)
    return payload
