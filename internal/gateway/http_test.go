package gateway

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	appconfig "github.com/remram-ai/moltbox-gateway/internal/config"
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
