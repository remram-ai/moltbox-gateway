from __future__ import annotations

import argparse

from .service import run_server


def main() -> None:
    parser = argparse.ArgumentParser(description="Run the Moltbox debug service.")
    parser.add_argument("command", choices=["serve"], help="Command to execute.")
    parser.add_argument("--host", help="Host/IP to bind.")
    parser.add_argument("--port", type=int, help="Port to bind.")
    args = parser.parse_args()
    run_server(args)


if __name__ == "__main__":
    main()
