from __future__ import annotations

import argparse

from .commands.version import handle_version
from .errors import RemramError, TargetNotFoundError
from .jsonio import emit_json


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Canonical Remram control-plane CLI.")
    parser.add_argument("--config-path")
    parser.add_argument("--state-root")
    parser.add_argument("--runtime-artifacts-root")
    parser.add_argument("--internal-host")
    parser.add_argument("--internal-port", type=int)
    parser.add_argument("--cli-path")
    subparsers = parser.add_subparsers(dest="command", required=True)

    subparsers.add_parser("version", help="Print control-plane version metadata.")
    subparsers.add_parser("health", help="Read control-plane health.")
    subparsers.add_parser("serve", help="Run the control-plane service in the foreground.")
    subparsers.add_parser("list-targets", help="List canonical targets from the registry.")
    status = subparsers.add_parser("status", help="Read target status.")
    status.add_argument("--target", required=True, help="Canonical target id or supported alias.")
    deploy = subparsers.add_parser("deploy", help="Deploy a canonical target.")
    deploy.add_argument("target", help="Canonical target id or supported alias.")
    rollback = subparsers.add_parser("rollback", help="Rollback a canonical target to its latest snapshot.")
    rollback.add_argument("target", help="Canonical target id or supported alias.")
    render_assets = subparsers.add_parser("render-assets", help="Render canonical deployment assets for diagnostics.")
    render_assets.add_argument("--target", required=True, help="Canonical target id or supported alias.")
    render_assets.add_argument("--profile", help="Explicit profile for runtime targets.")
    runtime = subparsers.add_parser("runtime", help="Operate on runtime environments.")
    runtime_subparsers = runtime.add_subparsers(dest="runtime_command", required=True)
    for action in ("start", "stop", "restart"):
        runtime_action = runtime_subparsers.add_parser(action, help=f"{action.title()} a runtime environment.")
        runtime_action.add_argument("env", help="One of dev, test, or prod.")
    return parser


def main() -> None:
    parser = build_parser()
    args = parser.parse_args()
    try:
        if args.command == "version":
            emit_json(handle_version())
            return

        from .config import resolve_config
        from .registry import get_target
        from .target_resolution import resolve_target_identifier

        config = resolve_config(args)
        if args.command == "health":
            from .commands.health import handle_health

            emit_json(handle_health(config))
            return

        if args.command == "list-targets":
            from .commands.targets import handle_list_targets

            emit_json(handle_list_targets(config))
            return

        if args.command == "status":
            from .commands.status import handle_status

            get_target(config, args.target)
            payload = handle_status(config, args.target)
            emit_json(payload)
            raise SystemExit(int(payload.get("exit_code", 0)))

        if args.command == "deploy":
            from .commands.deploy import handle_deploy

            get_target(config, args.target)
            payload = handle_deploy(config, args.target)
            emit_json(payload)
            raise SystemExit(int(payload.get("exit_code", 0)))

        if args.command == "rollback":
            from .commands.rollback import handle_rollback

            get_target(config, args.target)
            payload = handle_rollback(config, args.target)
            emit_json(payload)
            raise SystemExit(int(payload.get("exit_code", 0)))

        if args.command == "render-assets":
            from .commands.render_assets import handle_render_assets

            get_target(config, args.target)
            payload = handle_render_assets(config, args.target, args.profile)
            emit_json(payload)
            raise SystemExit(int(payload.get("exit_code", 0)))

        if args.command == "runtime":
            from .commands.runtime import handle_runtime

            resolved = resolve_target_identifier(args.env)
            if resolved not in {"dev", "test", "prod"}:
                raise TargetNotFoundError(args.env)
            get_target(config, resolved)
            payload = handle_runtime(config, resolved, args.runtime_command)
            emit_json(payload)
            raise SystemExit(int(payload.get("exit_code", 0)))

        if args.command == "serve":
            from .commands.serve import handle_serve

            handle_serve(config)
            return

        raise RemramError(
            error_type="unsupported_command",
            error_message=f"unsupported command '{args.command}'",
            recovery_message="run `remram --help` to see available commands",
        )
    except RemramError as exc:
        emit_json(exc.to_payload())
        raise SystemExit(exc.exit_code) from exc
    except KeyboardInterrupt:
        raise SystemExit(130)
    except Exception as exc:  # noqa: BLE001
        emit_json(
            RemramError(
                error_type="execution_failure",
                error_message=str(exc),
                recovery_message="inspect the control-plane logs and rerun the command",
            ).to_payload()
        )
        raise SystemExit(1) from exc
