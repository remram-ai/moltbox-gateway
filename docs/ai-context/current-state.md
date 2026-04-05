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
- baseline web capability is `web_search` and `web_fetch`
- the Playwright/browser detour is removed from the baseline

Important current caveats:

- native OpenClaw restore is not yet a proven primary recovery path
- native browser support is not yet a proven gold-baseline dependency
- tool-choice reliability through ordinary local-model chat still needs human judgment
