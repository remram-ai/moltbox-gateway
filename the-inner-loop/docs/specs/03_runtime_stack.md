# Runtime Stack

## Purpose

Deploy the infrastructure and runtime environments that make the Remram inner
loop real.

This domain defines the runtime environments, shared-service expectations,
runtime identities, runtime configuration roots, and shared-infrastructure
connectivity used by the system.

It assumes earlier capabilities already provide the control-plane interface and
the container deployment system.

## Capability Provided

This capability provides:

- runtime target definitions for `dev`, `test`, and `prod`
- runtime configuration roots
- runtime startup against already-defined shared services
- runtime status inspection

It makes the system runnable, but does not yet provide the full developer
tooling or promotion workflow.

It does not provide:

- shared-service installation
- runtime container installation
- container image creation
- deployment asset rendering
- feature or application deployment

## Owned Components

The runtime stack owns:

- shared services:
  - `ollama`
  - `opensearch`
- runtime targets:
  - `dev`
  - `test`
  - `prod`
- runtime container identities
- runtime configuration roots

It does not own:

- container image build or installation
- deployment asset rendering
- diagnostics commands
- runtime health inspection tooling
- runtime log inspection
- validation or smoke-test commands
- debugging tools
- diagnostics bundle collection
- packaging of runtime tools
- CLI or MCP tooling version promotion
- MCP exposure rules
- production approval boundaries

Container installation and mutation remain the responsibility of the Container
Deployment System capability.

CLI and MCP tooling are versioned independently from the runtime stack. The
tooling deployment model belongs to later tooling capabilities and does not
change the runtime contracts defined here.

## Shared Services Contract

`ollama` and `opensearch` are installed once and shared across all runtimes.

They must not be installed separately per runtime.

Runtime commands may:

- verify shared-service health
- start a shared service if it is already defined and installed

Runtime commands must not:

- install shared services
- render shared-service deployment assets
- create shared-service definitions

Runtime containers connect to these shared services after startup.

Minimum shared-service health checks:

- `GET http://localhost:11434/api/tags` returns HTTP 200
- `GET http://localhost:9200/_cluster/health` returns HTTP 200

## Runtime Registry Contract

Registry definitions are stored as static files under a control-plane registry
directory.

Each file defines one managed target.

The control-plane CLI loads these definitions at runtime.

Control-plane-owned registry directory:

```text
~/.remram/state/targets/
```

Valid target ids:

- `dev`
- `test`
- `prod`
- `ollama`
- `opensearch`

Target classes:

- runtime targets:
  - `dev`
  - `test`
  - `prod`
- shared-service targets:
  - `ollama`
  - `opensearch`

Rules:

- one registry file exists per target
- target ids are unique
- each file defines target type and target metadata
- target ids are the same identifiers used by `--target`
- runtime targets and shared-service targets may share one registry but remain
  distinct target classes
- commands may apply different validation behavior by target class
- health probes may be defined per target but are optional
- absence of a health probe must not cause command failure

## Runtime Configuration Contract

Each runtime reads configuration from its runtime root:

```text
~/.openclaw/<env>
```

Examples:

```text
~/.openclaw/dev
~/.openclaw/test
~/.openclaw/prod
```

The runtime container must read configuration from the selected runtime root.

The runtime stack assumes:

- the runtime image already exists
- rendered runtime configuration already exists
- shared services are already installed

Rules:

- the runtime stack does not create runtime roots
- runtime roots must exist before runtime start is attempted
- missing runtime roots cause runtime start failure
- missing rendered configuration causes runtime start failure

## Runtime Container Identity Contract

Runtime container identity must be derived deterministically from the runtime
environment identifier.

Canonical naming pattern:

```text
openclaw-<env>
```

Examples:

- `openclaw-dev`
- `openclaw-test`
- `openclaw-prod`

Rules:

- the same environment always resolves to the same container identity
- different environments must resolve to different container identities
- implementations must not invent environment-specific aliases for the same
  runtime container

## Canonical Interfaces

Representative command surface:

- `remram list-targets`
- `remram status --target <target>`
- `remram runtime start <env>`

Where `<env>` is one of:

- `dev`
- `test`
- `prod`

Rules:

- the same runtime command surface exists across environments
- environment selection is explicit in the command
- authorization policy determines whether a command is allowed, not command
  presence
- `remram status --target <target>` performs structural inspection
- CLI errors must be informative and machine-usable
- errors must identify the target or environment, state the missing
  prerequisite, and tell the caller what to do next

Status is the authoritative command for inspecting whether a target is defined,
present, or active.

Runtime start exit codes:

- `0`: runtime start completed successfully
- `1`: runtime is already running and no new start was performed
- `2`: required runtime configuration or runtime root is missing
- `3`: runtime start was attempted but did not reach a started state

Registry status states:

- `defined`: target exists in the registry but no runtime artifacts are present
- `not_initialized`: expected root or service path does not exist
- `rendered`: runtime configuration exists but runtime is not running
- `running`: runtime or service process is detected as active
- `degraded`: runtime exists but health indicates failure
- `unknown`: the control plane cannot determine state

## Validation

This capability exists when:

- shared-service health endpoints are reachable
- `remram list-targets` returns the required shared-service and runtime targets
- a runtime environment can be started with `remram runtime start <env>`
- `remram runtime start <env>` fails when the runtime root or rendered
  configuration is missing
- runtime container identity is deterministic for each environment
- status inspection remains read-only and does not mutate the system

## Deliverable Example

```text
remram runtime start dev
remram status --target ollama
```
