from __future__ import annotations

import argparse
import json
import shlex
import subprocess
import sys
from dataclasses import dataclass
from pathlib import Path


DEFAULT_STATE_ROOT = "/srv/moltbox-state"
DEFAULT_LOGS_ROOT = "/srv/moltbox-logs"
DEFAULT_GATEWAY_PORT = 7474
DEFAULT_GATEWAY_HTTP_PORT = 17474


@dataclass(frozen=True)
class RemoteHostInfo:
    ssh_target: str
    hostname: str
    system: str
    kernel_release: str
    distro_id: str
    version_id: str
    pretty_name: str

    def as_dict(self) -> dict[str, str]:
        return {
            "ssh_target": self.ssh_target,
            "hostname": self.hostname,
            "system": self.system,
            "kernel_release": self.kernel_release,
            "distro_id": self.distro_id,
            "version_id": self.version_id,
            "pretty_name": self.pretty_name,
        }


def _run(command: list[str]) -> subprocess.CompletedProcess[str]:
    return subprocess.run(command, capture_output=True, text=True, check=False)


def _ssh(ssh_target: str, remote_command: str) -> subprocess.CompletedProcess[str]:
    return _run(["ssh", ssh_target, "bash", "-lc", remote_command])


def _require(completed: subprocess.CompletedProcess[str], *, error_message: str, recovery_message: str) -> str:
    if completed.returncode == 0:
        return completed.stdout.strip()
    raise SystemExit(
        json.dumps(
            {
                "ok": False,
                "error": error_message,
                "recovery": recovery_message,
                "stdout": completed.stdout.strip(),
                "stderr": completed.stderr.strip(),
            },
            indent=2,
        )
    )


def detect_remote_host(ssh_target: str) -> RemoteHostInfo:
    probe = """
python3 - <<'PY'
import json
import pathlib
import platform
import socket

payload = {
    "hostname": socket.gethostname(),
    "system": platform.system(),
    "kernel_release": platform.release(),
    "distro_id": "",
    "version_id": "",
    "pretty_name": "",
}
os_release = pathlib.Path("/etc/os-release")
if os_release.exists():
    for line in os_release.read_text(encoding="utf-8").splitlines():
        if "=" not in line:
            continue
        key, value = line.split("=", 1)
        payload[key.lower()] = value.strip().strip('"')
payload["distro_id"] = payload.get("id", "")
payload["version_id"] = payload.get("version_id", "")
payload["pretty_name"] = payload.get("pretty_name", "")
print(json.dumps(payload))
PY
""".strip()
    completed = _ssh(ssh_target, probe)
    raw = _require(
        completed,
        error_message="failed to detect the remote host runtime",
        recovery_message="verify SSH access and that python3 is installed on the remote host",
    )
    payload = json.loads(raw)
    info = RemoteHostInfo(
        ssh_target=ssh_target,
        hostname=str(payload.get("hostname") or ""),
        system=str(payload.get("system") or ""),
        kernel_release=str(payload.get("kernel_release") or ""),
        distro_id=str(payload.get("distro_id") or ""),
        version_id=str(payload.get("version_id") or ""),
        pretty_name=str(payload.get("pretty_name") or ""),
    )
    if info.system.lower() != "linux":
        raise SystemExit(
            json.dumps(
                {
                    "ok": False,
                    "error": "remote host is not Linux",
                    "recovery": "bootstrap is only supported against a Linux Moltbox host",
                    "host": info.as_dict(),
                },
                indent=2,
            )
        )
    return info


def require_remote_sudo(ssh_target: str) -> None:
    completed = _ssh(ssh_target, "sudo -n true")
    _require(
        completed,
        error_message="passwordless sudo is required to prepare machine-scoped storage roots",
        recovery_message="grant the operator account passwordless sudo for the bootstrap commands or pre-create the appliance directories",
    )


