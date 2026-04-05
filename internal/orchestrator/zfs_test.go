package orchestrator

import (
	"context"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	appconfig "github.com/remram-ai/moltbox-gateway/internal/config"
	"github.com/remram-ai/moltbox-gateway/internal/command"
)

func TestSnapshotServiceStateRecordsRecursiveSnapshots(t *testing.T) {
	t.Parallel()
	if runtime.GOOS != "linux" {
		t.Skip("zfs snapshot helper only runs on linux")
	}

	runner := &fakeRunner{
		results: []command.Result{
			{ExitCode: 0, Stdout: "zfs moltbox/state\n"},
			{ExitCode: 0},
			{ExitCode: 0, Stdout: "moltbox/state@service-deploy-openclaw-test-20260405t080500z\nmoltbox/state/repos@service-deploy-openclaw-test-20260405t080500z\n"},
		},
	}
	manager := NewManager(appconfig.Config{
		Paths: appconfig.PathsConfig{
			StateRoot: filepath.Clean("/srv/moltbox-state"),
		},
	}, fakeInspector{}, runner, nil)

	record, err := manager.snapshotServiceState(context.Background(), "deploy", "test")
	if err != nil {
		t.Fatalf("snapshotServiceState() error = %v", err)
	}
	if record == nil {
		t.Fatal("snapshotServiceState() record = nil, want snapshot metadata")
	}
	if record.Dataset != "moltbox/state" {
		t.Fatalf("record.Dataset = %q, want moltbox/state", record.Dataset)
	}
	if len(record.Snapshots) == 0 {
		t.Fatalf("record.Snapshots = %#v, want recursive snapshot metadata", record.Snapshots)
	}
	if got := strings.Join(runner.commands[1], " "); !strings.Contains(got, "zfs snapshot -r moltbox/state@service-deploy-openclaw-test-") {
		t.Fatalf("snapshot command = %q, want recursive zfs snapshot", got)
	}
}

func TestSnapshotServiceStateSkipsNonZFSMount(t *testing.T) {
	t.Parallel()
	if runtime.GOOS != "linux" {
		t.Skip("zfs snapshot helper only runs on linux")
	}

	runner := &fakeRunner{
		results: []command.Result{
			{ExitCode: 0, Stdout: "ext4 /dev/nvme1n1p2\n"},
		},
	}
	manager := NewManager(appconfig.Config{
		Paths: appconfig.PathsConfig{
			StateRoot: filepath.Clean("/srv/moltbox-state"),
		},
	}, fakeInspector{}, runner, nil)

	record, err := manager.snapshotServiceState(context.Background(), "deploy", "test")
	if err != nil {
		t.Fatalf("snapshotServiceState() error = %v", err)
	}
	if record != nil {
		t.Fatalf("snapshotServiceState() record = %#v, want nil for non-zfs mount", record)
	}
	if len(runner.commands) != 1 {
		t.Fatalf("runner.commands len = %d, want 1 findmnt probe", len(runner.commands))
	}
}
