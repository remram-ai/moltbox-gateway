package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/remram-ai/moltbox-gateway/pkg/cli"
)

const (
	verifySandboxAgentID         = "coder"
	verifySandboxWorkdir         = "/workspace"
	verifySandboxWriteCheckFile  = ".moltbox-sandbox-write-check"
	verifySandboxWriteCheckValue = "workspace-write-ok"
	verifySandboxProofFile       = ".moltbox-sandbox-proof.json"
	verifySandboxGatewayURL      = "http://gateway:7460"
)

type sandboxExplainResponse struct {
	AgentID string `json:"agentId"`
	Sandbox struct {
		Mode               string `json:"mode"`
		Scope              string `json:"scope"`
		WorkspaceAccess    string `json:"workspaceAccess"`
		SessionIsSandboxed bool   `json:"sessionIsSandboxed"`
	} `json:"sandbox"`
}

type sandboxAgentRunResponse struct {
	Payloads []struct {
		Text string `json:"text"`
	} `json:"payloads"`
	Meta struct {
		SystemPromptReport struct {
			WorkspaceDir string `json:"workspaceDir"`
			Sandbox      struct {
				Mode      string `json:"mode"`
				Sandboxed bool   `json:"sandboxed"`
			} `json:"sandbox"`
		} `json:"systemPromptReport"`
	} `json:"meta"`
}

type sandboxListResponse struct {
	Containers []struct {
		ContainerName string `json:"containerName"`
		BackendID     string `json:"backendId"`
		SessionKey    string `json:"sessionKey"`
		Image         string `json:"image"`
		Running       bool   `json:"running"`
	} `json:"containers"`
}

type sandboxMount struct {
	Source      string `json:"Source"`
	Destination string `json:"Destination"`
	RW          bool   `json:"RW"`
}

type sandboxProof struct {
	GeneratedAt         string                 `json:"generated_at"`
	Runtime             string                 `json:"runtime"`
	AgentID             string                 `json:"agent_id"`
	SandboxContainer    string                 `json:"sandbox_container,omitempty"`
	SandboxImage        string                 `json:"sandbox_image,omitempty"`
	SandboxHostname     string                 `json:"sandbox_hostname,omitempty"`
	RuntimeContainer    string                 `json:"runtime_container,omitempty"`
	RuntimeHostname     string                 `json:"runtime_hostname,omitempty"`
	WorkspaceHostPath   string                 `json:"workspace_host_path,omitempty"`
	WorkspaceWritePath  string                 `json:"workspace_write_path,omitempty"`
	WorkspaceWriteValue string                 `json:"workspace_write_value,omitempty"`
	NetworkMode         string                 `json:"network_mode,omitempty"`
	User                string                 `json:"user,omitempty"`
	Steps               []cli.VerifyStepResult `json:"steps"`
}

