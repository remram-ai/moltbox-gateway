# Runtime Monitor API

Status: proposed

This document defines the required monitor and operator API contract for managed Moltbox runtimes.

It is the runtime-facing companion to:

```text
docs/specs/08_cli_refactor_control_plane.md
```

## 1. Required Endpoint

- method: `GET`
- path: `/moltbox/monitor`

Every managed runtime must implement this endpoint.

## 2. Purpose

The monitor endpoint gives the Moltbox control plane a stable health and readiness view without inspecting container internals.

It is used by:

- `moltbox runtime <environment> monitor`
- `moltbox runtime <environment> status`
- `moltbox host doctor`
- tools-service MCP diagnostics

## 3. Required Response Shape

```json
{
  "runtime": "openclaw",
  "environment": "dev",
  "status": "healthy",
  "version": "0.0.0",
  "started_at": "2026-03-11T00:00:00Z",
  "provider": "ollama",
  "model": "qwen3:8b",
  "chat_ready": true,
  "dependencies": {
    "ollama": "healthy",
    "opensearch": "healthy"
  }
}
```

Required fields:

- `runtime`
- `environment`
- `status`
- `version`
- `started_at`
- `provider`
- `model`
- `chat_ready`
- `dependencies`

## 4. Optional Fields

Optional fields:

- `uptime_seconds`
- `ready`
- `gateway`
- `routing`
- `channels`
- `doctor_summary`
- `warnings`
- `config_revision`
- `capabilities`

Recommended optional shape:

```json
{
  "gateway": {
    "port": 18789,
    "bind": "lan"
  },
  "routing": {
    "enabled": true,
    "provider": "remram-router",
    "default_model": "ollama/qwen3:8b"
  },
  "channels": {
    "discord": {
      "enabled": true,
      "status": "healthy"
    }
  },
  "warnings": []
}
```

## 5. Health Semantics

Allowed `status` values:

- `healthy`
- `degraded`
- `starting`
- `unhealthy`
- `stopped`

Rules:

- `healthy` means the runtime can serve normal chat traffic
- `degraded` means the runtime is up but one or more dependencies or capabilities are impaired
- `starting` means the runtime is still warming or probing dependencies
- `unhealthy` means the runtime is reachable but not ready for normal operations
- `stopped` means the runtime is intentionally not running

Additional rules:

- `chat_ready: true` is required for `healthy`
- dependency failures should normally produce `degraded` or `unhealthy`
- optional capability warnings should not force `unhealthy` unless that capability is required by the environment contract

## 6. Required Companion Operator Endpoints

The monitor endpoint is the minimum contract. Managed runtimes must also expose:

- `GET /moltbox/doctor`
- `GET /moltbox/models/status`
- `GET /moltbox/routing/status`
- `GET /moltbox/config/inspect`
- `POST /moltbox/chat`

Rules:

- every endpoint returns structured JSON
- configuration inspection must redact secrets
- channel-specific diagnostics may live under `/moltbox/channels/<capability>/...`

## 7. Response Rules

All runtime operator endpoints must return:

- `ok`
- `source`
- `timestamp`

Where applicable they must also return:

- `runtime`
- `environment`
- `diagnostics`
- `warnings`
- `errors`

## 8. Security Rules

The runtime operator API is for the Moltbox control plane.

Rules:

- it must not expose raw secret material
- it must not become a general-purpose admin shell
- it must support the tools service as the canonical remote caller
- any direct remote exposure outside the Moltbox control plane must be treated as a separate security decision
