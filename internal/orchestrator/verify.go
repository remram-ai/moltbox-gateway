package orchestrator

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

const (
	verifyBrowserDefaultURL = "https://example.com"
	verifySearchURL         = "http://searxng:8080/search?q=searxng&format=json"
	verifyLocalBaselineID   = "gemma4:e4b-it-q4_K_M"
	verifyLocalBaseline     = "ollama/gemma4:e4b-it-q4_K_M"
	verifyLocalContext      = 131072
)

type runtimeWebConfig struct {
	Agents struct {
		Defaults struct {
			ContextTokens int `json:"contextTokens"`
			Model         struct {
				Primary string `json:"primary"`
			} `json:"model"`
		} `json:"defaults"`
	} `json:"agents"`
	Plugins struct {
		Allow []string `json:"allow"`
		Slots struct {
			Memory string `json:"memory"`
		} `json:"slots"`
		Entries map[string]json.RawMessage `json:"entries"`
	} `json:"plugins"`
	Tools struct {
		Allow []string `json:"allow"`
		Web   struct {
			Search struct {
				Enabled  bool   `json:"enabled"`
				Provider string `json:"provider"`
			} `json:"search"`
			Fetch struct {
				Enabled bool `json:"enabled"`
			} `json:"fetch"`
		} `json:"web"`
	} `json:"tools"`
}

type searxSearchResponse struct {
	Results []struct {
		URL string `json:"url"`
	} `json:"results"`
}

func (m *Manager) RuntimeVerify(ctx context.Context, route *cli.Route) (cli.RuntimeVerifyResult, error) {
	result := cli.RuntimeVerifyResult{
		OK:          true,
		Route:       route,
		Environment: route.Environment,
		Runtime:     route.Runtime,
		Check:       route.Subject,
	}

	switch route.Subject {
	case "runtime":
		m.verifyRuntimeStatus(ctx, route, &result)
	case "browser":
		targetURL := verifyBrowserDefaultURL
		if len(route.NativeArgs) > 0 && strings.TrimSpace(route.NativeArgs[0]) != "" {
			targetURL = strings.TrimSpace(route.NativeArgs[0])
		}
		result.TargetURL = targetURL
		m.verifyRuntimeBrowser(ctx, route, targetURL, &result)
	case "web":
		m.verifyRuntimeWeb(ctx, route, &result)
	case "sandbox":
		m.verifyRuntimeSandbox(ctx, route, &result)
	default:
		return cli.RuntimeVerifyResult{}, fmt.Errorf("unsupported verify check %q", route.Subject)
	}

	if result.OK {
		result.Summary = fmt.Sprintf("%s verify %s passed", route.Environment, route.Subject)
	} else {
		result.Summary = fmt.Sprintf("%s verify %s found one or more failures", route.Environment, route.Subject)
	}
	return result, nil
}

func (m *Manager) verifyRuntimeStatus(ctx context.Context, route *cli.Route, result *cli.RuntimeVerifyResult) {
	statusRoute := &cli.Route{Resource: "service", Kind: cli.KindService, Action: "status", Subject: route.Resource}
	status, err := m.ServiceStatus(ctx, statusRoute, route.Resource)
	if err != nil {
		appendVerifyStep(result, cli.VerifyStepResult{
			Name:    "service-status",
			OK:      false,
			Summary: fmt.Sprintf("failed to inspect service status: %v", err),
		})
		return
	}

	serviceOK := status.Running
	healthValue := ""
	if len(status.Containers) > 0 {
		healthValue = status.Containers[0].Health
		if strings.TrimSpace(healthValue) != "" {
			serviceOK = serviceOK && strings.EqualFold(strings.TrimSpace(healthValue), "healthy")
		}
	}
	appendVerifyStep(result, cli.VerifyStepResult{
		Name:    "service-status",
		OK:      serviceOK,
		Summary: fmt.Sprintf("%s is %s", route.Resource, strings.TrimSpace(status.Status)),
		Details: map[string]string{
			"service": route.Resource,
			"status":  strings.TrimSpace(status.Status),
			"health":  healthValue,
			"image":   strings.TrimSpace(status.Image),
		},
	})

	health := runVerifyOpenClawCommand(ctx, m, route.Runtime, "health", "--json")
	appendVerifyStep(result, commandVerifyStep(
		"runtime-health",
		health,
		health.OK && strings.Contains(health.Stdout, `"ok": true`),
		"OpenClaw health returned ok=true",
	))

	models := runVerifyOpenClawCommand(ctx, m, route.Runtime, "models", "status", "--json")
	appendVerifyStep(result, commandVerifyStep(
		"model-status",
		models,
		models.OK && strings.Contains(models.Stdout, verifyLocalBaselineID),
		"model inventory includes the local Gemma 4 E4B baseline",
	))

	appendVerifyStep(result, m.verifyRuntimeBaselineConfig(ctx, route.Runtime))
}

