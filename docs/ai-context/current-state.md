# Current State

Use this file when the task depends on what the live box looks like today.

Important current facts:

- host is `moltbox-prime`
- current OS is Ubuntu 24.04.4
- current active root is EXT4, not ZFS
- backup disk exists at `/mnt/moltbox-backup`
- live services still include `openclaw-dev` and `opensearch`
- `test` and `prod` currently chat through local Ollama using `mistral:7b-instruct-32k`
- current chat path still shows gateway pairing failure followed by embedded fallback
- critical paths are still owned by the personal user `jpekovitch`
- current CLI surface is too large and still carries `dev`, `opensearch`, `checkpoint`, `skill`, `plugin`, and `gateway service`

Interpretation:

- the live box is useful for extracting facts and backups
- the live box is not the target architecture
- the current host is extraction state only until it is rebuilt on ZFS

Canonical source:

- `../design/current-state.md`
