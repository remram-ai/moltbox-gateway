# Control Plane Specification

## Purpose

Define the new Remram control plane that orchestrates deployment and operations
without being part of the execution runtimes.

The control plane lives under `dev-manager/` and remains logically separate from
`dev`, `test`, and `prod`.

## Capability Definition

Capability name:

- Control-plane orchestration and Codex tool surface

Implementation order:

- 5

Depends on:

- `filesystem_layout.md`
- `repository_structure.md`
- `runtime_model.md`
- `deployment_primitives.md`

Delivers:

- orchestration boundaries
- runtime registry ownership
- pipeline state ownership
- the approved Codex-facing tool categories

Primary implementation reference for:

- control-plane service design
- MCP/API surface design
- policy enforcement implementation
- run-history and registry state implementation

Acceptance criteria:

- control-plane subcomponents are defined
- Codex-safe tool categories are defined
- arbitrary shell execution is explicitly prohibited
- production deployment is explicitly blocked for Codex

## Responsibilities

- orchestrate deployments and promotions
- maintain the runtime registry
- maintain pipeline state and deployment history
- invoke allowlisted host-side primitives
- expose an MCP/API surface for Codex
- enforce environment and promotion policy
- collect structured status, validation, and diagnostics metadata

## Non-Responsibilities

The control plane must not:

- run as part of a runtime container
- allow arbitrary shell execution
- let Codex deploy directly to `prod`
- embed environment-specific runtime state
- replace the shared services themselves

## Components

### Runtime Registry

Stores:

- runtime definitions
- runtime root paths
- shared-service definitions
- currently deployed commit per runtime
- current status and health summary

### Orchestration Engine

Coordinates:

- runtime lifecycle operations
- shared-service deployment
- deployment primitive invocation
- pipeline stage transitions

### Pipeline State Manager

Stores:

- publish run ids
- candidate commit metadata
- validation results
- approval state
- rollback metadata
- tool version used during a run

### Policy Enforcement Layer

Enforces:

- runtime mutation policy
- Codex scope restrictions
- promotion boundaries
- allowlisted primitive invocation

### MCP/API Surface

Exposes curated tools for:

- status inspection
- DEV iteration operations
- TEST deployment operations
- validation and diagnostics access
- report retrieval

## Interfaces

Primary inputs:

- repository metadata
- runtime registry data
- operator approvals
- host-tool results

Primary outputs:

- structured tool results
- deployment reports
- validation reports
- pipeline state transitions

## Required API/Tool Surface for Codex

Read-only tools:

- get control-plane status
- list runtimes
- get runtime status
- get shared-service status
- read validation report
- read diagnostics report
- read deployment run history

DEV-scoped mutating tools:

- deploy branch or commit to DEV
- start DEV
- stop DEV
- restart DEV
- validate DEV
- inspect DEV logs

TEST-scoped mutating tools:

- deploy selected commit to TEST
- start TEST
- stop TEST
- validate TEST
- finalize TEST validation report

Codex restrictions:

- no direct `prod` deployment tool
- no arbitrary command execution
- no unrestricted script runner

## Dependencies

- filesystem layout specification
- runtime model specification
- deployment primitive specification
- promotion pipeline specification
- tools container specification

## Example Flows

DEV iteration:

1. control plane resolves `dev`
2. invokes `deploy_commit`
3. invokes `restart_runtime`
4. invokes `validate_runtime`
5. exposes results to Codex

TEST candidate validation:

1. control plane resolves commit from `main`
2. invokes `deploy_commit(test)`
3. invokes `start_runtime(test)`
4. invokes `validate_runtime(test)`
5. records report in pipeline state
