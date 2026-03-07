#!/usr/bin/env bash
set -euo pipefail

# Moltbox bootstrap routine.
# Creates runtime files under ~/.openclaw, starts the stack, and pre-pulls the local routing model.

timestamp() { date +"%Y-%m-%dT%H:%M:%S%z"; }
log_info() { echo "[$(timestamp)] [INFO] $*"; }
log_warn() { echo "[$(timestamp)] [WARN] $*" >&2; }
log_error() { echo "[$(timestamp)] [ERROR] $*" >&2; }

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MOLTBOX_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
REPO_ROOT="$(cd "${MOLTBOX_DIR}/.." && pwd)"
CONFIG_DIR="${MOLTBOX_DIR}/config"
OPENCLAW_TEMPLATE_DIR="${MOLTBOX_DIR}/.openclaw"
COMPOSE_FILE="${CONFIG_DIR}/docker-compose.yml"
OPENCLAW_CONFIG_TEMPLATE="${CONFIG_DIR}/openclaw.json"
ENV_TEMPLATE="${CONFIG_DIR}/.env.example"
CONTAINER_ENV_TEMPLATE="${CONFIG_DIR}/container.env.example"
MODEL_RUNTIME_TEMPLATE="${CONFIG_DIR}/model-runtime.yml"
OPENSEARCH_TEMPLATE="${CONFIG_DIR}/opensearch.yml"

resolve_runtime_root() {
  local target_user="${SUDO_USER:-${USER}}"
  local target_home=""

  if command -v getent >/dev/null 2>&1; then
    target_home="$(getent passwd "${target_user}" | cut -d: -f6)"
  fi

  if [[ -z "${target_home}" ]]; then
    target_home="${HOME}"
  fi

  printf '%s\n' "${MOLTBOX_RUNTIME_ROOT:-${target_home}/.openclaw}"
}

RUNTIME_ROOT="$(resolve_runtime_root)"
USER_HOME="$(getent passwd "${SUDO_USER:-${USER}}" | cut -d: -f6 2>/dev/null || printf '%s\n' "${HOME}")"
RUNTIME_ENV_FILE="${RUNTIME_ROOT}/.env"
RUNTIME_OPENCLAW_CONFIG_FILE="${RUNTIME_ROOT}/openclaw.json"
RUNTIME_CONTAINER_ENV_FILE="${RUNTIME_ROOT}/container.env"
RUNTIME_MODEL_RUNTIME_FILE="${RUNTIME_ROOT}/model-runtime.yml"
RUNTIME_OPENSEARCH_FILE="${RUNTIME_ROOT}/opensearch.yml"
RUNTIME_AGENT_DIR="${RUNTIME_ROOT}/agents/main/agent"
RUNTIME_MODELS_FILE="${RUNTIME_AGENT_DIR}/models.json"
RUNTIME_AUTH_PROFILES_FILE="${RUNTIME_AGENT_DIR}/auth-profiles.json"
RUNTIME_AGENT_CONFIG_FILE="${RUNTIME_AGENT_DIR}/agent-config.json"
GIT_WORKSPACE_ROOT="${USER_HOME}/git"

docker_cmd() {
  if docker info >/dev/null 2>&1; then
    env "MOLTBOX_RUNTIME_ROOT=${RUNTIME_ROOT}" docker "$@"
  else
    sudo env "MOLTBOX_RUNTIME_ROOT=${RUNTIME_ROOT}" docker "$@"
  fi
}

require_file() {
  local path="$1"
  if [[ ! -f "${path}" ]]; then
    log_error "Required file not found: ${path}"
    exit 1
  fi
}

require_host_cmd() {
  local cmd="$1"
  if ! command -v "${cmd}" >/dev/null 2>&1; then
    log_error "Required command not found on host: ${cmd}"
    exit 1
  fi
}

display_path() {
  local path="$1"
  if [[ "${path}" == "${RUNTIME_ROOT}"* ]]; then
    printf '%s\n' "~/.openclaw${path#"${RUNTIME_ROOT}"}"
    return
  fi
  printf '%s\n' "${path}"
}

copy_if_missing() {
  local source_path="$1"
  local dest_path="$2"
  local display_dest
  display_dest="$(display_path "${dest_path}")"

  require_file "${source_path}"
  mkdir -p "$(dirname "${dest_path}")"

  if [[ ! -f "${dest_path}" ]]; then
    log_info "Creating runtime config: ${display_dest}"
    cp "${source_path}" "${dest_path}"
  else
    log_info "Skipping existing runtime config: ${display_dest}"
  fi
}

