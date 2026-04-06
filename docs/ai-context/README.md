# AI Context

This directory is the AI bootstrap layer for `moltbox-gateway`.

Use it to load the current operator contract without reading every historical plan and review.

## Minimum Import Set

1. `overview.md`
2. `current-state.md`
3. `future-state.md`
4. `operator-workflows.md`
5. `cortex-implementation-thread-prompt.md` when starting or redirecting a Cortex implementation thread

Then add the task-specific context file you need.

## Rule

These files summarize the current guides and design docs.

If they conflict with dated review or plan docs, the current guides and design docs win.

## Practical Loading Order

- start with `../guides/operator-guide.md` if the task touches the live appliance or CLI
- load this directory next for compact AI-ready context
- load `cli-gateway.md`, `runtime-services.md`, or `host-ops.md` only if the task needs them
