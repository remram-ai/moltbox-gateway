# Host Tools

This directory contains the source-controlled host-side deployment primitives for
the new `dev-manager` control plane.

Design rules:

- one script per operation
- structured JSON output on stdout
- non-zero exit code on failure
- no silent mutation outside the requested scope
- safe to compose from the control pane API

Initial primitive set:

- `runtime/create-runtime.sh`
- `runtime/destroy-runtime.sh`
- `stack/start-runtime.sh`
- `stack/stop-runtime.sh`
- `stack/restart-runtime.sh`
- `deploy/deploy-commit.sh`
- `validate/validate-runtime.sh`
- `diagnostics/collect-diagnostics.sh`
- `snapshot/snapshot-runtime.sh`
- `snapshot/restore-snapshot.sh`

The current scripts are scaffolds that establish the contract and file layout.
Implementation will be filled in incrementally as the new control plane replaces
the legacy Moltbox workflow.
