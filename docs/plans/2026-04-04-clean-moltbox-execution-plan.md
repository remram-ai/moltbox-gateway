# Clean Moltbox Execution Plan

Status: Draft execution plan
Date: 2026-04-04

This is the implementation-grade plan for taking the current Moltbox appliance from its present mixed state to a clean, usable box that an operator can manage through the `moltbox` CLI.

This document is meant to be executed by an AI builder with full repo access and temporary full SSH access to the live box. The goal is to end with CLI-driven operations and a validation suite that proves the box is done.

Related records:

- [`../runbooks/2026-04-04-zfs-rebuild-ssh-takeover-runbook.md`](../runbooks/2026-04-04-zfs-rebuild-ssh-takeover-runbook.md)
- [`../reviews/2026-04-04-cli-surface-review.md`](../reviews/2026-04-04-cli-surface-review.md)
- [`../decisions/2026-04-04-openclaw-operating-model.md`](../decisions/2026-04-04-openclaw-operating-model.md)
- [`./2026-04-04-clean-moltbox-validation-plan.md`](./2026-04-04-clean-moltbox-validation-plan.md)
- [`./2026-04-04-clean-moltbox-builder-prompt.md`](./2026-04-04-clean-moltbox-builder-prompt.md)

## Hard Precondition: Host Filesystem Rebuild With ZFS

The current host is not suitable in its present EXT4-based form for the intended backup-first, snapshot-capable operating model.

Before any destructive cutover or appliance rebuild work begins, the host must be rebuilt with ZFS as the storage foundation.

This is not an optional enhancement. It is a hard precondition for the rest of this plan.

### Rationale

The target architecture requires:

- real restore points
- fast rollback capability
- consistent system-wide recovery semantics
- alignment with a backup-first, managed-pet runtime model

EXT4 does not satisfy those requirements for this build.

### Scope

This is a host-level reset, not an in-place migration.

- do not attempt filesystem conversion
- do not layer fake snapshot behavior on top of EXT4
- treat the current host as a source of data to extract, not a base to evolve

### Required Outcome

The rebuilt host must provide:

- a ZFS-backed storage pool
- dataset-backed mounts for all Moltbox-critical paths
- verified snapshot and rollback capability before proceeding

### Minimum Dataset Coverage

At minimum, these paths must reside on ZFS-backed datasets:

- `/srv/moltbox-state`
- `/srv/moltbox-logs`
- `/var/lib/moltbox`

No required runtime, state, or secret path may depend on EXT4 or another non-snapshot-capable filesystem in the final appliance.

### Relationship To The Rest Of This Plan

- this precondition precedes Workstream 0
- all existing workstreams remain valid after the rebuild
- the "current baseline host" facts below should be read as pre-rebuild extraction state, not target environment

## Goal

Build and validate a clean Moltbox appliance with:

- `gateway`
- `caddy`
- `ollama`
- `openclaw-test`
- `openclaw-prod`

and nothing else.

The finished box must satisfy all of these:

- every target container is running and healthy
- `moltbox` is the normal deployment and management path
- `test` and `prod` can both answer chat through the CLI
- ordinary chat uses local Ollama with `mistral:7b-instruct-32k`
- Together escalation remains available through official OpenClaw plugin/provider mechanisms
- no runtime state, repo checkout, secret, or service mount is owned by a personal login account
- the primary gateway-facing access name is `moltbox-gateway`
- metal admin access on `moltbox-prime` is preserved
- backups, restore workflow, SSH identities, and OS patching are in place

## Locked Decisions

These are not open questions for this execution pass:

- remove appliance `dev`
- remove OpenSearch
- keep `mistral:7b-instruct-32k` as the local model baseline
- keep `contextTokens=32768` unless testing proves a change is required
- keep `test` and `prod` as separate OpenClaw runtimes
- treat `prod` as a managed pet
- use full SSH access only to bootstrap and repair the box
- end with service deployment and validation performed through the CLI
- no service or path may rely on `jpekovitch`, `jpkovic`, or any other personal home directory for runtime operation
- do not do a deep Go-internal cleanup until the lighter CLI and appliance flow are working end to end

