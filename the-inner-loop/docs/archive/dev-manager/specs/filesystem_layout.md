# Filesystem Layout Specification

## Purpose

Define the canonical host-side operational filesystem for the new Remram
control-plane architecture.

This layout is the single source of truth for runtime state, shared service
state, deployment state, tool installations, and logs.

## Capability Definition

Capability name:

- Host operational state layout

Implementation order:

- 1

Depends on:

- architecture document only

Delivers:

- the canonical `~/.remram` directory contract
- ownership rules for tools, runtimes, shared services, snapshots, deploy
  metadata, and logs

Primary implementation reference for:

- filesystem creation logic
- runtime-root path resolution
- tool installation and version switching
- log and snapshot storage

Acceptance criteria:

- all required top-level directories are defined
- runtime roots and shared-service roots are separated
- tool version directories and aliases are defined
- later specs can reference paths here without redefining them

## Responsibilities

- provide a stable root for all Remram operational state
- separate tools state from runtime state
- separate shared infrastructure state from per-environment state
- support snapshots, rollback, and versioned tool installations
- provide predictable paths for control-plane code and host-side scripts

## Canonical Root

All operational state lives under:

```text
~/.remram
```

## Top-Level Layout

```text
~/.remram
  tools/
  host-tools/
  runtimes/
  snapshots/
  deploy/
  logs/
```

## Directory Definitions

### `tools/`

Purpose:

- installed tool bundles for the control plane
- version switching between `stable` and `rc`

Required structure:

```text
tools/
  versions/
    vX.Y.Z/
  stable -> versions/vX.Y.Z
  rc -> versions/vX.Y.Z
```

Rules:

- `stable` is the default active production-grade tool version
- `rc` is the candidate tool version under verification
- version directories are immutable once installed
- control-plane launch paths must resolve through `stable` or `rc`, never by
  hardcoding a version directory

### `host-tools/`

Purpose:

- installed host-side privileged and semi-privileged deployment primitives

Rules:

- each primitive is a single-purpose executable
- host tools are invoked only through allowlisted control-plane entrypoints
- no arbitrary shell execution is exposed through this directory

Expected categories:

```text
host-tools/
  runtime/
  stack/
  deploy/
  validate/
  diagnostics/
  snapshot/
  shared/
```

### `runtimes/`

Purpose:

- per-environment runtime roots for `dev`, `test`, and `prod`

Required structure:

```text
runtimes/
  dev/
  test/
  prod/
```

Rules:

- each runtime root contains only environment-specific state
- shared infrastructure state must not be stored here
- direct mutation policy depends on environment definition

### `snapshots/`

Purpose:

- rollback points for runtime and tools state

Rules:

- snapshot names must be unique and timestamped
- production mutations require a pre-existing snapshot record
- snapshot metadata must include the source runtime, commit, and tool version

### `deploy/`

Purpose:

- publish runs, validation reports, approval state, deployment manifests, and
  pipeline metadata

Expected contents:

```text
deploy/
  runs/
  reports/
  approvals/
  manifests/
```

### `logs/`

Purpose:

- structured logs for the control plane, host tools, runtimes, and shared
  services

Expected structure:

```text
logs/
  control-plane/
  host-tools/
  shared/
    ollama/
    opensearch/
  runtimes/
    dev/
    test/
    prod/
```

## Runtime Root Shape

Each runtime root under `runtimes/<env>/` must contain only state required to
run that environment.

Minimum structure:

```text
runtimes/<env>/
  config/
  data/
  cache/
  tmp/
  logs/
  release/
```

Definitions:

- `config/`: rendered runtime configuration and policy files
- `data/`: runtime-owned persistent data
- `cache/`: rebuildable caches
- `tmp/`: disposable working files
- `logs/`: runtime-local logs if needed by the runtime
- `release/`: metadata for the currently deployed commit and manifest

## Shared Infrastructure State

Shared infrastructure state must not live under any runtime root.

Required structure:

```text
~/.remram/shared/
  ollama/
  opensearch/
```

Each shared-service directory may contain:

- `config/`
- `data/`
- `logs/`
- `release/`

## Interfaces

Consumers of this layout:

- tools container/service
- host-side deployment primitives
- snapshot and rollback system
- validation system
- diagnostics collection

## Dependencies

- host filesystem permissions
- versioned tool installer
- runtime registry
- deployment primitive contract

## Example Flow

Tool install:

1. install new bundle into `tools/versions/v0.4.0`
2. verify against control-plane self-checks
3. repoint `rc`
4. repoint `stable` only after successful verification

Runtime deployment:

1. resolve runtime root at `~/.remram/runtimes/test`
2. render config into `config/`
3. write commit metadata into `release/`
4. write logs into `logs/runtimes/test/`
