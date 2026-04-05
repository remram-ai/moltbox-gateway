# Clean Moltbox Validation Plan

Status: Draft validation plan
Date: 2026-04-04

This is the completion standard for the clean Moltbox rebuild. It is intentionally stricter than the earlier lightweight CLI test plan.

Related plan:

- [`./2026-04-04-clean-moltbox-execution-plan.md`](./2026-04-04-clean-moltbox-execution-plan.md)
- [`../runbooks/2026-04-04-zfs-rebuild-ssh-takeover-runbook.md`](../runbooks/2026-04-04-zfs-rebuild-ssh-takeover-runbook.md)

## Validation Philosophy

The box is not done because code changed. The box is done only when:

- the host has been rebuilt on ZFS
- Moltbox-critical paths live on ZFS-backed datasets
- snapshot and rollback capability have been verified
- manual human intervention was limited to the minimum needed to restore remote SSH control
- the repo-level tests pass
- the service plane deploys through the CLI
- the live appliance is clean
- `test` and `prod` chat locally through Mistral
- Together escalation is observable through official runtime/plugin/provider mechanisms
- backup, restore, SSH access, and host ownership all satisfy the target posture

## Required Validation Artifacts To Implement

The builder should leave a reproducible validation suite under `scripts/validation/` or an equivalent documented location.

Minimum required scripts:

- `scripts/validation/00-preflight-capture.sh`
- `scripts/validation/05-zfs-precondition.sh`
- `scripts/validation/10-go-unit.sh`
- `scripts/validation/20-cli-contract.sh`
- `scripts/validation/30-service-plane.sh`
- `scripts/validation/40-runtime-chat.sh`
- `scripts/validation/50-runtime-escalation.sh`
- `scripts/validation/60-ssh-policy.sh`
- `scripts/validation/70-backup-restore.sh`
- `scripts/validation/80-host-hygiene.sh`
- `scripts/validation/90-final-acceptance.sh`

If the builder uses different filenames, the same coverage still has to exist.

## Required Output Directory

Every validation run must write to a timestamped output directory, for example:

- `artifacts/validation/<timestamp>/`

The directory must capture:

- command executed
- key output
- pass/fail result
- host name
- repo revisions
- timestamp

## Layer 0: Host Filesystem Precondition

### Gate 0.1: ZFS Foundation

Validate:

- the rebuilt host is no longer operating from the original EXT4-only layout as the active Moltbox foundation
- a ZFS pool exists
- ZFS datasets exist for the Moltbox-critical paths

Minimum required dataset coverage:

- `/srv/moltbox-state`
- `/srv/moltbox-logs`
- `/var/lib/moltbox`

### Gate 0.2: Dataset Mount Verification

Capture and verify:

- `zpool status`
- `zfs list`
- `findmnt /srv/moltbox-state`
- `findmnt /srv/moltbox-logs`
- `findmnt /var/lib/moltbox`

Expected:

- each required path resolves to a ZFS-backed dataset

### Gate 0.3: Snapshot And Rollback Proof

Run:

- create a ZFS snapshot on the Moltbox dataset boundary
- make a reversible test change in one covered dataset
- roll back or otherwise prove snapshot restore

Expected:

- snapshot creation succeeds
- rollback succeeds
- evidence is captured

### Gate 0.4: Remote SSH Takeover

Validate:

- the rebuilt host is reachable over SSH
- the preserved human admin path still works
- the builder can take over remote operations immediately after the rebuild

Expected:

- no long manual Linux runbook remains after SSH access is restored
- the approved manual path was bounded by [`../runbooks/2026-04-04-zfs-rebuild-ssh-takeover-runbook.md`](../runbooks/2026-04-04-zfs-rebuild-ssh-takeover-runbook.md)

## Layer 1: Repo And Unit Tests

### Gate 1.1: Gateway Go Tests

Run:

- `go test ./...`

Must pass:

- `internal/config`
- `internal/deploystate`
- `internal/docker`
- `internal/gateway`
- `internal/mcpstdio`
- `internal/orchestrator`
- `internal/secrets`
- `internal/sshwrap`
- any new `pkg/cli` parser tests

### Gate 1.2: Sibling Repo Checks

Run whatever is required for changed files in:

- `moltbox-services`
- `moltbox-runtime`
- `remram-skills` if touched

Minimum:

- syntax/format validation on changed YAML or templates
- plugin build/test if a new Remram plugin was created

## Layer 2: CLI Contract Tests

### Gate 2.1: Help Surface

Run:

