# Host And Operations Context

Use this file when the task is mostly about storage, identities, permissions, backups, patching, or rebuild procedure.

Required host posture:

- Ubuntu Server baseline
- ZFS-backed critical paths
- system-owned state and secret paths
- preserved human admin access
- restricted AI SSH identities
- snapshot-first recovery

Required ZFS-backed paths:

- `/srv/moltbox-state`
- `/srv/moltbox-logs`
- `/var/lib/moltbox`

Required host paths:

- `/usr/local/bin/moltbox`
- `/etc/moltbox/config.yaml`
- `/opt/moltbox/repos/...`
- `/srv/moltbox-state`
- `/srv/moltbox-logs`
- `/var/lib/moltbox/secrets`
- `/mnt/moltbox-backup`

Identity model:

- human admin retains sudo
- `moltbox-ai-test` gets forced-command test-side access
- `moltbox-ai-prod` gets forced-command prod-inspection access
- one break-glass admin account exists

Backup model:

- ZFS snapshots are primary restore points
- appliance backups go to `/mnt/moltbox-backup`
- OpenClaw native backups remain required

Manual-ops rule:

- only do manual keyboard work to get the rebuilt host back to SSH
- then hand off to the AI

Canonical sources:

- `../design/host-and-operations.md`
- `../design/backup-and-recovery.md`
- `../runbooks/2026-04-04-zfs-rebuild-ssh-takeover-runbook.md`
