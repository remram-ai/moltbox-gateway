# Backup And Recovery

This document defines the durability model for the target appliance.

## Recovery Principles

### Snapshots first

Machine-level or dataset-level snapshots are the preferred recovery path.

### Appliance backups second

Appliance backups exist to survive host loss, storage mistakes, and restore beyond a snapshot window.

### Native OpenClaw backup stays in the model

OpenClaw native backups remain part of the recovery stack for runtime-local state.

### Replay is not backup

Checkpoint and replay are not the durability model.

## Backup Classes

| Class | Scope | Purpose | Frequency |
| --- | --- | --- | --- |
| ZFS snapshot | Moltbox datasets | fast local restore point | before risky change and on schedule |
| appliance backup | host critical paths | host-loss recovery | nightly |
| OpenClaw backup | `test` and `prod` runtime state | runtime restore | nightly and before risky change |
| config and repo evidence | rendered config and revision evidence | rebuild and audit trace | per deploy |

## Minimum Backup Scope

Back up at least:

- `/srv/moltbox-state`
- `/srv/moltbox-logs` if operationally useful
- `/var/lib/moltbox/secrets`
- `/etc/moltbox`
- Caddy data and config
- OpenClaw native backup archives

Primary backup target for this pass:

- `/mnt/moltbox-backup`

That target must be system-owned and excluded from Git.

## Restore Model

### Case 1: bad change, host still alive

Preferred path:

1. use a recent ZFS snapshot or restore point
2. roll back the affected dataset or host state
3. validate services

### Case 2: runtime-specific recovery

Preferred path:

1. restore recent runtime backup
2. verify OpenClaw health
3. validate chat and provider behavior

### Case 3: host failure

Preferred path:

1. restore host from snapshot or rebuild base host
2. restore appliance-critical paths from backup
3. restore OpenClaw native backups
4. redeploy or validate the service plane through the CLI

The clean rebuild from Git repos alone is emergency fallback only.

## Snapshot Policy

Required minimum posture:

- take a restore point before risky `prod` changes
- keep routine scheduled snapshots
- verify rollback at least once before declaring the box done

## Backup Verification

Backups do not count unless they are verified.

Required proof:

- snapshot creation works
- snapshot rollback works
- OpenClaw backup creation works
- the nightly backup job actually writes output to the target disk

## What Checkpoint Is Not

Checkpoint does not replace:

- snapshots
- appliance backups
- OpenClaw native backups

Checkpoint is operational convenience at most. It is not the main durability story.

## Later Enhancements

Not required for the first clean-box pass:

- off-host backup replication
- Glacier or cloud retention
- automated cold-restore drills on a second host

Those are later hardening steps.

## Related Documents

- [Host And Operations](host-and-operations.md)
- [Target State](target-state.md)
