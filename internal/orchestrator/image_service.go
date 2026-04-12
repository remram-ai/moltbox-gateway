package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/remram-ai/moltbox-gateway/internal/deploystate"
	"github.com/remram-ai/moltbox-gateway/pkg/cli"
)

const (
	serviceKindCompose = "compose"
	serviceKindImage   = "image"
)

type imageServiceMetadata struct {
	Service            string `json:"service"`
	ImageName          string `json:"image_name"`
	LatestTag          string `json:"latest_tag"`
	ImmutableTag       string `json:"immutable_tag"`
	BuildContextDigest string `json:"build_context_digest,omitempty"`
	BuiltAt            string `json:"built_at"`
}

func isImageServiceDefinition(definition ServiceDefinition) bool {
	return strings.EqualFold(strings.TrimSpace(definition.Kind), serviceKindImage)
}

func (m *Manager) deployImageService(ctx context.Context, route *cli.Route, service string, definition ServiceDefinition) (cli.ServiceDeployResult, error) {
	outputDir, _, err := m.RenderServiceAssets(service, definition)
	if err != nil {
		return cli.ServiceDeployResult{}, err
	}

	if _, err := os.Stat(filepath.Join(outputDir, "Dockerfile")); err != nil {
		return cli.ServiceDeployResult{}, fmt.Errorf("image-backed service %s requires a Dockerfile in %s", service, outputDir)
	}

	logPath := m.imageServiceLogPath(service)
	metadataPath := m.imageServiceMetadataPath(service)
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return cli.ServiceDeployResult{}, fmt.Errorf("create image service log dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(metadataPath), 0o755); err != nil {
		return cli.ServiceDeployResult{}, fmt.Errorf("create image service metadata dir: %w", err)
	}

	if err := m.prepareImageServiceSourceContext(service, outputDir); err != nil {
		return cli.ServiceDeployResult{}, err
	}

	digest, err := m.stateStore.DirectoryDigest(outputDir)
	if err != nil {
		return cli.ServiceDeployResult{}, fmt.Errorf("compute image build context digest: %w", err)
	}
	immutableTag := imageImmutableTag(definition.ImageName, digest)
	latestTag := imageLatestTag(definition)
	commandArgs := []string{"build", "-f", filepath.Join(outputDir, "Dockerfile"), "-t", latestTag, "-t", immutableTag, outputDir}
	var logBuilder strings.Builder

	prebuildOutput, err := m.prepareImageServiceBinaryAssets(ctx, service, outputDir)
	appendImageServiceLog(&logBuilder, prebuildOutput)
	if err != nil {
		_ = os.WriteFile(logPath, []byte(logBuilder.String()), 0o644)
		return cli.ServiceDeployResult{}, err
	}

	result, err := m.runner.Run(ctx, outputDir, "docker", commandArgs...)
	if err != nil {
		appendImageServiceLog(&logBuilder, result.Stdout)
		_ = os.WriteFile(logPath, []byte(logBuilder.String()), 0o644)
		return cli.ServiceDeployResult{}, err
	}
	appendImageServiceLog(&logBuilder, result.Stdout)
	if writeErr := os.WriteFile(logPath, []byte(logBuilder.String()), 0o644); writeErr != nil {
		return cli.ServiceDeployResult{}, fmt.Errorf("write image service build log: %w", writeErr)
	}
	if result.ExitCode != 0 {
		return cli.ServiceDeployResult{}, fmt.Errorf("docker build failed: %s", strings.TrimSpace(logBuilder.String()))
	}

	metadata := imageServiceMetadata{
		Service:            service,
		ImageName:          definition.ImageName,
		LatestTag:          latestTag,
		ImmutableTag:       immutableTag,
		BuildContextDigest: digest,
		BuiltAt:            time.Now().UTC().Format(time.RFC3339),
	}
	if err := m.saveImageServiceMetadata(service, metadata); err != nil {
		return cli.ServiceDeployResult{}, err
	}

	record := deploystate.DeploymentRecord{
		DeploymentID:    newGatewayID("deploy"),
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
		Actor:           deploymentActor(route),
		Target:          service,
		ArtifactVersion: immutableTag,
		Result:          "success",
		Operation:       "service_deploy",
		Details: map[string]string{
			"service_kind":  serviceKindImage,
			"log_path":      logPath,
			"metadata_path": metadataPath,
			"latest_tag":    latestTag,
		},
	}
	if err := m.stateStore.AppendDeployment(record); err != nil {
		return cli.ServiceDeployResult{}, fmt.Errorf("record image service deployment: %w", err)
	}

	return cli.ServiceDeployResult{
		OK:              true,
		Route:           route,
		Service:         service,
		ServiceKind:     serviceKindImage,
		ComposeProject:  definition.ComposeProject,
		OutputDir:       outputDir,
		ArtifactVersion: immutableTag,
		LogPath:         logPath,
		MetadataPath:    metadataPath,
		Command:         append([]string{"docker"}, commandArgs...),
	}, nil
}

