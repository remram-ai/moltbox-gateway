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
- `test` and `prod` use local Mistral first with Together fallback
- baseline web capability is `web_search`, `web_fetch`, and native `browser`
- the Playwright/browser detour is removed from the baseline and service inventory

Important current caveats:

- native OpenClaw restore is not yet a proven primary recovery path
- tool-choice reliability through ordinary local-model chat still needs human judgment, especially for `web_fetch` and `browser`
