package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/remram-ai/moltbox-gateway/pkg/cli"
)

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/status", s.handleStatus)
	mux.HandleFunc("/service/list", s.handleServiceList)
	mux.HandleFunc("/service/status", s.handleServiceStatus)
	mux.HandleFunc("/service/deploy", s.handleServiceDeploy)
	mux.HandleFunc("/service/restart", s.handleServiceRestart)
	mux.HandleFunc("/service/remove", s.handleServiceRemove)
	mux.HandleFunc("/service/logs", s.handleServiceLogs)
	mux.HandleFunc("/service/passthrough", s.handleServicePassthrough)
	mux.HandleFunc("/logs", s.handleGatewayLogs)
	mux.HandleFunc("/update", s.handleGatewayUpdate)
	mux.HandleFunc("/repo-sync", s.handleGatewayRepoSync)
	mux.HandleFunc("/runtime/openclaw", s.handleRuntimeOpenClaw)
	mux.HandleFunc("/runtime/verify", s.handleRuntimeVerify)
	mux.HandleFunc("/token/create", s.handleTokenCreate)
	mux.HandleFunc("/token/list", s.handleTokenList)
	mux.HandleFunc("/token/delete", s.handleTokenDelete)
	mux.HandleFunc("/token/rotate", s.handleTokenRotate)
	mux.HandleFunc("/mcp", s.handleMCP)
	mux.HandleFunc("/execute", s.handleExecute)
	return mux
}

type runtimePluginRESTTarget struct {
	Environment string
	Action      string
	Plugin      string
}

func (s *Server) handleHealth(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		s.writeJSON(writer, http.StatusMethodNotAllowed, cli.Error(nil, "parse_error", "method not allowed", "use GET /health"))
		return
	}

	s.writeJSON(writer, http.StatusOK, cli.GatewayHealthResult{
		OK:      true,
		Service: "gateway",
		Version: cli.Version,
	})
}

func (s *Server) handleStatus(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		s.writeJSON(writer, http.StatusMethodNotAllowed, cli.Error(nil, "parse_error", "method not allowed", "use GET /status"))
		return
	}

	s.writeJSON(writer, http.StatusOK, cli.GatewayStatusResult{
		OK:            true,
		Route:         &cli.Route{Resource: "gateway", Kind: cli.KindGateway, Action: "status"},
		Service:       "gateway",
		Version:       cli.Version,
		ListenAddress: s.listenAddress,
		DockerSocket:  s.dockerSocketPath,
	})
}

func (s *Server) handleServiceList(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		s.writeJSON(writer, http.StatusMethodNotAllowed, cli.Error(nil, "parse_error", "method not allowed", "use GET /service/list"))
		return
	}

	route := &cli.Route{Resource: "service", Kind: cli.KindService, Action: "list"}
	ctx, cancel := context.WithTimeout(request.Context(), 10*time.Second)
	defer cancel()

	result, err := s.orchestrator.ServiceList(ctx, route)
	if err != nil {
		s.writeJSON(writer, http.StatusBadGateway, cli.Error(
			route,
			"service_list_failed",
			"failed to list managed services",
			err.Error(),
		))
		return
	}

	s.writeJSON(writer, http.StatusOK, result)
}

func (s *Server) handleServiceStatus(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		s.writeJSON(writer, http.StatusMethodNotAllowed, cli.Error(nil, "parse_error", "method not allowed", "use GET /service/status"))
		return
	}

	service := strings.TrimSpace(request.URL.Query().Get("service"))
	route := &cli.Route{Resource: "service", Kind: cli.KindService, Action: "status", Subject: service}
	if service == "" {
		s.writeJSON(writer, http.StatusBadRequest, cli.Error(route, "parse_error", "missing service query parameter", "use GET /service/status?service=<service>"))
		return
	}

	ctx, cancel := context.WithTimeout(request.Context(), 5*time.Second)
	defer cancel()

	result, err := s.orchestrator.ServiceStatus(ctx, route, service)
	if err != nil {
		s.writeJSON(writer, http.StatusBadGateway, cli.Error(
			route,
			"service_status_failed",
			fmt.Sprintf("failed to inspect service '%s'", service),
			err.Error(),
		))
		return
	}

	s.writeJSON(writer, http.StatusOK, result)
}

func (s *Server) handleServiceDeploy(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		s.writeJSON(writer, http.StatusMethodNotAllowed, cli.Error(nil, "parse_error", "method not allowed", "use POST /service/deploy"))
		return
	}

	var payload cli.RouteRequest
	if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
		s.writeJSON(writer, http.StatusBadRequest, cli.Error(nil, "parse_error", "invalid JSON request body", "send JSON with the target service"))
		return
	}

	service := strings.TrimSpace(payload.Service)
	route := &cli.Route{Resource: "service", Kind: cli.KindService, Action: "deploy", Subject: service}
	if payload.Route != nil {
		route = payload.Route
	}
	if strings.TrimSpace(route.Subject) == "" {
		route.Subject = service
	}
	if strings.TrimSpace(route.Subject) == "" {
		s.writeJSON(writer, http.StatusBadRequest, cli.Error(route, "parse_error", "missing service name", "use: service deploy <service>"))
		return
	}

	timeout := 2 * time.Minute
	if route.Subject == "dev-sandbox" {
		timeout = 10 * time.Minute
	}
	ctx, cancel := context.WithTimeout(request.Context(), timeout)
	defer cancel()

	result, err := s.orchestrator.DeployService(ctx, route, route.Subject)
	if err != nil {
		s.writeJSON(writer, http.StatusBadGateway, cli.Error(
			route,
			"service_deploy_failed",
			fmt.Sprintf("failed to deploy service '%s'", route.Subject),
			err.Error(),
		))
		return
	}

	s.writeJSON(writer, http.StatusOK, result)
}

