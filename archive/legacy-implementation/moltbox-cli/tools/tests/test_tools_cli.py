from __future__ import annotations

import json
import os
import subprocess
import sys
from pathlib import Path


CLI_ROOT = Path(__file__).resolve().parents[2]
SRC_DIR = CLI_ROOT / "tools" / "src"

sys.path.insert(0, str(SRC_DIR))

from moltbox_cli.layout import find_repo_root
from moltbox_cli.v2_actions import component_config_sync_action, skill_deploy_action


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


def _git(path: Path, *args: str) -> subprocess.CompletedProcess[str]:
    return subprocess.run(["git", "-C", str(path), *args], capture_output=True, text=True, check=False)


def create_git_repo(path: Path, files: dict[str, str]) -> Path:
    path.mkdir(parents=True, exist_ok=True)
    init = subprocess.run(["git", "init", str(path)], capture_output=True, text=True, check=False)
    assert init.returncode == 0, init.stderr
    for relative, content in files.items():
        file_path = path / relative
        file_path.parent.mkdir(parents=True, exist_ok=True)
        file_path.write_text(content, encoding="utf-8")
    add = _git(path, "add", ".")
    assert add.returncode == 0, add.stderr
    commit = subprocess.run(
        [
            "git",
            "-C",
            str(path),
            "-c",
            "user.email=test@example.com",
            "-c",
            "user.name=Test User",
            "commit",
            "-m",
            "init",
        ],
        capture_output=True,
        text=True,
        check=False,
    )
    assert commit.returncode == 0, commit.stderr
    return path


class Args:
    config_path = None
    policy_path = None
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
    assert "moltbox gateway" in completed.stdout
    assert "moltbox service" in completed.stdout
    assert "moltbox skill deploy <skill>" in completed.stdout
    assert "moltbox openclaw-dev" in completed.stdout
    assert "render-assets" not in completed.stdout
    assert "list-targets" not in completed.stdout


def test_gateway_inspect_lists_targets(tmp_path: Path) -> None:
    env = {
        "MOLTBOX_STATE_ROOT": str(tmp_path / ".remram"),
        "MOLTBOX_RUNTIME_ROOT": str(tmp_path / "Moltbox"),
    }
    completed = run_cli("gateway", "inspect", env=env)
    assert completed.returncode == 0
    payload = json.loads(completed.stdout)
    target_ids = {target["id"] for target in payload["targets"]}
    assert {"tools", "ollama", "dev", "ssl"}.issubset(target_ids)


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
    }
    completed = run_cli("service", "list", env=env)
    assert completed.returncode == 0
    payload = json.loads(completed.stdout)
    service_names = {service["name"] for service in payload["services"]}
    assert {"openclaw-dev", "openclaw-test", "caddy"} == service_names
    supported = {service["name"] for service in payload["services"] if service["supported_by_gateway"]}
    assert {"openclaw-dev", "openclaw-test", "caddy"} == supported


def test_runtime_config_sync_reads_external_runtime_repository(tmp_path: Path, monkeypatch) -> None:
    runtime_repo = create_git_repo(
        tmp_path / "moltbox-runtime",
        {
            "openclaw-dev/openclaw.json": json.dumps({"gateway": {"controlUi": {"allowedOrigins": []}}}, indent=2) + "\n",
            "openclaw-dev/model-runtime.yml": "model: qwen3:8b\n",
            "openclaw-dev/opensearch.yml": "cluster.name: remram\n",
            "openclaw-dev/openclaw/channels.yaml": "channels: []\n",
        },
    )
    monkeypatch.setenv("MOLTBOX_STATE_ROOT", str(tmp_path / ".remram"))
    monkeypatch.setenv("MOLTBOX_RUNTIME_ROOT", str(tmp_path / "Moltbox"))
    monkeypatch.setenv("MOLTBOX_RUNTIME_REPO_URL", str(runtime_repo))
    from moltbox_cli.config import resolve_config

    config = resolve_config(Args())
    payload = component_config_sync_action(config, "openclaw-dev")

    assert payload["ok"] is True
    runtime_root = Path(str(payload["runtime_root"]))
    assert (runtime_root / "openclaw.json").exists()
    synced = json.loads((runtime_root / "openclaw.json").read_text(encoding="utf-8"))
    allowed_origins = synced["gateway"]["controlUi"]["allowedOrigins"]
    assert any(origin.startswith("http://127.0.0.1:18790") for origin in allowed_origins)
    assert (runtime_root / "openclaw" / "channels.yaml").exists()


def test_skill_deploy_uses_skill_manifest_plan(tmp_path: Path, monkeypatch) -> None:
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
    from moltbox_cli.config import resolve_config
    import moltbox_cli.v2_actions as v2_actions

    calls: list[tuple[str, str]] = []

    def fake_service_deploy(config, name):  # noqa: ANN001
        calls.append(("service_deploy", name))
        return {"ok": True, "name": name}

    def fake_config_sync(config, name):  # noqa: ANN001
        calls.append(("runtime_sync", name))
        return {"ok": True, "name": name}

    def fake_component_action(config, name, verb, **kwargs):  # noqa: ANN001, ARG001
        calls.append((verb, name))
        return {"ok": True, "name": name, "verb": verb}

    monkeypatch.setattr(v2_actions, "service_deploy_action", fake_service_deploy)
    monkeypatch.setattr(v2_actions, "component_config_sync_action", fake_config_sync)
    monkeypatch.setattr(v2_actions, "component_action", fake_component_action)

    config = resolve_config(Args())
    payload = skill_deploy_action(config, "discord")

    assert payload["ok"] is True
    assert calls == [
        ("service_deploy", "caddy"),
        ("runtime_sync", "openclaw-dev"),
        ("reload", "openclaw-dev"),
    ]


def test_find_repo_root_accepts_baked_container_checkout(tmp_path: Path, monkeypatch) -> None:
    repo_root = tmp_path / "remram-gateway"
    (repo_root / "moltbox").mkdir(parents=True)
    (repo_root / "moltbox-cli").mkdir(parents=True)
    monkeypatch.setenv("MOLTBOX_REPO_ROOT", str(repo_root))
    assert find_repo_root() == repo_root.resolve()
