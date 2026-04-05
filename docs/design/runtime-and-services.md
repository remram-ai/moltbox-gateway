# Runtime And Services

This document defines the current service inventory and the managed-pet runtime model.

## Service Inventory

| Service | Source | Role |
| --- | --- | --- |
| `gateway` | `moltbox-services` | control plane |
| `caddy` | `moltbox-services` | ingress and TLS |
| `ollama` | `moltbox-services` | local model backend |
| `searxng` | `moltbox-services` | local search backend |
| `openclaw-test` | `moltbox-services` + `moltbox-runtime` | proving runtime |
| `openclaw-prod` | `moltbox-services` + `moltbox-runtime` | protected runtime |

## Runtime Baseline Rules

- `moltbox-runtime` owns the approved baseline
- `test` proves baseline changes before `prod`
- `prod` is not rebuilt from replay history
- service deploy syncs the runtime baseline files into persisted runtime state

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