func (s *Server) handleServiceRestart(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		s.writeJSON(writer, http.StatusMethodNotAllowed, cli.Error(nil, "parse_error", "method not allowed", "use POST /service/restart"))
		return
	}

	route, ok := s.parseServiceRouteRequest(writer, request, "restart")
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(request.Context(), 45*time.Second)
	defer cancel()

	result, err := s.orchestrator.RestartService(ctx, route, route.Subject)
	if err != nil {
		s.writeJSON(writer, http.StatusBadGateway, cli.Error(
			route,
			"service_restart_failed",
			fmt.Sprintf("failed to restart service '%s'", route.Subject),
			err.Error(),
		))
		return
	}

	s.writeJSON(writer, http.StatusOK, result)
}

func (s *Server) handleServiceRemove(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		s.writeJSON(writer, http.StatusMethodNotAllowed, cli.Error(nil, "parse_error", "method not allowed", "use POST /service/remove"))
		return
	}

	route, ok := s.parseServiceRouteRequest(writer, request, "remove")
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(request.Context(), 45*time.Second)
	defer cancel()

	result, err := s.orchestrator.RemoveService(ctx, route, route.Subject)
	if err != nil {
		s.writeJSON(writer, http.StatusBadGateway, cli.Error(
			route,
			"service_remove_failed",
			fmt.Sprintf("failed to remove service '%s'", route.Subject),
			err.Error(),
		))
		return
	}

	s.writeJSON(writer, http.StatusOK, result)
}

func (s *Server) handleServiceLogs(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		s.writeJSON(writer, http.StatusMethodNotAllowed, cli.Error(nil, "parse_error", "method not allowed", "use GET /service/logs"))
		return
	}

	service := strings.TrimSpace(request.URL.Query().Get("service"))
	route := &cli.Route{Resource: "service", Kind: cli.KindService, Action: "logs", Subject: service}
	if service == "" {
		s.writeJSON(writer, http.StatusBadRequest, cli.Error(route, "parse_error", "missing service query parameter", "use GET /service/logs?service=<service>"))
		return
	}

	ctx, cancel := context.WithTimeout(request.Context(), 30*time.Second)
	defer cancel()

	result, err := s.orchestrator.ServiceLogs(ctx, route, service)
	if err != nil {
		s.writeJSON(writer, http.StatusBadGateway, cli.Error(
			route,
			"service_logs_failed",
			fmt.Sprintf("failed to read logs for service '%s'", service),
			err.Error(),
		))
		return
	}
	if !result.OK {
		s.writeJSON(writer, http.StatusBadGateway, cli.Error(
			route,
			"service_logs_failed",
			fmt.Sprintf("failed to read logs for service '%s'", service),
			result.Stdout,
		))
		return
	}

	s.writeJSON(writer, http.StatusOK, result)
}

func (s *Server) handleServicePassthrough(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		s.writeJSON(writer, http.StatusMethodNotAllowed, cli.Error(nil, "parse_error", "method not allowed", "use POST /service/passthrough"))
		return
	}

	payload, ok := s.parseRouteRequest(writer, request, "send JSON with the parsed service route")
	if !ok {
		return
	}
	if payload.Route == nil || payload.Route.Kind != cli.KindServiceNative {
		s.writeJSON(writer, http.StatusBadRequest, cli.Error(payload.Route, "parse_error", "missing service passthrough route", "use a documented service passthrough command"))
		return
	}

	timeout := 2 * time.Minute
	if payload.Route.Subject == "sandbox" {
		timeout = 10 * time.Minute
	}
	ctx, cancel := context.WithTimeout(request.Context(), timeout)
	defer cancel()

	result, err := s.orchestrator.ServicePassthrough(ctx, payload.Route)
	if err != nil {
		s.writeJSON(writer, http.StatusBadGateway, cli.Error(
			payload.Route,
			"service_passthrough_failed",
			fmt.Sprintf("failed to execute %s passthrough", payload.Route.Resource),
			err.Error(),
		))
		return
	}
	if !result.OK {
		s.writeJSON(writer, http.StatusBadGateway, cli.Error(
			payload.Route,
			"service_passthrough_failed",
			fmt.Sprintf("%s passthrough command failed", payload.Route.Resource),
			result.Stdout,
		))
		return
	}

	s.writeJSON(writer, http.StatusOK, result)
}

