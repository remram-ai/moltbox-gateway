# Gemma E4B Baseline Review

Date: `2026-04-07`

This review records the live `test` spike used to choose the next local-first baseline.

## Scope

- `prod` stayed down for the full pass
- `test` was restored to `gemma4:e4b-it-q4_K_M`
- the pass compared raw Ollama behavior and real OpenClaw behavior
- browser stayed installed but out of the default chat tool lane

## Final Baseline Decision

Keep the local-first baseline on:

- model: `gemma4:e4b-it-q4_K_M`
- context: `65536`
- Ollama tuning:
  - `OLLAMA_NUM_PARALLEL=2`
  - `OLLAMA_FLASH_ATTENTION=1`
  - `OLLAMA_KV_CACHE_TYPE=q8_0`
- default chat tools:
  - `web_search`
  - `web_fetch`

Do not restore `browser` to the default chat tool lane.

## What The Evidence Showed

### Raw Ollama

- `65536` loaded successfully on GPU and stayed within the 16 GB card budget
- `32768` was a little lighter and a little faster on cold-load chat, but not enough to justify giving up `65536`
- concurrent raw requests remained stable through client-side overlap at `1`, `2`, `3`, and `4`
- the difference between `32768` and `65536` was modest:
  - `32768` used about `10.5 GiB`
  - `65536` used about `11.2 GiB`
  - `32768` cold chat was about `13%` faster, which did not clear the planned `15%` threshold

### OpenClaw

- clean `main` workspace plus the slimmer tool lane brought prompt tokens down into the `~2.6k` range for simple chat
- real OpenClaw chat, logic, and deterministic web probes stayed local on Gemma through concurrency `2`, `3`, and `4`
- deterministic web probe results were good:
  - `HOST=docs.openclaw.ai`
- weather remained the weakest flow:
  - lighter web payloads alone were not enough
  - `thinking off` materially improved the weather result quality
- OpenClaw wall time remained far above raw Ollama wall time, so a meaningful share of perceived slowness is runtime/CLI overhead rather than the base model alone

## Important Findings

- `gemma4:e4b-it-q4_K_M` is the right place to start
- `65536` is the right context target for the managed baseline
- the default chat lane should stay slim:
  - `web_search`
  - `web_fetch`
- `thinking off` is a real speed/quality lever for some tool-heavy flows, but it is not yet exposed as a managed default runtime knob

## Stop Point

This review formalizes the next managed baseline target in repo state.

It does not prove that the managed baseline was deployed through the full service-plane release path on the appliance. That remained blocked by the current host repo/update gap captured separately in `2026-04-07-gemma-e4b-cli-gap-review.md`.
