# Lightweight CLI Test Plan

Status: Draft test plan
Date: 2026-04-04

This is an earlier scoping test plan. Use [`./2026-04-04-clean-moltbox-validation-plan.md`](./2026-04-04-clean-moltbox-validation-plan.md) as the primary builder-facing validation package.

This plan validates the new lightweight Moltbox CLI and the supporting appliance posture.

Related plan:

- [`./2026-04-04-lightweight-cli-implementation-plan.md`](./2026-04-04-lightweight-cli-implementation-plan.md)

## Purpose

This plan is meant to catch:

- parser drift
- route and handler drift
- service-plane deployment regressions
- runtime model-routing regressions
- chat-path regressions
- cloud-escalation regressions
- SSH wrapper policy regressions
- user-home or user-ownership regressions
- missing backup and patching hygiene

## Validation Buckets

### Bucket A: Parser And Help Tests

Validate:

- `moltbox --help`
- `moltbox --version`
- active namespaces
- removed namespaces fail explicitly

Must fail explicitly:

- `dev`
- `checkpoint`
- `skill`
- `plugin`
- `opensearch`
- `gateway docker ...`
- `gateway service ...`
- `<env> secrets ...`
- `service secrets ...`

### Bucket B: HTTP Client And Handler Routing Tests

Validate route mapping for:

- `bootstrap gateway`
- `gateway status`
- `gateway logs`
- `gateway update`
- `service list`
- `service status <service>`
- `service deploy <service>`
- `service restart <service>`
- `service logs <service>`
- `test openclaw <command>`
- `prod openclaw <command>`
- `ollama <native command>`
- `secret set|list|delete <scope> ...`

### Bucket C: Orchestrator And Service Tests

Validate:

- service status for `gateway`, `caddy`, `ollama`, `test`, `prod`
- service deploy and restart for `caddy`, `ollama`, `test`, `prod`
- gateway update remains separate from service deploy and restart
- service logs return the expected container target

### Bucket D: SSH Wrapper Policy Tests

Validate identity policy for:

- `moltbox-ai-test`
- `moltbox-ai-prod`

Expectations:

- both identities are forced-command only
- neither identity can run arbitrary shell
- `moltbox-ai-test` can perform the allowed test-appliance mutations
- `moltbox-ai-prod` is restricted to safe diagnostics and CLI-only access

### Bucket E: Host Ownership And Path Tests

Validate:

- no production config lookup uses `/home/*`
- `/usr/local/bin/moltbox` is the effective CLI path
- `/etc/moltbox/config.yaml` is the effective config path
- `/srv/moltbox-state`, `/srv/moltbox-logs`, `/var/lib/moltbox`, and `/opt/moltbox/repos` are system-owned
- no runtime mount path is owned by a personal account

### Bucket F: Backup And Patching Hygiene Tests

Validate:

- nightly backup timer exists and is enabled
- pre-change restore-point workflow exists and is documented
- OpenClaw native backup creation works for `test` and `prod`
- patching timer or unattended patching configuration exists
- patch status is visible through the documented host workflow

### Bucket G: Runtime Model And Plugin Tests

Validate:

- `mistral:7b-instruct-32k` is present in the Ollama inventory
- `test` and `prod` report `ollama/mistral:7b-instruct-32k` as the ordinary local-first chat model
- Together escalation is exposed through official plugin/provider surfaces
- any Remram-specific escalation capability is present as a plugin-backed capability, not only as hard-coded gateway behavior or skill documentation

### Bucket H: End-To-End Functional Acceptance

Validate:

- all five target containers are healthy at the same time
- CLI service-plane commands reflect the real appliance state
- CLI chat reaches both `test` and `prod`
- ordinary chat stays local on `mistral:7b-instruct-32k`
- cloud escalation is observable when intentionally exercised
- the end-to-end CLI path succeeds cleanly without hidden fallback masking broken routing

## Scenario Matrix

### 0. Live Baseline Capture

Capture before refactor work:

- host identity and hostname
- current SSH login user
- running container set and health
- container mounts and resource limits
- current Ollama model inventory
- current `test` and `prod` `openclaw.json`
- current `test` and `prod` `model-runtime.yml`
- current `openclaw status` output for `test` and `prod`
- current `openclaw plugins list` and `openclaw skills list` output for `test` and `prod`