func (s *Server) handleGatewayLogs(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		s.writeJSON(writer, http.StatusMethodNotAllowed, cli.Error(nil, "parse_error", "method not allowed", "use GET /logs"))
		return
	}

	route := &cli.Route{Resource: "gateway", Kind: cli.KindGateway, Action: "logs"}
	ctx, cancel := context.WithTimeout(request.Context(), 30*time.Second)
	defer cancel()

	result, err := s.orchestrator.GatewayLogs(ctx, route)
	if err != nil {
		s.writeJSON(writer, http.StatusBadGateway, cli.Error(
			route,
			"gateway_logs_failed",
			"failed to read gateway logs",
			err.Error(),
		))
		return
	}
	if !result.OK {
		s.writeJSON(writer, http.StatusBadGateway, cli.Error(
			route,
			"gateway_logs_failed",
			"failed to read gateway logs",
			result.Stdout,
		))
		return
	}
	s.writeJSON(writer, http.StatusOK, result)
}

func (s *Server) handleGatewayUpdate(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		s.writeJSON(writer, http.StatusMethodNotAllowed, cli.Error(nil, "parse_error", "method not allowed", "use POST /update"))
		return
	}

	route := &cli.Route{Resource: "gateway", Kind: cli.KindGateway, Action: "update", Subject: "gateway"}
	ctx, cancel := context.WithTimeout(request.Context(), 30*time.Second)
	defer cancel()

	result, err := s.orchestrator.GatewayUpdate(ctx, route)
	if err != nil {
		s.writeJSON(writer, http.StatusBadGateway, cli.Error(
			route,
			"gateway_update_failed",
			"failed to deploy gateway service",
			err.Error(),
		))
		return
	}
	s.writeJSON(writer, http.StatusOK, result)
}

func (s *Server) handleGatewayRepoSync(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		s.writeJSON(writer, http.StatusMethodNotAllowed, cli.Error(nil, "parse_error", "method not allowed", "use POST /repo-sync"))
		return
	}

	payload, ok := s.parseRouteRequest(writer, request, "send JSON with the parsed gateway repo-sync route")
	if !ok {
		return
	}
	if payload.Route == nil || payload.Route.Kind != cli.KindGateway || payload.Route.Action != "repo-sync" {
		s.writeJSON(writer, http.StatusBadRequest, cli.Error(payload.Route, "parse_error", "missing gateway repo-sync route", "use: gateway repo-sync services|runtime|all"))
		return
	}

	ctx, cancel := context.WithTimeout(request.Context(), 5*time.Minute)
	defer cancel()

	result, err := s.orchestrator.GatewayRepoSync(ctx, payload.Route)
	if err != nil {
		s.writeJSON(writer, http.StatusBadGateway, cli.Error(
			payload.Route,
			"gateway_repo_sync_failed",
			"failed to sync managed repos",
			err.Error(),
		))
		return
	}

	s.writeJSON(writer, http.StatusOK, result)
}

func (s *Server) handleTokenCreate(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		s.writeJSON(writer, http.StatusMethodNotAllowed, cli.Error(nil, "parse_error", "method not allowed", "use POST /token/create"))
		return
	}
	payload, ok := s.parseRouteRequest(writer, request, "send JSON with the named token route")
	if !ok {
		return
	}
	route := &cli.Route{Resource: "gateway", Kind: cli.KindGatewayToken, Action: "create"}
	if payload.Route != nil {
		route = payload.Route
	}
	result, err := s.tokenManager.Create(route)
	if err != nil {
		s.writeJSON(writer, http.StatusBadGateway, cli.Error(route, "token_create_failed", "failed to create MCP token", err.Error()))
		return
	}
	s.writeJSON(writer, http.StatusOK, result)
}

func (s *Server) handleTokenList(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		s.writeJSON(writer, http.StatusMethodNotAllowed, cli.Error(nil, "parse_error", "method not allowed", "use GET /token/list"))
		return
	}
	route := &cli.Route{Resource: "gateway", Kind: cli.KindGatewayToken, Action: "list"}
	result, err := s.tokenManager.List(route)
	if err != nil {
		s.writeJSON(writer, http.StatusBadGateway, cli.Error(route, "token_list_failed", "failed to list MCP tokens", err.Error()))
		return
	}
	s.writeJSON(writer, http.StatusOK, result)
}

func (s *Server) handleTokenDelete(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		s.writeJSON(writer, http.StatusMethodNotAllowed, cli.Error(nil, "parse_error", "method not allowed", "use POST /token/delete"))
		return
	}
	payload, ok := s.parseRouteRequest(writer, request, "send JSON with the named token route")
	if !ok {
		return
	}
	route := &cli.Route{Resource: "gateway", Kind: cli.KindGatewayToken, Action: "delete"}
	if payload.Route != nil {
		route = payload.Route
	}
	result, err := s.tokenManager.Delete(route)
	if err != nil {
		s.writeJSON(writer, http.StatusBadGateway, cli.Error(route, "token_delete_failed", "failed to delete MCP token", err.Error()))
		return
	}
	s.writeJSON(writer, http.StatusOK, result)
}

