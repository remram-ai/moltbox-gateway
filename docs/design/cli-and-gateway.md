# CLI And Gateway

This document defines the public Moltbox CLI contract and the current gateway role.

## Public CLI Contract

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

This is the supported operator surface.

## Gateway Role

The gateway owns:

- service-plane orchestration
- gateway update
- secret storage and injection
- snapshot hooks
- deployment history
- CLI/API presentation
- SSH role enforcement

The gateway does not own:

- replay-based runtime reconstruction for `test` / `prod`
- gateway-managed plugin or skill orchestration as the normal runtime model
- browser or search logic

## Runtime Lifecycle Rule

Normal runtime mutation happens through native OpenClaw surfaces:

- `moltbox test openclaw ...`
- `moltbox prod openclaw ...`

The gateway wraps those calls for policy and snapshot guardrails, but it does not replace them with its old replay model.

The gateway also owns a small verification surface for restricted operators:

- `moltbox test verify runtime`
- `moltbox test verify browser`
- `moltbox test verify web`
- `moltbox prod verify runtime`

Those checks exist so routine validation can stay inside the restricted SSH roles without shell chaining or break-glass access.

## Snapshot Guardrails

The gateway snapshots before:

- `service deploy`
- `service restart`
- `service remove`
- mutating native `openclaw` runtime commands

Help and dry-run commands are not treated as mutations.

## Retired Public Surfaces

These are retired from the public contract:

- `dev`
- `opensearch`
- `runtime`
- `skill`
- `plugin`
- `gateway service`
- `gateway docker`

## Gateway Bootstrap Rule

- use `moltbox bootstrap gateway` for first install or recovery
- use `moltbox gateway update` for normal gateway upgrades
