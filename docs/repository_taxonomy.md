# Repository Taxonomy

This is the canonical repository taxonomy and rules artifact for future AI and human changes.

## Top-level rules

- `archive/` contains frozen historical material only.
- `docs/` contains repository-level documentation.
- `moltbox/` contains appliance artifacts only.
- `moltbox-cli/` contains the software project that implements the `moltbox` CLI.

## Ownership boundaries

- `openclaw.*` owns the runtime substrate.
- `remram.*` owns the AI orchestration and knowledge system.
- `moltbox.*` owns appliance infrastructure and operator tooling.

## Active layout

```text
archive/
  MoltBox-Legacy/

docs/

moltbox/
  containers/
  hardware/
  config/

moltbox-cli/
  runtime/
  host/
  tools/
```

## Non-negotiable rules

- Do not place active MoltBox code in `archive/`.
- Do not place appliance artifacts in `moltbox-cli/`.
- Do not place CLI software in `moltbox/`.
- Do not reintroduce `inner-loop`, `control-plane`, or similar legacy roots as active project folders.
- The folder `moltbox-cli` is a repository path, not a CLI namespace token.
