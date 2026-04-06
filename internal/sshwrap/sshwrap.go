package sshwrap

import (
	"fmt"
	"path"
	"strings"
)

const (
	ModeTestOperator = "test-operator"
	ModeProdOperator = "prod-operator"
	ModeBootstrap    = "bootstrap"
)

// Resolve validates and tokenizes SSH_ORIGINAL_COMMAND without invoking a shell.
// It returns the argv after the leading "moltbox" token.
func Resolve(mode, raw string) ([]string, string, error) {
	tokens, err := split(raw)
	if err != nil {
		return nil, "", err
	}
	if len(tokens) == 0 {
		return nil, "expected a moltbox command", nil
	}
	if !isMoltboxCommandToken(tokens[0]) {
		return nil, "only moltbox commands are allowed", nil
	}

	args := append([]string(nil), tokens[1:]...)
	if len(args) == 0 {
		return nil, "missing moltbox arguments", nil
	}

	switch mode {
	case ModeTestOperator:
		return applyTestOperatorPolicy(args)
	case ModeProdOperator:
		return applyProdOperatorPolicy(args)
	case ModeBootstrap:
		return applyBootstrapPolicy(args)
	case "automation":
		return applyTestOperatorPolicy(args)
	default:
		return nil, "", fmt.Errorf("unsupported ssh wrapper mode %q", mode)
	}
}

func DenyPrefix(mode string) string {
	switch mode {
	case ModeTestOperator, "automation":
		return "test-operator"
	case ModeProdOperator:
		return "prod-operator"
	case ModeBootstrap:
		return "bootstrap"
	default:
		return "operator"
	}
}

func applyTestOperatorPolicy(args []string) ([]string, string, error) {
	switch args[0] {
	case "service":
		if len(args) == 2 && args[1] == "list" {
			return args, "", nil
		}
		if len(args) == 3 {
			switch args[1] {
			case "status", "logs":
				return args, "", nil
			case "deploy", "restart", "remove":
				switch args[2] {
				case "test", "ollama", "searxng":
					return args, "", nil
				}
				return nil, "test operator mutation is limited to test, ollama, and searxng", nil
			}
		}
		return nil, "service access is limited to list, status, logs, and test-lane mutation targets", nil
	case "gateway":
		if len(args) == 2 && (args[1] == "status" || args[1] == "logs" || args[1] == "mcp-stdio") {
			return args, "", nil
		}
		return nil, "gateway access is limited to status, logs, and mcp-stdio", nil
	case "test":
		if len(args) >= 3 {
			if args[1] == "openclaw" {
				return args, "", nil
			}
			if args[1] == "verify" {
				switch args[2] {
				case "runtime", "browser", "web":
					return args, "", nil
				}
			}
		}
		return nil, "test operator access is limited to native test OpenClaw commands and verify runtime|browser|web", nil
	case "ollama":
		if len(args) >= 2 {
			switch args[1] {
			case "list", "ps", "show":
				return args, "", nil
			}
		}
		return nil, "ollama access is limited to list, ps, and show", nil
	case "secret":
		if len(args) >= 3 && args[2] == "test" {
			return args, "", nil
		}
		return nil, "test operator secret access is limited to the test scope", nil
	case "prod":
		return nil, "test operator access does not include the prod runtime", nil
	default:
		return nil, "unsupported command", nil
	}
}

func applyProdOperatorPolicy(args []string) ([]string, string, error) {
	switch args[0] {
	case "gateway":
		if len(args) == 2 && (args[1] == "status" || args[1] == "logs" || args[1] == "mcp-stdio") {
			return args, "", nil
		}
		return nil, "gateway access is limited to status, logs, and mcp-stdio", nil
	case "service":
		if len(args) == 2 && args[1] == "list" {
			return args, "", nil
		}
		if len(args) == 3 && (args[1] == "status" || args[1] == "logs") {
			return args, "", nil
		}
		return nil, "prod operator service access is limited to list, status, and logs", nil
	case "prod":
		if len(args) >= 3 {
			if args[1] == "openclaw" {
				if isRuntimeMutationArgs(args[2:]) {
					return nil, "prod operator cannot run mutating OpenClaw lifecycle commands", nil
				}
				return args, "", nil
			}
			if args[1] == "verify" && args[2] == "runtime" {
				return args, "", nil
			}
		}
		return nil, "prod operator access is limited to non-mutating native prod OpenClaw commands and verify runtime", nil
	case "ollama":
		if len(args) >= 2 {
			switch args[1] {
			case "list", "ps", "show":
				return args, "", nil
			}
		}
		return nil, "ollama access is limited to list, ps, and show", nil
	case "secret":
		return nil, "secret access is not permitted for prod operator sessions", nil
	default:
		return nil, "unsupported command", nil
	}
}

