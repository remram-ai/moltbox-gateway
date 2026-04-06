# Overview

Moltbox Gateway is the control-plane and CLI repo for the Moltbox appliance.

Current appliance shape:

- `gateway`
- `caddy`
- `ollama`
- `searxng`
- `openclaw-test`
- `openclaw-prod`

Core operating model:

- `prod` is a managed pet
- `test` is the proving lane
- replay and checkpoint are not the normal `test` / `prod` lifecycle
- gateway is thin and service-plane focused
- runtime-local mutation uses native OpenClaw surfaces
- snapshot-first recovery is the primary restore model

Runtime baseline:

- local first with `ollama/mistral:7b-instruct-32k`
- Together fallback remains configured
- baseline web capability is `web_search`, `web_fetch`, and native `browser`
- the old Playwright/browser detour is gone from the baseline

Repository split:

- `moltbox-services` owns baseline service inputs and service docs
- `moltbox-runtime` holds the final deployable runtime layer used by the current release path
- `moltbox-gateway` owns CLI, procedures, verification, and recovery workflow
