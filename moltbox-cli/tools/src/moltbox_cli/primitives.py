from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path

from .layout import HostLayout, RepoLayout, build_host_layout, build_repo_layout


@dataclass(frozen=True)
class PrimitiveDefinition:
    name: str
    category: str
    summary: str
    relative_source_path: str
    mutates_state: bool

    def source_path(self, repo_layout: RepoLayout) -> Path:
        return repo_layout.moltbox_cli_dir / Path(self.relative_source_path)

    def installed_path(self, host_layout: HostLayout) -> Path:
        return host_layout.host_tools_dir / Path(self.relative_source_path)

    def as_dict(self, repo_layout: RepoLayout, host_layout: HostLayout) -> dict[str, str | bool]:
        source_path = self.source_path(repo_layout)
        return {
            "name": self.name,
            "category": self.category,
            "summary": self.summary,
            "source_path": str(source_path),
            "installed_path": str(self.installed_path(host_layout)),
            "mutates_state": self.mutates_state,
            "source_exists": source_path.exists(),
        }


PRIMITIVES: tuple[PrimitiveDefinition, ...] = (
    PrimitiveDefinition("render_assets", "deploy", "Render canonical deployment assets.", "host/commands/deploy/render-assets.py", False),
    PrimitiveDefinition("inspect_target", "shared", "Inspect a deploy-managed target.", "host/commands/shared/inspect-target.py", False),
    PrimitiveDefinition("tail_target_logs", "shared", "Tail container logs for a target.", "host/commands/shared/tail-target-logs.py", False),
    PrimitiveDefinition("deploy_target", "deploy", "Deploy a rendered target.", "host/commands/deploy/deploy-target.py", True),
    PrimitiveDefinition("start_target", "stack", "Start a host-service target.", "host/commands/stack/start-target.py", True),
    PrimitiveDefinition("stop_target", "stack", "Stop a host-service target.", "host/commands/stack/stop-target.py", True),
    PrimitiveDefinition("restart_target", "stack", "Restart a host-service target.", "host/commands/stack/restart-target.py", True),
    PrimitiveDefinition("start_runtime", "stack", "Start a runtime target.", "runtime/commands/stack/start-runtime.py", True),
    PrimitiveDefinition("stop_runtime", "stack", "Stop a runtime target.", "runtime/commands/stack/stop-runtime.py", True),
    PrimitiveDefinition("restart_runtime", "stack", "Restart a runtime target.", "runtime/commands/stack/restart-runtime.py", True),
    PrimitiveDefinition("snapshot_target", "snapshot", "Create a target-scoped snapshot.", "host/commands/snapshot/snapshot-target.py", True),
    PrimitiveDefinition("restore_target_snapshot", "snapshot", "Restore a target-scoped snapshot.", "host/commands/snapshot/restore-target-snapshot.py", True),
    PrimitiveDefinition("validate_target", "validate", "Validate a target using the baseline container validator.", "host/commands/validate/validate-target.py", False),
)


def list_primitives(
    repo_layout: RepoLayout | None = None,
    host_layout: HostLayout | None = None,
) -> list[dict[str, str | bool]]:
    repo = repo_layout or build_repo_layout()
    host = host_layout or build_host_layout()
    return [primitive.as_dict(repo, host) for primitive in PRIMITIVES]
