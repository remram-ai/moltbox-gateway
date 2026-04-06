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

- primary: `ollama/mistral:7b-instruct-32k`
- fallback: Together `Kimi K2.5`
- context: `32768`

Web baseline:

- `web_search` backed by `searxng`
- built-in `web_fetch`
- native OpenClaw `browser`
- no Playwright detour in the current gold baseline

Current caveat:

- local chat reliably handles short answers
- heavier tool choice through chat is still model-limited on local Mistral