## Current Baseline Facts

Observed from the live host on 2026-04-04:

- SSH alias `moltbox` reaches host `moltbox-prime`
- current login lands as personal user `jpekovitch`
- current root filesystem is EXT4 on `/dev/nvme1n1p2`
- `/dev/sda` is a 12.7T EXT4 disk already mounted at `/mnt/moltbox-backup`
- `/dev/nvme0n1` is a separate 1.8T EXT4 filesystem currently not mounted
- `/dev/nvme1n1p3` is a separate 1T EXT4 filesystem currently not mounted
- running containers are `gateway`, `caddy`, `ollama`, `openclaw-dev`, `openclaw-test`, `openclaw-prod`, and `opensearch`
- all current containers are healthy
- current containers have no explicit Docker memory or CPU limits
- current Ollama inventory includes `mistral:7b-instruct-32k`
- current Ollama inventory does not include the exact tag `gemma3:4b-it`
- current `test` and `prod` chat baseline is `ollama/mistral:7b-instruct-32k`
- current `test` and `prod` `contextTokens` is `32768`
- current `test` and `prod` CLI chat works through `moltbox <env> openclaw agent --agent main ... --json`
- current chat succeeds after a gateway pairing failure by falling back to embedded execution
- current runtimes warn that `plugins.allow` is empty and `behavior-guard` is loading as untracked local code
- `/srv/moltbox-state`, `/srv/moltbox-logs`, `/var/lib/moltbox/secrets`, and `/mnt/moltbox-backup` are still owned by `jpekovitch`

Those facts are the migration baseline. They are not the target architecture.

## What This Plan Produces

The builder must leave behind:

- code changes in `moltbox-gateway`
- required sibling-repo changes in `moltbox-services`, `moltbox-runtime`, and, if needed, `remram-skills`
- builder-facing docs in this repo
- host provisioning changes for SSH identities, ownership, backup, and patching
- validation scripts or command files that prove the final state
- a concise evidence bundle showing the box passed the validation suite

## What This Plan Intentionally Defers

Do not expand scope into these during the initial cut unless they are blocking:

- broad refactors of `internal/orchestrator/runtime_state.go`
- redesign of gateway internals beyond the surfaces needed for the lighter CLI
- non-essential visual/UI work
- Postgres or Cortex integration
- extra model experiments
- adding new appliance-side runtimes beyond `test` and `prod`

## Execution Rules For The Builder

The builder must follow these rules:

- take backups before destructive changes
- capture live facts before changing the box
- commit the public CLI contract first, then simplify internals behind it
- preserve working behavior where possible, but do not preserve `dev`, OpenSearch, or personal-account ownership
- keep manual human steps to the absolute minimum
- if manual Linux commands are unavoidable, use them only to restore remote SSH control as fast as possible
- once remote SSH to the rebuilt host is working, the AI should take over the remaining IT and appliance work
- use [`../runbooks/2026-04-04-zfs-rebuild-ssh-takeover-runbook.md`](../runbooks/2026-04-04-zfs-rebuild-ssh-takeover-runbook.md) as the approved human path for the rebuild-to-SSH portion
- do not declare success because a unit test passed
- do not stop at "containers run"; finish only after the full validation plan passes
- if a step fails, record the failure, fix it, and continue
- if an input is genuinely missing, stop and report the exact missing input rather than guessing

## Required Inputs Before Execution

Resolved execution inputs for this pass:

- first look for existing SSH public keys in the normal local workstation SSH path
- first try to recover the Together API key from the current runtime appliance or existing configured secret/context paths
- primary backup target for this pass is the 14 TB spinning disk on `moltbox-prime`
- preferred restore model is real machine-level snapshots or restore points backed by ZFS
- gateway-facing primary access name should become `moltbox-gateway`
- preserve metal access on `moltbox-prime`
- preserve the operator's existing sudo-capable human admin path

These become hard blockers only if discovery fails at execution time:

- no usable public keys can be found for `moltbox-ai-test`, `moltbox-ai-prod`, or the break-glass admin account
- the Together key cannot be recovered or validated for the needed scopes
- the 14 TB backup disk cannot be identified, mounted, or used
- a true ZFS-backed snapshot facility cannot be established for the rebuilt host

If one of those failures occurs, stop at that point and ask for the missing input explicitly.

## Workstream -1: Host Extraction And ZFS Rebuild

### Objective

Extract recoverable state from the current EXT4 host, rebuild the machine on ZFS, and verify snapshot capability before any appliance rebuild work begins.

### Rules

- do not attempt in-place filesystem conversion
- do not proceed with appliance rebuild work on the current EXT4 root
- do not treat appliance backups alone as satisfying the snapshot requirement
- optimize the rebuild flow for the earliest possible recovery of remote SSH access
- any manual console or installer interaction should stop as soon as the rebuilt host can be reached over SSH

### Extraction Tasks On The Current Host

1. Capture the current storage layout and mount state.
2. Preserve these paths to the backup target:
   - `/srv/moltbox-state`
   - `/srv/moltbox-logs`
   - `/var/lib/moltbox/secrets`
   - `/etc/moltbox`
3. Create and verify native OpenClaw backups for both `test` and `prod`.
4. Preserve current rendered configs and runtime metadata needed for rebuild validation.
5. Preserve any required host-side CLI or gateway config for reference only, not as a promise of direct reuse.

### Rebuild Tasks

1. Reinstall or rebuild the host with ZFS as the storage foundation.
2. Create a single ZFS pool if one pool is sufficient for the build.
3. Create dataset-backed mounts at minimum for:
   - `/srv/moltbox-state`
   - `/srv/moltbox-logs`
   - `/var/lib/moltbox`
4. Recreate the target host directory layout on ZFS-backed datasets.
5. Verify the rebuilt host still preserves direct human admin access on `moltbox-prime`.
6. Restore SSH service and authorized key access as early as possible so the AI can take over remotely.

### Manual-Step Budget

If manual steps are required, they should be limited to the fastest path to a remotely reachable rebuilt host:

- boot installer or recovery environment
- complete the ZFS-backed base install
- bring up networking
- ensure the preserved human admin account or equivalent sudo-capable admin path exists
- ensure SSH is enabled and reachable
- ensure the builder can authenticate over SSH

Use [`../runbooks/2026-04-04-zfs-rebuild-ssh-takeover-runbook.md`](../runbooks/2026-04-04-zfs-rebuild-ssh-takeover-runbook.md) for the explicit operator steps.

Everything after that should be driven remotely.

### Snapshot Verification Tasks

1. Create a test ZFS snapshot covering the Moltbox datasets.
2. Make a reversible filesystem change inside a covered dataset.
3. Roll back the snapshot or otherwise prove the rollback path.
4. Record the exact snapshot and rollback commands used.

### Stop Condition

Do not start Workstream 0 until:

- the old host state is backed up
- the rebuilt host is on ZFS
- the minimum Moltbox-critical paths are on ZFS-backed datasets
- snapshot and rollback capability have been verified

### Exit Criteria

- current EXT4 host is treated only as extraction state
- rebuilt host is ZFS-backed
- minimum dataset coverage is in place
- snapshot capability is proven, not assumed

## Workstream 0: Builder Prep And Safety Rails

### Objective

Make the execution traceable and reversible before any destructive change lands.

### Tasks

1. Create a dedicated working branch in each touched repo.
2. Record current revisions of:
   - `moltbox-gateway`
   - `moltbox-services`
   - `moltbox-runtime`
   - `remram-skills`
3. Create a working evidence directory, for example:
   - `artifacts/clean-moltbox/<timestamp>/`