def require_remote_git_access(ssh_target: str, *, repo_name: str, repo_url: str) -> None:
    completed = _ssh(ssh_target, f"git ls-remote {shlex.quote(repo_url)} HEAD")
    _require(
        completed,
        error_message=f"remote host cannot read the required Git repository '{repo_name}'",
        recovery_message="configure host-side Git credentials for the private repositories before retrying bootstrap",
    )


def prepare_remote_storage(ssh_target: str, *, state_root: str, logs_root: str) -> None:
    command = " && ".join(
        [
            f"sudo install -d -o $USER -g $USER {shlex.quote(state_root)}",
            f"sudo install -d -o $USER -g $USER {shlex.quote(logs_root)}",
            f"sudo install -d -o $USER -g $USER {shlex.quote(state_root + '/upstream')}",
            f"sudo install -d -o $USER -g $USER {shlex.quote(state_root + '/repos')}",
            f"sudo install -d -o $USER -g $USER {shlex.quote(state_root + '/runtime')}",
        ]
    )
    completed = _ssh(ssh_target, command)
    _require(
        completed,
        error_message="failed to prepare machine-scoped Moltbox storage roots",
        recovery_message="verify sudo access and filesystem permissions on the remote host",
    )


def sync_remote_checkout(ssh_target: str, *, repo_url: str, checkout_dir: str, ref: str | None = None) -> None:
    parent_dir = str(Path(checkout_dir).parent).replace("\\", "/")
    if ref:
        command = (
            f"mkdir -p {shlex.quote(parent_dir)}"
            f" && if [ -d {shlex.quote(checkout_dir + '/.git')} ]; then "
            f"git -C {shlex.quote(checkout_dir)} fetch --all --tags --prune"
            f" && git -C {shlex.quote(checkout_dir)} checkout {shlex.quote(ref)}; "
            f"else git clone {shlex.quote(repo_url)} {shlex.quote(checkout_dir)}"
            f" && git -C {shlex.quote(checkout_dir)} checkout {shlex.quote(ref)}; fi"
        )
    else:
        command = (
            f"mkdir -p {shlex.quote(parent_dir)}"
            f" && if [ -d {shlex.quote(checkout_dir + '/.git')} ]; then "
            f"git -C {shlex.quote(checkout_dir)} pull --ff-only; "
            f"else git clone {shlex.quote(repo_url)} {shlex.quote(checkout_dir)}; fi"
        )
    completed = _ssh(ssh_target, command)
    _require(
        completed,
        error_message=f"failed to sync the remote checkout for {repo_url}",
        recovery_message="verify remote Git credentials and rerun bootstrap",
    )


def deploy_gateway(
    ssh_target: str,
    *,
    gateway_checkout: str,
    state_root: str,
    logs_root: str,
    services_repo_url: str,
    runtime_repo_url: str,
    skills_repo_url: str,
    gateway_ref: str,
) -> dict[str, object]:
    runtime_root = f"{state_root}/runtime"
    pythonpath = ":".join(
        [
            f"{gateway_checkout}/cli/src",
            f"{gateway_checkout}/commands/src",
            f"{gateway_checkout}/services/src",
            f"{gateway_checkout}/runtime/src",
            f"{gateway_checkout}/docker/src",
            f"{gateway_checkout}/repos/src",
        ]
    )
    command = " && ".join(
        [
            f"export PYTHONPATH={shlex.quote(pythonpath)}",
            f"export MOLTBOX_REPO_ROOT={shlex.quote(gateway_checkout)}",
            f"export MOLTBOX_STATE_ROOT={shlex.quote(state_root)}",
            f"export MOLTBOX_LOGS_ROOT={shlex.quote(logs_root)}",
            f"export MOLTBOX_RUNTIME_ROOT={shlex.quote(runtime_root)}",
            f"export MOLTBOX_SERVICES_REPO_URL={shlex.quote(services_repo_url)}",
            f"export MOLTBOX_RUNTIME_REPO_URL={shlex.quote(runtime_repo_url)}",
            f"export MOLTBOX_SKILLS_REPO_URL={shlex.quote(skills_repo_url)}",
            "export MOLTBOX_INTERNAL_HOST=0.0.0.0",
            f"export MOLTBOX_INTERNAL_PORT={DEFAULT_GATEWAY_PORT}",
            "python3 -m moltbox_cli service deploy gateway --commit " + shlex.quote(gateway_ref),
        ]
    )
    completed = _ssh(ssh_target, command)
    raw = _require(
        completed,
        error_message="gateway deploy command failed on the remote host",
        recovery_message="inspect the remote deployment logs and rerun bootstrap after correcting the failure",
    )
    return json.loads(raw)


