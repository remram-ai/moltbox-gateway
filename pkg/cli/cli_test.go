package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseRuntimeOpenClawAgentNormalizesMessageFlag(t *testing.T) {
	t.Parallel()

	result := Parse([]string{
		"test",
		"openclaw",
		"agent",
		"--agent",
		"main",
		"--local",
		"--thinking",
		"off",
		"--message",
		"Say",
		"hello",
		"in",
		"one",
		"sentence.",
		"--json",
	})
	if result.Route == nil {
		t.Fatal("Parse() route = nil")
	}

	want := []string{"agent", "--agent", "main", "--local", "--thinking", "off", "--message", "Say hello in one sentence.", "--json"}
	if !equalArgs(result.Route.NativeArgs, want) {
		t.Fatalf("Parse() native_args = %#v, want %#v", result.Route.NativeArgs, want)
	}
}

func TestParseServiceContract(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		args        []string
		wantKind    string
		wantAction  string
		wantSubject string
	}{
		{
			name:       "service list",
			args:       []string{"service", "list"},
			wantKind:   KindService,
			wantAction: "list",
		},
		{
			name:        "service status",
			args:        []string{"service", "status", "test"},
			wantKind:    KindService,
			wantAction:  "status",
			wantSubject: "test",
		},
		{
			name:        "service deploy",
			args:        []string{"service", "deploy", "searxng"},
			wantKind:    KindService,
			wantAction:  "deploy",
			wantSubject: "searxng",
		},
		{
			name:        "service restart",
			args:        []string{"service", "restart", "caddy"},
			wantKind:    KindService,
			wantAction:  "restart",
			wantSubject: "caddy",
		},
		{
			name:        "service logs",
			args:        []string{"service", "logs", "searxng"},
			wantKind:    KindService,
			wantAction:  "logs",
			wantSubject: "searxng",
		},
		{
			name:        "service remove",
			args:        []string{"service", "remove", "test"},
			wantKind:    KindService,
			wantAction:  "remove",
			wantSubject: "test",
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			result := Parse(testCase.args)
			if result.Route == nil {
				t.Fatal("Parse() route = nil")
			}
			if result.Route.Kind != testCase.wantKind || result.Route.Action != testCase.wantAction || result.Route.Subject != testCase.wantSubject {
				t.Fatalf("Parse() route = %#v", result.Route)
			}
		})
	}
}

func TestParseRuntimeOpenClawAcrossEnvironments(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		args        []string
		wantEnv     string
		wantRuntime string
	}{
		{
			name:        "test openclaw",
			args:        []string{"test", "openclaw", "health", "--json"},
			wantEnv:     "test",
			wantRuntime: "openclaw-test",
		},
		{
			name:        "prod openclaw",
			args:        []string{"prod", "openclaw", "health", "--json"},
			wantEnv:     "prod",
			wantRuntime: "openclaw-prod",
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			result := Parse(testCase.args)
			if result.Route == nil {
				t.Fatal("Parse() route = nil")
			}
			if result.Route.Kind != KindRuntimeNative || result.Route.Action != "openclaw" {
				t.Fatalf("Parse() route = %#v, want runtime native openclaw route", result.Route)
			}
			if result.Route.Environment != testCase.wantEnv || result.Route.Runtime != testCase.wantRuntime {
				t.Fatalf("Parse() route = %#v, want env=%s runtime=%s", result.Route, testCase.wantEnv, testCase.wantRuntime)
			}
		})
	}
}

func TestParseSecretsSetJoinsInlineValue(t *testing.T) {
	t.Parallel()

	result := Parse([]string{"secret", "set", "test", "TEST_SECRET", "value", "with", "spaces"})
	if result.Route == nil {
		t.Fatal("Parse() route = nil")
	}
	if len(result.Route.NativeArgs) != 1 || result.Route.NativeArgs[0] != "value with spaces" {
		t.Fatalf("Parse() native_args = %#v, want [\"value with spaces\"]", result.Route.NativeArgs)
	}
}

