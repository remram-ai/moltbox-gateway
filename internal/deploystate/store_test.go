package deploystate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStoreWritesGatewayStateWithoutLeavingTemps(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := New(root)

	if err := store.AppendDeployment(DeploymentRecord{
		DeploymentID:    "deploy-1",
		Timestamp:       "2026-03-14T20:00:00Z",
		Actor:           "dev",
		Target:          "openclaw-dev",
		ArtifactVersion: "v1",
		Result:          "success",
		Operation:       "service_deploy",
		Runtime:         "openclaw-dev",
	}); err != nil {
		t.Fatalf("AppendDeployment() error = %v", err)
	}
	if err := store.AppendDeployment(DeploymentRecord{
		DeploymentID:    "deploy-2",
		Timestamp:       "2026-03-14T20:01:00Z",
		Actor:           "dev",
		Target:          "openclaw-dev/skill/together-escalation",
		ArtifactVersion: "digest-1",
		Result:          "success",
		Operation:       "runtime_skill_deploy",
		Runtime:         "openclaw-dev",
	}); err != nil {
		t.Fatalf("AppendDeployment() second error = %v", err)
	}

	if err := store.SaveReplayLog("openclaw-dev", ReplayLog{
		Runtime:            "openclaw-dev",
		BaselineCheckpoint: "checkpoint-1",
		Events: []ReplayEvent{
			{
				EventID:       "event-1",
				DeploymentID:  "deploy-2",
				Timestamp:     "2026-03-14T20:01:00Z",
				Runtime:       "openclaw-dev",
				Type:          "skill_install",
				Skill:         "together-escalation",
				PackageDir:    "/srv/moltbox-state/deploy/runtime/openclaw-dev/packages/event-1",
				PackageDigest: "digest-1",
			},
		},
	}); err != nil {
		t.Fatalf("SaveReplayLog() error = %v", err)
	}

	if err := store.SaveCheckpoint("openclaw-dev", CheckpointMetadata{
		Runtime:      "openclaw-dev",
		CheckpointID: "checkpoint-1",
		Timestamp:    "2026-03-14T20:02:00Z",
		Image:        "moltbox-runtime:openclaw-dev-checkpoint-1",
		SnapshotDir:  "/srv/moltbox-state/runtime-baselines/openclaw-dev/checkpoint-1/snapshot",
		DeploymentID: "deploy-3",
		Skills: []CheckpointSkill{
			{Name: "together-escalation", Digest: "digest-1"},
		},
	}); err != nil {
		t.Fatalf("SaveCheckpoint() error = %v", err)
	}

	history, err := store.ReadDeploymentHistory()
	if err != nil {
		t.Fatalf("ReadDeploymentHistory() error = %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("deployment history len = %d, want 2", len(history))
	}

	log, err := store.LoadReplayLog("openclaw-dev")
	if err != nil {
		t.Fatalf("LoadReplayLog() error = %v", err)
	}
	if len(log.Events) != 1 {
		t.Fatalf("replay log = %#v, want one event", log.Events)
	}

	checkpoint, ok, err := store.LoadCheckpoint("openclaw-dev")
	if err != nil {
		t.Fatalf("LoadCheckpoint() error = %v", err)
	}
	if !ok || checkpoint.CheckpointID != "checkpoint-1" {
		t.Fatalf("checkpoint = %#v, ok=%v, want checkpoint-1", checkpoint, ok)
	}

	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		name := info.Name()
		if strings.HasSuffix(name, ".lock") || strings.Contains(name, ".tmp-") {
			t.Fatalf("unexpected temp or lock file left behind: %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Walk() error = %v", err)
	}
}
