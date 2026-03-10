# Control Plane

## Purpose

Provide the canonical operator interface for the Remram inner loop.

The control plane is the system entrypoint. It owns command routing, process
lifecycle, target discovery, and the MCP adapter that exposes CLI commands to
Codex and other tool clients.

## Capability Provided

This capability provides:

- a canonical CLI executable available on `PATH`
- a long-running control-plane service process
- command routing for runtime, shared-service, deployment, and tooling
  operations
- registry-backed target discovery
- an MCP adapter that delegates to the CLI

The control plane itself does not perform arbitrary shell execution and does not
embed runtime-specific business logic inside MCP.

## Target Model

Targets represent managed objects addressable through the control plane.

For the current system design, the canonical targets are:

- runtime targets:
  - `control`
  - `dev`
  - `test`
  - `prod`
- shared-service targets:
  - `ollama`
  - `opensearch`
  - `caddy`

Runtime targets correspond to runtime configuration roots and runtime container
instances.

Shared-service targets correspond to shared infrastructure components.

`caddy` is the shared ingress proxy for the Remram system. It is responsible
for HTTPS termination, trusted local certificate handling, and hostname routing
to control-plane and runtime services. It is infrastructure and is not owned by
the control-plane capability.

Example runtime roots:

```text
~/.openclaw/dev
~/.openclaw/test
~/.openclaw/prod
```

Rules:

- targets may represent runtime environments or shared services
- the control plane maintains a target registry
- canonical target identifiers are short stable ids such as `control`, `dev`,
  `test`, `prod`, `ollama`, `opensearch`, and `caddy`
- runtime target metadata may include canonical runtime container identities
  such as `openclaw-control`, `openclaw-dev`, `openclaw-test`, and
  `openclaw-prod`
- shared-service targets include `ollama`, `opensearch`, and `caddy`
- user-facing aliases may map `cli` to `control` and `prime` to `prod`
- other capability domains populate or manage those targets
- the control plane discovers and reports them
- later domains may define lifecycle, validation, and diagnostics commands for
  either target class, but the control plane remains the canonical routing
  surface

## Owned Components

The control plane owns:

- the `remram` CLI entrypoint
- control-plane configuration loading
- path resolution for control-plane state
- process lifecycle for `serve`
- target listing and target status reads
- CLI-to-MCP command mapping
- control-plane logs and service state

It does not own:

- deployment asset rendering semantics
- shared-service installation
- runtime container startup details
- production promotion policy

Those capabilities live in the later domain specs.

## Canonical Interfaces

Canonical executable name:

- `remram`

Representative command surface:

- `remram version`
- `remram health`
- `remram serve`
- `remram list-targets`
- `remram status --target <target>`

Rules:

- the CLI must be callable from `PATH`
- operational commands return machine-readable JSON on stdout
- the CLI remains usable locally or over SSH
- all exposed services must be accessed over HTTPS
- `version` and `health` must function even when `serve` is not running unless
  explicitly documented otherwise
- CLI errors must be informative and machine-usable
- errors and failures must be actionable for operators and automated agents
- errors must state what failed, include the relevant identifier when possible,
  and tell the caller what to do next

Version resolution:

1. build-injected version string
2. git commit SHA
3. `dev`

## Exit Code Contract

CLI exit codes must remain stable for automation and MCP clients.

Exit code meanings:

- `0`: success
- `1`: generic execution failure
- `2`: configuration error
- `3`: control-plane unavailable
- `4`: target not found
- `5`: validation failure

Rules:

- CLI commands must return structured JSON on stdout
- human-readable errors may be written to stderr
- exit codes must remain stable for automation and MCP clients
- exit code `0` means the CLI executed successfully, not that the system is
  healthy
- agents must evaluate `ok` and `status` to determine operational state
- error payloads must explain how the caller can recover

## Path And State Contract

The control plane resolves paths using deterministic precedence:

```text
CLI flags
  -> environment variables
  -> config file values
  -> compiled defaults
```

