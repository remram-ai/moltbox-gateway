from __future__ import annotations

import sys
from pathlib import Path


sys.path.insert(0, str(Path(__file__).resolve().parents[2] / "tools" / "src"))

from moltbox_cli.config import resolve_config
from moltbox_cli.deployment_service import render_assets
from moltbox_cli.jsonio import read_json_file


class Args:
    config_path = None
    state_root = None
    runtime_artifacts_root = None
    internal_host = None
    internal_port = None
    cli_path = None


def test_host_render_uses_moltbox_container_assets(tmp_path: Path, monkeypatch) -> None:
    monkeypatch.setenv("MOLTBOX_STATE_ROOT", str(tmp_path / ".remram"))
    monkeypatch.setenv("MOLTBOX_RUNTIME_ROOT", str(tmp_path / "Moltbox"))
    config = resolve_config(Args())
    payload = render_assets(config, "ollama")
    assert payload["ok"] is True
    assert payload["command"] == "moltbox host ollama deploy"
    rendered = payload["render"]
    asset_path = rendered["asset_path"].replace("/", "\\")
    assert "moltbox\\containers\\shared-services\\ollama" in asset_path
    assert Path(rendered["output_dir"]) == config.layout.deploy_dir / "rendered" / "shared" / "ollama"
    manifest = read_json_file(Path(rendered["render_manifest_path"]))
    source_paths = [path.replace("/", "\\") for path in manifest["source_asset_paths"]]
    assert source_paths
    assert all("moltbox\\containers\\shared-services\\ollama" in path for path in source_paths)
