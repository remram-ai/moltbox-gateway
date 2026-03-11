# Moltbox CLI and Control-Plane Refactor

Status: proposed

This document defines the refactor target for the Moltbox CLI and control plane before implementation begins.

It is intentionally a design and migration spec, not a description of the current `main` behavior.

## 1. Goals

The refactor has four goals:

- remove SSH as a normal operational dependency
- make Docker the canonical infrastructure interface instead of reimplementing container logic
- make runtime behavior observable through runtime APIs instead of container internals
- keep the operator interface consistent across local CLI and remote MCP control

## 2. Non-Goals

This refactor does not:

- replace Docker or Docker Compose as the appliance runtime
- move runtime-owned behavior into the tools container
- make every local-only operation remotely available through MCP
- redefine OpenClaw's full application API beyond the Moltbox operator requirements in this document

## 3. Current System Analysis

### 3.1 Current CLI structure

The current CLI is split into three domains:

- `moltbox tools <verb>`
- `moltbox host <service> <verb>`
- `moltbox runtime <environment> <verb>`

Implemented verbs are currently centered on:

- tools: `version`, `health`, `serve`, `status`, `inspect`, `update`, `rollback`, `logs`
- host: `deploy`, `rollback`, `status`, `inspect`, `logs`, `start`, `stop`, `restart`
- runtime: `deploy`, `rollback`, `status`, `inspect`, `logs`, `start`, `stop`, `restart`, `chat`

The current implementation is not purely thin. The CLI package owns:

- argument parsing
- target registry bootstrap
- deployment rendering
- snapshot and rollback coordination
- Docker inspection and lifecycle execution
- runtime chat via `docker exec ... openclaw agent ...`
- tools service hosting
- MCP wrapping

### 3.2 Current tools service responsibilities

The `tools` container currently:

- runs `moltbox tools serve`
- exposes `/health`
- mounts the Docker socket
- mounts the Moltbox state root and runtime artifacts root
- hosts the MCP server
- re-invokes the CLI for most remote actions

This makes the tools service the de facto remote control plane already, but its implementation is still a CLI wrapper instead of a clear control-plane service boundary.

### 3.3 Current Docker usage

Docker is already the real infrastructure authority. The current code uses:

- `docker inspect` for target inspection
- `docker logs` for logs
- `docker start|stop|restart` for lifecycle
- `docker compose ... up|stop|restart` for deployment and runtime lifecycle
- `docker network inspect|create` for internal network setup
- `docker rm -f` during replace-and-redeploy flows

The issue is not that Docker is absent. The issue is that Docker access is spread across primitive scripts, deployment services, and manual operator procedures instead of a single normalized proxy layer.

### 3.4 Current runtime container layout

The current topology has three distinct layers:

- shared services such as `ollama`, `opensearch`, and `ssl`
- runtime containers such as `openclaw-dev` and `openclaw-test`
- the singleton `moltbox-tools` container

Important current details:

- runtimes and shared services use the shared `moltbox_moltbox_internal` network
- the tools container is separate and reaches infrastructure through the Docker socket
- runtime config is rendered from `moltbox/config/` into host runtime roots
- runtime health and behavior are still partly inferred from container state and container-local CLI calls

### 3.5 Current OpenClaw runtime surface

The current managed runtime already exposes useful behavior, but Moltbox is not consuming it consistently through APIs.

Current Moltbox-facing behavior includes:

- the OpenClaw gateway on the runtime port
- runtime health via container health checks
- runtime chat via `docker exec ... openclaw agent --json`
- runtime diagnostics via manual `docker exec ... openclaw doctor`
- runtime config discovery via manual `openclaw config ...` and file inspection
- model diagnostics via manual `openclaw models status --json`

This means runtime behavior exists, but the control plane still depends on shelling into the container instead of calling a stable runtime operator API.

## 4. Architectural Problems

### 4.1 Duplicated functionality

