# Discord Channel

This document defines the active MVP external messaging ingress for `remram-gateway`.

The archived Signal design remains preserved in:

```text
docs/specs/later_signal_channel.md
docs/specs/later_signal_channel_implementation_plan.md
```

## 1. Implementation Decision

Discord is implemented through OpenClaw's built-in Discord channel, not through a separate gateway-side Discord adapter or shared Discord appliance container.

That decision is binding for the MVP unless a later topology requirement proves the native channel insufficient.

The chosen posture is:

- use the built-in OpenClaw Discord channel
- keep Discord inside the normal runtime process
- avoid a second orchestration loop
- avoid a shared Discord service container
- avoid a custom router unless native Discord fails a real runtime requirement

## 2. Architecture Boundary

Discord is transport only.

It does:

- receive DMs, channel messages, mentions, and slash-command interactions through the native OpenClaw Discord channel
- enter the normal OpenClaw reply lifecycle
- preserve the Semantic Router and existing model escalation behavior
- return replies through Discord
- support outbound Discord sends from the runtime for broadcasts and notices

It does not:

- create a second execution stack
- call models directly outside the Agent Loop
- bypass the Gateway authority boundary
- bypass the Semantic Router
- mutate runtime config from chat

Canonical request path:

```text
Discord DM or channel event
-> OpenClaw built-in Discord channel
-> normal OpenClaw inbound reply dispatch
-> Agent Loop / Semantic Router / orchestration
-> runtime reply
-> Discord reply
```

## 3. Runtime Ownership Model

Runtime environments remain:

- `dev`
- `test`
- `prod`

Each runtime owns exactly one Discord bot identity.

Recommended MVP posture:

- `dev` runtime -> `remram-dev` bot
- `test` runtime -> `remram-test` bot
- `prod` runtime -> `remram-prod` bot

This is the thinnest safe routing model because it keeps environment routing explicit without a cross-runtime dispatcher.

Consequences:

- DM routing is explicit by bot identity
- channel routing is explicit by bot identity plus allowlisted guild/channel placement
- ambiguous cross-runtime routing is rejected structurally because a runtime only receives events for its own bot token

## 4. Deployment Model

Discord is runtime-native, not a shared Moltbox capability container.

There is no MVP deployment target such as:

```text
moltbox discord deploy
```

Instead, Discord rides with the existing runtime deployment surface:

```text
moltbox runtime <environment> deploy
moltbox runtime <environment> start
moltbox runtime <environment> stop
moltbox runtime <environment> restart
moltbox runtime <environment> status
moltbox runtime <environment> logs
```

The runtime container connects outbound to Discord and requires no inbound webhook listener or public firewall exposure.

## 5. Native Discord Capability Requirements

The selected native channel must support:

- DM messages
- guild channel messages
- slash commands
- mention-triggered interaction
- outbound replies and message sends

This spec assumes those capabilities through the current upstream OpenClaw Discord implementation and does not add a custom gateway adapter while they remain available.

## 6. Environment Routing Rules

Routing must remain explicit and safe.

For MVP, explicit routing is achieved through identity and channel placement rather than a gateway-side environment parser.

Required posture:

- one Discord bot token per runtime
- one runtime process per bot token
- allowlisted guilds and channels per runtime
- no single shared bot spanning `dev`, `test`, and `prod`

Recommended operator UX:

- DMs: message the environment bot directly
- channels: use dedicated environment channels such as `#remram-dev`, `#remram-test`, and `#remram-prod`
- mentions and slash commands: allowed inside those environment-owned channels only

Ambiguous requests must be rejected by configuration posture:

- if a guild or channel is not allowlisted for that runtime, OpenClaw must ignore it
- if an operator needs cross-environment access, they must address the correct bot or channel explicitly

## 7. Runtime Configuration Posture

The checked-in runtime defaults must be conservative.

Required default posture:

- Discord wired into runtime config
- Discord disabled by default in git-tracked config so deployments do not fail without credentials
- DM policy set to pairing
- guild message policy set to allowlist
- config writes from Discord disabled
- group DMs disabled

Per-runtime enablement happens during bootstrap after the operator provisions:

- the runtime-specific bot token
- the runtime-specific guild and channel IDs
- the operator user IDs or allowlists required for safe access

## 8. Broadcast Model

Discord broadcast uses the native OpenClaw outbound message path from the selected runtime.

Examples include:

- deployment announcements
- runtime health summaries
- operator notices

For MVP, broadcast may be invoked through the runtime's OpenClaw CLI until the Moltbox control plane exposes a higher-level wrapper.

Discord broadcast must still originate from the runtime that owns the bot identity sending the message.

## 9. Logs and State

Because Discord is runtime-native, it uses runtime-owned logs and runtime-owned state rather than a shared Discord state root.

Canonical operator surfaces:

- logs: `${MOLTBOX_RUNTIME_ROOT}/logs/openclaw-<environment>/`
- runtime root: `${MOLTBOX_RUNTIME_ROOT}/openclaw/<environment>/`

Discord-specific materials are expected inside the runtime boundary, including:

- runtime env files containing the bot token
- runtime config files such as `channels.yaml`
- normal OpenClaw session and channel state for that runtime

This MVP does not define a shared Discord durable-state root under `${MOLTBOX_STATE_ROOT}/shared/discord/` because no shared Discord service is being deployed.

## 10. Operator Surface

Primary operator lifecycle remains runtime-scoped:

```text
moltbox runtime dev deploy
moltbox runtime dev status
moltbox runtime dev logs
```

Discord chat is not a control-plane surface.

Discord is the transport used by operators after the runtime is configured and running.

## 11. Bootstrap

Bootstrap is required because each runtime needs its own Discord application and bot token.

Bootstrap includes:

- create one Discord bot per runtime
- enable required Discord intents for each bot
- store `DISCORD_BOT_TOKEN` in the target runtime env file
- enable Discord in the runtime's `channels.yaml`
- add guild and channel allowlists for that runtime
- deploy or restart the runtime
- complete DM pairing for the operator account

Steady-state operation returns to the normal runtime lifecycle commands after bootstrap.

## 12. Thin-Adapter Rule

No custom Discord router should be built while all of the following remain true:

- the native OpenClaw Discord channel supports DMs
- the native OpenClaw Discord channel supports guild channels
- the native OpenClaw Discord channel supports mentions and slash commands
- outbound runtime sends cover the required broadcast use cases
- per-runtime bot identities keep environment routing explicit

Only if one shared Discord surface must route across multiple runtimes should a thin router be reconsidered.

If that ever becomes necessary, the router may only:

- receive Discord events
- resolve the target runtime
- call the canonical runtime entry
- forward the reply

It must not become a second agent executor.

## 13. Known Gaps

Current known gaps for MVP:

- no dedicated runtime-scoped helper exists yet for Discord bootstrap or allowlist management
- no Moltbox wrapper exists yet for outbound Discord broadcast
- no automated promotion helper exists yet for copying Discord runtime config from `dev` to `test`
- end-to-end verification still requires real Discord credentials and real guild/channel IDs

These are implementation and operations gaps, not reasons to add a separate Discord appliance.

## 14. Companion Plan

Implementation sequencing is tracked in:

```text
docs/specs/07_discord_channel_implementation_plan.md
```

## 15. Verification Criteria

Discord MVP is not complete until:

- requests enter the normal OpenClaw inbound reply path
- the Semantic Router still runs for Discord-originated requests
- no second execution stack exists
- DM mode works
- channel mode works
- broadcast works
- environment routing remains explicit
- ambiguous routing is rejected by configuration and topology