func applyBootstrapPolicy(args []string) ([]string, string, error) {
	switch args[0] {
	case "bootstrap":
		if len(args) == 2 && args[1] == "gateway" {
			return args, "", nil
		}
		return nil, "bootstrap access is limited to 'bootstrap gateway'", nil
	case "gateway":
		if len(args) == 2 && (args[1] == "status" || args[1] == "logs") {
			return args, "", nil
		}
		return nil, "gateway access is limited to status and logs", nil
	case "service":
		if len(args) == 2 && args[1] == "list" {
			return args, "", nil
		}
		if len(args) == 3 && (args[1] == "status" || args[1] == "logs") {
			return args, "", nil
		}
		return nil, "service access is limited to list, status, and logs", nil
	case "test", "prod":
		if len(args) >= 3 && args[1] == "openclaw" {
			switch args[2] {
			case "status", "inspect", "logs", "health":
				return args, "", nil
			}
		}
		return nil, "test/prod access is limited to openclaw status, inspect, logs, and health", nil
	case "ollama":
		if len(args) >= 2 {
			switch args[1] {
			case "list", "ps", "show":
				return args, "", nil
			}
		}
		return nil, "ollama access is limited to list, ps, and show", nil
	case "secret":
		return nil, "secret access is not permitted for bootstrap sessions", nil
	default:
		return nil, "unsupported command", nil
	}
}

func isRuntimeMutationArgs(args []string) bool {
	if len(args) == 0 {
		return false
	}
	if isHelpLikeRuntimeArgs(args) || isDryRunRuntimeArgs(args) {
		return false
	}
	switch args[0] {
	case "backup":
		return len(args) >= 2 && args[1] == "restore"
	case "config":
		if len(args) < 2 {
			return false
		}
		switch args[1] {
		case "apply", "patch", "set", "unset", "restore":
			return true
		}
	case "plugins":
		if len(args) < 2 {
			return false
		}
		switch args[1] {
		case "install", "remove", "uninstall", "enable", "disable", "update":
			return true
		}
	case "skills":
		if len(args) < 2 {
			return false
		}
		switch args[1] {
		case "install", "remove", "enable", "disable":
			return true
		}
	case "reset":
		return true
	}
	return false
}

func isHelpLikeRuntimeArgs(args []string) bool {
	for _, arg := range args {
		switch strings.TrimSpace(arg) {
		case "-h", "--help", "help":
			return true
		}
	}
	return false
}

func isDryRunRuntimeArgs(args []string) bool {
	for _, arg := range args {
		switch strings.TrimSpace(arg) {
		case "--dry-run", "--check":
			return true
		}
	}
	return false
}

func split(raw string) ([]string, error) {
	var (
		args         []string
		current      strings.Builder
		tokenStarted bool
		inSingle     bool
		inDouble     bool
		escaped      bool
	)

	flush := func() {
		if !tokenStarted {
			return
		}
		args = append(args, current.String())
		current.Reset()
		tokenStarted = false
	}

	for _, r := range raw {
		switch {
		case escaped:
			current.WriteRune(r)
			tokenStarted = true
			escaped = false
		case inSingle:
			if r == '\'' {
				inSingle = false
				continue
			}
			current.WriteRune(r)
			tokenStarted = true
		case inDouble:
			switch r {
			case '"':
				inDouble = false
			case '\\':
				escaped = true
			default:
				current.WriteRune(r)
				tokenStarted = true
			}
		default:
			switch r {
			case ' ', '\t':
				flush()
			case '\'':
				inSingle = true
				tokenStarted = true
			case '"':
				inDouble = true
				tokenStarted = true
			case '\\':
				escaped = true
				tokenStarted = true
			case ';', '|', '&', '<', '>', '\n', '\r', '(', ')':
				return nil, fmt.Errorf("unsupported shell operator %q", string(r))
			default:
				current.WriteRune(r)
				tokenStarted = true
			}
		}
	}

	if escaped {
		return nil, fmt.Errorf("unterminated escape sequence")
	}
	if inSingle || inDouble {
		return nil, fmt.Errorf("unterminated quoted string")
	}

	flush()
	return args, nil
}

func isMoltboxCommandToken(token string) bool {
	if token == "moltbox" {
		return true
	}
	normalized := strings.ReplaceAll(strings.TrimSpace(token), `\`, `/`)
	if normalized == "" {
		return false
	}
	return path.Base(normalized) == "moltbox"
}
