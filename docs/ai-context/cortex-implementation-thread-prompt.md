# Cortex Implementation Thread Prompt

Use this prompt to start a Cortex implementation thread against the current Moltbox appliance baseline.

---

You are implementing Cortex against the live Moltbox appliance baseline.

Before doing any work, load these Gateway docs in this order:

1. `moltbox-gateway/docs/guides/operator-guide.md`
2. `moltbox-gateway/docs/guides/service-catalog.md`
3. `moltbox-gateway/docs/design/system-overview.md`
4. `moltbox-gateway/docs/design/cli-and-gateway.md`
5. `moltbox-gateway/docs/design/runtime-and-services.md`
6. `moltbox-gateway/docs/design/backup-and-recovery.md`
7. `moltbox-gateway/docs/design/web-tooling.md`
8. `moltbox-gateway/docs/ai-context/overview.md`
9. `moltbox-gateway/docs/ai-context/current-state.md`
10. `moltbox-gateway/docs/ai-context/future-state.md`
11. `moltbox-gateway/docs/ai-context/operator-workflows.md`

Treat `moltbox-gateway` as the current source of truth for Gateway/OpenClaw appliance behavior. Older `remram` docs may be useful background, but they do not override the Gateway repo for live appliance work.

## Current appliance contract

Managed services:

- `gateway`
- `caddy`
- `ollama`
- `searxng`
- `test`
- `prod`

Public CLI:

```text
moltbox
  bootstrap gateway
  gateway status|logs|update|mcp-stdio
  service list|status|deploy|restart|remove|logs <service>
  test openclaw <native args>
  test verify runtime|browser|web
  prod openclaw <native args>
  prod verify runtime
  ollama <native args>
  secret set|list|delete <scope>
```

## Operating rules

- `test` is the proving lane.
- `prod` is a protected managed pet.
- Normal runtime mutation uses native OpenClaw CLI surfaces, not replay-era Gateway internals.
- Service-plane mutation uses the `moltbox service ...` surface.
- Recovery is snapshot-first:
  - ZFS snapshot before risky change
  - ZFS rollback first when backing out appliance/runtime state
  - OpenClaw native backup as the runtime backup layer
- Web baseline is:
  - `web_search` via `searxng`
  - built-in `web_fetch`
  - native OpenClaw `browser`
- Do not reintroduce Playwright wrapper architecture as the intended baseline.
- Do not use raw Docker as the normal path.
- Do not use break-glass SSH for routine operations if the CLI surface can do the job.

## SSH/operator model

- human admin path exists
- AI test operator:
  - CLI-driven mutation on `test`
  - limited service-plane mutation for approved services
  - may run `moltbox test verify ...`
- AI prod operator:
  - CLI diagnostics on `prod`
  - may run `moltbox prod verify runtime`
  - no normal prod mutation rights
- break-glass admin:
  - emergency/debug only

## Expectations for this thread

- build on the current tracked Moltbox baseline, not on host-only drift
- prefer implementation through official repo changes, commits, pushes, host pull, and CLI deploy
- keep Gateway thin
- keep OpenClaw in the managed-pet model
- use `test` to prove changes before proposing or performing promotion to `prod`
- preserve the current operator contract unless a change is clearly necessary
- if you add a meaningful new operator capability, update:
  - `docs/guides/operator-guide.md`
  - `docs/design/cli-and-gateway.md`
  - `docs/ai-context/operator-workflows.md`

## Validation posture

Use the official verification surfaces whenever possible:

- `moltbox test verify runtime`
- `moltbox test verify browser`
- `moltbox test verify web`
- `moltbox prod verify runtime`

If a routine validation requires break-glass SSH or raw shell orchestration, treat that as a missing CLI/operator surface and close the gap instead of normalizing the workaround.

## Delivery posture

Finish changes on the tracked release path:

1. implement in the appropriate Moltbox repo
2. commit cleanly
3. push
4. pull exact revisions on the host
5. deploy through the official CLI/service-plane path
6. validate from that tracked deployed state

When reporting progress, prefer:

- what changed
- what was validated
- what remains blocked
- whether the change is safe to keep in `test`
- whether it is ready for later promotion to `prod`

---
