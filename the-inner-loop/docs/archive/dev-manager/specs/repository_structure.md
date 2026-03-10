# Repository Structure Specification

## Purpose

Define how the new deployment architecture is represented inside the
`remram-gateway` repository.

The repository layout must make the new control plane explicit without mutating
the legacy `moltbox/` structure during initial phases.

## Capability Definition

Capability name:

- Repository separation for the new architecture

Implementation order:

- 2

Depends on:

- `filesystem_layout.md`

Delivers:

- the canonical repo layout for specs, control-plane code, host tools, compose
  definitions, manifests, and validation assets

Primary implementation reference for:

- where new code is written
- where new scripts are written
- where compose and manifest artifacts belong
- how `dev-manager/` stays separate from `moltbox/`

Acceptance criteria:

- `dev-manager/` responsibilities are explicit
- `moltbox/` is preserved as legacy during migration
- implementation teams can place new artifacts without ambiguity

## Responsibilities

- keep new architecture work under `dev-manager/`
- separate specifications from implementation
- separate control-plane code from host-side scripts
- preserve `moltbox/` as the legacy hardware-oriented area until migration is
  complete

## Required Repository Areas

```text
remram-gateway/
  dev-manager/
    docs/
      remram_deployment_architecture.md
      specs/
    control-pane/
    host-tools/
    compose/
    manifests/
    validation/
  moltbox/
  schemas/
```

## Directory Definitions

### `dev-manager/docs/`

Purpose:

- architecture documents
- formal specifications
- implementation planning documents

### `dev-manager/control-pane/`

Purpose:

- tools container or tools service source code
- runtime registry logic
- pipeline state management
- MCP/API surface for Codex
- host-tool orchestration layer

Rules:

- this directory contains unprivileged control-plane code
- it must not embed environment-specific runtime state
- it must not expose unrestricted shell access

### `dev-manager/host-tools/`

Purpose:

- source-controlled host-side deployment primitives

Rules:

- one primitive per file
- machine-readable output
- safe composition from the control plane
- privileged actions are isolated here, not in the tools container

### `dev-manager/compose/`

Purpose:

- compose manifests or equivalent deployment descriptors for:
  - shared services
  - runtime containers
  - tools container

Rules:

- runtime manifests must be environment-agnostic templates
- environment-specific values are injected from runtime roots or manifests
- shared-service manifests are separate from runtime manifests

### `dev-manager/manifests/`

Purpose:

- rendered deployment manifests, registry templates, and release descriptors

### `dev-manager/validation/`

Purpose:

- validation definitions, smoke-test recipes, and schema-driven checks used by
  the pipeline

## Interfaces

Primary relationships:

- `control-pane/` calls `host-tools/`
- `control-pane/` reads `validation/`
- `compose/` defines container topology
- `docs/specs/` defines required behavior for all other directories

## Dependencies

- formal specs in `dev-manager/docs/specs/`
- host filesystem specification
- deployment primitive specification
- runtime model specification

## Non-Goals

- moving legacy Moltbox code during the initial spec and scaffold phase
- mixing `moltbox/` deployment logic into the new `dev-manager/` layout

## Example Flow

Adding a new primitive:

1. define behavior in `docs/specs/deployment_primitives.md`
2. implement the host-side executable in `host-tools/`
3. add orchestration and allowlisting in `control-pane/`
4. add validation coverage in `validation/`

Adding a new runtime-facing API tool:

1. define the tool contract in `docs/specs/control_plane.md`
2. implement it in `control-pane/`
3. bind it only to approved primitives or read-only registry access
