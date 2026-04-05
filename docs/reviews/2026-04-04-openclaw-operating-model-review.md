# OpenClaw Operating Model Review

Status: Draft review
Date: 2026-04-04

This review is the evidence-heavy companion to the proposed decision record in [`../decisions/2026-04-04-openclaw-operating-model.md`](../decisions/2026-04-04-openclaw-operating-model.md).

How to read this document:

- Repo-grounded findings are the primary basis for the verdict.
- Recommendations are separated from current-state facts.
- One section incorporates an additional external perspective on a more OpenClaw-native model. That section is intentionally non-binding and is used to stress-test the rebuild direction, not to replace repo evidence.

## Topline Summary

The repo split and thin local CLI were basically right. The OpenClaw integration model was not.

Moltbox currently treats OpenClaw as something to reconstruct and replay from the outside. That is the wrong operating posture now that OpenClaw has stronger native plugin lifecycle, tool approval, multi-agent sandboxing, config validation, and backup support.

The right move is not to abandon the gateway. It is to stop using the gateway as a custom runtime-state machine for OpenClaw internals, stop carrying an appliance-level `dev` environment, and treat `prod` OpenClaw as a managed pet recovered primarily by backup and restore.

## Artifact 1: Executive Architecture Verdict

### Overall Verdict

The system got the appliance split mostly right and the OpenClaw operating model wrong.

### Core Strengths

- repository boundaries are mostly sensible: `moltbox-gateway` orchestrates, `moltbox-runtime` holds approved runtime guardrails, `moltbox-services` packages services, and `remram-skills` holds capability payloads
- the host CLI is thin and mostly forwards into the gateway over local HTTP, which is appropriate for a single builder/operator model
- the broad Moltbox appliance idea is valid: one appliance-local orchestrator, one operator surface, one place for secrets and deployment history

### Core Weaknesses

- gateway-owned replay has become a normal runtime lifecycle instead of a narrow recovery or migration tool
- checkpoint is overloaded as both convenience and implied durability
- OpenClaw plugin inventory and config are not treated as native runtime truth
- runtime config exists in two places and the gateway still mutates runtime-local files directly
- the appliance model carries too much environment complexity by trying to host `dev`
- redeploy is too heavyweight because the architecture assumes rebuild-through-replay

### Biggest Architectural Mistake

The gateway became a pseudo package manager and runtime reconciler for OpenClaw internals. That is why skill deploy, plugin install, config drift, and redeploy are all heavier than they need to be.

### Biggest Thing We Got Right

Keeping the host CLI thin and treating the gateway as a local appliance control plane was the right structural move.

### Rebuild Thesis

If rebuilding today, keep the repo split and the gateway appliance role, but make OpenClaw-native lifecycle and backup-first recovery the center of the `prod` runtime model.

Git-defined baselines should still define the service plane, approved runtime guardrails, and the appliance `test` lane, but they should not be the primary mechanism for reconstructing `prod` from zero. Development should happen locally outside the appliance, the appliance should expose `test` and `prod` only, and the CLI should mostly manage the service plane plus provide access to the official OpenClaw CLI.

## Artifact 2: Good Assumptions / Bad Assumptions Ledger

### Good Assumptions

- Assumption: the host CLI should be thin and mostly forward to a long-running gateway.
  - Why it helped: it keeps operator ergonomics stable while centralizing orchestration logic.
  - Keep, revise, or discard: Keep.
- Assumption: repo boundaries should separate control plane, runtime baseline, services, and skills.
  - Why it helped: the split is still understandable and avoids stuffing unrelated concerns into one repo.
  - Keep, revise, or discard: Keep.
- Assumption: a single strong builder/operator should be the first-class target.
  - Why it helped: it kept the architecture local-first and reduced premature multi-node complexity.
  - Keep, revise, or discard: Keep.

### Bad Assumptions

- Assumption: gateway replay history should be the authoritative record of OpenClaw runtime mutation.
  - Why it hurt: it turned the gateway into the owner of runtime state rather than the orchestrator of runtime state.
  - Keep, revise, or discard: Discard.
- Assumption: checkpoint plus replay is an acceptable steady-state runtime model.
  - Why it hurt: it makes redeploy slow, brittle, and harder to reason about than official lifecycle plus backup-first recovery.
  - Keep, revise, or discard: Discard.
- Assumption: plugin-backed capability should still flow through Moltbox-specific install and replay mechanics.
  - Why it hurt: it duplicates OpenClaw's native plugin lifecycle and creates more drift points.
  - Keep, revise, or discard: Discard.
