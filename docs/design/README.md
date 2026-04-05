# Gateway Design

This directory is the authoritative design package for `moltbox-gateway`.

Until these concepts are promoted back up into `remram`, this directory is the local source of truth for:

- the appliance shape
- the gateway and CLI contract
- the host and operations model
- the OpenClaw runtime integration model
- the backup and recovery model
- the delivery and migration shape

If an older document in `remram` conflicts with this directory, this directory wins for gateway implementation work.

## Reading Order

Read these in order:

1. `system-overview.md`
2. `current-state.md`
3. `target-state.md`
4. `cli-and-gateway.md`
5. `runtime-and-services.md`
6. `host-and-operations.md`
7. `backup-and-recovery.md`
8. `delivery-and-migration.md`

## Supporting Material

Use these alongside the design docs:

- `../decisions/` for formal direction changes
- `../reviews/` for evidence-heavy critique and audits
- `../plans/` for execution and validation plans
- `../runbooks/` for bounded operator procedures
- `../ai-context/` for AI-friendly summaries and import bundles

## Current Design Stance

The design package assumes all of these:

- appliance `dev` is gone
- OpenSearch is gone
- Postgres is not part of this box rebuild
- the appliance service set is `gateway`, `caddy`, `ollama`, `openclaw-test`, and `openclaw-prod`
- `prod` is a managed pet
- `test` is the appliance proving lane
- local development happens outside the appliance
- OpenClaw is managed through native config, plugin, skill, and backup surfaces
- the gateway is a thin appliance orchestrator, not the authority for OpenClaw internals
- ZFS-backed snapshots are a hard host precondition