Default configuration path:

```text
~/.remram/control-plane/config.yaml
```

Default control-plane state root:

```text
~/.remram/
  control-plane/
  logs/
  state/
```

The target registry persists under:

```text
~/.remram/state/targets/
```

Target registry records use a minimal extensible schema:

- `id`
- `target_class` (`runtime` | `shared_service`)
- `display_name`
- `runtime_root` or `service_name`
- `container_name` (optional)
- `created_at`
- `updated_at`
- `metadata` (extensible object)

Initial targets:

- runtime:
  - `control`
  - `dev`
  - `test`
  - `prod`
- shared services:
  - `ollama`
  - `opensearch`
  - `caddy`

Rules:

- repository root detection must not rely on arbitrary current working
  directory
- repository source paths and host operational state paths remain distinct
- presence of repository files does not imply runtime state exists
- the control plane may initialize `~/.remram` safely if it is absent

## Hostname Routing Model

Remram uses deterministic hostnames so operator tooling, MCP clients, Windows
tooling, and local development workflows can reach stable HTTPS endpoints.

Canonical HTTPS hostnames:

- `moltbox-cli`
- `moltbox-dev`
- `moltbox-test`
- `moltbox-prod`

Local development requires hosts-file entries mapping these names to loopback.

Example:

```text
127.0.0.1  moltbox-cli
127.0.0.1  moltbox-dev
127.0.0.1  moltbox-test
127.0.0.1  moltbox-prod
```

Rules:

- deterministic hostnames are part of the local operator contract
- `moltbox-cli` is the canonical control-plane HTTPS hostname
- `moltbox-prod` is the canonical production runtime hostname
- `moltbox-dev` and `moltbox-test` are the canonical non-production runtime
  hostnames
- required hostnames must be present in the local hosts file
- Caddy routes each hostname to the appropriate internal service port
- hostname routing provides stable endpoints independent of process-local port
  choices
- Windows and Visual Studio tooling must be able to reach local services
  through these HTTPS hostnames

## Serve Lifecycle Contract

`remram serve` defines the long-running lifecycle of the control-plane process.

Rules:

- `serve` runs in the foreground
- process managers such as Docker or systemd handle daemonization
- startup failures exit non-zero
- `SIGINT` and `SIGTERM` trigger graceful shutdown
- readiness is reached only after config and path resolution succeed
- a `ready` event must be logged when the service becomes operational
- bind, port, or config conflicts cause deterministic startup failure

Ingress model:

```text
CLI executable (always available)
  -> optional running control-plane process (`remram serve`)
  -> HTTPS ingress through Caddy
```

Rules:

- the CLI executable is always available even when `remram serve` is not
  running
- the control-plane service may bind to a local internal port
- external operator, CLI, and MCP access must occur through Caddy
- direct HTTP access is not part of the external contract
- internal bind address and port remain implementation details unless
  explicitly surfaced by later domains
- if an internal port is configurable, config and flags must determine it using
  the standard precedence rules
- all exposed services use HTTPS with TLS terminated by Caddy
- Caddy uses a trusted local root certificate
- if a trusted local root certificate does not exist, environment setup must
  generate or install one
- health endpoints must be exposed through Caddy from day one

Operational states:

- starting
- ready
- stopping
- stopped
- failed

## Health Contract

`remram health` is the authoritative health interface and defines the common
health model that Remram subsystems may reuse.

Required health fields:

- `ok`
- `status`
- `version`
- `serve_state` (when applicable)
- `config_status`
- `paths`
- `started_at` or `uptime`
- `logs`
- `error_message`
- `recovery_message`

Structured log entry fields:

- `name`
- `path`
- `tail`

Behavior:

- `ok` represents system readiness
- if a serve process exists, `remram health` reports live process state
- implementations may use local IPC or internal local transport to obtain
  running state
- CLI output must remain identical regardless of mechanism
- service communication must remain local-only
- maximum CLI wait time for service communication is 2 seconds
- if communication with the running service fails, the CLI must degrade
  gracefully
