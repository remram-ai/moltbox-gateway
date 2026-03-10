# Runtime Model Specification

## Purpose

Define the execution-plane environments and the shared infrastructure services
used by the new Remram deployment architecture.

## Capability Definition

Capability name:

- Runtime registry and environment model

Implementation order:

- 3

Depends on:

- `filesystem_layout.md`
- `repository_structure.md`

Delivers:

- formal definitions for `dev`, `test`, `prod`, `ollama`, and `opensearch`
- mutation policy and ownership boundaries for each environment

Primary implementation reference for:

- runtime registry implementation
- environment policy enforcement
- runtime-root and shared-service binding
- namespace isolation rules

Acceptance criteria:

- each environment is defined with purpose and restrictions
- shared services are explicitly outside runtime ownership
- runtime registry required fields are defined
- promotion code can reference a stable environment model

## Responsibilities

- define `dev`, `test`, and `prod` environments
- define environment mutation policy and operational boundaries
- define shared infrastructure service usage
- define runtime roots and ownership boundaries
- define what may and may not be promoted between environments

## Environments

Supported runtime environments:

```text
dev
test
prod
```

Runtime roots:

```text
~/.remram/runtimes/dev
~/.remram/runtimes/test
~/.remram/runtimes/prod
```

## Shared Infrastructure Services

Shared services:

```text
ollama
opensearch
```

Rules:

- shared services are not duplicated per runtime
- shared services are environment-agnostic
- environment isolation is achieved through configuration, namespaces, and
  release metadata
- runtime containers depend on shared services but do not own their persistent
  state

OpenSearch isolation requirement:

- every environment must use its own index namespace
- example prefixes:
  - `remram-dev-*`
  - `remram-test-*`
  - `remram-prod-*`

## Runtime Responsibilities

Each runtime container is responsible for:

- running the OpenClaw-based execution environment
- reading configuration from its runtime root
- connecting to shared services
- exposing only the interfaces allowed for that environment

Runtime containers must not:

- own deployment logic
- mutate tool installations
- store shared-service persistent state
- embed environment-specific configuration into the image

## Environment Definitions

### DEV

Purpose:

- rapid iteration and debugging

Characteristics:

- feature branches may be deployed here
- direct restart and validation are allowed
- controlled debugging access is allowed
- drift is tolerated

Restrictions:

- DEV is not promotable by itself
- branch state in DEV is not a release candidate

### TEST

Purpose:

- validate a release candidate before approval

Characteristics:

- runs a specific commit candidate
- deployment occurs through pipeline operations only
- validation and smoke tests execute here

Restrictions:

- TEST may not be manually repurposed for ad hoc development
- TEST must be reconstructable from commit and manifest metadata

### PROD

Purpose:

- serve real workloads

Characteristics:

- minimal operational surface
- health and diagnostics remain available
- only approved promotion artifacts may be deployed

Restrictions:

- no direct Codex deployment
- no arbitrary shell access
- no manual runtime mutation outside rollback and approved recovery procedures

## Runtime Registry Requirements

The control plane must maintain a runtime registry containing, at minimum:

- runtime name
- runtime root path
- current deployed commit
- active manifest id
- current tools version
- health status
- last validation result
- current mutation policy

## Interfaces

Consumers of the runtime model:

- control plane
- deployment primitive layer
- validation layer
- promotion pipeline
- Codex MCP tool surface

## Dependencies

- filesystem layout specification
- deployment primitive specification
- promotion pipeline specification

## Example Flows

DEV iteration:

1. deploy branch commit to `dev`
2. restart `dev`
3. validate `dev`
4. inspect logs and iterate

TEST validation:

1. select commit from `main`
2. deploy same commit to `test`
3. run validation and smoke tests
4. generate validation report

PROD promotion:

1. resolve approved TEST artifact
2. snapshot `prod`
3. deploy same commit to `prod`
4. run post-deploy validation
