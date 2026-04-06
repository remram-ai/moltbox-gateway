package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/remram-ai/moltbox-gateway/pkg/cli"
)

func TestCLIForwardsLightweightPublicSurface(t *testing.T) {
	testCases := []struct {
		name       string
		args       []string
		wantMethod string
		wantPath   string
		wantCode   int
		handler    func(t *testing.T, writer http.ResponseWriter, request *http.Request)
	}{
		{
			name:       "gateway status",
			args:       []string{"gateway", "status"},
			wantMethod: http.MethodGet,
			wantPath:   "/status",
			wantCode:   cli.ExitOK,
			handler: func(t *testing.T, writer http.ResponseWriter, request *http.Request) {
				t.Helper()
				_ = json.NewEncoder(writer).Encode(cli.GatewayStatusResult{
					OK:            true,
					Route:         &cli.Route{Resource: "gateway", Kind: cli.KindGateway, Action: "status"},
					Service:       "gateway",
					Version:       cli.Version,
					ListenAddress: ":7460",
					DockerSocket:  cli.DefaultDockerSocket,
				})
			},
		},
		{
			name:       "gateway logs",
			args:       []string{"gateway", "logs"},
			wantMethod: http.MethodGet,
			wantPath:   "/logs",
			wantCode:   cli.ExitOK,
			handler: func(t *testing.T, writer http.ResponseWriter, request *http.Request) {
				t.Helper()
				_ = json.NewEncoder(writer).Encode(cli.CommandResult{
					OK:            true,
					Route:         &cli.Route{Resource: "gateway", Kind: cli.KindGateway, Action: "logs"},
					ContainerName: "gateway",
					ExitCode:      0,
					Stdout:        "gateway log line",
				})
			},
		},
		{
			name:       "service list",
			args:       []string{"service", "list"},
			wantMethod: http.MethodGet,
			wantPath:   "/service/list",
			wantCode:   cli.ExitOK,
			handler: func(t *testing.T, writer http.ResponseWriter, request *http.Request) {
				t.Helper()
				_ = json.NewEncoder(writer).Encode(cli.ServiceListResult{
					OK:    true,
					Route: &cli.Route{Resource: "service", Kind: cli.KindService, Action: "list"},
					Services: []cli.ServiceListItem{
						{Service: "gateway", CanonicalName: "gateway", Running: true, Health: "healthy"},
						{Service: "searxng", CanonicalName: "searxng", Running: true, Health: "healthy"},
						{Service: "test", CanonicalName: "openclaw-test", Running: true, Health: "healthy"},
					},
				})
			},
		},
		{
			name:       "service status",
			args:       []string{"service", "status", "test"},
			wantMethod: http.MethodGet,
			wantPath:   "/service/status",
			wantCode:   cli.ExitOK,
			handler: func(t *testing.T, writer http.ResponseWriter, request *http.Request) {
				t.Helper()
				if got := request.URL.Query().Get("service"); got != "test" {
					t.Fatalf("query service = %q, want test", got)
				}
				_ = json.NewEncoder(writer).Encode(cli.ServiceStatusResult{
					OK:      true,
					Route:   &cli.Route{Resource: "service", Kind: cli.KindService, Action: "status", Subject: "test"},
					Service: "test",
					Status:  "running",
					Running: true,
				})
			},
		},
		{
			name:       "service deploy",
			args:       []string{"service", "deploy", "test"},
			wantMethod: http.MethodPost,
			wantPath:   "/service/deploy",
			wantCode:   cli.ExitOK,
			handler: func(t *testing.T, writer http.ResponseWriter, request *http.Request) {
				t.Helper()
				var payload cli.RouteRequest
				if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
					t.Fatalf("decode request: %v", err)
				}
				if payload.Service != "test" {
					t.Fatalf("payload.service = %q, want test", payload.Service)
				}
				_ = json.NewEncoder(writer).Encode(cli.ServiceDeployResult{
					OK:      true,
					Route:   payload.Route,
					Service: "test",
				})
			},
		},
		{
			name:       "service restart",
			args:       []string{"service", "restart", "searxng"},
			wantMethod: http.MethodPost,
			wantPath:   "/service/restart",
			wantCode:   cli.ExitOK,
			handler: func(t *testing.T, writer http.ResponseWriter, request *http.Request) {
				t.Helper()
				var payload cli.RouteRequest
				if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
					t.Fatalf("decode request: %v", err)
				}
				if payload.Service != "searxng" {
					t.Fatalf("payload.service = %q, want searxng", payload.Service)
				}
				_ = json.NewEncoder(writer).Encode(cli.ServiceActionResult{
					OK:      true,
					Route:   payload.Route,
					Service: "searxng",
					Action:  "restart",
				})
			},
		},
		{
			name:       "service remove",
			args:       []string{"service", "remove", "searxng"},
			wantMethod: http.MethodPost,
			wantPath:   "/service/remove",
			wantCode:   cli.ExitOK,
			handler: func(t *testing.T, writer http.ResponseWriter, request *http.Request) {
				t.Helper()
				var payload cli.RouteRequest
				if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
					t.Fatalf("decode request: %v", err)
				}
				if payload.Service != "searxng" {
					t.Fatalf("payload.service = %q, want searxng", payload.Service)
				}
				_ = json.NewEncoder(writer).Encode(cli.ServiceActionResult{
					OK:      true,
					Route:   payload.Route,
					Service: "searxng",
					Action:  "remove",
				})
			},
		},
		{
			name:       "service logs",
			args:       []string{"service", "logs", "prod"},
			wantMethod: http.MethodGet,
			wantPath:   "/service/logs",
			wantCode:   cli.ExitOK,
			handler: func(t *testing.T, writer http.ResponseWriter, request *http.Request) {
				t.Helper()
				if got := request.URL.Query().Get("service"); got != "prod" {
					t.Fatalf("query service = %q, want prod", got)
				}
				_ = json.NewEncoder(writer).Encode(cli.CommandResult{
					OK:            true,
					Route:         &cli.Route{Resource: "service", Kind: cli.KindService, Action: "logs", Subject: "prod"},
					ContainerName: "openclaw-prod",
					ExitCode:      0,
					Stdout:        "prod log line",
				})
			},
		},
		{
			name:       "test openclaw passthrough",
			args:       []string{"test", "openclaw", "health", "--json"},
			wantMethod: http.MethodPost,
			wantPath:   "/runtime/openclaw",
			wantCode:   cli.ExitOK,
			handler: func(t *testing.T, writer http.ResponseWriter, request *http.Request) {
				t.Helper()
				var payload cli.RouteRequest
				if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
					t.Fatalf("decode request: %v", err)
				}
				if payload.Route == nil || payload.Route.Environment != "test" || payload.Route.Runtime != "openclaw-test" {
					t.Fatalf("payload.route = %#v, want test runtime route", payload.Route)
				}
				_ = json.NewEncoder(writer).Encode(cli.CommandResult{
					OK:            true,
					Route:         payload.Route,
					ContainerName: "openclaw-test",
					ExitCode:      0,
					Stdout:        `{"ok":true}`,
				})
			},
		},
		{
			name:       "prod openclaw passthrough",
			args:       []string{"prod", "openclaw", "health", "--json"},
			wantMethod: http.MethodPost,
			wantPath:   "/runtime/openclaw",
			wantCode:   cli.ExitOK,
			handler: func(t *testing.T, writer http.ResponseWriter, request *http.Request) {
				t.Helper()
				var payload cli.RouteRequest
				if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
					t.Fatalf("decode request: %v", err)
				}
				if payload.Route == nil || payload.Route.Environment != "prod" || payload.Route.Runtime != "openclaw-prod" {
					t.Fatalf("payload.route = %#v, want prod runtime route", payload.Route)
				}
				_ = json.NewEncoder(writer).Encode(cli.CommandResult{
					OK:            true,
					Route:         payload.Route,
					ContainerName: "openclaw-prod",
					ExitCode:      0,
					Stdout:        `{"ok":true}`,
				})
			},
		},
		{
			name:       "service passthrough",
			args:       []string{"ollama", "list"},
			wantMethod: http.MethodPost,
			wantPath:   "/service/passthrough",
			wantCode:   cli.ExitOK,
			handler: func(t *testing.T, writer http.ResponseWriter, request *http.Request) {
				t.Helper()
				var payload cli.RouteRequest
				if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
					t.Fatalf("decode request: %v", err)
				}
				if payload.Route == nil || payload.Route.Resource != "ollama" || payload.Route.Kind != cli.KindServiceNative {
					t.Fatalf("payload.route = %#v, want ollama passthrough", payload.Route)
				}
				_ = json.NewEncoder(writer).Encode(cli.CommandResult{
					OK:            true,
					Route:         payload.Route,
					ContainerName: "ollama",
					ExitCode:      0,
					Stdout:        "mistral:7b-instruct-32k\n",
				})
			},
		},
		{
			name:       "scoped secrets list",
			args:       []string{"secret", "list", "test"},
			wantMethod: http.MethodPost,
			wantPath:   "/execute",
			wantCode:   cli.ExitOK,
			handler: func(t *testing.T, writer http.ResponseWriter, request *http.Request) {
				t.Helper()
				var payload cli.RouteRequest
				if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
					t.Fatalf("decode request: %v", err)
				}
				if payload.Route == nil || payload.Route.Kind != cli.KindScopedSecrets || payload.Route.Resource != "test" || payload.Route.Action != "list" {
					t.Fatalf("payload.route = %#v, want test scoped secrets list route", payload.Route)
				}
				_ = json.NewEncoder(writer).Encode(cli.SecretListResult{
					OK:    true,
					Route: payload.Route,
					Scope: "test",
					Secrets: []cli.SecretListItem{
						{Scope: "test", Name: "TOGETHER_API_KEY"},
					},
				})
			},
		},
		{
			name:       "scoped secrets set inline value",
			args:       []string{"secret", "set", "prod", "TOGETHER_API_KEY", "inline-secret"},
			wantMethod: http.MethodPost,
			wantPath:   "/execute",
			wantCode:   cli.ExitOK,
			handler: func(t *testing.T, writer http.ResponseWriter, request *http.Request) {
				t.Helper()
				var payload cli.RouteRequest
				if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
					t.Fatalf("decode request: %v", err)
				}
				if payload.Route == nil || payload.Route.Kind != cli.KindScopedSecrets || payload.Route.Action != "set" {
					t.Fatalf("payload.route = %#v, want scoped secrets set route", payload.Route)
				}
				if payload.SecretValue != "inline-secret" {
					t.Fatalf("payload.secret_value = %q, want inline-secret", payload.SecretValue)
				}
				_ = json.NewEncoder(writer).Encode(cli.SecretSetResult{
					OK:     true,
					Route:  payload.Route,
					Scope:  "prod",
					Name:   "TOGETHER_API_KEY",
					Stored: true,
				})
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				if request.Method != testCase.wantMethod {
					t.Fatalf("method = %s, want %s", request.Method, testCase.wantMethod)
				}
				if request.URL.Path != testCase.wantPath {
					t.Fatalf("path = %s, want %s", request.URL.Path, testCase.wantPath)
				}
				testCase.handler(t, writer, request)
			}))
			defer server.Close()

			t.Setenv("MOLTBOX_GATEWAY_URL", server.URL)

			var output strings.Builder
			code := run(testCase.args, &output, ioDiscard{})
			if code != testCase.wantCode {
				t.Fatalf("exit code = %d, want %d", code, testCase.wantCode)
			}
			if output.Len() == 0 {
				t.Fatal("expected gateway response output")
			}
		})
	}
}