enforce_runtime_outside_git_workspace() {
  if [[ "${RUNTIME_ROOT}" == "${GIT_WORKSPACE_ROOT}"* ]]; then
    log_error "Runtime root cannot be inside ~/git workspace: ${RUNTIME_ROOT}"
    log_error "Set MOLTBOX_RUNTIME_ROOT outside the git directory."
    exit 1
  fi
}

ensure_runtime_dirs() {
  mkdir -p "${RUNTIME_ROOT}"
  mkdir -p "${RUNTIME_AGENT_DIR}"
  mkdir -p "${RUNTIME_ROOT}/logs"
}

ensure_runtime_templates() {
  ensure_runtime_dirs

  copy_if_missing "${OPENCLAW_CONFIG_TEMPLATE}" "${RUNTIME_OPENCLAW_CONFIG_FILE}"
  copy_if_missing "${ENV_TEMPLATE}" "${RUNTIME_ENV_FILE}"
  copy_if_missing "${CONTAINER_ENV_TEMPLATE}" "${RUNTIME_CONTAINER_ENV_FILE}"
  copy_if_missing "${MODEL_RUNTIME_TEMPLATE}" "${RUNTIME_MODEL_RUNTIME_FILE}"
  copy_if_missing "${OPENSEARCH_TEMPLATE}" "${RUNTIME_OPENSEARCH_FILE}"
  copy_if_missing "${OPENCLAW_TEMPLATE_DIR}/agents.yaml" "${RUNTIME_ROOT}/agents.yaml"
  copy_if_missing "${OPENCLAW_TEMPLATE_DIR}/channels.yaml" "${RUNTIME_ROOT}/channels.yaml"
  copy_if_missing "${OPENCLAW_TEMPLATE_DIR}/routing.yaml" "${RUNTIME_ROOT}/routing.yaml"
  copy_if_missing "${OPENCLAW_TEMPLATE_DIR}/tools.yaml" "${RUNTIME_ROOT}/tools.yaml"
  copy_if_missing "${OPENCLAW_TEMPLATE_DIR}/escalation.yaml" "${RUNTIME_ROOT}/escalation.yaml"
}

remove_zero_length_file() {
  local path="$1"
  local label="$2"

  if [[ -f "${path}" && ! -s "${path}" ]]; then
    log_warn "Removing zero-byte ${label}: $(display_path "${path}")"
    rm -f "${path}"
  fi
}

remove_invalid_seeded_models_json() {
  if [[ ! -f "${RUNTIME_MODELS_FILE}" ]]; then
    return
  fi

  if grep -Eq '"(local_routing_model|cloud_reasoning_model|deep_thinking_model|coding_model)"' "${RUNTIME_MODELS_FILE}"; then
    local backup_path="${RUNTIME_MODELS_FILE}.moltbox-invalid-template.bak"
    if [[ ! -f "${backup_path}" ]]; then
      cp "${RUNTIME_MODELS_FILE}" "${backup_path}"
      log_warn "Backed up invalid seeded models.json to $(display_path "${backup_path}")"
    fi
    log_warn "Removing invalid seeded models.json so OpenClaw can regenerate it."
    rm -f "${RUNTIME_MODELS_FILE}"
  fi
}

reconcile_agent_runtime_files() {
  remove_zero_length_file "${RUNTIME_MODELS_FILE}" "models registry"
  remove_zero_length_file "${RUNTIME_AUTH_PROFILES_FILE}" "auth profile store"
  remove_zero_length_file "${RUNTIME_AGENT_CONFIG_FILE}" "agent config"
  remove_invalid_seeded_models_json
}

ensure_gateway_mode_local() {
  local tmp_file
  tmp_file="$(mktemp)"

  awk '
    BEGIN { seen=0 }
    /^OPENCLAW_GATEWAY_MODE=/ {
      if (seen == 0) {
        print "OPENCLAW_GATEWAY_MODE=local"
        seen=1
      }
      next
    }
    { print }
    END {
      if (seen == 0) {
        print "OPENCLAW_GATEWAY_MODE=local"
      }
    }
  ' "${RUNTIME_ENV_FILE}" > "${tmp_file}"

  mv "${tmp_file}" "${RUNTIME_ENV_FILE}"
  log_info "Ensured runtime config: $(display_path "${RUNTIME_ENV_FILE}") contains OPENCLAW_GATEWAY_MODE=local"
}

require_key() {
  local file="$1"
  local key="$2"
  if ! grep -Eq "^${key}=" "${file}"; then
    log_error "Missing key '${key}' in ${file}"
    exit 1
  fi
}