func (m *Manager) verifyRuntimeBrowser(ctx context.Context, route *cli.Route, targetURL string, result *cli.RuntimeVerifyResult) {
	start := runVerifyOpenClawCommand(ctx, m, route.Runtime, "browser", "start")
	appendVerifyStep(result, commandVerifyStep(
		"browser-start",
		start,
		start.OK && strings.Contains(start.Stdout, "running: true"),
		"native browser starts successfully",
	))

	open := runVerifyOpenClawCommand(ctx, m, route.Runtime, "browser", "open", targetURL)
	appendVerifyStep(result, commandVerifyStep(
		"browser-open",
		open,
		open.OK && strings.Contains(open.Stdout, "opened:"),
		fmt.Sprintf("browser opened %s", targetURL),
	))

	snapshot := runVerifyOpenClawCommand(ctx, m, route.Runtime, "browser", "snapshot")
	snapshotOK := snapshot.OK
	if targetURL == verifyBrowserDefaultURL {
		snapshotOK = snapshotOK && strings.Contains(snapshot.Stdout, "Example Domain")
	}
	appendVerifyStep(result, commandVerifyStep(
		"browser-snapshot",
		snapshot,
		snapshotOK,
		"browser snapshot captured the active page",
	))

	stop := runVerifyOpenClawCommand(ctx, m, route.Runtime, "browser", "stop")
	appendVerifyStep(result, commandVerifyStep(
		"browser-stop",
		stop,
		stop.OK && strings.Contains(stop.Stdout, "running: false"),
		"native browser stops cleanly",
	))
}

func (m *Manager) verifyRuntimeWeb(ctx context.Context, route *cli.Route, result *cli.RuntimeVerifyResult) {
	searxStatusRoute := &cli.Route{Resource: "service", Kind: cli.KindService, Action: "status", Subject: "searxng"}
	searxStatus, err := m.ServiceStatus(ctx, searxStatusRoute, "searxng")
	if err != nil {
		appendVerifyStep(result, cli.VerifyStepResult{
			Name:    "searxng-status",
			OK:      false,
			Summary: fmt.Sprintf("failed to inspect searxng: %v", err),
		})
	} else {
		serviceOK := searxStatus.Running
		healthValue := ""
		if len(searxStatus.Containers) > 0 {
			healthValue = searxStatus.Containers[0].Health
			if strings.TrimSpace(healthValue) != "" {
				serviceOK = serviceOK && strings.EqualFold(strings.TrimSpace(healthValue), "healthy")
			}
		}
		appendVerifyStep(result, cli.VerifyStepResult{
			Name:    "searxng-status",
			OK:      serviceOK,
			Summary: "searxng service is healthy",
			Details: map[string]string{
				"status": strings.TrimSpace(searxStatus.Status),
				"health": healthValue,
			},
		})
	}

	appendVerifyStep(result, m.verifySearXNGHTTP(ctx))
	appendVerifyStep(result, m.verifyWebRuntimeConfig(ctx, route.Runtime))

	result.Caveats = append(result.Caveats,
		"web verify proves backend/config availability, not that the local chat model will reliably choose web_search or web_fetch on its own",
	)
}

