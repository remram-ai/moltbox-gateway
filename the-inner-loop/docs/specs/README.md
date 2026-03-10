# The Inner Loop Capability Specs

This directory defines the Remram inner loop as five capability domains rather
than as many small milestones.

These documents are the primary implementation references for the system.

## Capability Taxonomy

The inner loop is organized into five functional domains:

1. Control Plane
2. Container Deployment System
3. Runtime Stack
4. Runtime Tooling
5. Deployment Pipeline

Each spec describes:

- what capability the domain provides
- what CLI commands or interfaces exist
- what components the domain owns
- what validation proves the capability exists

## Architectural Rules

These rules apply across all five specs:

- CLI-first architecture: the CLI is the real system interface
- MCP is a thin adapter that calls the CLI
- MCP does not contain business logic
- commands operate on explicit identifiers such as target ids, commit hashes,
  snapshot ids, and tool versions
- moving references such as `HEAD` or branch names are rejected where immutable
  identifiers are required
- the same CLI command surface exists across `dev`, `test`, and `prod`
- environment restrictions are enforced through policy, not by removing CLI
  verbs
- `ollama` and `opensearch` are shared services, not per-runtime services
- deployment behavior is composed from deterministic commands and structured
  outputs
- machine-readable JSON is the default CLI output format for operational
  commands
- CLI errors are informative and machine-usable, explain what requirement is
  missing, and tell the caller what to do next

## Intended System Progression

The system capability progression is:

```text
Control Plane
  -> Container Deployment System
  -> Runtime Stack
  -> Runtime Tooling
  -> Deployment Pipeline
```

This means:

- the control plane provides the canonical `remram` operator interface
- the deployment system teaches that interface how to render and execute
  container operations
- the runtime stack makes shared services and OpenClaw environments real
- runtime tooling closes the development inner loop
- the deployment pipeline promotes validated changes from development to
  production

## Spec Map

### `01_control_plane.md`

Defines the canonical operator interface, control-plane process lifecycle,
command routing, the target registry for runtime and shared-service targets, and
the MCP adapter contract.

### `02_container_deployment_system.md`

Defines how the control plane renders deployment assets, resolves deployment
targets, invokes host-side primitives, and performs container lifecycle
actions.

### `03_runtime_stack.md`

Defines the deployed system components: shared services, runtime configuration
roots, runtime identities, shared-service connectivity, and OpenClaw runtime
containers.

### `04_runtime_tooling.md`

Defines operator and agent tooling for diagnostics, health, validation, logs,
tool deployment, and stable toolchain promotion.

### `05_deployment_pipeline.md`

Defines commit candidate lifecycle, TEST validation proof, PR-gated PROD
promotion, and production snapshot/rollback capability.