The current CLI duplicates infrastructure responsibilities that Docker already solves:

- container inspection modeling
- log collection shaping
- lifecycle command routing
- topology reconstruction

The CLI should normalize Docker output, not behave like a second container runtime.

### 4.2 Fragile runtime operations

Runtime chat and diagnostics currently depend on container-local CLI execution.

That creates several problems:

- remote operators need host-level access paths for deeper diagnosis
- runtime behavior depends on command availability inside the container
- MCP cannot expose the same behavior cleanly without wrapping shell execution
- runtime capabilities do not have stable HTTP contracts

### 4.3 SSH-only diagnostics

The current operator runbook still falls back to:

- `ssh moltbox`
- `docker ps`
- `docker logs`
- `docker inspect`
- `docker exec openclaw ...`

That is acceptable for break-glass recovery. It is not acceptable as the normal diagnostic model.

### 4.4 Inconsistent capability integration

Capabilities can exist in one of three incomplete forms:

- implemented in a runtime config or plugin with no Moltbox command
- reachable locally through shell-only procedures
- partially exposed through MCP without a matching local diagnostic surface

This happens because capability integration is currently checked per feature, not enforced structurally by the control plane.

### 4.5 MCP surface drift risk

The tools service is the current remote entrypoint, but:

- MCP calls the CLI rather than a dedicated application service
- a standalone MCP adapter entrypoint also exists
- runtime-specific behavior is still outside the tools service boundary

This creates unnecessary drift risk between local CLI behavior, tools service behavior, and future remote automation.

## 5. Refactored Architecture

### 5.1 Responsibility split

The target architecture has four layers.

```text
operator CLI
  -> control-plane application service
     -> docker proxy adapter
     -> runtime API client
     -> deployment renderer/state store
  -> MCP tools surface in the tools container
```

Responsibilities:

- CLI
  - parse commands
  - validate arguments
  - select the correct control-plane operation
  - format structured JSON or human-friendly text views
- control-plane application service
  - own command semantics
  - combine Docker, runtime API, and deployment state into one response
  - provide a single implementation path for local CLI and MCP
- Docker proxy adapter
  - execute allowlisted Docker and Docker Compose actions
  - normalize infrastructure output
  - never accept arbitrary shell payloads
- runtime API client
  - call runtime monitor and operator endpoints
  - handle runtime-specific readiness, doctor, model, and config inspection

### 5.2 Thin CLI rule

The CLI remains the operator interface, but it must stop embedding infrastructure logic directly.

The thin CLI rule is:

- no feature-specific `docker exec` workflows in command handlers
- no duplicate container-state inference outside the Docker proxy adapter
- no direct MCP policy decisions in command handlers
- no separate local-vs-remote semantics for the same verb

### 5.3 Single MCP control plane

The `tools` container hosts the only Moltbox MCP server.

Rules:

- all remote operator management flows through the tools service
- runtime containers do not host separate Moltbox MCP servers
- MCP tools call the same control-plane application service used by the local CLI
- policy is enforced at the tools service boundary, not reimplemented per runtime

### 5.4 Docker as the infrastructure interface

Infrastructure commands are resolved through a Docker proxy layer with registered targets.

The Docker proxy owns:

- container status
- logs
- inspect
- start
- stop
- restart
- topology
- compose-based deploy and rollback primitives

### 5.5 Runtime APIs for runtime behavior

Runtime-owned behavior must move behind runtime APIs.

Runtime API ownership includes:

- chat
- doctor and readiness
- model and routing status
- runtime config inspection
- channel capability diagnostics when the channel is runtime-owned

The control plane should ask the runtime what it is doing. It should not infer runtime behavior by attaching to the container.

### 5.6 Control-plane topology

```text
Local operator
  -> moltbox CLI
  -> control-plane service layer
     -> docker proxy
     -> runtime API clients
     -> state/render store

Remote operator / agent
  -> tools MCP server
  -> same control-plane service layer
     -> docker proxy
     -> runtime API clients
     -> state/render store
```

