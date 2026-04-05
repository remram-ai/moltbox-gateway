# Current State

Observed baseline: `2026-04-05`

## Live Host Facts

- host OS: Ubuntu 24.04.4 LTS
- host name: `moltbox-prime`
- base OS root remains on `ext4`
- Moltbox-critical paths are on ZFS:
  - `/srv/moltbox-state`
  - `/srv/moltbox-logs`
  - `/var/lib/moltbox`
  - `/opt/moltbox/repos`
- backup disk is mounted at `/mnt/moltbox-backup`

The host meets the appliance storage boundary even though the base OS itself is still on `ext4`.

## Live Service Plane

The live managed service set is:

- `gateway`
- `caddy`
- `ollama`
- `searxng`
- `openclaw-test`
- `openclaw-prod`

`playwright` is no longer part of the steady-state service plane.

## Live CLI Contract

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

Retired public namespaces fail explicitly:

- `dev`
- `opensearch`
- `runtime`
- `skill`
- `plugin`
- `gateway service`
- `gateway docker`

## Live Runtime Facts

- `test` and `prod` both use local Ollama with `mistral:7b-instruct-32k`
- configured context window is `32768`
- Together remains configured as the fallback provider
- heartbeat is disabled
- baseline web capability is:
  - `web_search` backed by `searxng`
  - built-in `web_fetch`
- the previous browser detour is removed from the baseline
- `test` and `prod` no longer replay runtime history during normal service deploy

## Live Recovery Facts

- service deploy and restart take ZFS snapshots
- mutating native `moltbox test|prod openclaw ...` commands take ZFS snapshots
- ZFS rollback has been manually proven on the live host
- native OpenClaw backup create and verify are proven
- native OpenClaw restore is not yet a proven primary recovery path

## Live Operator Model

- human admin access is preserved
- `moltbox-ai-test` is a forced-command test-side CLI identity
- `moltbox-ai-prod` is a forced-command prod diagnostics identity
- break-glass admin remains available for emergencies

## Remaining Gaps

- native browser support is present upstream but is not yet a proven appliance baseline
- the local model is still not perfectly reliable at choosing tools through ordinary chat
- native OpenClaw restore is still unproven as a primary runtime recovery method
