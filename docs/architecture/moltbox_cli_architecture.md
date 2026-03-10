# MoltBox CLI Architecture

This document defines the active MoltBox CLI architecture in `remram-gateway`.

## Repository split

- `moltbox/` is the appliance artifact tree.
- `moltbox-cli/` is the CLI software project.
- `docs/` is the repository documentation root.
- `archive/MoltBox-Legacy/` is a frozen historical reference.

## Domain structure

The CLI is organized by the object being operated on:

- `runtime/` manages OpenClaw runtime environments
- `host/` manages shared services running on the MoltBox machine
- `tools/` manages the MoltBox deploy tooling itself

## Grammar

The canonical command is `moltbox`.

The canonical grammar is:

```text
moltbox <domain> <target> <verb>
```

Concrete forms:

- `moltbox runtime <environment> <verb>`
- `moltbox host <service> <verb>`
- `moltbox tools <verb>`

## Verb safety

- Inspection: `status`, `inspect`, `logs`, `health`, `version`
- Mutation: `deploy`, `rollback`, `start`, `stop`, `restart`, `update`

Inspection commands must not mutate state.

## Directory mapping

```text
moltbox/
  containers/
  hardware/
  config/

moltbox-cli/
  runtime/
  host/
  tools/
```

The `tools/` domain also contains the shared Python package source because the tooling domain owns the CLI implementation itself.
