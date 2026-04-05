# Target State

This document defines the intended future-state appliance.

## Appliance Definition

The target appliance is:

- one host: `moltbox-prime`
- one gateway access name: `moltbox-gateway`
- two runtime access names: `moltbox-test` and `moltbox-prod`
- five managed services
- ZFS-backed state
- backup-first recovery

Final services:

- `gateway`
- `caddy`
- `ollama`
- `openclaw-test`
- `openclaw-prod`

Removed:

- `openclaw-dev`
- `opensearch`

Deferred:

- Postgres
- Cortex shared services
- broader data services

## Runtime Model

### Test

`test` is the appliance proving lane.

Use it for:

- validating config changes
- validating plugin and skill installation flows
- validating Together escalation behavior
- validating service-plane changes before promotion

### Prod

`prod` is a managed pet.

Rules:

- mutate it only through official OpenClaw config, plugin, skill, and backup surfaces
- create a restore point before risky changes
- do not rely on replay as the normal recovery path
- restore the runtime or the machine as a box when needed

### Local development

Local development happens outside the appliance. There is no appliance `dev` runtime in the target design.

## Model And Provider Policy

Local-first model baseline:

- provider: `ollama`
- model: `mistral:7b-instruct-32k`
- context window: `32768`

Cloud escalation:

- Together remains available
- Together must be configured through official OpenClaw provider or plugin surfaces
- the gateway must not hard-code the routing logic

## Cortex MVP Config Seam

The current Cortex Phase 0 and Phase 1 prep assets reinforce this design:

- Phase 0 keeps `memory.backend = qmd`
- Phase 0 keeps `plugins.slots.contextEngine = legacy`
- Phase 0 keeps the Cortex bridge plugin disabled
- Phase 1 still keeps the context engine on `legacy`
- Phase 1 enables only the `cortex-phase1-bridge` plugin
- Phase 1 turns that bridge on in `dry-run` mode first

That is the correct integration seam: OpenClaw stays native, and Cortex overlays or plugins attach at bounded points.

## CLI Model

The CLI is intentionally small:

```text
moltbox
  bootstrap gateway
  gateway status|logs|update|mcp-stdio
  service list|status|deploy|restart|logs <service>
  test openclaw <native args>
  prod openclaw <native args>
  ollama <native args>
  secret set|list|delete <scope>
```

The CLI is the service-plane tool and the access path to official OpenClaw CLI surfaces. It is not a second OpenClaw lifecycle language.

## Host Model

The host must provide:

- ZFS-backed datasets for the Moltbox-critical paths
- system-owned runtime paths
- preserved human admin access
- restricted AI SSH identities
- snapshot and rollback capability
- local backup target on the 14 TB disk

## Success Conditions

The target state is only reached when all of these are true:

- the host is rebuilt on ZFS
- the five target services are healthy
- `test` and `prod` chat through the CLI
- ordinary chat uses local `ollama/mistral:7b-instruct-32k`
- Together escalation is observable without gateway hard-coding
- no required path is owned by a personal user
- backup and restore paths are real, not implied

## Related Documents

- [CLI And Gateway](cli-and-gateway.md)
- [Runtime And Services](runtime-and-services.md)
- [Host And Operations](host-and-operations.md)
- [Backup And Recovery](backup-and-recovery.md)