The 2026-04-04 captured baseline currently shows:

- host `moltbox-prime`
- SSH alias still logging in as personal user `jpekovitch`
- live services `gateway`, `caddy`, `ollama`, `openclaw-dev`, `openclaw-test`, `openclaw-prod`, `opensearch`
- all containers healthy but no Docker memory or CPU limits
- local default chat model `ollama/mistral:7b-instruct-32k`
- runtime context window `32768`
- `plugins.allow` unset with untracked `behavior-guard` auto-loading warnings

### 1. Parser Contract

Check that these succeed:

- `moltbox bootstrap gateway`
- `moltbox gateway status`
- `moltbox gateway logs`
- `moltbox gateway update`
- `moltbox service list`
- `moltbox service status gateway`
- `moltbox service deploy test`
- `moltbox service restart caddy`
- `moltbox service logs prod`
- `moltbox test openclaw health --json`
- `moltbox prod openclaw health --json`
- `moltbox ollama list`
- `moltbox secret list service`

Check that these fail with explicit guidance:

- `moltbox dev reload`
- `moltbox test checkpoint`
- `moltbox prod skill list`
- `moltbox prod plugin list`
- `moltbox opensearch --version`
- `moltbox gateway docker ping`
- `moltbox gateway service deploy caddy`
- `moltbox service secrets list`

### 2. Bootstrap And Gateway Lifecycle

Validate on a clean host:

1. `moltbox bootstrap gateway`
2. `moltbox gateway status`
3. `moltbox service list`
4. `moltbox service status gateway`

Expected:

- gateway comes up without manual Docker commands
- the CLI can reach the gateway on localhost
- the service list contains only `gateway`, `caddy`, `ollama`, `test`, `prod`

### 3. Service Plane

Validate:

- `moltbox service deploy ollama`
- `moltbox service deploy test`
- `moltbox service deploy prod`
- `moltbox service deploy caddy`
- `moltbox service restart caddy`
- `moltbox service logs gateway`
- `moltbox service status test`

After every mutation, verify:

- container health
- deployment provenance
- the running image matches the recorded image
- `moltbox service status <service>` matches the real container health

### 4. OpenClaw Passthrough

Validate safe diagnostics:

- `moltbox test openclaw health --json`
- `moltbox prod openclaw health --json`
- `moltbox test openclaw plugins list`
- `moltbox prod openclaw skills list`
- `moltbox test openclaw status`
- `moltbox prod openclaw status`

Validate controlled mutation on the test appliance only:

- install a known plugin through `moltbox test openclaw plugins install ...`
- list it natively
- remove it natively

Validate protected prod workflow:

- create a restore point through the documented host workflow
- run one official OpenClaw config or plugin action through `moltbox prod openclaw ...`
- verify health afterward
- verify `plugins.allow` and plugin trust posture remain explicit

### 5. Ollama Passthrough

Validate:

- `moltbox ollama list`
- `moltbox ollama pull <required-model>`

Expected:

- commands reach the native Ollama CLI
- model files land in the system-owned shared model store
- `mistral:7b-instruct-32k` is present and remains the approved default local model

### 6. Runtime Model And Chat Path

Validate on `test`:

- `moltbox test openclaw agent --agent main --message "Reply with TEST_OK only." --json`

Expected:

- command succeeds
- payload contains `TEST_OK`
- metadata reports `provider=ollama`
- metadata reports `model=mistral:7b-instruct-32k`
- no gateway pairing error or hidden embedded fallback is required for success in the final target state

Validate on `prod`:

- `moltbox prod openclaw agent --agent main --message "Reply with PROD_OK only." --json`

Expected:

- command succeeds
- payload contains `PROD_OK`
- metadata reports `provider=ollama`
- metadata reports `model=mistral:7b-instruct-32k`
- no gateway pairing error or hidden embedded fallback is required for success in the final target state

Validate command-shape contract:

- `moltbox test openclaw agent --message "bad invocation"` must fail with explicit guidance that `--agent`, `--to`, or `--session-id` is required
- same requirement must hold for `prod`

### 7. Together Escalation Path

Validate on `test` only first:

