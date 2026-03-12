from __future__ import annotations

import os
import subprocess
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
SOURCE_DIRS = [
    ROOT / "cli" / "src",
    ROOT / "commands" / "src",
    ROOT / "services" / "src",
    ROOT / "runtime" / "src",
    ROOT / "docker" / "src",
    ROOT / "repos" / "src",
]

for source_dir in reversed(SOURCE_DIRS):
    sys.path.insert(0, str(source_dir))


def run_cli(*args: str, env: dict[str, str] | None = None) -> subprocess.CompletedProcess[str]:
    merged_env = os.environ.copy()
    merged_env["PYTHONPATH"] = os.pathsep.join(str(item) for item in SOURCE_DIRS) + os.pathsep + merged_env.get("PYTHONPATH", "")
    merged_env["MOLTBOX_REPO_ROOT"] = str(ROOT)
    if env:
        merged_env.update(env)
    return subprocess.run(
        [sys.executable, "-m", "moltbox_cli", *args],
        cwd=str(ROOT),
        capture_output=True,
        text=True,
        check=False,
        env=merged_env,
    )


def create_git_repo(path: Path, files: dict[str, str]) -> Path:
    path.mkdir(parents=True, exist_ok=True)
    init = subprocess.run(["git", "init", str(path)], capture_output=True, text=True, check=False)
    assert init.returncode == 0, init.stderr
    for relative_path, content in files.items():
        file_path = path / relative_path
        file_path.parent.mkdir(parents=True, exist_ok=True)
        file_path.write_text(content, encoding="utf-8")
    add = subprocess.run(["git", "-C", str(path), "add", "."], capture_output=True, text=True, check=False)
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