- health output must reflect control-plane unavailability when the running
  service cannot be reached
- `remram health` returns exit code `0` for reachable control-plane responses,
  including degraded and `down` states
- reachable degraded responses use `ok: false` when appropriate
- if `remram serve` is not running, `serve_state` is `down`
- if `remram serve` is not running, the response must include the persisted
  serve log path, a bounded recent log tail, an `error_message`, and a
  `recovery_message`
- `recovery_message` must tell the caller how to recover, for example by
  starting `remram serve` or inspecting logs
- health responses include a `logs` array
- each log entry includes `name`, `path`, and `tail`
- log tails in health responses are deterministic and bounded to the last `50`
  lines

HTTPS health endpoint contract:

- it must be exposed through the Caddy-routed HTTPS hostname model
- the control-plane health endpoint is `https://moltbox-cli/health`
- it must expose `GET /health`
- it must return the same health model as the CLI
- HTTPS routing support must exist from day one even if some routes are not yet
  used by all clients

## MCP Adapter Contract

MCP is an adapter over the CLI, not a second control plane.

Required execution flow:

```text
MCP request
  -> MCP adapter
  -> remram CLI command
  -> structured CLI result
  -> MCP response
```

Rules:

- every MCP tool maps to a single CLI command
- MCP tools must not rename CLI flags or change argument semantics
- MCP must not perform business logic or custom error semantics
- CLI commands invoked by MCP return JSON on stdout
- MCP returns the parsed CLI JSON directly
- MCP failures follow the same recovery-oriented error structure as the CLI
- non-zero CLI exits are propagated as errors with exit code, stderr, and
  executed command context
- shell execution such as `sh -c` is prohibited

## Target Registry Contract

The target registry is responsible for:

- listing targets
- resolving target identifiers
- reading target runtime status

Rules:

- the registry persists under `~/.remram/state/targets/`
- the registry contains metadata for runtime targets and shared-service targets
- canonical target identifiers are `control`, `dev`, `test`, `prod`,
  `ollama`, `opensearch`, and `caddy`
- aliases map `cli` to `control` and `prime` to `prod`
- supported shared-service target identifiers include `ollama`, `opensearch`,
  and `caddy`
- lifecycle and validation semantics may differ by target class
- later domains manage target creation, lifecycle, and validation behavior

## Logging Contract

The control plane writes structured logs to stdout and stderr.

Preferred format:

- JSON lines

Minimum log fields:

- `ts`
- `level`
- `event`
- `component`
- `msg`

Required events:

- `startup`
- `ready`
- `shutdown`
- `fatal_error`

Rules:

- structured logs are emitted to stdout and stderr
- persisted control-plane logs under `~/.remram/logs/` are required
- health responses include structured log references in the `logs` array
- file logging must never replace stream logging

## Recovery-Oriented Error Contract

All CLI and MCP failures must support automated recovery.

Minimum failure fields:

- `error_type`
- `error_message`
- `recovery_message`

Rules:

- failures must clearly state what failed
- failures must identify the missing resource or configuration
- failures must include relevant identifiers when applicable
- failures must suggest the next corrective action

## Validation

This capability exists when:

- `remram version` returns a stable version string
- `remram health` returns a valid health report
- `remram serve` starts and reaches ready state
- a second `remram serve` attempt fails deterministically on bind conflict
- invalid config causes non-zero health or serve failure
- CLI commands return machine-readable JSON
- invalid targets return exit code `4`
- the CLI works when `serve` is not running
- `remram health` returns `serve_state: down` with exit code `0` when the
  serve process is not running
- health responses include the `logs` array with bounded tails
- failure responses include recovery-oriented error fields
- config precedence follows flags -> environment -> config -> defaults
- MCP can call read-only CLI-backed commands without diverging from CLI output

## Deliverable Example

```text
remram version
remram health
remram serve
remram list-targets
```