func (m *Manager) verifyRuntimeSandbox(ctx context.Context, route *cli.Route, result *cli.RuntimeVerifyResult) {
	result.Caveats = append(result.Caveats,
		"the gateway CLI smoke step intentionally uses a separate debug container on moltbox_internal; the real coder sandbox remains network-disabled",
	)

	expectedWorkspace := filepath.ToSlash(filepath.Join(m.config.RuntimeComponentDir(route.Runtime), "workspace"))

	explain := runVerifyOpenClawCommand(ctx, m, route.Runtime, "sandbox", "explain", "--agent", verifySandboxAgentID, "--json")
	explainStep := commandVerifyStep(
		"sandbox-explain",
		explain,
		explain.OK,
		"runtime reports the coder agent as sandboxed",
	)
	if explain.OK {
		var parsed sandboxExplainResponse
		if err := json.Unmarshal([]byte(extractJSONDocument(explain.Stdout)), &parsed); err != nil {
			explainStep.OK = false
			explainStep.Summary = fmt.Sprintf("failed to parse sandbox explain JSON: %v", err)
		} else {
			explainStep.OK = parsed.AgentID == verifySandboxAgentID &&
				parsed.Sandbox.Mode == "all" &&
				parsed.Sandbox.Scope == "agent" &&
				parsed.Sandbox.WorkspaceAccess == "rw" &&
				parsed.Sandbox.SessionIsSandboxed
			explainStep.Details = map[string]string{
				"agent_id":         parsed.AgentID,
				"mode":             parsed.Sandbox.Mode,
				"scope":            parsed.Sandbox.Scope,
				"workspace_access": parsed.Sandbox.WorkspaceAccess,
			}
		}
	}
	appendVerifyStep(result, explainStep)

	coderRun := runVerifyOpenClawCommand(
		ctx,
		m,
		route.Runtime,
		"agent",
		"--agent",
		verifySandboxAgentID,
		"--message",
		"Use the exec tool to run pwd and return only the output.",
		"--json",
	)
	coderStep := commandVerifyStep(
		"coder-pwd",
		coderRun,
		coderRun.OK,
		"real coder-agent execution resolves to /workspace inside the sandbox",
	)
	var runtimeWorkspaceDir string
	if coderRun.OK {
		var parsed sandboxAgentRunResponse
		if err := json.Unmarshal([]byte(extractJSONDocument(coderRun.Stdout)), &parsed); err != nil {
			coderStep.OK = false
			coderStep.Summary = fmt.Sprintf("failed to parse coder-agent JSON: %v", err)
		} else {
			payloadText := strings.TrimSpace(firstSandboxPayloadText(parsed))
			runtimeWorkspaceDir = filepath.ToSlash(strings.TrimSpace(parsed.Meta.SystemPromptReport.WorkspaceDir))
			coderStep.OK = payloadText == verifySandboxWorkdir &&
				runtimeWorkspaceDir == expectedWorkspace &&
				parsed.Meta.SystemPromptReport.Sandbox.Sandboxed &&
				parsed.Meta.SystemPromptReport.Sandbox.Mode == "all"
			coderStep.Details = map[string]string{
				"payload":                 payloadText,
				"workspace_dir":           runtimeWorkspaceDir,
				"expected_cwd":            verifySandboxWorkdir,
				"expected_host_workspace": expectedWorkspace,
			}
		}
	}
	appendVerifyStep(result, coderStep)

	sandboxList := runVerifyOpenClawCommand(ctx, m, route.Runtime, "sandbox", "list", "--json")
	listStep := commandVerifyStep(
		"sandbox-list",
		sandboxList,
		sandboxList.OK,
		"OpenClaw reports an active docker sandbox for the coder agent",
	)
	selected := struct {
		ContainerName string
		Image         string
		Running       bool
	}{}
	if sandboxList.OK {
		var parsed sandboxListResponse
		if err := json.Unmarshal([]byte(extractJSONDocument(sandboxList.Stdout)), &parsed); err != nil {
			listStep.OK = false
			listStep.Summary = fmt.Sprintf("failed to parse sandbox list JSON: %v", err)
		} else {
			for _, container := range parsed.Containers {
				if !strings.EqualFold(strings.TrimSpace(container.BackendID), "docker") {
					continue
				}
				if !strings.Contains(strings.TrimSpace(container.SessionKey), verifySandboxAgentID) {
					continue
				}
				selected.ContainerName = strings.TrimSpace(container.ContainerName)
				selected.Image = strings.TrimSpace(container.Image)
				selected.Running = container.Running
				break
			}
			listStep.OK = selected.ContainerName != ""
			listStep.Details = map[string]string{
				"container": selected.ContainerName,
				"image":     selected.Image,
				"running":   fmt.Sprintf("%t", selected.Running),
			}
		}
	}
	appendVerifyStep(result, listStep)

	if selected.ContainerName == "" {
		m.writeSandboxProof(route.Runtime, sandboxProof{
			GeneratedAt:       time.Now().UTC().Format(time.RFC3339),
			Runtime:           route.Runtime,
			AgentID:           verifySandboxAgentID,
			WorkspaceHostPath: expectedWorkspace,
			Steps:             result.Steps,
		}, result)
		return
	}

	if !selected.Running {
		start := runVerifyDockerCommand(ctx, m, "start", selected.ContainerName)
		startStep := commandVerifyStep(
			"sandbox-start",
			start,
			start.OK,
			"started the selected sandbox container for direct inspection",
		)
		appendVerifyStep(result, startStep)
		selected.Running = start.OK
	}

	proof := sandboxProof{
		GeneratedAt:       time.Now().UTC().Format(time.RFC3339),
		Runtime:           route.Runtime,
		AgentID:           verifySandboxAgentID,
		SandboxContainer:  strings.TrimSpace(selected.ContainerName),
		SandboxImage:      strings.TrimSpace(selected.Image),
		WorkspaceHostPath: expectedWorkspace,
	}

	runtimeContainerName := m.runtimeContainerName(route.Runtime)
	runtimeIdentity := runVerifyDockerCommand(ctx, m, "inspect", runtimeContainerName, "--format", "{{.Name}}|{{.Config.Image}}|{{.Config.Hostname}}")
	sandboxIdentity := runVerifyDockerCommand(ctx, m, "inspect", selected.ContainerName, "--format", "{{.Name}}|{{.Config.Image}}|{{.Config.Hostname}}")
	identityStep := cli.VerifyStepResult{
		Name:    "sandbox-identity",
		OK:      runtimeIdentity.OK && sandboxIdentity.OK,
		Summary: "sandbox container identity differs from the OpenClaw runtime container",
	}
	if runtimeIdentity.OK && sandboxIdentity.OK {
		runtimeParts := splitPipeParts(runtimeIdentity.Stdout, 3)
		sandboxParts := splitPipeParts(sandboxIdentity.Stdout, 3)
		runtimeName := strings.TrimPrefix(runtimeParts[0], "/")
		runtimeImage, runtimeHostname := runtimeParts[1], runtimeParts[2]
		sandboxName := strings.TrimPrefix(sandboxParts[0], "/")
		sandboxImage, sandboxHostname := sandboxParts[1], sandboxParts[2]
		identityStep.OK = runtimeName != "" &&
			sandboxName != "" &&
			runtimeName != sandboxName &&
			runtimeHostname != "" &&
			sandboxHostname != "" &&
			runtimeHostname != sandboxHostname &&
			sandboxImage != ""
		identityStep.Details = map[string]string{
			"runtime_container": runtimeName,
			"runtime_image":     runtimeImage,
			"runtime_hostname":  runtimeHostname,
			"sandbox_container": sandboxName,
			"sandbox_image":     sandboxImage,
			"sandbox_hostname":  sandboxHostname,
		}
		selected.Image = firstString(selected.Image, sandboxImage)
		proof.RuntimeContainer = runtimeName
		proof.RuntimeHostname = runtimeHostname
		proof.SandboxContainer = sandboxName
		proof.SandboxImage = firstString(proof.SandboxImage, sandboxImage)
		proof.SandboxHostname = sandboxHostname
	} else {
		identityStep.Command = sandboxIdentity.Command
		identityStep.ExitCode = sandboxIdentity.ExitCode
		identityStep.StdoutSnippet = verifySnippet(sandboxIdentity.Stdout)
		identityStep.StderrSnippet = verifySnippet(sandboxIdentity.Stderr)
	}
	appendVerifyStep(result, identityStep)

	networkResult := runVerifyDockerCommand(ctx, m, "inspect", selected.ContainerName, "--format", "{{.HostConfig.NetworkMode}}")
	userResult := runVerifyDockerCommand(ctx, m, "inspect", selected.ContainerName, "--format", "{{.Config.User}}")
	mountsResult := runVerifyDockerCommand(ctx, m, "inspect", selected.ContainerName, "--format", "{{json .Mounts}}")
	guardrailStep := cli.VerifyStepResult{
		Name:    "sandbox-guardrails",
		OK:      networkResult.OK && userResult.OK && mountsResult.OK,
		Summary: "sandbox keeps network disabled, runs as the non-root user, and mounts only the workspace",
	}
	if guardrailStep.OK {
		var mounts []sandboxMount
		if err := json.Unmarshal([]byte(strings.TrimSpace(mountsResult.Stdout)), &mounts); err != nil {
			guardrailStep.OK = false
			guardrailStep.Summary = fmt.Sprintf("failed to parse sandbox mounts: %v", err)
		} else {
			workspaceMounted := false
			dockerSocketMounted := false
			for _, mount := range mounts {
				if mount.Destination == verifySandboxWorkdir && filepath.ToSlash(strings.TrimSpace(mount.Source)) == expectedWorkspace && mount.RW {
					workspaceMounted = true
				}
				if mount.Destination == "/var/run/docker.sock" {
					dockerSocketMounted = true
				}
			}
			networkMode := strings.TrimSpace(networkResult.Stdout)
			userValue := strings.TrimSpace(userResult.Stdout)
			guardrailStep.OK = networkMode == "none" && userValue == "10001:10001" && workspaceMounted && !dockerSocketMounted
			guardrailStep.Details = map[string]string{
				"network_mode":      networkMode,
				"user":              userValue,
				"workspace_dir":     expectedWorkspace,
				"workspace_mounted": fmt.Sprintf("%t", workspaceMounted),
				"docker_socket":     fmt.Sprintf("%t", dockerSocketMounted),
			}
			proof.NetworkMode = networkMode
			proof.User = userValue
		}
	}
	appendVerifyStep(result, guardrailStep)

	toolchain := runVerifyDockerCommand(ctx, m, "exec", selected.ContainerName, "sh", "-lc", "/opt/moltbox/dev-sandbox/scripts/toolchain-smoke.sh")
	toolchainStep := commandVerifyStep(
		"sandbox-toolchain",
		toolchain,
		toolchain.OK &&
			strings.Contains(toolchain.Stdout, "whoami=moltbox") &&
			strings.Contains(toolchain.Stdout, "pwd=/workspace") &&
			strings.Contains(toolchain.Stdout, "path="),
		"toolchain commands resolve in non-interactive sh -lc inside the sandbox",
	)
	appendVerifyStep(result, toolchainStep)

	headless := runVerifyDockerCommand(ctx, m, "exec", selected.ContainerName, "node", "/opt/moltbox/dev-sandbox/scripts/playwright-smoke.mjs")
	headlessStep := commandVerifyStep(
		"sandbox-playwright-headless",
		headless,
		headless.OK &&
			strings.Contains(headless.Stdout, `"status":"ok"`) &&
			strings.Contains(headless.Stdout, `"headless":true`),
		"Playwright launches Chromium headless inside the sandbox",
	)
	appendVerifyStep(result, headlessStep)

	headful := runVerifyDockerCommand(ctx, m, "exec", selected.ContainerName, "sh", "-lc", "/opt/moltbox/dev-sandbox/scripts/playwright-headful-smoke.sh")
	headfulStep := commandVerifyStep(
		"sandbox-playwright-headful",
		headful,
		headful.OK &&
			strings.Contains(headful.Stdout, `"status":"ok"`) &&
			strings.Contains(headful.Stdout, `"headless":false`),
		"Playwright launches Chromium headful via xvfb-run inside the sandbox",
	)
	appendVerifyStep(result, headfulStep)

	writeCheckContainerPath := filepath.ToSlash(filepath.Join(verifySandboxWorkdir, verifySandboxWriteCheckFile))
	writeCheckHostPath := filepath.Join(m.config.RuntimeComponentDir(route.Runtime), "workspace", verifySandboxWriteCheckFile)
	writeCheck := runVerifyDockerCommand(
		ctx,
		m,
		"exec",
		selected.ContainerName,
		"sh",
		"-lc",
		fmt.Sprintf("printf %s > %s", verifySandboxWriteCheckValue, writeCheckContainerPath),
	)
	writeStep := commandVerifyStep(
		"sandbox-workspace-write",
		writeCheck,
		writeCheck.OK,
		"sandbox writes land on the host-backed workspace",
	)
	if writeCheck.OK {
		data, err := os.ReadFile(writeCheckHostPath)
		if err != nil {
			writeStep.OK = false
			writeStep.Summary = fmt.Sprintf("failed to read host workspace proof: %v", err)
		} else {
			writeValue := strings.TrimSpace(string(data))
			writeStep.OK = writeValue == verifySandboxWriteCheckValue
			writeStep.Details = map[string]string{
				"host_path": writeCheckHostPath,
				"value":     writeValue,
			}
			proof.WorkspaceWritePath = writeCheckHostPath
			proof.WorkspaceWriteValue = writeValue
		}
	}
	appendVerifyStep(result, writeStep)

	if strings.TrimSpace(selected.Image) == "" {
		appendVerifyStep(result, cli.VerifyStepResult{
			Name:    "sandbox-gateway-cli",
			OK:      false,
			Summary: "missing sandbox image for the debug-networked gateway CLI smoke test",
		})
		m.writeSandboxProof(route.Runtime, proof, result)
		return
	}

	debugGatewayCLI := runVerifyDockerCommand(
		ctx,
		m,
		"run",
		"--rm",
		"--network",
		internalNetworkName,
		"-e",
		"MOLTBOX_GATEWAY_URL="+verifySandboxGatewayURL,
		selected.Image,
		"sh",
		"-lc",
		"/opt/moltbox/dev-sandbox/scripts/moltbox-cli-smoke.sh",
	)
	debugStep := commandVerifyStep(
		"sandbox-gateway-cli",
		debugGatewayCLI,
		debugGatewayCLI.OK &&
			strings.Contains(debugGatewayCLI.Stdout, `"gateway"`) &&
			strings.Contains(debugGatewayCLI.Stdout, `"service"`),
		"the sandbox image can use the Moltbox Gateway CLI in a separate debug-networked launch",
	)
	appendVerifyStep(result, debugStep)

	m.writeSandboxProof(route.Runtime, proof, result)
}

