from __future__ import annotations

import argparse
from dataclasses import dataclass
from typing import Sequence

from moltbox_commands import ComponentCommand, GatewayCommand, ServiceCommand, SkillCommand, dispatch
from moltbox_commands.core.config import GatewayConfig, resolve_config
from moltbox_commands.core.errors import GatewayError, ValidationError
from moltbox_commands.core.jsonio import emit_json
from moltbox_commands.core.versioning import resolve_version_info
from . import gateway_server


@dataclass(frozen=True)
class ParsedCli:
    config: GatewayConfig
    request: GatewayCommand | ServiceCommand | ComponentCommand | SkillCommand | None
    show_help: bool = False
    show_version: bool = False


def help_text() -> str:
    return (
        "usage: moltbox <component> <command>\n\n"
        "Components:\n"
        "  gateway                    gateway status | inspect | logs | health | update\n"
        "  service                    service list | inspect <service> | status <service> | logs <service>\n"
        "                             service deploy <service> [--version <tag>] [--commit <sha>]\n"
        "                             service start|stop|restart|rollback|doctor <service>\n"
        "  skill                      skill deploy <skill>\n"
        "  openclaw                   alias for openclaw-prod\n"
        "  openclaw-dev               config sync | reload | status | inspect | logs | doctor | monitor\n"
        "  openclaw-test              config sync | reload | status | inspect | logs | doctor | monitor\n"
        "  openclaw-prod              config sync | reload | status | inspect | logs | doctor | monitor\n"
        "  caddy                      status | inspect | logs | start | stop | restart | doctor\n"
        "  opensearch                 status | inspect | logs | start | stop | restart | doctor\n\n"
        "Examples:\n"
        "  moltbox gateway status\n"
        "  moltbox service deploy gateway\n"
        "  moltbox service deploy openclaw-dev --version 1.4.2\n"
        "  moltbox openclaw reload\n"
        "  moltbox openclaw-dev config sync\n"
        "  moltbox skill deploy discord\n"
    )


def _global_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(add_help=False)
    parser.add_argument("--help", "-h", action="store_true")
    parser.add_argument("--version", action="store_true")
    parser.add_argument("--config-path")
    parser.add_argument("--state-root")
    parser.add_argument("--runtime-artifacts-root")
    parser.add_argument("--services-repo-url")
    parser.add_argument("--runtime-repo-url")
    parser.add_argument("--skills-repo-url")
    parser.add_argument("--internal-host")
    parser.add_argument("--internal-port", type=int)
    parser.add_argument("--cli-path")
    return parser


def _parse_gateway(args: list[str]) -> GatewayCommand:
    if not args:
        raise ValidationError(
            "gateway commands require a verb",
            "run `moltbox --help` for the supported gateway commands",
        )
    verb = args[0]
    if verb == "serve":
        if len(args) > 1:
            raise ValidationError(
                "gateway serve does not accept additional arguments",
                "run `moltbox gateway serve` without extra flags",
            )
        return GatewayCommand(verb="serve")
    parser = argparse.ArgumentParser(add_help=False)
    parser.add_argument("verb", choices=["health", "status", "inspect", "logs", "update"])
    parser.add_argument("--version")
    parser.add_argument("--commit")
    parsed = parser.parse_args(args)
    return GatewayCommand(verb=parsed.verb, version=parsed.version, commit=parsed.commit)


def _parse_service(args: list[str]) -> ServiceCommand:
    if not args:
        raise ValidationError(
            "service commands require a verb",
            "run `moltbox --help` for the supported service commands",
        )
    if args[0] == "list":
        return ServiceCommand(verb="list")
    parser = argparse.ArgumentParser(add_help=False)
    parser.add_argument("verb", choices=["inspect", "status", "logs", "deploy", "restart", "start", "stop", "rollback", "doctor"])
    parser.add_argument("service_name")
    parser.add_argument("--version")
    parser.add_argument("--commit")
    parsed = parser.parse_args(args)
    return ServiceCommand(
        verb=parsed.verb,
        service_name=parsed.service_name,
        version=parsed.version,
        commit=parsed.commit,
    )


