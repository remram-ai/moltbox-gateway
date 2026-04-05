package orchestrator

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	appconfig "github.com/remram-ai/moltbox-gateway/internal/config"
)

func TestPrepareRuntimeDeploySyncsManagedWorkspaceBaseline(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	runtimeRepoRoot := filepath.Join(root, "runtime-repo")
	runtimeStateRoot := filepath.Join(root, "runtime-state")

	mustWriteFile(t, filepath.Join(runtimeRepoRoot, "openclaw-test", "workspace", "AGENTS.md"), "# AGENTS\nlean\n")
	mustWriteFile(t, filepath.Join(runtimeRepoRoot, "openclaw-test", "workspace", "SOUL.md"), "# SOUL\nlean\n")

	existingWorkspace := filepath.Join(runtimeStateRoot, "openclaw-test", "workspace")
	mustWriteFile(t, filepath.Join(existingWorkspace, "BOOTSTRAP.md"), "old bootstrap\n")
	mustWriteFile(t, filepath.Join(existingWorkspace, "memory", "2026-04-05.md"), "keep me\n")

	manager := NewManager(appconfig.Config{
		Paths: appconfig.PathsConfig{
			StateRoot:   filepath.Join(root, "state"),
			RuntimeRoot: runtimeStateRoot,
			LogsRoot:    filepath.Join(root, "logs"),
		},
		Repos: appconfig.ReposConfig{
			Runtime: appconfig.RepoConfig{URL: runtimeRepoRoot},
		},
	}, fakeInspector{}, &fakeRunner{}, nil)

	if err := manager.prepareRuntimeDeploy(context.Background(), nil, "openclaw-test"); err != nil {
		t.Fatalf("prepareRuntimeDeploy() error = %v", err)
	}

	agentsData, err := os.ReadFile(filepath.Join(existingWorkspace, "AGENTS.md"))
	if err != nil {
		t.Fatalf("read synced AGENTS.md: %v", err)
	}
	if string(agentsData) != "# AGENTS\nlean\n" {
		t.Fatalf("AGENTS.md = %q, want committed baseline", string(agentsData))
	}

	if _, err := os.Stat(filepath.Join(existingWorkspace, "BOOTSTRAP.md")); !os.IsNotExist(err) {
		t.Fatalf("BOOTSTRAP.md err = %v, want removed when absent from baseline", err)
	}

	memoryData, err := os.ReadFile(filepath.Join(existingWorkspace, "memory", "2026-04-05.md"))
	if err != nil {
		t.Fatalf("read existing memory file: %v", err)
	}
	if string(memoryData) != "keep me\n" {
		t.Fatalf("memory file = %q, want preserved workspace data", string(memoryData))
	}
}
