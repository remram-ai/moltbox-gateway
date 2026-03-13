# Signal Channel Adapter

Archived future capability design.

This document is preserved as a future reference implementation. The active MVP external ingress path is defined in `docs/specs/07_discord_channel.md`.

This document defines the canonical design intent for Signal as a baseline Moltbox ingress channel in `remram-gateway`.

It is also the capability-integration contract for Signal inside the gateway repository and is intended to conform to the platform standard `docs/standards/capability-integration.md` in the main `remram` repository.

## 1. Overview

Signal support is part of the baseline Moltbox runtime experience.

It is not:

- an optional `remram-app` feature
- a new architectural layer
- a parallel bot runtime

It is:

- a runtime ingress channel owned by `remram-gateway`
- a first-class managed capability in the Moltbox control plane
- a temporary human/operator interaction surface before a fuller Remram app layer exists
- an OpenClaw-backed channel that must feed the existing runtime entry and orchestration path

The intended request path is:

```text
Signal
-> shared signal capability container
-> Signal identity selected by destination phone number
-> OpenClaw Signal channel in the matching runtime environment
-> OpenClaw reply lifecycle / Semantic Router
-> models, tools, and Cortex integrations
```

Signal remains a transport surface only. Gateway runtime authority, OpenClaw orchestration, and Cortex knowledge policy stay where they already belong.

## 2. Goals

The Signal channel adapter must satisfy these goals:

- let operators or early users DM dedicated Signal bot numbers
- run locally on Moltbox
- avoid any public inbound Signal webhook surface
- preserve the normal OpenClaw reply lifecycle and existing model escalation behavior
- satisfy the platform capability-integration contract before implementation begins
- support multiple Signal identities concurrently through one shared capability container
- fit a CLI-first Moltbox operational model for steady-state use
- keep the implementation small enough for MVP and avoid feature drift into a general messaging product

## 3. Architectural Boundaries

The Signal adapter does:

- accept inbound Signal direct messages through OpenClaw's Signal integration
- route those messages by Signal identity to the matching runtime environment
- hand those messages to the same runtime path used by other OpenClaw chat surfaces
- return replies through the same Signal channel
- preserve normal session, routing, and escalation behavior

The Signal adapter does not:

- create a second orchestration engine
- bypass the OpenClaw lifecycle with custom message handling
- own memory policy or durable knowledge policy
- grant elevated privileges because the request arrived through Signal
- expose group chat as part of MVP
- accept attachments as part of MVP
- allow runtime or config mutation through chat

This spec follows the same principle established in `06_semantic_router.md`: channel choice must not create a parallel execution stack.

## 4. Capability Framing

Under the capability integration standard, Signal is a first-class gateway capability.

That means the repository must treat Signal as:

- a deployable, long-lived subsystem integration
- a control-plane managed target with stable identity
- a capability with its own management namespace
- a capability with durable state and visible logs

The Signal management surface exists for lifecycle, inspection, diagnostics, and policy-aware remote control.

It does not replace the Signal service interface used by OpenClaw itself.

Correct pattern:

```text
moltbox signal ... -> deploy, inspect, logs, health, restart
OpenClaw Signal channel -> ordinary application traffic
```

Incorrect pattern:

```text
moltbox signal ... -> acts like the ordinary Signal chat endpoint
```

## 5. Shared Container Deployment Model

Signal uses a local daemon model backed by `signal-cli`.

In conformance with the capability integration standard, containerized deployment is the default posture for this capability.

Canonical deployment identity:

- capability target: `signal`
- container/service name: `signal`

Canonical deployment asset root:

```text
moltbox/containers/shared-services/signal/
```

Required posture:

- Signal runs as one shared capability container
- the shared container supports multiple Signal accounts concurrently
- the service API remains local-only
- no public inbound webhooks are introduced
- no host port is published to the LAN or internet for the Signal daemon API
- persistent account and session state lives outside the container image and outside the git checkout

This architecture explicitly does not create per-environment Signal containers such as:

- `signal-dev`
- `signal-test`
- `signal-prod`

The canonical network posture is an internal-only shared service on the Moltbox Docker network, reached by runtimes through a local-only URL such as:

```text
http://signal:8080
```

Literal runtime loopback is not the canonical deployment assumption for this implementation run.

## 6. Multi-Identity Model

MVP requires dedicated Signal identities rather than a single shared bot number.

Each runtime environment gets its own Signal identity and phone number.

Example identity set:

- `dev` -> `+1406DEV`
- `test` -> `+1406TEST`
- `prod` -> `+1406PROD`

These values are placeholders for real provisioned numbers, not literal phone-number syntax requirements.

This spec explicitly does not adopt the older archive posture of linking an operator's existing personal Signal account.

Required identity rules:

