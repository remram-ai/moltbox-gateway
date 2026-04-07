# Service Catalog

This is the operator-facing service catalog for the Moltbox appliance.

It is not the long-term authority for service definitions or baseline service config.

Use:

- `moltbox-services` for service definitions, baseline configs, and service-local docs
- `moltbox-runtime` for the final deployable runtime artifacts used by the current release path

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
- `browser`
  - built-in OpenClaw browser surface
  - runs inside the OpenClaw runtime container with headless Chromium
  - kept out of the default chat tool allowlist to reduce prompt/tool overhead

Not part of the current baseline:

- the old Playwright service
- the old plugin-backed `web_browser` detour

Current caveat:

- browser is part of the runtime baseline but not the default chat lane
- chat-level tool choice for fetched pages still needs human judgment on the local Gemma baseline

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
- rendering and secret injection during deployment
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
- `https://github.com/remram-ai/moltbox-services`
- `https://github.com/remram-ai/moltbox-runtime`
