from __future__ import annotations

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


def config_path_for_target(target_class: str) -> Path | None:
    if target_class != "runtime":
        return None
    return build_repo_layout().config_dir / "openclaw"


def rendered_output_dir(config: AppConfig, target: str, profile: str | None) -> Path:
    bucket = profile if profile else "shared"
    return config.layout.deploy_dir / "rendered" / bucket / target


def render_context(config: AppConfig, target: str) -> dict[str, str]:
    record = get_target(config, target)
    runtime_root = record.runtime_root or ""
    shared_root = str(config.layout.shared_dir / target) if record.target_class == "shared_service" else ""
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
        "state_root": str(config.state_root),
        "runtime_artifacts_root": str(config.runtime_artifacts_root),
        "gateway_port": gateway_port,
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
    config_dir = config_path_for_target(record.target_class)
    if record.target_class == "runtime" and (config_dir is None or not config_dir.exists()):
        raise ValidationError(
            f"deployment config for target '{record.id}' was not found",
            "create the canonical runtime config directory under `moltbox/config/` and rerun the command",
            target=record.id,
            config_path=str(config_dir) if config_dir is not None else "",
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
    rendered_config_dir: Path | None = None
    if config_dir is not None:
        rendered_config_dir = output_dir / "config" / config_dir.name
        config_source_paths = _render_tree(config_dir, rendered_config_dir, context)

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
    if config_dir is not None and rendered_config_dir is not None:
        payload["config_path"] = str(config_dir)
        payload["rendered_config_dir"] = str(rendered_config_dir)
    return payload
