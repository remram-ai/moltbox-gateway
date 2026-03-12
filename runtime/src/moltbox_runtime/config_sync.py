from __future__ import annotations

import json
import os
import shutil
import socket
from pathlib import Path
from typing import Any

from moltbox_commands.core.components import ComponentSpec
from moltbox_commands.core.config import GatewayConfig
from moltbox_repos.adapters import runtime_resource


def _component_gateway_port(component_name: str, default_port: int) -> int:
    return {
        "openclaw-dev": 18790,
        "openclaw-test": 28789,
        "openclaw-prod": 38789,
    }.get(component_name, default_port)


def _replace_tree(source: Path, destination: Path) -> None:
    if destination.exists():
        shutil.rmtree(destination)
    shutil.copytree(source, destination)


def _resolve_host_ip() -> str:
    configured = os.environ.get("MOLTBOX_PUBLIC_HOST_IP") or os.environ.get("REMRAM_PUBLIC_HOST_IP")
    if configured and configured.strip():
        return configured.strip()
    try:
        with socket.socket(socket.AF_INET, socket.SOCK_DGRAM) as sock:
            sock.connect(("1.1.1.1", 80))
            return sock.getsockname()[0]
    except OSError:
        return "127.0.0.1"


def _seed_openclaw_config(runtime_root: Path, port: int) -> None:
    config_path = runtime_root / "openclaw.json"
    if not config_path.exists():
        return
    payload = json.loads(config_path.read_text(encoding="utf-8"))
    if not isinstance(payload, dict):
        payload = {}
    gateway = payload.setdefault("gateway", {})
    if not isinstance(gateway, dict):
        gateway = {}
        payload["gateway"] = gateway
    gateway["mode"] = "local"
    control_ui = gateway.setdefault("controlUi", {})
    if not isinstance(control_ui, dict):
        control_ui = {}
        gateway["controlUi"] = control_ui

    merged: list[str] = []
    existing = control_ui.get("allowedOrigins")
    if isinstance(existing, list):
        for item in existing:
            if isinstance(item, str) and item and item not in merged:
                merged.append(item)

    for host in ("127.0.0.1", "localhost", _resolve_host_ip()):
        for scheme in ("http", "https"):
            for suffix in ("", f":{port}"):
                value = f"{scheme}://{host}{suffix}"
                if value not in merged:
                    merged.append(value)
    control_ui["allowedOrigins"] = merged
    config_path.write_text(json.dumps(payload, indent=2) + "\n", encoding="utf-8")


def sync_component_config(config: GatewayConfig, spec: ComponentSpec) -> dict[str, Any]:
    source = runtime_resource(config, spec.service_name)
    staging_dir = config.layout.deploy_root / "runtime-sync" / spec.canonical_name
    destination = config.layout.runtime_component_dir(spec.canonical_name)
    if staging_dir.exists():
        shutil.rmtree(staging_dir)
    shutil.copytree(source.path, staging_dir)
    _seed_openclaw_config(staging_dir, _component_gateway_port(spec.canonical_name, config.internal_port))
    _replace_tree(staging_dir, destination)
    return {
        "runtime_source": source.as_dict(),
        "staging_dir": str(staging_dir),
        "runtime_root": str(destination),
    }
