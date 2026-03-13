# remram-gateway

`remram-gateway` contains multiple subsystems. The active MoltBox work in this repository is split into two artifacts:

- `moltbox/` for appliance deployment artifacts
- `moltbox-cli/` for the software project that implements the `moltbox` command

The repository layout is intentionally narrow:

```text
remram-gateway/
├─ archive/
│  └─ MoltBox-Legacy/
├─ docs/
├─ moltbox/
│  ├─ containers/
│  ├─ hardware/
│  └─ config/
└─ moltbox-cli/
   ├─ runtime/
   ├─ host/
   └─ tools/
```

Start with these references:

- `docs/repository_taxonomy.md`
- `docs/cli/README.md`
- `docs/architecture/moltbox_cli_architecture.md`

Archive material is reference-only. Active code must not import from `archive/MoltBox-Legacy/`.
