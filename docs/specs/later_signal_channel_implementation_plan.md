# Signal Capability Implementation Plan

Archived future capability planning artifact.

This plan is preserved as a future reference implementation. The active MVP external ingress plan is defined in `docs/specs/07_discord_channel_implementation_plan.md`.

This document is the execution companion to `docs/specs/later_signal_channel.md`.

It finalizes the implementation sequence for the Signal capability without starting runtime or CLI implementation yet.

## 1. Planning Goal

Deliver Signal as a first-class Moltbox capability that:

- runs as one shared container named `signal`
- supports multiple Signal identities concurrently
- maps one Signal identity to each runtime environment
- preserves the normal OpenClaw runtime lifecycle
- is operated through the dedicated `moltbox signal ...` namespace
- validates the experimental CLI-to-MCP wrapper pattern

## 2. Finalized Architecture Summary

Confirmed architecture for this implementation run:

- one shared capability container: `signal`
- canonical asset root: `moltbox/containers/shared-services/signal/`
- one Signal identity per runtime environment
- canonical identity mapping:
  - `dev` runtime -> `dev` identity
  - `test` runtime -> `test` identity
  - `prod` runtime -> `prod` identity
- canonical durable state roots:
  - `${MOLTBOX_STATE_ROOT}/shared/signal/dev`
  - `${MOLTBOX_STATE_ROOT}/shared/signal/test`
  - `${MOLTBOX_STATE_ROOT}/shared/signal/prod`
- canonical log root:
  - `${MOLTBOX_RUNTIME_ROOT}/logs/signal/`
- canonical local-only daemon endpoint:
  - `http://signal:8080`
- canonical management surface:
  - `moltbox signal deploy|start|stop|restart|status|inspect|logs|health`

## 3. Planned Deliverables

This planning run targets these future implementation outputs:

- shared-service container assets for Signal
- runtime config wiring for per-environment Signal identity selection
- dedicated `moltbox signal ...` CLI namespace
- control-plane registration for the `signal` capability
- experimental generic CLI-to-MCP wrapper path for Signal
- operator guide at `docs/operator/signal.md`
- verification coverage for lifecycle, routing, pairing, and runtime execution

## 4. Workstreams

The implementation should be executed through these workstreams:

1. Upstream verification
2. Shared capability container and asset layout
3. Durable state and logging layout
4. Runtime identity wiring
5. CLI namespace and control-plane integration
6. Experimental MCP wrapper integration
7. Operator documentation and bootstrap runbook
8. Verification and rollout

## 5. Phase 0: Upstream Verification

Purpose:

- confirm the upstream contracts before writing runtime code

Tasks:

- verify the supported `signal-cli` daemon/API posture for containerized local-only use
- verify how multiple Signal accounts can be hosted concurrently in one deployment
- verify the OpenClaw Signal channel config contract for account selection and daemon endpoint wiring
- verify any upstream constraints on account registration, linking, and concurrent identity state

Outputs:

- explicit confirmed assumptions for the `signal-cli` deployment model
- explicit confirmed assumptions for OpenClaw channel config fields
- a short list of any upstream blockers that would require spec revision before coding

Stop condition:

- do not begin implementation work until the multi-account shared-container assumption is either confirmed or revised

## 6. Phase 1: Shared Capability Container and Assets

Purpose:

- establish the canonical shared `signal` capability artifact

Primary repo surfaces:

- `moltbox/containers/shared-services/signal/`
- `moltbox/config/`

Tasks:

- create the Signal shared-service asset directory
- define the compose template for one shared `signal` container
- define image and runtime command posture for `signal-cli`
- mount the canonical durable state roots into the container
- mount the canonical Signal log root into the container
- keep the daemon API internal-only on the Moltbox network
- define container restart and health posture suitable for appliance use

Expected outcomes:

- a repeatable shared-service render target for Signal
- a single managed container rather than one container per environment

## 7. Phase 2: Durable State and Logging Layout

Purpose:

- codify the host-owned state and logging contract

Primary repo surfaces:

- `moltbox/config/`
- `moltbox-cli/tools/src/moltbox_cli/layout.py`
- `moltbox-cli/tools/src/moltbox_cli/log_paths.py`

Tasks:

- define the durable state root `${MOLTBOX_STATE_ROOT}/shared/signal/`
- create per-identity subdirectory expectations for `dev`, `test`, and `prod`
- define what persistent data each identity directory stores
- define the canonical Signal log root `${MOLTBOX_RUNTIME_ROOT}/logs/signal/`
- ensure logs are host-mounted and not trapped only in the container filesystem
- define how identity-specific activity will be separated within the log root

Expected outcomes:

- a stable host layout for Signal state
- a stable host layout for Signal logs
- no ambiguity about what survives container replacement

## 8. Phase 3: Runtime Identity Wiring

Purpose:

- bind each runtime environment to its own Signal identity without creating a second execution path

Primary repo surfaces:

- `moltbox/config/openclaw/`
- `moltbox/config/container.env.example`
- runtime render logic in `moltbox-cli/tools/src/moltbox_cli/`

Tasks:

- define config inputs for the environment-to-identity mapping
- render `dev` with its own Signal account identity
- render `test` with its own Signal account identity
- render `prod` with its own Signal account identity
- keep the shared daemon endpoint fixed at the shared `signal` capability
- ensure Signal-originated requests still enter the normal OpenClaw lifecycle
- avoid introducing any second runtime ingress stack

Expected outcomes:

- environment-specific OpenClaw config that differs only by identity-specific Signal account values
- clear separation between shared capability lifecycle and runtime-specific channel ownership

