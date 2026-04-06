# Gemma 4 Test Spike Plan

Status: Draft spike plan
Date: 2026-04-06

This plan is a controlled live-appliance spike, not a baseline change package.

The goal is to get `test` to a healthy, trustworthy Gemma runtime before we formalize anything in `moltbox-services`, `moltbox-runtime`, or `moltbox-gateway`, and before we consider any `prod` cutover.

Related current docs:

- [`../guides/operator-guide.md`](../guides/operator-guide.md)
- [`../design/runtime-and-services.md`](../design/runtime-and-services.md)
- [`../design/backup-and-recovery.md`](../design/backup-and-recovery.md)
- [`./2026-04-04-clean-moltbox-validation-plan.md`](./2026-04-04-clean-moltbox-validation-plan.md)

## Spike Boundaries

Do this:

- pause `prod` first
- use `test` as the proving lane
- use normal CLI surfaces where they exist
- use SSH or host access if a required tuning or runtime control is missing from the current CLI/operator surface
- record every host-only change so it can be converted into the proper managed path later

Do not do this yet:

- do not formalize repo baseline changes
- do not clean up Gateway verification or docs to assume Gemma is the baseline
- do not cut both `test` and `prod` to Gemma
- do not treat this spike as the final managed deployment path

## Target Runtime Assumptions

The spike target is:

- local model: `gemma4:e4b-it-q4_K_M`
- context: `131072`
- parallel: `2`
- `OLLAMA_FLASH_ATTENTION=1`
- `OLLAMA_KV_CACHE_TYPE=q8_0`
- Together fallback remains in place
- OpenClaw baseline config otherwise stays aligned with the current good baseline unless a change is required to make Gemma healthy

## Current Operator-Surface Gaps

These gaps are acceptable for the spike, but they must be recorded:

- the current public CLI does not expose a normal `pause prod` surface
- the current `ollama` service template does not yet render the required tuning environment variables
- the current `moltbox test verify runtime` flow still asserts the Mistral baseline and is therefore not sufficient by itself to validate the Gemma spike
- `moltbox test verify web` proves backend and config availability, not that the local model actually chooses `web_search` correctly through chat

## Required Evidence Before Change

Capture and save:

- current repo revisions in `moltbox-gateway`, `moltbox-services`, and `moltbox-runtime`
- `moltbox gateway status`
- `moltbox service status ollama`
- `moltbox service status test`
- `moltbox service status prod`
- `moltbox ollama list`
- `moltbox ollama ps`
- `moltbox test openclaw health --json`
- `moltbox test openclaw models status --json`
- `moltbox prod openclaw health --json`
- `moltbox prod openclaw models status --json`
- `moltbox test openclaw agent --agent main --message "Reply with TEST_PRECHECK only." --json`
- `moltbox prod openclaw agent --agent main --message "Reply with PROD_PRECHECK only." --json`

Record whether the current agent output exposes provider and model metadata directly. If it does not, note that and capture the nearest trustworthy runtime evidence instead.

## Rollback Protection Before Spike

Before changing anything:

- run `moltbox test openclaw backup create --verify`
- run `moltbox prod openclaw backup create --verify`
- create a ZFS snapshot or the best available host restore point for the appliance state
- record the exact snapshot or restore-point identifiers
- write down the exact rollback procedure before making the model swap

If the host is not yet on ZFS for the relevant paths, explicitly record that the rollback posture is weaker than the intended snapshot-first model.

## Pause Production First

`prod` should be paused before the Gemma spike begins.

If the current CLI/operator surface still lacks a normal pause control, use the smallest reversible host-side action available on the appliance and record:

- the exact command used
- what was paused
- how to resume it

The spike should not mutate `prod` runtime config. The point of pausing `prod` is to reduce contention and reduce risk while `test` is being pushed harder.

## Allowed Spike Changes

Only change what is needed to make `test` run Gemma credibly:

- pull `gemma4:e4b-it-q4_K_M` through `moltbox ollama ...` if possible
- apply Ollama tuning needed for:
  - `OLLAMA_CONTEXT_LENGTH=131072`
  - `OLLAMA_NUM_PARALLEL=2`
  - `OLLAMA_FLASH_ATTENTION=1`
  - `OLLAMA_KV_CACHE_TYPE=q8_0`
- update the `test` runtime only so that:
  - `agents.defaults.model.primary` points to `ollama/gemma4:e4b-it-q4_K_M`
  - `agents.defaults.contextTokens` is `131072`
  - the Ollama provider model inventory contains `gemma4:e4b-it-q4_K_M`
  - Together fallback remains intact