func (s *Server) handleTokenRotate(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		s.writeJSON(writer, http.StatusMethodNotAllowed, cli.Error(nil, "parse_error", "method not allowed", "use POST /token/rotate"))
		return
	}
	payload, ok := s.parseRouteRequest(writer, request, "send JSON with the named token route")
	if !ok {
		return
	}
	route := &cli.Route{Resource: "gateway", Kind: cli.KindGatewayToken, Action: "rotate"}
	if payload.Route != nil {
		route = payload.Route
	}
	result, err := s.tokenManager.Rotate(route)
	if err != nil {
		s.writeJSON(writer, http.StatusBadGateway, cli.Error(route, "token_rotate_failed", "failed to rotate MCP token", err.Error()))
		return
	}
	s.writeJSON(writer, http.StatusOK, result)
}

func (s *Server) handleMCP(writer http.ResponseWriter, request *http.Request) {
	result, err := s.tokenManager.ValidateBearerToken(request.Header.Get("Authorization"))
	if err != nil {
		s.logMCPAuth(request, "", false, "validation_error")
		s.writeJSON(writer, http.StatusUnauthorized, cli.Error(nil, "unauthorized", "failed to validate MCP token", err.Error()))
		return
	}
	if !result.Authorized {
		if s.mcpAuthLimiter.RecordFailure(request.RemoteAddr) {
			s.logMCPAuth(request, "", false, "rate_limited")
			s.writeJSON(writer, http.StatusTooManyRequests, cli.Error(nil, "rate_limited", "too many failed MCP authentication attempts", "wait and retry with a valid bearer token"))
			return
		}
		s.logMCPAuth(request, "", false, "invalid_token")
		s.writeJSON(writer, http.StatusUnauthorized, cli.Error(nil, "unauthorized", "missing or invalid MCP token", "send Authorization: Bearer <token>"))
		return
	}
	s.mcpAuthLimiter.RecordSuccess(request.RemoteAddr)
	s.logMCPAuth(request, result.Name, true, "authorized")
	if request.Method != http.MethodPost {
		s.writeJSON(writer, http.StatusMethodNotAllowed, cli.Error(nil, "parse_error", "method not allowed", "use POST /mcp"))
		return
	}

	body, err := io.ReadAll(request.Body)
	if err != nil {
		s.writeJSON(writer, http.StatusBadRequest, cli.Error(nil, "parse_error", "failed to read MCP request body", err.Error()))
		return
	}
	response, ok, err := s.mcpServer.HandleMessage(body)
	if err != nil {
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(writer).Encode(map[string]any{
			"jsonrpc": "2.0",
			"error": map[string]any{
				"code":    -32700,
				"message": "parse error",
			},
		})
		return
	}
	if !ok {
		writer.WriteHeader(http.StatusNoContent)
		return
	}
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(writer).Encode(response)
}

func (s *Server) logMCPAuth(request *http.Request, tokenName string, authorized bool, reason string) {
	if s.logger == nil {
		return
	}
	attrs := []any{
		"token_name", tokenName,
		"success", authorized,
		"remote_address", authRemoteKey(request.RemoteAddr),
		"reason", reason,
	}
	if authorized {
		s.logger.Info("mcp auth", attrs...)
		return
	}
	s.logger.Warn("mcp auth", attrs...)
}

func (s *Server) handleRuntimeReload(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		s.writeJSON(writer, http.StatusMethodNotAllowed, cli.Error(nil, "parse_error", "method not allowed", "use POST /runtime/reload"))
		return
	}

	payload, ok := s.parseRouteRequest(writer, request, "send JSON with the parsed runtime route")
	if !ok {
		return
	}
	if payload.Route == nil || payload.Route.Kind != cli.KindRuntimeAction || payload.Route.Action != "reload" {
		s.writeJSON(writer, http.StatusBadRequest, cli.Error(payload.Route, "parse_error", "missing runtime reload route", "use: dev|test|prod reload"))
		return
	}

	ctx, cancel := context.WithTimeout(request.Context(), 2*time.Minute)
	defer cancel()

	result, err := s.orchestrator.RuntimeReload(ctx, payload.Route)
	if err != nil {
		s.writeJSON(writer, http.StatusBadGateway, cli.Error(
			payload.Route,
			"runtime_reload_failed",
			fmt.Sprintf("failed to reload runtime '%s'", payload.Route.Runtime),
			err.Error(),
		))
		return
	}
	s.writeJSON(writer, http.StatusOK, result)
}

func (s *Server) handleRuntimeVerify(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		s.writeJSON(writer, http.StatusMethodNotAllowed, cli.Error(nil, "parse_error", "method not allowed", "use POST /runtime/verify"))
		return
	}

	payload, ok := s.parseRouteRequest(writer, request, "send JSON with the parsed runtime verify route")
	if !ok {
		return
	}
	if payload.Route == nil || payload.Route.Kind != cli.KindRuntimeVerify || payload.Route.Action != "verify" {
		s.writeJSON(writer, http.StatusBadRequest, cli.Error(payload.Route, "parse_error", "missing runtime verify route", "use: test verify <check> | prod verify runtime"))
		return
	}

	ctx, cancel := context.WithTimeout(request.Context(), 2*time.Minute)
	defer cancel()

	result, err := s.orchestrator.RuntimeVerify(ctx, payload.Route)
	if err != nil {
		s.writeJSON(writer, http.StatusBadGateway, cli.Error(
			payload.Route,
			"runtime_verify_failed",
			fmt.Sprintf("failed to verify runtime '%s'", payload.Route.Runtime),
			err.Error(),
		))
		return
	}

	s.writeJSON(writer, http.StatusOK, result)
}