## 6. Final CLI Command Tree

This is the post-refactor command tree.

It defines the stable operator shape, even if some commands are introduced in phases.

### 6.1 Top-level groups

- `moltbox tools ...`
- `moltbox host ...`
- `moltbox runtime ...`

### 6.2 `moltbox tools`

Responsibility:

- inspect and operate the singleton tools service and control-plane surface

Implementation path:

- Docker proxy for lifecycle and logs
- local tools service state for health and identity

Commands:

- `moltbox tools version`
- `moltbox tools health`
- `moltbox tools status`
- `moltbox tools inspect`
- `moltbox tools logs`
- `moltbox tools update`
- `moltbox tools rollback`
- `moltbox tools serve`

### 6.3 `moltbox host`

Responsibility:

- inspect the appliance-level infrastructure shared across runtimes

Implementation path:

- Docker proxy only

Commands:

- `moltbox host topology`
- `moltbox host doctor`
- `moltbox host ssl status`
- `moltbox host ssl inspect`
- `moltbox host ssl logs`
- `moltbox host ssl deploy`
- `moltbox host ssl rollback`
- `moltbox host ssl start`
- `moltbox host ssl stop`
- `moltbox host ssl restart`
- `moltbox host ollama status`
- `moltbox host ollama inspect`
- `moltbox host ollama logs`
- `moltbox host ollama deploy`
- `moltbox host ollama rollback`
- `moltbox host ollama start`
- `moltbox host ollama stop`
- `moltbox host ollama restart`
- `moltbox host opensearch status`
- `moltbox host opensearch inspect`
- `moltbox host opensearch logs`
- `moltbox host opensearch deploy`
- `moltbox host opensearch rollback`
- `moltbox host opensearch start`
- `moltbox host opensearch stop`
- `moltbox host opensearch restart`

`host topology` returns normalized output from:

- `docker ps`
- `docker network inspect`
- the target registry

`host doctor` performs infrastructure-only checks such as:

- required containers present
- required networks present
- published ports bound as expected
- tools service reachable

### 6.4 `moltbox runtime`

Responsibility:

- operate managed runtime environments and inspect runtime-owned behavior

Implementation path:

- Docker proxy for infrastructure lifecycle and logs
- runtime APIs for behavior and diagnostics

Environment root:

- `moltbox runtime <environment> ...`

Core commands:

- `moltbox runtime <environment> status`
- `moltbox runtime <environment> inspect`
- `moltbox runtime <environment> logs`
- `moltbox runtime <environment> deploy`
- `moltbox runtime <environment> rollback`
- `moltbox runtime <environment> start`
- `moltbox runtime <environment> stop`
- `moltbox runtime <environment> restart`

Runtime behavior commands:

- `moltbox runtime <environment> monitor`
- `moltbox runtime <environment> doctor`
- `moltbox runtime <environment> chat --message ...`
- `moltbox runtime <environment> models status`
- `moltbox runtime <environment> routing status`
- `moltbox runtime <environment> config inspect`

Runtime-owned capability pattern:

- `moltbox runtime <environment> channels discord status`
- `moltbox runtime <environment> channels discord doctor`

Command routing rules:

- `status`, `inspect`, `logs`, `start`, `stop`, `restart`, `deploy`, `rollback` use Docker proxy and deployment state
- `monitor`, `doctor`, `chat`, `models status`, `routing status`, `config inspect`, and runtime-owned capability diagnostics use runtime APIs

### 6.5 Output contract

Every command returns structured JSON.

Required top-level fields:

- `ok`
- `domain`
- `target`
- `command`
- `source`
- `timestamp`

Optional normalized sections:

- `docker`
- `runtime`
- `deployment`
- `diagnostics`
- `warnings`
- `errors`

## 7. Docker Proxy Design

### 7.1 Design rule

The Docker proxy is an allowlisted adapter, not a generic shell escape hatch.