4. Record the current live appliance facts:
   - `whoami`
   - `hostname`
   - `uname -a`
   - `docker ps`
   - `docker inspect` for all live containers
   - `docker network ls`
   - `docker volume ls`
   - `docker exec ollama ollama list`
   - `moltbox gateway status`
   - `moltbox test openclaw status`
   - `moltbox prod openclaw status`
   - `moltbox test openclaw plugins list`
   - `moltbox prod openclaw plugins list`
   - `moltbox test openclaw skills list`
   - `moltbox prod openclaw skills list`
   - `moltbox test openclaw agent --agent main --message "Reply with TEST_OK only." --json`
   - `moltbox prod openclaw agent --agent main --message "Reply with PROD_OK only." --json`
5. Save live copies of:
   - current `openclaw.json`
   - current `model-runtime.yml`
   - current rendered service configs under `/srv/moltbox-state/services/`
6. Discover the current workstation SSH key material under the normal local SSH path before asking for new keys.
7. Discover the current Together credential from the live appliance or configured secret/context paths before asking for it again.

### Exit Criteria

- current state is captured in an evidence directory
- the builder can prove what changed later

## Workstream 1: Backup And Recovery Prep

### Objective

Create a recovery path before wiping the current EXT4 host and before rebuilding the appliance on ZFS.

### Tasks

1. Create official OpenClaw backups for both runtimes:
   - `openclaw backup create --verify`
2. Copy the resulting archives to `/mnt/moltbox-backup` and into the evidence bundle metadata.
3. Discover the 14 TB spinning disk on `moltbox-prime`.
4. If the disk is not already mounted:
   - identify it safely
   - mount it under a system-owned path
   - record the mount configuration used for backup jobs
5. Back up these host paths:
   - `/srv/moltbox-state`
   - `/srv/moltbox-logs`
   - `/var/lib/moltbox/secrets`
   - `/etc/moltbox`
   - `/usr/local/bin/moltbox*`
6. Preserve the extracted host backups on the 14 TB disk before the filesystem rebuild begins.
7. Record the absence of real snapshot capability on the current EXT4 host as the reason for the ZFS rebuild.
8. Record backup locations and timestamps.

### Stop Condition

Do not wipe or rebuild the host until at least one verified OpenClaw backup per runtime exists and host-state backup is complete on the backup disk.

### Exit Criteria

- verified runtime backups exist
- host-state backup exists
- the 14 TB disk is discovered and configured
- restore path is documented in the evidence bundle

## Workstream 2: Repo Cleanup And Builder Package Cleanup

### Objective

Make this repo navigable for the AI team and stop forcing them through stale plans and legacy assumptions.

### Tasks In `moltbox-gateway`

1. Add a top-level archive guide that says what is historical and what is still useful.
2. Make the new execution plan, validation plan, and kickoff prompt the primary docs linked from the root `README.md`.
3. Mark the earlier lightweight plan and test plan as scoping documents, not the primary execution package.
4. Do not delete the archive contents in this pass.
5. Do not port legacy Python code.

### Exit Criteria

- a fresh builder landing in the repo can find the execution package in under one minute
- archive material is clearly labeled as reference-only

## Workstream 3: Public CLI Contract Refactor In `moltbox-gateway`

### Objective

Replace the heavy runtime-oriented CLI with the lightweight service-plane-plus-OpenClaw surface.

### Files To Change

- `pkg/cli/cli.go`
- `cmd/moltbox/main.go`
- `internal/client/http.go`
- `internal/gateway/http.go`
- `internal/sshwrap/sshwrap.go`
- `internal/config/config.go`
- `scripts/provision-automation-ssh.sh`
- tests under:
  - `internal/gateway/`
  - `internal/sshwrap/`
  - `internal/config/`
  - `pkg/cli/` if new parser tests are added

### Required CLI Contract

```text
moltbox
  bootstrap gateway
  gateway status|logs|update|mcp-stdio
  service list|status|deploy|restart|logs <service>
  test openclaw <native args>
  prod openclaw <native args>
  ollama <native args>
  secret set|list|delete <scope>
```

### Commands To Remove From The Public Contract

- `dev`
- `opensearch`
- `checkpoint`
- `skill ...`
- `plugin ...`
- `gateway docker ...`
- `gateway service ...`
- `<env> secrets ...`
- `service secrets ...`

