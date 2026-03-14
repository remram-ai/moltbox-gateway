package orchestrator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/remram-ai/moltbox-gateway/internal/deploystate"
	"github.com/remram-ai/moltbox-gateway/pkg/cli"
)

const defaultRuntimeImage = "ghcr.io/openclaw/openclaw:latest"

type deployableSkill struct {
	Name      string
	SourceDir string
	Digest    string
}

func isRuntimeService(service string) bool {
	switch canonicalServiceName(service) {
	case "openclaw-dev", "openclaw-test", "openclaw-prod":
		return true
	default:
		return false
	}
}

func (m *Manager) prepareRuntimeDeploy(_ context.Context, route *cli.Route, service string) error {
	if err := m.restoreRuntimeBaseline(service); err != nil {
		return err
	}
	return m.registerRuntimeDeployEvents(route, service)
}

func (m *Manager) restoreRuntimeBaseline(service string) error {
	destination := m.config.RuntimeComponentDir(service)
	checkpoint, ok, err := m.stateStore.LoadCheckpoint(service)
	if err != nil {
		return err
	}

	if err := os.RemoveAll(destination); err != nil {
		return fmt.Errorf("reset runtime state for %s: %w", service, err)
	}
	if err := os.MkdirAll(destination, 0o755); err != nil {
		return fmt.Errorf("create runtime state dir for %s: %w", service, err)
	}

	if !ok || strings.TrimSpace(checkpoint.SnapshotDir) == "" {
		return nil
	}
	if err := copyTree(checkpoint.SnapshotDir, destination); err != nil {
		return fmt.Errorf("restore checkpoint snapshot for %s: %w", service, err)
	}
	return nil
}

func (m *Manager) registerRuntimeDeployEvents(route *cli.Route, service string) error {
	skills, err := m.discoverPureSkills()
	if err != nil {
		return err
	}
	if len(skills) == 0 {
		return nil
	}

	checkpoint, _, err := m.stateStore.LoadCheckpoint(service)
	if err != nil {
		return err
	}
	log, err := m.stateStore.LoadReplayLog(service)
	if err != nil {
		return err
	}

	effective := checkpointSkillState(checkpoint)
	for _, event := range log.Events {
		if event.Type != "skill_install" || strings.TrimSpace(event.Skill) == "" {
			continue
		}
		effective[event.Skill] = event.PackageDigest
	}

	changed := false
	for _, skill := range skills {
		if effective[skill.Name] == skill.Digest {
			continue
		}

		deploymentID := newGatewayID("deploy")
		eventID := newGatewayID("event")
		stagedDir, err := m.stateStore.StageReplayPackage(service, eventID, skill.SourceDir)
		if err != nil {
			return err
		}

		event := deploystate.ReplayEvent{
			EventID:       eventID,
			DeploymentID:  deploymentID,
			Timestamp:     time.Now().UTC().Format(time.RFC3339),
			Runtime:       service,
			Type:          "skill_install",
			Skill:         skill.Name,
			PackageDir:    stagedDir,
			PackageDigest: skill.Digest,
			ContainerPath: filepath.ToSlash(filepath.Join("/home/node/.openclaw/skills", skill.Name)),
		}
		log.Events = append(log.Events, event)
		changed = true

		record := deploystate.DeploymentRecord{
			DeploymentID:    deploymentID,
			Timestamp:       event.Timestamp,
			Actor:           deploymentActor(route),
			Target:          service + "/skill/" + skill.Name,
			ArtifactVersion: skill.Digest,
			PreviousVersion: effective[skill.Name],
			Result:          "recorded",
			Operation:       "runtime_skill_deploy",
			Runtime:         service,
			Details: map[string]string{
				"event_id":    eventID,
				"package_dir": stagedDir,
			},
		}
		if err := m.stateStore.AppendDeployment(record); err != nil {
			return err
		}
		effective[skill.Name] = skill.Digest
	}

	if !changed {
		return nil
	}
	return m.stateStore.SaveReplayLog(service, log)
}