Inputs must be resolved from:

- a registered target id
- a known operation
- a bounded option set such as `tail`, `since`, or `follow`

### 7.2 Command mapping

Infrastructure operations map as follows.

| Moltbox operation | Docker operation |
| --- | --- |
| `tools inspect`, `host <service> inspect`, `runtime <env> inspect` | `docker inspect <container...>` |
| `tools logs`, `host <service> logs`, `runtime <env> logs` | `docker logs <container>` |
| `host <service> start|stop|restart` | `docker start|stop|restart <container>` |
| `runtime <env> start|stop|restart` | `docker compose ... start|stop|restart` |
| `deploy` | `docker compose ... up -d` |
| `rollback` | `docker compose ... up -d` against the stored snapshot render |
| `host topology` | `docker ps`, `docker network inspect`, `docker volume inspect` when needed |

### 7.3 JSON normalization

The proxy normalizes Docker output into stable Moltbox models.

Required normalization behavior:

- convert Docker state strings into stable enums where useful
- preserve raw Docker output under a `docker.raw` field when debugging is needed
- expose container identity, image, health, ports, mounts, and network attachments in a consistent shape
- return topology as target-oriented data instead of raw container lists only

Example normalized inspect envelope:

```json
{
  "ok": true,
  "domain": "runtime",
  "target": "dev",
  "command": "moltbox runtime dev inspect",
  "source": "docker_proxy",
  "docker": {
    "containers": [
      {
        "name": "openclaw-dev",
        "state": "running",
        "health": "healthy",
        "image": "openclaw:local"
      }
    ]
  }
}
```

### 7.4 Error handling

The proxy must map Docker failures into stable error categories.

Required error types:

- `target_not_found`
- `docker_unavailable`
- `docker_command_failed`
- `compose_render_missing`
- `snapshot_not_found`
- `permission_denied`
- `topology_incomplete`

Error payloads must include:

- `error_type`
- `error_message`
- `recovery_message`
- `details`

### 7.5 Security constraints

The Docker proxy must never expose arbitrary container execution through MCP.

Rules:

- no raw `docker exec`
- no raw `docker run`
- no arbitrary image names or container ids from remote callers
- no free-form Docker flags through MCP
- target resolution must come from the registry
- mutation verbs remain policy-gated in the tools service

Local break-glass shell access may still exist for human operators, but it is outside the normal control-plane contract.

## 8. Runtime Monitor Specification

### 8.1 Endpoint

- method: `GET`
- path: `/moltbox/monitor`

This endpoint is required for every managed runtime container.

### 8.2 Purpose

The monitor endpoint provides a fast, non-invasive runtime health and capability summary for the control plane.

It is not a full admin API. It is the baseline status contract used by:

- `moltbox runtime <environment> monitor`
- `moltbox runtime <environment> status`
- `moltbox host doctor`
- tools-service MCP diagnostics

### 8.3 Required response fields

```json
{
  "runtime": "openclaw",
  "environment": "dev",
  "status": "healthy",
  "version": "0.0.0",
  "started_at": "2026-03-11T00:00:00Z",
  "provider": "ollama",
  "model": "qwen3:8b",
  "chat_ready": true,
  "dependencies": {
    "ollama": "healthy",
    "opensearch": "healthy"
  }
}
```

Required fields:

- `runtime`
- `environment`
- `status`
- `version`
- `started_at`
- `provider`
- `model`
- `chat_ready`
- `dependencies`

### 8.4 Optional response fields

Optional fields:

- `uptime_seconds`
- `ready`
- `gateway`
- `routing`
- `channels`
- `doctor_summary`
- `warnings`
- `config_revision`
- `capabilities`

Recommended optional shape:

```json
{
  "gateway": {
    "port": 18789,
    "bind": "lan"
  },
  "routing": {
    "enabled": true,
    "provider": "remram-router",
    "default_model": "ollama/qwen3:8b"
  },
  "channels": {
    "discord": {
      "enabled": true,
      "status": "healthy"
    }
  },
  "warnings": []
}
```