### Detailed Tasks

1. Rewrite parser help and route resolution in `pkg/cli/cli.go`.
2. Add a top-level `service list` route if it does not exist.
3. Add a top-level `service logs <service>` route if it does not exist.
4. Move service lifecycle grammar out of `gateway service ...`.
5. Move secrets grammar to top-level `secret ...`.
6. Keep `gateway mcp-stdio`, but do not grow that surface.
7. Update `cmd/moltbox/main.go` so local bootstrap behavior is limited to `bootstrap gateway` and local MCP stdio.
8. Update `internal/client/http.go` routing to match the new surface.
9. Update `internal/gateway/http.go` endpoints to support the new surface.
10. Remove or hide obsolete runtime mutation endpoints from the public contract.
11. Keep internal compatibility only if it reduces migration risk, and do not expose it in help text.

### Test Requirements

- parser tests for every allowed command
- parser tests proving every removed command fails explicitly
- route mapping tests for service, runtime, ollama, and secret paths
- gateway handler tests for new list/logs/deploy/restart/status behavior

### Exit Criteria

- `moltbox --help` matches the new contract
- removed verbs fail explicitly
- no public parser surface mentions `dev`, `opensearch`, `checkpoint`, `skill`, or `plugin`

## Workstream 4: Gateway Internals And Orchestrator Simplification

### Objective

Make the gateway capable of running the new contract without forcing a deep rewrite first.

### Files To Inspect And Change Carefully

- `internal/orchestrator/manager.go`
- `internal/orchestrator/runtime_state.go`
- `internal/docker/client.go`
- `internal/localexec/runner.go`

### Rules

- do not rip out old runtime-state code just because it exists
- remove or bypass it only where it is directly on the path of the new contract
- prefer shrinking public entry points first

### Detailed Tasks

1. Implement or expose a real service inventory path for:
   - `gateway`
   - `caddy`
   - `ollama`
   - `test`
   - `prod`
2. Implement service logs retrieval by service name.
3. Keep gateway update working, but remove any ownership behavior that keys off a personal-user install location.
4. Preserve existing deploy/restart behavior until the lighter templates are ready.
5. Demote replay/checkpoint code to legacy/internal-only status.
6. Do not keep plugin inventory derived from replay metadata as the authoritative truth.

### Exit Criteria

- the gateway can deploy, restart, inspect, and log the target service plane without using removed CLI verbs

## Workstream 5: `moltbox-services` Baseline Simplification

### Objective

Strip the appliance down to the final service set.

### Required Changes

1. Delete:
   - `services/openclaw-dev`
   - `services/opensearch`
2. Keep only:
   - `services/gateway`
   - `services/caddy`
   - `services/ollama`
   - `services/openclaw-test`
   - `services/openclaw-prod`
3. Remove OpenSearch wiring from OpenClaw service templates:
   - `OPENSEARCH_URL`
   - `opensearch.yml`
4. Keep service healthchecks.
5. Keep service names and container naming consistent with the target CLI surface.
6. Update Caddy config assumptions to only:
   - `moltbox-test`
   - `moltbox-prod`
7. Make the primary gateway-facing access name `moltbox-gateway` while preserving metal-host admin access on `moltbox-prime`

### Resource Policy

Do not invent new CPU or RAM limits in the first cut just to make the templates look clean. The live box currently runs without container limits and the local model balance is known-good on that posture.

If limits are introduced, they must be:

- documented
- justified by testing
- applied consistently

### Exit Criteria

- a rendered clean appliance contains only the five target services
- no service template references OpenSearch or `openclaw-dev`

## Workstream 6: `moltbox-runtime` Baseline Simplification

### Objective

Make `moltbox-runtime` the clean desired-state baseline for `test` and `prod`.

### Required Changes

1. Delete:
   - `openclaw-dev`
   - `opensearch`
2. Remove OpenSearch-backed tools and config.
3. Keep `mistral:7b-instruct-32k` as:
   - `agents.defaults.model.primary`
   - local routing model