func (s *Server) handleRuntimeCheckpoint(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		s.writeJSON(writer, http.StatusMethodNotAllowed, cli.Error(nil, "parse_error", "method not allowed", "use POST /runtime/checkpoint"))
		return
	}

	payload, ok := s.parseRouteRequest(writer, request, "send JSON with the parsed runtime route")
	if !ok {
		return
	}
	if payload.Route == nil {
		s.writeJSON(writer, http.StatusBadRequest, cli.Error(nil, "parse_error", "missing route in checkpoint request", "send JSON with the parsed route"))
		return
	}

	ctx, cancel := context.WithTimeout(request.Context(), 10*time.Minute)
	defer cancel()

	result, err := s.orchestrator.RuntimeCheckpoint(ctx, payload.Route)
	if err != nil {
		s.writeJSON(writer, http.StatusBadGateway, cli.Error(
			payload.Route,
			"runtime_checkpoint_failed",
			fmt.Sprintf("failed to checkpoint runtime '%s'", payload.Route.Runtime),
			err.Error(),
		))
		return
	}
	s.writeJSON(writer, http.StatusOK, result)
}

func (s *Server) handleRuntimeREST(writer http.ResponseWriter, request *http.Request) {
	target, ok := parseRuntimePluginRESTPath(request.URL.Path)
	if !ok {
		http.NotFound(writer, request)
		return
	}

	switch target.Action {
	case "install":
		s.handleRuntimePluginInstall(writer, request)
	case "list":
		s.handleRuntimePluginList(writer, request)
	case "remove":
		s.handleRuntimePluginRemove(writer, request)
	default:
		http.NotFound(writer, request)
	}
}

func (s *Server) handleRuntimeSkillDeploy(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		s.writeJSON(writer, http.StatusMethodNotAllowed, cli.Error(nil, "parse_error", "method not allowed", "use POST /runtime/skill/deploy"))
		return
	}

	payload, ok := s.parseRouteRequest(writer, request, "send JSON with the parsed runtime skill route")
	if !ok {
		return
	}
	if payload.Route == nil || payload.Route.Kind != cli.KindRuntimeSkill || payload.Route.Action != "deploy" {
		s.writeJSON(writer, http.StatusBadRequest, cli.Error(payload.Route, "parse_error", "missing runtime skill deploy route", "use: dev|test|prod skill deploy <skill>"))
		return
	}

	ctx, cancel := context.WithTimeout(request.Context(), 2*time.Minute)
	defer cancel()

	result, err := s.orchestrator.RuntimeSkillDeploy(ctx, payload.Route)
	if err != nil {
		s.writeJSON(writer, http.StatusBadGateway, cli.Error(
			payload.Route,
			"runtime_skill_deploy_failed",
			fmt.Sprintf("failed to deploy skill '%s' into runtime '%s'", payload.Route.Subject, payload.Route.Runtime),
			err.Error(),
		))
		return
	}
	s.writeJSON(writer, http.StatusOK, result)
}

func (s *Server) handleRuntimeSkillList(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		s.writeJSON(writer, http.StatusMethodNotAllowed, cli.Error(nil, "parse_error", "method not allowed", "use POST /runtime/skill/list"))
		return
	}

	payload, ok := s.parseRouteRequest(writer, request, "send JSON with the parsed runtime skill route")
	if !ok {
		return
	}
	if payload.Route == nil || payload.Route.Kind != cli.KindRuntimeSkill || payload.Route.Action != "list" {
		s.writeJSON(writer, http.StatusBadRequest, cli.Error(payload.Route, "parse_error", "missing runtime skill list route", "use: dev|test|prod skill list"))
		return
	}

	ctx, cancel := context.WithTimeout(request.Context(), 2*time.Minute)
	defer cancel()

	result, err := s.orchestrator.RuntimeSkillList(ctx, payload.Route)
	if err != nil {
		s.writeJSON(writer, http.StatusBadGateway, cli.Error(
			payload.Route,
			"runtime_skill_list_failed",
			fmt.Sprintf("failed to list skills in runtime '%s'", payload.Route.Runtime),
			err.Error(),
		))
		return
	}
	if !result.OK {
		s.writeJSON(writer, http.StatusBadGateway, cli.Error(
			payload.Route,
			"runtime_skill_list_failed",
			fmt.Sprintf("failed to list skills in runtime '%s'", payload.Route.Runtime),
			result.Stdout,
		))
		return
	}
	s.writeJSON(writer, http.StatusOK, result)
}