- one dedicated Signal identity per runtime environment
- one shared Signal capability container hosts those identities concurrently
- identities are used only for Remram interaction
- operators contact the environment-specific bot identity by DM
- account credentials, linked-device state, and session data remain isolated per identity

## 7. Identity-to-Runtime Mapping and Failure Scope

Each OpenClaw environment owns exactly one Signal identity.

Canonical mapping model:

- `dev` runtime -> `dev` Signal identity
- `test` runtime -> `test` Signal identity
- `prod` runtime -> `prod` Signal identity

Required rules:

- inbound Signal routing is determined by the destination phone number or account identity
- each runtime renders Signal config for exactly one identity
- a runtime must never bind to another environment's identity
- the shared Signal capability container is appliance-scoped, but identity ownership is runtime-scoped

Request-path role:

- Signal is ingress only
- the shared Signal capability container terminates transport concerns
- the matching OpenClaw runtime consumes the request through its normal lifecycle
- the rest of the runtime stack remains unchanged

Failure modes:

- if one identity's credentials or session state break, that environment loses Signal ingress while other identities may remain healthy
- if the shared Signal container is unavailable, all Signal ingress is unavailable
- non-Signal runtime surfaces remain intact when Signal is degraded

## 8. Runtime-Resolved Channel Posture

Each runtime renders the same Signal channel posture except for the identity-specific account value.

Example resolved runtime posture:

```json
{
  "channels": {
    "signal": {
      "enabled": true,
      "account": "+1RUNTIME_SIGNAL_NUMBER",
      "httpUrl": "http://signal:8080",
      "autoStart": false,
      "dmPolicy": "pairing",
      "groupPolicy": "disabled",
      "configWrites": false,
      "sendReadReceipts": false,
      "ignoreAttachments": true,
      "dmHistoryLimit": 20,
      "textChunkLimit": 4000,
      "chunkMode": "newline"
    }
  }
}
```

Runtime examples:

- `dev` renders its own `account` value
- `test` renders its own `account` value
- `prod` renders its own `account` value

Shared runtime expectations:

- `httpUrl` points at the shared local-only Signal capability endpoint
- only the account identity differs per environment
- Signal config remains runtime-local even though the Signal capability container is shared

The exact upstream OpenClaw key names and config file placement must be verified before implementation. The authoritative requirement in this spec is the posture, not the assumption that the above JSON maps one-to-one to current upstream config schema.

Within this repository, implementation should converge the existing placeholders in:

- `moltbox/config/openclaw/channels.yaml`
- `moltbox/config/container.env.example`
- per-runtime rendered OpenClaw config

It should not introduce a second independent Signal policy source.

## 9. Management Namespace and Lifecycle

In conformance with the capability integration standard, Signal should have a dedicated Moltbox management namespace.

The canonical CLI surface should be:

```text
moltbox signal deploy
moltbox signal start
moltbox signal stop
moltbox signal restart
moltbox signal status
moltbox signal inspect
moltbox signal logs
moltbox signal health
```

Optional but compatible verbs may include:

- `rollback`
- `test`
- `debug`

This dedicated namespace manages the shared Signal capability itself.

It does not replace runtime deployment:

- `moltbox runtime <environment> deploy` still renders OpenClaw runtime config
- runtime deployment remains responsible for that environment's identity mapping
- Signal lifecycle and Signal-enabled runtime wiring remain distinct management concerns

Operational changes must happen through managed redeploy of the Signal capability inputs and assets rather than by mutating a live daemon or container.

Steady-state operation is CLI-driven.

## 10. Logging and Persistent State

Signal must define canonical durable locations for both logs and state.

Canonical log root:

```text
${MOLTBOX_RUNTIME_ROOT}/logs/signal/
```

Example default:

```text
~/Moltbox/logs/signal/
```

Logging requirements:

- logs are mounted from the host
- logs must not exist only inside a writable container filesystem
- `moltbox signal logs` must expose the normal operator retrieval path
- the control plane must know the canonical Signal log location
- log output must make identity-specific activity distinguishable

Canonical durable state roots:

```text
${MOLTBOX_STATE_ROOT}/shared/signal/dev
${MOLTBOX_STATE_ROOT}/shared/signal/test
${MOLTBOX_STATE_ROOT}/shared/signal/prod
```

Each identity directory stores:

- account credentials
- linked-device state
- Signal session data

Redeploying the shared Signal container must not destroy this state.

## 11. Control-Plane and Experimental MCP Wrapper

Signal must be registered as a first-class control-plane capability target.

Required control-plane posture:

- stable target identity: `signal`
- canonical deployment asset root for the capability
- structured `status`, `inspect`, `logs`, and `health` behavior
- control-plane awareness of the canonical log root and state roots
- clear ownership boundary between Signal capability lifecycle and runtime channel consumption