### 8.5 Health semantics

`status` uses these meanings:

- `healthy`
  - runtime is up and can serve normal chat traffic
- `degraded`
  - runtime is up but one or more dependencies or capabilities are impaired
- `starting`
  - runtime process is starting or warming dependencies
- `unhealthy`
  - runtime is reachable but not ready for normal operations
- `stopped`
  - runtime is intentionally not running

Rules:

- `chat_ready: true` is required for `healthy`
- dependency failures should normally produce `degraded` or `unhealthy`
- missing optional capabilities should not force `unhealthy` unless they are part of the environment's declared required capabilities

## 9. Runtime Operator API Surface

The runtime monitor is the minimum required endpoint. The refactor also requires a small operator API family for runtime-owned behavior.

Required runtime operator endpoints:

- `GET /moltbox/monitor`
- `GET /moltbox/doctor`
- `GET /moltbox/models/status`
- `GET /moltbox/routing/status`
- `GET /moltbox/config/inspect`
- `POST /moltbox/chat`

Rules:

- every endpoint returns structured JSON
- every endpoint must be safe for local control-plane consumption
- configuration inspection must return a sanitized operator view, not raw secrets
- channel-specific diagnostics may live under `/moltbox/channels/<capability>/...`

## 10. Capability Integration Rule v2

Every new capability must expose three integration surfaces.

### 10.1 Required surfaces

1. Implementation surface
   The service, runtime component, plugin, or subsystem that actually implements the capability.

2. CLI surface
   A Moltbox CLI command that lets operators invoke, inspect, and diagnose the capability.

3. MCP surface when appropriate
   A tools-service MCP tool or MCP action path for remote management when remote control is allowed.

### 10.2 Required diagnostic coverage

Every capability must also provide:

- structured JSON output
- at least one diagnostic command
- a documented ownership boundary

Examples:

- `moltbox runtime dev doctor`
- `moltbox runtime dev channels discord doctor`
- `moltbox host ollama status`

### 10.3 Placement rule

Capabilities must appear under the owner that actually governs them.

Examples:

- shared infrastructure capability -> `moltbox host ...`
- runtime-owned capability -> `moltbox runtime <environment> ...`
- tools-service capability -> `moltbox tools ...`

### 10.4 Hidden capability prohibition

A capability is incomplete if it requires one of these as the normal path:

- SSH
- raw `docker exec`
- manual file inspection inside the container
- undocumented host mutation

Break-glass procedures may still exist, but they do not satisfy capability integration.

## 11. Risks and Mitigations

### 11.1 Docker compatibility risk

Risk:

- `docker compose` behavior and JSON formatting differ across host versions

Mitigation:

- keep the Docker proxy allowlist narrow
- normalize all Docker output before returning it
- test against the supported host Docker version explicitly

### 11.2 Runtime API gap risk

Risk:

- OpenClaw does not yet expose the full Moltbox operator API contract

Mitigation:

- implement the monitor endpoint first
- add operator endpoints in a Moltbox-specific runtime plugin layer
- keep temporary fallback code local-only during migration, not as the final contract

### 11.3 Security risk from Docker socket access

Risk:

- the tools container is highly privileged because of the Docker socket

Mitigation:

- expose only allowlisted operations through MCP
- keep mutation verbs policy-gated
- never forward arbitrary shell or Docker arguments from remote callers

### 11.4 Backward-compatibility risk

Risk:

- operators and automation already rely on the current grammar and payload shape

Mitigation:

- preserve existing command forms where possible
- add new runtime subcommands without breaking the current top-level domains
- keep compatibility aliases such as `caddy -> ssl` until deprecation is completed

### 11.5 Migration drift risk

Risk:

- local CLI, MCP, and runtime APIs may move at different speeds

Mitigation:

