#!/bin/sh
set -eu

if [ "$#" -ne 4 ]; then
  printf '%s\n' "usage: provision-automation-ssh.sh <jason-codex.pub> <codex-bootstrap.pub> <cli-path> <bootstrap-wrapper-path>" >&2
  exit 2
fi

JASON_CODEX_KEY_PATH="$1"
BOOTSTRAP_KEY_PATH="$2"
CLI_PATH="$3"
BOOTSTRAP_WRAPPER_PATH="$4"

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

append_key_if_missing() {
  user="$1"
  entry="$2"
  auth_file="/home/$user/.ssh/authorized_keys"

  if ! grep -Fqx "$entry" "$auth_file"; then
    printf '%s\n' "$entry" >> "$auth_file"
  fi

  chown "$user:$user" "$auth_file"
  chmod 600 "$auth_file"
}

ensure_user "jason-codex"
ensure_user "codex-bootstrap"

jason_key="$(cat "$JASON_CODEX_KEY_PATH")"
bootstrap_key="$(cat "$BOOTSTRAP_KEY_PATH")"

jason_entry="command=\"$CLI_PATH gateway mcp-stdio\",restrict,no-port-forwarding,no-agent-forwarding,no-pty,no-X11-forwarding $jason_key"
bootstrap_entry="command=\"$BOOTSTRAP_WRAPPER_PATH\",restrict,no-port-forwarding,no-agent-forwarding,no-pty,no-X11-forwarding $bootstrap_key"

append_key_if_missing "jason-codex" "$jason_entry"
append_key_if_missing "codex-bootstrap" "$bootstrap_entry"
