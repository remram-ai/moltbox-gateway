# Gateway Capability Integration

This document explains how the Remram platform capability integration standard is implemented in the MoltBox gateway appliance.

The authoritative platform standard lives in the Remram repository at:

```text
remram/docs/standards/capability-integration.md
```

Gateway capability integrations must follow that platform standard.

This document records the gateway-specific implementation details:

- MoltBox host paths
- MoltBox CLI conventions
- container and config placement under `moltbox/`
- control-plane and MCP expectations inside `remram-gateway`

## Scope

This guide applies to capabilities integrated into `remram-gateway`, including:

- shared appliance services such as model or search services
- ingress adapters and channels
- runtime-adjacent services managed by the MoltBox control plane

It is an implementation guide, not the authoritative cross-platform contract.

## MoltBox Implementation Surfaces

Gateway capability integrations are implemented through these repository surfaces:

- `moltbox/containers/` for deployment assets
- `moltbox/config/` for checked-in configuration and render inputs
- `moltbox-cli/` for operator CLI and control-plane implementation
- `docs/operator/` for gateway operator guides

## Canonical Host Layout

The current gateway uses two primary host roots:

- `MOLTBOX_RUNTIME_ROOT`
  - default: `~/Moltbox`
  - owns runtime artifacts and the shared log root
- `MOLTBOX_STATE_ROOT`
  - current code default: `~/.remram`
  - owns control-plane state, rendered deployment assets, snapshots, and shared service state

Gateway capability integrations use these canonical locations:

- logs: `${MOLTBOX_RUNTIME_ROOT}/logs/<capability>/`
- durable shared-service state: `${MOLTBOX_STATE_ROOT}/shared/<capability>/`
- rendered deployment assets: `${MOLTBOX_STATE_ROOT}/deploy/rendered/...`
- target registration and control-plane metadata: `${MOLTBOX_STATE_ROOT}/state/targets/`

Examples:

```text
~/Moltbox/logs/discord/
~/.remram/shared/discord/
```

For edge and appliance concerns, prefer MoltBox-scoped naming. The current `~/.remram` state root is existing implementation state, not a reason to invent new `remram-*` edge namespaces.

## Container and Config Placement

Gateway capability deployment assets should be placed in the canonical MoltBox tree:

- shared services: `moltbox/containers/shared-services/<capability>/`
- runtime-specific capability assets: the appropriate runtime-scoped path under `moltbox/containers/runtimes/`
- checked-in config templates or render inputs: `moltbox/config/`

Operator documentation for each gateway capability should live at:

```text
docs/operator/<capability>.md
```

Examples:

```text
docs/operator/discord.md
docs/operator/ollama.md
docs/operator/opensearch.md
```

## MoltBox CLI Conventions

The current gateway management surface is the `moltbox` CLI.

Canonical grammar:

```text
moltbox <domain> <target> <verb>
```

Current domain forms:

```text
moltbox tools <verb>
moltbox host <service> <verb>
moltbox runtime <environment> <verb>
```

Current gateway capability integrations usually appear through:

- `moltbox host <service> <verb>` for shared appliance services
- `moltbox runtime <environment> <verb>` for environment-owned runtime behavior

Typical lifecycle and inspection verbs include:

- `deploy`
- `start`
- `stop`
- `restart`
- `status`
- `logs`
- `inspect`
- `health`
- `test`
- `rollback`

In the gateway, the CLI is for lifecycle management and diagnostics. It is not the service's primary application interface.

## Logging Implementation

Containerized gateway capabilities should write logs to the standard MoltBox log mount:

```text
${MOLTBOX_RUNTIME_ROOT}/logs/<capability>/
```

Default example:

```text
~/Moltbox/logs/discord/
```

Container posture:

- mount the capability log directory from the host
- use `/var/log/moltbox` as the in-container log mount base when a mounted file log is needed
- document the primary log file name in the capability guide

Operator posture:

- use `moltbox ... logs` as the normal retrieval path
- capability docs must state whether `logs` returns container stdout, mounted log files, or both

## Persistent State Implementation

Durable gateway capability state must live outside the container.

Shared-service default:

```text
${MOLTBOX_STATE_ROOT}/shared/<capability>/
```

Default example:

```text
~/.remram/shared/discord/
```

Environment-owned runtime state may also live under the environment runtime root when that is the correct ownership boundary, for example under `${MOLTBOX_RUNTIME_ROOT}/openclaw/<environment>/`.

Destroying and recreating the container must not destroy the capability's durable state.

## Control-Plane and MCP Implementation

Gateway capabilities are not complete until they are integrated with the MoltBox control plane.

That includes:

- target registration under `${MOLTBOX_STATE_ROOT}/state/targets/`
- rendered deployment assets under `${MOLTBOX_STATE_ROOT}/deploy/rendered/...`
- structured `status`, `inspect`, and `logs` behavior through the CLI
- policy-aware remote behavior when surfaced through MCP

The current MCP implementation wraps the MoltBox CLI and control-plane path rather than reimplementing service logic. Remote verbs may be narrower than local operator verbs because policy can restrict MCP access.

## Required Gateway Capability Guide Content

Each gateway capability guide under `docs/operator/` should include:

- Purpose
- Access Method
- Ports
- CLI Operations
- Runtime Behavior
- Logs
- Persistent State
- Health / Debugging
- Bootstrap Steps
- Failure Recovery

That content satisfies the platform documentation requirement in a MoltBox-specific way.

## Completion Rule

A gateway capability is not complete until:

- it satisfies the Remram platform capability integration standard
- its MoltBox deployment assets and config placement are defined
- its logs and durable state paths are defined
- its CLI and control-plane behavior are defined
- its operator guide exists under `docs/operator/`

If those conditions are missing, the gateway integration is still incomplete.
