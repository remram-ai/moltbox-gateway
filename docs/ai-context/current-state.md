# Current State

Important current facts:

- host is `moltbox-prime`
- critical appliance paths are on ZFS
- live managed services are:
  - `gateway`
  - `caddy`
  - `ollama`
  - `searxng`
  - `test`
  - `prod`
- the public CLI is already the lightweight surface
- the managed local baseline is Gemma 4 E4B 128K on the minimal Gemma-only lane
- baseline web capability is `web_search` + `web_fetch`
- the default chat lane disables native `memory-core`
- the Playwright/browser detour is removed from the baseline and service inventory

Important current caveats:

- native OpenClaw restore is not yet a proven primary recovery path
- tool-choice reliability through ordinary local-model chat still needs human judgment on noisy fetched pages
- the current AI operator path still lacks a clean service/runtime repo-sync surface for managed deploy
