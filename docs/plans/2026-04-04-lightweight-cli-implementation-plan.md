# Lightweight CLI And Appliance Refactor Plan

Status: Draft implementation plan
Date: 2026-04-04

This is an earlier scoping plan. Use [`./2026-04-04-clean-moltbox-execution-plan.md`](./2026-04-04-clean-moltbox-execution-plan.md) as the primary builder-facing implementation package.

This plan implements the lighter Moltbox appliance model:

- no appliance `dev`
- no OpenSearch
- `test` and `prod` only
- service-plane management plus official OpenClaw CLI access
- keep `ollama/mistral:7b-instruct-32k` as the local default model for both runtimes
- keep Together escalation available through OpenClaw plugin/provider surfaces, not gateway hard-coding
- fully system-owned host paths
- explicit AI SSH identities
- backup and patching hygiene handled as host operations, not as replay-era runtime logic

Related review:

- [`../reviews/2026-04-04-cli-surface-review.md`](../reviews/2026-04-04-cli-surface-review.md)
- [`../decisions/2026-04-04-openclaw-operating-model.md`](../decisions/2026-04-04-openclaw-operating-model.md)

## Target Outcomes

After this refactor, the appliance should contain only:

- `gateway`
- `caddy`
- `ollama`
- `openclaw-test`
- `openclaw-prod`

The CLI should be lightweight enough to explain in one screen and strong enough to bootstrap and operate the appliance without shelling out to ad hoc Docker commands.

## Functional End State

The refactor is not done until all of these are true at the same time:

- `gateway`, `caddy`, `ollama`, `openclaw-test`, and `openclaw-prod` are all running and healthy
- `moltbox service list` and `moltbox service status <service>` reflect the real running appliance state
- `moltbox test openclaw agent --agent main --message ... --json` returns a successful reply through the official OpenClaw CLI path
- `moltbox prod openclaw agent --agent main --message ... --json` returns a successful reply through the official OpenClaw CLI path
- both runtime chat paths report `provider=ollama` and `model=mistral:7b-instruct-32k` for ordinary local-first chat
- the local Ollama inventory includes `mistral:7b-instruct-32k`
- Together escalation remains available through OpenClaw plugin/provider surfaces, and any Remram-specific escalation behavior is packaged as a plugin-backed capability rather than hidden in gateway orchestration
- Caddy terminates and routes `moltbox-test` and `moltbox-prod`

## Captured Live Appliance Facts (2026-04-04)

These are current-state facts observed from the live appliance and should be treated as the migration baseline rather than the target architecture:

- the active SSH alias `moltbox` reaches host `moltbox-prime`, but it still logs in as personal user `jpekovitch`
- the live box is currently running `gateway`, `caddy`, `ollama`, `openclaw-dev`, `openclaw-test`, `openclaw-prod`, and `opensearch`
- every current container is healthy, but no container has a configured Docker memory or CPU limit
- the live Ollama inventory includes `mistral:7b-instruct-32k` and does not include the exact tag `gemma3:4b-it`
- both live `test` and `prod` runtimes currently use `ollama/mistral:7b-instruct-32k` as `agents.defaults.model.primary`
- both live `test` and `prod` runtimes currently set `contextTokens` to `32768`
- both live runtimes currently keep Together fallback chains in `model-runtime.yml`
- official chat through `moltbox test openclaw agent --agent main ... --json` and `moltbox prod openclaw agent --agent main ... --json` works today and reports `provider=ollama` and `model=mistral:7b-instruct-32k`
- `openclaw agent` is not sessionless; it requires `--agent`, `--to`, or `--session-id`
- both live runtimes warn that `plugins.allow` is empty and that local `behavior-guard` code is loading without explicit trust provenance

## Final CLI Contract

```text
moltbox
  bootstrap
    gateway

  gateway
    status
    logs
    update
    mcp-stdio

  service
    list
    status <service>
    deploy <service>
    restart <service>
    logs <service>

  test
    openclaw <command>

  prod
    openclaw <command>

  ollama
    <native ollama command>

  secret
    set <scope> <name> [value]
    list <scope>
    delete <scope> <name>
```

Valid service identifiers:

- `gateway`
- `caddy`
- `ollama`
- `test`
- `prod`

Valid secret scopes:

- `service`
- `test`
- `prod`

Removed from the public contract:

- `dev`
- `checkpoint`
- `skill ...`
- `plugin ...`
- `opensearch`
- `gateway docker ping`
- `gateway docker run <image>`
- `gateway service ...`
- `<env> secrets ...`
- `service secrets ...`

## Host Layout And Ownership

### System-Owned Paths

