# CLI Surface Review

Status: Draft review
Date: 2026-04-04

This review covers the current live CLI contract in `moltbox-gateway` and evaluates it against the lighter-weight service-plane-plus-OpenClaw direction.

## Scope

Reviewed sources:

- parser and help text in [`../../pkg/cli/cli.go`](../../pkg/cli/cli.go)
- CLI entrypoint in [`../../cmd/moltbox/main.go`](../../cmd/moltbox/main.go)
- HTTP client routing in [`../../internal/client/http.go`](../../internal/client/http.go)
- gateway handlers in [`../../internal/gateway/http.go`](../../internal/gateway/http.go)
- SSH wrapper policy in [`../../internal/sshwrap/sshwrap.go`](../../internal/sshwrap/sshwrap.go)
- gateway update and host install logic in [`../../internal/orchestrator/manager.go`](../../internal/orchestrator/manager.go)
- config defaults and config-path lookup in [`../../internal/config/config.go`](../../internal/config/config.go)
- SSH provisioning helper in [`../../scripts/provision-automation-ssh.sh`](../../scripts/provision-automation-ssh.sh)
- current contract docs in [`../../../remram/docs/overview/cli-architecture.md`](../../../remram/docs/overview/cli-architecture.md) and [`../../../remram/docs/audits/cli-contract-audit.md`](../../../remram/docs/audits/cli-contract-audit.md)

Live appliance spot-check performed on 2026-04-04:

- SSH alias `moltbox` currently reaches host `moltbox-prime`
- live login still lands as personal user `jpekovitch`
- live services currently include `gateway`, `caddy`, `ollama`, `openclaw-dev`, `openclaw-test`, `openclaw-prod`, and `opensearch`
- all live containers are healthy, but none currently declare Docker memory or CPU limits
- both live `test` and `prod` answer `moltbox <env> openclaw agent --agent main ... --json` through the CLI path and report `provider=ollama` and `model=mistral:7b-instruct-32k`
- the live box still warns that `plugins.allow` is empty and an untracked local `behavior-guard` extension is auto-loading

## Current Implemented Surface

| Resource | Current commands | Backend shape | Current status |
| --- | --- | --- | --- |
| `gateway` | `status`, `update`, `logs`, `mcp-stdio`, `token ...`, `service deploy|restart|status <service>`, `docker ping`, `docker run <image>` | parser + explicit HTTP routes + local MCP stdio special-case | implemented |
| `dev` | `reload`, `checkpoint`, `skill ...`, `plugin ...`, `openclaw <command>`, `secrets ...` | explicit runtime routes plus scoped secrets fallback | implemented |
| `test` | same as `dev` | same as `dev` | implemented |
| `prod` | same as `dev` | same as `dev` | implemented |
| `service` | `secrets ...` only | `/execute` scoped-secrets path | implemented but misleading |
| `ollama` | native passthrough | `/service/passthrough` | implemented |
| `opensearch` | native passthrough | `/service/passthrough` | implemented but now out of scope |
| `caddy` | native passthrough | `/service/passthrough` | implemented |
| retired namespaces | `runtime`, `skill`, `tools`, `host`, `openclaw-dev`, `openclaw-test`, `openclaw-prod` | parser rejection | implemented |

## Findings

### 1. The public CLI still reflects the old heavyweight runtime model

Evidence:

- [`../../pkg/cli/cli.go`](../../pkg/cli/cli.go) still exposes `dev`, `checkpoint`, `skill`, `plugin`, `opensearch`, and gateway Docker helpers.
- [`../../../remram/docs/overview/cli-architecture.md`](../../../remram/docs/overview/cli-architecture.md) still documents the same surface.

Why it matters:

- the public surface still assumes replay-heavy runtime management
- the CLI shape still embeds the appliance `dev` environment you now want removed
- `opensearch` is still treated as first-class even though the system is pivoting away from it

Verdict:

- this is not a small cleanup
- the contract itself needs to change

### 2. `test` and `prod` expose mutating surfaces that do not match the desired AI access model

Evidence:

- [`../../pkg/cli/cli.go`](../../pkg/cli/cli.go) parses `reload`, `checkpoint`, `skill deploy`, `plugin install`, and `secrets set` for `test` and `prod`.
- [`../../internal/sshwrap/sshwrap.go`](../../internal/sshwrap/sshwrap.go) only restricts these commands in `bootstrap` mode.
- `automation` mode allows any `moltbox` command after shell-token validation.

Why it matters:

- parser shape and wrapper policy do not align with the desired split of "open access for test AI" versus "debug and CLI-only access for prod AI"
- the current restriction model is tied to old account names and wrapper modes rather than a clear environment policy