func (m *Manager) removeImageService(ctx context.Context, route *cli.Route, service string, definition ServiceDefinition) (cli.ServiceActionResult, error) {
	metadata, ok, err := m.loadImageServiceMetadata(service)
	if err != nil {
		return cli.ServiceActionResult{}, err
	}

	tags := []string{imageLatestTag(definition)}
	if ok && strings.TrimSpace(metadata.ImmutableTag) != "" && !strings.EqualFold(metadata.ImmutableTag, tags[0]) {
		tags = append(tags, metadata.ImmutableTag)
	}

	commandArgs := []string{"image", "rm", "-f"}
	commandArgs = append(commandArgs, tags...)
	result, err := m.runner.Run(ctx, "", "docker", commandArgs...)
	if err != nil {
		return cli.ServiceActionResult{}, err
	}
	if result.ExitCode != 0 && ok {
		return cli.ServiceActionResult{}, fmt.Errorf("docker image rm failed: %s", strings.TrimSpace(result.Stdout))
	}

	artifactVersion := m.currentImageServiceArtifactVersion(service, definition)
	if ok && strings.TrimSpace(metadata.ImmutableTag) != "" {
		artifactVersion = metadata.ImmutableTag
	}
	if err := os.Remove(m.imageServiceMetadataPath(service)); err != nil && !errorsIsNotExist(err) {
		return cli.ServiceActionResult{}, fmt.Errorf("remove image service metadata: %w", err)
	}

	record := deploystate.DeploymentRecord{
		DeploymentID:    newGatewayID("deploy"),
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
		Actor:           deploymentActor(route),
		Target:          service,
		ArtifactVersion: artifactVersion,
		Result:          "success",
		Operation:       "service_remove",
		Details: map[string]string{
			"service_kind":  serviceKindImage,
			"log_path":      m.imageServiceLogPath(service),
			"metadata_path": m.imageServiceMetadataPath(service),
		},
	}
	if err := m.stateStore.AppendDeployment(record); err != nil {
		return cli.ServiceActionResult{}, fmt.Errorf("record image service removal: %w", err)
	}

	return cli.ServiceActionResult{
		OK:              true,
		Route:           route,
		Service:         service,
		ServiceKind:     serviceKindImage,
		Action:          route.Action,
		ArtifactVersion: artifactVersion,
		LogPath:         m.imageServiceLogPath(service),
		MetadataPath:    m.imageServiceMetadataPath(service),
		Command:         append([]string{"docker"}, commandArgs...),
	}, nil
}