func TestBootstrapGatewayFailsClearlyWithoutConfig(t *testing.T) {
	t.Parallel()

	var output strings.Builder
	code := run([]string{"bootstrap", "gateway"}, &output, ioDiscard{})
	if code != cli.ExitFailure {
		t.Fatalf("exit code = %d, want %d", code, cli.ExitFailure)
	}

	var payload cli.Envelope
	if err := json.Unmarshal([]byte(output.String()), &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.ErrorType != "bootstrap_config_failed" {
		t.Fatalf("error_type = %q, want bootstrap_config_failed", payload.ErrorType)
	}
}

func TestGatewayUpdateFailsClearlyWithoutConfig(t *testing.T) {
	t.Parallel()

	var output strings.Builder
	code := run([]string{"gateway", "update"}, &output, ioDiscard{})
	if code != cli.ExitFailure {
		t.Fatalf("exit code = %d, want %d", code, cli.ExitFailure)
	}

	var payload cli.Envelope
	if err := json.Unmarshal([]byte(output.String()), &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.ErrorType != "gateway_update_config_failed" {
		t.Fatalf("error_type = %q, want gateway_update_config_failed", payload.ErrorType)
	}
}

func TestRetiredNamespacesFailExplicitly(t *testing.T) {
	t.Parallel()

	retired := []string{
		"dev",
		"runtime",
		"skill",
		"plugin",
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

func TestGatewayUnavailable(t *testing.T) {
	t.Setenv("MOLTBOX_GATEWAY_URL", "http://127.0.0.1:1")

	var output strings.Builder
	code := run([]string{"gateway", "status"}, &output, ioDiscard{})
	if code != cli.ExitFailure {
		t.Fatalf("exit code = %d, want %d", code, cli.ExitFailure)
	}

	var payload cli.Envelope
	if err := json.Unmarshal([]byte(output.String()), &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.ErrorType != "gateway_unreachable" {
		t.Fatalf("error_type = %q, want gateway_unreachable", payload.ErrorType)
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

func TestScopedSecretsCommandsUseGatewayForSecretValue(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", request.Method)
		}
		if request.URL.Path != "/execute" {
			t.Fatalf("path = %s, want /execute", request.URL.Path)
		}

		var payload cli.RouteRequest
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if payload.SecretValue != "stdin-secret" {
			t.Fatalf("payload.secret_value = %q, want stdin-secret", payload.SecretValue)
		}

		_ = json.NewEncoder(writer).Encode(cli.SecretSetResult{
			OK:     true,
			Route:  payload.Route,
			Scope:  "test",
			Name:   "TOGETHER_API_KEY",
			Stored: true,
		})
	}))
	defer server.Close()

	t.Setenv("MOLTBOX_GATEWAY_URL", server.URL)
	t.Setenv("MOLTBOX_SECRET_VALUE", "stdin-secret")

	var output strings.Builder
	code := run([]string{"secret", "set", "test", "TOGETHER_API_KEY"}, &output, ioDiscard{})
	if code != cli.ExitOK {
		t.Fatalf("set exit code = %d, want %d", code, cli.ExitOK)
	}
}

func TestLoadSecretValueReturnsAfterFirstNewline(t *testing.T) {
	t.Parallel()

	reader, writer := io.Pipe()
	result := make(chan struct {
		value string
		err   error
	}, 1)

	go func() {
		value, err := loadSecretValue(reader)
		result <- struct {
			value string
			err   error
		}{value: value, err: err}
	}()

	if _, err := writer.Write([]byte("interactive-secret\n")); err != nil {
		t.Fatalf("write stdin: %v", err)
	}

	select {
	case got := <-result:
		if got.err != nil {
			t.Fatalf("loadSecretValue() error = %v", got.err)
		}
		if got.value != "interactive-secret" {
			t.Fatalf("loadSecretValue() value = %q, want interactive-secret", got.value)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("loadSecretValue() blocked waiting for EOF after newline")
	}

	_ = writer.Close()
}

func TestLoadSecretValueAcceptsEOFWithoutNewline(t *testing.T) {
	t.Parallel()

	value, err := loadSecretValue(strings.NewReader("piped-secret"))
	if err != nil {
		t.Fatalf("loadSecretValue() error = %v", err)
	}
	if value != "piped-secret" {
		t.Fatalf("loadSecretValue() value = %q, want piped-secret", value)
	}
}

func TestSSHWrapperModePreservesQuotedArgs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", request.Method)
		}
		if request.URL.Path != "/runtime/openclaw" {
			t.Fatalf("path = %s, want /runtime/openclaw", request.URL.Path)
		}

		var payload cli.RouteRequest
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		wantArgs := []string{"agent", "--agent", "main", "--local", "--thinking", "off", "--message", "Say hello in one sentence.", "--json"}
		if got := payload.Route.NativeArgs; !equalStrings(got, wantArgs) {
			t.Fatalf("payload.route.native_args = %#v, want %#v", got, wantArgs)
		}

		_ = json.NewEncoder(writer).Encode(cli.CommandResult{
			OK:            true,
			Route:         payload.Route,
			ContainerName: "openclaw-test",
			ExitCode:      0,
			Stdout:        `{"ok":true}`,
		})
	}))
	defer server.Close()

	t.Setenv("MOLTBOX_GATEWAY_URL", server.URL)

	var stdout strings.Builder
	code := run([]string{
		"__ssh-wrapper=test-operator",
		`moltbox test openclaw agent --agent main --local --thinking off --message Say hello in one sentence. --json`,
	}, &stdout, ioDiscard{})
	if code != cli.ExitOK {
		t.Fatalf("exit code = %d, want %d", code, cli.ExitOK)
	}
}

