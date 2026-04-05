# Moltbox Gateway

`moltbox-gateway` owns the Moltbox control plane and the `moltbox` CLI.

This repository is now the local authority for gateway design, implementation, and builder execution material. Broader platform concepts can be promoted back to `remram` later.

## Humans Start Here

Read these first:

- `docs/design/README.md`
- `docs/design/system-overview.md`
- `docs/design/current-state.md`
- `docs/design/target-state.md`

Then use:

- `docs/decisions/2026-04-04-openclaw-operating-model.md`
- `docs/reviews/2026-04-04-openclaw-operating-model-review.md`
- `docs/reviews/2026-04-04-cli-surface-review.md`
- `docs/plans/2026-04-04-clean-moltbox-execution-plan.md`
- `docs/plans/2026-04-04-clean-moltbox-validation-plan.md`
- `docs/runbooks/2026-04-04-zfs-rebuild-ssh-takeover-runbook.md`

## AIs Start Here

Import:

- `docs/ai-context/README.md`
- `docs/ai-context/overview.md`
- `docs/ai-context/current-state.md`
- `docs/ai-context/future-state.md`

Then import the themed context file that matches the task:

- `docs/ai-context/cli-gateway.md`
- `docs/ai-context/runtime-services.md`
- `docs/ai-context/host-ops.md`
- `docs/ai-context/implementation-plan.md`

Current implementation posture:

- the legacy Python implementation is preserved under `archive/legacy-implementation/`
- `archive/README.md` explains which archive materials are still useful and which assumptions must be ignored
- the host-installed `moltbox` binary is a thin HTTP client
- the long-running gateway server runs in a Docker container named `gateway`
- builds tag the gateway image as `moltbox-gateway:latest`
- the host CLI talks directly to the gateway over `http://127.0.0.1:7460`
- the gateway orchestrates service deploy, restart, runtime reload, runtime checkpoint, managed skill deploy and rollback, scoped secrets, and MCP tokens

Extra implementation commands currently exist for bootstrap and diagnostics:

- `moltbox gateway docker ping`
- `moltbox gateway docker run <image>`
- `moltbox gateway mcp-stdio`

Future service images and orchestration flows should come from `moltbox-services`, with gateway treated as a service there.
