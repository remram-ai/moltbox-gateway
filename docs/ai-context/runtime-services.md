# Runtime And Services Context

Use this file when the task is mostly about service templates, runtime baseline, or OpenClaw integration.

Service inventory:

- `gateway`
- `caddy`
- `ollama`
- `searxng`
- `openclaw-test`
- `openclaw-prod`

Runtime rules:

- `moltbox-services` owns the baseline service inputs and baseline service docs
- `moltbox-runtime` holds the final deployable runtime artifacts used by the current release path
- `test` is the proving lane
- `prod` changes only after `test` proves the baseline
- replay is not the normal runtime model
- native OpenClaw surfaces are the official runtime mutation path

Model/provider baseline:

- primary: `ollama/gemma4:e4b-it-q4_K_M`
- fallback: Together `Kimi K2.5`
- context: `65536`
- Ollama tuning:
  - `OLLAMA_NUM_PARALLEL=2`
  - `OLLAMA_FLASH_ATTENTION=1`
  - `OLLAMA_KV_CACHE_TYPE=q8_0`

Web baseline:

- `web_search` backed by `searxng`
- built-in `web_fetch`
- native OpenClaw `browser`
- default chat lane excludes `browser`
- no Playwright detour in the current gold baseline

Current caveat:

- local chat reliably handles concise answers, basic logic, and deterministic fetch/search probes
- heavier tool choice through chat is still model-limited on noisy fetched pages
