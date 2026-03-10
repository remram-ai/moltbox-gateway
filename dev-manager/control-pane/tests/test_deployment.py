from __future__ import annotations

import json
from pathlib import Path

from remram_dev_manager_control_pane.config import resolve_config
from remram_dev_manager_control_pane.deployment_assets import render_target
from remram_dev_manager_control_pane.deployment_service import build_target_status, deploy_target, rollback_target
from remram_dev_manager_control_pane.jsonio import write_json_file
from remram_dev_manager_control_pane.registry import get_target


class Args:
    config_path = None
    state_root = None
    runtime_artifacts_root = None
    internal_host = None
    internal_port = None
    cli_path = None


def _config(tmp_path: Path, monkeypatch) -> object:
    monkeypatch.setenv("REMRAM_STATE_ROOT", str(tmp_path / ".remram"))
    monkeypatch.setenv("REMRAM_RUNTIME_ROOT", str(tmp_path / "Moltbox"))
    return resolve_config(Args())


def test_render_assets_writes_deterministic_files(tmp_path: Path, monkeypatch) -> None:
    config = _config(tmp_path, monkeypatch)
    first = render_target(config, "ollama")
    second = render_target(config, "ollama")
    first_compose = Path(first["output_dir"]) / "compose.yml"
    second_compose = Path(second["output_dir"]) / "compose.yml"
    assert first_compose.read_text(encoding="utf-8") == second_compose.read_text(encoding="utf-8")
    manifest = json.loads((Path(first["output_dir"]) / "render-manifest.json").read_text(encoding="utf-8"))
    assert manifest["target"] == "ollama"
    assert manifest["render_outcome"] == "success"


def test_control_alias_resolves_to_control_plane(tmp_path: Path, monkeypatch) -> None:
    config = _config(tmp_path, monkeypatch)
    record = get_target(config, "control")
    assert record.id == "control-plane"


def test_status_reads_latest_records_and_log_tail(tmp_path: Path, monkeypatch) -> None:
    config = _config(tmp_path, monkeypatch)

    def fake_run_primitive(config_arg, name: str, payload: dict[str, object]) -> dict[str, object]:
        if name == "inspect_target":
            return {
                "ok": True,
                "details": {
                    "container_state": {"state": "running", "containers": []},
                    "container_ids": ["abc123"],
                },
            }
        if name == "tail_target_logs":
            return {"ok": True, "details": {"log_tail": "ready"}}
        raise AssertionError(name)

    monkeypatch.setattr("remram_dev_manager_control_pane.deployment_service.run_primitive", fake_run_primitive)
    write_json_file(
        config.layout.deploy_dir / "logs" / "dev" / "deploy-1.json",
        {
            "record_type": "deployment",
            "deployment_id": "deploy-1",
            "validator_result": {"result": "pass"},
        },
    )
    write_json_file(
        config.layout.snapshots_dir / "dev" / "snap-1" / "metadata.json",
        {"snapshot_id": "snap-1"},
    )
    payload = build_target_status(config, "dev")
    assert payload["last_deployment_id"] == "deploy-1"
    assert payload["last_snapshot_id"] == "snap-1"
    assert payload["validator_result"] == {"result": "pass"}
    assert payload["log_tail"] == "ready"


def test_deploy_writes_canonical_record(tmp_path: Path, monkeypatch) -> None:
    config = _config(tmp_path, monkeypatch)

    def fake_run_primitive(config_arg, name: str, payload: dict[str, object]) -> dict[str, object]:
        if name == "render_assets":
            return {
                "ok": True,
                "details": {
                    "output_dir": str(config.layout.deploy_dir / "rendered" / "shared" / "control-plane"),
                    "render_manifest_path": str(config.layout.deploy_dir / "rendered" / "shared" / "control-plane" / "render-manifest.json"),
                },
            }
        if name == "inspect_target":
            return {"ok": True, "details": {"container_state": {"state": "not_found"}, "container_ids": []}}
        if name == "deploy_target":
            return {"ok": True, "stdout": "", "stderr": "", "details": {"new_container_ids": ["new-1"]}}
        if name == "validate_target":
            return {"ok": True, "details": {"result": "pass"}}
        if name == "tail_target_logs":
            return {"ok": True, "details": {"log_tail": "started"}}
        raise AssertionError(name)

    monkeypatch.setattr("remram_dev_manager_control_pane.deployment_service.run_primitive", fake_run_primitive)
    payload = deploy_target(config, "control")
    assert payload["ok"] is True
    assert payload["target"] == "control-plane"
    records = sorted((config.layout.deploy_dir / "logs" / "control-plane").glob("*.json"))
    assert records
    persisted = json.loads(records[-1].read_text(encoding="utf-8"))
    assert persisted["target"] == "control-plane"


def test_rollback_uses_latest_snapshot(tmp_path: Path, monkeypatch) -> None:
    config = _config(tmp_path, monkeypatch)
    write_json_file(
        config.layout.snapshots_dir / "dev" / "snap-1" / "metadata.json",
        {"snapshot_id": "snap-1"},
    )

    def fake_run_primitive(config_arg, name: str, payload: dict[str, object]) -> dict[str, object]:
        if name == "restore_target_snapshot":
            return {"ok": True, "stdout": "", "stderr": "", "details": {"restored_container_ids": ["restored-1"]}}
        if name == "validate_target":
            return {"ok": True, "details": {"result": "pass"}}
        if name == "tail_target_logs":
            return {"ok": True, "details": {"log_tail": "restored"}}
        raise AssertionError(name)

    monkeypatch.setattr("remram_dev_manager_control_pane.deployment_service.run_primitive", fake_run_primitive)
    payload = rollback_target(config, "dev")
    assert payload["ok"] is True
    assert payload["snapshot_id"] == "snap-1"
    records = sorted((config.layout.deploy_dir / "logs" / "dev").glob("*.json"))
    assert records
