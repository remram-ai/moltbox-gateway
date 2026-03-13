# Moltbox CLI Reference

This document describes the current Moltbox CLI implemented in the gateway control plane.

Architecture authority:

- `remram/architecture-v2/gateway.md`
- `remram/architecture-v2/services.md`
- `remram/architecture-v2/runtime.md`
- `remram/architecture-v2/skills.md`

## Command Shape

The canonical grammar is:

```text
moltbox <component> <command>
```

Primary components:

- `gateway`
- `service`
- `skill`
- `openclaw`
- `openclaw-dev`
- `openclaw-test`
- `openclaw-prod`
- `caddy`
- `opensearch`

Alias rule:

- `openclaw == openclaw-prod`

## Host Invocation

On the appliance host, bootstrap installs a PATH wrapper so operators can run:

```bash
moltbox ...
```

directly from the Linux host shell.

The wrapper executes the CLI inside the running `gateway` container. Inside that container, the CLI now auto-discovers the canonical mounted config at:

```text
/etc/moltbox/config.yaml
```

Normal operator use should not require `--config-path`.

## Global Options

Advanced overrides remain available:

```text
--config-path
--state-root
--logs-root
--runtime-artifacts-root
--services-repo-url
--runtime-repo-url
--skills-repo-url
--internal-host
--internal-port
--cli-path
```

These are mainly for bootstrap, testing, and controlled overrides.

## Gateway Commands

Gateway self-management:

```bash
moltbox gateway health
moltbox gateway status
moltbox gateway inspect
moltbox gateway logs
moltbox gateway update
```

Host-side upstream repo mirror management:

```bash
moltbox gateway repo refresh
moltbox gateway repo refresh services
moltbox gateway repo refresh runtime
moltbox gateway repo refresh skills

moltbox gateway repo seed services --bundle /path/to/moltbox-services.bundle
moltbox gateway repo seed runtime --bundle /path/to/moltbox-runtime.bundle
moltbox gateway repo seed skills --bundle /path/to/remram-skills.bundle
```

Notes:

- `repo refresh` updates the configured host-side upstream checkout and then refreshes the cached working checkout used by the gateway.
- `repo seed` supports advancing a mirror from a Git bundle when direct GitHub access is unavailable on the appliance.

## Service Commands

`service` is the deployment pipeline for container lifecycle operations.

```bash
moltbox service list
moltbox service inspect gateway
moltbox service status caddy
moltbox service logs opensearch

moltbox service deploy gateway
moltbox service deploy openclaw-dev
moltbox service deploy openclaw-test
moltbox service deploy openclaw-prod
moltbox service deploy caddy
moltbox service deploy opensearch

moltbox service start caddy
moltbox service stop caddy
moltbox service restart caddy
moltbox service rollback caddy
moltbox service doctor caddy
```

Artifact overrides:

```bash
moltbox service deploy openclaw-prod --version 1.4.2
moltbox service deploy gateway --commit abc1234
```

## Runtime Component Commands

Runtime operations target real runtime components:

```bash
moltbox openclaw status
moltbox openclaw inspect
moltbox openclaw logs
moltbox openclaw config sync
moltbox openclaw reload
moltbox openclaw doctor
moltbox openclaw monitor
```

Environment-specific forms:

```bash
moltbox openclaw-dev status
moltbox openclaw-dev config sync
moltbox openclaw-dev reload

moltbox openclaw-test status
moltbox openclaw-test config sync
moltbox openclaw-test reload

moltbox openclaw-prod status
moltbox openclaw-prod config sync
moltbox openclaw-prod reload
```

## Skill Commands

`skill deploy` is an orchestration pipeline.

Default runtime targeting:

```bash
moltbox skill deploy semantic-router
```

Explicit runtime targeting:

```bash
moltbox skill deploy semantic-router --runtime openclaw-test
moltbox openclaw-test skill deploy semantic-router
```

Notes:

- plugin-backed skills and pure skills are both supported
- runtime targeting should use canonical Moltbox runtimes such as `openclaw-dev`, `openclaw-test`, `openclaw-prod`, or `openclaw`

## Output Model

The CLI emits structured JSON for both success and failure.

Success payloads include command metadata and operation details.

Failure payloads include:

- `error_type`
- `error_message`
- `recovery_message`

This output is intended for both operators and automation.

## Legacy Command Surface

The older `tools`, `host`, and `runtime` namespaces are no longer canonical.

Examples:

```text
moltbox tools status      -> moltbox gateway status
moltbox host ssl deploy   -> moltbox service deploy caddy
moltbox runtime dev reload -> moltbox openclaw-dev reload
```

The CLI now returns explicit replacement guidance when those legacy forms are used.
