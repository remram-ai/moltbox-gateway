# Service Catalog

This is the current service catalog for the Moltbox appliance.

## Service Inventory

| Public service | Runtime/container | Purpose | Notes |
| --- | --- | --- | --- |
| `gateway` | `gateway` | control plane and API | updated with `moltbox gateway update` |
| `caddy` | `caddy` | ingress and TLS | fronts `moltbox-gateway`, `moltbox-test`, and `moltbox-prod` |
| `ollama` | `ollama` | local model serving | primary local model backend |
| `searxng` | `searxng` | local search backend | backs `web_search` |
| `test` | `openclaw-test` | proving runtime | first lane for baseline and runtime changes |
| `prod` | `openclaw-prod` | protected runtime | managed pet |

## Runtime Web Model

Current baseline web tooling:

- `web_search`
  - built-in OpenClaw tool
  - backed by `searxng`
- `web_fetch`
  - built-in OpenClaw tool
  - unchanged

Not part of the current baseline:

- the old Playwright service
- the old plugin-backed `web_browser` detour

Native browser support is not part of the current gold baseline until it is proven on this appliance through official OpenClaw browser surfaces.

## Deployment And Inspection

All services are managed through the service plane:

- `moltbox service list`
- `moltbox service status <service>`
- `moltbox service deploy <service>`
- `moltbox service restart <service>`
- `moltbox service remove <service>`
- `moltbox service logs <service>`

Examples:

- `moltbox service deploy searxng`
- `moltbox service deploy test`
- `moltbox service status prod`
- `moltbox service logs gateway`

## Mutation And Recovery Boundary

The gateway owns:

- service deploy, restart, remove, logs, and status
- rendered config and secret injection
- snapshot hooks and deployment history

OpenClaw owns:

- runtime-local config interpretation
- native config changes
- native backup operations
- model/provider behavior
- runtime-local plugin behavior

## Related Documents

- `../design/runtime-and-services.md`
- `../design/web-tooling.md`
- `operator-guide.md`
