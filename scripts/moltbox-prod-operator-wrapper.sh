#!/bin/sh
set -eu

CLI_PATH="__MOLTBOX_CLI_PATH__"

deny() {
  printf '%s\n' "prod-operator access denied: $1" >&2
  exit 126
}

if [ -z "${SSH_ORIGINAL_COMMAND:-}" ]; then
  deny "missing command"
fi

if [ "$(id -u)" -eq 0 ]; then
  exec "$CLI_PATH" "__ssh-wrapper=prod-operator" "$SSH_ORIGINAL_COMMAND"
fi

exec sudo -n "$CLI_PATH" "__ssh-wrapper=prod-operator" "$SSH_ORIGINAL_COMMAND"