func TestParseGatewayMCPStdio(t *testing.T) {
	t.Parallel()

	result := Parse([]string{"gateway", "mcp-stdio"})
	if result.Route == nil {
		t.Fatal("Parse() route = nil")
	}
	if result.Route.Kind != KindGatewayMCP || result.Route.Action != "mcp-stdio" {
		t.Fatalf("Parse() route = %#v, want gateway mcp-stdio route", result.Route)
	}
}

func TestParseBootstrapGateway(t *testing.T) {
	t.Parallel()

	result := Parse([]string{"bootstrap", "gateway"})
	if result.Route == nil {
		t.Fatal("Parse() route = nil")
	}
	if result.Route.Kind != KindBootstrap || result.Route.Action != "gateway" || result.Route.Subject != "gateway" {
		t.Fatalf("Parse() route = %#v, want bootstrap gateway route", result.Route)
	}
}

func TestParseRetiredNamespacesFailExplicitly(t *testing.T) {
	t.Parallel()

	retired := []string{
		"dev",
		"opensearch",
		"caddy",
		"runtime",
		"skill",
		"plugin",
		"tools",
		"host",
		"openclaw-dev",
		"openclaw-test",
		"openclaw-prod",
	}

	for _, value := range retired {
		value := value
		t.Run(value, func(t *testing.T) {
			t.Parallel()

			result := Parse([]string{value})
			if result.Code != ExitParseError {
				t.Fatalf("Parse() code = %d, want %d", result.Code, ExitParseError)
			}
			if result.Envelope == nil || result.Envelope.ErrorType != "retired_namespace" {
				t.Fatalf("Parse() envelope = %#v, want retired namespace", result.Envelope)
			}
		})
	}
}

func TestParseGatewayLegacySurfacesFailExplicitly(t *testing.T) {
	t.Parallel()

	tests := []struct {
		args []string
		want string
	}{
		{
			args: []string{"gateway", "service", "status", "test"},
			want: "'gateway service' is no longer the public service lifecycle surface",
		},
		{
			args: []string{"gateway", "docker", "ping"},
			want: "'gateway docker' is no longer part of the public CLI contract",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(strings.Join(test.args, " "), func(t *testing.T) {
			t.Parallel()

			result := Parse(test.args)
			if result.Envelope == nil {
				t.Fatal("Parse() envelope = nil")
			}
			if result.Envelope.ErrorType != "retired_namespace" {
				t.Fatalf("Parse() envelope = %#v, want retired_namespace", result.Envelope)
			}
			if result.Envelope.ErrorMessage != test.want {
				t.Fatalf("Parse() error_message = %q, want %q", result.Envelope.ErrorMessage, test.want)
			}
		})
	}
}

func TestWriteHelpServiceListsCurrentServices(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	if err := WriteHelp(&out, "service"); err != nil {
		t.Fatalf("WriteHelp(service): %v", err)
	}

	text := out.String()
	for _, needle := range []string{"gateway", "caddy", "ollama", "searxng", "test", "prod"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("service help missing %q in %q", needle, text)
		}
	}
}

func TestWriteHelpGlobalDocumentsLightweightSurface(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	if err := WriteHelp(&out, "global"); err != nil {
		t.Fatalf("WriteHelp(global): %v", err)
	}

	text := out.String()
	for _, needle := range []string{
		"bootstrap",
		"gateway",
		"service",
		"test|prod",
		"ollama",
		"secret",
		"gateway docker",
		"gateway service",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("global help missing %q in %q", needle, text)
		}
	}
}

func TestValidatePublicServiceRejectsGatewayMutations(t *testing.T) {
	t.Parallel()

	for _, action := range []string{"deploy", "restart", "remove"} {
		if errEnvelope := validatePublicService(action, "gateway"); errEnvelope == nil {
			t.Fatalf("validatePublicService(%q, gateway) = nil, want error", action)
		}
	}
}

func equalArgs(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
