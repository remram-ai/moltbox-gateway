package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

const (
	Version                  = "0.1.0-dev"
	DefaultDockerSocket      = "/var/run/docker.sock"
	DefaultGatewayURL        = "http://127.0.0.1:7460"
	DefaultGatewayListenAddr = ":7460"

	ExitOK             = 0
	ExitFailure        = 1
	ExitParseError     = 2
	ExitNotImplemented = 3
)

const (
	KindBootstrap      = "bootstrap"
	KindGateway        = "gateway"
	KindService        = "service"
	KindGatewayService = "gateway_service"
	KindGatewayMCP     = "gateway_mcp"
	KindGatewayToken   = "gateway_token"
	KindScopedSecrets  = "scoped_secrets"
	KindRuntimeAction  = "runtime_action"
	KindRuntimePlugin  = "runtime_plugin"
	KindRuntimeSkill   = "runtime_skill"
	KindRuntimeNative  = "runtime_openclaw"
	KindRuntimeVerify  = "runtime_verify"
	KindServiceNative  = "service_passthrough"
)

var retiredNamespaces = map[string]string{
	"dev":           "the appliance no longer provides a dev runtime; use local development or the test runtime",
	"opensearch":    "OpenSearch is removed from the appliance",
	"caddy":         "Caddy is managed through the service plane, not a native CLI passthrough",
	"runtime":       "the runtime namespace is retired",
	"skill":         "skill deployment is no longer a public Moltbox CLI surface",
	"plugin":        "plugin deployment is no longer a public Moltbox CLI surface",
	"tools":         "the tools namespace is retired",
	"host":          "the host namespace is retired",
	"openclaw-dev":  "internal runtime identifiers are not public CLI namespaces",
	"openclaw-test": "internal runtime identifiers are not public CLI namespaces",
	"openclaw-prod": "internal runtime identifiers are not public CLI namespaces",
}

var runtimeMappings = map[string]string{
	"test": "openclaw-test",
	"prod": "openclaw-prod",
}

var publicServices = map[string]string{
	"gateway":     "gateway",
	"caddy":       "caddy",
	"dev-sandbox": "dev-sandbox",
	"ollama":      "ollama",
	"searxng":     "searxng",
	"test":        "openclaw-test",
	"prod":        "openclaw-prod",
}

var secretScopes = map[string]struct{}{
	"service": {},
	"test":    {},
	"prod":    {},
}

type Route struct {
	Resource    string   `json:"resource"`
	Kind        string   `json:"kind"`
	Tokens      []string `json:"tokens,omitempty"`
	Action      string   `json:"action,omitempty"`
	Subject     string   `json:"subject,omitempty"`
	Environment string   `json:"environment,omitempty"`
	Runtime     string   `json:"runtime,omitempty"`
	NativeArgs  []string `json:"native_args,omitempty"`
}

type Envelope struct {
	OK              bool   `json:"ok"`
	Route           *Route `json:"route,omitempty"`
	ErrorType       string `json:"error_type,omitempty"`
	ErrorMessage    string `json:"error_message,omitempty"`
	RecoveryMessage string `json:"recovery_message,omitempty"`
}

type RouteRequest struct {
	Route       *Route `json:"route,omitempty"`
	Service     string `json:"service,omitempty"`
	SecretValue string `json:"secret_value,omitempty"`
}

