# Moltbox CLI

The Moltbox CLI follows the architecture-v2 grammar:

```text
moltbox <component> <command>
```

Examples:

```text
moltbox gateway status
moltbox service deploy caddy
moltbox openclaw-dev reload
moltbox skill deploy semantic-router --runtime openclaw-test
```

Current component groups:

- `gateway`
- `service`
- `skill`
- runtime components such as `openclaw-dev`, `openclaw-test`, and `openclaw-prod`
- service components such as `caddy` and `opensearch`

The authoritative architecture lives in:

- `remram/architecture-v2/gateway.md`
- `remram/architecture-v2/services.md`
- `remram/architecture-v2/runtime.md`
- `remram/architecture-v2/skills.md`

Operator reference:

- `../operator/moltbox-cli-reference.md`
