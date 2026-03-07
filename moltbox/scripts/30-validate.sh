#!/usr/bin/env bash
set -euo pipefail

# Moltbox validation script.
# Verifies service health, gateway readiness, internal connectivity, and exposure policy.

timestamp() { date +"%Y-%m-%dT%H:%M:%S%z"; }
log_info() { echo "[$(timestamp)] [INFO] $*"; }
log_warn() { echo "[$(timestamp)] [WARN] $*" >&2; }
log_error() { echo "[$(timestamp)] [ERROR] $*" >&2; }

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MOLTBOX_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
CONFIG_DIR="${MOLTBOX_DIR}/config"
COMPOSE_FILE="${CONFIG_DIR}/docker-compose.yml"

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
RUNTIME_ENV_FILE="${RUNTIME_ROOT}/.env"
RUNTIME_CONTAINER_ENV_FILE="${RUNTIME_ROOT}/container.env"
RUNTIME_AGENT_DIR="${RUNTIME_ROOT}/agents/main/agent"
RUNTIME_MODELS_FILE="${RUNTIME_AGENT_DIR}/models.json"
RUNTIME_AUTH_PROFILES_FILE="${RUNTIME_AGENT_DIR}/auth-profiles.json"
RUNTIME_AGENT_CONFIG_FILE="${RUNTIME_AGENT_DIR}/agent-config.json"

docker_cmd() {
  if docker info >/dev/null 2>&1; then
    env "MOLTBOX_RUNTIME_ROOT=${RUNTIME_ROOT}" docker "$@"
  else
    sudo env "MOLTBOX_RUNTIME_ROOT=${RUNTIME_ROOT}" docker "$@"
  fi
}

compose() {
  docker_cmd compose --env-file "${RUNTIME_ENV_FILE}" -f "${COMPOSE_FILE}" "$@"
}

require_host_cmd() {
  local cmd="$1"
  if ! command -v "${cmd}" >/dev/null 2>&1; then
    log_error "Required command not found on host: ${cmd}"
    exit 1
  fi
}

require_runtime_files() {
  [[ -f "${COMPOSE_FILE}" ]] || { log_error "Missing compose file: ${COMPOSE_FILE}"; exit 1; }
  [[ -f "${RUNTIME_ENV_FILE}" ]] || { log_error "Missing runtime env file: ${RUNTIME_ENV_FILE}"; exit 1; }
  [[ -f "${RUNTIME_CONTAINER_ENV_FILE}" ]] || { log_error "Missing runtime container env file: ${RUNTIME_CONTAINER_ENV_FILE}"; exit 1; }
  [[ -f "${RUNTIME_ROOT}/openclaw.json" ]] || { log_error "Missing runtime config: ${RUNTIME_ROOT}/openclaw.json"; exit 1; }
  [[ -f "${RUNTIME_ROOT}/agents.yaml" ]] || { log_error "Missing runtime config: ${RUNTIME_ROOT}/agents.yaml"; exit 1; }
  [[ -f "${RUNTIME_ROOT}/routing.yaml" ]] || { log_error "Missing runtime config: ${RUNTIME_ROOT}/routing.yaml"; exit 1; }
  [[ -f "${RUNTIME_ROOT}/tools.yaml" ]] || { log_error "Missing runtime config: ${RUNTIME_ROOT}/tools.yaml"; exit 1; }
  [[ -f "${RUNTIME_ROOT}/escalation.yaml" ]] || { log_error "Missing runtime config: ${RUNTIME_ROOT}/escalation.yaml"; exit 1; }
  [[ -f "${RUNTIME_ROOT}/channels.yaml" ]] || { log_error "Missing runtime config: ${RUNTIME_ROOT}/channels.yaml"; exit 1; }
  [[ -f "${RUNTIME_ROOT}/model-runtime.yml" ]] || { log_error "Missing runtime config: ${RUNTIME_ROOT}/model-runtime.yml"; exit 1; }
  [[ -f "${RUNTIME_ROOT}/opensearch.yml" ]] || { log_error "Missing runtime config: ${RUNTIME_ROOT}/opensearch.yml"; exit 1; }
  [[ -f "${RUNTIME_MODELS_FILE}" ]] || { log_error "Missing runtime config: ${RUNTIME_MODELS_FILE}"; exit 1; }
}

