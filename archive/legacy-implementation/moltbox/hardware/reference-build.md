# MoltBox Reference Build

MoltBox is the physical appliance that hosts the local OpenClaw runtime and its supporting services.

This hardware layer defines:

- the reference machine profile
- GPU expectations for local model execution
- storage and log volume placement
- the capacity envelope for the appliance

This directory is intentionally descriptive rather than executable. Runtime behavior and operator workflows are documented under `docs/`, while deployment artifacts live under `moltbox/containers/` and `moltbox/config/`.
