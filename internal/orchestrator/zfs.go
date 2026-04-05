package orchestrator

import (
	"context"
	"fmt"
	"path/filepath"
	stdruntime "runtime"
	"strings"
	"time"
)

type zfsSnapshotRecord struct {
	Dataset   string
	Suffix    string
	Snapshots []string
}

func (r *zfsSnapshotRecord) DetailMap() map[string]string {
	if r == nil {
		return nil
	}
	details := map[string]string{
		"zfs_dataset":         r.Dataset,
		"zfs_snapshot_suffix": r.Suffix,
	}
	if len(r.Snapshots) > 0 {
		details["zfs_snapshots"] = strings.Join(r.Snapshots, ",")
	}
	return details
}

func (m *Manager) snapshotServiceState(ctx context.Context, action, service string) (*zfsSnapshotRecord, error) {
	service = canonicalServiceName(service)
	action = strings.ToLower(strings.TrimSpace(action))
	if action == "" {
		action = "deploy"
	}
	label := fmt.Sprintf("service-%s-%s-%s", action, service, time.Now().UTC().Format("20060102t150405z"))
	return m.snapshotZFSTree(ctx, m.config.Paths.StateRoot, label)
}

func (m *Manager) snapshotZFSTree(ctx context.Context, hostPath, label string) (*zfsSnapshotRecord, error) {
	cleanPath := filepath.Clean(hostPath)
	if stdruntime.GOOS != "linux" || filepath.VolumeName(cleanPath) != "" {
		return nil, nil
	}
	if !strings.HasPrefix(cleanPath, "/srv/moltbox-state") && !strings.HasPrefix(cleanPath, "/var/lib/moltbox") {
		return nil, nil
	}
	dataset, ok, err := m.zfsDatasetForPath(ctx, cleanPath)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}

	suffix := sanitizeZFSSnapshotToken(label)
	spec := fmt.Sprintf("%s@%s", dataset, suffix)
	result, err := m.runner.Run(ctx, "", "zfs", "snapshot", "-r", spec)
	if isMissingExecutableError(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if result.ExitCode != 0 {
		return nil, fmt.Errorf("create zfs snapshot %s failed: %s", spec, strings.TrimSpace(result.Stdout))
	}

	snapshots, err := m.listZFSSnapshots(ctx, dataset, suffix)
	if err != nil {
		return nil, err
	}
	return &zfsSnapshotRecord{
		Dataset:   dataset,
		Suffix:    suffix,
		Snapshots: snapshots,
	}, nil
}

func (m *Manager) zfsDatasetForPath(ctx context.Context, hostPath string) (string, bool, error) {
	result, err := m.runner.Run(ctx, "", "findmnt", "-no", "FSTYPE,SOURCE", "-T", hostPath)
	if isMissingExecutableError(err) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	if result.ExitCode != 0 {
		return "", false, nil
	}

	fields := strings.Fields(strings.TrimSpace(result.Stdout))
	if len(fields) < 2 || strings.TrimSpace(fields[0]) != "zfs" {
		return "", false, nil
	}
	return strings.TrimSpace(fields[1]), true, nil
}

func (m *Manager) listZFSSnapshots(ctx context.Context, dataset, suffix string) ([]string, error) {
	result, err := m.runner.Run(ctx, "", "zfs", "list", "-H", "-t", "snapshot", "-o", "name", "-r", dataset)
	if err != nil {
		return nil, err
	}
	if result.ExitCode != 0 {
		return nil, fmt.Errorf("list zfs snapshots for %s failed: %s", dataset, strings.TrimSpace(result.Stdout))
	}

	wantSuffix := "@" + suffix
	snapshots := make([]string, 0, 4)
	for _, raw := range strings.Split(result.Stdout, "\n") {
		name := strings.TrimSpace(raw)
		if name == "" || !strings.HasSuffix(name, wantSuffix) {
			continue
		}
		snapshots = append(snapshots, name)
	}
	if len(snapshots) == 0 {
		snapshots = append(snapshots, fmt.Sprintf("%s@%s", dataset, suffix))
	}
	return snapshots, nil
}

func sanitizeZFSSnapshotToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "snapshot"
	}
	var builder strings.Builder
	lastDash := false
	for _, r := range value {
		isAllowed := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if isAllowed {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if lastDash {
			continue
		}
		builder.WriteByte('-')
		lastDash = true
	}
	token := strings.Trim(builder.String(), "-")
	if token == "" {
		return "snapshot"
	}
	return token
}

func isMissingExecutableError(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "executable file not found") || strings.Contains(text, "file not found")
}