func (s *Server) handleRuntimeSkillRemove(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		s.writeJSON(writer, http.StatusMethodNotAllowed, cli.Error(nil, "parse_error", "method not allowed", "use POST /runtime/skill/remove"))
		return
	}

	payload, ok := s.parseRouteRequest(writer, request, "send JSON with the parsed runtime skill route")
	if !ok {
		return
	}
	if payload.Route == nil || payload.Route.Kind != cli.KindRuntimeSkill || payload.Route.Action != "remove" {
		s.writeJSON(writer, http.StatusBadRequest, cli.Error(payload.Route, "parse_error", "missing runtime skill remove route", "use: dev|test|prod skill remove <skill>"))
		return
	}

	ctx, cancel := context.WithTimeout(request.Context(), 2*time.Minute)
	defer cancel()

	result, err := s.orchestrator.RuntimeSkillRemove(ctx, payload.Route)
	if err != nil {
		s.writeJSON(writer, http.StatusBadGateway, cli.Error(
			payload.Route,
			"runtime_skill_remove_failed",
			fmt.Sprintf("failed to remove skill '%s' from runtime '%s'", payload.Route.Subject, payload.Route.Runtime),
			err.Error(),
		))
		return
	}
	s.writeJSON(writer, http.StatusOK, result)
}

func (s *Server) handleRuntimeSkillRollback(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		s.writeJSON(writer, http.StatusMethodNotAllowed, cli.Error(nil, "parse_error", "method not allowed", "use POST /runtime/skill/rollback"))
		return
	}

	payload, ok := s.parseRouteRequest(writer, request, "send JSON with the parsed runtime skill route")
	if !ok {
		return
	}
	if payload.Route == nil || payload.Route.Kind != cli.KindRuntimeSkill {
		s.writeJSON(writer, http.StatusBadRequest, cli.Error(payload.Route, "parse_error", "missing runtime skill remove route", "use: dev|test|prod skill remove <skill>"))
		return
	}
	if payload.Route.Action == "rollback" {
		payload.Route.Action = "remove"
	}
	if payload.Route.Action != "remove" {
		s.writeJSON(writer, http.StatusBadRequest, cli.Error(payload.Route, "parse_error", "missing runtime skill remove route", "use: dev|test|prod skill remove <skill>"))
		return
	}

	ctx, cancel := context.WithTimeout(request.Context(), 2*time.Minute)
	defer cancel()

	result, err := s.orchestrator.RuntimeSkillRemove(ctx, payload.Route)
	if err != nil {
		s.writeJSON(writer, http.StatusBadGateway, cli.Error(
			payload.Route,
			"runtime_skill_remove_failed",
			fmt.Sprintf("failed to remove skill '%s' from runtime '%s'", payload.Route.Subject, payload.Route.Runtime),
			err.Error(),
		))
		return
	}
	s.writeJSON(writer, http.StatusOK, result)
}

func (s *Server) handleRuntimePluginInstall(writer http.ResponseWriter, request *http.Request) {
	target, restPath := parseRuntimePluginRESTPath(request.URL.Path)
	recovery := "use POST /runtime/plugin/install"
	if restPath {
		recovery = fmt.Sprintf("use POST /runtime/%s/plugins/install", target.Environment)
	}
	if request.Method != http.MethodPost {
		s.writeJSON(writer, http.StatusMethodNotAllowed, cli.Error(nil, "parse_error", "method not allowed", recovery))
		return
	}

	route, ok := s.parseRuntimePluginInstallRoute(writer, request, target, restPath)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(request.Context(), 10*time.Minute)
	defer cancel()

	result, err := s.orchestrator.RuntimePluginInstall(ctx, route)
	if err != nil {
		s.writeJSON(writer, http.StatusBadGateway, cli.Error(
			route,
			"runtime_plugin_install_failed",
			fmt.Sprintf("failed to install plugin '%s' into runtime '%s'", route.Subject, route.Runtime),
			err.Error(),
		))
		return
	}
	s.writeJSON(writer, http.StatusOK, result)
}

func (s *Server) handleRuntimePluginList(writer http.ResponseWriter, request *http.Request) {
	target, restPath := parseRuntimePluginRESTPath(request.URL.Path)
	expectedMethod := http.MethodPost
	recovery := "use POST /runtime/plugin/list"
	if restPath {
		expectedMethod = http.MethodGet
		recovery = fmt.Sprintf("use GET /runtime/%s/plugins", target.Environment)
	}
	if request.Method != expectedMethod {
		s.writeJSON(writer, http.StatusMethodNotAllowed, cli.Error(nil, "parse_error", "method not allowed", recovery))
		return
	}

	route, ok := s.parseRuntimePluginRoute(writer, request, target, restPath, "list")
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(request.Context(), 2*time.Minute)
	defer cancel()

	result, err := s.orchestrator.RuntimePluginList(ctx, route)
	if err != nil {
		s.writeJSON(writer, http.StatusBadGateway, cli.Error(
			route,
			"runtime_plugin_list_failed",
			fmt.Sprintf("failed to list plugins in runtime '%s'", route.Runtime),
			err.Error(),
		))
		return
	}
	s.writeJSON(writer, http.StatusOK, result)
}

