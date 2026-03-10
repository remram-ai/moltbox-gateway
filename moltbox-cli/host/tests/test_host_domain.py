from __future__ import annotations

import sys
from pathlib import Path


sys.path.insert(0, str(Path(__file__).resolve().parents[2] / "tools" / "src"))

from moltbox_cli.config import resolve_config
from moltbox_cli.deployment_assets import render_target


class Args:
    config_path = None
    state_root = None
    runtime_artifacts_root = None
    internal_host = None
    internal_port = None
    cli_path = None


def test_host_render_uses_moltbox_container_assets(tmp_path: Path, monkeypatch) -> None:
    monkeypatch.setenv("REMRAM_STATE_ROOT", str(tmp_path / ".remram"))
    monkeypatch.setenv("REMRAM_RUNTIME_ROOT", str(tmp_path / "Moltbox"))
    config = resolve_config(Args())
    rendered = render_target(config, "ollama")
    asset_path = rendered["asset_path"].replace("/", "\\")
    assert "moltbox\\containers\\shared-services\\ollama" in asset_path
