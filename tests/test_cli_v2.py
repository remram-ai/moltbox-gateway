from __future__ import annotations

import json
from pathlib import Path

from moltbox_cli.cli import execute
from moltbox_commands.core.config import resolve_config
from moltbox_services import pipeline as service_pipeline

from .conftest import create_git_repo, run_cli


class Args:
    config_path = None
    state_root = None
    runtime_artifacts_root = None
    services_repo_url = None
    runtime_repo_url = None
    skills_repo_url = None
    internal_host = None
    internal_port = None
    cli_path = None


def test_version_returns_json() -> None:
    completed = run_cli("--version")
    assert completed.returncode == 0
    payload = json.loads(completed.stdout)
    assert "version" in payload


def test_help_lists_v2_namespaces() -> None:
    completed = run_cli("--help")
    assert completed.returncode == 0
    assert "usage: moltbox" in completed.stdout
    assert "service deploy <service>" in completed.stdout
    assert "openclaw-dev" in completed.stdout
    assert "skill deploy <skill>" in completed.stdout


def test_service_list_reads_external_services_repository(tmp_path: Path) -> None:
    services_repo = create_git_repo(
        tmp_path / "moltbox-services",
        {
            "services/openclaw-dev/compose.yml.template": "services:\n  openclaw-dev:\n    image: example/openclaw-dev:latest\n",
            "services/openclaw-test/compose.yml.template": "services:\n  openclaw-test:\n    image: example/openclaw-test:latest\n",
            "services/caddy/compose.yml.template": "services:\n  caddy:\n    image: caddy:latest\n",
        },
    )
    env = {
        "MOLTBOX_STATE_ROOT": str(tmp_path / ".remram"),
        "MOLTBOX_RUNTIME_ROOT": str(tmp_path / "Moltbox"),
        "MOLTBOX_SERVICES_REPO_URL": str(services_repo),
        "MOLTBOX_REPO_ROOT": str(Path(__file__).resolve().parents[1]),
    }
    completed = run_cli("service", "list", env=env)
    assert completed.returncode == 0
    payload = json.loads(completed.stdout)
    service_names = {service["name"] for service in payload["services"]}
    assert {"openclaw-dev", "openclaw-test", "caddy"} == service_names


def test_runtime_config_sync_reads_external_runtime_repository(tmp_path: Path, monkeypatch) -> None:
    runtime_repo = create_git_repo(
        tmp_path / "moltbox-runtime",
        {
            "openclaw-dev/openclaw.json": json.dumps({"gateway": {"controlUi": {"allowedOrigins": []}}}, indent=2) + "\n",
            "openclaw-dev/model-runtime.yml": "model: qwen3:8b\n",
            "openclaw-dev/openclaw/channels.yaml": "channels: []\n",
        },
    )
    monkeypatch.setenv("MOLTBOX_STATE_ROOT", str(tmp_path / ".remram"))
    monkeypatch.setenv("MOLTBOX_RUNTIME_ROOT", str(tmp_path / "Moltbox"))
    monkeypatch.setenv("MOLTBOX_RUNTIME_REPO_URL", str(runtime_repo))
    monkeypatch.setenv("MOLTBOX_PUBLIC_HOST_IP", "192.168.1.50")
    monkeypatch.setenv("MOLTBOX_REPO_ROOT", str(Path(__file__).resolve().parents[1]))
    resolve_config(Args())

    payload = execute(["openclaw-dev", "config", "sync"])

    assert payload["ok"] is True
    runtime_root = Path(str(payload["runtime_root"]))
    synced = json.loads((runtime_root / "openclaw.json").read_text(encoding="utf-8"))
    allowed_origins = synced["gateway"]["controlUi"]["allowedOrigins"]
    assert "http://127.0.0.1:18790" in allowed_origins
    assert "http://192.168.1.50:18790" in allowed_origins
    assert (runtime_root / "openclaw" / "channels.yaml").exists()


