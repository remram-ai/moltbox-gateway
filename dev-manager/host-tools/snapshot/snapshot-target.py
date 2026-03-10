from pathlib import Path
import sys

SRC_DIR = Path(__file__).resolve().parents[2] / "control-pane" / "src"
sys.path.insert(0, str(SRC_DIR))

from remram_dev_manager_control_pane.host_tool_cli import main


if __name__ == "__main__":
    raise SystemExit(main("snapshot_target"))
