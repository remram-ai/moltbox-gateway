# Operator Guide

This is the current human operator guide for the Moltbox appliance.

## Managed Services

The managed appliance services are:

- `gateway`
- `caddy`
- `ollama`
- `searxng`
- `dev-sandbox`
- `test`
- `prod`

Public service names `test` and `prod` map to `openclaw-test` and `openclaw-prod`.

## Public CLI Contract

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

Use this contract for normal operations. Do not use raw Docker as the normal path.

## Common Workflows

Inspect the appliance:

- `moltbox gateway status`
- `moltbox gateway repo-sync services runtime`
- `moltbox service list`
- `moltbox service status gateway`
- `moltbox service status dev-sandbox`
- `moltbox service status test`
- `moltbox service status prod`

Mutate the service plane:

- `moltbox service deploy searxng`
- `moltbox service deploy dev-sandbox`
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

Run operator-grade verification on `test`:

- `moltbox test verify runtime`
- `moltbox test verify web`
- `moltbox test verify sandbox`

Use native OpenClaw CLI on `prod`:

- `moltbox prod openclaw health --json`
- `moltbox prod openclaw models status --json`
- `moltbox prod openclaw backup create --verify`

Run operator-grade verification on `prod`:

- `moltbox prod verify runtime`

Use Ollama:

- `moltbox ollama list`
- `moltbox ollama show gemma4:e4b-it-q4_K_M`
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

AI test operator:

- forced-command SSH
- current host account: `moltbox-ai-test`
- current SSH alias: `moltbox-ai-test`
- CLI-driven mutation on `test`
- service-plane mutation for `dev-sandbox` and the test-lane dependencies
- repo promotion through `moltbox gateway repo-sync services|runtime|all`
- routine verification through `moltbox test verify ...`

AI prod operator:

- forced-command SSH
- current host account: `moltbox-ai-prod`
- current SSH alias: `moltbox-ai-prod`
- non-mutating diagnostics on `prod`
- no prod mutation rights
- routine verification through `moltbox prod verify runtime`

Break-glass admin:

- emergency use only
- current host account: `moltbox-breakglass`
- current SSH alias: `moltbox-breakglass`
- current live behavior: full shell plus passwordless `sudo`
- use this lane for one-into-admin recovery or bootstrap operations such as
  the initial `moltbox gateway update` that exposes new restricted CLI surfaces

## SSH Key Discovery

On the current operator workstation, the active key path for all of the accounts above is:

- `C:\Users\Jason\.ssh\id_ed25519`

Current live host state uses that one key for:

- `moltbox-ai-test`
- `moltbox-ai-prod`
- `moltbox-breakglass`

Legacy local keys:

- `C:\Users\Jason\.ssh\jason-codex`
- `C:\Users\Jason\.ssh\codex-bootstrap`

Those legacy key names are not the current installed host keys and should not be treated as the current account names or the current onboarding path.

## Sandbox Rollout

For the sandboxed coding lane, the normal operator sequence is:

1. `ssh moltbox-breakglass 'moltbox gateway update'`
2. `ssh moltbox-ai-test 'moltbox gateway repo-sync services runtime'`
3. `ssh moltbox-ai-test 'moltbox service deploy dev-sandbox'`
4. `ssh moltbox-ai-test 'moltbox service deploy test'`
5. `ssh moltbox-ai-test 'moltbox test verify sandbox'`

Promotion rule:

- keep `test` as the acceptance lane
- prepare matching `prod` templates in source
- do not deploy `prod` sandbox changes without an explicit promotion step

## Web Capability

Baseline web capability is:

- `web_search` backed by `searxng`
- built-in `web_fetch`
- native `memory-core` disabled in the default local lane

The old Playwright-based browser detour stays removed from the baseline.

Important caveat:

- ordinary local Gemma chat is faster and more trustworthy for `web_search` and `web_fetch` than the previous Mistral baseline, but chat-level tool choice still needs human judgment on noisy pages
- browser is not part of the default local baseline and should only be staged intentionally for a separate diagnostic path
- `moltbox test verify web` proves backend/config availability, not that the local chat model will choose `web_search` or `web_fetch` reliably

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
