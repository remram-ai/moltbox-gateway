package orchestrator

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	stdruntime "runtime"
	"strings"
	"testing"

	appconfig "github.com/remram-ai/moltbox-gateway/internal/config"
)

func TestPrepareRuntimeDeployRefreshesManagedRootFiles(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	runtimeRepoRoot := filepath.Join(root, "runtime-repo")
	runtimeStateRoot := filepath.Join(root, "runtime-state")

	mustWriteFile(t, filepath.Join(runtimeRepoRoot, "openclaw-test", "openclaw.json.template"), "{\n  \"agents\": {\n    \"defaults\": {\n      \"workspace\": \"{{ runtime_component_dir }}/workspace\"\n    }\n  }\n}\n")
	mustWriteFile(t, filepath.Join(runtimeRepoRoot, "openclaw-test", "model-runtime.yml"), "model: refreshed\n")

	runtimeRoot := filepath.Join(runtimeStateRoot, "openclaw-test")
	mustWriteFile(t, filepath.Join(runtimeRoot, "openclaw.json"), "{\"stale\":true}\n")
	mustWriteFile(t, filepath.Join(runtimeRoot, "model-runtime.yml"), "model: stale\n")

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

	data, err := os.ReadFile(filepath.Join(runtimeRoot, "openclaw.json"))
	if err != nil {
		t.Fatalf("read openclaw.json: %v", err)
	}
	wantWorkspace := filepath.Join(runtimeRoot, "workspace")
	if string(data) == "{\"stale\":true}\n" || !strings.Contains(string(data), filepath.ToSlash(wantWorkspace)) {
		t.Fatalf("openclaw.json = %q, want refreshed managed config with workspace %q", string(data), wantWorkspace)
	}

	modelRuntime, err := os.ReadFile(filepath.Join(runtimeRoot, "model-runtime.yml"))
	if err != nil {
		t.Fatalf("read model-runtime.yml: %v", err)
	}
	if string(modelRuntime) != "model: refreshed\n" {
		t.Fatalf("model-runtime.yml = %q, want refreshed managed file", string(modelRuntime))
	}
}

func TestPrepareRuntimeDeploySyncsManagedWorkspaceBaseline(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	runtimeRepoRoot := filepath.Join(root, "runtime-repo")
	runtimeStateRoot := filepath.Join(root, "runtime-state")

	mustWriteFile(t, filepath.Join(runtimeRepoRoot, "openclaw-test", "workspace", "AGENTS.md"), "# AGENTS\nlean\n")
	mustWriteFile(t, filepath.Join(runtimeRepoRoot, "openclaw-test", "workspace", "MEMORY.md"), "# MEMORY\nbaseline\n")
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

	memoryBaselineData, err := os.ReadFile(filepath.Join(existingWorkspace, "MEMORY.md"))
	if err != nil {
		t.Fatalf("read synced MEMORY.md: %v", err)
	}
	if string(memoryBaselineData) != "# MEMORY\nbaseline\n" {
		t.Fatalf("MEMORY.md = %q, want committed baseline", string(memoryBaselineData))
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

func TestPrepareRuntimeDeployChownsOpenClawTestWorkspaceForSandboxUser(t *testing.T) {
	t.Parallel()

	if stdruntime.GOOS == "windows" {
		t.Skip("ownership checks require unix stat fields")
	}

	root := t.TempDir()
	runtimeRepoRoot := filepath.Join(root, "runtime-repo")
	runtimeStateRoot := filepath.Join(root, "runtime-state")

	mustWriteFile(t, filepath.Join(runtimeRepoRoot, "openclaw-test", "openclaw.json.template"), "{\n  \"agents\": {\n    \"defaults\": {\n      \"workspace\": \"/srv/moltbox-state/runtime/openclaw-test/workspace\"\n    }\n  }\n}\n")
	mustWriteFile(t, filepath.Join(runtimeRepoRoot, "openclaw-test", "workspace", "AGENTS.md"), "# AGENTS\nlean\n")

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

	runtimeRoot := filepath.Join(runtimeStateRoot, "openclaw-test")
	workspaceRoot := filepath.Join(runtimeRoot, "workspace")
	workspaceFile := filepath.Join(workspaceRoot, "AGENTS.md")
	configFile := filepath.Join(runtimeRoot, "openclaw.json")

	assertOwnership(t, workspaceRoot, sandboxWorkspaceUID, sandboxWorkspaceGID)
	assertOwnership(t, workspaceFile, sandboxWorkspaceUID, sandboxWorkspaceGID)
	assertOwnership(t, configFile, runtimeStateUID, runtimeStateGID)
}

func assertOwnership(t *testing.T, path string, wantUID, wantGID int) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	gotUID, ok := statIntField(info, "Uid")
	if !ok {
		t.Fatalf("stat %s missing uid field", path)
	}
	gotGID, ok := statIntField(info, "Gid")
	if !ok {
		t.Fatalf("stat %s missing gid field", path)
	}
	if gotUID != wantUID || gotGID != wantGID {
		t.Fatalf("ownership for %s = %d:%d, want %d:%d", path, gotUID, gotGID, wantUID, wantGID)
	}
}

func statIntField(info os.FileInfo, name string) (int, bool) {
	value := reflect.ValueOf(info.Sys())
	if !value.IsValid() {
		return 0, false
	}
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return 0, false
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return 0, false
	}
	field := value.FieldByName(name)
	if !field.IsValid() {
		return 0, false
	}
	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return int(field.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return int(field.Uint()), true
	default:
		return 0, false
	}
}