func TestSSHWrapperModeBootstrapDeniesRestrictedCommand(t *testing.T) {
	t.Parallel()

	var stdout strings.Builder
	var stderr strings.Builder
	code := run([]string{
		"__ssh-wrapper=bootstrap",
		`moltbox service deploy test`,
	}, &stdout, &stderr)
	if code != 126 {
		t.Fatalf("exit code = %d, want 126", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "bootstrap access denied: service access is limited to list, status, and logs") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestSSHWrapperModePreservesQuotedSecretValue(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", request.Method)
		}
		if request.URL.Path != "/execute" {
			t.Fatalf("path = %s, want /execute", request.URL.Path)
		}

		var payload cli.RouteRequest
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if payload.SecretValue != "value with spaces" {
			t.Fatalf("payload.secret_value = %q, want %q", payload.SecretValue, "value with spaces")
		}

		_ = json.NewEncoder(writer).Encode(cli.SecretSetResult{
			OK:     true,
			Route:  payload.Route,
			Scope:  "test",
			Name:   "TEST_SECRET",
			Stored: true,
		})
	}))
	defer server.Close()

	t.Setenv("MOLTBOX_GATEWAY_URL", server.URL)

	var stdout strings.Builder
	code := run([]string{
		"__ssh-wrapper=test-operator",
		`moltbox secret set test TEST_SECRET value with spaces`,
	}, &stdout, ioDiscard{})
	if code != cli.ExitOK {
		t.Fatalf("exit code = %d, want %d", code, cli.ExitOK)
	}
}

