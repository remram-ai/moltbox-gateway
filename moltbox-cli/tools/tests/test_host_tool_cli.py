from __future__ import annotations

from pathlib import Path
import sys


sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "src"))

from moltbox_cli import host_tool_cli


def test_validate_target_waits_for_starting_health(monkeypatch) -> None:
    states = iter(
        [
            [
                {
                    "name": "moltbox-ollama",
                    "present": True,
                    "container_id": "abc",
                    "state": "running",
                    "health": "starting",
                    "image": "ollama/ollama:0.6.8",
                    "mounts": [],
                }
            ],
            [
                {
                    "name": "moltbox-ollama",
                    "present": True,
                    "container_id": "abc",
                    "state": "running",
                    "health": "healthy",
                    "image": "ollama/ollama:0.6.8",
                    "mounts": [],
                }
            ],
        ]
    )

    monkeypatch.setattr(host_tool_cli, "_docker_available", lambda: True)
    monkeypatch.setattr(host_tool_cli, "_container_details", lambda names: next(states))
    monkeypatch.setattr(host_tool_cli.time, "sleep", lambda seconds: None)

    result = host_tool_cli._validate_target(
        {
            "target": "ollama",
            "validator_key": "container_baseline",
            "container_names": ["moltbox-ollama"],
            "validation_timeout_seconds": 1,
            "validation_poll_interval_seconds": 0,
        }
    )

    assert result["ok"] is True
    assert result["details"]["result"] == "pass"
