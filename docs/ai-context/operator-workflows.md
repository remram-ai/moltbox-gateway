# Operator Workflows Context

Use this file when the task is about operating the live appliance through supported surfaces.

Official command families:

- `moltbox gateway ...`
- `moltbox service ...`
- `moltbox test openclaw ...`
- `moltbox test verify ...`
- `moltbox prod openclaw ...`
- `moltbox prod verify runtime`
- `moltbox ollama ...`
- `moltbox secret ...`

Current SSH entry points:

- `ssh moltbox` or `ssh moltbox-admin` for the human admin path
- `ssh moltbox-ai-test` for the restricted test operator lane
- `ssh moltbox-ai-prod` for the restricted prod diagnostics lane
- `ssh moltbox-breakglass` for emergency admin only

Normal workflows:

- inspect appliance:
  - `moltbox gateway status`
  - `moltbox service list`
- verify runtimes:
  - `moltbox test verify runtime`
  - `moltbox test verify browser`
  - `moltbox test verify web`
  - `moltbox prod verify runtime`
- mutate service plane:
  - `moltbox service deploy <service>`
  - `moltbox service restart <service>`
  - `moltbox service remove <legacy-service>`
- mutate runtime:
  - `moltbox test openclaw ...`
- inspect prod:
  - `moltbox prod openclaw ...`

Do not use as normal paths:

- raw Docker
- replay-era namespaces
- break-glass SSH when the official CLI can do the job

Legacy naming note:

- do not use `jason-codex` or `codex-bootstrap` as the current role names
- use the role-based aliases above and the Gateway host-ops docs as the current source of truth
