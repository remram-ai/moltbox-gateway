# Implementation Plan

Use this file when the task is execution-oriented.

High-level sequence:

1. capture the live box and preserve backups
2. rebuild the host on ZFS
3. restore SSH as fast as possible
4. implement the lighter CLI and gateway contract
5. remove `openclaw-dev` and OpenSearch from the service and runtime repos
6. fix ownership, SSH identities, backup, and patching
7. deploy the final five-service appliance
8. run the full validation suite

Execution package:

- `../plans/2026-04-04-clean-moltbox-execution-plan.md`
- `../plans/2026-04-04-clean-moltbox-validation-plan.md`
- `../plans/2026-04-04-clean-moltbox-builder-prompt.md`
- `../runbooks/2026-04-04-zfs-rebuild-ssh-takeover-runbook.md`

Completion is defined by the validation plan, not by code changes alone.

Canonical design bridge:

- `../design/delivery-and-migration.md`