- Assumption: checkpoint is close enough to backup for practical operations.
  - Why it hurt: checkpoint does not cover the actual DR story, especially when secrets, OpenSearch data, and off-host recovery matter.
  - Keep, revise, or discard: Discard.
- Assumption: the appliance should host a full `dev` / `test` / `prod` ladder.
  - Why it hurt: it drags coding-agent development into the appliance, expands the CLI and runtime surface, and makes the box own concerns that should stay local.
  - Keep, revise, or discard: Discard at the appliance level.
- Assumption: `prod` OpenClaw should be treated like a disposable runtime that is routinely rebuilt from clean state.
  - Why it hurt: it fights the way OpenClaw is actually operated, overvalues replay, and ignores the simpler backup-first model that fits a single-operator appliance.
  - Keep, revise, or discard: Discard.

### Assumptions Invalidated By Newer OpenClaw Capabilities

- Assumption: custom gateway orchestration is the main way to safely add tools and runtime behavior.
  - Why it is weaker now: OpenClaw plugins can register tools, commands, hooks, routes, and services directly, with approval gating and optional tool allowlisting.
  - Keep, revise, or discard: Revise heavily.
- Assumption: runtime isolation requires heavyweight runtime-level redeploy patterns.
  - Why it is weaker now: OpenClaw supports per-agent sandbox and tool policy configuration, which is a better fit for `test` and restricted operation than replay-heavy runtime rebuilds.
  - Keep, revise, or discard: Revise heavily.
- Assumption: custom snapshot logic is the only credible backup path.
  - Why it is weaker now: OpenClaw now has a native backup command and verify flow.
  - Keep, revise, or discard: Revise heavily.

### Assumptions That Still Hold

- Assumption: Moltbox still needs an appliance-level orchestrator.
  - Why it still holds: Docker services, repo syncing, secret storage, rendered assets, and operator workflows still need a control plane.
  - Keep, revise, or discard: Keep.
- Assumption: Git-defined baselines should matter.
  - Why it still holds: they are still the right place for service-plane configuration, approved runtime guardrails, and `test` defaults.
  - Keep, revise, or discard: Keep, but stop pretending they are the sole recovery path for `prod`.

## Artifact 3: OpenClaw Operating Model Review

### How OpenClaw Is Currently Being Managed

Current-state facts:

- runtime deploy restores a saved baseline into host runtime state and then replays runtime deploy history through the gateway orchestration path
- managed skill deploy stages a package into gateway state, appends replay metadata, redeploys the runtime, and re-applies content into OpenClaw
- managed plugin install snapshots runtime state, stages the package, uses helper flows to pack and install it, mutates runtime config, then redeploys and verifies
- plugin inventory is derived from gateway-managed state while skill inventory is queried natively from OpenClaw
- the runtime service template copies selected rendered config into the mutable OpenClaw state directory on startup, which is one reason config ownership is blurry

Evidence:

- [`../../internal/orchestrator/manager.go`](../../internal/orchestrator/manager.go)
- [`../../internal/orchestrator/runtime_state.go`](../../internal/orchestrator/runtime_state.go)
- [`../../../moltbox-services/services/openclaw-dev/compose.yml.template`](../../../moltbox-services/services/openclaw-dev/compose.yml.template)
- [`../../../moltbox-runtime/openclaw-dev/openclaw.json.template`](../../../moltbox-runtime/openclaw-dev/openclaw.json.template)
- [`../../../remram/docs/concepts/gateway.md`](../../../remram/docs/concepts/gateway.md)
- [`../../../remram/docs/concepts/runtime.md`](../../../remram/docs/concepts/runtime.md)

### What Is Wrong With That Model

- it duplicates native OpenClaw lifecycle with a second lifecycle owned by the gateway
- it creates mixed sources of truth for config, plugin state, and runtime contents
- it forces heavy redeploys for changes that should often be native plugin or config operations
- it encourages replay to act as a substitute for baseline clarity
- it confuses operational convenience with durability
- it treats `prod` like livestock even though the practical operating model is closer to a managed pet

### What Should Shift Into Native OpenClaw Config And Plugin Patterns

- plugin enablement and plugin-specific config
- tool exposure and optional tool approval policy
- runtime-local maintenance logic that genuinely belongs next to the agent
- per-agent sandbox and tool restrictions for `test` or restricted operation
- native backup creation and verification
- official OpenClaw plugin and skill install paths for `prod`

