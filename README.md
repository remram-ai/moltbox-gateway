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
- the public CLI surface is intentionally lightweight: `gateway`, `service`, `test openclaw`, `prod openclaw`, `ollama`, and `secret`
- the gateway remains the service-plane orchestrator and still carries some legacy internal runtime routes during the transition
- scoped secrets and MCP tokens are still gateway-owned

Extra implementation commands currently exist for bootstrap and diagnostics:

- `moltbox bootstrap gateway` currently exists as a transition stub and is not fully implemented yet
- `moltbox gateway mcp-stdio`

Future service images and orchestration flows should come from `moltbox-services`, with gateway treated as a service there.