- `moltbox --help`
- `moltbox gateway --help`
- `moltbox service --help`
- `moltbox test --help`
- `moltbox prod --help`
- `moltbox secret --help`

Must include:

- `bootstrap gateway`
- `gateway status`
- `gateway logs`
- `gateway update`
- `service list`
- `service status`
- `service deploy`
- `service restart`
- `service logs`
- `test openclaw`
- `prod openclaw`
- `ollama`
- `secret set|list|delete`

Must not include:

- `dev`
- `opensearch`
- `checkpoint`
- `skill`
- `plugin`
- `gateway docker`
- `gateway service`
- `service secrets`

### Gate 2.2: Removed Surface Failures

Run and confirm explicit failure:

- `moltbox dev reload`
- `moltbox opensearch --version`
- `moltbox prod plugin list`
- `moltbox test checkpoint`
- `moltbox gateway docker ping`
- `moltbox gateway service deploy caddy`
- `moltbox service secrets list`

Expected:

- non-zero exit
- clear message that the surface is unsupported or retired

## Layer 3: Route And Handler Tests

### Gate 3.1: Route Mapping

Validate these route classes:

- `bootstrap gateway`
- `gateway status|logs|update`
- `service list|status|deploy|restart|logs`
- `test openclaw <native args>`
- `prod openclaw <native args>`
- `ollama <native args>`
- `secret set|list|delete`

### Gate 3.2: Handler Coverage

Must be covered by tests or scripted live checks:

- gateway status
- service list
- service status
- service deploy
- service restart
- service logs
- OpenClaw passthrough
- secret set/list/delete

## Layer 4: Live Appliance Baseline Capture

Run before destructive cutover:

- `whoami`
- `hostname`
- `uname -a`
- `docker ps`
- `docker inspect gateway caddy ollama openclaw-dev openclaw-test openclaw-prod opensearch`
- `docker exec ollama ollama list`
- `moltbox gateway status`
- `moltbox test openclaw status`
- `moltbox prod openclaw status`
- `moltbox test openclaw agent --agent main --message "Reply with TEST_OK only." --json`
- `moltbox prod openclaw agent --agent main --message "Reply with PROD_OK only." --json`

Capture:

- current local model
- current context window
- current plugin warnings
- current fallback behavior

## Layer 5: Service Plane Tests

### Gate 5.1: Service Inventory

Run:

- `moltbox service list`

Expected final result:

- `gateway`
- `caddy`
- `ollama`
- `test`
- `prod`

No extras.

### Gate 5.2: Service Status

Run:

- `moltbox service status gateway`
- `moltbox service status caddy`
- `moltbox service status ollama`
- `moltbox service status test`
- `moltbox service status prod`

Expected:

- each target service present
- each target service healthy

### Gate 5.3: Service Deployment And Restart

Run:

- `moltbox service deploy ollama`
- `moltbox service deploy test`
- `moltbox service deploy prod`
- `moltbox service deploy caddy`
- `moltbox service restart caddy`

Expected:

- commands succeed through the CLI
- no manual `docker compose` steps are required
- health remains good after mutation

### Gate 5.4: Service Logs

Run:

- `moltbox service logs gateway`
- `moltbox service logs test`
- `moltbox service logs prod`

Expected:

- logs come back for the requested service
- service name maps to the correct container

## Layer 6: Runtime And Model Tests

### Gate 6.1: Ollama Inventory

Run:

- `moltbox ollama list`

Expected:

- `mistral:7b-instruct-32k` is present

### Gate 6.2: Runtime Health

Run:

- `moltbox test openclaw status`
- `moltbox prod openclaw status`
- `moltbox test openclaw health --json`
- `moltbox prod openclaw health --json`

Expected:

- status succeeds
- health succeeds
- no warnings about missing baseline config files

### Gate 6.3: Plugin Trust

Run:

- `moltbox test openclaw plugins list`
- `moltbox prod openclaw plugins list`

Expected:

- plugin inventory comes from OpenClaw
- `plugins.allow` is explicit
- no untracked extension warning remains

## Layer 7: End-To-End Chat Tests

### Gate 7.1: Test Runtime Chat

Run:

- `moltbox test openclaw agent --agent main --message "Reply with TEST_OK only." --json`

Expected:

- success
- payload text contains `TEST_OK`
- provider is `ollama`
- model is `mistral:7b-instruct-32k`
- command does not succeed only because it silently fell back after gateway failure

### Gate 7.2: Prod Runtime Chat

Run:

- `moltbox prod openclaw agent --agent main --message "Reply with PROD_OK only." --json`

Expected:

