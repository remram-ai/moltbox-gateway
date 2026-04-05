# Target State

This document defines the intended managed-pet appliance model.

## Target Appliance

Steady-state services:

- `gateway`
- `caddy`
- `ollama`
- `searxng`
- `openclaw-test`
- `openclaw-prod`

Not part of the target:

- `openclaw-dev`
- `opensearch`
- the Playwright browser detour

## Target Operating Model

- `prod` is a managed pet
- `test` is the proving lane
- gateway stays thin
- runtime-local mutation uses native OpenClaw surfaces
- replay and checkpoint are legacy-only, not normal operations
- service deploy syncs baseline state and compose assets; it is not replay reconstruction

## Target Web Model

Gold baseline web capability:

- `web_search`
- `web_fetch`

Native browser support is allowed as a future extension only after it is proven through official OpenClaw browser surfaces on this appliance.

## Target Recovery Model

Primary recovery path:

1. ZFS snapshot before risky change
2. risky change through official CLI
3. ZFS rollback if needed

Secondary runtime backup layer:

- native OpenClaw backup create and verify

Checkpoint is not backup. Replay is not recovery.

## Target Operator Model

Human admin:

- trusted administration path

AI test operator:

- CLI-driven `test` mutation path

AI prod operator:

- CLI-driven prod diagnostics
- no prod mutation rights

Break-glass admin:

- emergency use only
