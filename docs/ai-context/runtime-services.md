# Runtime And Services Context

Use this file when the task is mostly about service templates, runtime baselines, model/provider wiring, or OpenClaw integration.

Final service inventory:

- `gateway`
- `caddy`
- `ollama`
- `openclaw-test`
- `openclaw-prod`

Removed from the target box:

- `openclaw-dev`
- `opensearch`

Runtime rules:

- `moltbox-runtime` owns desired baseline
- OpenClaw owns native runtime behavior
- `test` is the proving lane
- `prod` is protected and only changes through official runtime surfaces

Local model baseline:

- provider `ollama`
- model `mistral:7b-instruct-32k`
- context `32768`

Together rule:

- must remain available
- must be expressed through official OpenClaw provider or plugin behavior
- must not be hidden in gateway logic

Cortex MVP seam:

- Phase 0 uses `memory.backend = qmd`
- Phase 0 keeps `contextEngine = legacy`
- Phase 1 allows only `cortex-phase1-bridge`
- Phase 1 enables that bridge in `dry-run` mode first

Plugin trust rule:

- `plugins.allow` must be explicit
- no untracked steady-state plugin code

Canonical source:

- `../design/runtime-and-services.md`
