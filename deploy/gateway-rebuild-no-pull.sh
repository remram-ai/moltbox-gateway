#!/usr/bin/env bash
set -euo pipefail

REPO="${REPO:-/opt/moltbox/repos/moltbox-gateway}"
STAGING_ROOT="${STAGING_ROOT:-/srv/moltbox-state/updates/gateway}"
CLI_PATH="${CLI_PATH:-/usr/local/bin/moltbox}"
CLI_CONFIG_PATH="${CLI_CONFIG_PATH:-/etc/moltbox/config.yaml}"
CONFIG_SOURCE="${CONFIG_SOURCE:-/srv/moltbox-state/services/gateway/config/gateway/config.yaml}"
GATEWAY_OUTPUT_DIR="${GATEWAY_OUTPUT_DIR:-/srv/moltbox-state/services/gateway}"
COMPOSE_PROJECT="${COMPOSE_PROJECT:-gateway}"
SECRETS_ROOT="${SECRETS_ROOT:-/var/lib/moltbox/secrets}"
TEST_WRAPPER_SOURCE="${TEST_WRAPPER_SOURCE:-$REPO/scripts/moltbox-test-operator-wrapper.sh}"
TEST_WRAPPER_PATH="${TEST_WRAPPER_PATH:-/usr/local/bin/moltbox-test-operator-wrapper}"
PROD_WRAPPER_SOURCE="${PROD_WRAPPER_SOURCE:-$REPO/scripts/moltbox-prod-operator-wrapper.sh}"
PROD_WRAPPER_PATH="${PROD_WRAPPER_PATH:-/usr/local/bin/moltbox-prod-operator-wrapper}"
BOOTSTRAP_WRAPPER_SOURCE="${BOOTSTRAP_WRAPPER_SOURCE:-$REPO/scripts/moltbox-bootstrap-wrapper.sh}"
BOOTSTRAP_WRAPPER_PATH="${BOOTSTRAP_WRAPPER_PATH:-/usr/local/bin/moltbox-bootstrap-wrapper}"
SHARED_CLI_PATH="${SHARED_CLI_PATH:-/usr/local/bin/moltbox}"
SYSTEM_CONFIG_PATH="${SYSTEM_CONFIG_PATH:-/etc/moltbox/config.yaml}"
HISTORY_PATH="${HISTORY_PATH:-/var/lib/moltbox/history.jsonl}"

mkdir -p "$STAGING_ROOT" "$(dirname "$CLI_PATH")" "$(dirname "$CLI_CONFIG_PATH")" "$(dirname "$SYSTEM_CONFIG_PATH")"
mkdir -p "$SECRETS_ROOT" "$(dirname "$HISTORY_PATH")"

docker run --rm -v "$REPO:/src" -v "$STAGING_ROOT:/out" -w /src golang:1.23-bookworm sh -lc \
  'set -eu; /usr/local/go/bin/go build -buildvcs=false -o /out/moltbox ./cmd/moltbox && /usr/local/go/bin/go build -buildvcs=false -o /out/gateway ./cmd/gateway'

install_cli() {
  local target="$1"
  local tmp="${target}.tmp.$$"
  cp "$STAGING_ROOT/moltbox" "$tmp"
  chmod 0755 "$tmp"
  mv -f "$tmp" "$target"
}

install_cli "$CLI_PATH"
if [ "$SHARED_CLI_PATH" != "$CLI_PATH" ]; then
  install_cli "$SHARED_CLI_PATH"
fi

cp "$CONFIG_SOURCE" "$CLI_CONFIG_PATH"
chmod 0644 "$CLI_CONFIG_PATH"
cp "$CONFIG_SOURCE" "$SYSTEM_CONFIG_PATH"
chmod 0644 "$SYSTEM_CONFIG_PATH"

sed "s|__MOLTBOX_CLI_PATH__|$SHARED_CLI_PATH|g" "$TEST_WRAPPER_SOURCE" > "$TEST_WRAPPER_PATH"
chmod 0755 "$TEST_WRAPPER_PATH"
sed "s|__MOLTBOX_CLI_PATH__|$SHARED_CLI_PATH|g" "$PROD_WRAPPER_SOURCE" > "$PROD_WRAPPER_PATH"
chmod 0755 "$PROD_WRAPPER_PATH"
sed "s|__MOLTBOX_CLI_PATH__|$SHARED_CLI_PATH|g" "$BOOTSTRAP_WRAPPER_SOURCE" > "$BOOTSTRAP_WRAPPER_PATH"
chmod 0755 "$BOOTSTRAP_WRAPPER_PATH"

CLI_OWNER="$(stat -c '%u:%g' "$(dirname "$CLI_PATH")")"
chown -R "$CLI_OWNER" "$SECRETS_ROOT"
find "$SECRETS_ROOT" -type d -exec chmod 0700 {} +
find "$SECRETS_ROOT" -type f -name '*.json' -exec chmod 0600 {} +
if [ -f "$SECRETS_ROOT/master.key" ]; then
  chmod 0600 "$SECRETS_ROOT/master.key"
fi

docker build -t moltbox-gateway:latest "$REPO"
docker rm -f gateway >/dev/null 2>&1 || true
cd "$GATEWAY_OUTPUT_DIR"
docker compose -f compose.yml -p "$COMPOSE_PROJECT" up -d --remove-orphans

echo "gateway rebuild/install complete"