- implement the control-plane application service first
- require both CLI and MCP to call the same service layer
- gate new commands behind tests that cover both local and remote entrypoints

## 12. Recommended Improvements

Repository analysis suggests these additional simplifications.

### 12.1 Remove primitive-script indirection

The current primitive runner and command-script split should collapse into a normal Python service layer with clear adapters.

The current split adds process hops without adding strong isolation.

### 12.2 Separate infrastructure state from runtime behavior in status payloads

`status` should report both:

- Docker/container state
- runtime monitor state

That will make it obvious whether the issue is infrastructure, runtime readiness, or a capability-level regression.

### 12.3 Add a first-class topology command

Operators currently reconstruct topology manually from Docker.

`moltbox host topology` should become the canonical answer for:

- running containers
- network attachments
- published ports
- target-to-container mapping

### 12.4 Add runtime capability namespaces early

If Discord and future channels remain runtime-owned, the command tree should reflect that now instead of waiting for more capability drift.

## 13. Implementation Plan

### Phase 1. Stabilize the control-plane service layer

Modules affected:

- `moltbox-cli/tools/src/moltbox_cli/cli.py`
- `moltbox-cli/tools/src/moltbox_cli/deployment_service.py`
- `moltbox-cli/tools/src/moltbox_cli/mcp_adapter.py`
- `moltbox-cli/tools/src/moltbox_cli/http_app.py`

Work:

- introduce explicit service objects for tools, host, runtime, Docker proxy, and runtime API clients
- keep current commands working through the new service layer

### Phase 2. Implement the Docker proxy

Modules affected:

- current Docker primitives
- target inspection and topology code
- deployment and rollback orchestration

Work:

- replace scattered Docker helpers with one allowlisted adapter
- normalize inspect, logs, lifecycle, and topology payloads
- add tests for failure mapping and policy boundaries

### Phase 3. Add runtime monitor and operator endpoints

Modules affected:

- OpenClaw runtime integration under `moltbox/config/openclaw/extensions/remram-runtime/`
- runtime command handlers

Work:

- add `/moltbox/monitor`
- add runtime doctor, models, routing, config, and chat operator endpoints
- stop using `docker exec` as the normal runtime capability path

### Phase 4. Expand the CLI tree

Modules affected:

- CLI parser
- runtime command handlers
- operator docs

Work:

- add `host topology`
- add `host doctor`
- add runtime behavior subcommands
- add runtime capability namespaces such as `channels discord`

### Phase 5. Converge MCP on the same service path

Modules affected:

- tools MCP server
- policy layer

Work:

- remove MCP-specific logic duplication
- expose new read-only runtime diagnostics through tools MCP
- preserve policy restrictions on mutation verbs

### Phase 6. Migrate docs and operator workflows

Modules affected:

- operator references
- runbooks
- capability docs

Work:

- move normal diagnostics away from SSH and raw Docker procedures
- keep break-glass manual steps only in recovery sections

### Phase 7. Verification and rollout

Testing strategy:

- unit tests for Docker proxy normalization and error mapping
- unit tests for runtime API clients
- CLI parser tests for the new grammar
- MCP tests for policy-aware remote access
- integration tests covering:
  - deploy
  - inspect
  - topology
  - runtime monitor
  - runtime doctor
  - runtime chat
  - channel diagnostics

Migration steps:

1. land service-layer refactor without changing the public grammar
2. land Docker proxy normalization
3. land runtime monitor and operator APIs
4. add new CLI commands
5. switch existing runtime behavior commands to runtime APIs
6. update operator docs and remove SSH as a normal operational requirement

## 14. Acceptance Criteria

The refactor is complete when all of the following are true:

- normal operator diagnosis does not require SSH
- local CLI and tools MCP use the same service implementation path
- runtime behavior commands do not rely on `docker exec`
- infrastructure commands are backed by the Docker proxy
- every managed capability has a CLI surface and a diagnostic command
- runtime monitor is implemented for every managed runtime