func (s *Server) handleRuntimePluginRemove(writer http.ResponseWriter, request *http.Request) {
	target, restPath := parseRuntimePluginRESTPath(request.URL.Path)
	expectedMethod := http.MethodPost
	recovery := "use POST /runtime/plugin/remove"
	if restPath {
		expectedMethod = http.MethodDelete
		recovery = fmt.Sprintf("use DELETE /runtime/%s/plugins/<plugin>", target.Environment)
	}
	if request.Method != expectedMethod {
		s.writeJSON(writer, http.StatusMethodNotAllowed, cli.Error(nil, "parse_error", "method not allowed", recovery))
		return
	}

	route, ok := s.parseRuntimePluginRoute(writer, request, target, restPath, "remove")
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(request.Context(), 2*time.Minute)
	defer cancel()

	result, err := s.orchestrator.RuntimePluginRemove(ctx, route)
	if err != nil {
		s.writeJSON(writer, http.StatusBadGateway, cli.Error(
			route,
			"runtime_plugin_remove_failed",
			fmt.Sprintf("failed to remove plugin '%s' from runtime '%s'", route.Subject, route.Runtime),
			err.Error(),
		))
		return
	}
	s.writeJSON(writer, http.StatusOK, result)
}

func (s *Server) handleRuntimeOpenClaw(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		s.writeJSON(writer, http.StatusMethodNotAllowed, cli.Error(nil, "parse_error", "method not allowed", "use POST /runtime/openclaw"))
		return
	}

	payload, ok := s.parseRouteRequest(writer, request, "send JSON with the parsed runtime route")
	if !ok {
		return
	}
	if payload.Route == nil || payload.Route.Kind != cli.KindRuntimeNative {
		s.writeJSON(writer, http.StatusBadRequest, cli.Error(payload.Route, "parse_error", "missing runtime openclaw route", "use: test|prod openclaw <command>"))
		return
	}

	ctx, cancel := context.WithTimeout(request.Context(), 2*time.Minute)
	defer cancel()

	result, err := s.orchestrator.RuntimeOpenClaw(ctx, payload.Route)
	if err != nil {
		s.writeJSON(writer, http.StatusBadGateway, cli.Error(
			payload.Route,
			"runtime_openclaw_failed",
			fmt.Sprintf("failed to execute OpenClaw command in '%s'", payload.Route.Runtime),
			err.Error(),
		))
		return
	}
	if !result.OK {
		s.writeJSON(writer, http.StatusBadGateway, cli.Error(
			payload.Route,
			"runtime_openclaw_failed",
			fmt.Sprintf("OpenClaw command failed in '%s'", payload.Route.Runtime),
			result.Stdout,
		))
		return
	}
	s.writeJSON(writer, http.StatusOK, result)
}

func (s *Server) handleExecute(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		s.writeJSON(writer, http.StatusMethodNotAllowed, cli.Error(nil, "parse_error", "method not allowed", "use POST /execute"))
		return
	}

	payload, ok := s.parseRouteRequest(writer, request, "send JSON with the parsed route")
	if !ok {
		return
	}

	if payload.Route == nil {
		s.writeJSON(writer, http.StatusBadRequest, cli.Error(nil, "parse_error", "missing route in execute request", "send JSON with the parsed route"))
		return
	}
	if payload.Route.Kind != cli.KindScopedSecrets {
		s.writeJSON(writer, http.StatusBadRequest, cli.Error(payload.Route, "parse_error", "direct execute only supports scoped secrets", "use the documented gateway, environment, or passthrough route"))
		return
	}

	s.logScopedSecretsRequest(request, payload.Route)
	s.writeJSON(writer, http.StatusOK, s.secretHandler.Execute(payload.Route, payload.SecretValue))
}

func (s *Server) logScopedSecretsRequest(request *http.Request, route *cli.Route) {
	if s.logger == nil || route == nil {
		return
	}
	s.logger.Info(
		"scoped secrets request",
		"scope", route.Resource,
		"action", route.Action,
		"name", route.Subject,
		"remote_address", authRemoteKey(request.RemoteAddr),
	)
}

func (s *Server) writeJSON(writer http.ResponseWriter, status int, payload any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = cli.WriteJSON(writer, payload)
}

func (s *Server) parseRouteRequest(writer http.ResponseWriter, request *http.Request, recovery string) (cli.RouteRequest, bool) {
	var payload cli.RouteRequest
	if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
		s.writeJSON(writer, http.StatusBadRequest, cli.Error(nil, "parse_error", "invalid JSON request body", recovery))
		return cli.RouteRequest{}, false
	}
	return payload, true
}

func (s *Server) parseServiceRouteRequest(writer http.ResponseWriter, request *http.Request, action string) (*cli.Route, bool) {
	payload, ok := s.parseRouteRequest(writer, request, "send JSON with the target service")
	if !ok {
		return nil, false
	}

	service := strings.TrimSpace(payload.Service)
	route := &cli.Route{Resource: "service", Kind: cli.KindService, Action: action, Subject: service}
	if payload.Route != nil {
		route = payload.Route
	}
	if strings.TrimSpace(route.Subject) == "" {
		route.Subject = service
	}
	if strings.TrimSpace(route.Subject) == "" {
		s.writeJSON(writer, http.StatusBadRequest, cli.Error(route, "parse_error", "missing service name", fmt.Sprintf("use: service %s <service>", action)))
		return nil, false
	}

	return route, true
}