func firstSandboxPayloadText(response sandboxAgentRunResponse) string {
	if len(response.Payloads) == 0 {
		return ""
	}
	return strings.TrimSpace(response.Payloads[0].Text)
}

func extractJSONDocument(text string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" || json.Valid([]byte(trimmed)) {
		return trimmed
	}
	for index := len(trimmed) - 1; index >= 0; index-- {
		if trimmed[index] != '{' && trimmed[index] != '[' {
			continue
		}
		candidate := strings.TrimSpace(trimmed[index:])
		if json.Valid([]byte(candidate)) {
			return candidate
		}
	}
	return trimmed
}

func splitPipeParts(text string, want int) []string {
	parts := strings.SplitN(strings.TrimSpace(text), "|", want)
	for len(parts) < want {
		parts = append(parts, "")
	}
	for index := 0; parts != nil && index < len(parts); index++ {
		parts[index] = strings.TrimSpace(parts[index])
	}
	return parts
}

func firstString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func runVerifyDockerCommand(ctx context.Context, m *Manager, args ...string) cli.CommandResult {
	result, err := m.runner.Run(ctx, "", "docker", args...)
	if err != nil {
		return cli.CommandResult{
			OK:       false,
			Command:  append([]string{"docker"}, args...),
			ExitCode: 1,
			Stderr:   err.Error(),
		}
	}
	return cli.CommandResult{
		OK:       result.ExitCode == 0,
		Command:  append([]string{"docker"}, args...),
		Stdout:   result.Stdout,
		Stderr:   result.Stderr,
		ExitCode: result.ExitCode,
	}
}

func (m *Manager) writeSandboxProof(runtime string, proof sandboxProof, result *cli.RuntimeVerifyResult) {
	proof.Steps = append([]cli.VerifyStepResult(nil), result.Steps...)
	proofPath := filepath.Join(m.config.RuntimeComponentDir(runtime), "workspace", verifySandboxProofFile)
	data, err := json.MarshalIndent(proof, "", "  ")
	step := cli.VerifyStepResult{
		Name:    "sandbox-proof-file",
		OK:      err == nil,
		Summary: "sandbox verification proof file written to the runtime workspace",
		Details: map[string]string{"path": proofPath},
	}
	if err == nil {
		err = os.WriteFile(proofPath, append(data, '\n'), 0o644)
		step.OK = err == nil
	}
	if err != nil {
		step.Summary = fmt.Sprintf("failed to write sandbox proof file: %v", err)
	}
	appendVerifyStep(result, step)
}
