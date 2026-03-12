from __future__ import annotations

import os
from dataclasses import dataclass
from pathlib import Path


@dataclass(frozen=True)
class GatewayLayout:
    repo_root: Path
    state_root: Path
    deploy_root: Path
    upstream_root: Path
    repos_root: Path
    runtime_root: Path
    logs_root: Path
    service_state_root: Path
    gateway_state_root: Path
    gateway_logs_root: Path

    def ensure(self) -> "GatewayLayout":
        for path in (
            self.state_root,
            self.deploy_root,
            self.upstream_root,
            self.repos_root,
            self.runtime_root,
            self.logs_root,
            self.service_state_root,
            self.gateway_state_root,
            self.gateway_logs_root,
        ):
            path.mkdir(parents=True, exist_ok=True)
        return self

    def rendered_service_dir(self, service_name: str) -> Path:
        return self.deploy_root / "rendered" / service_name

    def runtime_component_dir(self, component_name: str) -> Path:
        return self.runtime_root / component_name

    def service_state_dir(self, service_name: str) -> Path:
        return self.service_state_root / service_name


def find_repo_root(start: Path | None = None) -> Path:
    configured = os.environ.get("MOLTBOX_REPO_ROOT")
    if configured:
        return Path(configured).expanduser().resolve()
    current = (start or Path(__file__)).resolve()
    for candidate in (current, *current.parents):
        if (candidate / ".git").exists():
            return candidate
    raise RuntimeError("unable to locate remram-gateway repo root")


def build_layout(state_root: Path, runtime_artifacts_root: Path, logs_root: Path) -> GatewayLayout:
    repo_root = find_repo_root()
    layout = GatewayLayout(
        repo_root=repo_root,
        state_root=state_root,
        deploy_root=state_root / "deploy",
        upstream_root=state_root / "upstream",
        repos_root=state_root / "repos",
        runtime_root=runtime_artifacts_root,
        logs_root=logs_root,
        service_state_root=state_root / "services",
        gateway_state_root=state_root / "gateway",
        gateway_logs_root=logs_root / "gateway",
    )
    return layout.ensure()
