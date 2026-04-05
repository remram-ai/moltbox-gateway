# Future State

Target appliance:

- `gateway`
- `caddy`
- `ollama`
- `searxng`
- `openclaw-test`
- `openclaw-prod`

Target rules:

- no appliance `dev`
- no OpenSearch
- no Playwright detour in the baseline
- `test` proves changes before `prod`
- `prod` is protected
- gateway is not the hidden replay engine
- native OpenClaw lifecycle is the official runtime mutation path
- snapshots are the primary restore-point mechanism
