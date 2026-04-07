# Gemma E4B CLI Gap Review

Date: `2026-04-07`

This review captures the operator-surface gaps found while running the live Gemma E4B baseline pass.

## Confirmed Gaps

### 1. Managed deploy depends on host repo state, but the AI operator path cannot update that state cleanly

- `moltbox service deploy <service>` renders from `/opt/moltbox/repos/...`
- the host-side `moltbox-services` and `moltbox-runtime` directories were not writable to the AI test operator
- the host-side directories were not usable git checkouts for a normal `pull exact revision` flow until `safe.directory` was added manually
- even after adding `safe.directory`, the operator still could not write `.git/FETCH_HEAD`
- result: repo changes could be committed and pushed locally, but not promoted through the official host service-deploy path from the same operator session

### 2. No clean service/runtime repo-sync surface

- `moltbox gateway update` exists for the gateway repo
- there is no equivalent official repo-sync/update surface for:
  - `moltbox-services`
  - `moltbox-runtime`
- this forces either host-level break-glass repo mutation or a stale managed baseline

### 3. `moltbox gateway update` is not currently healthy for the AI operator path

- the official `moltbox gateway update` surface was invoked on the live host
- it failed with: `open /srv/moltbox-state/services/gateway/Dockerfile: permission denied`
- result: even the one official repo-update surface that exists is not currently reliable for this operator identity

### 4. `--session-id` isolation is not trustworthy

- concurrent OpenClaw turns were launched with unique requested session ids
- the returned session ids still collapsed onto shared runtime sessions inside each concurrency tier
- result: session-id cannot currently be treated as a hard isolation control for benchmarking or multi-lane work

### 5. No safe official session reset surface

- the benchmark pass needed host-level session directory rotation to keep the `main` lane clean
- `openclaw reset` exists, but it is broader than a safe "clear sessions only" operator action

### 6. Tool-call observability in normal CLI output is weak

- `moltbox test openclaw agent --json` exposes prompt stats and final payloads
- it does not cleanly expose the actual tool-call chain in the standard operator surface
- tool-use verification still had to rely on inference from answers and runtime logs

### 7. Runtime verify was baseline-stale before this change

- `moltbox test verify runtime` was still asserting the old Mistral baseline
- `moltbox test verify web` assumed browser had to stay in the default chat tool allowlist
- this change set fixes that in repo state

### 8. ZFS snapshot-first recovery is not fully reachable from the AI test operator identity

- verified OpenClaw backups worked
- direct `zfs snapshot ...` from the AI operator path did not
- current recovery posture is therefore stronger for a human admin than for the AI test operator

### 9. Default `thinking` is not yet a managed runtime knob

- `--thinking off` materially improved some tool-heavy turns
- there is no clear committed runtime-default setting for that behavior in the current baseline path

## Recommended Follow-Up

1. Add an official repo-sync/update flow for `moltbox-services` and `moltbox-runtime`.
2. Fix the current `moltbox gateway update` permission path for the AI operator role.
3. Add an official "clear runtime sessions" operator surface for `test`.
4. Expose tool-call traces more clearly in the standard `agent --json` operator path.
5. Treat returned session identity as a bug until it matches requested `--session-id` isolation.
6. Decide whether default `thinking off` should become a managed baseline knob or remain a per-run override.
