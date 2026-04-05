# CLI And Gateway Context

Use this file when the task is mostly about CLI behavior, gateway handlers, or operator workflow.

Public CLI:

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

Current public service names:

- `gateway`
- `caddy`
- `ollama`
- `searxng`
- `test`
- `prod`

Gateway role:

- service-plane manager
- snapshot guardrail owner
- gateway update owner
- secret owner
- SSH role policy owner

Not the gateway role:

- normal runtime replay engine
- owner of browser/search logic
- owner of native OpenClaw extension behavior
