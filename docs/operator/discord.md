# Discord Operator Guide

## Purpose

Discord is the active MVP external messaging ingress for `remram-gateway`.

The gateway uses OpenClaw's built-in Discord channel inside each runtime. There is no separate Discord service container for MVP.

## Access Method

One Discord bot identity is assigned to each runtime environment:

- `dev`
- `test`
- `prod`

Recommended posture:

- DM the environment bot directly for private operator access
- use dedicated environment channels such as `#remram-dev`, `#remram-test`, and `#remram-prod` for guild interaction

## Ports

No inbound Discord webhook port is required.

The runtime connects outbound to the Discord gateway.

The OpenClaw runtime still exposes its normal local gateway port through the runtime container:

- `dev`: `18790`
- `test`: `28789`
- `prod`: `38789`

## CLI Operations

Discord lifecycle is runtime-scoped.

Primary commands:

```text
moltbox runtime <environment> deploy
moltbox runtime <environment> start
moltbox runtime <environment> stop
moltbox runtime <environment> restart
moltbox runtime <environment> status
moltbox runtime <environment> logs
```

Examples:

```text
moltbox runtime dev deploy
moltbox runtime dev status
moltbox runtime dev logs
```

There is no `moltbox discord ...` namespace in the current MVP implementation.

## Runtime Behavior

Discord traffic enters the built-in OpenClaw Discord channel and then follows the normal OpenClaw reply lifecycle.

Supported upstream behaviors include:

- DMs
- guild channels
- mentions
- slash commands
- outbound message sends

The runtime remains authoritative for:

- Agent Loop entry
- Semantic Router execution
- model escalation
- tool execution

## Checked-In Config

Repo config files:

- [channels.yaml](/D:/Development/RemRam/remram-gateway/moltbox/config/openclaw/channels.yaml)
- [agents.yaml](/D:/Development/RemRam/remram-gateway/moltbox/config/openclaw/agents.yaml)

Runtime-owned files after deployment:

- `${MOLTBOX_RUNTIME_ROOT}/openclaw/<environment>/container.env`
- `${MOLTBOX_RUNTIME_ROOT}/openclaw/<environment>/channels.yaml`

Default posture in git:

- Discord wired into runtime config
- Discord disabled by default
- DM policy set to pairing
- guild policy set to allowlist

## Logs

Use the runtime log surface:

- `moltbox runtime <environment> logs`

Host log roots:

- `~/Moltbox/logs/openclaw-dev/`
- `~/Moltbox/logs/openclaw-test/`
- `~/Moltbox/logs/openclaw-prod/`

Discord activity is logged inside the owning runtime because Discord is not a separate service.

## Persistent State

Discord state is runtime-owned for MVP.

Primary runtime root:

- `~/Moltbox/openclaw/dev/`
- `~/Moltbox/openclaw/test/`
- `~/Moltbox/openclaw/prod/`

Relevant files include:

- `container.env` for runtime-local secrets such as `DISCORD_BOT_TOKEN`
- `channels.yaml` for Discord channel policy
- normal OpenClaw runtime state and session data

## Bootstrap Steps

### 1. Create one Discord bot per runtime

In the Discord Developer Portal, create a separate application and bot for each runtime you want to expose.

Recommended names:

- `remram-dev`
- `remram-test`
- `remram-prod`

### 2. Enable required intents

For each bot, enable:

- Message Content Intent
- Server Members Intent

Presence Intent is optional.

### 3. Invite the bot

Invite the bot to the target Discord server with:

- `bot`
- `applications.commands`

Minimum useful permissions:

- View Channels
- Send Messages
- Read Message History
- Embed Links
- Attach Files

### 4. Store the token in the runtime env file

Set the runtime-specific token in:

```text
${MOLTBOX_RUNTIME_ROOT}/openclaw/<environment>/container.env
```

Example:

```env
DISCORD_BOT_TOKEN=replace-with-runtime-specific-token
```

### 5. Enable Discord in the runtime config

Edit the deployed runtime file:

```text
${MOLTBOX_RUNTIME_ROOT}/openclaw/<environment>/channels.yaml
```

Set:

```yaml
channels:
  discord:
    enabled: true
```

For render-time bootstrap through the Moltbox CLI, the runtime renderer also supports:

```text
MOLTBOX_DISCORD_ENABLED_DEV=true
MOLTBOX_DISCORD_GUILD_ID_DEV=<guild-id>
MOLTBOX_DISCORD_CHANNEL_ID_DEV=<channel-id>
```

Equivalent `_TEST` and `_PROD` variants exist for the other runtimes.

### 6. Add guild and channel allowlists

Use numeric Discord IDs.

Example:

```yaml
channels:
  discord:
    enabled: true
    dmPolicy: pairing
    groupPolicy: allowlist
    configWrites: false
    dm:
      enabled: true
      groupEnabled: false
    guilds:
      "123456789012345678":
        requireMention: true
        users:
          - "234567890123456789"
        channels:
          "345678901234567890":
            requireMention: true
```

### 7. Deploy or restart the runtime

Example:

```text
moltbox runtime dev deploy
```

### 8. Complete DM pairing

DM the runtime bot from the operator account.

Approve the pairing flow through the normal OpenClaw pairing process for Discord.

## Broadcast

Initial broadcast operations use the native OpenClaw CLI inside the runtime container.

Example:

```text
docker exec openclaw-dev openclaw message send --channel discord --target channel:345678901234567890 --message "New build deployed to dev runtime."
```

This is the current operator path until Moltbox exposes a runtime-scoped wrapper.

## Health / Debugging

Check runtime status first:

```text
moltbox runtime dev status
```

Then inspect runtime logs:

```text
moltbox runtime dev logs
```

Common causes of failure:

- `DISCORD_BOT_TOKEN` missing or invalid
- Discord left disabled in `channels.yaml`
- bot missing Message Content intent
- guild or channel not allowlisted
- operator user ID not allowlisted for the selected guild

## Failure Recovery

If Discord traffic stops working:

1. confirm the runtime is healthy with `moltbox runtime <environment> status`
2. confirm the token is present in `container.env`
3. confirm `channels.discord.enabled: true`
4. confirm guild and channel IDs are correct
5. restart or redeploy the owning runtime

If the runtime must be promoted to another environment, remember that Discord bootstrap state is environment-specific and may require manual token and allowlist setup in the target runtime.
