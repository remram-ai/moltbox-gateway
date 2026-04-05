# OpenClaw Operating Model Direction

Status: Proposed
Date: 2026-04-04

## Summary

This document records a directional architecture change, not a finalized target design.

The current Moltbox gateway model over-manages OpenClaw through runtime baseline restore, replay-heavy mutation, staged package installation, and direct runtime config manipulation. That direction should change.

The likely replacement is a thinner gateway that treats OpenClaw `prod` as a managed pet: official plugin, skill, and config flows only, backup-first recovery, no gateway-owned replay model, no appliance-level development environment, and a CLI reduced to service-plane management plus OpenClaw CLI access.

## Current Model Being Reconsidered

Today the gateway acts as both appliance orchestrator and de facto runtime state authority for OpenClaw:

- runtime deploy restores a baseline snapshot before replaying runtime history
- managed skill and plugin flows stage packages into gateway state and redeploy the runtime
- plugin install and removal paths mutate OpenClaw runtime config and host runtime state directly
- checkpoint and replay are used as a normal operating path, not just as migration or repair tools

Primary repo evidence:

- [`../../internal/orchestrator/manager.go`](../../internal/orchestrator/manager.go)
- [`../../internal/orchestrator/runtime_state.go`](../../internal/orchestrator/runtime_state.go)
- [`../../../remram/docs/concepts/gateway.md`](../../../remram/docs/concepts/gateway.md)
- [`../../../remram/docs/concepts/runtime.md`](../../../remram/docs/concepts/runtime.md)

## Locked Directional Decisions

These points are not final implementation detail, but they are the working direction:

- `moltbox-gateway` should remain the appliance orchestrator and service-plane manager, not the normal source of truth for OpenClaw runtime internals.
- `prod` OpenClaw should be treated as a managed pet. Primary recovery should be machine restore plus backup restore, not clean rebuild plus full redeploy.
- The appliance should not host a `dev` runtime. Development belongs in the coding agent's local environment.
- The appliance should keep `test` as the proving lane for new plugins, skills, and operational changes before promotion to `prod`.
- The Moltbox CLI should be reduced to service-plane management plus access to the official OpenClaw CLI, not act as a custom OpenClaw orchestration layer.
- `moltbox-runtime` should become the approved baseline and guardrail set for OpenClaw config, plugin enablement, provider routing, and maintenance roles, not a promise that `prod` is always reconstructed from zero.
- OpenClaw should own normal plugin lifecycle and runtime-local config behavior wherever upstream already supports it.
- replay and checkpoint should be demoted from steady-state operations to migration, repair, or legacy compatibility tools.
- backup and restore must become a first-class durability mechanism instead of being implied by checkpoint.

## Additional External Perspective

The review also considered a second, more OpenClaw-native take on the problem. This is intentionally not treated as authoritative by itself.

Validated pieces from upstream docs:

- OpenClaw plugins can register tools, commands, services, hooks, and HTTP routes.
- Plugins can expose optional tools that stay disabled until explicitly allowed in config.
- Hook decisions can require approval before tool execution.
- OpenClaw supports per-agent sandbox and tool policy overrides.
- OpenClaw now has native backup creation and verification commands.

Exploratory pieces that should inform design, but not settle it:

- packaging operational behavior as a dedicated maintenance plugin plus paired skill
- using a dedicated sandboxed `test` agent as the default appliance-side proving ground for new plugins or tools
- evaluating external ecosystem options such as NemoClaw for hardened deployments
- using community backup plugins as optional extensions beyond the baseline DR model

Official references:

- <https://docs.openclaw.ai/plugins/building-plugins#registering-agent-tools>
- <https://docs.openclaw.ai/tools/plugin>
- <https://docs.openclaw.ai/tools/multi-agent-sandbox-tools>
- <https://docs.openclaw.ai/cli/config>
- <https://docs.openclaw.ai/cli/backup>
- <https://docs.nvidia.com/nemoclaw/index.html>

## Open Design Questions

- Should the appliance `test` lane be a separate runtime, a sandboxed `test` agent, or both?
- What is the exact promotion workflow from local development to appliance `test` to appliance `prod`?
- What remains gateway-managed for skills that are not plugin-backed?
- What is the final restore contract between full-machine restore points, OpenClaw native backup restore, and OpenSearch recovery?
- How much legacy replay state should be preserved, migrated, or discarded?
- Is NemoClaw only a research track, or does it deserve a later hardened-deployment evaluation path?

## Consequences If This Direction Holds

- Gateway scope will shrink.
- Appliance `dev` goes away.
- `prod` OpenClaw changes should prefer official in-place install and config flows with a restore point taken first.
- The CLI becomes a service-plane manager and OpenClaw CLI access path instead of a custom runtime workflow engine.
- Backup, restore, and DR become explicit contracts instead of side effects of checkpoint logic.
- Existing replay-based flows need a migration and deprecation story.

## Promotion Criteria

Promote this record to `Accepted` only when the following are settled:

- the target gateway/runtime boundary is written down clearly
- the `test` and `prod` operating model is explicit and no longer assumes an appliance `dev` environment
- the backup and restore contract is decision-complete
- the CLI reduction to service-plane commands plus OpenClaw CLI access is explicit
- the maintenance model is chosen
- the fate of replay and checkpoint is explicit
- the migration path from the current gateway-managed runtime model is concrete enough to implement

## Supporting Review

- [`../reviews/2026-04-04-openclaw-operating-model-review.md`](../reviews/2026-04-04-openclaw-operating-model-review.md)
