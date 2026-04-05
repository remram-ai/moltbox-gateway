# Future State

Use this file when the task depends on the desired end state rather than on the old mainline model.

Target appliance:

- `gateway`
- `caddy`
- `ollama`
- `openclaw-test`
- `openclaw-prod`

Target rules:

- no appliance `dev`
- no OpenSearch
- no Postgres in this rebuild
- local development stays off-appliance
- `test` is the proving lane
- `prod` is a managed pet
- gateway stays thin
- OpenClaw owns native config, plugin, skill, and backup behavior

CLI target:

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

Storage and recovery target:

- ZFS-backed critical paths
- snapshot-first restore
- appliance backup on local backup disk
- OpenClaw native backups retained

Canonical sources:

- `../design/target-state.md`
- `../design/backup-and-recovery.md`
