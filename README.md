# remram-gateway

Execution substrate for the Remram system.

Remram Gateway provisions and runs the local OpenClaw control plane that powers Remram. It wires models, registers agents, connects to supporting services, and defines concrete appliance profiles such as Moltbox.

## Runtime Architecture

This repository is an installer and template source.

Repository:

```text
~/git/remram-gateway
```

Live runtime configuration and state:

```text
~/.openclaw
```

The `moltbox/.openclaw/` and `moltbox/config/` directories contain templates only. `scripts/20-bootstrap.sh` copies those templates into `~/.openclaw` on first run and leaves existing runtime files in place on subsequent runs.

Containers must read runtime configuration from `~/.openclaw`, not from repository paths.
They must never read live configuration from `~/git/remram-gateway`.

## Repository Structure

```text
remram-gateway/
  README.md
  moltbox/
    .openclaw/                 # Template OpenClaw YAML files
    config/                    # Template env/config files and compose definition
    scripts/                   # Installer, bootstrap, validation, maintenance
    moltbox-operator-runbook.md
    moltbox-implementation-guide.md
```

## Moltbox Runtime Layout

After bootstrap, the runtime root should look like:

```text
~/.openclaw/
  .env
  container.env
  model-runtime.yml
  opensearch.yml
  agents.yaml
  channels.yaml
  routing.yaml
  tools.yaml
  escalation.yaml
  agents/main/agent/models.json
  debug-service/
  logs/
```

## Usage

Use the operator runbook for deployment:

[`moltbox/moltbox-operator-runbook.md`](/d:/Development/RemRam/remram-gateway/moltbox/moltbox-operator-runbook.md)

Typical lifecycle:

1. Run [`10-install.sh`](/d:/Development/RemRam/remram-gateway/moltbox/scripts/10-install.sh) on the Ubuntu host.
2. Build or supply the OpenClaw image referenced by `OPENCLAW_IMAGE`.
3. Run [`20-bootstrap.sh`](/d:/Development/RemRam/remram-gateway/moltbox/scripts/20-bootstrap.sh) to create `~/.openclaw` and start the stack.
4. Run [`30-validate.sh`](/d:/Development/RemRam/remram-gateway/moltbox/scripts/30-validate.sh) to verify container health and internal connectivity.
5. Run [`70-debug-service.sh`](/d:/Development/RemRam/remram-gateway/moltbox/scripts/70-debug-service.sh) `install` to provision the host-level Moltbox debug service.

For manual Docker Compose operations, export the runtime root first:

```bash
export MOLTBOX_RUNTIME_ROOT="$HOME/.openclaw"
cd ~/git/remram-gateway/moltbox/config
docker compose ps
```

Quick runtime mount verification:

```bash
docker inspect "$(grep '^OPENCLAW_CONTAINER_NAME=' ~/.openclaw/.env | cut -d= -f2- || echo moltbox-openclaw)" | grep openclaw
```

Expected output should reference `~/.openclaw` and never the repository.

## Remote Development

Recommended workflow: VS Code with Remote-SSH connected directly to the Moltbox host.

Edit runtime files under `~/.openclaw` for live configuration changes. Edit repository files under `~/git/remram-gateway/moltbox/` only when changing templates, scripts, or compose definitions.