### What Should Stay In Gateway Orchestration

- appliance bootstrap and service deployment
- repo and tag selection for service-plane updates
- approved guardrail rendering and validation
- secret storage and host-side injection
- health, deployment history, and reconciliation reporting
- backup scheduling, restore-point handling, and retention policy at the appliance level
- CLI access brokering into official OpenClaw CLI surfaces when the operator is targeting appliance runtimes

### What Should Move Out Of Gateway Orchestration

- direct mutation of OpenClaw runtime config as a normal install step
- gateway-owned plugin inventory as the operational truth
- replay-based runtime reconstruction as the normal redeploy path
- packaging logic whose only purpose is to bypass OpenClaw's native plugin lifecycle
- appliance-level `dev` environment management
- custom CLI verbs that exist only because the gateway is reimplementing OpenClaw behavior

### Additional External Perspective: More OpenClaw-Native Agent Mode

This section incorporates a second take on the problem. It is not the primary basis for the verdict.

Validated against official docs:

- a single plugin can register tools, commands, hooks, services, and routes
- tools can be optional and require explicit allowlisting
- hooks can require approval before a tool call proceeds
- per-agent sandbox and tool policies allow a restricted `test` agent without changing the main agent
- native backup creation and verification reduce the need for custom snapshot logic

Useful design ideas from that perspective:

- treat a plugin plus skill pair as the default unit for operational capabilities
- test new operational capabilities in a sandboxed `test` agent or dedicated appliance `test` runtime before enabling them for the main agent
- use native OpenClaw backup first, then add external sync only as a second layer
- keep the gateway focused on install, config validation, backup orchestration, and status rather than workflow replay

Exploratory ecosystem signals, not baseline architecture:

- community backup tooling exists, but it often moves sensitive OpenClaw state off-host and should not be treated as the default DR answer
- NemoClaw looks relevant for hardened always-on deployments, but NVIDIA documents it as alpha and not for production yet

Sources:

- <https://docs.openclaw.ai/plugins/building-plugins#registering-agent-tools>
- <https://docs.openclaw.ai/tools/plugin>
- <https://docs.openclaw.ai/tools/multi-agent-sandbox-tools>
- <https://docs.openclaw.ai/cli/config>
- <https://docs.openclaw.ai/cli/backup>
- <https://docs.nvidia.com/nemoclaw/index.html>
- <https://docs.nvidia.com/nemoclaw/0.0.6/reference/commands.html>
- <https://clawhub.ai/plugins/%40clawbackup-ai%2Fclawbackup-plugin>
- <https://clawhub.ai/skills/claw-backup>

### Whether A Dedicated Maintenance Agent Or Plugin Is The Right Move

Recommendation:

- yes, probably, but keep it narrow

Reasoning:

- a dedicated maintenance plugin is a better place for runtime-local backup triggers, health and reporting helpers, and controlled maintenance tools than custom gateway replay logic
- a dedicated sandboxed maintenance or `test` agent is attractive for proving new operational capabilities without granting them to the main agent immediately
- the gateway should trigger, validate, and observe this behavior, not become the place where the behavior is reimplemented

### What The Ideal Update Path Should Look Like

- Service-plane changes should render desired state from pinned repo inputs, restart only changed services, verify health, and record deployment provenance in the gateway.
- `prod` OpenClaw changes should take a restore point first, then use official OpenClaw install and config paths, verify health, and record what changed.
- `test` should remain the place for proving new plugins, skills, and maintenance flows before they touch `prod`.

That path should not depend on restoring a mutable runtime tree and replaying historical install events unless recovering legacy state.

## Artifact 4: Backup / Restore Architecture

### Source-Of-Truth Map

- Git repos and pinned tags: desired service-plane configs, approved runtime guardrails, `test` defaults, and skill/plugin source packages
- gateway state: deployment history, rendered assets, replay metadata kept only for legacy compatibility, and host-side operational state
- gateway secrets store plus its master key: authoritative secret recovery material
- `prod` OpenClaw live state plus native backup archives: authoritative recovery material for `prod`
- OpenSearch snapshots or volume backups: authoritative search and index recovery

### Backup Classes

- Appliance backup
  - nightly host image or filesystem snapshot covering gateway state, secrets, rendered assets, service config, and appliance runtime state
- Restore point backup
  - pre-change snapshot or restore point taken before risky `prod` changes
- Runtime backup
  - scheduled `openclaw backup create --verify` archives for appliance runtimes