- CLI binary: `/usr/local/bin/moltbox`
- CLI config: `/etc/moltbox/config.yaml`
- wrapper scripts: `/usr/local/bin/moltbox-cli-wrapper`, `/usr/local/bin/moltbox-bootstrap-wrapper`
- repo roots: `/opt/moltbox/repos/moltbox-gateway`, `/opt/moltbox/repos/moltbox-services`, `/opt/moltbox/repos/moltbox-runtime`, `/opt/moltbox/repos/remram-skills`
- mutable state root: `/srv/moltbox-state`
- logs root: `/srv/moltbox-logs`
- secrets root: `/var/lib/moltbox/secrets`
- appliance ledger: `/var/lib/moltbox/history.jsonl`

### Ownership Rules

- no runtime state, repo checkout, CLI binary, or config file may live under `/home/*`
- no mount path may be owned by a personal user account
- service and state directories should be owned by `root:root` or `root:moltbox`, never by a human login account
- secret files must remain `0600` and the secret root must remain `0700`

### Human And Emergency Access

- keep the human admin account `jpkovic` as a system owner and sudo-capable operator
- do not create a second UID 0 user
- create one separate break-glass admin account with a distinct SSH key and sudo access
- disable password-based remote admin login

## AI SSH Access Model

Replace personal-name automation identities with system identities:

- `moltbox-ai-test`
  - forced-command wrapper
  - may run only `moltbox ...`
  - full CLI access on the designated test appliance
  - may mutate `service`, `test`, `ollama`, and `secret` scopes on the test appliance
  - may not access arbitrary shell
- `moltbox-ai-prod`
  - forced-command wrapper
  - may run only `moltbox ...`
  - read-only service-plane visibility on the production appliance
  - allowed runtime access limited to `prod openclaw status|inspect|logs|health` and equivalent safe diagnostics
  - no mutation of `prod`, `service`, or system configuration

Wrapper policy should be environment-based and identity-based, not tied to legacy names such as `jason-codex` or `codex-bootstrap`.

## Service And Runtime Responsibilities

### Gateway

- host-local control plane
- service deployment and restart
- rendered asset generation
- secret storage
- deployment provenance
- backup scheduling visibility
- system-health reporting
- no direct ownership of model routing logic beyond rendering approved baseline config

### OpenClaw `test`

- appliance proving lane
- official plugin and skill install testing
- AI experimentation
- backup verification
- first place where Together escalation plugin behavior is validated

### OpenClaw `prod`

- managed pet runtime
- official OpenClaw install and config flows only
- restore point required before risky changes
- recovery is backup-first, not rebuild-first
- keeps the same proven local default model as `test` unless a deliberate promotion changes it

### Caddy

- ingress and TLS termination for `moltbox-test` and `moltbox-prod`
- generated local cert assets only for those two hosts

### Ollama

- local model serving
- persistent model store under system-owned shared state
- required baseline local model: `mistral:7b-instruct-32k`

## Backup, Restore, And Patching

These are part of the appliance implementation, but not part of the public CLI contract.

### Backups

- nightly machine backup or host snapshot
- pre-change restore point before risky `prod` changes
- scheduled `openclaw backup create --verify` for `test` and `prod`
- off-host encrypted retention for machine backups and OpenClaw backups

### Restore

- primary restore path: restore machine or restore point first
- secondary restore path: apply newer OpenClaw backup if needed
- clean rebuild from repos is fallback only

### Patching

- configure unattended OS security patching or an equivalent system timer
- configure a scheduled gateway-maintained patch-status check
- document reboot policy after kernel or Docker updates

## Code Changes By Repo

### `moltbox-gateway`

- replace the parser in [`../../pkg/cli/cli.go`](../../pkg/cli/cli.go) with the new CLI tree
- add a `bootstrap gateway` route and local bootstrap execution path
- remove `dev`, `checkpoint`, `skill`, `plugin`, `opensearch`, and gateway Docker helper parsing
- move service lifecycle from `gateway service ...` to top-level `service ...`
- replace scoped secret parsing with top-level `secret ...`
- simplify HTTP client routing in [`../../internal/client/http.go`](../../internal/client/http.go)
- remove obsolete runtime skill and plugin HTTP handlers from [`../../internal/gateway/http.go`](../../internal/gateway/http.go)
- keep `runtime/openclaw` passthrough only for `test` and `prod`
- update SSH wrapper policy in [`../../internal/sshwrap/sshwrap.go`](../../internal/sshwrap/sshwrap.go) for the new AI identities
- remove user-home config lookup from [`../../internal/config/config.go`](../../internal/config/config.go)
- make `/etc/moltbox/config.yaml` the only supported config path in production
- replace personal-name SSH provisioning script inputs and account creation logic in [`../../scripts/provision-automation-ssh.sh`](../../scripts/provision-automation-ssh.sh)
- remove ownership inference from the CLI install path in the gateway update helper