validate_required_keys() {
  require_key "${RUNTIME_ENV_FILE}" "OPENCLAW_IMAGE"
  require_key "${RUNTIME_ENV_FILE}" "OPENCLAW_GATEWAY_BIND"
  require_key "${RUNTIME_ENV_FILE}" "OPENCLAW_GATEWAY_TOKEN"
  require_key "${RUNTIME_ENV_FILE}" "OLLAMA_IMAGE"
  require_key "${RUNTIME_ENV_FILE}" "OPENSEARCH_IMAGE"
  require_key "${RUNTIME_ENV_FILE}" "GATEWAY_PORT"
  require_key "${RUNTIME_ENV_FILE}" "ESCALATION_MAX_TOKENS"
  require_key "${RUNTIME_ENV_FILE}" "ESCALATION_DAILY_USD_CAP"
  require_key "${RUNTIME_CONTAINER_ENV_FILE}" "LOCAL_ROUTING_MODEL"
  require_key "${RUNTIME_CONTAINER_ENV_FILE}" "OLLAMA_BASE_URL"
  require_key "${RUNTIME_CONTAINER_ENV_FILE}" "OPENSEARCH_URL"
  require_key "${RUNTIME_CONTAINER_ENV_FILE}" "CLOUD_PROVIDER"
}

ensure_gateway_token() {
  local token_value
  token_value="$(grep -E '^OPENCLAW_GATEWAY_TOKEN=' "${RUNTIME_ENV_FILE}" | head -n1 | cut -d= -f2-)"
  if [[ -z "${token_value}" || "${token_value}" == "CHANGE_ME_STRONG_TOKEN" ]]; then
    local generated_token=""
    if command -v openssl >/dev/null 2>&1; then
      generated_token="$(openssl rand -hex 24)"
    else
      generated_token="$(date +%s%N | sha256sum | cut -c1-48)"
    fi

    sed -i "s|^OPENCLAW_GATEWAY_TOKEN=.*$|OPENCLAW_GATEWAY_TOKEN=${generated_token}|" "${RUNTIME_ENV_FILE}"
    log_warn "OPENCLAW_GATEWAY_TOKEN was unset/placeholder. Generated and saved a strong token in ${RUNTIME_ENV_FILE}."
  fi
}

load_env() {
  # shellcheck disable=SC1090
  set -a
  source "${RUNTIME_ENV_FILE}"
  # shellcheck disable=SC1090
  source "${RUNTIME_CONTAINER_ENV_FILE}"
  set +a
}

compose() {
  docker_cmd compose --env-file "${RUNTIME_ENV_FILE}" -f "${COMPOSE_FILE}" "$@"
}

ensure_openclaw_image_available() {
  local image="${OPENCLAW_IMAGE}"
  if docker_cmd image inspect "${image}" >/dev/null 2>&1; then
    log_info "OpenClaw image already available locally: ${image}"
    return
  fi

  if [[ "${image}" == "openclaw:local" ]]; then
    log_error "OPENCLAW_IMAGE is '${image}', but that image is not present locally."
    log_error "Build it first from the OpenClaw repository, then rerun bootstrap."
    log_error "Example:"
    log_error "  git clone https://github.com/openclaw/openclaw.git"
    log_error "  cd openclaw"
    log_error "  docker build -t openclaw:local ."
    exit 1
  fi

  log_info "Pulling OpenClaw image: ${image}"
  compose pull openclaw
}

pull_images() {
  # Ollama/OpenSearch are always registry-based in this profile.
  log_info "Pulling Ollama and OpenSearch images."
  compose pull ollama opensearch
  ensure_openclaw_image_available
}

bring_up_stack() {
  pull_images

  log_info "Starting Moltbox stack using runtime config from ${RUNTIME_ROOT}."
  compose up -d --force-recreate
}

wait_for_ollama() {
  local sleep_seconds="${BOOTSTRAP_WAIT_INTERVAL_SECONDS:-2}"
  local wait_seconds="${BOOTSTRAP_OLLAMA_WAIT_SECONDS:-60}"
  local max_attempts=$(( (wait_seconds + sleep_seconds - 1) / sleep_seconds ))
  local attempt=1

  log_info "Waiting for Ollama service readiness (timeout: ${wait_seconds}s)."
  while (( attempt <= max_attempts )); do
    # Use native CLI check to avoid assuming curl exists in the image.
    if compose exec -T ollama ollama list >/dev/null 2>&1; then
      log_info "Ollama is ready."
      return 0
    fi
    log_warn "Ollama not ready yet (attempt ${attempt}/${max_attempts})."
    sleep "${sleep_seconds}"
    attempt=$((attempt + 1))
  done

  log_error "Ollama did not become ready."
  return 1
}