- Data-service backup
  - OpenSearch snapshot or volume-level backup
- Off-host retention
  - encrypted copy of all of the above to a second storage target

### Restore Sequence

1. Restore the appliance from the latest machine backup or pre-change restore point.
2. Restore gateway secrets and the secret master key if they were not already part of the machine restore.
3. Restore OpenClaw from the latest native backup archive if the machine restore is older than the desired runtime state.
4. Restore OpenSearch data if it is not already covered by the machine restore or needs a newer snapshot.
5. Run OpenClaw verification and gateway health checks.
6. Reconcile legacy replay metadata only if needed for old installations.

Fallback path only:

7. If machine restore is unavailable, rebuild the service plane from pinned inputs and then restore OpenClaw and OpenSearch from backups.

### Disaster Recovery Posture

- checkpoint is not enough
- replay is not enough
- appliance-level backup alone is not enough
- runtime-level backup alone is not enough

The minimum viable DR story is a backup-first pet model: nightly appliance backup, pre-change restore points for `prod`, native OpenClaw runtime backup, separate OpenSearch recovery, and off-host secret-safe retention.

### Minimum Viable Implementation

- take nightly machine backups of the Moltbox appliance
- take pre-change restore points before risky `prod` changes
- back up `/srv/moltbox-state`
- back up `/var/lib/moltbox/history.jsonl`
- back up `/var/lib/moltbox/secrets` and `/var/lib/moltbox/secrets/master.key`
- run `openclaw backup create --verify` on a schedule
- back up or snapshot OpenSearch separately
- copy all backup artifacts off-host

### Later Enhancements

- automated restore drills
- backup catalog and checksum reporting in the gateway
- immutable retention buckets
- optional external sync plugins or skills for convenience, not baseline correctness

## Artifact 5: Rebuild Blueprint

### Repo Responsibilities

- `moltbox-gateway`: appliance orchestration, service-plane management, secrets, deployment provenance, backup scheduling, restore-point handling, and operator access to runtime CLI surfaces
- `moltbox-runtime`: approved OpenClaw guardrails, plugin enablement defaults, agent definitions, tool policy, and maintenance roles
- `moltbox-services`: service packaging and compose or runtime wrappers
- `remram-skills`: skill content and plugin-adjacent capability packaging
- `remram`: platform architecture, operator concepts, and canonical cross-repo docs

### Runtime Model

- no appliance-level `dev` runtime
- local development belongs in the coding agent's local environment
- appliance `test` is the proving lane for new plugins, skills, and maintenance flows
- appliance `prod` is a managed pet changed only through official OpenClaw install and config surfaces
- OpenClaw-native plugin lifecycle is the default for plugin-backed behavior
- per-agent sandbox and tool policy are used where isolation is needed
- replay exists only for legacy import, migration, or break-glass repair

### Gateway Model

- thin service-plane orchestrator with strong provenance
- backup and restore coordinator for the appliance
- operator access path into official OpenClaw CLI surfaces
- no gateway-owned runtime package manager behavior for normal OpenClaw changes
- no direct runtime config edits as a steady-state install mechanism

### CLI Model

- keep the thin host CLI
- remove the appliance `dev` environment from the operator surface
- keep service-plane commands
- make runtime operations mostly an access path to the official OpenClaw CLI for `test` and `prod`
- remove commands whose whole reason to exist is gateway replay management or custom OpenClaw orchestration
- add commands for backup status, restore points, validation, and health that reflect the thinner model

### Backup Model

- nightly appliance backup plus pre-change restore points plus runtime-native backup plus data-service backup
- `prod` recovery is backup-first, not rebuild-first
- checkpoint is no longer treated as durability

### Maintenance Model

- narrow maintenance plugin or service for runtime-local operations
- optional sandboxed `test` agent as a proving lane for operational capabilities inside the appliance
- no appliance-level development environment

### Update / Redeploy Flow

- service-plane updates: select pinned inputs, render service definitions, restart only changed services, verify health, record deployment
- `prod` OpenClaw changes: take a restore point, use official OpenClaw install and config commands, verify health, record promotion
- recovery: restore the appliance and runtime from backups first; use clean rebuild only as a fallback path

### Migration Path From Current Architecture

- remove appliance `dev` from the target model and operator vocabulary
- freeze growth of replay-heavy features
- strip the CLI down to service-plane commands plus official OpenClaw CLI access
- move plugin-backed capabilities into approved guardrails plus native plugin lifecycle
- preserve legacy replay only long enough to migrate existing installations
- delete checkpoint and replay paths once backup-first recovery and migration are proven

