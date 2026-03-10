from __future__ import annotations

from datetime import UTC, datetime
from pathlib import Path

from .config import AppConfig
from .jsonio import display_path, read_json_file
from .layout import ensure_host_layout
from .log_paths import service_log_dir, service_log_file
from .models import TargetRecord
from .registry_store import load_target_record, target_file_path, write_target_record


def _iso_now() -> str:
    return datetime.now(tz=UTC).isoformat()


def _base_metadata(config: AppConfig, service_name: str, aliases: list[str] | None = None) -> dict[str, object]:
    log_dir = service_log_dir(config, service_name)
    return {
        "aliases": aliases or [],
        "log_dir": display_path(log_dir),
        "primary_log": display_path(service_log_file(config, service_name)),
        "mount_path": "/var/log/remram",
    }


def _control_plane_target(config: AppConfig) -> TargetRecord:
    now = _iso_now()
    return TargetRecord(
        id="control-plane",
        target_class="control_plane",
        display_name="MoltBox Tools",
        asset_path="tools",
        compose_project="moltbox-control-plane",
        container_names=["control-plane"],
        snapshot_scope="target",
        validator_key="container_baseline",
        log_source="docker_logs",
        runtime_root=str(config.layout.control_plane_dir),
        service_name="control-plane",
        container_name="control-plane",
        created_at=now,
        updated_at=now,
        metadata={
            **_base_metadata(config, "control-plane", aliases=["cli", "control", "tools"]),
            "hostname": "moltbox-cli",
        },
    )


def _runtime_target(config: AppConfig, target_id: str, display_name: str, hostname: str) -> TargetRecord:
    now = _iso_now()
    runtime_root = Path.home() / ".openclaw" / target_id
    service_name = f"openclaw-{target_id}"
    return TargetRecord(
        id=target_id,
        target_class="runtime",
        display_name=display_name,
        asset_path="runtimes/openclaw",
        compose_project=f"remram-{target_id}",
        container_names=[service_name],
        snapshot_scope="target",
        validator_key="container_baseline",
        log_source="docker_logs",
        profile=target_id,
        runtime_root=str(runtime_root.expanduser()),
        service_name=service_name,
        container_name=service_name,
        created_at=now,
        updated_at=now,
        metadata={
            **_base_metadata(config, service_name, aliases=["prime"] if target_id == "prod" else []),
            "hostname": hostname,
        },
    )


def _shared_service_target(config: AppConfig, target_id: str, display_name: str) -> TargetRecord:
    now = _iso_now()
    shared_root = config.layout.shared_dir / target_id
    return TargetRecord(
        id=target_id,
        target_class="shared_service",
        display_name=display_name,
        asset_path=f"shared-services/{target_id}",
        compose_project=f"remram-{target_id}",
        container_names=[target_id],
        snapshot_scope="target",
        validator_key="container_baseline",
        log_source="docker_logs",
        service_name=target_id,
        container_name=target_id,
        created_at=now,
        updated_at=now,
        metadata={
            **_base_metadata(config, target_id),
            "shared_root": display_path(shared_root),
        },
    )


def canonical_target_records(config: AppConfig) -> list[TargetRecord]:
    return [
        _control_plane_target(config),
        _shared_service_target(config, "ollama", "Ollama"),
        _shared_service_target(config, "opensearch", "OpenSearch"),
        _shared_service_target(config, "caddy", "Caddy"),
        _runtime_target(config, "dev", "Development Runtime", "moltbox-dev"),
        _runtime_target(config, "test", "Test Runtime", "moltbox-test"),
        _runtime_target(config, "prod", "Production Runtime", "moltbox-prod"),
    ]


def _migrate_legacy_control_record(config: AppConfig) -> None:
    legacy_path = target_file_path(config.layout, "control")
    canonical_path = target_file_path(config.layout, "control-plane")
    if not legacy_path.exists() or canonical_path.exists():
        return
    legacy_payload = read_json_file(legacy_path, default={}) or {}
    legacy_display_name = str(legacy_payload.get("display_name") or "Control Plane")
    legacy_runtime_root = legacy_payload.get("runtime_root")
    legacy_created_at = str(legacy_payload.get("created_at") or _iso_now())
    legacy_metadata = legacy_payload.get("metadata") if isinstance(legacy_payload.get("metadata"), dict) else {}
    migrated = TargetRecord(
        id="control-plane",
        target_class="control_plane",
        display_name=legacy_display_name,
        asset_path="control-plane",
        compose_project="moltbox-control-plane",
        container_names=["control-plane"],
        snapshot_scope="target",
        validator_key="container_baseline",
        log_source="docker_logs",
        runtime_root=str(legacy_runtime_root or config.layout.control_plane_dir),
        service_name="control-plane",
        container_name="control-plane",
        created_at=legacy_created_at,
        updated_at=_iso_now(),
        metadata={**legacy_metadata, "aliases": ["cli", "control"], "hostname": "moltbox-cli"},
    )
    write_target_record(canonical_path, migrated)
    legacy_path.unlink(missing_ok=True)


def ensure_registry_bootstrap(config: AppConfig) -> list[TargetRecord]:
    ensure_host_layout(config.layout)
    _migrate_legacy_control_record(config)
    records: list[TargetRecord] = []
    for canonical in canonical_target_records(config):
        path = target_file_path(config.layout, canonical.id)
        if path.exists():
            records.append(load_target_record(path))
            continue
        write_target_record(path, canonical)
        records.append(canonical)
    return records