prepull_model() {
  local model="${LOCAL_ROUTING_MODEL:-qwen3:8b}"
  log_info "Pre-pulling local routing model: ${model}"
  compose exec -T ollama ollama pull "${model}"
}

show_discovery_diagnostics() {
  log_warn "OpenClaw model discovery diagnostics:"
  log_warn "--- openclaw models list ---"
  compose exec -T openclaw openclaw models list || true
  log_warn "--- /home/node/.openclaw/openclaw.json ---"
  compose exec -T openclaw sh -lc 'cat /home/node/.openclaw/openclaw.json' || true
  log_warn "--- curl http://ollama:11434/api/tags ---"
  compose exec -T openclaw sh -lc 'curl -fsS http://ollama:11434/api/tags' || true
}

assert_ollama_model_registered() {
  local model="${LOCAL_ROUTING_MODEL:-qwen3:8b}"
  local model_ref="ollama/${model}"
  local model_list=""

  if ! model_list="$(compose exec -T openclaw openclaw models list 2>&1)"; then
    log_error "Failed to read OpenClaw model registry."
    printf '%s\n' "${model_list}" >&2
    show_discovery_diagnostics
    return 1
  fi

  if ! awk -v ref="${model_ref}" '
    $1 == ref {
      found = 1
      if ($0 ~ /missing/) {
        missing = 1
      }
    }
    END {
      exit(found == 1 && missing != 1 ? 0 : 1)
    }
  ' <<<"${model_list}"; then
    log_error "OpenClaw did not register ${model_ref} in its model registry."
    printf '%s\n' "${model_list}" >&2
    show_discovery_diagnostics
    return 1
  fi

  log_info "Confirmed OpenClaw registered ${model_ref}."
}

configure_ollama_provider() {
  log_info "Configuring OpenClaw Ollama provider for Docker service DNS (http://ollama:11434)."
  compose exec -T openclaw openclaw config set models.providers.ollama.baseUrl '"http://ollama:11434"'
  compose exec -T openclaw openclaw config set models.providers.ollama.api '"ollama"'
  compose exec -T openclaw openclaw config set models.providers.ollama.apiKey '"ollama-local"'

  log_info "Triggering OpenClaw model discovery."
  if ! compose exec -T openclaw openclaw models discover; then
    log_warn "openclaw models discover failed; falling back to openclaw models list to force registry generation."
    if ! compose exec -T openclaw sh -lc 'openclaw models list >/dev/null'; then
      log_error "OpenClaw model discovery failed."
      show_discovery_diagnostics
      return 1
    fi
  fi

  assert_ollama_model_registered
}

wait_for_gateway() {
  local port="${GATEWAY_PORT:-18789}"
  local sleep_seconds="${BOOTSTRAP_WAIT_INTERVAL_SECONDS:-2}"
  local wait_seconds="${BOOTSTRAP_GATEWAY_WAIT_SECONDS:-80}"
  local max_attempts=$(( (wait_seconds + sleep_seconds - 1) / sleep_seconds ))
  local attempt=1

  log_info "Waiting for OpenClaw gateway health on http://127.0.0.1:${port}/healthz (timeout: ${wait_seconds}s)"
  while (( attempt <= max_attempts )); do
    if curl -fsS "http://127.0.0.1:${port}/healthz" >/dev/null 2>&1; then
      log_info "OpenClaw gateway is healthy."
      return 0
    fi
    log_warn "Gateway not ready yet (attempt ${attempt}/${max_attempts})."
    sleep "${sleep_seconds}"
    attempt=$((attempt + 1))
  done

  log_error "Gateway did not become healthy on port ${port}."
  return 1
}

main() {
  require_host_cmd docker
  require_host_cmd curl
  require_file "${COMPOSE_FILE}"
  enforce_runtime_outside_git_workspace
  log_info "Repository root: ${REPO_ROOT}"
  log_info "Runtime root: ${RUNTIME_ROOT}"
  ensure_runtime_templates
  reconcile_agent_runtime_files
  ensure_gateway_mode_local
  validate_required_keys
  ensure_gateway_token
  load_env

  bring_up_stack
  wait_for_ollama
  prepull_model
  wait_for_gateway
  configure_ollama_provider

  log_info "Runtime root: ${RUNTIME_ROOT}"
  log_info "Gateway token for first login: ${OPENCLAW_GATEWAY_TOKEN}"
  log_info "Open browser on your LAN: http://<MOLTBOX_HOST>:${GATEWAY_PORT:-18789}"
  log_info "Bootstrap complete."
}

main "$@"
