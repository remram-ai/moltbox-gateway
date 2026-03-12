# Moltbox Manual Operations

This note describes the current operator workflow for the Linux appliance host.

Architecture authority:

- `remram/architecture-v2/gateway.md`
- `remram/architecture-v2/services.md`
- `remram/architecture-v2/runtime.md`
- `remram/architecture-v2/skills.md`

Detailed command coverage:

- [moltbox-cli-reference.md](/D:/Development/RemRam/remram-gateway/docs/operator/moltbox-cli-reference.md)

## Host Baseline

- host target: `moltbox-prime`
- OS: Ubuntu 24.04.x LTS
- platform model: Linux appliance host with Docker Engine
- durable state root: `/srv/moltbox-state`
- durable logs root: `/srv/moltbox-logs`

The appliance host should expose `moltbox` on PATH through the bootstrap-installed host wrapper.

## Preferred Operator Interface

Run the CLI directly on the appliance host:

```bash
moltbox gateway status
moltbox service list
moltbox service status gateway
moltbox service status caddy
moltbox service status opensearch
moltbox service status openclaw-dev
moltbox service status openclaw-test
moltbox service status openclaw-prod
```

Runtime operations:

```bash
moltbox openclaw-dev config sync
moltbox openclaw-dev reload
moltbox openclaw-test doctor
moltbox openclaw monitor
```

Skill deployment:

```bash
moltbox skill deploy semantic-router
moltbox skill deploy semantic-router --runtime openclaw-test
```

Repo mirror maintenance:

```bash
moltbox gateway repo refresh
moltbox gateway repo seed runtime --bundle /path/to/moltbox-runtime.bundle
```

## Deployment Workflow

Shared services:

```bash
moltbox service deploy caddy
moltbox service deploy opensearch
```

Runtime environments:

```bash
moltbox service deploy openclaw-dev
moltbox service deploy openclaw-test
moltbox service deploy openclaw-prod
```

Gateway self-update:

```bash
moltbox gateway update
```

## Config Discovery

Inside the running gateway container, the CLI auto-discovers:

```text
/etc/moltbox/config.yaml
```

Normal operator execution should not require `--config-path`.

## Legacy Surface

Older forms such as:

```text
moltbox tools ...
moltbox host ssl ...
moltbox runtime dev ...
```

are no longer canonical. Use:

```text
moltbox gateway ...
moltbox service ...
moltbox openclaw-dev ...
```

The CLI returns explicit replacement guidance when a legacy command is used.

## Docker Fallback

If CLI output is insufficient during diagnosis, use Docker only as a secondary investigation path:

```bash
docker ps
docker logs -f gateway
docker logs -f caddy
docker logs -f opensearch
docker logs -f openclaw-dev
docker logs -f openclaw-test
docker logs -f openclaw-prod
docker inspect gateway
docker network inspect moltbox_moltbox_internal
```
