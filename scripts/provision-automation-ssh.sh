#!/bin/sh
set -eu

if [ "$#" -ne 5 ]; then
  printf '%s\n' "usage: provision-automation-ssh.sh <moltbox-ai-test.pub> <moltbox-ai-prod.pub> <cli-path> <test-wrapper-path> <prod-wrapper-path>" >&2
  exit 2
fi

TEST_KEY_PATH="$1"
PROD_KEY_PATH="$2"
CLI_PATH="$3"
TEST_WRAPPER_PATH="$4"
PROD_WRAPPER_PATH="$5"

ensure_user() {
  user="$1"
  home="/home/$user"

  if ! id "$user" >/dev/null 2>&1; then
    useradd --create-home --home-dir "$home" --shell /bin/sh "$user"
  fi

  passwd -l "$user" >/dev/null 2>&1 || usermod -L "$user" >/dev/null 2>&1 || true

  mkdir -p "$home/.ssh"
  chmod 700 "$home/.ssh"
  touch "$home/.ssh/authorized_keys"
  chmod 600 "$home/.ssh/authorized_keys"
  chown -R "$user:$user" "$home/.ssh"
}

upsert_key_entry() {
  user="$1"
  public_key="$2"
  entry="$3"
  auth_file="/home/$user/.ssh/authorized_keys"
  tmp_file="$(mktemp)"

  if [ -f "$auth_file" ]; then
    tr -d '\r' < "$auth_file" | grep -Fv "$public_key" > "$tmp_file" || true
  fi
  printf '%s\n' "$entry" >> "$tmp_file"
  mv "$tmp_file" "$auth_file"

  chown "$user:$user" "$auth_file"
  chmod 600 "$auth_file"
}

ensure_user "moltbox-ai-test"
ensure_user "moltbox-ai-prod"

test_key="$(tr -d '\r' < "$TEST_KEY_PATH")"
prod_key="$(tr -d '\r' < "$PROD_KEY_PATH")"

test_entry="command=\"$TEST_WRAPPER_PATH\",restrict,no-port-forwarding,no-agent-forwarding,no-pty,no-X11-forwarding $test_key"
prod_entry="command=\"$PROD_WRAPPER_PATH\",restrict,no-port-forwarding,no-agent-forwarding,no-pty,no-X11-forwarding $prod_key"

upsert_key_entry "moltbox-ai-test" "$test_key" "$test_entry"
upsert_key_entry "moltbox-ai-prod" "$prod_key" "$prod_entry"
