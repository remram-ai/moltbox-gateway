# Clean Moltbox Builder Prompt

Use this prompt to kick off the implementation loop.

## Prompt

You are implementing the clean Moltbox rebuild end to end.

Work in these repos:

- `moltbox-gateway`
- `moltbox-services`
- `moltbox-runtime`
- `remram-skills` only if needed for a real plugin-backed escalation capability
- update `remram` docs only after the implementation is stable

Authoritative docs for this run:

- `moltbox-gateway/docs/runbooks/2026-04-04-zfs-rebuild-ssh-takeover-runbook.md`
- `moltbox-gateway/docs/plans/2026-04-04-clean-moltbox-execution-plan.md`
- `moltbox-gateway/docs/plans/2026-04-04-clean-moltbox-validation-plan.md`
- `moltbox-gateway/docs/decisions/2026-04-04-openclaw-operating-model.md`
- `moltbox-gateway/docs/reviews/2026-04-04-cli-surface-review.md`
- `moltbox-gateway/archive/README.md`

Objective:

Take the current mixed appliance and deliver a clean Moltbox box with exactly:

- `gateway`
- `caddy`
- `ollama`
- `openclaw-test`
- `openclaw-prod`

Required end state:

- the host has been rebuilt on ZFS before appliance rebuild work proceeds
- Moltbox-critical paths are on ZFS-backed datasets
- snapshot and rollback capability are verified
- the public CLI surface is the new lightweight contract
- `dev` is gone
- OpenSearch is gone
- `test` and `prod` can both answer chat through `moltbox <env> openclaw agent --agent main ... --json`
- ordinary chat uses local Ollama with `mistral:7b-instruct-32k`
- Together escalation remains available through official OpenClaw plugin/provider behavior
- the gateway does not hard-code routing or escalation logic
- all runtime/state/config paths are system-owned
- AI SSH identities are in place with forced-command policy
- backup, restore, and OS patching are configured
- the final appliance is deployed and validated through the CLI

Known live starting facts:

- live host is `moltbox-prime`
- current SSH lands as personal user `jpekovitch`
- current root host filesystem is EXT4 and must be treated as extraction state only
- the 14 TB local backup disk is present and mounted at `/mnt/moltbox-backup`
- current containers are `gateway`, `caddy`, `ollama`, `openclaw-dev`, `openclaw-test`, `openclaw-prod`, and `opensearch`
- current local model already works on `mistral:7b-instruct-32k`
- current `test` and `prod` chat through the CLI, but success currently depends on embedded fallback after gateway pairing failure
- current plugin trust is weak because `plugins.allow` is empty and `behavior-guard` is untracked

Rules:

- use full SSH access only to bootstrap, inspect, back up, and repair the host
- keep manual human steps to the absolute minimum
- if manual Linux steps are required, use them only for the fastest path to restored remote SSH access
- use the runbook as the bounded operator path for that manual portion
- once remote SSH to the rebuilt ZFS host works, the AI takes over the remaining IT and appliance work
- end with the CLI as the deployment and validation path
- take backups before destructive changes
- capture live baseline evidence before changing anything
- do not preserve `dev`, OpenSearch, personal ownership, or replay-heavy runtime management
- do not do a large Go cleanup unless it directly serves the target flow
- do not stop at planning
- do not stop when code compiles
- continue until the full validation plan passes or you hit a real blocker

Implementation sequence:

1. capture current live state and save evidence
2. create verified OpenClaw backups and host-state backups on `/mnt/moltbox-backup`
3. rebuild the host on ZFS instead of evolving the current EXT4 host in place
4. restore remote SSH access as early as possible and hand control back to the AI
5. clean up the repo entry points so the execution package is primary
6. implement the new CLI contract in `moltbox-gateway`
7. simplify gateway handlers and orchestrator paths enough to support that contract
8. remove `openclaw-dev` and OpenSearch from `moltbox-services`
9. remove `openclaw-dev` and OpenSearch from `moltbox-runtime`
10. keep `mistral:7b-instruct-32k` and `contextTokens=32768` unless tests force change
11. make Together escalation official through plugin/provider surfaces rather than gateway hard-coding
12. make plugin trust explicit and resolve the untracked `behavior-guard` state
13. implement system-owned host layout and AI SSH identities
14. implement backup and patching jobs
15. rebuild the clean appliance on the ZFS host
16. validate with the full suite
17. leave an evidence bundle showing the build passed

Specific code expectations:

- `moltbox-gateway/pkg/cli/cli.go` must match the new CLI contract
- `moltbox-gateway/internal/client/http.go` and `internal/gateway/http.go` must support the new routes
- `moltbox-gateway/internal/sshwrap/sshwrap.go` must enforce the new AI identity model
- `moltbox-gateway/internal/config/config.go` must stop depending on user-home config in production
- `moltbox-services` must render only the final service set
- `moltbox-runtime` must render only clean `test` and `prod` baselines

Validation requirements:

- add or update validation scripts so the suite is repeatable
- prove the ZFS precondition and snapshot rollback before appliance acceptance
- run `go test ./...`
- run the CLI contract checks
- run service-plane deployment checks
- run end-to-end chat checks for `test` and `prod`
- run Together escalation checks
- run SSH policy checks
- run host ownership/path checks
- run backup and patching checks

Completion criteria:

- the rebuilt host is on ZFS and snapshot rollback is proven
- the full validation plan passes
- all target containers are healthy
- ordinary chat reports `provider=ollama` and `model=mistral:7b-instruct-32k`
- Together escalation is observable without gateway hard-coding
- no required path is owned by a personal account
- the final state is reproducible through the CLI and documented host workflows

If blocked, stop only for one of these:

- missing SSH keys for the new identities
- missing Together credentials
- inability to establish the required ZFS-backed snapshot facility
- inability to create any verified runtime backup
- inability to restore or validate the box without risking data loss

When you finish, report:

- files changed
- repos changed
- commands run
- validation results
- remaining risks if any
