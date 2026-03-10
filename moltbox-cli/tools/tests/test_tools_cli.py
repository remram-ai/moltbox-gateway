from __future__ import annotations

import json
import os
import subprocess
import sys
from pathlib import Path


CLI_ROOT = Path(__file__).resolve().parents[2]
SRC_DIR = CLI_ROOT / "tools" / "src"


def run_cli(*args: str, env: dict[str, str] | None = None) -> subprocess.CompletedProcess[str]:
    merged_env = os.environ.copy()
    merged_env["PYTHONPATH"] = str(SRC_DIR) + os.pathsep + merged_env.get("PYTHONPATH", "")
    if env:
        merged_env.update(env)
    return subprocess.run(
        [sys.executable, "-m", "moltbox_cli", *args],
        cwd=str(CLI_ROOT),
        capture_output=True,
        text=True,
        check=False,
        env=merged_env,
    )


def test_tools_version_returns_json() -> None:
    completed = run_cli("tools", "version")
    assert completed.returncode == 0
    payload = json.loads(completed.stdout)
    assert "version" in payload


def test_tools_inspect_lists_targets(tmp_path: Path) -> None:
    env = {
        "REMRAM_STATE_ROOT": str(tmp_path / ".remram"),
        "REMRAM_RUNTIME_ROOT": str(tmp_path / "Moltbox"),
    }
    completed = run_cli("tools", "inspect", env=env)
    assert completed.returncode == 0
    payload = json.loads(completed.stdout)
    target_ids = {target["id"] for target in payload["targets"]}
    assert {"control-plane", "ollama", "dev"}.issubset(target_ids)