- restart only the services required for the spike

Keep the current browser and SearXNG web baseline intact unless a change is required just to restore the known-good baseline behavior.

## Host-Only Change Ledger

If any part of the spike is done outside the current CLI/operator surface, capture each item in this form:

| Area | Exact path or service | Exact change | Why needed | How to revert | Needs later formalization in |
| --- | --- | --- | --- | --- | --- |
| example | rendered Ollama env on host | set `OLLAMA_CONTEXT_LENGTH=131072` | CLI has no tuning surface yet | restore previous env and restart `ollama` | `moltbox-services` |

This ledger is mandatory output from the spike.

## Validation Requirements

The spike is not done because Gemma boots. The spike is done only if real chat on `test` looks healthy.

### 1. Confirm `test` Is Actually Using Gemma

Verify through CLI and runtime inspection, not assumption.

Minimum evidence:

- `moltbox test openclaw models status --json`
- `moltbox ollama ps`
- at least one successful `moltbox test openclaw agent ... --json` run whose metadata, logs, or paired runtime evidence shows:
  - provider is local Ollama
  - active model is `gemma4:e4b-it-q4_K_M`

Reject the spike if:

- the runtime silently stays on Mistral
- Together fallback is silently carrying normal chat
- model evidence is ambiguous and cannot be resolved from the available runtime signals

### 2. Run Real End-To-End Chat Through The CLI Path

Do not accept config inspection as a substitute for chat.

Run at least these surfaces through the normal CLI:

- short smoke:
  - `moltbox test openclaw agent --agent main --session-id gemma-smoke --message "Reply with GEMMA_TEST_OK only." --json`
- same-session follow-up:
  - `moltbox test openclaw agent --agent main --session-id gemma-smoke --message "What exact token did I ask you to reply with?" --json`
- ordinary factual prompt:
  - one normal non-tool prompt that checks coherence and formatting

Capture the exact commands used and the resulting provider/model evidence.

### 3. Verify Baseline Chat Quality

Responses should be:

- coherent
- on-topic
- correctly formatted for simple instruction-following
- free of obvious nonsense, severe drift, or broken formatting

At minimum, evaluate:

- simple instruction following
- a short factual answer
- a same-session follow-up that confirms the runtime is behaving sanely under ordinary chat

### 4. Verify Basic Web-Search Tool Use

This is required because web-tool choice has been a known weak spot on the previous local model.

First prove the backend path is up:

- `moltbox test verify web`

Then run at least one real tool-using chat prompt through the normal CLI that should cause the model to choose `web_search`, for example:

- ask it to use web search to find an official project page and return the URL plus a short description

Accept only if you can show:

- the model chose the tool or the runtime evidence clearly shows the tool path was exercised
- the final response is sensible and grounded in the retrieved result

If the official chat output does not expose tool calls directly, use the closest trustworthy runtime evidence already available in this stack and record that limitation.

The debrief must say whether web-search behavior is materially better, the same, or worse than the old Mistral baseline.

### 5. Verify Overall Runtime Health

Confirm:

- `test` starts cleanly
- `ollama` starts cleanly
- there is no obvious mismatch around model, context, or cache settings
- there are no immediate repeats of the prior failure modes

Look for:

- OOM or near-OOM behavior
- repeated restarts
- KV or cache allocation failures
- degraded CLI responsiveness
- hidden fallback masking local failure

Because the target includes `parallel=2`, include a small overlap test:

- start one longer CLI chat request on `test`
- while it is still running, issue a second CLI chat request on `test`
- record whether both complete, whether latency becomes unusable, and whether the runtime remains stable

If full `131072` plus `parallel=2` is unstable, reduce context to `98304` and then `65536` before giving up `parallel=2`.

## Debrief Required Before Any Formalization

Stop after the spike and come back with a debrief. Do not package the repo changes yet.

The debrief must include:

- whether `test` is healthy on Gemma
- exactly how the active model was verified
- what chat test surfaces were used
- what web and tool-use tests were run
- whether the responses looked sane
- whether web-search behavior improved
- any issues still blocking confidence
- every change made outside the current CLI/operator surface
- what needs to be formalized later in:
  - `moltbox-services`
  - `moltbox-runtime`
  - `moltbox-gateway`

## Stop Point

Stop once:

- `test` appears healthy on Gemma
- the debrief is written
- the host-only change ledger is complete

Then hand back for manual testing.

Do not do the formal requirement pass yet.
Do not do the hard cut for both `test` and `prod` yet.