def test_skill_deploy_uses_skill_recipe_plan(tmp_path: Path, monkeypatch) -> None:
    skills_repo = create_git_repo(
        tmp_path / "remram-skills",
        {
            "skills/discord/deployment.yaml": (
                "service_deploy:\n"
                "  - caddy\n"
                "runtime_sync:\n"
                "  - openclaw-dev\n"
                "runtime_reload:\n"
                "  - openclaw-dev\n"
            ),
        },
    )
    monkeypatch.setenv("MOLTBOX_STATE_ROOT", str(tmp_path / ".remram"))
    monkeypatch.setenv("MOLTBOX_RUNTIME_ROOT", str(tmp_path / "Moltbox"))
    monkeypatch.setenv("MOLTBOX_SKILLS_REPO_URL", str(skills_repo))
    monkeypatch.setenv("MOLTBOX_REPO_ROOT", str(Path(__file__).resolve().parents[1]))

    from moltbox_commands import skill as skill_commands

    calls: list[tuple[str, str]] = []

    monkeypatch.setattr(skill_commands, "deploy_service", lambda config, name, version=None, commit=None: calls.append(("service_deploy", name)) or {"ok": True})
    monkeypatch.setattr(skill_commands, "sync_component_config", lambda config, name: calls.append(("runtime_sync", name)) or {"ok": True})
    monkeypatch.setattr(skill_commands, "execute_component", lambda config, name, verb: calls.append((verb, name)) or {"ok": True})

    payload = execute(["skill", "deploy", "discord"])

    assert payload["ok"] is True
    assert calls == [
        ("service_deploy", "caddy"),
        ("runtime_sync", "openclaw-dev"),
        ("reload", "openclaw-dev"),
    ]


def test_service_deploy_gateway_uses_clean_service_pipeline(tmp_path: Path, monkeypatch) -> None:
    services_repo = create_git_repo(
        tmp_path / "moltbox-services",
        {
            "services/gateway/compose.yml.template": (
                "services:\n"
                "  gateway:\n"
                "    image: example/gateway:{{ selected_artifact }}\n"
                "    container_name: {{ container_name }}\n"
            ),
            "services/gateway/service.yaml": (
                "compose_project: gateway\n"
                "container_names:\n"
                "  - gateway\n"
            ),
        },
    )
    monkeypatch.setenv("MOLTBOX_STATE_ROOT", str(tmp_path / ".remram"))
    monkeypatch.setenv("MOLTBOX_RUNTIME_ROOT", str(tmp_path / "Moltbox"))
    monkeypatch.setenv("MOLTBOX_SERVICES_REPO_URL", str(services_repo))
    monkeypatch.setenv("MOLTBOX_REPO_ROOT", str(Path(__file__).resolve().parents[1]))
    resolve_config(Args())

    calls: list[str] = []
    monkeypatch.setattr(service_pipeline.docker_engine, "docker_available", lambda: True)
    monkeypatch.setattr(service_pipeline.docker_engine, "inspect_containers", lambda names: {"ok": True, "details": {"container_state": {"containers": []}, "container_ids": []}})
    monkeypatch.setattr(service_pipeline.docker_engine, "pull_stack", lambda render_dir, compose_project: calls.append("pull") or {"ok": True})
    monkeypatch.setattr(
        service_pipeline.docker_engine,
        "deploy_stack",
        lambda **kwargs: calls.append("deploy") or {"ok": True, "details": {"compose_command": ["docker", "compose"]}},
    )
    monkeypatch.setattr(service_pipeline.docker_engine, "validate_containers", lambda names, timeout_seconds=30, poll_interval_seconds=2: calls.append("validate") or {"ok": True, "details": {"result": "pass"}})

    payload = execute(["service", "deploy", "gateway"])

    assert payload["ok"] is True
    assert payload["resolved_service"] == "gateway"
    assert payload["service_source"]["relative_path"] == "services/gateway"
    assert calls == ["pull", "deploy", "validate"]


def test_openclaw_alias_reload_targets_prod(monkeypatch, tmp_path: Path) -> None:
    monkeypatch.setenv("MOLTBOX_STATE_ROOT", str(tmp_path / ".remram"))
    monkeypatch.setenv("MOLTBOX_RUNTIME_ROOT", str(tmp_path / "Moltbox"))
    monkeypatch.setenv("MOLTBOX_REPO_ROOT", str(Path(__file__).resolve().parents[1]))

    from moltbox_runtime import operations as runtime_operations

    monkeypatch.setattr(runtime_operations, "reload_component", lambda config, spec: {"runtime": spec.canonical_name, "reload": {"ok": True}})

    payload = execute(["openclaw", "reload"])

    assert payload["ok"] is True
    assert payload["resolved_component"] == "openclaw-prod"


def test_gateway_update_delegates_to_gateway_service_pipeline(monkeypatch, tmp_path: Path) -> None:
    monkeypatch.setenv("MOLTBOX_STATE_ROOT", str(tmp_path / ".remram"))
    monkeypatch.setenv("MOLTBOX_RUNTIME_ROOT", str(tmp_path / "Moltbox"))
    monkeypatch.setenv("MOLTBOX_REPO_ROOT", str(Path(__file__).resolve().parents[1]))

    from moltbox_commands import gateway as gateway_commands

    monkeypatch.setattr(gateway_commands, "deploy_service", lambda config, service_name, version=None, commit=None: {"ok": True, "resolved_service": service_name, "command": "moltbox service deploy gateway"})

    payload = execute(["gateway", "update"])

    assert payload["ok"] is True
    assert payload["command"] == "moltbox gateway update"
