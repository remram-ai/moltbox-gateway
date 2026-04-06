# Operator Workflows Context

Use this file when the task is about operating the live appliance through supported surfaces.

Official command families:

- `moltbox gateway ...`
- `moltbox service ...`
- `moltbox test openclaw ...`
- `moltbox test verify ...`
- `moltbox prod openclaw ...`
- `moltbox prod verify ...`
- `moltbox ollama ...`
- `moltbox secret ...`

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