- install or enable the approved Together escalation capability through official OpenClaw plugin/config surfaces
- confirm `TOGETHER_API_KEY` is present in the runtime scope
- intentionally exercise the escalation path using the documented trigger or a deliberate local-provider failure test

Expected:

- provider/model evidence shows the request moved to Together
- the observed provider/model path matches the approved escalation chain
- evidence comes from official runtime output, logs, or plugin telemetry
- the behavior is attributable to plugin/provider configuration, not gateway replay or hidden hard-coded logic

Promotion gate for `prod`:

- do not accept `prod` promotion until the same escalation evidence is reproducible there with a restore point taken first

### 8. Secret Management

Validate:

- `moltbox secret set service TEST_SECRET <value>`
- `moltbox secret list service`
- `moltbox secret delete service TEST_SECRET`
- `moltbox secret list test`
- `moltbox secret list prod`

Expected:

- values are never echoed back
- secrets land under `/var/lib/moltbox/secrets/<scope>/`
- file permissions remain restricted

### 9. SSH Identity Policy

Validate for `moltbox-ai-test`:

- allowed: `moltbox service status test`
- allowed: `moltbox test openclaw health --json`
- allowed on the designated test appliance: `moltbox service deploy test`
- denied: `uname -a`

Validate for `moltbox-ai-prod`:

- allowed: `moltbox gateway status`
- allowed: `moltbox service logs prod`
- allowed: `moltbox prod openclaw health --json`
- denied: `moltbox service deploy prod`
- denied: `moltbox secret set prod TEST value`
- denied: `uname -a`

### 10. Host Ownership And Config Path

Validate:

- `moltbox` is installed at `/usr/local/bin/moltbox`
- config is read from `/etc/moltbox/config.yaml`
- `ConfigPath()` tests no longer pass when only a user-home config exists
- repo checkouts live under `/opt/moltbox/repos`
- no service or secret path is owned by a personal account

### 11. Backup And Patching Hygiene

Validate:

- nightly backup timer is installed and enabled
- restore-point procedure exists and can be invoked before a prod change
- `openclaw backup create --verify` succeeds for both runtimes
- patch timer or unattended patching configuration is installed
- patch status and last-run evidence are visible in the documented host workflow

### 12. Final End-To-End Acceptance

Run in order:

1. `moltbox gateway status`
2. `moltbox service list`
3. `moltbox service status gateway`
4. `moltbox service status caddy`
5. `moltbox service status ollama`
6. `moltbox service status test`
7. `moltbox service status prod`
8. `moltbox ollama list`
9. `moltbox test openclaw agent --agent main --message "Reply with TEST_OK only." --json`
10. `moltbox prod openclaw agent --agent main --message "Reply with PROD_OK only." --json`
11. documented Together escalation test on `test`
12. documented Together escalation test on `prod` after restore point

Accept only when:

- all target containers are healthy
- both runtime chat commands succeed through the CLI
- both ordinary chat commands report `provider=ollama` and `model=mistral:7b-instruct-32k`
- Together escalation is observable and reproducible
- the box was brought to that state through the CLI and documented host workflows, not ad hoc manual Docker surgery

## Evidence Capture

Capture for every validation run:

- date
- target appliance name
- gateway repo revision
- exact command
- CLI output or key fields
- before and after deployment records for service mutations
- before and after backup evidence for backup tests
- ownership and permission output for host-path checks
- runtime provider/model evidence for chat and escalation tests
- whether any command succeeded only by falling back to an unintended path

## Regression Gates

Do not accept the refactor when any of these are true:

- `moltbox --help` still exposes removed surfaces
- `service` is not the primary service-plane namespace
- any production path still depends on `/home/*` for runtime operation
- any service or secret path is owned by a personal account
- `moltbox-ai-prod` can mutate production state
- `moltbox-ai-test` can escape to arbitrary shell
- OpenClaw passthrough does not reach the official runtime CLI
- ordinary chat does not report `provider=ollama` and `model=mistral:7b-instruct-32k`
- Together escalation cannot be observed through official plugin/provider behavior
- chat success depends on hidden fallback after a gateway pairing or routing failure
- backup timers or patching timers are absent
- the clean-host bootstrap requires manual Docker commands outside the documented bootstrap path
