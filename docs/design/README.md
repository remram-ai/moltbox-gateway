# Gateway Design

This directory is the current design package for `moltbox-gateway`.

It is the local source of truth for:

- the appliance shape
- the CLI and gateway contract
- the managed-pet OpenClaw model
- the recovery model
- the host and operator model
- the current web-tooling story

## Reading Order

1. `system-overview.md`
2. `current-state.md`
3. `target-state.md`
4. `cli-and-gateway.md`
5. `runtime-and-services.md`
6. `backup-and-recovery.md`
7. `host-and-operations.md`
8. `web-tooling.md`

## Current Design Stance

The design package assumes all of these:

- appliance `dev` is gone
- OpenSearch is gone
- the steady-state appliance service set is `gateway`, `caddy`, `ollama`, `searxng`, `openclaw-test`, and `openclaw-prod`
- `prod` is a managed pet
- `test` is the proving lane
- replay and checkpoint are not the normal `test` / `prod` lifecycle
- snapshots are first-class recovery primitives
- normal runtime mutation uses native OpenClaw surfaces
- the gateway is a thin appliance orchestrator, not the authority for OpenClaw internals
- baseline web capability is `web_search` + built-in `web_fetch`
- native `memory-core` is disabled in the default local lane
- the Playwright detour is removed from the current gold baseline

## Historical Material

Use:

- `../decisions/`
- `../reviews/`
- `../plans/`
- `../runbooks/`

for historical evidence, audits, and dated execution records.
