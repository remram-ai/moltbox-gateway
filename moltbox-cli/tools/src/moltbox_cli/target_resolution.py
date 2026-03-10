ALIASES = {
    "cli": "control-plane",
    "control": "control-plane",
    "tools": "control-plane",
    "prime": "prod",
}

RUNTIME_TARGETS = {"dev", "test", "prod"}
HOST_TARGETS = {"ollama", "opensearch", "caddy"}
TOOLS_TARGET = "control-plane"


def resolve_target_identifier(target_id: str) -> str:
    return ALIASES.get(target_id, target_id)


def target_domain(target_id: str) -> str:
    resolved = resolve_target_identifier(target_id)
    if resolved == TOOLS_TARGET:
        return "tools"
    if resolved in RUNTIME_TARGETS:
        return "runtime"
    return "host"


def canonical_cli_command(target_id: str, verb: str) -> str:
    resolved = resolve_target_identifier(target_id)
    domain = target_domain(resolved)
    if domain == "tools":
        return f"moltbox tools {verb}"
    return f"moltbox {domain} {resolved} {verb}"
