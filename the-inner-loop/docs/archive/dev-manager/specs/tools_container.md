# Tools Container Specification

## Purpose

Define the tools container or tools service that hosts the control-plane logic.

This component is the operational front door for the new `dev-manager`
architecture.

## Capability Definition

Capability name:

- Versioned tools container or service runtime

Implementation order:

- 6

Depends on:

- `filesystem_layout.md`
- `repository_structure.md`
- `deployment_primitives.md`
- `control_plane.md`

Delivers:

- the runtime packaging model for the control plane
- version-switching rules for `stable` and `rc`
- mount and access boundaries for the tools runtime

Primary implementation reference for:

- control-plane packaging
- tool version promotion and rollback
- host-tool mounting and invocation boundaries

Acceptance criteria:

- tool version layout is defined
- stable and rc switching behavior is defined
- required access paths are defined
- arbitrary shell access remains prohibited

## Responsibilities

- run the control-plane service
- expose MCP tools for Codex
- maintain runtime registry access
- maintain deployment and validation state
- call allowlisted host-side primitives
- load versioned tool installations through `stable` or `rc`

## Boundaries

The tools container:

- is separate from runtime containers
- does not own runtime state
- does not directly run privileged host mutation logic
- does not expose arbitrary shell or generic subprocess tools

Privileged or sensitive host mutation is delegated to host-side scripts under
`~/.remram/host-tools`.

## Required Mounts or Access Paths

The tools container or service must be able to read:

- `~/.remram/deploy`
- `~/.remram/logs/control-plane`
- `~/.remram/runtimes` metadata
- `~/.remram/tools/stable` or `~/.remram/tools/rc`

The tools container or service must be able to invoke:

- allowlisted host-side primitives from `~/.remram/host-tools`

The tools container or service must not require direct write access to runtime
data directories except through approved orchestration flows.

## Internal Responsibilities

### Tool Routing

- map external MCP/API requests to approved internal operations
- reject tools outside the allowlist

### Version Awareness

- record active tool version
- include tool version in publish reports
- support switching between `stable` and `rc`

### State Management

- persist deployment run metadata
- persist validation metadata
- persist approval state

### Access Control

- separate read-only tools from mutating tools
- block direct production deployment paths for Codex

## Tool Versioning Model

Required structure:

```text
tools/
  versions/
    vX.Y.Z/
  stable -> versions/vX.Y.Z
  rc -> versions/vX.Y.Z
```

Rules:

- `stable` is the default live control-plane toolchain
- `rc` is the candidate toolchain under test
- switch operations must be explicit and logged
- a failed `rc` verification must restore the previous `stable`

## Dependencies

- control-plane specification
- filesystem layout specification
- deployment primitive specification
- host tool allowlist

## Example Flows

Self-upgrade verification:

1. install candidate bundle into `tools/versions/v0.4.0`
2. repoint `rc`
3. run control-plane self-checks through `rc`
4. promote `stable` only if checks pass

Codex request flow:

1. Codex calls an MCP tool
2. tools container resolves runtime policy
3. tools container invokes an allowlisted primitive or registry read
4. tools container returns structured output
