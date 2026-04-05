# Operator Guide

This is the current human operator guide for the Moltbox appliance.

## Managed Services

The managed appliance services are:

- `gateway`
- `caddy`
- `ollama`
- `searxng`
- `test`
- `prod`

Public service names `test` and `prod` map to `openclaw-test` and `openclaw-prod`.

## Public CLI Contract

```text
moltbox
  bootstrap gateway
  gateway status|logs|update|mcp-stdio
  service list|status|deploy|restart|remove|logs <service>
  test openclaw <native args>
  prod openclaw <native args>
  ollama <native args>
  secret set|list|delete <scope>
```

Use this contract for normal operations. Do not use raw Docker as the normal path.

## Common Workflows

Inspect the appliance:

- `moltbox gateway status`
- `moltbox service list`
- `moltbox service status gateway`
- `moltbox service status test`
- `moltbox service status prod`

Mutate the service plane:

- `moltbox service deploy searxng`
- `moltbox service deploy test`
- `moltbox service restart caddy`
- `moltbox service remove <legacy-service>`
- `moltbox service logs test`

Update the control plane:

- `moltbox gateway update`
- `moltbox gateway logs`

Use native OpenClaw CLI on `test`:

- `moltbox test openclaw health --json`
- `moltbox test openclaw models status --json`
- `moltbox test openclaw backup create --verify`
- `moltbox test openclaw agent --session-id smoke-hi --message "hi" --json`
- `moltbox test openclaw browser status --json`
- `moltbox test openclaw browser tabs --json`

Use native OpenClaw CLI on `prod`:

- `moltbox prod openclaw health --json`
- `moltbox prod openclaw models status --json`
- `moltbox prod openclaw backup create --verify`

Use Ollama:

- `moltbox ollama list`
- `moltbox ollama show mistral:7b-instruct-32k`
- `moltbox ollama ps`

Manage secrets:

- `moltbox secret list test`
- `moltbox secret set test TOGETHER_API_KEY`
- `moltbox secret delete service SOME_KEY`

## Runtime Mutation Rule

Normal runtime changes happen through native OpenClaw surfaces:

- `moltbox test openclaw ...`
- `moltbox prod openclaw ...`

The gateway is not the owner of replay-based runtime mutation anymore.

## Recovery Rule

Normal recovery posture:

1. snapshot before risky change
2. native mutation through official CLI
3. ZFS rollback first if the change is bad
4. OpenClaw backup as the runtime backup layer

Checkpoint and replay are not the normal recovery story.

## SSH Role Model

Human admin:

- retained for trusted administration

AI test operator:

- forced-command SSH
- CLI-driven mutation on `test`
- limited service-plane mutation for `test`, `ollama`, and `searxng`

AI prod operator:

- forced-command SSH
- non-mutating diagnostics on `prod`
- no prod mutation rights

Break-glass admin:

- emergency use only

## Web Capability

Baseline web capability is:

- `web_search` backed by `searxng`
- built-in `web_fetch`
- native OpenClaw `browser`

The old Playwright-based browser detour is removed from the baseline.

Important caveat:

- the native browser surface is enabled and healthy
- ordinary local Mistral chat is still not fully reliable at choosing `web_fetch` or `browser` through chat on every turn
- if a tool-heavy web task matters, validate it in `test` before treating it as a gold-path local-chat behavior

## Retired Public Surfaces

These are not part of the current operator contract:

- `dev`
- `opensearch`
- `runtime`
- `skill`
- `plugin`
- `gateway service`
- `gateway docker`
- top-level `caddy`

## Read Next

- `service-catalog.md`
- `../design/cli-and-gateway.md`
- `../design/runtime-and-services.md`
- `../design/backup-and-recovery.md`
