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

AI test operator:

- forced-command SSH identity
- may use the CLI for `test`
- may mutate `test`, `ollama`, and `searxng`
- no arbitrary shell

AI prod operator:

- forced-command SSH identity
- may use the CLI for prod diagnostics
- no prod mutation
- no arbitrary shell

Break-glass admin:

- elevated emergency access

## Normal Operations Rule

Normal work should happen through:

- the host-installed `moltbox` CLI
- the service plane
- native `openclaw` commands through the CLI wrappers

Break-glass SSH is for recovery, debugging, and research.