### `moltbox-services`

- delete `services/openclaw-dev`
- delete `services/opensearch`
- remove OpenSearch environment injection from `openclaw-test` and `openclaw-prod`
- keep `gateway`, `caddy`, `ollama`, `openclaw-test`, `openclaw-prod`
- update Caddy templates and cert generation assumptions to only `moltbox-test` and `moltbox-prod`

### `moltbox-runtime`

- delete `openclaw-dev`
- delete `opensearch`
- remove `opensearch.yml` from `openclaw-test` and `openclaw-prod`
- remove `retrieval_search` and OpenSearch-backed memory/tool references
- keep `ollama/mistral:7b-instruct-32k` as the approved local-first default for `test` and `prod`
- keep only approved runtime guardrails for `test` and `prod`
- move Remram-specific Together escalation behavior into a declared plugin-backed capability instead of leaving it as baseline-only implicit policy

### `remram-skills` and plugin payloads

- stop treating a documentation skill as the whole Together escalation implementation
- keep operator-facing skill content only as usage guidance
- introduce a real plugin-backed capability for Remram-specific escalation behavior if stock OpenClaw provider wiring is not sufficient
- ensure the gateway only passes through official OpenClaw plugin/config operations rather than carrying bespoke escalation logic

### `remram` docs

- update CLI architecture and operator docs to the new contract
- remove references to `dev`, `opensearch`, `jason-codex`, `codex-bootstrap`, and `~/.local/bin`

## Migration Order

### Phase 1: Contract And Host Hygiene

- freeze the current CLI contract
- publish the new CLI contract
- remove user-home install and config assumptions
- define final system-owned paths
- define final AI SSH identity policy

### Phase 2: Parser, Routing, And Docs

- implement the new parser
- update help text
- update HTTP routing
- remove dead and compatibility-only runtime management verbs
- regenerate unit tests and CLI docs

### Phase 3: Service Baseline Simplification

- delete `openclaw-dev`
- delete `opensearch`
- simplify runtime/service templates
- update Caddy and runtime configs
- preserve the current `mistral:7b-instruct-32k` local model baseline while removing unrelated complexity

### Phase 4: Appliance Bootstrap And Access

- implement `bootstrap gateway`
- update provisioning scripts for system identities
- install system-owned wrappers and config
- validate AI SSH access policy

### Phase 5: Backup And Patching Hygiene

- install backup timers
- install restore-point workflow
- install patch-status and security-update timers
- validate restore and patch reporting

### Phase 6: Functional Runtime Acceptance

- prove local-first chat on `test` through `moltbox test openclaw agent --agent main ... --json`
- prove local-first chat on `prod` through `moltbox prod openclaw agent --agent main ... --json`
- prove both runtime paths report `provider=ollama` and `model=mistral:7b-instruct-32k`
- prove Together escalation remains reachable through plugin/provider mechanisms with observable provider/model evidence
- prove the CLI path succeeds cleanly without hidden fallback behavior that masks routing or auth problems

## Acceptance Criteria

- `moltbox --help` shows only the final lightweight surface
- no public CLI command mentions `dev`, `checkpoint`, `skill`, `plugin`, or `opensearch`
- no production host config path falls back to a user-home directory
- no service state, repo state, or secrets are owned by a personal login account
- `moltbox bootstrap gateway` can bring up the gateway on a clean host
- `moltbox service list` returns only `gateway`, `caddy`, `ollama`, `test`, and `prod`
- `moltbox test openclaw ...` and `moltbox prod openclaw ...` work as official passthroughs
- `moltbox ollama list` works and includes `mistral:7b-instruct-32k`
- `moltbox test openclaw agent --agent main ... --json` returns success and reports `provider=ollama` and `model=mistral:7b-instruct-32k`
- `moltbox prod openclaw agent --agent main ... --json` returns success and reports `provider=ollama` and `model=mistral:7b-instruct-32k`
- Together escalation is observable through official plugin/provider behavior rather than gateway-specific hard-coded logic
- `moltbox-ai-test` and `moltbox-ai-prod` obey the new forced-command policy
- nightly backup, restore-point creation, and OS patching are configured and testable

## Assumptions

- the target appliance is a Linux host with Docker and systemd
- `prod` recovery is backup-first rather than rebuild-first
- Postgres will be introduced later as part of Cortex, not in this appliance refactor
- `mcp-stdio` remains available for internal workflows even though it is not a primary operator surface
