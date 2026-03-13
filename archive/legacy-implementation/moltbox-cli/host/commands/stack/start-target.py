from pathlib import Path
import sys

SRC_DIR = Path(__file__).resolve().parents[3] / "tools" / "src"
sys.path.insert(0, str(SRC_DIR))

from moltbox_cli.host_tool_cli import main


if __name__ == "__main__":
    raise SystemExit(main("start_target"))