## Artifact 6: Action Plan

### Phase 0: Immediate Corrections

- Remove appliance-level `dev` from the target architecture and CLI direction.
  - Why: development belongs on the coding agent's local machine, not on the appliance.
  - Expected benefit: smaller runtime and CLI surface.
  - Risk level: Low.
  - Prerequisite: Yes.
- Reframe `prod` as a managed pet with backup-first recovery.
  - Why: it matches how OpenClaw actually wants to be operated.
  - Expected benefit: simpler and more honest recovery model.
  - Risk level: Low.
  - Prerequisite: Yes.
- Change docs so checkpoint is no longer presented as backup.
  - Why: the current framing is operationally misleading.
  - Expected benefit: clearer durability posture.
  - Risk level: Low.
  - Prerequisite: Yes.
- Stop treating gateway-managed plugin state as the long-term source of truth.
  - Why: it conflicts with native OpenClaw inventory.
  - Expected benefit: fewer mixed-truth bugs.
  - Risk level: Low.
  - Prerequisite: Yes.

### Phase 1: Simplifications

- Strip the CLI down to service-plane commands plus OpenClaw CLI access for appliance runtimes.
  - Why: the gateway should stop pretending to be OpenClaw.
  - Expected benefit: simpler operator surface and less duplicated behavior.
  - Risk level: Medium.
  - Prerequisite: Yes.
- Move more approved plugin enablement and config guardrails into `moltbox-runtime`.
  - Why: approved baseline beats replay, even if `prod` is recovered from backup rather than rebuilt from scratch.
  - Expected benefit: clearer boundary between guardrails and live runtime state.
  - Risk level: Medium.
  - Prerequisite: Yes.
- Add restore-point handling plus native OpenClaw config validation and backup commands into the gateway operating flow.
  - Why: upstream already supports this.
  - Expected benefit: less custom control-plane logic.
  - Risk level: Medium.
  - Prerequisite: Yes.

### Phase 2: Structural Changes

- Replace gateway-managed plugin and skill orchestration for plugin-backed capabilities with native OpenClaw lifecycle.
  - Why: the current model is the main architectural problem.
  - Expected benefit: lower complexity and faster operations.
  - Risk level: Medium-High.
  - Prerequisite: Yes.
- Introduce a narrow maintenance plugin or service and decide whether to add a default sandboxed `test` agent profile.
  - Why: runtime-local operations should live closer to OpenClaw.
  - Expected benefit: cleaner maintenance posture.
  - Risk level: Medium.
  - Prerequisite: Yes.

### Phase 3: Nice-To-Have Improvements

- Evaluate NemoClaw as a separate hardened-deployment track.
  - Why: it may become relevant for stronger isolation later.
  - Expected benefit: possible future security posture improvement.
  - Risk level: Medium because the project is alpha.
  - Prerequisite: No.
- Evaluate optional ecosystem backup tooling after the baseline DR path is complete.
  - Why: convenience layers should come after correctness.
  - Expected benefit: easier off-host sync and retention.
  - Risk level: Medium.
  - Prerequisite: No.

## Non-Negotiable Recommendations

- Treat `prod` OpenClaw as a managed pet.
- Remove appliance-level `dev`.
- Make the CLI a service-plane manager plus OpenClaw CLI access path.
- Stop using replay as the normal OpenClaw operating model.
- Stop treating checkpoint as backup.
- Make `moltbox-runtime` the approved guardrail home for normal OpenClaw behavior.
- Make OpenClaw native plugin lifecycle the default path for plugin-backed capability.
- Keep the gateway, but shrink it hard.

## What I Would Do If This Were My System

I would freeze further investment in replay-heavy OpenClaw orchestration immediately.

Then I would build a thin native path in parallel:

- approved runtime guardrails
- nightly appliance backup and pre-change restore points
- native config validation
- native plugin lifecycle
- native backup scheduling
- one narrow maintenance plugin or service
- one appliance `test` lane for proving new operational capabilities

I would keep NemoClaw and community backup tools in a separate research lane until the baseline Moltbox model is already clean.

## Final Answer

You were basically right about the repo split and the local gateway appliance model.

You were wrong about the OpenClaw integration model itself. The complexity is not just extra polish that accumulated on a good core idea. The complexity is mostly a symptom of the wrong control boundary and of treating OpenClaw like livestock when it should be managed more like a pet.
