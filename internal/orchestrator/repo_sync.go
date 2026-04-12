package orchestrator

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/remram-ai/moltbox-gateway/internal/config"
	"github.com/remram-ai/moltbox-gateway/internal/deploystate"
	"github.com/remram-ai/moltbox-gateway/pkg/cli"
)

func (m *Manager) GatewayRepoSync(ctx context.Context, route *cli.Route) (cli.RepoSyncResult, error) {
	targets := normalizedRepoSyncTargets(route.NativeArgs)
	if len(targets) == 0 {
		return cli.RepoSyncResult{}, fmt.Errorf("gateway repo-sync requires at least one target")
	}

	repoRoots, err := m.repoSyncRoots(targets)
	if err != nil {
		return cli.RepoSyncResult{}, err
	}

	script := buildRepoSyncScript(repoRoots, m.config.Paths.SecretsRoot, defaultApplianceHistoryPath)
	commandArgs := repoSyncHelperCommand(m.config, repoRoots, script, defaultApplianceHistoryPath, currentOperator())
	result, err := m.runner.Run(ctx, "", "docker", commandArgs...)
	if err != nil {
		return cli.RepoSyncResult{}, err
	}
	if result.ExitCode != 0 {
		return cli.RepoSyncResult{}, fmt.Errorf("repo-sync helper failed: %s", strings.TrimSpace(result.Stdout))
	}

	syncedTargets := parseRepoSyncOutput(result.Stdout)
	if len(syncedTargets) == 0 {
		return cli.RepoSyncResult{}, fmt.Errorf("repo-sync helper returned no target results")
	}

	for _, target := range syncedTargets {
		if err := m.stateStore.AppendDeployment(deploystate.DeploymentRecord{
			DeploymentID:    newGatewayID("deploy"),
			Timestamp:       time.Now().UTC().Format(time.RFC3339),
			Actor:           deploymentActor(route),
			Target:          target.Target,
			ArtifactVersion: target.NewVersion,
			PreviousVersion: target.PreviousVersion,
			Result:          "success",
			Operation:       "gateway_repo_sync",
			Details: map[string]string{
				"source":    target.Source,
				"repo_root": target.RepoRoot,
				"changed":   fmt.Sprintf("%t", target.Changed),
				"history":   defaultApplianceHistoryPath,
				"component": "repo-sync",
			},
		}); err != nil {
			return cli.RepoSyncResult{}, fmt.Errorf("record repo-sync deployment: %w", err)
		}
	}

	return cli.RepoSyncResult{
		OK:      true,
		Route:   route,
		Summary: fmt.Sprintf("repo-sync completed for %s", strings.Join(targets, ",")),
		Command: append([]string{"docker"}, commandArgs...),
		Targets: syncedTargets,
	}, nil
}

func (m *Manager) repoSyncRoots(targets []string) (map[string]string, error) {
	roots := make(map[string]string, len(targets))
	for _, target := range targets {
		switch target {
		case "services":
			root := strings.TrimSpace(m.config.ServicesRepoRoot())
			if root == "" {
				return nil, fmt.Errorf("gateway repo-sync requires repos.services.url in gateway config")
			}
			roots[target] = root
		case "runtime":
			root := strings.TrimSpace(m.config.RuntimeRepoRoot())
			if root == "" {
				return nil, fmt.Errorf("gateway repo-sync requires repos.runtime.url in gateway config")
			}
			roots[target] = root
		default:
			return nil, fmt.Errorf("unsupported repo-sync target %q", target)
		}
	}
	return roots, nil
}

func repoSyncHelperCommand(cfg config.Config, repoRoots map[string]string, syncScript, historyPath, operator string) []string {
	commandArgs := []string{
		"run",
		"--rm",
		"--name",
		fmt.Sprintf("gateway-repo-sync-%d", time.Now().Unix()),
		"--entrypoint",
		"sh",
	}

	mounts := []string{
		cfg.Paths.SecretsRoot,
		filepath.Dir(historyPath),
	}
	for _, target := range []string{"services", "runtime"} {
		if root := strings.TrimSpace(repoRoots[target]); root != "" {
			mounts = append(mounts, root)
		}
	}
	for _, mount := range uniqueMountRoots(mounts...) {
		commandArgs = append(commandArgs, "-v", fmt.Sprintf("%s:%s", mount, mount))
	}

	commandArgs = append(commandArgs,
		"-e", fmt.Sprintf("MOLTBOX_OPERATOR=%s", operator),
		"moltbox-gateway:latest",
		"-lc",
		syncScript,
	)
	return commandArgs
}

