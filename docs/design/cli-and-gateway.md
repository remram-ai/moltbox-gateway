# CLI And Gateway

This document defines the public CLI contract and the intended role of the gateway.

## Gateway Role

The gateway is the appliance control plane.

It should own:

- CLI routing
- gateway status and logs
- service deployment and restart orchestration
- secret storage and scoped injection
- gateway self-update
- deployment validation surfaces
- SSH policy integration for AI identities

It should not own:

- OpenClaw plugin semantics
- OpenClaw skill semantics
- runtime-local model routing logic
- replay-heavy reconstruction as the normal runtime model

## Public CLI Contract

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

## Service Identifiers

The public service inventory is:

- `gateway`
- `caddy`
- `ollama`
- `test`
- `prod`

Runtime service identifiers stay public because the service plane needs to deploy and inspect them, but the runtime container names remain implementation details.

## Command Model

### Bootstrap

`bootstrap gateway` is the only host-local escape hatch.

Its job is narrow:

- start or recover the gateway when the gateway is absent
- hand normal control back to the gateway immediately

### Gateway

`gateway` covers:

- `status`
- `logs`
- `update`
- `mcp-stdio`

Do not grow this namespace casually.

### Service

`service` is the appliance lifecycle namespace:

- `list`
- `status`
- `deploy`
- `restart`
- `logs`

This replaces the old `gateway service ...` public shape.

### OpenClaw

`test openclaw ...` and `prod openclaw ...` are native CLI passthrough surfaces.

The gateway should get out of the way here as much as possible.

### Ollama

`ollama ...` remains a thin passthrough to the native Ollama CLI.

### Secrets

`secret ...` is the only public secret surface.

Secret writes must still flow through the gateway-owned encrypted store.

## Retired Public Surface

These should not appear in help or normal operator docs:

- `dev`
- `opensearch`
- `checkpoint`
- `skill ...`
- `plugin ...`
- `gateway docker ...`
- `gateway service ...`
- `service secrets ...`

If legacy compatibility remains internally for migration, keep it invisible.

## Gateway Transport Model

Normal execution path:

```text
SSH
  -> host-installed moltbox client
    -> http://127.0.0.1:7460
      -> gateway
```

That means:

- workstation operators do not need direct Docker access
- AI SSH identities do not need arbitrary shell
- service deployment remains observable and auditable through one control plane

## Service Deployment Semantics

`moltbox service deploy <service>` should:

1. resolve the desired service definition
2. render or load the required config
3. deploy or reconcile the service
4. wait for health
5. report a clear result

`restart` should use the same health-aware control path rather than a blind container restart.

## Update Semantics

`moltbox gateway update` remains the canonical gateway mutation path.

The gateway should not be self-mutated by pretending it is just another service deploy target.

## Design Constraints

- keep the CLI explainable in one screen
- prefer native OpenClaw and Ollama semantics to custom wrappers
- do not let the gateway become the authority for runtime-local behavior

## Related Documents

- [Target State](target-state.md)
- [Runtime And Services](runtime-and-services.md)
- [Host And Operations](host-and-operations.md)