def probe_gateway(ssh_target: str, *, gateway_http_port: int) -> dict[str, object]:
    completed = _ssh(ssh_target, f"curl -fsS http://127.0.0.1:{gateway_http_port}/health")
    raw = _require(
        completed,
        error_message="gateway health probe failed after deployment",
        recovery_message="inspect the gateway container and logs on the remote host",
    )
    return json.loads(raw)


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Bootstrap the Moltbox gateway onto a remote Linux host.")
    parser.add_argument("--host", default="moltbox", help="SSH target for the Moltbox host")
    parser.add_argument("--state-root", default=DEFAULT_STATE_ROOT)
    parser.add_argument("--logs-root", default=DEFAULT_LOGS_ROOT)
    parser.add_argument("--gateway-http-port", type=int, default=DEFAULT_GATEWAY_HTTP_PORT)
    parser.add_argument("--gateway-repo-url", default="https://github.com/remram-ai/remram-gateway.git")
    parser.add_argument("--gateway-ref", default="main")
    parser.add_argument("--services-repo-url", default="https://github.com/remram-ai/moltbox-services.git")
    parser.add_argument("--runtime-repo-url", default="https://github.com/remram-ai/moltbox-runtime.git")
    parser.add_argument("--skills-repo-url", default="https://github.com/remram-ai/remram-skills.git")
    return parser


def main(argv: list[str] | None = None) -> int:
    args = build_parser().parse_args(argv)
    host = detect_remote_host(args.host)
    require_remote_sudo(args.host)

    require_remote_git_access(args.host, repo_name="remram-gateway", repo_url=args.gateway_repo_url)
    require_remote_git_access(args.host, repo_name="moltbox-services", repo_url=args.services_repo_url)
    require_remote_git_access(args.host, repo_name="moltbox-runtime", repo_url=args.runtime_repo_url)
    require_remote_git_access(args.host, repo_name="remram-skills", repo_url=args.skills_repo_url)

    prepare_remote_storage(args.host, state_root=args.state_root, logs_root=args.logs_root)

    gateway_checkout = f"{args.state_root}/upstream/remram-gateway"
    services_checkout = f"{args.state_root}/upstream/moltbox-services"
    runtime_checkout = f"{args.state_root}/upstream/moltbox-runtime"
    skills_checkout = f"{args.state_root}/upstream/remram-skills"

    sync_remote_checkout(args.host, repo_url=args.gateway_repo_url, checkout_dir=gateway_checkout, ref=args.gateway_ref)
    sync_remote_checkout(args.host, repo_url=args.services_repo_url, checkout_dir=services_checkout)
    sync_remote_checkout(args.host, repo_url=args.runtime_repo_url, checkout_dir=runtime_checkout)
    sync_remote_checkout(args.host, repo_url=args.skills_repo_url, checkout_dir=skills_checkout)

    deploy_payload = deploy_gateway(
        args.host,
        gateway_checkout=gateway_checkout,
        state_root=args.state_root,
        logs_root=args.logs_root,
        services_repo_url=services_checkout,
        runtime_repo_url=runtime_checkout,
        skills_repo_url=skills_checkout,
        gateway_ref=args.gateway_ref,
    )
    health_payload = probe_gateway(args.host, gateway_http_port=args.gateway_http_port)
    print(
        json.dumps(
            {
                "ok": True,
                "host": host.as_dict(),
                "state_root": args.state_root,
                "logs_root": args.logs_root,
                "deploy": deploy_payload,
                "health": health_payload,
            },
            indent=2,
        )
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
