from __future__ import annotations

import subprocess
from dataclasses import dataclass

from . import __version__
from .layout import find_repo_root


@dataclass(frozen=True)
class VersionInfo:
    version: str
    source: str
    git_commit: str | None = None

    def as_dict(self) -> dict[str, str]:
        payload = {"version": self.version, "source": self.source}
        if self.git_commit:
            payload["git_commit"] = self.git_commit
        return payload


def resolve_version_info() -> VersionInfo:
    try:
        completed = subprocess.run(
            ["git", "rev-parse", "--short", "HEAD"],
            cwd=str(find_repo_root()),
            capture_output=True,
            text=True,
            check=True,
            timeout=5,
        )
        git_commit = completed.stdout.strip()
    except (OSError, subprocess.SubprocessError):
        git_commit = ""
    if git_commit:
        return VersionInfo(version=git_commit, source="git", git_commit=git_commit)
    return VersionInfo(version=__version__, source="package")