func (m *Manager) imageServiceStatus(ctx context.Context, route *cli.Route, service string, definition ServiceDefinition) (cli.ServiceStatusResult, error) {
	metadata, ok, err := m.loadImageServiceMetadata(canonicalServiceName(service))
	if err != nil {
		return cli.ServiceStatusResult{}, err
	}

	image := imageLatestTag(definition)
	imagePresent := false
	commandArgs := []string{"image", "inspect", image, "--format", "{{.Id}}"}
	result, err := m.runner.Run(ctx, "", "docker", commandArgs...)
	if err != nil {
		return cli.ServiceStatusResult{}, err
	}
	if result.ExitCode == 0 && strings.TrimSpace(result.Stdout) != "" {
		imagePresent = true
	}

	activeContainers, err := m.inspectImageServiceContainers(ctx, definition.ActiveContainerPrefix)
	if err != nil {
		return cli.ServiceStatusResult{}, err
	}

	artifactVersion := image
	if ok && strings.TrimSpace(metadata.ImmutableTag) != "" {
		artifactVersion = metadata.ImmutableTag
	}

	status := "not-built"
	present := ok || imagePresent
	if imagePresent {
		status = "ready"
	} else if ok {
		status = "stale-metadata"
	}

	return cli.ServiceStatusResult{
		OK:              true,
		Route:           route,
		Service:         service,
		ServiceKind:     serviceKindImage,
		Present:         present,
		ComposeProject:  definition.ComposeProject,
		Image:           image,
		ArtifactVersion: artifactVersion,
		Status:          status,
		Running:         len(activeContainers) > 0,
		LogPath:         m.imageServiceLogPath(canonicalServiceName(service)),
		MetadataPath:    m.imageServiceMetadataPath(canonicalServiceName(service)),
		Containers:      activeContainers,
	}, nil
}

func (m *Manager) imageServiceLogs(_ context.Context, route *cli.Route, service string, _ ServiceDefinition) (cli.CommandResult, error) {
	logPath := m.imageServiceLogPath(service)
	data, err := os.ReadFile(logPath)
	if err != nil {
		if errorsIsNotExist(err) {
			return cli.CommandResult{
				OK:       true,
				Route:    route,
				ExitCode: 0,
			}, nil
		}
		return cli.CommandResult{}, fmt.Errorf("read image service log %s: %w", logPath, err)
	}
	return cli.CommandResult{
		OK:       true,
		Route:    route,
		Command:  []string{"cat", logPath},
		Stdout:   string(data),
		ExitCode: 0,
	}, nil
}

func (m *Manager) currentImageServiceArtifactVersion(service string, definition ServiceDefinition) string {
	metadata, ok, err := m.loadImageServiceMetadata(canonicalServiceName(service))
	if err == nil && ok && strings.TrimSpace(metadata.ImmutableTag) != "" {
		return metadata.ImmutableTag
	}
	if strings.TrimSpace(definition.ImageName) != "" {
		return imageLatestTag(definition)
	}
	return serviceArtifactVersion(service, "")
}

func (m *Manager) inspectImageServiceContainers(ctx context.Context, prefix string) ([]cli.ServiceContainerStatus, error) {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return nil, nil
	}

	commandArgs := []string{"ps", "-a", "--filter", "name=" + prefix, "--format", "{{.Names}}\t{{.Image}}\t{{.Status}}"}
	result, err := m.runner.Run(ctx, "", "docker", commandArgs...)
	if err != nil {
		return nil, err
	}
	if result.ExitCode != 0 {
		return nil, fmt.Errorf("docker ps for image service containers failed: %s", strings.TrimSpace(result.Stdout))
	}

	lines := strings.Split(strings.TrimSpace(result.Stdout), "\n")
	containers := make([]cli.ServiceContainerStatus, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) < 3 {
			continue
		}
		status := strings.TrimSpace(parts[2])
		containers = append(containers, cli.ServiceContainerStatus{
			Name:          strings.TrimSpace(parts[0]),
			Present:       true,
			ContainerName: strings.TrimSpace(parts[0]),
			Image:         strings.TrimSpace(parts[1]),
			Status:        status,
			Running:       strings.HasPrefix(strings.ToLower(status), "up"),
		})
	}
	return containers, nil
}

func (m *Manager) saveImageServiceMetadata(service string, metadata imageServiceMetadata) error {
	payload, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal image service metadata: %w", err)
	}
	if err := os.WriteFile(m.imageServiceMetadataPath(service), append(payload, '\n'), 0o644); err != nil {
		return fmt.Errorf("write image service metadata: %w", err)
	}
	return nil
}

