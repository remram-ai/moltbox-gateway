# Deployment Primitives Specification

## Purpose

Define the formal interface for the small host-side deployment primitives that
replace the legacy monolithic bootstrap model.

## Capability Definition

Capability name:

- Host-side deployment primitive contract

Implementation order:

- 4

Depends on:

- `filesystem_layout.md`
- `repository_structure.md`
- `runtime_model.md`

Delivers:

- the required primitive inventory
- the standard JSON result envelope
- operational boundaries for each primitive

Primary implementation reference for:

- host-tool script implementation
- control-plane invocation contracts
- validation and rollback flow composition

Acceptance criteria:

- every required primitive is defined
- each primitive includes purpose, inputs, outputs, failure conditions, and side
  effects
- the shared JSON envelope is defined once for all primitives
- control-plane code can invoke primitives without inventing new contracts

## Responsibilities

- provide single-purpose deployment operations
- provide machine-readable results
- provide safe composition into higher-level flows
- isolate privileged host mutation away from the tools container

## Global Contract

Each primitive must:

- perform exactly one logical operation
- accept structured inputs
- return structured output
- emit logs without requiring human parsing for success detection
- exit non-zero on failure
- avoid unrelated host mutation

## Required Output Envelope

Each primitive returns JSON with, at minimum:

- `ok`: boolean
- `operation`: string
- `runtime`: string or `shared`
- `started_at`: timestamp
- `finished_at`: timestamp
- `details`: object
- `warnings`: array
- `errors`: array

Optional fields:

- `artifacts`
- `validation`
- `snapshot_id`
- `commit`
- `manifest_id`

## Primitive Definitions

### `create_runtime`

Purpose:

- create or initialize a runtime root for `dev`, `test`, or `prod`

Inputs:

- runtime name
- runtime root path
- manifest template version
- optional force flag

Outputs:

- created directories
- initialized config/template set
- runtime registry metadata

Failure conditions:

- invalid runtime name
- unsafe target path
- missing template inputs
- permission denied

Side effects:

- creates runtime directories
- writes initial config and metadata

### `destroy_runtime`

Purpose:

- remove an existing runtime root and associated runtime-owned state

Inputs:

- runtime name
- runtime root path
- optional force flag

Outputs:

- removed paths
- cleanup report

Failure conditions:

- runtime does not exist
- unsafe path resolution
- runtime is protected and operation is not allowed
- runtime is still active

Side effects:

- deletes runtime-owned files and directories

### `start_runtime`

Purpose:

- start the runtime execution stack for one environment

Inputs:

- runtime name
- runtime manifest reference

Outputs:

- container identifiers
- startup result
- health probe summary

Failure conditions:

- missing runtime root
- missing config
- container launch failure
- failed health checks

Side effects:

- starts runtime container(s)

### `stop_runtime`

Purpose:

- stop the runtime execution stack for one environment

Inputs:

- runtime name
- optional timeout

Outputs:

- stopped containers
- shutdown summary

Failure conditions:

- runtime not found
- container engine failure
- timeout exceeded

Side effects:

- stops runtime container(s)

### `restart_runtime`

Purpose:

- restart the runtime execution stack for one environment

Inputs:

- runtime name
- optional validation flag

Outputs:

- restart summary
- optional validation result

Failure conditions:

- stop failure
- start failure
- health check failure

Side effects:

- stops and starts runtime container(s)

### `deploy_commit`

Purpose:

- apply a specific git commit and rendered manifest to a target runtime

Inputs:

- runtime name
- commit sha
- manifest id
- repo source reference

Outputs:

- deployed commit
- rendered manifest reference
- release metadata path

Failure conditions:

- commit not found
- working tree mismatch for protected flows
- render failure
- deployment artifact write failure

Side effects:

- updates runtime release metadata
- writes rendered runtime configuration

### `validate_runtime`

Purpose:

- run static, stack, application, and smoke validation for a runtime

Inputs:

- runtime name
- validation profile
- optional smoke-test selection

Outputs:

- pass, warn, fail result set
- individual check results
- report artifact path

Failure conditions:

- runtime unavailable
- validation definition missing
- health endpoint failure
- smoke test failure

Side effects:

- writes validation report artifacts

### `collect_diagnostics`

Purpose:

- collect bounded, non-mutating diagnostic artifacts for a runtime

Inputs:

- runtime name
- collection profile
- artifact retention policy

Outputs:

- artifact bundle path
- collected categories

Failure conditions:

- runtime unavailable
- diagnostics tool failure
- artifact write failure

Side effects:

- writes diagnostics bundle

### `snapshot_runtime`

Purpose:

- create a rollback snapshot for a target runtime

Inputs:

- runtime name
- snapshot reason
- operator or workflow id

Outputs:

- snapshot id
- snapshot location
- captured metadata

Failure conditions:

- runtime missing
- snapshot storage unavailable
- metadata capture failure

Side effects:

- writes snapshot artifacts

### `restore_snapshot`

Purpose:

- restore a target runtime from a previously created snapshot

Inputs:

- runtime name
- snapshot id
- optional force flag

Outputs:

- restored snapshot id
- restored paths
- post-restore status

Failure conditions:

- snapshot not found
- restore conflict
- post-restore validation failure

Side effects:

- overwrites runtime state with snapshot contents

### `deploy_shared_services`

Purpose:

- create or update shared infrastructure services `ollama` and `opensearch`

Inputs:

- service selection
- rendered shared-service manifest
- optional recreate flag

Outputs:

- deployed service set
- container identifiers
- health summary

Failure conditions:

- manifest invalid
- image pull failure
- startup failure
- health probe failure

Side effects:

- creates or updates shared-service containers
- mutates shared-service state and release metadata

## Primitive Dependencies

- filesystem layout
- runtime model
- control-plane orchestration
- container runtime and image registry access

## Example Flow

Candidate deploy to TEST:

1. `create_runtime(test)` if needed
2. `deploy_commit(test, commit)`
3. `start_runtime(test)`
4. `validate_runtime(test)`
5. `collect_diagnostics(test)` only if validation fails

Shared-services bring-up:

1. `deploy_shared_services(ollama, opensearch)`
2. confirm health
3. allow runtime start operations afterward
