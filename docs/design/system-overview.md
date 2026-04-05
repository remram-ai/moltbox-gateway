# System Overview

This document defines the Moltbox appliance boundary and repository responsibilities.

## Appliance Boundary

The appliance consists of:

- host `moltbox-prime`
- ingress names `moltbox-gateway`, `moltbox-test`, and `moltbox-prod`
- system-owned state, logs, secrets, and repo checkouts
- one thin service plane managed through the `moltbox` CLI

Current steady-state service inventory:

- `gateway`
- `caddy`
- `ollama`
- `searxng`
- `openclaw-test`
- `openclaw-prod`

Not part of the target appliance:

- `openclaw-dev`
- `opensearch`
- the old `playwright` browser detour service

## Design Principles

Thin gateway:

- gateway orchestrates services and host-side policy
- gateway does not own replay-based runtime mutation

Managed-pet production:

- `prod` is protected
- restore is snapshot-first and backup-aware, not rebuild-first

One proving lane:

- `test` is the only appliance-side proving lane

Native OpenClaw lifecycle:

- runtime-local mutation uses native OpenClaw config and backup surfaces
- service deploy is not the same thing as runtime replay

System-owned operations:

- runtime operation must not depend on a personal account or home directory

## Repository Responsibilities

| Repo | Role |
| --- | --- |
| `moltbox-gateway` | CLI, gateway API, orchestration, SSH role policy, current docs |
| `moltbox-services` | service definitions and service templates |
| `moltbox-runtime` | desired OpenClaw baseline for `test` and `prod` |
| `remram-skills` | future tracked extension source when needed |
| `remram-cortex` | Cortex overlays and future integration inputs |

## Source Of Truth Map

| Concern | Primary source |
| --- | --- |
| public CLI contract | `cli-and-gateway.md` |
| service inventory and behavior | `runtime-and-services.md` plus `moltbox-services` |
| runtime baseline | `moltbox-runtime` plus `runtime-and-services.md` |
| live appliance state | host ZFS-backed state and native OpenClaw runtime state |
| durable recovery | ZFS snapshots plus verified backups |

Git is the source of desired baseline. The machine is the source of current live state.

## Control Paths

Human operator path:

```text
workstation
  -> SSH
    -> host-installed moltbox CLI
      -> local gateway API
```

AI operator path:

```text
restricted SSH identity
  -> forced-command wrapper
    -> host-installed moltbox CLI
      -> local gateway API
```

Native runtime path:

```text
moltbox test openclaw <native args>
moltbox prod openclaw <native args>
```

## Authority Boundaries

The gateway owns:

- service deployment and restart orchestration
- gateway update
- secret storage and injection
- deployment history
- snapshot hooks
- SSH role enforcement

OpenClaw owns:

- runtime-local config interpretation
- native config mutation
- native runtime backup operations
- model/provider behavior
- runtime-local extension behavior