- success
- payload text contains `PROD_OK`
- provider is `ollama`
- model is `mistral:7b-instruct-32k`
- command does not succeed only because it silently fell back after gateway failure

### Gate 7.3: Command-Shape Failure

Run:

- `moltbox test openclaw agent --message "bad"`
- `moltbox prod openclaw agent --message "bad"`

Expected:

- explicit failure explaining that `--agent`, `--to`, or `--session-id` is required

## Layer 8: Together Escalation Tests

### Goal

Prove that Together escalation exists and that the gateway is not secretly owning it.

### Gate 8.1: Configuration Presence

Validate:

- Together API key exists in the proper scope
- runtime config points to the approved Together model chain
- the provider path is official OpenClaw provider/plugin configuration

### Gate 8.2: Escalation Evidence On `test`

Run the documented escalation trigger on `test`.

Acceptable trigger methods:

- a documented explicit reasoning invocation that should use Together
- a deliberate local-provider failure test that forces fallback
- a plugin-specific invocation if a dedicated Remram plugin is introduced

Expected evidence:

- provider/model output shows a Together-backed path
- evidence comes from OpenClaw runtime output, logs, or plugin telemetry
- behavior is attributable to plugin/provider configuration, not gateway replay or hard-coded gateway logic

### Gate 8.3: Escalation Evidence On `prod`

Run only after a restore point exists.

Expected:

- same provider/model evidence as `test`
- no manual host-side mutation needed outside the approved workflow

## Layer 9: SSH Identity Policy Tests

### Gate 9.1: `moltbox-ai-test`

Allowed:

- `moltbox service status test`
- `moltbox test openclaw status`
- `moltbox test openclaw agent --agent main --message "Reply with TEST_OK only." --json`
- allowed test-side mutation through the documented service-plane or runtime workflow

Denied:

- `uname -a`
- arbitrary shell commands

### Gate 9.2: `moltbox-ai-prod`

Allowed:

- `moltbox gateway status`
- `moltbox service logs prod`
- `moltbox prod openclaw status`

Denied:

- `moltbox service deploy prod`
- `moltbox secret set prod TEST value`
- arbitrary shell commands

## Layer 10: Host Ownership And Path Tests

Validate:

- `moltbox` binary lives at `/usr/local/bin/moltbox`
- config lives at `/etc/moltbox/config.yaml`
- repo checkouts live under `/opt/moltbox/repos`
- state lives under `/srv/moltbox-state`
- logs live under `/srv/moltbox-logs`
- secrets live under `/var/lib/moltbox/secrets`
- no production runtime path depends on `/home/*`
- no required path is owned by a personal account

Evidence commands:

- `stat`
- `find ... -user <personal-user>`
- `ls -ld`

## Layer 11: Backup And Restore Tests

### Gate 11.1: Backup Job

Validate:

- backup timer exists
- backup timer is enabled
- last run evidence exists

### Gate 11.2: OpenClaw Native Backups

Run:

- `moltbox test openclaw backup create --verify`
- `moltbox prod openclaw backup create --verify`

or the documented equivalent if backup runs through a scheduled wrapper.

### Gate 11.3: Restore Drill

Minimum acceptable drill:

- restore `test` runtime from a recent backup in a controlled check

Preferred:

- restore from machine snapshot or appliance backup on a disposable validation target

## Layer 12: OS Patching Tests

Validate:

- unattended upgrades or equivalent patch timer exists
- patch status is discoverable
- last patch run evidence exists
- reboot policy is documented

## Layer 13: Final Acceptance Script

The final acceptance run must execute the full chain in one ordered pass:

1. preflight metadata capture
2. `go test ./...`
3. CLI help/contract checks
4. service-plane checks
5. runtime health checks
6. chat tests on `test`
7. chat tests on `prod`
8. Together escalation tests
9. SSH identity tests
10. host ownership/path tests
11. backup tests
12. patching tests

## Hard Fail Conditions

Reject the build if any of these are true:

- the rebuilt host still requires extended manual Linux administration after SSH should have been restored
- `dev` container or service still exists
- OpenSearch still exists
- any target container is unhealthy
- ordinary chat does not report local Ollama Mistral usage
- chat succeeds only through hidden fallback after gateway failure
- Together escalation cannot be observed
- plugin trust remains implicit
- any required path is owned by a personal account
- `moltbox-ai-prod` can mutate the box
- backups or patching remain unconfigured

## Completion Standard

The builder may mark the work complete only when:

- the full validation suite passes
- the evidence bundle exists
- the operator can paste the final acceptance commands and get the same result
