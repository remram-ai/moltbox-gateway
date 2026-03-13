# Discord Channel Implementation Plan

This document is the execution companion to `docs/specs/07_discord_channel.md`.

## 1. Final Architecture

The MVP implementation uses OpenClaw's built-in Discord channel inside each runtime.

Chosen posture:

- no shared Discord capability container
- no custom Discord router
- no second orchestration loop
- one Discord bot identity per runtime
- runtime-scoped deployment through `moltbox runtime <environment> ...`

Runtime mapping:

- `dev` -> one dedicated Discord bot
- `test` -> one dedicated Discord bot
- `prod` -> one dedicated Discord bot

## 2. Canonical Integration Points

The native Discord implementation is accepted because upstream OpenClaw already provides:

- a built-in Discord channel plugin
- DM and guild-channel handling
- native slash-command support
- mention gating and allowlists
- outbound Discord sends

Canonical runtime path:

```text
Discord message
-> Discord preflight and route resolution
-> dispatchInboundMessage
-> dispatchReplyFromConfig
-> runPreparedReply
-> embedded agent runtime
```

That is the path that must remain authoritative for MVP.

## 3. Local Repo Deliverables

This implementation pass should produce:

- native Discord runtime config in `moltbox/config/openclaw/`
- removal of Signal as the active runtime ingress default
- operator documentation for per-runtime Discord bootstrap
- an updated Discord spec that reflects the native built-in design
- test coverage proving runtime render artifacts now ship Discord config

## 4. Implementation Steps

### Phase 1: Runtime Config

Update the checked-in runtime config so that:

- `channels.yaml` defines Discord instead of Signal
- Discord is disabled by default until a token is provisioned
- DM policy defaults to pairing
- guild policy defaults to allowlist
- config writes are disabled
- group DMs are disabled
- `agents.yaml` allows the `discord` channel instead of `signal`

### Phase 2: Operator Bootstrap

Add `docs/operator/discord.md` covering:

- one bot per runtime
- Discord Developer Portal setup
- runtime env token placement
- runtime channel enablement
- guild/channel allowlist setup
- DM pairing
- runtime-scoped broadcast examples

### Phase 3: Runtime Validation

Validate locally with repository tests that:

- runtime render still works
- rendered runtime config includes Discord
- rendered runtime config no longer includes Signal as the active ingress default

### Phase 4: Live Dev Validation

When credentials are available, validate in `dev`:

- DM mode
- channel mode
- slash commands or mentions
- outbound broadcast
- Semantic Router preservation

### Phase 5: Test Promotion

If runtime promotion can be done through the existing CLI, use:

```text
moltbox runtime test deploy
```

If that is not sufficient for Discord-specific bootstrap state:

- document the gap
- promote manually
- record the missing control-plane capability

## 5. Recommended Routing Posture

Use bot identity as the primary environment boundary.

Recommended channel pattern:

- `remram-dev` bot in `#remram-dev`
- `remram-test` bot in `#remram-test`
- `remram-prod` bot in `#remram-prod`

Recommended DM posture:

- DM the environment bot directly

Recommended channel posture:

- require mentions or slash commands in shared channels
- use dedicated environment channels when possible
- reject all non-allowlisted guilds or channels

## 6. Broadcast Posture

Broadcast should use the native OpenClaw outbound message path from the owning runtime.

Initial operator path:

- execute `openclaw message send --channel discord ...` inside the target runtime container

Follow-up improvement path:

- add a Moltbox runtime-scoped wrapper later if broadcast becomes a routine operator action

## 7. CLI Gaps Discovered

Current gaps discovered during planning and implementation:

- there is no `moltbox runtime <environment> discord-enable` helper
- there is no `moltbox runtime <environment> discord-broadcast` helper
- there is no control-plane wrapper for editing Discord allowlists safely
- there is no automated promotion primitive for Discord-specific bootstrap state across runtimes
- end-to-end Discord validation still depends on external bot credentials and Discord server state

These are the follow-up CLI/control-plane tasks. They do not require a custom Discord transport layer for MVP.

## 8. Rejection of the Shared-Capability Design

The earlier shared Discord capability design is intentionally not implemented because it would introduce extra control-plane and routing surface without solving an MVP blocker.

The native built-in approach is thinner because it:

- reuses the upstream OpenClaw Discord channel
- keeps routing inside the runtime boundary
- avoids a gateway-side event service
- avoids a runtime fan-out router

## 9. Completion Gate

Do not declare implementation complete until:

- the runtime config changes are merged
- the operator guide exists
- render/runtime tests pass
- live Discord validation has confirmed DM, channel, and broadcast behavior
- the final report can truthfully state whether native Discord was sufficient

