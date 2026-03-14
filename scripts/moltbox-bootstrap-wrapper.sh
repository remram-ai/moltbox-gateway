#!/bin/sh
set -eu

CLI_PATH="__MOLTBOX_CLI_PATH__"

deny() {
  printf '%s\n' "bootstrap access denied: $1" >&2
  exit 126
}

if [ -z "${SSH_ORIGINAL_COMMAND:-}" ]; then
  deny "missing command"
fi

set -- ${SSH_ORIGINAL_COMMAND}

if [ "$#" -lt 2 ]; then
  deny "expected a moltbox command"
fi

if [ "$1" != "moltbox" ]; then
  deny "only moltbox commands are allowed"
fi

shift

case "$1" in
  dev)
    exec "$CLI_PATH" "$@"
    ;;
  gateway)
    if [ "$#" -eq 2 ] && [ "$2" = "status" ]; then
      exec "$CLI_PATH" "$@"
    fi
    if [ "$#" -eq 2 ] && [ "$2" = "logs" ]; then
      exec "$CLI_PATH" "$@"
    fi
    if [ "$#" -eq 4 ] && [ "$2" = "service" ] && [ "$3" = "status" ]; then
      exec "$CLI_PATH" "$@"
    fi
    deny "gateway access is limited to status, logs, and service status"
    ;;
  test|prod)
    if [ "$#" -eq 2 ] && [ "$2" = "reload" ]; then
      deny "reload is not permitted for diagnostic-only environments"
    fi
    if [ "$#" -ge 3 ] && [ "$2" = "openclaw" ]; then
      case "$3" in
        status|inspect|logs|health)
          exec "$CLI_PATH" "$@"
          ;;
      esac
    fi
    if [ "$#" -eq 3 ] && [ "$2" = "secrets" ] && [ "$3" = "list" ]; then
      deny "secret access is not permitted for diagnostic-only environments"
    fi
    deny "test/prod access is limited to openclaw status, inspect, logs, and health"
    ;;
  *)
    deny "unsupported command"
    ;;
esac