assert_file_not_empty_if_present() {
  local path="$1"
  local label="$2"
  if [[ -f "${path}" && ! -s "${path}" ]]; then
    log_error "${label} is zero bytes: ${path}"
    exit 1
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

ensure_service_running() {
  local service="$1"
  local cid
  cid="$(compose ps -q "${service}")"
  if [[ -z "${cid}" ]]; then
    log_error "Service '${service}' is not present."
    return 1
  fi
  local running
  running="$(docker_cmd inspect -f '{{.State.Running}}' "${cid}")"
  if [[ "${running}" != "true" ]]; then
    log_error "Service '${service}' is not running."
    return 1
  fi
}

ensure_service_healthy() {
  local service="$1"
  local cid
  cid="$(compose ps -q "${service}")"
  local health
  health="$(docker_cmd inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}' "${cid}")"
  if [[ "${health}" != "healthy" ]]; then
    log_error "Service '${service}' health is '${health}', expected 'healthy'."
    return 1
  fi
}

check_gateway() {
  local port="${GATEWAY_PORT:-18789}"
  log_info "Checking gateway health/readiness endpoints on port ${port}."
  curl -fsS "http://127.0.0.1:${port}/healthz" >/dev/null
  curl -fsS "http://127.0.0.1:${port}/readyz" >/dev/null
}

check_internal_connectivity() {
  log_info "Checking OpenClaw -> Ollama connectivity."
  compose exec -T openclaw node -e "const http=require('http');http.get('http://ollama:11434/api/tags',r=>{if(r.statusCode&&r.statusCode>=200&&r.statusCode<300){process.exit(0)}process.exit(1)}).on('error',()=>process.exit(1));"

  log_info "Checking OpenClaw -> OpenSearch connectivity."
  compose exec -T openclaw node -e "const http=require('http');http.get('http://opensearch:9200/_cluster/health',r=>{if(r.statusCode&&r.statusCode>=200&&r.statusCode<300){process.exit(0)}process.exit(1)}).on('error',()=>process.exit(1));"
}

assert_internal_only_ports() {
  log_info "Verifying Ollama/OpenSearch are not published on host."
  local ps_output
  ps_output="$(compose ps)"

  if grep -Eq '->11434/tcp|->9200/tcp' <<<"${ps_output}"; then
    log_error "Detected forbidden published port mapping for Ollama or OpenSearch."
    echo "${ps_output}" >&2
    return 1
  fi
}

check_agent_runtime_files() {
  assert_file_not_empty_if_present "${RUNTIME_MODELS_FILE}" "OpenClaw models.json"
  assert_file_not_empty_if_present "${RUNTIME_AUTH_PROFILES_FILE}" "OpenClaw auth-profiles.json"
  assert_file_not_empty_if_present "${RUNTIME_AGENT_CONFIG_FILE}" "OpenClaw agent-config.json"
}

check_ollama_model_registry() {
  local model="${LOCAL_ROUTING_MODEL:-qwen3:8b}"
  local model_ref="ollama/${model}"
  local model_list=""

  log_info "Checking OpenClaw model registry for ${model_ref}."
  if ! model_list="$(compose exec -T openclaw openclaw models list 2>&1)"; then
    log_error "Failed to read OpenClaw model registry."
    printf '%s\n' "${model_list}" >&2
    exit 1
  fi

  printf '%s\n' "${model_list}" | grep -F "ollama/" >/dev/null || {
    log_error "OpenClaw model registry does not contain any Ollama models. Provider configuration failed."
    printf '%s\n' "${model_list}" >&2
    exit 1
  }

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
    log_error "OpenClaw model registry did not register ${model_ref}. Ollama provider configuration or discovery failed."
    printf '%s\n' "${model_list}" >&2
    exit 1
  fi
}

main() {
  require_host_cmd docker
  require_host_cmd curl
  require_runtime_files
  load_env

  log_info "Validating compose service status using runtime root ${RUNTIME_ROOT}."
  compose ps

  ensure_service_running "openclaw"
  ensure_service_running "ollama"
  ensure_service_running "opensearch"

  ensure_service_healthy "openclaw"
  ensure_service_healthy "ollama"
  ensure_service_healthy "opensearch"

  check_gateway
  check_internal_connectivity
  check_agent_runtime_files
  check_ollama_model_registry
  assert_internal_only_ports

  log_info "Validation passed."
}

main "$@"
