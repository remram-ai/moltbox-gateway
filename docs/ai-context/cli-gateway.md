# CLI And Gateway Context

Use this file when the task is mostly about CLI behavior, gateway handlers, or operator workflow.

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

Current public service names:

- `gateway`
- `caddy`
- `ollama`
- `searxng`
- `dev-sandbox`
- `test`
- `prod`

Gateway role:

- service-plane manager
- snapshot guardrail owner
- gateway update owner
- repo promotion owner for `moltbox-services` and `moltbox-runtime`
- secret owner
- SSH role policy owner
- owner of routine operator verification surfaces

Not the gateway role:

- normal runtime replay engine
- owner of browser/search logic
- owner of native OpenClaw extension behavior