Signal is also the first capability for an experimental generic CLI-to-MCP wrapper pattern:

```text
CLI command
-> control plane
-> MCP wrapper library
-> editor / agent tools
```

Rules for this implementation run:

- MCP wraps the canonical `moltbox signal ...` management path
- the wrapper pattern is experimental
- existing capabilities do not need to migrate to this pattern yet
- Signal is the validation capability for the pattern
- policy may restrict which verbs are exposed remotely
- local CLI behavior and remote MCP behavior must not silently drift
- sensitive bootstrap actions may remain local-only even if read-only diagnostics are remotely exposed

The implementation-facing operator docs must explicitly state which Signal actions are MCP-exposed and which remain local-only.

## 12. Security Model

MVP security posture is intentionally narrow:

- dedicated environment-specific bot numbers only
- one shared local-only Signal capability container
- local-only daemon API
- pairing-gated DM access
- Signal groups disabled
- config writes from chat disabled
- attachments ignored
- read receipts disabled
- channel selection does not grant privilege elevation

Signal must remain a standard-privilege ingress path unless future policy explicitly changes that through normal gateway controls.

## 13. Operator Documentation Requirement

Signal is not complete under the capability integration standard unless the gateway repository ships an operator guide at:

```text
docs/operator/signal.md
```

That guide must include:

- Purpose
- Access Method
- Ports
- CLI or control-plane Operations
- Runtime Behavior
- Logs
- Persistent State
- Health / Debugging
- Bootstrap Steps
- Failure Recovery

This spec defines the design contract. The operator guide is the required operational companion document.

## 14. Bootstrap Exception

Initial setup may require direct host access.

Acceptable bootstrap steps include:

- `ssh` into the Moltbox host
- temporary SSH tunneling when needed for first-time setup
- account registration for each environment identity
- device linking for each environment identity
- first daemon initialization

This exception is for bootstrap only.

Steady-state expectations are:

- CLI-managed lifecycle through `moltbox signal ...`
- control-plane visible `status`, `inspect`, `logs`, and `health`
- documented manual steps only where upstream Signal registration or linking still requires human interaction

Bootstrap and recovery procedures must be written into `docs/operator/signal.md`, not left implicit in this spec.

## 15. Explicit Gaps Relative to the Final Architecture

The current repository does not yet have the primitives needed for full Signal capability compliance.

Known gaps:

- no dedicated `moltbox signal ...` namespace exists yet
- the current CLI grammar only supports `tools`, `host`, and `runtime`
- no `signal` control-plane target identity exists yet
- there is no canonical shared-service asset directory for Signal under `moltbox/containers/shared-services/signal/`
- there is no canonical Signal log-root contract implemented yet at `${MOLTBOX_RUNTIME_ROOT}/logs/signal/`
- there is no per-identity durable-state layout implemented yet under `${MOLTBOX_STATE_ROOT}/shared/signal/<environment>`
- runtime render logic has no explicit per-environment Signal identity mapping yet
- there is no generic CLI-to-MCP wrapper library yet
- MCP policy and adapter surfaces do not yet know about a `signal` capability namespace
- the checked-in Signal config placeholders are not sufficient for a multi-identity dedicated-bot deployment
- `docs/operator/signal.md` does not exist yet

## 16. Companion Implementation Plan

Detailed execution sequencing for this spec is tracked in:

```text
docs/specs/later_signal_channel_implementation_plan.md
```

That plan is the implementation companion to this design contract.

## 17. Verification Criteria

Signal MVP is not complete until all of the following are true:

- the shared `signal` capability deploys repeatably through the canonical Moltbox management surface
- the Signal daemon runs without publishing a public host port
- the `dev`, `test`, and `prod` identities have isolated durable state directories
- each runtime can reach the shared Signal capability over a local-only path
- each runtime binds only to its own configured identity
- the dedicated Signal identities can receive DMs
- first contact from an unpaired sender is gated by pairing policy
- a paired sender can complete a DM turn and receive a reply through the matching runtime
- the reply traverses the normal OpenClaw runtime path rather than a sidecar chat path
- model escalation behavior still works for Signal-originated requests
- operators can retrieve Signal status, inspect data, logs, and health through the documented management surface
- durable Signal state survives capability redeploy or container replacement
- `docs/operator/signal.md` exists and documents bootstrap and recovery
- MCP management coverage exists where remote control is appropriate

## 18. Upstream Verification Gate

Before implementation starts, the implementation owner must verify the current upstream contracts for:

- OpenClaw Signal channel configuration and lifecycle hooks
- the exact daemon/API posture expected by the supported `signal-cli` integration
- concurrent multi-account support in the chosen `signal-cli` deployment model
- account registration or linking flows needed for dedicated per-environment Signal identities

This verification is required because the active repository currently contains only Signal placeholders, not a completed integration.