func (m *Manager) replayRuntimeDeployHistory(ctx context.Context, route *cli.Route, service string) error {
	log, err := m.stateStore.LoadReplayLog(service)
	if err != nil {
		return err
	}
	for _, event := range log.Events {
		if err := m.executeReplayEvent(ctx, service, event); err != nil {
			return err
		}
	}

	if len(log.Events) == 0 {
		return nil
	}

	record := deploystate.DeploymentRecord{
		DeploymentID:    newGatewayID("deploy"),
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
		Actor:           deploymentActor(route),
		Target:          service,
		ArtifactVersion: m.selectedRuntimeImage(service),
		Result:          "replayed",
		Operation:       "runtime_replay",
		Runtime:         service,
		Details: map[string]string{
			"event_count": fmt.Sprintf("%d", len(log.Events)),
		},
	}
	return m.stateStore.AppendDeployment(record)
}

func (m *Manager) executeReplayEvent(ctx context.Context, service string, event deploystate.ReplayEvent) error {
	switch event.Type {
	case "skill_install":
		return m.installSkillFromGatewayState(ctx, service, event)
	default:
		return fmt.Errorf("unsupported replay event type %q for %s", event.Type, service)
	}
}

func (m *Manager) installSkillFromGatewayState(ctx context.Context, service string, event deploystate.ReplayEvent) error {
	if strings.TrimSpace(event.PackageDir) == "" {
		return fmt.Errorf("replay event %s is missing package dir", event.EventID)
	}
	if _, err := os.Stat(event.PackageDir); err != nil {
		return fmt.Errorf("replay event %s package dir unavailable: %w", event.EventID, err)
	}

	destination := strings.TrimSpace(event.ContainerPath)
	if destination == "" {
		destination = filepath.ToSlash(filepath.Join("/home/node/.openclaw/skills", event.Skill))
	}
	command := fmt.Sprintf("rm -rf %s && mkdir -p %s", shellQuote(destination), shellQuote(destination))
	resetResult, err := m.runner.Run(ctx, "", "docker", "exec", service, "sh", "-lc", command)
	if err != nil {
		return fmt.Errorf("reset skill destination for %s: %w", event.Skill, err)
	}
	if resetResult.ExitCode != 0 {
		return fmt.Errorf("reset skill destination for %s failed: %s", event.Skill, strings.TrimSpace(resetResult.Stdout))
	}

	copySource := filepath.Join(event.PackageDir, ".")
	copyResult, err := m.runner.Run(ctx, "", "docker", "cp", copySource, fmt.Sprintf("%s:%s", service, destination))
	if err != nil {
		return fmt.Errorf("copy skill package for %s: %w", event.Skill, err)
	}
	if copyResult.ExitCode != 0 {
		return fmt.Errorf("copy skill package for %s failed: %s", event.Skill, strings.TrimSpace(copyResult.Stdout))
	}
	return nil
}

func (m *Manager) RuntimeCheckpoint(ctx context.Context, route *cli.Route) (cli.RuntimeCheckpointResult, error) {
	service := canonicalServiceName(route.Resource)
	if strings.TrimSpace(route.Runtime) != "" {
		service = canonicalServiceName(route.Runtime)
	}
	if !isRuntimeService(service) {
		return cli.RuntimeCheckpointResult{}, fmt.Errorf("checkpoint is only supported for runtime services")
	}

	checkpointID := newGatewayID("checkpoint")
	selectedImage := m.selectedRuntimeImage(service)
	if selectedImage == "" {
		selectedImage = defaultRuntimeImage
	}

	snapshotDir := m.stateStore.CheckpointSnapshotDir(service, checkpointID)
	if err := m.captureRuntimeState(ctx, service, snapshotDir); err != nil {
		return cli.RuntimeCheckpointResult{}, err
	}

	checkpointImage, err := m.buildCheckpointImage(ctx, service, checkpointID, selectedImage, snapshotDir)
	if err != nil {
		return cli.RuntimeCheckpointResult{}, err
	}

	previousCheckpoint, hadCheckpoint, err := m.stateStore.LoadCheckpoint(service)
	if err != nil {
		return cli.RuntimeCheckpointResult{}, err
	}
	previousReplayLog, err := m.stateStore.LoadReplayLog(service)
	if err != nil {
		return cli.RuntimeCheckpointResult{}, err
	}

	skills, err := m.currentCheckpointSkills(service)
	if err != nil {
		return cli.RuntimeCheckpointResult{}, err
	}
	metadata := deploystate.CheckpointMetadata{
		Runtime:      service,
		CheckpointID: checkpointID,
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		Image:        checkpointImage,
		SourceImage:  selectedImage,
		SnapshotDir:  snapshotDir,
		DeploymentID: newGatewayID("deploy"),
		Skills:       skills,
	}
	if err := m.stateStore.SaveCheckpoint(service, metadata); err != nil {
		return cli.RuntimeCheckpointResult{}, err
	}
	if err := m.stateStore.ClearReplayLog(service, checkpointID); err != nil {
		return cli.RuntimeCheckpointResult{}, err
	}

	if _, err := m.DeployService(ctx, route, service); err != nil {
		if hadCheckpoint {
			_ = m.stateStore.SaveCheckpoint(service, previousCheckpoint)
		} else {
			_ = m.stateStore.DeleteCheckpoint(service)
		}
		_ = m.stateStore.SaveReplayLog(service, previousReplayLog)
		return cli.RuntimeCheckpointResult{}, err
	}

	record := deploystate.DeploymentRecord{
		DeploymentID:    metadata.DeploymentID,
		Timestamp:       metadata.Timestamp,
		Actor:           deploymentActor(route),
		Target:          service,
		ArtifactVersion: checkpointImage,
		PreviousVersion: selectedImage,
		Result:          "success",
		Operation:       "runtime_checkpoint",
		Runtime:         service,
		Details: map[string]string{
			"checkpoint_id": checkpointID,
			"snapshot_dir":  snapshotDir,
		},
	}
	if err := m.stateStore.AppendDeployment(record); err != nil {
		return cli.RuntimeCheckpointResult{}, err
	}

	return cli.RuntimeCheckpointResult{
		OK:            true,
		Route:         route,
		Runtime:       service,
		CheckpointID:  checkpointID,
		Image:         checkpointImage,
		SnapshotDir:   snapshotDir,
		ReplayCleared: true,
	}, nil
}