func buildRepoSyncScript(repoRoots map[string]string, secretsRoot, historyPath string) string {
	gitSSHKeyPath := filepath.Join(secretsRoot, "git", "id_ed25519")
	gitKnownHostsPath := filepath.Join(secretsRoot, "git", "known_hosts")
	lines := []string{
		"set -eu",
		fmt.Sprintf("SECRETS_ROOT=%s", shellQuote(secretsRoot)),
		fmt.Sprintf("HISTORY_PATH=%s", shellQuote(historyPath)),
		fmt.Sprintf("GIT_SSH_KEY=%s", shellQuote(gitSSHKeyPath)),
		fmt.Sprintf("GIT_SSH_KNOWN_HOSTS=%s", shellQuote(gitKnownHostsPath)),
		`mkdir -p "$SECRETS_ROOT" "$(dirname "$HISTORY_PATH")"`,
		`command -v git >/dev/null 2>&1 || { echo "repo-sync requires git in the helper container"; exit 1; }`,
		`if [ -f "$GIT_SSH_KEY" ]; then chmod 0600 "$GIT_SSH_KEY"; touch "$GIT_SSH_KNOWN_HOSTS"; chmod 0644 "$GIT_SSH_KNOWN_HOSTS"; export GIT_SSH_COMMAND="ssh -i $GIT_SSH_KEY -o IdentitiesOnly=yes -o UserKnownHostsFile=$GIT_SSH_KNOWN_HOSTS -o StrictHostKeyChecking=yes"; fi`,
		`JSON_ESCAPE() { printf '%s' "$1" | sed 's/\\/\\\\/g; s/"/\\"/g'; }`,
		`sync_repo() { TARGET="$1"; REPO="$2"; if [ ! -d "$REPO/.git" ] && [ ! -f "$REPO/.git" ]; then echo "repo-sync requires a git checkout at $REPO" >&2; exit 1; fi; git config --global --add safe.directory "$REPO"; OLD_VERSION="$(git -C "$REPO" rev-parse HEAD)"; SOURCE="$(git -C "$REPO" remote get-url origin 2>/dev/null || true)"; if [ -n "$SOURCE" ]; then case "$SOURCE" in https://github.com/*) git -C "$REPO" remote set-url origin "git@github.com:${SOURCE#https://github.com/}" ;; esac; git -C "$REPO" fetch --all --tags --prune; git -C "$REPO" pull --ff-only; fi; NEW_VERSION="$(git -C "$REPO" rev-parse HEAD)"; CHANGED=false; if [ "$OLD_VERSION" != "$NEW_VERSION" ]; then CHANGED=true; fi; TIMESTAMP="$(date -u +%Y-%m-%dT%H:%M:%SZ)"; OPERATOR="${MOLTBOX_OPERATOR:-${SUDO_USER:-${USER:-gateway}}}"; printf '{"timestamp":"%s","component":"repo-sync","target":"%s","old_version":"%s","new_version":"%s","source":"%s","operator":"%s"}\n' "$TIMESTAMP" "$(JSON_ESCAPE "$TARGET")" "$(JSON_ESCAPE "$OLD_VERSION")" "$(JSON_ESCAPE "$NEW_VERSION")" "$(JSON_ESCAPE "$SOURCE")" "$(JSON_ESCAPE "$OPERATOR")" >> "$HISTORY_PATH"; printf '%s\t%s\t%s\t%s\t%s\t%s\n' "$TARGET" "$REPO" "$SOURCE" "$OLD_VERSION" "$NEW_VERSION" "$CHANGED"; }`,
	}
	for _, target := range []string{"services", "runtime"} {
		if root := strings.TrimSpace(repoRoots[target]); root != "" {
			lines = append(lines, fmt.Sprintf("sync_repo %s %s", shellQuote(target), shellQuote(root)))
		}
	}
	return strings.Join(lines, "; ")
}

func parseRepoSyncOutput(stdout string) []cli.RepoSyncTargetResult {
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	results := make([]cli.RepoSyncTargetResult, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 6)
		if len(parts) < 6 {
			continue
		}
		results = append(results, cli.RepoSyncTargetResult{
			Target:          strings.TrimSpace(parts[0]),
			RepoRoot:        strings.TrimSpace(parts[1]),
			Source:          strings.TrimSpace(parts[2]),
			PreviousVersion: strings.TrimSpace(parts[3]),
			NewVersion:      strings.TrimSpace(parts[4]),
			Changed:         strings.EqualFold(strings.TrimSpace(parts[5]), "true"),
		})
	}
	return results
}

func normalizedRepoSyncTargets(targets []string) []string {
	seen := map[string]struct{}{}
	ordered := make([]string, 0, len(targets))
	for _, target := range targets {
		trimmed := strings.TrimSpace(target)
		if trimmed == "" {
			continue
		}
		if trimmed == "all" {
			trimmed = ""
		}
		if trimmed == "" {
			for _, expanded := range []string{"services", "runtime"} {
				if _, ok := seen[expanded]; ok {
					continue
				}
				seen[expanded] = struct{}{}
				ordered = append(ordered, expanded)
			}
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		ordered = append(ordered, trimmed)
	}
	return ordered
}
