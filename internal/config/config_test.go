package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadParsesSystemOwnedCLIPaths(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	configPath := filepath.Join(root, "config.yaml")
	if err := os.WriteFile(configPath, []byte(
		"paths:\n"+
			"  state_root: /srv/moltbox-state\n"+
			"  runtime_root: /srv/moltbox-state/runtime\n"+
			"  logs_root: /srv/moltbox-logs\n"+
			"  secrets_root: /var/lib/moltbox/secrets\n"+
			"repos:\n"+
			"  gateway:\n"+
			"    url: /opt/moltbox/repos/moltbox-gateway\n"+
			"  services:\n"+
			"    url: /opt/moltbox/repos/moltbox-services\n"+
			"  runtime:\n"+
			"    url: /opt/moltbox/repos/moltbox-runtime\n"+
			"  skills:\n"+
			"    url: /opt/moltbox/repos/remram-skills\n"+
			"gateway:\n"+
			"  host: 0.0.0.0\n"+
			"  port: 7460\n"+
			"cli:\n"+
			"  path: /usr/local/bin/moltbox\n"+
			"  config_path: /etc/moltbox/config.yaml\n",
	), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got := cfg.GatewayRepoRoot(); got != "/opt/moltbox/repos/moltbox-gateway" {
		t.Fatalf("GatewayRepoRoot() = %q", got)
	}
	if got := cfg.CLI.Path; got != "/usr/local/bin/moltbox" {
		t.Fatalf("CLI.Path = %q", got)
	}
	if got := cfg.CLI.ConfigPath; got != "/etc/moltbox/config.yaml" {
		t.Fatalf("CLI.ConfigPath = %q", got)
	}
}

func TestConfigPathDefaultsToSystemPath(t *testing.T) {
	t.Setenv("MOLTBOX_CONFIG_PATH", "")

	if got := ConfigPath(); got != DefaultConfigPath {
		t.Fatalf("ConfigPath() = %q, want %q", got, DefaultConfigPath)
	}
}

func TestConfigPathPrefersExplicitEnvOverride(t *testing.T) {
	override := filepath.Join(t.TempDir(), "override.yaml")
	t.Setenv("MOLTBOX_CONFIG_PATH", override)

	if got := ConfigPath(); got != override {
		t.Fatalf("ConfigPath() = %q, want %q", got, override)
	}
}
