# Host And Operations

This document defines host layout, storage posture, and operator roles.

## Host Layout

Required host paths:

- `/usr/local/bin/moltbox`
- `/etc/moltbox/config.yaml`
- `/opt/moltbox/repos/moltbox-gateway`
- `/opt/moltbox/repos/moltbox-services`
- `/opt/moltbox/repos/moltbox-runtime`
- `/srv/moltbox-state`
- `/srv/moltbox-logs`
- `/var/lib/moltbox/secrets`
- `/mnt/moltbox-backup`

## Storage Posture

Critical appliance paths must be on ZFS:

- `/srv/moltbox-state`
- `/srv/moltbox-logs`
- `/var/lib/moltbox`
- `/opt/moltbox/repos`

The base OS may remain on `ext4`. The appliance-critical state boundary may not.

## Ownership Rules

Required paths must be:

- system-owned
- outside `/home/*`
- stable across reboots

## Operator Roles

Human admin:

- preserved trusted admin path
- current host account: `jpekovitch`
- current workstation alias: `moltbox` or `moltbox-admin`

AI test operator:

- forced-command SSH identity
- current host account: `moltbox-ai-test`
- current workstation alias: `moltbox-ai-test`
- may use the CLI for `test`
- may mutate `test`, `ollama`, and `searxng`
- no arbitrary shell

AI prod operator:

- forced-command SSH identity
- current host account: `moltbox-ai-prod`
- current workstation alias: `moltbox-ai-prod`
- may use the CLI for prod diagnostics
- no prod mutation
- no arbitrary shell

Break-glass admin:

- elevated emergency access
- current host account: `moltbox-breakglass`
- current workstation alias: `moltbox-breakglass`

## Current Key Discovery

Current workstation SSH aliases resolve through `C:\Users\Jason\.ssh\config`.

Current live host key state:

- `C:\Users\Jason\.ssh\id_ed25519` is currently installed on:
  - `jpekovitch`
  - `moltbox-ai-test`
  - `moltbox-ai-prod`
  - `moltbox-breakglass`
- the legacy local key names `jason-codex` and `codex-bootstrap` still exist on the workstation, but they are not the current installed host keys

Legacy-name rule:

- treat `jason-codex` and `codex-bootstrap` as historical names only
- use the role-based account names and SSH aliases above for current operations and future documentation

## Normal Operations Rule

Normal work should happen through:

- the host-installed `moltbox` CLI
- the service plane
- native `openclaw` commands through the CLI wrappers

Break-glass SSH is for recovery, debugging, and research.
