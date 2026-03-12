from __future__ import annotations

import json
import shutil
import subprocess
import time
from pathlib import Path
from typing import Any

import yaml


def _run(command: list[str], cwd: Path | None = None) -> subprocess.CompletedProcess[str]:
    return subprocess.run(command, cwd=str(cwd) if cwd else None, capture_output=True, text=True, check=False)


def docker_available() -> bool:
    return shutil.which("docker") is not None


def inspect_containers(container_names: list[str]) -> dict[str, Any]:
    if not docker_available():
        return {"ok": False, "details": {"reason": "docker_not_available"}}
    containers: list[dict[str, Any]] = []
    for name in container_names:
        completed = _run(["docker", "inspect", name])
        if completed.returncode != 0:
            containers.append(
                {
                    "name": name,
                    "present": False,
                    "state": "missing",
                    "health": None,
                    "container_id": None,
                    "image": None,
                }
            )
            continue
        payload = json.loads(completed.stdout)
        item = payload[0]
        state = item.get("State") or {}
        health = state.get("Health") or {}
        config = item.get("Config") or {}
        containers.append(
            {
                "name": name,
                "present": True,
                "state": str(state.get("Status") or "unknown"),
                "health": health.get("Status"),
                "container_id": item.get("Id"),
                "image": config.get("Image"),
            }
        )
    aggregate_state = "missing"
    if any(container["state"] == "running" for container in containers):
        aggregate_state = "running"
    elif any(container["present"] for container in containers):
        aggregate_state = "stopped"
    return {
        "ok": True,
        "details": {
            "container_state": {"state": aggregate_state, "containers": containers},
            "container_ids": [item["container_id"] for item in containers if item["container_id"]],
        },
    }


def tail_logs(container_names: list[str], tail_lines: int = 50) -> dict[str, Any]:
    if not docker_available():
        return {"ok": False, "details": {"reason": "docker_not_available"}}
    chunks: list[str] = []
    for name in container_names:
        completed = _run(["docker", "logs", "--tail", str(tail_lines), name])
        output = completed.stdout.strip()
        if completed.stderr.strip():
            output = "\n".join(part for part in [output, completed.stderr.strip()] if part)
        if output:
            chunks.append(f"[{name}]\n{output}")
    return {"ok": True, "details": {"log_tail": "\n\n".join(chunks)}}


def _compose_command(render_dir: Path, compose_project: str, args: list[str]) -> list[str]:
    return [
        "docker",
        "compose",
        "-f",
        str(render_dir / "compose.yml"),
        "-p",
        compose_project,
        *args,
    ]


def _load_compose(render_dir: Path) -> dict[str, Any]:
    compose_path = render_dir / "compose.yml"
    loaded = yaml.safe_load(compose_path.read_text(encoding="utf-8"))
    if loaded is None:
        return {}
    if not isinstance(loaded, dict):
        return {}
    return loaded


def ensure_external_networks(render_dir: Path) -> dict[str, Any]:
    compose = _load_compose(render_dir)
    networks = compose.get("networks")
    if not isinstance(networks, dict):
        return {"ok": True, "details": {"created": [], "existing": [], "required": []}}
    created: list[str] = []
    existing: list[str] = []
    required: list[str] = []
    for _, config in networks.items():
        if not isinstance(config, dict):
            continue
        external = config.get("external")
        if external not in {True, "true", "True"}:
            continue
        network_name = str(config.get("name") or "").strip()
        if not network_name:
            continue
        required.append(network_name)
        inspected = _run(["docker", "network", "inspect", network_name])
        if inspected.returncode == 0:
            existing.append(network_name)
            continue
        created_result = _run(["docker", "network", "create", network_name])
        if created_result.returncode != 0:
            return {
                "ok": False,
                "details": {
                    "created": created,
                    "existing": existing,
                    "required": required,
                    "failed_network": network_name,
                    "stderr": created_result.stderr.strip(),
                },
            }
        created.append(network_name)
    return {
        "ok": True,
        "details": {
            "created": created,
            "existing": existing,
            "required": required,
        },
    }