def _parse_component(component_name: str, args: list[str]) -> ComponentCommand:
    if not args:
        raise ValidationError(
            f"component '{component_name}' requires a command",
            "run `moltbox --help` for the supported commands",
            component=component_name,
        )
    if args[0] == "config":
        if len(args) >= 2 and args[1] == "sync":
            return ComponentCommand(component_name=component_name, verb="config_sync")
        raise ValidationError(
            f"unsupported config command for component '{component_name}'",
            "use `config sync` for runtime configuration synchronization",
            component=component_name,
        )
    if args[0] not in {"status", "inspect", "logs", "reload", "doctor", "monitor", "start", "stop", "restart"}:
        raise ValidationError(
            f"unsupported component command '{args[0]}'",
            "run `moltbox --help` for the supported commands",
            component=component_name,
        )
    return ComponentCommand(component_name=component_name, verb=args[0])


def _parse_skill(args: list[str]) -> SkillCommand:
    parser = argparse.ArgumentParser(add_help=False)
    parser.add_argument("verb", choices=["deploy"])
    parser.add_argument("skill_name")
    parsed = parser.parse_args(args)
    return SkillCommand(verb=parsed.verb, skill_name=parsed.skill_name)


def parse_cli(argv: Sequence[str] | None = None) -> ParsedCli:
    parsed_globals, remainder = _global_parser().parse_known_args(list(argv or []))
    config = resolve_config(parsed_globals)
    if parsed_globals.version and not remainder:
        return ParsedCli(config=config, request=None, show_version=True)
    if parsed_globals.help and not remainder:
        return ParsedCli(config=config, request=None, show_help=True)
    if not remainder:
        return ParsedCli(config=config, request=None, show_help=True)

    component = remainder[0]
    if component == "gateway":
        return ParsedCli(config=config, request=_parse_gateway(remainder[1:]))
    if component == "service":
        return ParsedCli(config=config, request=_parse_service(remainder[1:]))
    if component == "skill":
        return ParsedCli(config=config, request=_parse_skill(remainder[1:]))
    return ParsedCli(config=config, request=_parse_component(component, remainder[1:]))


def execute(argv: Sequence[str] | None = None) -> dict[str, object]:
    parsed = parse_cli(argv)
    if parsed.show_help:
        return {"ok": True, "help": help_text()}
    if parsed.show_version:
        return {"ok": True, **resolve_version_info().as_dict()}
    if parsed.request is None:
        return {"ok": True, "help": help_text()}
    if isinstance(parsed.request, GatewayCommand) and parsed.request.verb == "serve":
        raise ValidationError(
            "gateway serve must be run through the CLI entrypoint",
            "run `moltbox gateway serve` directly instead of invoking it through execute()",
        )
    return dispatch(parsed.config, parsed.request)


def run(argv: Sequence[str] | None = None) -> int:
    try:
        parsed = parse_cli(argv)
        if isinstance(parsed.request, GatewayCommand) and parsed.request.verb == "serve":
            return gateway_server.serve(parsed.config)
        if parsed.show_help:
            print(help_text())
            return 0
        if parsed.show_version:
            emit_json({"ok": True, **resolve_version_info().as_dict()})
            return 0
        if parsed.request is None:
            print(help_text())
            return 0
        payload = dispatch(parsed.config, parsed.request)
    except GatewayError as exc:
        emit_json(exc.to_payload())
        return exc.exit_code
    except SystemExit as exc:
        emit_json(
            ValidationError(
                "invalid CLI arguments",
                "run `moltbox --help` and correct the command arguments",
            ).to_payload()
        )
        code = exc.code if isinstance(exc.code, int) else 1
        return code if code != 0 else 1

    emit_json(payload)
    if payload.get("ok", True):
        return 0
    return int(payload.get("exit_code", 1) or 1)