func TestSSHWrapperModeTestOperatorAllowsVerifyBrowser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", request.Method)
		}
		if request.URL.Path != "/runtime/verify" {
			t.Fatalf("path = %s, want /runtime/verify", request.URL.Path)
		}

		var payload cli.RouteRequest
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if payload.Route == nil {
			t.Fatal("payload.route = nil")
		}
		if payload.Route.Kind != cli.KindRuntimeVerify || payload.Route.Subject != "browser" {
			t.Fatalf("payload.route = %#v, want verify browser route", payload.Route)
		}

		_ = json.NewEncoder(writer).Encode(cli.RuntimeVerifyResult{
			OK:          true,
			Route:       payload.Route,
			Environment: "test",
			Runtime:     "openclaw-test",
			Check:       "browser",
			Summary:     "test verify browser passed",
		})
	}))
	defer server.Close()

	t.Setenv("MOLTBOX_GATEWAY_URL", server.URL)

	var stdout strings.Builder
	code := run([]string{
		"__ssh-wrapper=test-operator",
		`moltbox test verify browser https://example.com`,
	}, &stdout, ioDiscard{})
	if code != cli.ExitOK {
		t.Fatalf("exit code = %d, want %d", code, cli.ExitOK)
	}
}

func TestSSHWrapperModeProdOperatorAllowsVerifyRuntime(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", request.Method)
		}
		if request.URL.Path != "/runtime/verify" {
			t.Fatalf("path = %s, want /runtime/verify", request.URL.Path)
		}

		var payload cli.RouteRequest
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if payload.Route == nil {
			t.Fatal("payload.route = nil")
		}
		if payload.Route.Kind != cli.KindRuntimeVerify || payload.Route.Subject != "runtime" {
			t.Fatalf("payload.route = %#v, want verify runtime route", payload.Route)
		}

		_ = json.NewEncoder(writer).Encode(cli.RuntimeVerifyResult{
			OK:          true,
			Route:       payload.Route,
			Environment: "prod",
			Runtime:     "openclaw-prod",
			Check:       "runtime",
			Summary:     "prod verify runtime passed",
		})
	}))
	defer server.Close()

	t.Setenv("MOLTBOX_GATEWAY_URL", server.URL)

	var stdout strings.Builder
	code := run([]string{
		"__ssh-wrapper=prod-operator",
		`moltbox prod verify runtime`,
	}, &stdout, ioDiscard{})
	if code != cli.ExitOK {
		t.Fatalf("exit code = %d, want %d", code, cli.ExitOK)
	}
}

func TestSSHWrapperModeRejectsShellOperators(t *testing.T) {
	t.Parallel()

	var stderr strings.Builder
	code := run([]string{
		"__ssh-wrapper=test-operator",
		`moltbox test openclaw health --json; whoami`,
	}, ioDiscard{}, &stderr)
	if code != cli.ExitFailure {
		t.Fatalf("exit code = %d, want %d", code, cli.ExitFailure)
	}
	if !strings.Contains(stderr.String(), `unsupported shell operator ";"`) {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func equalStrings(got, want []string) bool {
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

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) {
	return len(p), nil
}