type SecretSetRequest struct {
	Scope string `json:"scope"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

type SecretDeleteRequest struct {
	Scope string `json:"scope"`
	Name  string `json:"name"`
}

type GatewayHealthResult struct {
	OK      bool   `json:"ok"`
	Service string `json:"service"`
	Version string `json:"version"`
}

type GatewayStatusResult struct {
	OK            bool   `json:"ok"`
	Route         *Route `json:"route"`
	Service       string `json:"service"`
	Version       string `json:"version"`
	ListenAddress string `json:"listen_address"`
	DockerSocket  string `json:"docker_socket"`
}

type RepoSyncTargetResult struct {
	Target          string `json:"target"`
	RepoRoot        string `json:"repo_root,omitempty"`
	Source          string `json:"source,omitempty"`
	PreviousVersion string `json:"previous_version,omitempty"`
	NewVersion      string `json:"new_version,omitempty"`
	Changed         bool   `json:"changed"`
}

type RepoSyncResult struct {
	OK      bool                   `json:"ok"`
	Route   *Route                 `json:"route,omitempty"`
	Summary string                 `json:"summary,omitempty"`
	Command []string               `json:"command,omitempty"`
	Targets []RepoSyncTargetResult `json:"targets,omitempty"`
}

type ServiceStatusResult struct {
	OK              bool                     `json:"ok"`
	Route           *Route                   `json:"route"`
	Service         string                   `json:"service"`
	ServiceKind     string                   `json:"service_kind,omitempty"`
	Present         bool                     `json:"present"`
	ComposeProject  string                   `json:"compose_project,omitempty"`
	ContainerName   string                   `json:"container_name,omitempty"`
	Image           string                   `json:"image,omitempty"`
	ArtifactVersion string                   `json:"artifact_version,omitempty"`
	Status          string                   `json:"status,omitempty"`
	Running         bool                     `json:"running"`
	LogPath         string                   `json:"log_path,omitempty"`
	MetadataPath    string                   `json:"metadata_path,omitempty"`
	Containers      []ServiceContainerStatus `json:"containers,omitempty"`
}

type ServiceContainerStatus struct {
	Name          string `json:"name"`
	Present       bool   `json:"present"`
	ContainerName string `json:"container_name,omitempty"`
	Image         string `json:"image,omitempty"`
	Status        string `json:"status,omitempty"`
	Running       bool   `json:"running"`
	Health        string `json:"health,omitempty"`
}

type ServiceDeployResult struct {
	OK              bool                     `json:"ok"`
	Route           *Route                   `json:"route"`
	Service         string                   `json:"service"`
	ServiceKind     string                   `json:"service_kind,omitempty"`
	ComposeProject  string                   `json:"compose_project,omitempty"`
	OutputDir       string                   `json:"output_dir,omitempty"`
	ArtifactVersion string                   `json:"artifact_version,omitempty"`
	LogPath         string                   `json:"log_path,omitempty"`
	MetadataPath    string                   `json:"metadata_path,omitempty"`
	Command         []string                 `json:"command,omitempty"`
	Containers      []ServiceContainerStatus `json:"containers,omitempty"`
}

type ServiceActionResult struct {
	OK              bool                     `json:"ok"`
	Route           *Route                   `json:"route"`
	Service         string                   `json:"service"`
	ServiceKind     string                   `json:"service_kind,omitempty"`
	Action          string                   `json:"action"`
	ArtifactVersion string                   `json:"artifact_version,omitempty"`
	LogPath         string                   `json:"log_path,omitempty"`
	MetadataPath    string                   `json:"metadata_path,omitempty"`
	Command         []string                 `json:"command,omitempty"`
	Containers      []ServiceContainerStatus `json:"containers,omitempty"`
}

type ServiceListItem struct {
	Service         string `json:"service"`
	CanonicalName   string `json:"canonical_name,omitempty"`
	ServiceKind     string `json:"service_kind,omitempty"`
	Present         bool   `json:"present"`
	ComposeProject  string `json:"compose_project,omitempty"`
	ContainerName   string `json:"container_name,omitempty"`
	ArtifactVersion string `json:"artifact_version,omitempty"`
	Status          string `json:"status,omitempty"`
	Running         bool   `json:"running"`
	Health          string `json:"health,omitempty"`
}

type ServiceListResult struct {
	OK       bool              `json:"ok"`
	Route    *Route            `json:"route"`
	Services []ServiceListItem `json:"services"`
}

type RuntimeCheckpointResult struct {
	OK            bool   `json:"ok"`
	Route         *Route `json:"route"`
	Runtime       string `json:"runtime"`
	CheckpointID  string `json:"checkpoint_id"`
	Image         string `json:"image"`
	SnapshotDir   string `json:"snapshot_dir"`
	ReplayCleared bool   `json:"replay_cleared"`
}

type RuntimeSkillResult struct {
	OK             bool   `json:"ok"`
	Route          *Route `json:"route"`
	Runtime        string `json:"runtime"`
	Skill          string `json:"skill"`
	CanonicalSkill string `json:"canonical_skill"`
	Action         string `json:"action"`
	Message        string `json:"message,omitempty"`
	DeploymentID   string `json:"deployment_id,omitempty"`
	EventID        string `json:"event_id,omitempty"`
	PackageDir     string `json:"package_dir,omitempty"`
	ReplayCount    int    `json:"replay_count,omitempty"`
}

type RuntimePluginInfo struct {
	Plugin  string `json:"plugin"`
	Package string `json:"package,omitempty"`
	Version string `json:"version,omitempty"`
	Digest  string `json:"digest,omitempty"`
	Source  string `json:"source,omitempty"`
}

type RuntimePluginResult struct {
	OK           bool   `json:"ok"`
	Route        *Route `json:"route"`
	Runtime      string `json:"runtime"`
	Plugin       string `json:"plugin"`
	Package      string `json:"package,omitempty"`
	Version      string `json:"version,omitempty"`
	Digest       string `json:"digest,omitempty"`
	Source       string `json:"source,omitempty"`
	Action       string `json:"action"`
	Message      string `json:"message,omitempty"`
	DeploymentID string `json:"deployment_id,omitempty"`
	EventID      string `json:"event_id,omitempty"`
	PackageDir   string `json:"package_dir,omitempty"`
	SourcePath   string `json:"source_path,omitempty"`
	ReplayCount  int    `json:"replay_count,omitempty"`
}

type RuntimePluginListResult struct {
	OK      bool                `json:"ok"`
	Route   *Route              `json:"route"`
	Runtime string              `json:"runtime"`
	Plugins []RuntimePluginInfo `json:"plugins,omitempty"`
}

type CommandResult struct {
	OK            bool     `json:"ok"`
	Route         *Route   `json:"route,omitempty"`
	ContainerName string   `json:"container_name,omitempty"`
	Command       []string `json:"command,omitempty"`
	Stdout        string   `json:"stdout,omitempty"`
	Stderr        string   `json:"stderr,omitempty"`
	ExitCode      int      `json:"exit_code"`
}

type VerifyStepResult struct {
	Name          string            `json:"name"`
	OK            bool              `json:"ok"`
	Summary       string            `json:"summary,omitempty"`
	Command       []string          `json:"command,omitempty"`
	ExitCode      int               `json:"exit_code,omitempty"`
	StdoutSnippet string            `json:"stdout_snippet,omitempty"`
	StderrSnippet string            `json:"stderr_snippet,omitempty"`
	Details       map[string]string `json:"details,omitempty"`
}

type RuntimeVerifyResult struct {
	OK          bool               `json:"ok"`
	Route       *Route             `json:"route"`
	Environment string             `json:"environment"`
	Runtime     string             `json:"runtime"`
	Check       string             `json:"check"`
	TargetURL   string             `json:"target_url,omitempty"`
	Summary     string             `json:"summary,omitempty"`
	Caveats     []string           `json:"caveats,omitempty"`
	Steps       []VerifyStepResult `json:"steps"`
}

type SecretSetResult struct {
	OK     bool   `json:"ok"`
	Route  *Route `json:"route,omitempty"`
	Scope  string `json:"scope"`
	Name   string `json:"name"`
	Stored bool   `json:"stored"`
}

type SecretDeleteResult struct {
	OK      bool   `json:"ok"`
	Route   *Route `json:"route,omitempty"`
	Scope   string `json:"scope"`
	Name    string `json:"name"`
	Deleted bool   `json:"deleted"`
}

type SecretListItem struct {
	Scope string `json:"scope"`
	Name  string `json:"name"`
}

type SecretListResult struct {
	OK      bool             `json:"ok"`
	Route   *Route           `json:"route,omitempty"`
	Scope   string           `json:"scope,omitempty"`
	Secrets []SecretListItem `json:"secrets"`
}

type GatewayTokenInfo struct {
	Name string `json:"name"`
}

type GatewayTokenCreateResult struct {
	OK      bool   `json:"ok"`
	Route   *Route `json:"route,omitempty"`
	Name    string `json:"name"`
	Token   string `json:"token"`
	Created bool   `json:"created"`
}

type GatewayTokenRotateResult struct {
	OK      bool   `json:"ok"`
	Route   *Route `json:"route,omitempty"`
	Name    string `json:"name"`
	Token   string `json:"token"`
	Rotated bool   `json:"rotated"`
}

type GatewayTokenDeleteResult struct {
	OK      bool   `json:"ok"`
	Route   *Route `json:"route,omitempty"`
	Name    string `json:"name"`
	Deleted bool   `json:"deleted"`
}

type GatewayTokenListResult struct {
	OK     bool               `json:"ok"`
	Route  *Route             `json:"route,omitempty"`
	Tokens []GatewayTokenInfo `json:"tokens"`
}

type ParseResult struct {
	Route     *Route
	Envelope  *Envelope
	Code      int
	Help      bool
	HelpTopic string
	Version   bool
}

func Parse(args []string) ParseResult {
	if len(args) == 0 {
		return ParseResult{Help: true, HelpTopic: "global", Code: ExitOK}
	}

	if len(args) == 1 && isHelpFlag(args[0]) {
		return ParseResult{Help: true, HelpTopic: "global", Code: ExitOK}
	}

	if len(args) == 1 && args[0] == "--version" {
		return ParseResult{Version: true, Code: ExitOK}
	}

	if len(args) == 2 && isHelpFlag(args[1]) {
		if topic := normalizeHelpTopic(args[0]); topic != "" {
			return ParseResult{Help: true, HelpTopic: topic, Code: ExitOK}
		}
	}

	resource := args[0]
	if reason, retired := retiredNamespaces[resource]; retired {
		return retiredNamespaceResult(resource, reason)
	}

	switch resource {
	case "bootstrap":
		return parseBootstrap(args)
	case "gateway":
		return parseGateway(args)
	case "service":
		return parseService(args)
	case "test", "prod":
		return parseRuntimeOpenClaw(args)
	case "ollama":
		return parseServicePassthrough(args)
	case "secret":
		return parseSecret(args)
	default:
		return ParseResult{
			Envelope: Error(nil,
				"parse_error",
				fmt.Sprintf("unknown resource '%s'", resource),
				"use one of: bootstrap, gateway, service, test, prod, ollama, secret",
			),
			Code: ExitParseError,
		}
	}
}

func parseBootstrap(args []string) ParseResult {
	if len(args) != 2 || args[1] != "gateway" {
		return ParseResult{
			Envelope: Error(nil,
				"parse_error",
				"invalid bootstrap command",
				"use: bootstrap gateway",
			),
			Code: ExitParseError,
		}
	}

	return ParseResult{
		Route: &Route{
			Resource: "bootstrap",
			Kind:     KindBootstrap,
			Tokens:   append([]string(nil), args...),
			Action:   "gateway",
			Subject:  "gateway",
		},
	}
}

func parseGateway(args []string) ParseResult {
	if len(args) < 2 {
		return ParseResult{
			Envelope: Error(nil,
				"parse_error",
				"missing gateway command",
				"use: gateway status|update|logs|repo-sync|mcp-stdio",
			),
			Code: ExitParseError,
		}
	}

	switch args[1] {
	case "status", "update", "logs":
		if len(args) != 2 {
			return ParseResult{
				Envelope: Error(nil,
					"parse_error",
					fmt.Sprintf("unexpected arguments after 'gateway %s'", args[1]),
					fmt.Sprintf("use: gateway %s", args[1]),
				),
				Code: ExitParseError,
			}
		}
		return ParseResult{
			Route: &Route{
				Resource: "gateway",
				Kind:     KindGateway,
				Tokens:   append([]string(nil), args...),
				Action:   args[1],
			},
		}
	case "repo-sync":
		if len(args) < 3 {
			return ParseResult{
				Envelope: Error(nil,
					"parse_error",
					"missing repo-sync target",
					"use: gateway repo-sync services|runtime|all",
				),
				Code: ExitParseError,
			}
		}
		targets, errEnvelope := normalizeRepoSyncTargets(args[2:])
		if errEnvelope != nil {
			return ParseResult{Envelope: errEnvelope, Code: ExitParseError}
		}
		return ParseResult{
			Route: &Route{
				Resource:   "gateway",
				Kind:       KindGateway,
				Tokens:     append([]string(nil), args...),
				Action:     "repo-sync",
				NativeArgs: targets,
			},
		}
	case "mcp-stdio":
		if len(args) != 2 {
			return ParseResult{
				Envelope: Error(nil,
					"parse_error",
					"unexpected arguments after 'gateway mcp-stdio'",
					"use: gateway mcp-stdio",
				),
				Code: ExitParseError,
			}
		}
		return ParseResult{
			Route: &Route{
				Resource: "gateway",
				Kind:     KindGatewayMCP,
				Tokens:   append([]string(nil), args...),
				Action:   "mcp-stdio",
			},
		}
	case "service":
		return ParseResult{
			Envelope: Error(nil,
				"retired_namespace",
				"'gateway service' is no longer the public service lifecycle surface",
				"use: service list|status|deploy|restart|logs <service>",
			),
			Code: ExitParseError,
		}
	case "docker":
		return ParseResult{
			Envelope: Error(nil,
				"retired_namespace",
				"'gateway docker' is no longer part of the public CLI contract",
				"use the service plane or bootstrap gateway instead",
			),
			Code: ExitParseError,
		}
	case "token":
		return ParseResult{
			Envelope: Error(nil,
				"retired_namespace",
				"'gateway token' is no longer a public operator surface",
				"use the documented internal gateway token workflow instead",
			),
			Code: ExitParseError,
		}
	default:
		return ParseResult{
			Envelope: Error(nil,
				"parse_error",
				fmt.Sprintf("unknown gateway command '%s'", args[1]),
				"use: gateway status|update|logs|repo-sync|mcp-stdio",
			),
			Code: ExitParseError,
		}
	}
}

func parseService(args []string) ParseResult {
	if len(args) < 2 {
		return ParseResult{
			Envelope: Error(nil,
				"parse_error",
				"missing service command",
				"use: service list | service status <service> | service deploy <service> | service restart <service> | service remove <service> | service logs <service>",
			),
			Code: ExitParseError,
		}
	}

	switch args[1] {
	case "list":
		if len(args) != 2 {
			return ParseResult{
				Envelope: Error(nil,
					"parse_error",
					"unexpected arguments after 'service list'",
					"use: service list",
				),
				Code: ExitParseError,
			}
		}
		return ParseResult{
			Route: &Route{
				Resource: "service",
				Kind:     KindService,
				Tokens:   append([]string(nil), args...),
				Action:   "list",
			},
		}
	case "status", "deploy", "restart", "remove", "logs":
		if len(args) != 3 {
			return ParseResult{
				Envelope: Error(nil,
					"parse_error",
					fmt.Sprintf("invalid service %s command", args[1]),
					fmt.Sprintf("use: service %s <service>", args[1]),
				),
				Code: ExitParseError,
			}
		}
		service := strings.TrimSpace(args[2])
		if errEnvelope := validatePublicService(args[1], service); errEnvelope != nil {
			return ParseResult{Envelope: errEnvelope, Code: ExitParseError}
		}
		return ParseResult{
			Route: &Route{
				Resource: "service",
				Kind:     KindService,
				Tokens:   append([]string(nil), args...),
				Action:   args[1],
				Subject:  service,
			},
		}
	case "secrets":
		return ParseResult{
			Envelope: Error(nil,
				"retired_namespace",
				"'service secrets' is retired",
				"use: secret set <scope> <name> [value] | secret list <scope> | secret delete <scope> <name>",
			),
			Code: ExitParseError,
		}
	default:
		return ParseResult{
			Envelope: Error(nil,
				"parse_error",
				fmt.Sprintf("unknown service command '%s'", args[1]),
				"use: service list | service status <service> | service deploy <service> | service restart <service> | service remove <service> | service logs <service>",
			),
			Code: ExitParseError,
		}
	}
}

func parseRuntimeOpenClaw(args []string) ParseResult {
	if len(args) < 2 {
		return ParseResult{
			Envelope: Error(nil,
				"parse_error",
				fmt.Sprintf("missing command for environment '%s'", args[0]),
				runtimeUsage(args[0]),
			),
			Code: ExitParseError,
		}
	}

	route := &Route{
		Resource:    args[0],
		Tokens:      append([]string(nil), args...),
		Environment: args[0],
		Runtime:     runtimeMappings[args[0]],
	}

	switch args[1] {
	case "openclaw":
		if len(args) < 3 {
			return ParseResult{
				Envelope: Error(nil,
					"parse_error",
					fmt.Sprintf("missing native OpenClaw command for '%s'", args[0]),
					fmt.Sprintf("use: %s openclaw <command>", args[0]),
				),
				Code: ExitParseError,
			}
		}

		route.Kind = KindRuntimeNative
		route.Action = "openclaw"
		route.NativeArgs = normalizeRuntimeNativeArgs(args[2:])
		return ParseResult{Route: route}
	case "verify":
		if len(args) < 3 {
			return ParseResult{
				Envelope: Error(nil,
					"parse_error",
					fmt.Sprintf("missing verify check for '%s'", args[0]),
					verifyUsage(args[0]),
				),
				Code: ExitParseError,
			}
		}
		check := strings.TrimSpace(args[2])
		if errEnvelope := validateRuntimeVerifyTarget(args[0], check); errEnvelope != nil {
			return ParseResult{
				Envelope: errEnvelope,
				Code:     ExitParseError,
			}
		}

		route.Kind = KindRuntimeVerify
		route.Action = "verify"
		route.Subject = check
		if len(args) > 3 {
			route.NativeArgs = append([]string(nil), args[3:]...)
		}
		return ParseResult{Route: route}
	default:
		return ParseResult{
			Envelope: Error(nil,
				"parse_error",
				fmt.Sprintf("unknown environment command '%s'", args[1]),
				runtimeUsage(args[0]),
			),
			Code: ExitParseError,
		}
	}
}

func parseSecret(args []string) ParseResult {
	if len(args) < 3 {
		return ParseResult{
			Envelope: Error(nil,
				"parse_error",
				"missing secret command or scope",
				"use: secret set <scope> <name> [value] | secret list <scope> | secret delete <scope> <name>",
			),
			Code: ExitParseError,
		}
	}

	scope := strings.TrimSpace(args[2])
	if _, ok := secretScopes[scope]; !ok {
		return ParseResult{
			Envelope: Error(nil,
				"parse_error",
				fmt.Sprintf("unknown secret scope '%s'", scope),
				"use one of: service, test, prod",
			),
			Code: ExitParseError,
		}
	}

	switch args[1] {
	case "list":
		if len(args) != 3 {
			return ParseResult{
				Envelope: Error(nil,
					"parse_error",
					"unexpected arguments after 'secret list <scope>'",
					"use: secret list <scope>",
				),
				Code: ExitParseError,
			}
		}
		return ParseResult{
			Route: &Route{
				Resource: scope,
				Kind:     KindScopedSecrets,
				Tokens:   append([]string(nil), args...),
				Action:   "list",
			},
		}
	case "set":
		if len(args) < 4 {
			return ParseResult{
				Envelope: Error(nil,
					"parse_error",
					"invalid secret set command",
					"use: secret set <scope> <name> [value]",
				),
				Code: ExitParseError,
			}
		}
		nativeArgs := []string(nil)
		if len(args) >= 5 {
			nativeArgs = []string{strings.Join(args[4:], " ")}
		}
		return ParseResult{
			Route: &Route{
				Resource:   scope,
				Kind:       KindScopedSecrets,
				Tokens:     append([]string(nil), args...),
				Action:     "set",
				Subject:    args[3],
				NativeArgs: nativeArgs,
			},
		}
	case "delete":
		if len(args) != 4 {
			return ParseResult{
				Envelope: Error(nil,
					"parse_error",
					"invalid secret delete command",
					"use: secret delete <scope> <name>",
				),
				Code: ExitParseError,
			}
		}
		return ParseResult{
			Route: &Route{
				Resource: scope,
				Kind:     KindScopedSecrets,
				Tokens:   append([]string(nil), args...),
				Action:   "delete",
				Subject:  args[3],
			},
		}
	default:
		return ParseResult{
			Envelope: Error(nil,
				"parse_error",
				fmt.Sprintf("unknown secret command '%s'", args[1]),
				"use: secret set <scope> <name> [value] | secret list <scope> | secret delete <scope> <name>",
			),
			Code: ExitParseError,
		}
	}
}

func parseServicePassthrough(args []string) ParseResult {
	if len(args) < 2 {
		return ParseResult{
			Envelope: Error(nil,
				"parse_error",
				fmt.Sprintf("missing native command for service '%s'", args[0]),
				fmt.Sprintf("use: %s <native command>", args[0]),
			),
			Code: ExitParseError,
		}
	}

	return ParseResult{
		Route: &Route{
			Resource:   args[0],
			Kind:       KindServiceNative,
			Tokens:     append([]string(nil), args...),
			Action:     "passthrough",
			NativeArgs: append([]string(nil), args[1:]...),
		},
	}
}

func normalizeRuntimeNativeArgs(args []string) []string {
	if len(args) == 0 {
		return nil
	}

	switch args[0] {
	case "agent":
		return normalizeFlagTextValues(args, map[string]struct{}{
			"-m":         {},
			"--message":  {},
			"--reply-to": {},
		})
	default:
		return append([]string(nil), args...)
	}
}

func normalizeFlagTextValues(args []string, textFlags map[string]struct{}) []string {
	normalized := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		token := args[i]
		normalized = append(normalized, token)
		if _, ok := textFlags[token]; !ok {
			continue
		}
		if i+1 >= len(args) {
			continue
		}

		valueParts := []string{args[i+1]}
		j := i + 1
		for j+1 < len(args) && !looksLikeFlag(args[j+1]) {
			valueParts = append(valueParts, args[j+1])
			j++
		}
		normalized = append(normalized, strings.Join(valueParts, " "))
		i = j
	}
	return normalized
}

func looksLikeFlag(token string) bool {
	return strings.HasPrefix(token, "--") || (strings.HasPrefix(token, "-") && len(token) > 1)
}

func Error(route *Route, errorType, errorMessage, recoveryMessage string) *Envelope {
	return &Envelope{
		OK:              false,
		Route:           route,
		ErrorType:       errorType,
		ErrorMessage:    errorMessage,
		RecoveryMessage: recoveryMessage,
	}
}

func NotImplemented(route *Route, errorMessage, recoveryMessage string) *Envelope {
	return Error(route, "not_implemented", errorMessage, recoveryMessage)
}

func WriteJSON(out io.Writer, payload any) error {
	encoder := json.NewEncoder(out)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(payload)
}

func WriteHelp(out io.Writer, topic string) error {
	text, ok := helpTextByTopic[topic]
	if !ok {
		text = helpTextByTopic["global"]
	}
	_, err := io.WriteString(out, strings.TrimLeft(text, "\n"))
	return err
}

func WriteVersion(out io.Writer) error {
	_, err := fmt.Fprintf(out, "moltbox %s\n", Version)
	return err
}

func DockerSocketPath() string {
	if value := strings.TrimSpace(os.Getenv("MOLTBOX_DOCKER_SOCKET")); value != "" {
		return value
	}
	return DefaultDockerSocket
}

func GatewayURL() string {
	if value := strings.TrimSpace(os.Getenv("MOLTBOX_GATEWAY_URL")); value != "" {
		return strings.TrimRight(value, "/")
	}
	return DefaultGatewayURL
}

func GatewayListenAddress() string {
	if value := strings.TrimSpace(os.Getenv("MOLTBOX_GATEWAY_LISTEN_ADDR")); value != "" {
		return value
	}
	return DefaultGatewayListenAddr
}

func ExitCodeFromPayload(payload []byte) int {
	var envelope Envelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return ExitFailure
	}

	if envelope.OK {
		return ExitOK
	}

	switch envelope.ErrorType {
	case "not_implemented":
		return ExitNotImplemented
	case "parse_error", "retired_namespace":
		return ExitParseError
	default:
		return ExitFailure
	}
}

func isHelpFlag(value string) bool {
	return value == "-h" || value == "--help"
}

func normalizeHelpTopic(value string) string {
	switch strings.TrimSpace(value) {
	case "bootstrap", "gateway", "service", "test", "prod", "ollama", "secret":
		return strings.TrimSpace(value)
	default:
		return ""
	}
}

func retiredNamespaceResult(resource, reason string) ParseResult {
	return ParseResult{
		Envelope: Error(nil,
			"retired_namespace",
			fmt.Sprintf("'%s' is a retired top-level namespace", resource),
			reason,
		),
		Code: ExitParseError,
	}
}

func validatePublicService(action, service string) *Envelope {
	if _, ok := publicServices[service]; !ok {
		return Error(nil,
			"parse_error",
			fmt.Sprintf("unknown service '%s'", service),
			"use one of: gateway, caddy, dev-sandbox, ollama, searxng, test, prod",
		)
	}
	if service == "gateway" && (action == "deploy" || action == "restart" || action == "remove") {
		return Error(nil,
			"parse_error",
			fmt.Sprintf("service %s gateway is not supported", action),
			"use: gateway update",
		)
	}
	return nil
}

func runtimeUsage(environment string) string {
	return fmt.Sprintf("use: %s openclaw <command> | %s verify <check>", environment, environment)
}

func verifyUsage(environment string) string {
	switch environment {
	case "test":
		return "use: test verify runtime | test verify browser [url] | test verify web | test verify sandbox"
	case "prod":
		return "use: prod verify runtime"
	default:
		return fmt.Sprintf("use: %s verify <check>", environment)
	}
}

func validateRuntimeVerifyTarget(environment, check string) *Envelope {
	switch environment {
	case "test":
		switch check {
		case "runtime", "browser", "web", "sandbox":
			return nil
		}
		return Error(nil,
			"parse_error",
			fmt.Sprintf("unknown test verify check '%s'", check),
			verifyUsage(environment),
		)
	case "prod":
		if check == "runtime" {
			return nil
		}
		return Error(nil,
			"parse_error",
			fmt.Sprintf("unsupported prod verify check '%s'", check),
			verifyUsage(environment),
		)
	default:
		return Error(nil,
			"parse_error",
			fmt.Sprintf("unknown environment '%s'", environment),
			verifyUsage(environment),
		)
	}
}

func normalizeRepoSyncTargets(values []string) ([]string, *Envelope) {
	ordered := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		switch strings.TrimSpace(value) {
		case "all":
			values = []string{"services", "runtime"}
			ordered = ordered[:0]
			seen = map[string]struct{}{}
		case "services", "runtime":
		default:
			return nil, Error(nil,
				"parse_error",
				fmt.Sprintf("unknown repo-sync target '%s'", value),
				"use: gateway repo-sync services|runtime|all",
			)
		}
	}
	for _, value := range values {
		target := strings.TrimSpace(value)
		if target == "all" {
			continue
		}
		if _, ok := seen[target]; ok {
			continue
		}
		seen[target] = struct{}{}
		ordered = append(ordered, target)
	}
	if len(ordered) == 0 {
		return nil, Error(nil,
			"parse_error",
			"missing repo-sync target",
			"use: gateway repo-sync services|runtime|all",
		)
	}
	return ordered, nil
}

var helpTextByTopic = map[string]string{
	"global": `
moltbox <resource> <command>

Resources:
  bootstrap
    gateway

  gateway
    status
    update
    logs
    repo-sync services|runtime|all
    mcp-stdio

  service
    list
    status <service>
    deploy <service>
    restart <service>
    remove <service>
    logs <service>

  test|prod
    openclaw <command>
    verify <check>

  ollama
    <native command>

  secret
    set <scope> <name> [value]
    list <scope>
    delete <scope> <name>

Removed or retired surfaces fail explicitly:
  dev, opensearch, caddy, runtime, skill, plugin, tools, host,
  openclaw-dev, openclaw-test, openclaw-prod, gateway service, gateway docker
`,
	"bootstrap": `
moltbox bootstrap gateway

Bootstrap:
  gateway   Start or recover the local gateway control plane
`,
	"gateway": `
moltbox gateway <command>

Commands:
  status
  update
  logs
  repo-sync services|runtime|all
  mcp-stdio
`,
	"service": `
moltbox service <command>

Commands:
  list
  status <service>
  deploy <service>
  restart <service>
  remove <service>
  logs <service>

Services:
  gateway
  caddy
  dev-sandbox
  ollama
  searxng
  test
  prod
`,
	"test": `
moltbox test <command>

Commands:
  openclaw <command>
  verify runtime
  verify browser [url]
  verify web
  verify sandbox
`,
	"prod": `
moltbox prod <command>

Commands:
  openclaw <command>
  verify runtime
`,
	"ollama": `
moltbox ollama <native command>

This is a thin passthrough to the native Ollama CLI inside the managed service.
`,
	"secret": `
moltbox secret <command>

Commands:
  set <scope> <name> [value]
  list <scope>
  delete <scope> <name>

Scopes:
  service
  test
  prod
`,
}
