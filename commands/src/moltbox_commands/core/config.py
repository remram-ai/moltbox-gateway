from __future__ import annotations

import os
from dataclasses import dataclass
from pathlib import Path
from typing import Any

import yaml

from .errors import ConfigError
from .layout import GatewayLayout, build_layout

DEFAULT_STATE_ROOT = Path("/srv/moltbox-state")
DEFAULT_LOGS_ROOT = Path("/srv/moltbox-logs")


def _env_value(*env_names: str) -> str | None:
    for env_name in env_names:
        raw = os.environ.get(env_name)
        if raw and raw.strip():
            return raw.strip()
    return None


def _deep_get(payload: dict[str, Any], *keys: str) -> Any:
    current: Any = payload
    for key in keys:
        if not isinstance(current, dict):
            return None
        current = current.get(key)
    return current


def _read_yaml(path: Path) -> dict[str, Any]:
    if not path.exists():
        return {}
    try:
        loaded = yaml.safe_load(path.read_text(encoding="utf-8"))
    except yaml.YAMLError as exc:
        raise ConfigError(
            f"failed to parse config file '{path}'",
            "fix the YAML syntax or remove the file and rerun the command",
            config_path=str(path),
        ) from exc
    if loaded is None:
        return {}
    if not isinstance(loaded, dict):
        raise ConfigError(
            f"config file '{path}' must contain a mapping",
            "rewrite the config file as a YAML object and rerun the command",
            config_path=str(path),
        )
    return loaded


def _resolve_path(flag_value: str | None, env_names: tuple[str, ...], config_value: Any, default: Path) -> Path:
    raw = flag_value or _env_value(*env_names)
    if raw:
        return Path(raw).expanduser().resolve()
    if isinstance(config_value, str) and config_value:
        return Path(config_value).expanduser().resolve()
    return default.expanduser().resolve()


def _resolve_string(flag_value: str | None, env_names: tuple[str, ...], config_value: Any, default: str) -> str:
    raw = flag_value or _env_value(*env_names)
    if raw:
        return raw
    if isinstance(config_value, str) and config_value:
        return config_value
    return default


def _resolve_int(flag_value: int | None, env_names: tuple[str, ...], config_value: Any, default: int) -> int:
    if flag_value is not None:
        return flag_value
    raw = _env_value(*env_names)
    if raw:
        try:
            return int(raw)
        except ValueError as exc:
            raise ConfigError(
                "configured internal port must be an integer",
                "set a numeric internal port and rerun the command",
                env_value=raw,
            ) from exc
    if config_value is None:
        return default
    try:
        return int(config_value)
    except (TypeError, ValueError) as exc:
        raise ConfigError(
            "configured internal port must be an integer",
            "set a numeric internal port in config.yaml and rerun the command",
            configured_value=config_value,
        ) from exc


@dataclass(frozen=True)
class GatewayConfig:
    config_path: Path
    state_root: Path
    runtime_artifacts_root: Path
    logs_root: Path
    services_repo_url: str | None
    runtime_repo_url: str | None
    skills_repo_url: str | None
    internal_host: str
    internal_port: int
    cli_command: list[str]
    layout: GatewayLayout


def resolve_config(args: Any | None = None) -> GatewayConfig:
    state_root_default = DEFAULT_STATE_ROOT
    config_path_default = state_root_default / "gateway" / "config.yaml"
    config_path = _resolve_path(
        getattr(args, "config_path", None),
        ("MOLTBOX_CONFIG_PATH", "REMRAM_CONFIG_PATH"),
        None,
        config_path_default,
    )
    payload = _read_yaml(config_path)

    state_root = _resolve_path(
        getattr(args, "state_root", None),
        ("MOLTBOX_STATE_ROOT", "REMRAM_STATE_ROOT"),
        _deep_get(payload, "paths", "state_root"),
        state_root_default,
    )
    logs_root_configured = (
        getattr(args, "logs_root", None)
        or _env_value("MOLTBOX_LOGS_ROOT", "REMRAM_LOGS_ROOT")
        or _deep_get(payload, "paths", "logs_root")
    )
    logs_root = _resolve_path(
        getattr(args, "logs_root", None),
        ("MOLTBOX_LOGS_ROOT", "REMRAM_LOGS_ROOT"),
        _deep_get(payload, "paths", "logs_root"),
        DEFAULT_LOGS_ROOT if not logs_root_configured and state_root == state_root_default else state_root / "logs",
    )
    runtime_artifacts_root = _resolve_path(
        getattr(args, "runtime_artifacts_root", None),
        ("MOLTBOX_RUNTIME_ROOT", "REMRAM_RUNTIME_ROOT"),
        _deep_get(payload, "paths", "runtime_root"),
        state_root / "runtime",
    )
    services_repo_url = _resolve_string(
        getattr(args, "services_repo_url", None),
        ("MOLTBOX_SERVICES_REPO_URL", "REMRAM_SERVICES_REPO_URL"),
        _deep_get(payload, "repos", "services", "url"),
        "",
    )
    runtime_repo_url = _resolve_string(
        getattr(args, "runtime_repo_url", None),
        ("MOLTBOX_RUNTIME_REPO_URL", "REMRAM_RUNTIME_REPO_URL"),
        _deep_get(payload, "repos", "runtime", "url"),
        "",
    )
    skills_repo_url = _resolve_string(
        getattr(args, "skills_repo_url", None),
        ("MOLTBOX_SKILLS_REPO_URL", "REMRAM_SKILLS_REPO_URL"),
        _deep_get(payload, "repos", "skills", "url"),
        "",
    )
    internal_host = _resolve_string(
        getattr(args, "internal_host", None),
        ("MOLTBOX_INTERNAL_HOST", "REMRAM_INTERNAL_HOST"),
        _deep_get(payload, "gateway", "host"),
        "127.0.0.1",
    )
    internal_port = _resolve_int(
        getattr(args, "internal_port", None),
        ("MOLTBOX_INTERNAL_PORT", "REMRAM_INTERNAL_PORT"),
        _deep_get(payload, "gateway", "port"),
        7474,
    )
    cli_path = _resolve_string(
        getattr(args, "cli_path", None),
        ("MOLTBOX_CLI_PATH", "REMRAM_CLI_PATH"),
        _deep_get(payload, "cli", "path"),
        "moltbox",
    )
    layout = build_layout(state_root=state_root, runtime_artifacts_root=runtime_artifacts_root, logs_root=logs_root)
    return GatewayConfig(
        config_path=config_path,
        state_root=state_root,
        runtime_artifacts_root=runtime_artifacts_root,
        logs_root=logs_root,
        services_repo_url=services_repo_url or None,
        runtime_repo_url=runtime_repo_url or None,
        skills_repo_url=skills_repo_url or None,
        internal_host=internal_host,
        internal_port=internal_port,
        cli_command=[cli_path],
        layout=layout,
    )