4. Keep `contextTokens=32768` unless validation forces change.
5. Make the Together path official and explicit through OpenClaw plugin/provider surfaces.
6. Set `plugins.allow` explicitly.
7. Resolve the current untracked `behavior-guard` state by doing one of:
   - package and install it as a tracked plugin
   - remove it
   - replace it with an approved tracked equivalent

### Together Escalation Rule

The gateway must not own model-routing logic.

The builder must prove one of these is true:

- stock OpenClaw provider/plugin wiring is sufficient, so no custom Remram plugin is needed
- a dedicated plugin-backed Remram capability is required, and it is implemented as a plugin rather than hidden gateway behavior

Documentation skill content alone does not satisfy this requirement.

### Exit Criteria

- `test` and `prod` baseline configs are clean
- local-first chat is explicitly Mistral
- Together escalation is not hard-coded in the gateway
- plugin trust is explicit

## Workstream 7: SSH, Ownership, And Host Hygiene

### Objective

Make the appliance system-owned and stop relying on personal accounts.

### Required Host Layout

- `/usr/local/bin/moltbox`
- `/etc/moltbox/config.yaml`
- `/usr/local/bin/moltbox-cli-wrapper`
- `/usr/local/bin/moltbox-bootstrap-wrapper`
- `/opt/moltbox/repos/moltbox-gateway`
- `/opt/moltbox/repos/moltbox-services`
- `/opt/moltbox/repos/moltbox-runtime`
- `/opt/moltbox/repos/remram-skills`
- `/srv/moltbox-state`
- `/srv/moltbox-logs`
- `/var/lib/moltbox/secrets`
- `/var/lib/moltbox/history.jsonl`

### Required Identity Model

- preserve the operator's existing human admin SSH and sudo path
- create `moltbox-ai-test`
- create `moltbox-ai-prod`
- create one break-glass admin account with a distinct key
- do not create a second UID 0 account

### Required Wrapper Policy

- `moltbox-ai-test`
  - forced-command only
  - may run `moltbox ...`
  - may mutate test appliance service plane
  - may not escape to arbitrary shell
- `moltbox-ai-prod`
  - forced-command only
  - may run `moltbox ...`
  - may inspect `prod`
  - may not mutate `prod`, service state, or host state

### Detailed Tasks

1. Remove user-home config lookup from production code.
2. Remove user-home CLI install assumptions from docs and scripts.
3. Replace personal-name automation identities in provisioning scripts.
4. Recursively audit `/srv`, `/var/lib/moltbox`, `/opt/moltbox`, and `/etc/moltbox` for personal ownership.
5. Fix ownership and permissions before cutover.
6. Preserve direct human admin access on `moltbox-prime` rather than trying to eliminate it.

### Exit Criteria

- nothing required for runtime operation lives under `/home/*`
- no service/state/secret path is owned by a personal account
- SSH wrapper tests prove the new identity policy

## Workstream 8: Backup, Restore, And OS Patching

### Objective

Leave the box with basic real-world IT hygiene rather than a purely application-local story.

### Required Deliverables

1. A nightly backup job for appliance-critical state.
2. A documented pre-change ZFS snapshot workflow.
3. Verified `openclaw backup create --verify` for both runtimes.
4. OS patching configured through unattended upgrades or an equivalent timer.
5. Evidence of last patch run and last backup run.
6. Backup target on the 14 TB disk configured under a system-owned path.

### Minimum Backup Scope

- `/srv/moltbox-state`
- `/srv/moltbox-logs` if operationally important
- `/var/lib/moltbox/secrets`
- `/etc/moltbox`
- OpenClaw backup archives
- Caddy data and config

Do not store backup material in Git.

### Restore Policy

- preferred: ZFS snapshot rollback or machine image restore from the rebuilt ZFS host
- secondary: restore host-state backup, then restore OpenClaw backups
- clean rebuild from repos is emergency-only fallback