func (m *Manager) verifySearXNGHTTP(ctx context.Context) cli.VerifyStepResult {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, verifySearchURL, nil)
	if err != nil {
		return cli.VerifyStepResult{
			Name:    "searxng-http",
			OK:      false,
			Summary: fmt.Sprintf("failed to build SearXNG probe request: %v", err),
		}
	}

	httpClient := &http.Client{Timeout: 15 * time.Second}
	response, err := httpClient.Do(request)
	if err != nil {
		return cli.VerifyStepResult{
			Name:    "searxng-http",
			OK:      false,
			Summary: fmt.Sprintf("failed to query SearXNG: %v", err),
			Details: map[string]string{"url": verifySearchURL},
		}
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 64*1024))
	if err != nil {
		return cli.VerifyStepResult{
			Name:    "searxng-http",
			OK:      false,
			Summary: fmt.Sprintf("failed to read SearXNG response: %v", err),
			Details: map[string]string{"url": verifySearchURL},
		}
	}

	search := searxSearchResponse{}
	ok := response.StatusCode == http.StatusOK && json.Unmarshal(body, &search) == nil && len(search.Results) > 0
	details := map[string]string{
		"url":         verifySearchURL,
		"status_code": fmt.Sprintf("%d", response.StatusCode),
	}
	if len(search.Results) > 0 {
		details["first_result_url"] = search.Results[0].URL
	}

	return cli.VerifyStepResult{
		Name:          "searxng-http",
		OK:            ok,
		Summary:       "SearXNG search endpoint returned results",
		StdoutSnippet: verifySnippet(string(body)),
		Details:       details,
	}
}

func (m *Manager) verifyRuntimeBaselineConfig(ctx context.Context, runtime string) cli.VerifyStepResult {
	cfg, step, ok := m.loadRuntimeConfig(ctx, runtime)
	if !ok {
		step.Name = "runtime-baseline-config"
		return step
	}

	modelPrimary := strings.TrimSpace(cfg.Agents.Defaults.Model.Primary)
	contextTokens := cfg.Agents.Defaults.ContextTokens
	step.Name = "runtime-baseline-config"
	step.OK = strings.EqualFold(modelPrimary, verifyLocalBaseline) && contextTokens == verifyLocalContext
	step.Summary = "runtime config uses the Gemma 4 E4B 128K local baseline"
	step.Details = map[string]string{
		"primary_model":    modelPrimary,
		"context_tokens":   fmt.Sprintf("%d", contextTokens),
		"expected_model":   verifyLocalBaseline,
		"expected_context": fmt.Sprintf("%d", verifyLocalContext),
	}
	return step
}

func (m *Manager) verifyWebRuntimeConfig(ctx context.Context, runtime string) cli.VerifyStepResult {
	cfg, step, ok := m.loadRuntimeConfig(ctx, runtime)
	step.Name = "runtime-web-config"
	if !ok {
		return step
	}

	hasBrowserPlugin := containsString(cfg.Plugins.Allow, "browser")
	hasSearxPlugin := containsString(cfg.Plugins.Allow, "searxng")
	hasOllamaPlugin := containsString(cfg.Plugins.Allow, "ollama")
	hasWebSearchTool := containsString(cfg.Tools.Allow, "web_search")
	hasWebFetchTool := containsString(cfg.Tools.Allow, "web_fetch")
	hasMemoryTools := containsString(cfg.Tools.Allow, "memory_search") ||
		containsString(cfg.Tools.Allow, "memory_get")
	pluginAllow := normalizedDistinctStrings(cfg.Plugins.Allow)
	toolAllow := normalizedDistinctStrings(cfg.Tools.Allow)
	onlyExpectedPlugins := true
	for _, plugin := range pluginAllow {
		if !strings.EqualFold(plugin, "searxng") && !strings.EqualFold(plugin, "ollama") {
			onlyExpectedPlugins = false
			break
		}
	}
	memoryDisabled := strings.EqualFold(strings.TrimSpace(cfg.Plugins.Slots.Memory), "none")
	ok = !hasBrowserPlugin &&
		hasSearxPlugin &&
		(len(pluginAllow) == 1 || (len(pluginAllow) == 2 && hasOllamaPlugin)) &&
		onlyExpectedPlugins &&
		len(toolAllow) == 2 &&
		hasWebSearchTool &&
		hasWebFetchTool &&
		!hasMemoryTools &&
		cfg.Tools.Web.Search.Enabled &&
		cfg.Tools.Web.Search.Provider == "searxng" &&
		memoryDisabled &&
		cfg.Tools.Web.Fetch.Enabled

	step.OK = ok
	step.Summary = "runtime config exposes only web_search and web_fetch with SearXNG-backed search; browser and native memory stay out of the default lane"
	step.Details = map[string]string{
		"search_provider":             cfg.Tools.Web.Search.Provider,
		"search_enabled":              fmt.Sprintf("%t", cfg.Tools.Web.Search.Enabled),
		"fetch_enabled":               fmt.Sprintf("%t", cfg.Tools.Web.Fetch.Enabled),
		"plugins_allow":               strings.Join(pluginAllow, ","),
		"tools_allow":                 strings.Join(toolAllow, ","),
		"memory_slot":                 strings.TrimSpace(cfg.Plugins.Slots.Memory),
		"browser_plugin_enabled":      fmt.Sprintf("%t", hasBrowserPlugin),
		"browser_in_default_toolset":  fmt.Sprintf("%t", containsString(cfg.Tools.Allow, "browser")),
		"memory_tools_in_default_set": fmt.Sprintf("%t", hasMemoryTools),
	}
	return step
}