def pull_stack(render_dir: Path, compose_project: str) -> dict[str, Any]:
    if not docker_available():
        return {"ok": False, "details": {"reason": "docker_not_available"}}
    command = _compose_command(render_dir, compose_project, ["pull"])
    completed = _run(command)
    return {
        "ok": completed.returncode == 0,
        "stdout": completed.stdout.strip(),
        "stderr": completed.stderr.strip(),
        "details": {"compose_command": command},
    }


def _remove_existing_containers(container_names: list[str]) -> list[str]:
    removed: list[str] = []
    for name in container_names:
        inspected = _run(["docker", "inspect", name])
        if inspected.returncode != 0:
            continue
        removed_completed = _run(["docker", "rm", "-f", name])
        if removed_completed.returncode == 0:
            removed.append(name)
    return removed


def deploy_stack(
    *,
    render_dir: Path,
    compose_project: str,
    container_names: list[str],
    build_images: bool = False,
    remove_orphans: bool = True,
    force_recreate: bool = False,
    replace_existing_containers: bool = True,
) -> dict[str, Any]:
    if not docker_available():
        return {"ok": False, "details": {"reason": "docker_not_available"}}
    network_bootstrap = ensure_external_networks(render_dir)
    if not network_bootstrap.get("ok"):
        return {
            "ok": False,
            "details": {
                "reason": "network_bootstrap_failed",
                **(network_bootstrap.get("details") or {}),
            },
        }
    removed = _remove_existing_containers(container_names) if replace_existing_containers else []
    compose_args = ["up", "-d"]
    if build_images:
        compose_args.append("--build")
    if force_recreate:
        compose_args.append("--force-recreate")
    if remove_orphans:
        compose_args.append("--remove-orphans")
    command = _compose_command(render_dir, compose_project, compose_args)
    completed = _run(command)
    inspect = inspect_containers(container_names)
    return {
        "ok": completed.returncode == 0,
        "stdout": completed.stdout.strip(),
        "stderr": completed.stderr.strip(),
        "details": {
            "compose_command": command,
            "network_bootstrap": network_bootstrap.get("details"),
            "removed_existing_containers": removed,
            "new_container_ids": (inspect.get("details") or {}).get("container_ids", []),
        },
    }


def lifecycle(container_names: list[str], action: str) -> dict[str, Any]:
    if not docker_available():
        return {"ok": False, "details": {"reason": "docker_not_available"}}
    command = ["docker", action, *container_names]
    completed = _run(command)
    return {
        "ok": completed.returncode == 0,
        "stdout": completed.stdout.strip(),
        "stderr": completed.stderr.strip(),
        "details": {"docker_command": command},
    }


def validate_containers(container_names: list[str], timeout_seconds: int = 30, poll_interval_seconds: int = 2) -> dict[str, Any]:
    deadline = time.monotonic() + max(timeout_seconds, 0)
    inspected = inspect_containers(container_names)
    if not inspected.get("ok"):
        details = inspected.get("details") or {}
        return {
            "ok": False,
            "details": {
                "containers": [],
                "result": "fail",
                "reason": details.get("reason") or "inspect_failed",
            },
            "errors": [details.get("reason") or "inspect_failed"],
        }
    details = inspected.get("details") or {}
    containers = (details.get("container_state") or {}).get("containers") or []
    errors = [item["name"] for item in containers if item["state"] not in {"running", "created"}]
    while errors and time.monotonic() < deadline:
        time.sleep(max(poll_interval_seconds, 0))
        inspected = inspect_containers(container_names)
        if not inspected.get("ok"):
            details = inspected.get("details") or {}
            return {
                "ok": False,
                "details": {
                    "containers": containers,
                    "result": "fail",
                    "reason": details.get("reason") or "inspect_failed",
                },
                "errors": errors or [details.get("reason") or "inspect_failed"],
            }
        details = inspected.get("details") or {}
        containers = (details.get("container_state") or {}).get("containers") or []
        errors = [item["name"] for item in containers if item["state"] not in {"running", "created"}]
    return {
        "ok": not errors,
        "details": {
            "containers": containers,
            "result": "pass" if not errors else "fail",
        },
        "errors": errors,
    }
