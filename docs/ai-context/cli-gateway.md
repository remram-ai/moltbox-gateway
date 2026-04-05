# CLI And Gateway Context

Use this file when the task is mostly about the CLI contract, gateway routes, handlers, or operator workflow.

Desired gateway role:

- thin appliance orchestrator
- service-plane manager
- secret owner
- gateway status and update owner

Desired non-role:

- not the authority for OpenClaw plugin logic
- not the authority for runtime model routing
- not the normal runtime replay engine

Desired public CLI:

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

Public service names:

- `gateway`
- `caddy`
- `ollama`
- `test`
- `prod`

Removed public surfaces:

- `dev`
- `opensearch`
- `checkpoint`
- `skill`
- `plugin`
- `gateway docker`
- `gateway service`
- `service secrets`

Canonical source:

- `../design/cli-and-gateway.md`
