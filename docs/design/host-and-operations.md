# Host And Operations

This document defines the host baseline, storage layout, identity model, and operational hygiene requirements.

## Host Baseline

Target host assumptions:

- Ubuntu Server 24.04 LTS family
- Docker and required runtime dependencies installed
- ZFS-backed storage for Moltbox-critical state
- preserved human admin access on `moltbox-prime`

The current EXT4 host is not an acceptable final baseline.

## Required Storage Model

At minimum, the final host must provide ZFS-backed datasets for:

- `/srv/moltbox-state`
- `/srv/moltbox-logs`
- `/var/lib/moltbox`

A single pool is sufficient if it performs well and is operationally simple.

## Required Host Paths

| Path | Purpose |
| --- | --- |
| `/usr/local/bin/moltbox` | host CLI entrypoint |
| `/etc/moltbox/config.yaml` | host CLI config |
| `/opt/moltbox/repos/moltbox-gateway` | gateway repo checkout |
| `/opt/moltbox/repos/moltbox-services` | service repo checkout |
| `/opt/moltbox/repos/moltbox-runtime` | runtime repo checkout |
| `/opt/moltbox/repos/remram-skills` | skill and plugin source checkout |
| `/srv/moltbox-state` | appliance state |
| `/srv/moltbox-logs` | appliance logs |
| `/var/lib/moltbox/secrets` | encrypted secret material |
| `/var/lib/moltbox/history.jsonl` | host-level gateway history |
| `/mnt/moltbox-backup` | local backup target disk |

## Ownership Rules

Required paths must be:

- system-owned
- stable across reboots
- outside `/home/*`

Personal users may administer the box, but runtime operation must not depend on a personal account owning state.

## Identity Model

### Human admin

Keep the operator's sudo-capable human access.

That account exists for:

- emergency intervention
- high-trust human administration
- rebuild and bootstrap assistance

It is not the runtime ownership model.

### AI test identity

Create `moltbox-ai-test` with:

- forced-command wrapper
- access to `moltbox` CLI
- broad test-side mutation rights
- no arbitrary shell

### AI prod identity

Create `moltbox-ai-prod` with:

- forced-command wrapper
- access to `moltbox` CLI
- inspection and debugging rights
- no arbitrary shell
- no prod mutation rights

### Break-glass admin

Create one separate break-glass admin account with its own key.

Do not create a second root account.

## Networking And Naming

Names to preserve:

- metal host: `moltbox-prime`
- gateway ingress: `moltbox-gateway`
- runtime ingress: `moltbox-test`
- runtime ingress: `moltbox-prod`

The metal-host admin path must remain available even when Caddy or a runtime is unhealthy.

## Patching

The host must leave this rebuild with patching configured.

Minimum acceptable posture:

- unattended upgrades or equivalent timer
- visible evidence of last patch run
- documented reboot policy

## Manual Operations Rule

Manual human Linux work should be minimized to:

- backup extraction
- reinstall or rebuild steps
- fastest path back to SSH

After SSH works, the AI should take over the remaining operations.

Use:

- `../runbooks/2026-04-04-zfs-rebuild-ssh-takeover-runbook.md`

for the bounded manual path.

## Related Documents

- [Backup And Recovery](backup-and-recovery.md)
- [Delivery And Migration](delivery-and-migration.md)
