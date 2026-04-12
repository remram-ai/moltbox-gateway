package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/remram-ai/moltbox-gateway/internal/command"
	appconfig "github.com/remram-ai/moltbox-gateway/internal/config"
	"github.com/remram-ai/moltbox-gateway/internal/docker"
	"github.com/remram-ai/moltbox-gateway/internal/orchestrator"
	"github.com/remram-ai/moltbox-gateway/pkg/cli"
)

func TestHandleMCPRejectsMissingTokenBeforeMethodCheck(t *testing.T) {
	server, _ := newTestServer(t, nil)

	request := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	request.RemoteAddr = "10.2.3.4:9999"
	recorder := httptest.NewRecorder()

	server.handleMCP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
}

func TestHandleMCPLogsAuthorizedTokenName(t *testing.T) {
	server, logs := newTestServer(t, nil)
	created, err := server.tokenManager.Create(&cli.Route{Resource: "gateway", Kind: cli.KindGatewayToken, Subject: "search-agent"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{}}}`))
	request.Header.Set("Authorization", "Bearer "+created.Token)
	request.RemoteAddr = "172.20.0.8:41234"
	recorder := httptest.NewRecorder()

	server.handleMCP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	logOutput := logs.String()
	if !strings.Contains(logOutput, `"token_name":"search-agent"`) {
		t.Fatalf("expected token name in log output, got %s", logOutput)
	}
	if !strings.Contains(logOutput, `"success":true`) {
		t.Fatalf("expected success log output, got %s", logOutput)
	}
	if !strings.Contains(logOutput, `"remote_address":"172.20.0.8"`) {
		t.Fatalf("expected remote address in log output, got %s", logOutput)
	}
	if strings.Contains(logOutput, created.Token) {
		t.Fatalf("token value leaked into logs: %s", logOutput)
	}
}

func TestHandleMCPRateLimitsRepeatedFailures(t *testing.T) {
	limiter := newMCPAuthLimiter()
	limiter.maxFailures = 2
	limiter.failureWindow = time.Hour
	limiter.blockDuration = time.Hour

	server, logs := newTestServer(t, limiter)

	request := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
	request.RemoteAddr = "172.21.0.9:4555"

	first := httptest.NewRecorder()
	server.handleMCP(first, request.Clone(request.Context()))
	if first.Code != http.StatusUnauthorized {
		t.Fatalf("first failure status = %d, want %d", first.Code, http.StatusUnauthorized)
	}

	secondReq := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
	secondReq.RemoteAddr = "172.21.0.9:4555"
	second := httptest.NewRecorder()
	server.handleMCP(second, secondReq)
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second failure status = %d, want %d", second.Code, http.StatusTooManyRequests)
	}
	if !strings.Contains(logs.String(), `"reason":"rate_limited"`) {
		t.Fatalf("expected rate_limited log entry, got %s", logs.String())
	}
}

func TestHandleExecuteScopedSecretsUsesGatewayHandlerAndLogs(t *testing.T) {
	server, logs := newTestServer(t, nil)

	body := strings.NewReader(`{"route":{"resource":"test","kind":"scoped_secrets","action":"set","subject":"TOGETHER_API_KEY"},"secret_value":"inline-secret"}`)
	request := httptest.NewRequest(http.MethodPost, "/execute", body)
	request.RemoteAddr = "172.20.0.12:4550"
	recorder := httptest.NewRecorder()

	server.handleExecute(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var response cli.SecretSetResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.OK || !response.Stored || response.Scope != "test" || response.Name != "TOGETHER_API_KEY" {
		t.Fatalf("response = %#v, want stored test secret result", response)
	}

	if got, err := server.secretHandler.Get("test", "TOGETHER_API_KEY"); err != nil {
		t.Fatalf("secretHandler.Get() error = %v", err)
	} else if got != "inline-secret" {
		t.Fatalf("stored secret = %q, want inline-secret", got)
	}

	logOutput := logs.String()
	if !strings.Contains(logOutput, `"scope":"test"`) || !strings.Contains(logOutput, `"action":"set"`) {
		t.Fatalf("expected scoped secret log entry, got %s", logOutput)
	}
	if strings.Contains(logOutput, "inline-secret") {
		t.Fatalf("secret value leaked to logs: %s", logOutput)
	}
}

func TestHandleExecuteRejectsNonSecretRoutes(t *testing.T) {
	server, _ := newTestServer(t, nil)

	body := strings.NewReader(`{"route":{"resource":"test","kind":"runtime_action","action":"reload","environment":"test","runtime":"openclaw-test"}}`)
	request := httptest.NewRequest(http.MethodPost, "/execute", body)
	recorder := httptest.NewRecorder()

	server.handleExecute(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}

	var response cli.Envelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.ErrorType != "parse_error" {
		t.Fatalf("response = %#v, want parse_error", response)
	}
}

func TestParseRuntimePluginRESTPath(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		path       string
		wantTarget runtimePluginRESTTarget
		wantOK     bool
	}{
		{
			name:       "install path",
			path:       "/runtime/dev/plugins/install",
			wantTarget: runtimePluginRESTTarget{Environment: "dev", Action: "install"},
			wantOK:     true,
		},
		{
			name:       "list path",
			path:       "/runtime/test/plugins",
			wantTarget: runtimePluginRESTTarget{Environment: "test", Action: "list"},
			wantOK:     true,
		},
		{
			name:       "remove path",
			path:       "/runtime/prod/plugins/moltbox-telemetry",
			wantTarget: runtimePluginRESTTarget{Environment: "prod", Action: "remove", Plugin: "moltbox-telemetry"},
			wantOK:     true,
		},
		{
			name:   "legacy path ignored",
			path:   "/runtime/plugin/install",
			wantOK: false,
		},
		{
			name:   "invalid env ignored",
			path:   "/runtime/stage/plugins",
			wantOK: false,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, ok := parseRuntimePluginRESTPath(testCase.path)
			if ok != testCase.wantOK {
				t.Fatalf("parseRuntimePluginRESTPath(%q) ok = %v, want %v", testCase.path, ok, testCase.wantOK)
			}
			if !testCase.wantOK {
				return
			}
			if got != testCase.wantTarget {
				t.Fatalf("parseRuntimePluginRESTPath(%q) = %#v, want %#v", testCase.path, got, testCase.wantTarget)
			}
		})
	}
}

func TestRuntimePluginRouteBuildsCanonicalRoute(t *testing.T) {
	t.Parallel()

	route, err := runtimePluginRoute("dev", "install", "moltbox-telemetry")
	if err != nil {
		t.Fatalf("runtimePluginRoute() error = %v", err)
	}
	if route.Kind != cli.KindRuntimePlugin || route.Action != "install" || route.Subject != "moltbox-telemetry" {
		t.Fatalf("route = %#v, want dev plugin install moltbox-telemetry", route)
	}
	if route.Environment != "dev" || route.Runtime != "openclaw-dev" {
		t.Fatalf("route = %#v, want dev/openclaw-dev", route)
	}
}

func TestHandleServiceStatusReturnsOKForKnownImageServiceWithoutBuiltArtifact(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	servicesRoot := filepath.Join(root, "services-repo")
	if err := os.MkdirAll(filepath.Join(servicesRoot, "services", "dev-sandbox"), 0o755); err != nil {
		t.Fatalf("mkdir services root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(servicesRoot, "services", "dev-sandbox", "service.yaml"), []byte("kind: image\ncompose_project: dev-sandbox\nimage_name: moltbox-dev-sandbox\nactive_container_prefix: moltbox-dev-sandbox\n"), 0o644); err != nil {
		t.Fatalf("write service.yaml: %v", err)
	}

	manager := orchestrator.NewManager(appconfig.Config{
		Paths: appconfig.PathsConfig{
			StateRoot:   filepath.Join(root, "state"),
			RuntimeRoot: filepath.Join(root, "runtime"),
			LogsRoot:    filepath.Join(root, "logs"),
		},
		Repos: appconfig.ReposConfig{
			Services: appconfig.RepoConfig{URL: servicesRoot},
		},
		Gateway: appconfig.GatewayConfig{Host: "127.0.0.1", Port: 7460},
	}, fakeGatewayInspector{}, &fakeGatewayRunner{
		results: []command.Result{
			{ExitCode: 1},
			{ExitCode: 0},
		},
	}, nil)

	server := &Server{orchestrator: manager}
	request := httptest.NewRequest(http.MethodGet, "/service/status?service=dev-sandbox", nil)
	recorder := httptest.NewRecorder()

	server.handleServiceStatus(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response cli.ServiceStatusResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.ServiceKind != "image" || response.Status != "not-built" || response.Present {
		t.Fatalf("response = %#v, want known image service with not-built status", response)
	}
}

type fakeGatewayRunner struct {
	results []command.Result
}

func (f *fakeGatewayRunner) Run(_ context.Context, _ string, _ string, _ ...string) (command.Result, error) {
	if len(f.results) == 0 {
		return command.Result{}, nil
	}
	result := f.results[0]
	f.results = f.results[1:]
	return result, nil
}

type fakeGatewayInspector struct{}

func (fakeGatewayInspector) InspectContainer(_ context.Context, _ string) (docker.ContainerInfo, error) {
	return docker.ContainerInfo{}, docker.ErrContainerNotFound
}

func newTestServer(t *testing.T, limiter *mcpAuthLimiter) (*Server, *bytes.Buffer) {
	t.Helper()

	root := t.TempDir()
	logs := &bytes.Buffer{}
	cfg := appconfig.Default()
	cfg.Paths.StateRoot = filepath.Join(root, "state")
	cfg.Paths.RuntimeRoot = filepath.Join(root, "runtime")
	cfg.Paths.LogsRoot = filepath.Join(root, "logs")
	cfg.Paths.SecretsRoot = filepath.Join(root, "secrets")
	cfg.Gateway.Host = "127.0.0.1"
	cfg.Gateway.Port = 7460

	server := NewServer(Config{
		AppConfig: cfg,
		logger:    slog.New(slog.NewJSONHandler(logs, nil)),
		mcpAuthLimiter: func() *mcpAuthLimiter {
			if limiter != nil {
				return limiter
			}
			return newMCPAuthLimiter()
		}(),
	})
	return server, logs
}