## 9. Phase 4: CLI Namespace and Control-Plane Integration

Purpose:

- make Signal a first-class operable capability

Primary repo surfaces:

- `moltbox-cli/tools/src/moltbox_cli/cli.py`
- `moltbox-cli/tools/src/moltbox_cli/target_resolution.py`
- `moltbox-cli/tools/src/moltbox_cli/registry_bootstrap.py`
- `moltbox-cli/tools/src/moltbox_cli/deployment_service.py`
- `moltbox-cli/tools/src/moltbox_cli/primitive_runner.py`

Tasks:

- extend the CLI grammar to support a dedicated `signal` namespace
- register `signal` as a stable control-plane target
- implement canonical lifecycle verbs:
  - `deploy`
  - `start`
  - `stop`
  - `restart`
  - `status`
  - `inspect`
  - `logs`
  - `health`
- wire Signal through the render, deploy, inspect, and logging control-plane path
- define structured JSON responses consistent with the rest of the CLI

Expected outcomes:

- operators can manage Signal without raw Docker commands
- the control plane knows how to render, deploy, inspect, and diagnose the Signal capability

## 10. Phase 5: Experimental MCP Wrapper Integration

Purpose:

- validate the generic CLI-to-MCP wrapper pattern with Signal as the first capability

Primary repo surfaces:

- `moltbox-cli/tools/src/moltbox_cli/mcp_adapter.py`
- `moltbox-cli/tools/src/moltbox_cli/mcp_policy.py`
- any new wrapper library under `moltbox-cli/tools/src/moltbox_cli/`

Tasks:

- design a generic wrapper that invokes canonical CLI actions rather than reimplementing service logic
- expose policy-appropriate Signal capability verbs through MCP
- preserve behavior parity between local CLI and remote wrapper execution
- keep bootstrap-sensitive actions local-only where required
- avoid forcing existing capabilities to migrate to the new wrapper pattern in this run

Expected outcomes:

- Signal becomes the pilot capability for the wrapper pattern
- the repository gains a concrete reference for possible later standard updates

## 11. Phase 6: Operator Documentation and Bootstrap

Purpose:

- document normal operations and one-time setup before the capability is considered complete

Primary repo surfaces:

- `docs/operator/signal.md`
- `docs/operator/moltbox-cli-reference.md`
- `docs/operator/moltbox-manual-operations.md`

Tasks:

- add `docs/operator/signal.md`
- document Purpose, Access Method, Ports, CLI Operations, Runtime Behavior, Logs, Persistent State, Health / Debugging, Bootstrap Steps, and Failure Recovery
- document how each environment identity is provisioned and linked
- document which steps are bootstrap-only and which are steady-state CLI operations
- document which Signal actions are MCP-exposed and which remain local-only

Expected outcomes:

- the capability satisfies the operator-documentation requirement
- bootstrap does not remain tribal knowledge

## 12. Phase 7: Verification and Rollout

Purpose:

- prove the capability works end to end before rollout

Verification targets:

- the shared `signal` container deploys and reports healthy
- `dev`, `test`, and `prod` state directories remain isolated
- each runtime resolves to its own Signal identity
- inbound Signal traffic for each number reaches the correct runtime
- pairing gating works for first contact
- DM request and reply flow works
- the normal OpenClaw lifecycle is used
- Semantic Router escalation remains intact
- CLI `status`, `inspect`, `logs`, and `health` behave correctly
- MCP wrappers behave correctly for the verbs they expose

Rollout posture:

- bootstrap may require host access for first registration and linking
- steady-state operation must return to CLI-first management after bootstrap

## 13. Tests and Validation Coverage

Expected implementation coverage should include:

- CLI parser and grammar tests for `moltbox signal ...`
- control-plane target registration tests for `signal`
- render/deploy asset tests for the shared Signal capability
- state and log-path tests
- runtime config tests for per-environment identity mapping
- MCP policy and wrapper tests for Signal verbs
- operator verification steps in the Signal guide

## 14. CLI Gaps Discovered

The current gateway CLI and control-plane surfaces do not yet support this design cleanly.

Observed gaps:

- `moltbox` currently exposes only `tools`, `host`, and `runtime` domains
- there is no dedicated `signal` namespace in `moltbox_cli.cli`
- current target resolution only knows `tools`, `host` services, and runtime environments
- registry bootstrap has no `signal` target record
- log path mappings have no `signal` entry
- current shared-service lifecycle plumbing assumes `host` targets rather than dedicated capability namespaces
- host and runtime verbs do not currently include a generic `health` path for a new capability namespace
- the primitive layer has no Signal-specific render, deploy, or health primitives yet
- the MCP adapter exposes `tools`, `host_action`, and `runtime_action`, but no Signal capability wrapper surface
- the MCP policy model has no schema branch for `signal`
- the current CLI grammar docs would need updating to describe a capability namespace extension

These gaps should be turned into concrete implementation tasks rather than treated as incidental cleanup.

## 15. Out of Scope for This Run

The following are not part of the initial Signal implementation:

- Signal group-chat support
- attachment ingestion
- config mutation from chat
- migration of existing capabilities to the new MCP wrapper pattern
- any Remram app-layer UX beyond the Signal ingress channel itself

## 16. Completion Gate

Implementation should not be considered complete until:

- the spec and implementation still preserve Signal as runtime ingress only
- the shared-container and multi-identity design is operational
- CLI-first lifecycle management works for steady-state use
- state and logs follow the canonical host layout
- control-plane integration is complete
- the experimental Signal MCP wrapper path is functional where policy allows it
- `docs/operator/signal.md` exists
- verification passes for routing, pairing, runtime execution, and escalation behavior