func (m *Manager) loadImageServiceMetadata(service string) (imageServiceMetadata, bool, error) {
	data, err := os.ReadFile(m.imageServiceMetadataPath(service))
	if err != nil {
		if errorsIsNotExist(err) {
			return imageServiceMetadata{}, false, nil
		}
		return imageServiceMetadata{}, false, fmt.Errorf("read image service metadata: %w", err)
	}
	var metadata imageServiceMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return imageServiceMetadata{}, false, fmt.Errorf("decode image service metadata: %w", err)
	}
	return metadata, true, nil
}

func (m *Manager) imageServiceMetadataPath(service string) string {
	return filepath.Join(m.config.ServiceStateDir(service), "image-build.json")
}

func (m *Manager) imageServiceLogPath(service string) string {
	return filepath.Join(m.config.Paths.LogsRoot, service, "build.log")
}

func imageLatestTag(definition ServiceDefinition) string {
	return strings.TrimSpace(definition.ImageName) + ":latest"
}

func imageImmutableTag(imageName, digest string) string {
	suffix := strings.TrimSpace(digest)
	if len(suffix) > 12 {
		suffix = suffix[:12]
	}
	return fmt.Sprintf("%s:build-%s", strings.TrimSpace(imageName), suffix)
}

func (m *Manager) prepareImageServiceSourceContext(service, outputDir string) error {
	switch canonicalServiceName(service) {
	case "dev-sandbox":
		inputsPath := filepath.Join(outputDir, "runtime", "dev-sandbox", "build-inputs.json")
		if err := os.MkdirAll(filepath.Dir(inputsPath), 0o755); err != nil {
			return fmt.Errorf("create dev-sandbox build inputs dir: %w", err)
		}
		payload, err := json.MarshalIndent(map[string]string{
			"service":           canonicalServiceName(service),
			"gateway_revision":  gitRevision(m.config.GatewayRepoRoot()),
			"services_revision": gitRevision(m.config.ServicesRepoRoot()),
			"runtime_revision":  gitRevision(m.config.RuntimeRepoRoot()),
		}, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal dev-sandbox build inputs: %w", err)
		}
		if err := os.WriteFile(inputsPath, append(payload, '\n'), 0o644); err != nil {
			return fmt.Errorf("write dev-sandbox build inputs: %w", err)
		}
	}
	return nil
}

func (m *Manager) prepareImageServiceBinaryAssets(ctx context.Context, service, outputDir string) (string, error) {
	switch canonicalServiceName(service) {
	case "dev-sandbox":
		gatewayRepoRoot := strings.TrimSpace(m.config.GatewayRepoRoot())
		if gatewayRepoRoot == "" {
			return "", fmt.Errorf("dev-sandbox image build requires repos.gateway.url in gateway config")
		}
		binDir := filepath.Join(outputDir, "bin")
		if err := os.MkdirAll(binDir, 0o755); err != nil {
			return "", fmt.Errorf("create dev-sandbox bin dir: %w", err)
		}
		commandArgs := []string{
			"run",
			"--rm",
			"-v", fmt.Sprintf("%s:/src", gatewayRepoRoot),
			"-v", fmt.Sprintf("%s:/out", binDir),
			"-w", "/src",
			"golang:1.23-bookworm",
			"sh",
			"-lc",
			"set -eu; printf 'building moltbox cli\\n'; env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 /usr/local/go/bin/go build -buildvcs=false -o /out/moltbox ./cmd/moltbox",
		}
		result, err := m.runner.Run(ctx, "", "docker", commandArgs...)
		if err != nil {
			return result.Stdout, err
		}
		if result.ExitCode != 0 {
			return result.Stdout, fmt.Errorf("build dev-sandbox moltbox cli failed: %s", strings.TrimSpace(result.Stdout))
		}
		return result.Stdout, nil
	default:
		return "", nil
	}
}

func appendImageServiceLog(builder *strings.Builder, chunk string) {
	if builder == nil {
		return
	}
	trimmed := strings.TrimSpace(chunk)
	if trimmed == "" {
		return
	}
	if builder.Len() > 0 {
		builder.WriteString("\n")
	}
	builder.WriteString(trimmed)
	builder.WriteString("\n")
}
