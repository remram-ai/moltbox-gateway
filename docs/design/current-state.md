# Current State

This document describes the live baseline that exists before the clean rebuild.

It is the migration source, not the target design.

## Live Host Facts

Observed on `2026-04-04`:

- host OS: Ubuntu 24.04.4 LTS
- host name: `moltbox-prime`
- current LAN IP: `192.168.1.189`
- root filesystem: EXT4 on `/dev/nvme1n1p2`
- backup disk: `/dev/sda`, 12.7T, mounted at `/mnt/moltbox-backup`
- unused local disks or filesystems also exist, but the current active root is still EXT4

That means the host does not yet satisfy the snapshot-first storage model.

## Live Service Plane

The current box is still running:

- `gateway`
- `caddy`
- `ollama`
- `openclaw-dev`
- `openclaw-test`
- `openclaw-prod`
- `opensearch`

All of those containers were healthy when inspected, but that does not make the architecture acceptable.

## Live Runtime Facts

The useful live facts are:

- `test` and `prod` currently answer chat
- local provider is `ollama`
- local model is `mistral:7b-instruct-32k`
- `contextTokens` is `32768`
- current CLI chat succeeds after a gateway pairing failure by falling back to embedded execution

That fallback path is a bug or drift signal, not a design goal.

## Live Ownership Problems

The current box still has personal-account ownership on critical paths, including:

- `/srv/moltbox-state`
- `/srv/moltbox-logs`
- `/var/lib/moltbox/secrets`
- `/mnt/moltbox-backup`

That is wrong for the target system. Runtime operation must not depend on `jpekovitch`, `jpkovic`, or any other personal user path.

## Live CLI Problems

The current CLI surface still reflects the old heavier model:

- `dev`
- `opensearch`
- `checkpoint`
- `skill`
- `plugin`
- `gateway docker ...`
- `gateway service ...`

Some of that surface is also partially drifted relative to the intended contract, which means the CLI is both too large and not cleanly trustworthy.

## Live OpenClaw Integration Problems

The runtime model still leans on:

- replay-heavy mutation
- checkpoint as a normal operational mechanism
- gateway-managed install history
- runtime rebuild assumptions that are heavier than the real operating model wants

The current runtime also warns that:

- `plugins.allow` is empty
- `behavior-guard` is loading as untracked local code

That is weak trust posture.

## Why This Cannot Be The Target

The current system fails the intended design on all of these:

- no real snapshot-backed host baseline
- too many appliance services
- personal ownership on critical paths
- too much gateway ownership of OpenClaw internals
- too much CLI surface
- no clean authority package in this repo

The box is good enough to extract facts from. It is not good enough to keep as the operating model.

## Immediate Consequences

The rebuild must:

- treat the current host as extraction state
- preserve useful live facts such as working model settings
- discard the old service shape
- discard the old authority split
- produce new repo-local design authority in `moltbox-gateway`

## Evidence Sources

For the deeper critique and fact trail, use:

- `../reviews/2026-04-04-openclaw-operating-model-review.md`
- `../reviews/2026-04-04-cli-surface-review.md`
- `../plans/2026-04-04-clean-moltbox-execution-plan.md`
