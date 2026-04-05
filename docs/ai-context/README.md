# AI Context

This directory is the AI bootstrap layer for `moltbox-gateway`.

It exists so an AI can load a small number of files and understand:

- where the box is now
- what the target shape is
- what the gateway and CLI are supposed to do
- how the host and operations model works
- how to execute the rebuild

If these files conflict with older `remram/docs/ai-context` files, these files win for gateway implementation work.

## Minimum Import Sets

### Full rebuild import set

Import:

1. `overview.md`
2. `current-state.md`
3. `future-state.md`
4. `implementation-plan.md`

### CLI and gateway work

Import:

1. `overview.md`
2. `cli-gateway.md`
3. `future-state.md`

### Runtime and service work

Import:

1. `overview.md`
2. `runtime-services.md`
3. `future-state.md`

### Host and operations work

Import:

1. `overview.md`
2. `host-ops.md`
3. `current-state.md`
4. `implementation-plan.md`

## Canonical Sources

These AI-context files summarize the canonical docs in:

- `../design/`
- `../decisions/`
- `../reviews/`
- `../plans/`
- `../runbooks/`

## Recommended Order

1. Read `overview.md`.
2. Read `current-state.md` and `future-state.md`.
3. Read the themed context that matches the task.
4. Use the execution plans and runbooks only after the design intent is clear.
