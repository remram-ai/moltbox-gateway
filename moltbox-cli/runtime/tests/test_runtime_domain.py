from __future__ import annotations

import sys
from pathlib import Path


sys.path.insert(0, str(Path(__file__).resolve().parents[2] / "tools" / "src"))

from moltbox_cli.config import resolve_config
from moltbox_cli.deployment_service import runtime_lifecycle


class Args:
    config_path = None
    state_root = None
    runtime_artifacts_root = None
    internal_host = None
    internal_port = None
    cli_path = None


def test_runtime_lifecycle_reports_new_cli_command(tmp_path: Path, monkeypatch) -> None:
    monkeypatch.setenv("REMRAM_STATE_ROOT", str(tmp_path / ".remram"))
    monkeypatch.setenv("REMRAM_RUNTIME_ROOT", str(tmp_path / "Moltbox"))
    config = resolve_config(Args())

    def fake_run_primitive(config_arg, name: str, payload: dict[str, object]) -> dict[str, object]:
        if name == "render_assets":
            return {
                "ok": True,
                "details": {"output_dir": str(config.layout.deploy_dir / "rendered" / "dev" / "dev")},
            }
        if name == "restart_runtime":
            return {"ok": True, "stdout": "", "stderr": "", "details": {}}
        if name == "tail_target_logs":
            return {"ok": True, "details": {"log_tail": "ready"}}
        raise AssertionError(name)

    monkeypatch.setattr("moltbox_cli.deployment_service.run_primitive", fake_run_primitive)
    payload = runtime_lifecycle(config, "dev", "restart")
    assert payload["command"] == "moltbox runtime dev restart"
    assert payload["ok"] is True
