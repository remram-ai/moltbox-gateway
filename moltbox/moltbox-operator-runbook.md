# moltbox-operator-runbook.md

This checklist takes a fresh Ubuntu machine to the first OpenClaw chat in Moltbox.

## Architecture Rule

Moltbox is an installer plus template repository.

Repository:

```text
~/remram-gateway
```

Live runtime configuration and state:

```text
~/.openclaw
```

Containers must read runtime configuration from `~/.openclaw`. The repository must remain stateless templates and scripts.

## 0. Prerequisite: NVIDIA Driver Health

Moltbox requires a working NVIDIA driver on the host before deployment.

```bash
nvidia-smi
```

`nvidia-smi` must succeed. If it fails, fix NVIDIA driver installation before continuing.

## 1. Verify Ubuntu

```bash
cat /etc/os-release
```

`scripts/10-install.sh` requires Ubuntu.

## 2. Add the Non-Interactive Shell Guard

Before using VS Code Remote-SSH or other remote tooling, ensure `~/.bashrc` does not print output during non-interactive shells.

Add this block at the top of `~/.bashrc`:

```bash
# Prevent non-interactive shells from emitting output (breaks VSCode Remote-SSH)
case $- in
    *i*) ;;
      *) return;;
esac
```

This prevents Remote-SSH handshake failures caused by shell startup output.

## 3. Install Git and Clone the Repository

```bash
sudo apt-get update
sudo apt-get install -y git
cd ~
git clone https://github.com/Remram-AI/remram-gateway.git
cd ~/remram-gateway/moltbox
```

## 4. Run the Host Installer

```bash
bash ./scripts/10-install.sh
```

`scripts/10-install.sh` installs or validates:

- Docker Engine
- Docker Compose plugin
- `curl`
- NVIDIA Container Toolkit
- host GPU readiness via `nvidia-smi`
- Docker daemon availability
- `vm.max_map_count=262144` for OpenSearch

If Docker commands fail without `sudo` after this step, either log out and back in or run `newgrp docker`.

## 5. Confirm GPU Driver Visibility

```bash
nvidia-smi
```

`nvidia-smi` must succeed before continuing because the Ollama service is configured with `gpus: all`.

## 6. Build the Required Local OpenClaw Image

```bash
cd ~
git clone https://github.com/openclaw/openclaw.git
cd ~/openclaw
sudo docker build -t openclaw:local .
```

This creates the image name Moltbox expects by default: `OPENCLAW_IMAGE=openclaw:local`.

## 7. Bootstrap the Moltbox Runtime

```bash
cd ~/remram-gateway/moltbox
bash ./scripts/20-bootstrap.sh
```

Bootstrap performs the following:

- creates `~/.openclaw`
- creates `~/.openclaw/agents/main/agent`
- copies repository templates into `~/.openclaw` only when the runtime files are missing
- preserves existing runtime files on subsequent runs
- starts the container stack
- pre-pulls the local routing model
- waits for gateway readiness

If bootstrap times out on a slower machine, rerun with longer waits:

```bash
BOOTSTRAP_OLLAMA_WAIT_SECONDS=180 BOOTSTRAP_GATEWAY_WAIT_SECONDS=180 bash ./scripts/20-bootstrap.sh
```

## 8. Validate the Stack

```bash
bash ./scripts/30-validate.sh
```

`scripts/30-validate.sh` verifies:

- container health
- gateway endpoints
- OpenClaw to Ollama connectivity
- OpenClaw to OpenSearch connectivity
- internal-only port exposure policy

Signal integration is optional for this checklist and does not block first web chat.

## 9. Manual Compose Context

For direct `docker compose` commands, export the runtime root and run from the compose directory:

```bash
export MOLTBOX_RUNTIME_ROOT="$HOME/.openclaw"
cd ~/remram-gateway/moltbox/config
```

## 10. Post-Install Validation Commands

Check container state:

```bash
docker compose ps
```

Check OpenClaw configuration:

```bash
docker exec moltbox-openclaw openclaw doctor
```

Check gateway endpoints:

```bash
curl http://127.0.0.1:18789/healthz
curl http://127.0.0.1:18789/readyz
```

## 11. Verify the Gateway Token

Read the token from the runtime env file:

```bash
grep '^OPENCLAW_GATEWAY_TOKEN=' ~/.openclaw/.env | cut -d= -f2-
```

Verify the running container sees the same token:

```bash
docker exec -it moltbox-openclaw env | grep OPENCLAW_GATEWAY_TOKEN
```

If you change `~/.openclaw/.env`, restart the stack:

```bash
docker compose restart
```

If the container still reports the previous token after restart, rerun:

```bash
cd ~/remram-gateway/moltbox
bash ./scripts/20-bootstrap.sh
```

This forces the stack to reconcile against the current runtime files in `~/.openclaw`.

## 12. Open the Gateway and Send the First Chat

Get the host LAN IP:

```bash
hostname -I
```

Use the LAN IP, not loopback (`127.0.0.1`) or Docker bridge addresses.

Open OpenClaw:

- On the Moltbox machine: `http://127.0.0.1:18789`
- From another LAN device: `http://<MOLTBOX_LAN_IP>:18789`

When prompted, enter the token from `~/.openclaw/.env` and send the first message.

## 13. Remote Development Workflow

Recommended tooling:

- VS Code
- Remote-SSH extension

Example SSH config:

```sshconfig
Host moltbox
    HostName <MOLTBOX_IP>
    User jpekovitch
```

Recommended folder to open:

```text
/home/jpekovitch/remram-gateway
```

For live runtime edits, also open:

```text
/home/jpekovitch/.openclaw
```

This avoids SCP and lets you edit:

- `~/.openclaw/.env`
- `~/.openclaw/model-runtime.yml`
- `~/remram-gateway/moltbox/config/docker-compose.yml`
- repository templates under `~/remram-gateway/moltbox/`
