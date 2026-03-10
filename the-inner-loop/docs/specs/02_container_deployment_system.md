# Container Deployment System

## Purpose

Teach the control plane how to deploy and manage containers through a
deterministic, CLI-first deployment system.

This domain defines how deployment assets are rendered, how deployment targets
are resolved, and how approved host-side commands are invoked to perform
container lifecycle actions.

## Capability Provided

This capability provides:

- deployment asset rendering
- target and profile resolution
- deterministic rendered output paths
- host-side primitive invocation through an allowlist
- service and runtime lifecycle commands

This capability manages container deployment and replacement only.

It does not manage:

- feature deployment
- runtime configuration promotion
- DEV -> TEST -> PROD workflows
- approval gates
- feature validation pipelines

There is only one active version of each container at a time.

It defines how containers are deployed, but not which specific services or
runtimes exist. That ownership belongs to the Runtime Stack domain.

Promotion and approval behavior belongs to `05_deployment_pipeline.md`.

## Owned Components

The container deployment system owns:

- `the-inner-loop/deployment-assets/`
- render target and render profile resolution
- render manifests and deterministic rendered output
- primitive command contracts
- host-side script lookup and invocation rules
- service and runtime lifecycle command routing

It does not own:

- the shared-service inventory
- runtime identities and runtime roots
- runtime tooling packages
- approval and rollback policy

## Canonical Interfaces

Representative command surface:

- `remram render-assets --target <target> --profile <profile>`
- `remram run-primitive --name <primitive> [args...]`
- `remram deploy <service>`
- `remram runtime start <env>`
- `remram runtime stop <env>`
- `remram runtime restart <env>`

Profiles represent deployment environments:

- `dev`
- `test`
- `prod`

Render targets map directly to directories under:

```text
the-inner-loop/deployment-assets/
```

## Target Classes

Targets represent deployable container groups and fall into three classes.

Control Plane Targets

Containers that operate the Remram control plane.

Example:

- `control-plane`

Shared Service Targets

Infrastructure services shared across runtimes.

Examples:

- `shared-services/ollama`
- `shared-services/opensearch`

Runtime Targets

Containers that run OpenClaw runtimes.

Example:

- `runtimes/openclaw`

The container deployment system treats all targets generically.

The meaning of each target class is defined by other capability documents.

Initial target examples:

- `control-plane`
- `shared-services/ollama`
- `shared-services/opensearch`
- `runtimes/openclaw`

Rules:

- targets map directly to deployment asset directories
- environment selection occurs through `--profile`, not target naming
- nested directories are allowed but only leaf targets are deployable
- render operations are target-scoped
- rendering one target must not implicitly render other targets
- operational commands return structured JSON
- CLI errors must be informative and machine-usable
- errors must identify the failing target, profile, or primitive and tell the
  caller what to do next

## Render Contract

Rendered output root:

```text
~/.remram/deploy/rendered/
```

Rendered output path:

```text
~/.remram/deploy/rendered/<profile>/<target>/
```

Each render produces:

- rendered compose, config, and environment files
- `render-manifest.json`

Minimum render manifest fields:

- `target`
- `profile`
- `render_timestamp`
- `render_version`
- `render_outcome`
- `source_asset_paths`

Determinism rules:

- rendered compose files are deterministic for identical inputs
- rendered configuration files are deterministic for identical inputs
- rendered environment files are deterministic for identical inputs
- manifest metadata fields such as timestamps are excluded from determinism
  checks

Two renders with identical inputs must produce identical output files.

Render exit codes:

- `0`: render completed successfully
- `1`: invalid target
- `2`: invalid profile
- `3`: render failure

## Primitive Runner Contract

Approved host-side primitives are the only mutating execution path available to
the control plane.

Rules:

- each primitive maps to one logical operation
- primitive names are resolved through an internal allowlist
- callers cannot pass arbitrary executable names
- raw shell snippets are prohibited
- commands are executed directly as processes, not through `sh -c`
- primitive results are captured as structured output with stdout, stderr, exit
  code, timestamps, and parsed JSON payloads where applicable

Expected primitive families:

- runtime
- stack
- deploy
- validate
- diagnostics
- snapshot
- shared

The CLI is an interface layer that routes commands to approved primitives.

Example mappings:

```text
remram deploy shared-services/ollama
  -> run-primitive deploy_shared_services

remram runtime start dev
  -> run-primitive start_runtime_dev
```

The primitive allowlist remains the only mutating execution path.

## Container Snapshot And Rollback Contract

Container mutation must follow a snapshot-based safety model.

Deployment lifecycle:

1. identify the currently running container instance
2. stop or pause the container safely
3. capture a snapshot of the container state
4. store the snapshot in the snapshot repository
5. deploy the new container
6. run container health validation
7. if validation fails, restore from the snapshot

Snapshot is mandatory before mutating a running container.

Snapshot scope must include:

- container image reference
- runtime configuration
- mounted volume references
- container metadata required to recreate the container

Rollback must restore the container to the snapshot state.

Example commands:

```text
remram snapshot create --target <target>
remram snapshot restore --snapshot-id <snapshot_id>
remram deploy <target>
```

## Deployment Logging Contract

Deployment actions must produce a structured deployment log.

Each deployment record must include:

- `deployment_id`
- `target`
- `timestamp`
- `previous_container_id`
- `snapshot_id`
- `new_container_id`
- `deployment_status`
- `rollback_performed`

Deployment logs are stored under:

```text
~/.remram/deploy/logs/
```

This log is used for diagnostics and rollback tracing.

## Structured Output Contract

All operational commands must return JSON containing at least:

```json
{
  "command": "remram deploy shared-services/ollama",
  "target": "shared-services/ollama",
  "status": "success",
  "exit_code": 0,
  "stdout": "",
  "stderr": "",
  "timestamp": "2026-03-09T18:00:00Z",
  "duration_ms": 250
}
```

On failure, operational commands must return machine-usable error details that:

- identify the relevant target, primitive, or snapshot id
- state which prerequisite or validation failed
- tell the caller what command or condition is required next

## Validation

This capability exists when:

- a render command produces deterministic output for a selected target and
  profile
- `render-manifest.json` is written with the required fields
- an approved primitive can be invoked successfully through `remram
  run-primitive`
- unknown primitive names are rejected without launching arbitrary commands
- a deployment command captures a snapshot before mutating a running container
- validation failure can trigger snapshot restore
- deployment records are written under `~/.remram/deploy/logs/`
- a lifecycle command path exists for starting or stopping a concrete target

## Deliverable Example

```text
remram render-assets --target shared-services/ollama --profile dev
remram run-primitive --name deploy_shared_services
remram runtime start dev
```
