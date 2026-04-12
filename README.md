# Moltbox Gateway

`moltbox-gateway` is the control-plane and CLI repo for the Moltbox appliance.

This repo is the local source of truth for the current Gateway/OpenClaw operating model. Older `remram` docs are useful history, but they do not override this repo for Gateway implementation and operations.

Repository boundary:

- `moltbox-gateway` owns CLI, procedures, verification, and recovery workflows
- `moltbox-services` owns baseline service definitions and baseline config examples
- `moltbox-runtime` holds the final deployable runtime artifacts consumed by the current release path

## Documentation Tree

Follow the repo in this order:

- `docs/README.md`
- `docs/guides/README.md`
- `docs/design/README.md`
- `docs/ai-context/README.md`

Use the dated material under `docs/decisions/`, `docs/reviews/`, `docs/plans/`, and `docs/runbooks/` only after the current guides and design docs.

## Humans Start Here

Read these in order:

1. `docs/guides/operator-guide.md`
2. `docs/guides/service-catalog.md`
3. `docs/design/README.md`
4. `docs/design/system-overview.md`
5. `docs/design/cli-and-gateway.md`
6. `docs/design/runtime-and-services.md`
7. `docs/design/backup-and-recovery.md`
8. `docs/design/host-and-operations.md`
9. `docs/design/web-tooling.md`

Use these after the current docs:

- `docs/decisions/2026-04-04-openclaw-operating-model.md`
- `docs/runbooks/2026-04-04-zfs-rebuild-ssh-takeover-runbook.md`
- `docs/reviews/README.md`
- `docs/plans/README.md`

## AIs Start Here

Import these first:

1. `docs/ai-context/README.md`
2. `docs/ai-context/overview.md`
3. `docs/ai-context/current-state.md`
4. `docs/ai-context/future-state.md`
5. `docs/ai-context/operator-workflows.md`
6. `docs/ai-context/cortex-implementation-thread-prompt.md` if you are starting a Cortex implementation thread

Then add the task-specific context:

- `docs/ai-context/cli-gateway.md`
- `docs/ai-context/runtime-services.md`
- `docs/ai-context/host-ops.md`

## Current Appliance Contract

Public CLI:

```text
moltbox
  bootstrap gateway
  gateway status|logs|update|repo-sync services|runtime|all|mcp-stdio
  service list|status|deploy|restart|remove|logs <service>
  test openclaw <native args>
  test verify runtime|browser|web|sandbox
  prod openclaw <native args>
  prod verify runtime
  ollama <native args>
  secret set|list|delete <scope>
```

Current managed services:

- `gateway`
- `caddy`
- `ollama`
- `searxng`
- `dev-sandbox`
- `test`
- `prod`

Service authority:

- baseline service inputs and service docs live in `moltbox-services`
- the current deploy path still consumes final runtime artifacts from `moltbox-runtime`

Runtime service mapping:

- public `test` -> `openclaw-test`
- public `prod` -> `openclaw-prod`

Core operating model:

- `test` and `prod` are managed-pet OpenClaw runtimes
- `dev-sandbox` is an image-backed service that supplies the sibling Docker sandbox image used by the non-default `coder` agent path
- normal runtime mutation happens through native `openclaw` CLI surfaces
- routine runtime verification happens through `moltbox test verify ...` and `moltbox prod verify runtime`
- routine repo promotion for service/runtime source happens through `moltbox gateway repo-sync services|runtime|all`
- replay and checkpoint are not part of the normal `test` / `prod` lifecycle
- service deploy, service restart, and mutating native runtime commands are snapshot-guarded
- ZFS snapshots are the first restore-point mechanism
- OpenClaw native backup create and verify are in use; native restore is not yet a proven primary recovery path
- baseline web capability is `web_search` + built-in `web_fetch`
- native `memory-core` is disabled in the default local lane
- the previous Playwright detour is removed from the intended baseline and service inventory
- ordinary local chat is reliable for short answers, but web-tool choice through chat still needs human judgment on noisy pages

## Historical Material

The dated docs under `docs/reviews/` and `docs/plans/` are retained as decision history and execution records.

Treat them as historical evidence, not as the current operator contract.