If real ZFS-backed snapshot capability is absent after the rebuild, stop and treat that as a deviation from plan rather than pretending the operating model is satisfied.

### Exit Criteria

- backup and patch jobs exist
- restore instructions are testable
- backup evidence is captured
- ZFS snapshot availability is validated

## Workstream 9: Host Cutover And Appliance Rebuild

### Objective

Take the rebuilt ZFS host and bring it up as the clean target appliance.

### Detailed Sequence

1. Confirm Workstreams 0 and 1 are done.
2. Confirm backup and recovery prep is done.
3. Confirm Workstream -1 completed and the host is now ZFS-backed.
4. Update host OS packages.
5. Update Docker and NVIDIA toolkit if needed.
6. Reboot the host if kernel, Docker, or driver changes require it.
7. Recreate the target host directory layout on ZFS-backed datasets.
8. Install or update the host `moltbox` binary and config into system-owned paths.
9. Bootstrap the gateway only.
10. Deploy services through the CLI in this order:
    - `gateway` if bootstrap did not already place it
    - `ollama`
    - `openclaw-test`
    - `openclaw-prod`
    - `caddy`
11. Pull or verify `mistral:7b-instruct-32k` through the CLI.
12. Restore or re-seed runtime state as needed:
    - restore `prod` from native backup if preserving state
    - decide whether `test` starts clean or restores selected state
13. Apply or verify Together credentials through scoped secrets.
14. Verify Caddy ingress and naming for:
    - `moltbox-gateway`
    - `moltbox-test`
    - `moltbox-prod`
    while preserving direct metal-host admin access on `moltbox-prime`
15. Run the full validation suite.

### Stop Conditions

Stop and report if any of these occur:

- verified runtime backup is missing
- the builder does not have the credentials needed to restore or test the runtime
- Workstream -1 did not actually produce the required ZFS-backed host
- gateway bootstrap fails and no known-good gateway image exists
- local Mistral inference fails after service deployment
- Together escalation cannot be configured through OpenClaw plugin/provider surfaces

### Exit Criteria

- only the target services are running
- `test` and `prod` are reachable and chat-capable
- the builder can proceed to the validation suite without manual Docker surgery

## Workstream 10: Validation, Evidence, And Close-Out

### Objective

Prove the box is actually done.

### Required Validation Gates

1. `go test ./...` in `moltbox-gateway`
2. any required sibling-repo tests for touched runtime/service files
3. validation scripts defined in [`./2026-04-04-clean-moltbox-validation-plan.md`](./2026-04-04-clean-moltbox-validation-plan.md)
4. full end-to-end acceptance on the live host

### Required Evidence Bundle

The builder must leave:

- command transcript or summarized output for each validation script
- `docker ps` from the final state
- `moltbox service list`
- `moltbox gateway status`
- final `moltbox test openclaw agent --agent main ... --json`
- final `moltbox prod openclaw agent --agent main ... --json`
- final evidence for Together escalation
- backup and patching evidence
- ownership audit evidence

### Completion Standard

The task is not complete until the final validation gate passes.

## Sequence Summary

Recommended high-level order:

1. extract and back up the current EXT4 host
2. rebuild the host on ZFS and verify snapshot rollback
3. prep and evidence capture on the rebuilt host
4. repo cleanup and builder package cleanup
5. gateway CLI/public surface refactor
6. service/runtime baseline simplification
7. SSH/ownership/host hygiene
8. backup and patching implementation
9. appliance rebuild and service deployment
10. validation and evidence bundle

## Non-Negotiable Acceptance Conditions

Reject the run if any of these remain true:

- Moltbox-critical paths are not on ZFS-backed datasets
- `dev` still exists on the appliance
- OpenSearch still exists on the appliance
- ordinary chat is not local-first on `mistral:7b-instruct-32k`
- CLI chat only works because of hidden embedded fallback after gateway failure
- Together escalation depends on gateway-specific hard-coded routing
- service paths or secrets are owned by a personal user
- the AI identities can escape to arbitrary shell
- backups and patching are still "to do later"
