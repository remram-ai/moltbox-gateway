# MoltBox CLI Grammar

The CLI command is `moltbox`.

The folder `moltbox-cli` is not part of the CLI namespace.

Canonical grammar:

```text
moltbox <domain> <target> <verb>
```

Domains:

- `runtime`
- `host`
- `tools`

Domain forms:

- `moltbox runtime <environment> <verb>`
- `moltbox host <service> <verb>`
- `moltbox tools <verb>`

`tools` is a singleton domain, so it uses the shortened form without a separate target token.

Verb safety:

- Inspection verbs: `status`, `inspect`, `logs`, `health`, `version`
- Mutation verbs: `deploy`, `rollback`, `start`, `stop`, `restart`, `update`

Examples:

- `moltbox runtime dev deploy`
- `moltbox runtime prod logs`
- `moltbox host ollama deploy`
- `moltbox host opensearch status`
- `moltbox tools version`
- `moltbox tools update`