Verdict:

- current SSH policy is too implicit and too tied to legacy identities

### 3. The CLI still leaks user-home and user-owned installation assumptions

Evidence:

- [`../../deploy/README.md`](../../deploy/README.md) still says the host CLI is installed at `~/.local/bin/moltbox`.
- [`../../internal/config/config.go`](../../internal/config/config.go) searches user-home config locations before falling back to `/etc/moltbox/config.yaml`.
- [`../../internal/orchestrator/manager.go`](../../internal/orchestrator/manager.go) derives host config placement from the CLI path and infers secret ownership from the CLI install directory owner.
- tests still contain `jpekovitch` home-path fixtures in [`../../internal/config/config_test.go`](../../internal/config/config_test.go).

Why it matters:

- this directly conflicts with the target requirement that everything be system-owned
- user-home fallbacks make the host state model less deterministic
- ownership inference from the CLI path is the wrong rule for secrets

Verdict:

- this is a real hygiene problem, not just a documentation problem

### 4. The top-level `service` namespace is overloaded and misleading

Evidence:

- [`../../pkg/cli/cli.go`](../../pkg/cli/cli.go) accepts `service` only for scoped secrets.
- older internal and archived docs still talk about a broader service lifecycle surface.

Why it matters:

- `service` sounds like the place to manage the service plane
- in reality, service lifecycle lives under `gateway service ...`
- that makes the operator surface harder to explain than it needs to be

Verdict:

- move service-plane lifecycle to top-level `service`
- move secrets to a separate top-level namespace

### 5. There are still partial or dead compatibility surfaces

Evidence:

- parser rewrites `skill rollback` to `remove` in [`../../pkg/cli/cli.go`](../../pkg/cli/cli.go)
- gateway still has an explicit rollback handler in [`../../internal/gateway/http.go`](../../internal/gateway/http.go)
- the current audit already calls this out as a hidden compatibility surface in [`../../../remram/docs/audits/cli-contract-audit.md`](../../../remram/docs/audits/cli-contract-audit.md)

Why it matters:

- compatibility behavior exists without a clean contract
- dead or shadowed handler paths make the surface harder to reason about

Verdict:

- either remove compatibility aliases or document them explicitly as temporary

### 6. Plugin lifecycle surfaces are implemented, but they reinforce the wrong source of truth

Evidence:

- [`../../internal/orchestrator/runtime_state.go`](../../internal/orchestrator/runtime_state.go) implements `RuntimePluginInstall`, `RuntimePluginList`, and `RuntimePluginRemove`.
- `RuntimePluginList` reports gateway-derived replay state rather than native OpenClaw plugin inventory.

Why it matters:

- the commands work, but they encode the very integration model being replaced
- keeping them as first-class CLI verbs would preserve the wrong boundary

Verdict:

- remove these from the new public CLI
- use official OpenClaw plugin commands through `test openclaw ...` and `prod openclaw ...`

### 7. The current CLI is missing a few surfaces that matter for the lighter model

Missing or underspecified for the new target:

- a minimal `bootstrap gateway` command for a clean host
- top-level `service list`
- top-level `service logs <service>`
- a clean top-level secret scope such as `secret set|list|delete <scope>`
- explicit appliance-health checks around backups, restore points, and patch posture

Verdict:

- the new CLI should add these surfaces and remove more complex ones at the same time

## Recommended Replacement Shape

### Keep

- `gateway status`
- `gateway logs`
- `gateway update`
- `test openclaw <command>`
- `prod openclaw <command>`
- `ollama <native command>`

### Move

- service lifecycle from `gateway service ...` to top-level `service ...`
- secrets from `service secrets ...` and `<env> secrets ...` to top-level `secret ...`

### Remove

- `dev`
- `checkpoint`
- `skill ...`
- `plugin ...`
- `opensearch`
- `gateway docker ping`
- `gateway docker run <image>`

### Add

- `bootstrap gateway`
- `service list`
- `service status <service>`
- `service deploy <service>`
- `service restart <service>`
- `service logs <service>`
- `secret set <scope> <NAME> [VALUE]`
- `secret list <scope>`
- `secret delete <scope> <NAME>`

## Current Surface Verdict

The current CLI is coherent enough to operate the old architecture, but it is too wide, too replay-centric, too tied to legacy identities, and too permissive in the wrong places.

The lighter CLI should stop pretending to manage OpenClaw lifecycle itself. It should manage the service plane, pass through to official OpenClaw and Ollama CLIs, and make host ownership and AI access policy explicit.
