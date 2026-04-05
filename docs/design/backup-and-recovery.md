# Backup And Recovery

This document defines the actual recovery model for the appliance.

## Recovery Order

Primary restore-point mechanism:

- ZFS snapshot
- ZFS rollback

Secondary runtime backup layer:

- native OpenClaw backup create and verify

## What Is Proven

Proven:

- critical appliance paths are on ZFS
- service-plane mutations create snapshot metadata
- mutating native runtime commands create snapshot metadata
- ZFS rollback is proven on the live host
- native OpenClaw backup create and verify work

Not yet proven as a primary path:

- native OpenClaw restore end to end

## What Is Not The Model

- checkpoint is not backup
- replay is not the normal restore model
- clean rebuild from Git alone is not the primary recovery story

## Minimum Backup Scope

- `/srv/moltbox-state`
- `/srv/moltbox-logs` when needed
- `/var/lib/moltbox/secrets`
- `/etc/moltbox`
- OpenClaw native backup archives

Primary local backup target:

- `/mnt/moltbox-backup`

## Operational Rule

Before risky service or runtime changes:

1. take or rely on the automatic snapshot
2. make the change through the official CLI path
3. validate
4. roll back ZFS first if the change is bad
