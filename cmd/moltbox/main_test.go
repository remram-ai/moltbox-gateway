package main

import (
	"encoding/json"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/remram-ai/moltbox-gateway/pkg/cli"
)

func TestRecognizedRoutesReturnNotImplemented(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		args          []string
		wantKind      string
		wantAction    string
		wantSubject   string
		wantEnv       string
		wantRuntime   string
		wantNativeArg []string
	}{
		{
			name:       "gateway status",
			args:       []string{"gateway", "status"},
			wantKind:   cli.KindGateway,
			wantAction: "status",
		},
		{
			name:        "gateway service deploy",
			args:        []string{"gateway", "service", "deploy", "opensearch"},
			wantKind:    cli.KindGatewayService,
			wantAction:  "deploy",
			wantSubject: "opensearch",
		},
		{
			name:        "dev reload",
			args:        []string{"dev", "reload"},
			wantKind:    cli.KindRuntimeAction,
			wantAction:  "reload",
			wantEnv:     "dev",
			wantRuntime: "openclaw-dev",
		},
		{
			name:          "prod openclaw plugins list",
			args:          []string{"prod", "openclaw", "plugins", "list"},
			wantKind:      cli.KindRuntimeNative,
			wantAction:    "openclaw",
			wantEnv:       "prod",
			wantRuntime:   "openclaw-prod",
			wantNativeArg: []string{"plugins", "list"},
		},
		{
			name:          "ollama passthrough",
			args:          []string{"ollama", "ps"},
			wantKind:      cli.KindServiceNative,
			wantAction:    "passthrough",
			wantNativeArg: []string{"ps"},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var output strings.Builder
			code := run(testCase.args, &output, ioDiscard{})
			if code != cli.ExitNotImplemented {
				t.Fatalf("exit code = %d, want %d", code, cli.ExitNotImplemented)
			}

			var payload cli.Envelope
			if err := json.Unmarshal([]byte(output.String()), &payload); err != nil {
				t.Fatalf("decode payload: %v", err)
			}

			if payload.ErrorType != "not_implemented" {
				t.Fatalf("error_type = %q, want not_implemented", payload.ErrorType)
			}
			if payload.Route == nil {
				t.Fatal("route was nil")
			}
			if payload.Route.Kind != testCase.wantKind {
				t.Fatalf("route.kind = %q, want %q", payload.Route.Kind, testCase.wantKind)
			}
			if payload.Route.Action != testCase.wantAction {
				t.Fatalf("route.action = %q, want %q", payload.Route.Action, testCase.wantAction)
			}
			if payload.Route.Subject != testCase.wantSubject {
				t.Fatalf("route.subject = %q, want %q", payload.Route.Subject, testCase.wantSubject)
			}
			if payload.Route.Environment != testCase.wantEnv {
				t.Fatalf("route.environment = %q, want %q", payload.Route.Environment, testCase.wantEnv)
			}
			if payload.Route.Runtime != testCase.wantRuntime {
				t.Fatalf("route.runtime = %q, want %q", payload.Route.Runtime, testCase.wantRuntime)
			}
			if strings.Join(payload.Route.NativeArgs, "|") != strings.Join(testCase.wantNativeArg, "|") {
				t.Fatalf("route.native_args = %v, want %v", payload.Route.NativeArgs, testCase.wantNativeArg)
			}
		})
	}
}

func TestRetiredNamespacesFailExplicitly(t *testing.T) {
	t.Parallel()

	retired := []string{
		"runtime",
		"service",
		"skill",
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

			var output strings.Builder
			code := run([]string{value}, &output, ioDiscard{})
			if code != cli.ExitParseError {
				t.Fatalf("exit code = %d, want %d", code, cli.ExitParseError)
			}

			var payload cli.Envelope
			if err := json.Unmarshal([]byte(output.String()), &payload); err != nil {
				t.Fatalf("decode payload: %v", err)
			}
			if payload.ErrorType != "retired_namespace" {
				t.Fatalf("error_type = %q, want retired_namespace", payload.ErrorType)
			}
		})
	}
}

func TestUnknownResourceFails(t *testing.T) {
	t.Parallel()

	var output strings.Builder
	code := run([]string{"unknown"}, &output, ioDiscard{})
	if code != cli.ExitParseError {
		t.Fatalf("exit code = %d, want %d", code, cli.ExitParseError)
	}

	var payload cli.Envelope
	if err := json.Unmarshal([]byte(output.String()), &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.ErrorType != "parse_error" {
		t.Fatalf("error_type = %q, want parse_error", payload.ErrorType)
	}
}

func TestGatewayDockerPing(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "docker.sock")

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("listen unix socket: %v", err)
	}
	defer listener.Close()

	server := &http.Server{
		Handler: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if request.URL.Path != "/version" {
				http.NotFound(writer, request)
				return
			}
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{"Version":"29.3.0","ApiVersion":"1.48","MinAPIVersion":"1.24","GitCommit":"5927d80","GoVersion":"go1.24","Os":"linux","Arch":"amd64","KernelVersion":"6.8.0"}`))
		}),
	}

	go func() {
		_ = server.Serve(listener)
	}()
	defer server.Close()

	t.Setenv("MOLTBOX_DOCKER_SOCKET", socketPath)

	var output strings.Builder
	code := run([]string{"gateway", "docker", "ping"}, &output, ioDiscard{})
	if code != cli.ExitOK {
		t.Fatalf("exit code = %d, want %d", code, cli.ExitOK)
	}

	var payload cli.DockerPingResult
	if err := json.Unmarshal([]byte(output.String()), &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}

	if !payload.OK {
		t.Fatal("expected ok payload")
	}
	if payload.DockerVersion != "29.3.0" {
		t.Fatalf("docker_version = %q, want 29.3.0", payload.DockerVersion)
	}
	if payload.Route == nil || payload.Route.Kind != cli.KindGatewayDocker {
		t.Fatalf("route = %#v, want gateway docker route", payload.Route)
	}
}

func TestHelpAndVersion(t *testing.T) {
	t.Parallel()

	var helpOutput strings.Builder
	if code := run([]string{"--help"}, &helpOutput, ioDiscard{}); code != cli.ExitOK {
		t.Fatalf("help exit code = %d, want %d", code, cli.ExitOK)
	}
	if !strings.Contains(helpOutput.String(), "moltbox <resource> <command>") {
		t.Fatalf("help output missing grammar: %q", helpOutput.String())
	}

	var versionOutput strings.Builder
	if code := run([]string{"--version"}, &versionOutput, ioDiscard{}); code != cli.ExitOK {
		t.Fatalf("version exit code = %d, want %d", code, cli.ExitOK)
	}
	if !strings.Contains(versionOutput.String(), cli.Version) {
		t.Fatalf("version output missing version: %q", versionOutput.String())
	}
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) {
	return len(p), nil
}
