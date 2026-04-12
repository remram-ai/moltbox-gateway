package orchestrator

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/remram-ai/moltbox-gateway/internal/command"
	appconfig "github.com/remram-ai/moltbox-gateway/internal/config"
	"github.com/remram-ai/moltbox-gateway/pkg/cli"
)

func TestRuntimeVerifySandbox(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	runtimeRoot := filepath.Join(root, "runtime-state")
	workspaceRoot := filepath.Join(runtimeRoot, "openclaw-test", "workspace")
	if err := os.MkdirAll(workspaceRoot, 0o755); err != nil {
		t.Fatalf("mkdir workspace: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspaceRoot, verifySandboxWriteCheckFile), []byte(verifySandboxWriteCheckValue), 0o644); err != nil {
		t.Fatalf("write workspace proof fixture: %v", err)
	}

	runner := &fakeRunner{
		results: []command.Result{
			{Stdout: `{"agentId":"coder","sandbox":{"mode":"all","scope":"agent","workspaceAccess":"rw","sessionIsSandboxed":true}}`, ExitCode: 0},
			{Stdout: "gateway connect failed: pairing required\n" + `{"payloads":[{"text":"/workspace"}],"meta":{"systemPromptReport":{"workspaceDir":"` + filepath.ToSlash(workspaceRoot) + `","sandbox":{"mode":"all","sandboxed":true}}}}`, ExitCode: 0},
			{Stdout: `{"containers":[{"containerName":"moltbox-dev-sandbox-openclaw-test-agent-coder-abc123","backendId":"docker","sessionKey":"agent:coder","image":"moltbox-dev-sandbox:latest","running":true}]}`, ExitCode: 0},
			{Stdout: "/openclaw-test|moltbox-openclaw-test:local|runtime-host", ExitCode: 0},
			{Stdout: "/moltbox-dev-sandbox-openclaw-test-agent-coder-abc123|moltbox-dev-sandbox:latest|sandbox-host", ExitCode: 0},
			{Stdout: "none", ExitCode: 0},
			{Stdout: "10001:10001", ExitCode: 0},
			{Stdout: `[{"Source":"` + filepath.ToSlash(workspaceRoot) + `","Destination":"/workspace","RW":true}]`, ExitCode: 0},
			{Stdout: "whoami=moltbox\npwd=/workspace\npath=/usr/local/bin:/usr/bin\n", ExitCode: 0},
			{Stdout: `{"status":"ok","headless":true}`, ExitCode: 0},
			{Stdout: `{"status":"ok","headless":false}`, ExitCode: 0},
			{Stdout: "", ExitCode: 0},
			{Stdout: `{"route":{"resource":"gateway"}}` + "\n" + `{"route":{"resource":"service"}}`, ExitCode: 0},
		},
	}

	manager := NewManager(appconfig.Config{
		Paths: appconfig.PathsConfig{
			StateRoot:   filepath.Join(root, "state"),
			RuntimeRoot: runtimeRoot,
			LogsRoot:    filepath.Join(root, "logs"),
		},
	}, fakeInspector{}, runner, nil)

	route := &cli.Route{
		Resource:    "test",
		Kind:        cli.KindRuntimeVerify,
		Action:      "verify",
		Environment: "test",
		Runtime:     "openclaw-test",
		Subject:     "sandbox",
	}

	result, err := manager.RuntimeVerify(context.Background(), route)
	if err != nil {
		t.Fatalf("RuntimeVerify() error = %v", err)
	}
	if !result.OK {
		t.Fatalf("RuntimeVerify() result = %#v, want ok", result)
	}
	if len(result.Steps) != 11 {
		t.Fatalf("len(result.Steps) = %d, want 11", len(result.Steps))
	}

	proofPath := filepath.Join(workspaceRoot, verifySandboxProofFile)
	proofData, err := os.ReadFile(proofPath)
	if err != nil {
		t.Fatalf("read proof file: %v", err)
	}
	for _, fragment := range []string{
		`"runtime": "openclaw-test"`,
		`"sandbox_container": "moltbox-dev-sandbox-openclaw-test-agent-coder-abc123"`,
		`"workspace_write_value": "workspace-write-ok"`,
	} {
		if !strings.Contains(string(proofData), fragment) {
			t.Fatalf("proof file missing %q: %s", fragment, proofData)
		}
	}

	if len(runner.commands) != 13 {
		t.Fatalf("len(runner.commands) = %d, want 13", len(runner.commands))
	}
	if got := runner.commands[0]; len(got) < 6 || got[0] != "docker" || got[1] != "exec" || got[3] != "openclaw" {
		t.Fatalf("first command = %#v, want docker exec runtime openclaw ...", got)
	}
}
