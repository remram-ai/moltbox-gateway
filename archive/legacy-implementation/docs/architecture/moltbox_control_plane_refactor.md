# Moltbox Control-Plane Refactor

Status: proposed

This document is the architecture-level summary for the Moltbox CLI and control-plane refactor.

The detailed gateway specification lives in:

```text
docs/specs/08_cli_refactor_control_plane.md
docs/specs/09_runtime_monitor_api.md
docs/specs/10_cli_refactor_implementation_plan.md
```

## 1. Problem Statement

The current Moltbox control plane works, but it still depends too heavily on:

- SSH as a normal diagnostic path
- direct Docker inspection procedures in runbooks
- container-local OpenClaw CLI execution for runtime behavior
- CLI-owned infrastructure logic instead of a thin orchestration layer

That is not the steady-state architecture.

## 2. Target Architecture

The target architecture is:

```text
local CLI
  -> shared control-plane service layer
     -> docker proxy adapter
     -> runtime API client
     -> deployment renderer/state store

tools MCP server
  -> same control-plane service layer
```

Design rules:

- the CLI stays thin
- the tools container hosts the only Moltbox MCP server
- Docker remains the infrastructure authority
- runtime behavior is exposed through runtime APIs
- every capability has a CLI surface and a diagnostic surface

## 3. Ownership Model

Infrastructure ownership:

- `moltbox tools ...`
- `moltbox host ...`

Runtime ownership:

- `moltbox runtime <environment> ...`
- runtime-owned capability namespaces under the environment

Examples:

- `moltbox host topology`
- `moltbox runtime dev monitor`
- `moltbox runtime dev channels discord doctor`

## 4. Operational Consequences

After the refactor:

- normal status and inspect commands use the Docker proxy
- runtime readiness and doctor flows use runtime APIs
- remote operators use the tools MCP surface instead of SSH
- break-glass shell access remains possible but is no longer the primary operating model

## 5. Acceptance Bar

The architecture is only complete when:

- normal diagnostics do not require SSH
- runtime behavior no longer depends on `docker exec`
- local CLI and MCP share one service implementation path
- the runtime monitor contract exists for every managed runtime
