# Runtime And Services

This document defines the service plane and OpenClaw runtime model for the target appliance.

## Final Service Inventory

| Service | Source repo | Purpose | Notes |
| --- | --- | --- | --- |
| `gateway` | `moltbox-gateway` | control plane | one gateway only |
| `caddy` | `moltbox-services` | ingress and TLS | fronts gateway and runtimes |
| `ollama` | `moltbox-services` | local model serving | shared local provider |
| `openclaw-test` | `moltbox-services` + `moltbox-runtime` | proving runtime | mutation lane |
| `openclaw-prod` | `moltbox-services` + `moltbox-runtime` | protected runtime | managed pet |

Removed services:

- `openclaw-dev`
- `opensearch`

## Runtime Baseline Ownership

`moltbox-runtime` should define the desired baseline for:

- agent definitions
- provider definitions
- local model selection
- Together escalation wiring
- plugin allowlist and trust posture
- tool policy and routing

It should not try to encode every piece of live runtime drift as Git baseline.

## OpenClaw Runtime Rules

### Official mutation paths only

Normal runtime changes should happen through:

- native OpenClaw config application
- native plugin lifecycle
- native skill lifecycle
- native runtime backup and restore

### Test first

Capability changes land in `test` first.

Promotion to `prod` happens only after:

- config is validated
- health is good
- local model behavior is proven
- Together escalation is proven

### Prod protection

`prod` should not be treated like a throwaway container.

Expected posture:

- official install/config surfaces only
- restore point before risky change
- narrow, auditable change path

## Local Model Baseline

Keep the current known-good local inference shape:

- provider: `ollama`
- model: `mistral:7b-instruct-32k`
- context window: `32768`

Do not change model or container resource posture casually during the clean-box rebuild.

## Together Escalation

Together remains required, but it must not be a hidden gateway behavior.

Acceptable implementation shapes:

- pure OpenClaw provider configuration
- plugin-backed provider or maintenance capability
- a tracked Remram plugin plus runtime baseline config

Unacceptable shape:

- gateway hard-coded model routing

## Cortex MVP Overlay Shape

`remram-cortex` currently defines the Phase 0 and Phase 1 OpenClaw MVP seam like this:

- Phase 0 overlay keeps `memory.backend = qmd`
- Phase 0 overlay keeps `memory.qmd.sessions.enabled = false`
- Phase 0 overlay keeps `plugins.slots.contextEngine = legacy`
- Phase 0 overlay leaves `cortex-phase1-bridge` disabled
- Phase 1 overlay keeps `contextEngine = legacy`
- Phase 1 overlay sets `plugins.allow = ["cortex-phase1-bridge"]`
- Phase 1 overlay enables `cortex-phase1-bridge`
- Phase 1 overlay allows bounded prompt injection
- Phase 1 overlay starts the bridge in `dry-run` mode

That matches the target gateway model:

- gateway deploys services and approved packages
- OpenClaw stays owner of sessions, compaction, and runtime-local behavior
- Cortex attaches through bounded overlays and plugins instead of gateway replay ownership

## Plugin Trust Posture

The runtime baseline must make trust explicit.

That includes:

- explicit `plugins.allow`
- no untracked local plugin state in steady state
- tracked handling of any `behavior-guard` equivalent

## Caddy And Naming

Caddy should front:

- `moltbox-gateway`
- `moltbox-test`
- `moltbox-prod`

Direct host admin access on `moltbox-prime` stays separate from the application ingress names.

## Resource Policy

For the first clean rebuild:

- do not invent new CPU limits
- do not invent new memory limits
- keep the current known-good local model posture unless validation forces change

If resource limits are added later, they need explicit test evidence.

## Health Requirements

Each final service must have:

- a clear deploy path
- a clear healthy state
- log visibility through the CLI

Each runtime must prove:

- native status works
- chat works
- local provider usage is visible
- Together escalation is visible

## Later Work

Reasonable later extensions include:

- a narrow OpenClaw maintenance plugin
- richer runtime diagnostics
- additional data services such as Postgres

Those are not part of the clean appliance baseline.

## Related Documents

- [CLI And Gateway](cli-and-gateway.md)
- [Backup And Recovery](backup-and-recovery.md)
