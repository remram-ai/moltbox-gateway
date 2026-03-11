# CLI Refactor Implementation Plan

Status: proposed

This document sequences the Moltbox CLI and control-plane refactor defined in:

```text
docs/specs/08_cli_refactor_control_plane.md
```

## 1. Phase 1: Service-Layer Refactor

Goal:

- establish one control-plane service path for CLI and MCP

Modules affected:

- `moltbox-cli/tools/src/moltbox_cli/cli.py`
- `moltbox-cli/tools/src/moltbox_cli/deployment_service.py`
- `moltbox-cli/tools/src/moltbox_cli/mcp_adapter.py`
- `moltbox-cli/tools/src/moltbox_cli/http_app.py`

Deliverables:

- explicit service objects for tools, host, runtime, Docker proxy, and runtime API clients
- existing commands routed through the new service layer

## 2. Phase 2: Docker Proxy

Goal:

- replace scattered Docker helpers with one allowlisted adapter

Modules affected:

- current primitive scripts
- inspection helpers
- deployment and rollback paths

Deliverables:

- normalized inspect and log payloads
- topology support
- stable Docker failure mapping

## 3. Phase 3: Runtime Operator APIs

Goal:

- move runtime behavior out of `docker exec` workflows

Modules affected:

- `moltbox/config/openclaw/extensions/remram-runtime/`
- runtime command handlers

Deliverables:

- `/moltbox/monitor`
- `/moltbox/doctor`
- `/moltbox/models/status`
- `/moltbox/routing/status`
- `/moltbox/config/inspect`
- `/moltbox/chat`

## 4. Phase 4: CLI Tree Expansion

Goal:

- expose the new architecture through a stable operator surface

Modules affected:

- CLI parser
- runtime command modules
- operator documentation

Deliverables:

- `moltbox host topology`
- `moltbox host doctor`
- `moltbox runtime <environment> monitor`
- `moltbox runtime <environment> doctor`
- `moltbox runtime <environment> models status`
- `moltbox runtime <environment> routing status`
- `moltbox runtime <environment> config inspect`
- runtime-owned capability namespaces such as `channels discord`

## 5. Phase 5: MCP Convergence

Goal:

- make tools MCP a policy-gated wrapper around the same service layer

Modules affected:

- MCP server
- policy layer

Deliverables:

- no MCP-specific business-logic drift
- remote read-only diagnostics for runtime and host inspection
- preserved policy gating for mutation verbs

## 6. Phase 6: Documentation and Runbook Migration

Goal:

- remove SSH and raw Docker from the normal operational path

Modules affected:

- CLI reference
- manual operations docs
- capability docs

Deliverables:

- updated operator documentation
- break-glass Docker/SSH guidance moved to recovery-only sections

## 7. Testing Strategy

Required test coverage:

- unit tests for Docker proxy normalization and error mapping
- unit tests for runtime API clients
- parser tests for the expanded command tree
- MCP policy tests for allowed and denied verbs
- integration tests for:
  - deploy
  - inspect
  - logs
  - topology
  - runtime monitor
  - runtime doctor
  - runtime chat
  - runtime-owned capability diagnostics

## 8. Migration Strategy

Migration order:

1. land the service layer without changing the public grammar
2. replace internal Docker helpers with the Docker proxy
3. add runtime monitor and operator endpoints
4. switch runtime behavior commands to runtime APIs
5. add new CLI groups and subcommands
6. update MCP to the converged service path
7. update docs and deprecate SSH-first runbooks

## 9. Exit Criteria

The refactor is ready for completion review when:

- local CLI and MCP share one implementation path
- runtime behavior no longer depends on `docker exec`
- the runtime monitor contract is live
- normal diagnostics do not require SSH
