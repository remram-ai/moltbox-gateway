from __future__ import annotations

import json
import shutil
import subprocess
import sys
from typing import Any

from mcp.server.fastmcp import FastMCP

from .config import AppConfig
from .errors import MoltboxCliError
from .target_resolution import resolve_target_identifier, target_domain


def _cli_base_command(config: AppConfig) -> list[str]:
    if len(config.cli_command) > 1:
        return list(config.cli_command)
    cli_path = config.cli_command[0]
    resolved = shutil.which(cli_path)
    if resolved:
        return [resolved]
    return [sys.executable, "-m", "moltbox_cli"]


def _config_flags(config: AppConfig) -> list[str]:
    return [
        "--config-path",
        str(config.config_path),
        "--state-root",
        str(config.state_root),
        "--runtime-artifacts-root",
        str(config.runtime_artifacts_root),
        "--internal-host",
        config.internal_host,
        "--internal-port",
        str(config.internal_port),
    ]


def invoke_cli_json(config: AppConfig, args: list[str]) -> dict[str, Any]:
    command = _cli_base_command(config) + _config_flags(config) + args
    completed = subprocess.run(command, capture_output=True, text=True, check=False)
    stdout = completed.stdout.strip()
    payload = json.loads(stdout) if stdout else {}
    if completed.returncode != 0:
        raise MoltboxCliError(
            error_type=str(payload.get("error_type") or "cli_error"),
            error_message=str(payload.get("error_message") or "cli invocation failed"),
            recovery_message=str(payload.get("recovery_message") or "inspect stderr and rerun the command"),
            details={
                "exit_code": completed.returncode,
                "stderr": completed.stderr.strip(),
                "command": command,
            },
        )
    if not isinstance(payload, dict):
        raise MoltboxCliError(
            error_type="invalid_cli_output",
            error_message="cli returned a non-object JSON payload",
            recovery_message="inspect the MoltBox CLI implementation and rerun the command",
            details={"command": command},
        )
    return payload


def _status_args(target: str) -> list[str]:
    resolved = resolve_target_identifier(target)
    domain = target_domain(resolved)
    if domain == "tools":
        return ["tools", "status"]
    return [domain, resolved, "status"]


def create_mcp_server(config: AppConfig) -> FastMCP:
    mcp = FastMCP("MoltBox CLI", stateless_http=True, json_response=True)

    @mcp.tool(description="Return the MoltBox CLI version information.")
    async def version() -> dict:
        return invoke_cli_json(config, ["tools", "version"])

    @mcp.tool(description="Return the MoltBox CLI health model.")
    async def health() -> dict:
        return invoke_cli_json(config, ["tools", "health"])

    @mcp.tool(description="List the registered MoltBox targets.")
    async def list_targets() -> dict:
        return invoke_cli_json(config, ["tools", "inspect"])

    @mcp.tool(description="Read target status for a canonical target identifier.")
    async def status(target: str) -> dict:
        return invoke_cli_json(config, _status_args(target))

    return mcp