func (m *Manager) captureRuntimeState(ctx context.Context, service, snapshotDir string) error {
	if err := os.RemoveAll(snapshotDir); err != nil {
		return fmt.Errorf("reset checkpoint snapshot dir for %s: %w", service, err)
	}
	if err := os.MkdirAll(snapshotDir, 0o755); err != nil {
		return fmt.Errorf("create checkpoint snapshot dir for %s: %w", service, err)
	}

	result, err := m.runner.Run(ctx, "", "docker", "cp", fmt.Sprintf("%s:/home/node/.openclaw/.", service), snapshotDir)
	if err != nil {
		return fmt.Errorf("capture runtime state for %s: %w", service, err)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("capture runtime state for %s failed: %s", service, strings.TrimSpace(result.Stdout))
	}
	return nil
}

func (m *Manager) buildCheckpointImage(ctx context.Context, service, checkpointID, baseImage, snapshotDir string) (string, error) {
	buildDir := m.stateStore.CheckpointBuildContextDir(service, checkpointID)
	if err := os.RemoveAll(buildDir); err != nil {
		return "", fmt.Errorf("reset checkpoint build dir for %s: %w", service, err)
	}
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		return "", fmt.Errorf("create checkpoint build dir for %s: %w", service, err)
	}

	stateDir := filepath.Join(buildDir, "runtime-state")
	if err := copyTree(snapshotDir, stateDir); err != nil {
		return "", fmt.Errorf("stage checkpoint snapshot for %s: %w", service, err)
	}
	dockerfile := strings.Join([]string{
		"ARG BASE_IMAGE",
		"FROM ${BASE_IMAGE}",
		"ARG RUNTIME_NAME",
		"ARG CHECKPOINT_ID",
		"LABEL org.remram.runtime=${RUNTIME_NAME}",
		"LABEL org.remram.checkpoint=${CHECKPOINT_ID}",
		"COPY runtime-state /opt/moltbox/runtime-baseline/${RUNTIME_NAME}",
	}, "\n") + "\n"
	if err := os.WriteFile(filepath.Join(buildDir, "Dockerfile"), []byte(dockerfile), 0o644); err != nil {
		return "", fmt.Errorf("write checkpoint Dockerfile for %s: %w", service, err)
	}

	image := fmt.Sprintf("moltbox-runtime:%s-%s", service, checkpointID)
	result, err := m.runner.Run(ctx, buildDir, "docker", "build", "-t", image, "--build-arg", "BASE_IMAGE="+baseImage, "--build-arg", "RUNTIME_NAME="+service, "--build-arg", "CHECKPOINT_ID="+checkpointID, buildDir)
	if err != nil {
		return "", fmt.Errorf("build checkpoint image for %s: %w", service, err)
	}
	if result.ExitCode != 0 {
		return "", fmt.Errorf("build checkpoint image for %s failed: %s", service, strings.TrimSpace(result.Stdout))
	}
	return image, nil
}