func (m *Manager) loadRuntimeConfig(ctx context.Context, runtime string) (runtimeWebConfig, cli.VerifyStepResult, bool) {
	commandArgs := []string{"exec", m.runtimeContainerName(runtime), "cat", "/home/node/.openclaw/openclaw.json"}
	result, err := m.runner.Run(ctx, "", "docker", commandArgs...)
	if err != nil {
		return runtimeWebConfig{}, cli.VerifyStepResult{
			Name:    "runtime-config",
			OK:      false,
			Summary: fmt.Sprintf("failed to read runtime config: %v", err),
			Command: append([]string{"docker"}, commandArgs...),
		}, false
	}

	step := cli.VerifyStepResult{
		Name:          "runtime-config",
		OK:            false,
		Command:       append([]string{"docker"}, commandArgs...),
		ExitCode:      result.ExitCode,
		StdoutSnippet: verifySnippet(result.Stdout),
		StderrSnippet: verifySnippet(result.Stderr),
	}
	if result.ExitCode != 0 {
		step.Summary = "failed to read runtime config"
		return runtimeWebConfig{}, step, false
	}

	var cfg runtimeWebConfig
	if err := json.Unmarshal([]byte(result.Stdout), &cfg); err != nil {
		step.Summary = fmt.Sprintf("failed to parse runtime config JSON: %v", err)
		return runtimeWebConfig{}, step, false
	}
	return cfg, step, true
}

func runVerifyOpenClawCommand(ctx context.Context, m *Manager, runtime string, nativeArgs ...string) cli.CommandResult {
	result, err := m.runRuntimeOpenClaw(ctx, runtime, nativeArgs...)
	if err != nil {
		return cli.CommandResult{
			OK:            false,
			ContainerName: runtime,
			Command:       append([]string{"docker", "exec", runtime, "openclaw"}, nativeArgs...),
			ExitCode:      1,
			Stderr:        err.Error(),
		}
	}
	return result
}

func commandVerifyStep(name string, result cli.CommandResult, ok bool, summary string) cli.VerifyStepResult {
	return cli.VerifyStepResult{
		Name:          name,
		OK:            ok,
		Summary:       summary,
		Command:       result.Command,
		ExitCode:      result.ExitCode,
		StdoutSnippet: verifySnippet(result.Stdout),
		StderrSnippet: verifySnippet(result.Stderr),
	}
}

func appendVerifyStep(result *cli.RuntimeVerifyResult, step cli.VerifyStepResult) {
	result.Steps = append(result.Steps, step)
	result.OK = result.OK && step.OK
}

func verifySnippet(text string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return ""
	}
	if len(trimmed) > 600 {
		return trimmed[:600] + "..."
	}
	return trimmed
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item), strings.TrimSpace(target)) {
			return true
		}
	}
	return false
}

func normalizedDistinctStrings(items []string) []string {
	seen := map[string]struct{}{}
	normalized := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	return normalized
}
