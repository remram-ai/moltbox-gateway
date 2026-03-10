from __future__ import annotations

from pathlib import Path
from typing import Any

from .config import AppConfig
from .errors import ValidationError
from .jsonio import write_json_file
from .layout import build_repo_layout
from .operation_ids import utc_now_iso
from .registry import get_target
from .versioning import resolve_version_info


def deployment_assets_root() -> Path:
    return build_repo_layout().repo_root / "the-inner-loop" / "deployment-assets"


def asset_path_for_target(asset_path: str) -> Path:
    return deployment_assets_root() / asset_path


def rendered_output_dir(config: AppConfig, target: str, profile: str | None) -> Path:
    bucket = profile if profile else "shared"
    return config.layout.deploy_dir / "rendered" / bucket / target


def render_context(config: AppConfig, target: str) -> dict[str, str]:
    record = get_target(config, target)
    runtime_root = record.runtime_root or ""
    shared_root = str(config.layout.shared_dir / target) if record.target_class == "shared_service" else ""
    gateway_port = {
        "control-plane": "7474",
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


def render_target(config: AppConfig, target: str, profile: str | None = None) -> dict[str, Any]:
    record = get_target(config, target)
    render_profile = profile or record.profile
    if record.profile and render_profile != record.profile:
        raise ValidationError(
            f"target '{record.id}' requires profile '{record.profile}'",
            f"rerun `remram render-assets --target {record.id} --profile {record.profile}`",
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
    output_dir = rendered_output_dir(config, record.id, render_profile)
    if output_dir.exists():
        for child in sorted(output_dir.rglob("*"), reverse=True):
            if child.is_file():
                child.unlink()
            elif child.is_dir():
                child.rmdir()
    output_dir.mkdir(parents=True, exist_ok=True)

    source_paths: list[str] = []
    context = render_context(config, record.id)
    for source in sorted([path for path in asset_dir.rglob("*") if path.is_file()]):
        relative = source.relative_to(asset_dir)
        _render_file(source, output_dir / relative, context)
        source_paths.append(str(source))

    manifest = {
        "target": record.id,
        "profile": render_profile,
        "render_timestamp": utc_now_iso(),
        "render_version": resolve_version_info().version,
        "render_outcome": "success",
        "source_asset_paths": source_paths,
    }
    write_json_file(output_dir / "render-manifest.json", manifest)
    return {
        "target": record.id,
        "profile": render_profile,
        "output_dir": str(output_dir),
        "render_manifest_path": str(output_dir / "render-manifest.json"),
        "asset_path": str(asset_dir),
    }