func (s *Server) parseRuntimePluginInstallRoute(writer http.ResponseWriter, request *http.Request, target runtimePluginRESTTarget, restPath bool) (*cli.Route, bool) {
	if !restPath {
		payload, ok := s.parseRouteRequest(writer, request, "send JSON with the parsed runtime plugin route")
		if !ok {
			return nil, false
		}
		if payload.Route == nil || payload.Route.Kind != cli.KindRuntimePlugin || payload.Route.Action != "install" {
			s.writeJSON(writer, http.StatusBadRequest, cli.Error(payload.Route, "parse_error", "missing runtime plugin install route", "use: dev|test|prod plugin install <package>"))
			return nil, false
		}
		return payload.Route, true
	}

	payload, ok := s.parseRouteRequest(writer, request, "send JSON with the parsed runtime plugin route")
	if !ok {
		return nil, false
	}
	plugin := ""
	if payload.Route != nil {
		plugin = strings.TrimSpace(payload.Route.Subject)
	}
	if plugin == "" {
		s.writeJSON(writer, http.StatusBadRequest, cli.Error(payload.Route, "parse_error", "missing runtime plugin install route", "use: dev|test|prod plugin install <package>"))
		return nil, false
	}

	route, err := runtimePluginRoute(target.Environment, "install", plugin)
	if err != nil {
		s.writeJSON(writer, http.StatusBadRequest, cli.Error(payload.Route, "parse_error", "missing runtime plugin install route", err.Error()))
		return nil, false
	}
	return route, true
}

func (s *Server) parseRuntimePluginRoute(writer http.ResponseWriter, request *http.Request, target runtimePluginRESTTarget, restPath bool, action string) (*cli.Route, bool) {
	if !restPath {
		payload, ok := s.parseRouteRequest(writer, request, "send JSON with the parsed runtime plugin route")
		if !ok {
			return nil, false
		}
		if payload.Route == nil || payload.Route.Kind != cli.KindRuntimePlugin || payload.Route.Action != action {
			s.writeJSON(writer, http.StatusBadRequest, cli.Error(payload.Route, "parse_error", fmt.Sprintf("missing runtime plugin %s route", action), runtimePluginUsage(action)))
			return nil, false
		}
		return payload.Route, true
	}

	route, err := runtimePluginRoute(target.Environment, action, target.Plugin)
	if err != nil {
		s.writeJSON(writer, http.StatusBadRequest, cli.Error(nil, "parse_error", fmt.Sprintf("missing runtime plugin %s route", action), err.Error()))
		return nil, false
	}
	return route, true
}

func parseRuntimePluginRESTPath(requestPath string) (runtimePluginRESTTarget, bool) {
	trimmed := strings.Trim(strings.TrimSpace(requestPath), "/")
	if trimmed == "" {
		return runtimePluginRESTTarget{}, false
	}

	parts := strings.Split(trimmed, "/")
	if len(parts) < 3 || parts[0] != "runtime" || !isRuntimeEnvironment(parts[1]) || parts[2] != "plugins" {
		return runtimePluginRESTTarget{}, false
	}

	switch len(parts) {
	case 3:
		return runtimePluginRESTTarget{Environment: parts[1], Action: "list"}, true
	case 4:
		if parts[3] == "install" {
			return runtimePluginRESTTarget{Environment: parts[1], Action: "install"}, true
		}
		if parts[3] != "" {
			return runtimePluginRESTTarget{Environment: parts[1], Action: "remove", Plugin: parts[3]}, true
		}
	}

	return runtimePluginRESTTarget{}, false
}

func runtimePluginRoute(environment, action, plugin string) (*cli.Route, error) {
	environment = strings.TrimSpace(environment)
	if !isRuntimeEnvironment(environment) {
		return nil, fmt.Errorf("use GET|POST|DELETE /runtime/<dev|test|prod>/plugins")
	}
	runtime := "openclaw-" + environment
	route := &cli.Route{
		Resource:    environment,
		Kind:        cli.KindRuntimePlugin,
		Action:      action,
		Environment: environment,
		Runtime:     runtime,
	}
	if trimmedPlugin := strings.TrimSpace(plugin); trimmedPlugin != "" {
		route.Subject = trimmedPlugin
	}
	return route, nil
}

func runtimePluginUsage(action string) string {
	switch action {
	case "install":
		return "use POST /runtime/<dev|test|prod>/plugins/install"
	case "list":
		return "use GET /runtime/<dev|test|prod>/plugins"
	case "remove":
		return "use DELETE /runtime/<dev|test|prod>/plugins/<plugin>"
	default:
		return "use a documented runtime plugin command"
	}
}

func isRuntimeEnvironment(value string) bool {
	switch strings.TrimSpace(value) {
	case "dev", "test", "prod":
		return true
	default:
		return false
	}
}
