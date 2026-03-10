# Runtime Tooling

## Purpose

Allow operators and Codex to inspect, validate, debug, and exercise the running
system through CLI-first tooling.

This domain closes the development inner loop by making runtime operations and
runtime debugging practical from the shell and, through the MCP adapter, from AI
tools.

## Capability Provided

This capability provides:

- runtime diagnostics
- runtime health inspection
- log inspection
- operator and Codex access to runtime lifecycle commands
- validation and smoke testing
- debugging tools
- diagnostics bundle collection
- runtime tool packaging
- runtime tool installation, validation, and stable promotion
- global runtime tooling availability across `dev`, `test`, and `prod`

With this capability in place, operators and Codex can inspect, validate,
debug, and exercise runtimes across `dev`, `test`, and `prod` through one
global toolchain.

## Owned Components

The runtime tooling domain owns:

- diagnostics commands
- validation commands
- runtime log access
- operator-facing and Codex-facing wrappers for runtime lifecycle commands
- smoke-test and exercise commands
- debugging commands
- diagnostics bundle commands
- runtime tool package format
- runtime tool installation, validation, promotion, and removal workflow
- runtime tooling version inventory
- operational tooling used by Codex

It does not own:

- base runtime container startup semantics
- shared-service installation
- runtime configuration root creation
- promotion to TEST or PROD

Runtime Tooling may expose runtime lifecycle operations such as start, stop, and
restart through CLI and MCP surfaces, but the behavior of those lifecycle
operations remains owned by the Runtime Stack and Container Deployment System
capabilities.

## Canonical Interfaces

Representative command surface:

- `remram runtime start <env>`
- `remram runtime restart <env>`
- `remram runtime health <env>`
- `remram runtime logs <env>`
- `remram debug collect <env>`
- `remram validate <env>`
- `remram smoke-test <env>`
- `remram tools package`
- `remram tools install <commit>`
- `remram tools test <commit>`
- `remram tools promote <commit>`
- `remram remove-tools-version <commit>`

MCP may expose these capabilities, but only by delegating to the CLI.

If Runtime Tooling exposes lifecycle commands, it does so as an operator and
Codex access surface over the canonical CLI. It must not redefine lifecycle
semantics, target resolution, or startup behavior.

CLI errors must be informative and machine-usable.

Errors must:

- identify the relevant environment or tool version
- state which prerequisite is missing
- tell the caller what to do next

## Tool Packaging Contract

Runtime tooling is packaged before installation.

Package responsibilities:

- define tool categories
- provide manifests
- produce a versioned build artifact suitable for deployment

Expected categories:

- `control`
- `diagnostics`
- `testing`

Tool versions are identified by Git commit ID.

Validation for packaging:

- package build produces an artifact
- package build produces a manifest

## Runtime Tool Deployment Contract

Runtime tooling is deployed centrally rather than per environment.

Rules:

- tools are installed once
- tools are available to `dev`, `test`, and `prod` simultaneously
- runtime commands remain explicitly targeted by environment
- environment behavior differences are controlled by runtime configuration and
  feature flags
- environment behavior differences are not controlled by separate tool
  deployments
- all environments share the same active tool version at a given time
- the active toolchain is exposed through a `stable` symlink
- editors and automation point only to `stable`
- installation of a new tool version must not overwrite an existing installed
  version
- promotion updates `stable` to point at a validated installed version

Installed versions are stored under:

```text
~/.remram/tools/versions/<commit>/
```

Stable toolchain path:

```text
~/.remram/tools/stable -> versions/<commit>
```

Example structure:

```text
~/.remram/tools/
  versions/
    a1b2c3d/
    f8e9a21/
    7e4a12c/
  stable -> versions/a1b2c3d
```

Validation for deployment:

- a deployed version is addressable by commit ID
- multiple tool versions may exist simultaneously
- `stable` points to one validated installed version
- promotion can activate a previously installed validated version
- removal can delete a non-active installed version deterministically

## Tool Validation And Promotion Contract

Runtime tooling validation uses:

```text
remram tools test <commit>
```

The command produces a persistent validation artifact confirming that the
installed toolchain works.

Runtime tooling promotion uses:

```text
remram tools promote <commit>
```

Promotion requirements:

- the tool version exists under `~/.remram/tools/versions/<commit>/`
- a validation artifact exists for that same commit
- the validation artifact indicates success

Promotion effect:

- `~/.remram/tools/stable` points to `versions/<commit>`

Promotion must fail with a clear error if validation is missing or failed.

## Diagnostics And Validation Contract

Runtime tooling commands must return structured JSON suitable for machine
consumption.

These commands may surface:

- runtime lifecycle operations
- runtime health
- runtime logs
- runtime validation
- diagnostics bundles
- smoke-test results

The CLI is the source of truth.

MCP must return the CLI result rather than interpreting it.

MCP must not reimplement runtime logic.

## Runtime Health Contract

Runtime health validation uses:

```text
remram runtime health <env>
```

The command returns a structured JSON health report suitable for machine
consumption.

The report is for runtime targets only.

The report must include, at minimum:

- runtime state
- runtime container status
- gateway health
- shared-service connectivity
- configuration availability

The report may include additional implementation-specific fields, but the
minimum conceptual fields above must be present and stable enough for automation
to rely on them.

Health is the authoritative command for validating whether a runtime is usable.

## Observability Contract

All runtime tooling and runtime operations must log under:

```text
~/.remram/logs/
```

This log root stores:

- tool execution logs
- runtime operation logs
- validation outputs
- diagnostics metadata

## Diagnostics Bundle Contract

Runtime tooling must provide a diagnostics bundle command:

```text
remram debug collect <env>
```

The command gathers, at minimum:

- runtime logs
- runtime configuration
- container state
- environment metadata

The command outputs a diagnostics bundle suitable for troubleshooting.

## Validation

This capability exists when:

- runtime tooling packages can be built
- tools can be installed and versioned by commit ID
- tools can be validated by commit ID
- tools can be promoted to `stable` only after validation succeeds
- operators and Codex can inspect runtime health and logs
- diagnostics bundles can be collected
- diagnostics and validation commands return structured results
- runtime environments can be validated and smoke tested
- a single global toolchain operates across `dev`, `test`, and `prod`

## Deliverable Example

```text
remram tools package
remram tools install a1b2c3d
remram tools test a1b2c3d
remram tools promote a1b2c3d
remram runtime health dev
remram runtime logs prod
remram debug collect dev
remram validate dev
```
