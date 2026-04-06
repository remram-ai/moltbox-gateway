# Delivery And Migration

This document connects the design package to the execution package.

## Purpose

The design docs explain what the box should become. The plans explain how to get there.

This document gives the high-level migration shape so builders do not have to reverse-engineer the intended order from review notes.

## Migration Summary

1. capture the live box and preserve backups
2. rebuild the host on ZFS
3. restore SSH as fast as possible
4. simplify the CLI and gateway contract
5. simplify the service plane and runtime baselines
6. fix host ownership, identities, and hygiene
7. deploy the clean appliance
8. validate the full box end to end

## Workstream Map

| Design concern | Execution package |
| --- | --- |
| host rebuild and SSH recovery | `../runbooks/2026-04-04-zfs-rebuild-ssh-takeover-runbook.md` |
| box rebuild sequence | `../plans/2026-04-04-clean-moltbox-execution-plan.md` |
| validation gates | `../plans/2026-04-04-clean-moltbox-validation-plan.md` |
| builder kickoff loop | `../plans/2026-04-04-clean-moltbox-builder-prompt.md` |

## Repo-Level Deliverables

### `moltbox-gateway`

Must deliver:

- new CLI contract
- gateway route and handler support
- SSH wrapper policy
- docs and validation hooks

### `moltbox-services`

Must deliver:

- final six-service topology with `searxng`
- removed `openclaw-dev`
- removed OpenSearch
- updated Caddy assumptions

### `moltbox-runtime`

Must deliver:

- clean `test` and `prod` baselines only
- local Mistral baseline
- Together escalation through official runtime surfaces
- explicit plugin trust posture

### `remram-skills`

Touch only if a tracked plugin or skill change is actually required for Together or maintenance behavior.

### `remram-cortex`

Continue to own:

- Phase 0 and Phase 1 OpenClaw overlay assets
- `cortex-phase1-bridge` package
- later Cortex service contracts

Those assets should integrate into Moltbox through approved runtime baselines and official OpenClaw package flows, not gateway replay ownership.

## Acceptance Model

The build is not done because the repos look cleaner.

It is done only when:

- the host satisfies the ZFS precondition
- the final service set is healthy
- the CLI matches the new contract
- `test` and `prod` runtime validation succeeds through the supported surfaces
- local Mistral usage is visible
- Together escalation is visible
- ownership and SSH policy are correct
- backup and patching are configured

## Promotion Model

This repo is the current authority for the gateway design.

Once the design and implementation stabilize:

1. promote the durable concepts back into `remram`
2. retire or archive older conflicting `remram` docs
3. keep gateway-local execution material here

## Related Documents

- [System Overview](system-overview.md)
- [Target State](target-state.md)
- [Current State](current-state.md)
