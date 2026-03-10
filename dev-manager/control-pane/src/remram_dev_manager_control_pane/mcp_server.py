from __future__ import annotations

import argparse

from .config import resolve_config
from .mcp_adapter import create_mcp_server


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Run the Remram MCP adapter.")
    parser.add_argument("--config-path")
    parser.add_argument("--state-root")
    parser.add_argument("--runtime-artifacts-root")
    parser.add_argument("--host", default="127.0.0.1")
    parser.add_argument("--port", type=int, default=7475)
    return parser


def main() -> None:
    args = build_parser().parse_args()
    config = resolve_config(args)
    mcp = create_mcp_server(config)
    mcp.settings.host = args.host
    mcp.settings.port = args.port
    mcp.run(transport="streamable-http")
