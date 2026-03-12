from __future__ import annotations

from dataclasses import dataclass

from .errors import ValidationError


@dataclass(frozen=True)
class ComponentSpec:
    requested_name: str
    canonical_name: str
    kind: str
    service_name: str
    container_name: str
    compose_project: str
    runtime_name: str | None = None

    @property
    def supports_runtime(self) -> bool:
        return self.kind == "runtime"

    def as_dict(self) -> dict[str, object]:
        return {
            "requested_name": self.requested_name,
            "canonical_name": self.canonical_name,
            "kind": self.kind,
            "service_name": self.service_name,
            "container_name": self.container_name,
            "compose_project": self.compose_project,
            "runtime_name": self.runtime_name,
        }


_COMPONENTS = {
    "gateway": {
        "canonical_name": "gateway",
        "kind": "gateway",
        "service_name": "gateway",
        "container_name": "gateway",
        "compose_project": "gateway",
    },
    "openclaw": {
        "canonical_name": "openclaw-prod",
        "kind": "runtime",
        "service_name": "openclaw-prod",
        "container_name": "openclaw-prod",
        "compose_project": "openclaw-prod",
        "runtime_name": "openclaw-prod",
    },
    "openclaw-prod": {
        "canonical_name": "openclaw-prod",
        "kind": "runtime",
        "service_name": "openclaw-prod",
        "container_name": "openclaw-prod",
        "compose_project": "openclaw-prod",
        "runtime_name": "openclaw-prod",
    },
    "openclaw-dev": {
        "canonical_name": "openclaw-dev",
        "kind": "runtime",
        "service_name": "openclaw-dev",
        "container_name": "openclaw-dev",
        "compose_project": "openclaw-dev",
        "runtime_name": "openclaw-dev",
    },
    "openclaw-test": {
        "canonical_name": "openclaw-test",
        "kind": "runtime",
        "service_name": "openclaw-test",
        "container_name": "openclaw-test",
        "compose_project": "openclaw-test",
        "runtime_name": "openclaw-test",
    },
}


def try_resolve_component(name: str) -> ComponentSpec | None:
    raw = name.strip().lower()
    data = _COMPONENTS.get(raw)
    if data is None and raw:
        data = {
            "canonical_name": raw,
            "kind": "service",
            "service_name": raw,
            "container_name": raw,
            "compose_project": raw,
        }
    if data is None:
        return None
    return ComponentSpec(requested_name=name, **data)


def resolve_component(name: str) -> ComponentSpec:
    resolved = try_resolve_component(name)
    if resolved is not None:
        return resolved
    raise ValidationError(
        f"component '{name}' is not supported",
        "use `moltbox service list` to inspect available services or add a component mapping",
        component=name,
    )
