# System Overview

This document defines the Moltbox gateway system boundary and repository ownership model.

## Purpose

The appliance exists to provide a small, operable local AI box with:

- one control plane
- one ingress layer
- one shared local model service
- two OpenClaw runtimes
- real recovery semantics

This is not a generic cluster. It is one strong-operator appliance.

## System Boundary

The appliance consists of:

- metal host `moltbox-prime`
- gateway-facing name `moltbox-gateway`
- runtime names `moltbox-test` and `moltbox-prod`
- system-owned state, secrets, logs, and repo checkouts
- a small service plane managed through the `moltbox` CLI

Final service inventory:

- `gateway`
- `caddy`
- `ollama`
- `openclaw-test`
- `openclaw-prod`

Not part of the target appliance:

- `openclaw-dev`
- `opensearch`
- Postgres

Postgres belongs to the future Cortex work, not this appliance reset.

## Design Principles

### Thin gateway

The gateway owns appliance orchestration, not OpenClaw internals.

### Managed-pet production

`prod` is not treated like disposable livestock. Recovery is backup-first and snapshot-first.

### One proving lane

`test` is the only appliance-side proving lane. Local development belongs on a coding agent workstation or local dev environment, not on the appliance.

### Native OpenClaw lifecycle

OpenClaw config, plugin lifecycle, skill lifecycle, and backup flows should stay on official OpenClaw surfaces wherever possible.

### System-owned operations

Runtime operation must not depend on a personal login account, personal home directory, or ad hoc shell habits.

### Minimal moving parts

The box should be explainable in one screen. Complexity that only exists to support other complexity is a design failure.

## Repository Responsibilities

| Repo | Role | Authority |
| --- | --- | --- |
| `moltbox-gateway` | CLI, gateway API, orchestration, SSH policy, docs in this repo | appliance control plane |
| `moltbox-services` | service definitions and service templates | desired service plane |
| `moltbox-runtime` | OpenClaw baseline config for `test` and `prod` | desired runtime baseline |
| `remram-skills` | tracked skill and plugin source | capability source |
| `remram-cortex` | Cortex overlays, bridge packages, and later service contracts | Cortex integration inputs |
| `remram` | broader platform architecture and promotion target later | upstream documentation home later |

## Source Of Truth Map

| Concern | Primary source |
| --- | --- |
| public CLI contract | `moltbox-gateway/docs/design/cli-and-gateway.md` |
| gateway implementation design | `moltbox-gateway/docs/design/` |
| desired service layout | `moltbox-services` |
| desired OpenClaw baseline | `moltbox-runtime` plus approved overlays from `remram-cortex` when used |
| plugin and skill source | `remram-skills` and approved OpenClaw package artifacts |
| live appliance state | ZFS-backed host state plus OpenClaw native runtime state |
| durable recovery artifacts | ZFS snapshots, appliance backups, OpenClaw backups |

Git is the source of desired configuration. The machine is the source of current live state. Backups are the source of durability.

## Control Paths

Human operator path:

```text
Workstation
  -> SSH
    -> host-installed moltbox CLI
      -> local gateway API
        -> managed services
```

AI operator path:

```text
Restricted SSH identity
  -> forced-command wrapper
    -> host-installed moltbox CLI
      -> local gateway API
```

OpenClaw native path:

```text
moltbox test openclaw <native args>
moltbox prod openclaw <native args>
```

The normal operator path is not direct Docker usage.

## Authority Boundaries

The gateway owns:

- CLI contract
- service deployment and restart orchestration
- appliance status and logs
- secret storage and injection
- gateway self-update
- host-side policy enforcement for AI identities
- deployment history and validation surfaces

OpenClaw owns:

- runtime-local config interpretation
- plugin lifecycle
- skill lifecycle
- native runtime health
- native runtime backup creation
- model/provider behavior inside the runtime

## Related Documents

- [Current State](current-state.md)
- [Target State](target-state.md)
- [CLI And Gateway](cli-and-gateway.md)
- [Runtime And Services](runtime-and-services.md)
- [Host And Operations](host-and-operations.md)
