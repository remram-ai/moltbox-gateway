#!/usr/bin/env bash
set -euo pipefail

# Moltbox diagnostics bundle collector.
# Collects runtime state without mutating appliance configuration.

timestamp() { date +"%Y-%m-%dT%H:%M:%S%z"; }
log_info() { echo "[$(timestamp)] [INFO] $*"; }
log_warn() { echo "[$(timestamp)] [WARN] $*" >&2; }

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
OUT_DIR="${RUNTIME_ROOT}/logs/diagnostics/$(date +%Y%m%d-%H%M%S)"

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
    log_warn "Required command not found on host: ${cmd}"
    return 1
  fi
  return 0
}

capture_cmd() {
  local outfile="$1"
  shift
  {
    echo "# Command: $*"
    echo "# Timestamp: $(timestamp)"
    "$@"
  } >"${outfile}" 2>&1 || true
}

capture_shell() {
  local outfile="$1"
  local cmd="$2"
  {
    echo "# Command: ${cmd}"
    echo "# Timestamp: $(timestamp)"
    bash -lc "${cmd}"
  } >"${outfile}" 2>&1 || true
}

capture_inspect() {
  local service="$1"
  local outfile="$2"
  local cid=""

  cid="$(compose ps -q "${service}" 2>/dev/null || true)"
  if [[ -z "${cid}" ]]; then
    {
      echo "# Service: ${service}"
      echo "# Timestamp: $(timestamp)"
      echo "Container not found."
    } >"${outfile}"
    return
  fi

  capture_cmd "${outfile}" docker_cmd inspect "${cid}"
}

load_gateway_port() {
  local port="18789"
  if [[ -f "${RUNTIME_ENV_FILE}" ]]; then
    # shellcheck disable=SC1090
    source "${RUNTIME_ENV_FILE}"
    port="${GATEWAY_PORT:-18789}"
  fi
  echo "${port}"
}

main() {
  require_host_cmd docker || exit 1
  mkdir -p "${OUT_DIR}"
  log_info "Writing diagnostics to ${OUT_DIR}"

  capture_cmd "${OUT_DIR}/compose-ps.txt" compose ps
  capture_cmd "${OUT_DIR}/compose-config.txt" compose config
  capture_cmd "${OUT_DIR}/docker-network-ls.txt" docker_cmd network ls

  capture_cmd "${OUT_DIR}/logs-openclaw.txt" compose logs --tail=200 openclaw
  capture_cmd "${OUT_DIR}/logs-ollama.txt" compose logs --tail=200 ollama
  capture_cmd "${OUT_DIR}/logs-opensearch.txt" compose logs --tail=200 opensearch

  capture_shell "${OUT_DIR}/health-openclaw.txt" "curl -fsS http://127.0.0.1:$(load_gateway_port)/healthz"
  capture_shell "${OUT_DIR}/ready-openclaw.txt" "curl -fsS http://127.0.0.1:$(load_gateway_port)/readyz"

  capture_cmd "${OUT_DIR}/ollama-models.txt" compose exec -T ollama ollama list
  capture_cmd "${OUT_DIR}/opensearch-cluster-health.json" compose exec -T opensearch curl -sS http://127.0.0.1:9200/_cluster/health
  capture_cmd "${OUT_DIR}/opensearch-indices.txt" compose exec -T opensearch curl -sS http://127.0.0.1:9200/_cat/indices?v

  capture_inspect "openclaw" "${OUT_DIR}/inspect-openclaw.txt"
  capture_inspect "ollama" "${OUT_DIR}/inspect-ollama.txt"
  capture_inspect "opensearch" "${OUT_DIR}/inspect-opensearch.txt"

  if command -v nvidia-smi >/dev/null 2>&1; then
    capture_cmd "${OUT_DIR}/nvidia-smi.txt" nvidia-smi
  else
    log_warn "nvidia-smi not found; skipping GPU diagnostics."
  fi

  log_info "Diagnostics collection complete."
}

main "$@"