func (m *Manager) currentCheckpointSkills(service string) ([]deploystate.CheckpointSkill, error) {
	checkpoint, _, err := m.stateStore.LoadCheckpoint(service)
	if err != nil {
		return nil, err
	}
	state := checkpointSkillState(checkpoint)
	logSkills, err := m.stateStore.ReplaySkillState(service)
	if err != nil {
		return nil, err
	}
	for name, skill := range logSkills {
		state[name] = skill.Digest
	}

	names := make([]string, 0, len(state))
	for name := range state {
		names = append(names, name)
	}
	sort.Strings(names)

	skills := make([]deploystate.CheckpointSkill, 0, len(names))
	for _, name := range names {
		skills = append(skills, deploystate.CheckpointSkill{Name: name, Digest: state[name]})
	}
	return skills, nil
}

func (m *Manager) selectedRuntimeImage(service string) string {
	if !isRuntimeService(service) {
		return ""
	}
	checkpoint, ok, err := m.stateStore.LoadCheckpoint(service)
	if err != nil || !ok {
		return ""
	}
	return strings.TrimSpace(checkpoint.Image)
}

func (m *Manager) recordServiceDeployment(route *cli.Route, service, result string) error {
	record := deploystate.DeploymentRecord{
		DeploymentID:    newGatewayID("deploy"),
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
		Actor:           deploymentActor(route),
		Target:          service,
		ArtifactVersion: serviceArtifactVersion(service, m.selectedRuntimeImage(service)),
		Result:          result,
		Operation:       "service_deploy",
		Runtime: func() string {
			if isRuntimeService(service) {
				return service
			}
			return ""
		}(),
	}
	return m.stateStore.AppendDeployment(record)
}

func (m *Manager) discoverPureSkills() ([]deployableSkill, error) {
	skillsRoot := filepath.Join(m.config.SkillsRepoRoot(), "skills")
	if strings.TrimSpace(m.config.SkillsRepoRoot()) == "" {
		return nil, nil
	}

	entries, err := os.ReadDir(skillsRoot)
	if err != nil {
		if errorsIsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read skills repo: %w", err)
	}

	skills := make([]deployableSkill, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		sourceDir := filepath.Join(skillsRoot, entry.Name())
		skillFile := filepath.Join(sourceDir, "SKILL.md")
		if info, err := os.Stat(skillFile); err != nil || info.IsDir() {
			if err == nil || errorsIsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("stat skill %s: %w", entry.Name(), err)
		}
		if _, err := os.Stat(filepath.Join(sourceDir, "openclaw.plugin.json")); err == nil {
			continue
		} else if err != nil && !errorsIsNotExist(err) {
			return nil, fmt.Errorf("stat plugin manifest for %s: %w", entry.Name(), err)
		}

		digest, err := m.stateStore.DirectoryDigest(sourceDir)
		if err != nil {
			return nil, fmt.Errorf("digest skill %s: %w", entry.Name(), err)
		}
		skills = append(skills, deployableSkill{
			Name:      entry.Name(),
			SourceDir: sourceDir,
			Digest:    digest,
		})
	}

	sort.Slice(skills, func(i, j int) bool {
		return skills[i].Name < skills[j].Name
	})
	return skills, nil
}

func checkpointSkillState(checkpoint deploystate.CheckpointMetadata) map[string]string {
	state := make(map[string]string, len(checkpoint.Skills))
	for _, skill := range checkpoint.Skills {
		if strings.TrimSpace(skill.Name) == "" {
			continue
		}
		state[skill.Name] = skill.Digest
	}
	return state
}

func serviceArtifactVersion(service, image string) string {
	if isRuntimeService(service) {
		if strings.TrimSpace(image) != "" {
			return image
		}
		return defaultRuntimeImage
	}
	return service + ":latest"
}

func deploymentActor(route *cli.Route) string {
	if route == nil {
		return "gateway"
	}
	if strings.TrimSpace(route.Resource) == "" {
		return "gateway"
	}
	return route.Resource
}

func newGatewayID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UTC().UnixNano())
}
