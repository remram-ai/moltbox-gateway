# Runtime And Services

This document defines the current service inventory and the managed-pet runtime model.

## Service Inventory

| Service | Source | Role |
| --- | --- | --- |
| `gateway` | baseline in `moltbox-services`, deployed artifact via current release path | control plane |
| `caddy` | baseline in `moltbox-services`, deployed artifact via current release path | ingress and TLS |
| `ollama` | baseline in `moltbox-services`, deployed artifact via current release path | local model backend |
| `searxng` | baseline in `moltbox-services`, deployed artifact via current release path | local search backend |
| `openclaw-test` | baseline in `moltbox-services`, final runtime artifact in `moltbox-runtime` | proving runtime |
| `openclaw-prod` | baseline in `moltbox-services`, final runtime artifact in `moltbox-runtime` | protected runtime |

## Runtime Baseline Rules

- `moltbox-services` owns the baseline source inputs
- `moltbox-runtime` holds the final deployable runtime layer used by the current release path
- `test` proves baseline changes before `prod`
- `prod` is not rebuilt from replay history
- service deploy syncs the final runtime artifacts into persisted runtime state

## Replay And Checkpoint

Replay and checkpoint are not part of the normal `test` / `prod` lifecycle.

Legacy replay code may survive temporarily for legacy import or `openclaw-dev`, but it is not the steady-state path for managed runtimes.

## Native OpenClaw Lifecycle

Supported runtime operations use native OpenClaw surfaces:

- `config`
- `health`
- `models`
- `backup`
- `agent`
- other official runtime-local commands

These go through `moltbox test openclaw ...` and `moltbox prod openclaw ...`.

## Model And Provider Baseline

- primary provider: `ollama`
- primary model: `mistral:7b-instruct-32k`
- context window: `32768`
- fallback provider/model: Together with `Kimi K2.5`

## Web Baseline

Current gold baseline:

- `web_search` backed by `searxng`
- built-in `web_fetch`
- native OpenClaw `browser`

Not in the current baseline:

- the Playwright detour
- the old custom `web_browser` path

Current caveat:

- native browser is part of the baseline
- the selected local Mistral model is still unreliable at choosing `web_fetch` and `browser` through ordinary chat without extra steering

## Lean Default Posture

The baseline is intentionally not a heavy coding-agent posture.

Desired defaults:

- concise answers
- lightweight informational capability
- web verification when needed
- no broad coding or service-building behavior as the default assistant personality
