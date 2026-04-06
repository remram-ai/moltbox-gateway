# Gateway Docs

This directory is the documentation home for `moltbox-gateway`.

Current truth lives in:

- `guides/`
- `design/`
- `ai-context/`

Historical records live in:

- `decisions/`
- `reviews/`
- `plans/`
- `runbooks/`

## Directory Map

- `guides/`
  - human-facing operator documentation
  - start at `guides/README.md`
- `design/`
  - current architecture, boundaries, and operating model
  - start at `design/README.md`
- `ai-context/`
  - compact AI bootstrap context for implementation and operations
  - start at `ai-context/README.md`
- dated folders
  - historical records and execution artifacts
  - use only after the current guides/design docs

## Human Entry Path

Read these first:

1. `guides/README.md`
2. `guides/operator-guide.md`
3. `guides/service-catalog.md`
4. `design/README.md`
5. `design/system-overview.md`
6. `design/cli-and-gateway.md`
7. `design/runtime-and-services.md`
8. `design/backup-and-recovery.md`
9. `design/host-and-operations.md`
10. `design/web-tooling.md`

## AI Entry Path

Import these first:

1. `guides/operator-guide.md`
2. `ai-context/README.md`
3. `ai-context/overview.md`
4. `ai-context/current-state.md`
5. `ai-context/future-state.md`
6. `ai-context/operator-workflows.md`
7. `ai-context/cortex-implementation-thread-prompt.md` if you are spinning up a Cortex builder or implementation thread

Then add the task-specific context file you need.

## Rule

If a current guide or design doc conflicts with a dated review or plan, the current guide/design doc wins for live Gateway work.
