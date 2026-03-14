package cli

import "testing"

func TestParseRuntimeOpenClawAgentNormalizesMessageFlag(t *testing.T) {
	t.Parallel()

	result := Parse([]string{
		"dev",
		"openclaw",
		"agent",
		"--agent",
		"main",
		"--local",
		"--thinking",
		"off",
		"--message",
		"Say",
		"hello",
		"in",
		"one",
		"sentence.",
		"--json",
	})
	if result.Route == nil {
		t.Fatal("Parse() route = nil")
	}

	want := []string{"agent", "--agent", "main", "--local", "--thinking", "off", "--message", "Say hello in one sentence.", "--json"}
	if !equalArgs(result.Route.NativeArgs, want) {
		t.Fatalf("Parse() native_args = %#v, want %#v", result.Route.NativeArgs, want)
	}
}

func TestParseScopedSecretsSetJoinsInlineValue(t *testing.T) {
	t.Parallel()

	result := Parse([]string{"dev", "secrets", "set", "TEST_SECRET", "value", "with", "spaces"})
	if result.Route == nil {
		t.Fatal("Parse() route = nil")
	}
	if len(result.Route.NativeArgs) != 1 || result.Route.NativeArgs[0] != "value with spaces" {
		t.Fatalf("Parse() native_args = %#v, want [\"value with spaces\"]", result.Route.NativeArgs)
	}
}

func TestParseGatewayMCPStdio(t *testing.T) {
	t.Parallel()

	result := Parse([]string{"gateway", "mcp-stdio"})
	if result.Route == nil {
		t.Fatal("Parse() route = nil")
	}
	if result.Route.Kind != KindGatewayMCP || result.Route.Action != "mcp-stdio" {
		t.Fatalf("Parse() route = %#v, want gateway mcp-stdio route", result.Route)
	}
}

func TestParseRuntimeSkillDeploy(t *testing.T) {
	t.Parallel()

	result := Parse([]string{"dev", "skill", "deploy", "together"})
	if result.Route == nil {
		t.Fatal("Parse() route = nil")
	}
	if result.Route.Kind != KindRuntimeSkill || result.Route.Action != "deploy" || result.Route.Subject != "together" {
		t.Fatalf("Parse() route = %#v, want dev runtime skill deploy route", result.Route)
	}
}

func TestParseRuntimeSkillRemove(t *testing.T) {
	t.Parallel()

	result := Parse([]string{"dev", "skill", "remove", "together"})
	if result.Route == nil {
		t.Fatal("Parse() route = nil")
	}
	if result.Route.Kind != KindRuntimeSkill || result.Route.Action != "remove" || result.Route.Subject != "together" {
		t.Fatalf("Parse() route = %#v, want dev runtime skill remove route", result.Route)
	}
}

func TestParseRuntimePluginInstall(t *testing.T) {
	t.Parallel()

	result := Parse([]string{"dev", "plugin", "install", "semantic-router"})
	if result.Route == nil {
		t.Fatal("Parse() route = nil")
	}
	if result.Route.Kind != KindRuntimePlugin || result.Route.Action != "install" || result.Route.Subject != "semantic-router" {
		t.Fatalf("Parse() route = %#v, want dev runtime plugin install route", result.Route)
	}
}

func TestParseRuntimePluginRemove(t *testing.T) {
	t.Parallel()

	result := Parse([]string{"dev", "plugin", "remove", "semantic-router"})
	if result.Route == nil {
		t.Fatal("Parse() route = nil")
	}
	if result.Route.Kind != KindRuntimePlugin || result.Route.Action != "remove" || result.Route.Subject != "semantic-router" {
		t.Fatalf("Parse() route = %#v, want dev runtime plugin remove route", result.Route)
	}
}

func TestParseRuntimeContractAcrossEnvironments(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		args        []string
		wantKind    string
		wantAction  string
		wantSubject string
		wantEnv     string
		wantRuntime string
	}{
		{
			name:        "dev skill deploy",
			args:        []string{"dev", "skill", "deploy", "together"},
			wantKind:    KindRuntimeSkill,
			wantAction:  "deploy",
			wantSubject: "together",
			wantEnv:     "dev",
			wantRuntime: "openclaw-dev",
		},
		{
			name:        "dev skill remove",
			args:        []string{"dev", "skill", "remove", "together"},
			wantKind:    KindRuntimeSkill,
			wantAction:  "remove",
			wantSubject: "together",
			wantEnv:     "dev",
			wantRuntime: "openclaw-dev",
		},
		{
			name:        "dev skill list",
			args:        []string{"dev", "skill", "list"},
			wantKind:    KindRuntimeSkill,
			wantAction:  "list",
			wantEnv:     "dev",
			wantRuntime: "openclaw-dev",
		},
		{
			name:        "dev checkpoint",
			args:        []string{"dev", "checkpoint"},
			wantKind:    KindRuntimeAction,
			wantAction:  "checkpoint",
			wantEnv:     "dev",
			wantRuntime: "openclaw-dev",
		},
		{
			name:        "dev plugin install",
			args:        []string{"dev", "plugin", "install", "semantic-router"},
			wantKind:    KindRuntimePlugin,
			wantAction:  "install",
			wantSubject: "semantic-router",
			wantEnv:     "dev",
			wantRuntime: "openclaw-dev",
		},
		{
			name:        "dev plugin remove",
			args:        []string{"dev", "plugin", "remove", "semantic-router"},
			wantKind:    KindRuntimePlugin,
			wantAction:  "remove",
			wantSubject: "semantic-router",
			wantEnv:     "dev",
			wantRuntime: "openclaw-dev",
		},
		{
			name:        "dev plugin list",
			args:        []string{"dev", "plugin", "list"},
			wantKind:    KindRuntimePlugin,
			wantAction:  "list",
			wantEnv:     "dev",
			wantRuntime: "openclaw-dev",
		},
		{
			name:        "test skill deploy",
			args:        []string{"test", "skill", "deploy", "together"},
			wantKind:    KindRuntimeSkill,
			wantAction:  "deploy",
			wantSubject: "together",
			wantEnv:     "test",
			wantRuntime: "openclaw-test",
		},
		{
			name:        "test skill remove",
			args:        []string{"test", "skill", "remove", "together"},
			wantKind:    KindRuntimeSkill,
			wantAction:  "remove",
			wantSubject: "together",
			wantEnv:     "test",
			wantRuntime: "openclaw-test",
		},
		{
			name:        "test skill list",
			args:        []string{"test", "skill", "list"},
			wantKind:    KindRuntimeSkill,
			wantAction:  "list",
			wantEnv:     "test",
			wantRuntime: "openclaw-test",
		},
		{
			name:        "test checkpoint",
			args:        []string{"test", "checkpoint"},
			wantKind:    KindRuntimeAction,
			wantAction:  "checkpoint",
			wantEnv:     "test",
			wantRuntime: "openclaw-test",
		},
		{
			name:        "test plugin install",
			args:        []string{"test", "plugin", "install", "semantic-router"},
			wantKind:    KindRuntimePlugin,
			wantAction:  "install",
			wantSubject: "semantic-router",
			wantEnv:     "test",
			wantRuntime: "openclaw-test",
		},
		{
			name:        "test plugin remove",
			args:        []string{"test", "plugin", "remove", "semantic-router"},
			wantKind:    KindRuntimePlugin,
			wantAction:  "remove",
			wantSubject: "semantic-router",
			wantEnv:     "test",
			wantRuntime: "openclaw-test",
		},
		{
			name:        "test plugin list",
			args:        []string{"test", "plugin", "list"},
			wantKind:    KindRuntimePlugin,
			wantAction:  "list",
			wantEnv:     "test",
			wantRuntime: "openclaw-test",
		},
		{
			name:        "prod skill deploy",
			args:        []string{"prod", "skill", "deploy", "together"},
			wantKind:    KindRuntimeSkill,
			wantAction:  "deploy",
			wantSubject: "together",
			wantEnv:     "prod",
			wantRuntime: "openclaw-prod",
		},
		{
			name:        "prod skill remove",
			args:        []string{"prod", "skill", "remove", "together"},
			wantKind:    KindRuntimeSkill,
			wantAction:  "remove",
			wantSubject: "together",
			wantEnv:     "prod",
			wantRuntime: "openclaw-prod",
		},
		{
			name:        "prod skill list",
			args:        []string{"prod", "skill", "list"},
			wantKind:    KindRuntimeSkill,
			wantAction:  "list",
			wantEnv:     "prod",
			wantRuntime: "openclaw-prod",
		},
		{
			name:        "prod checkpoint",
			args:        []string{"prod", "checkpoint"},
			wantKind:    KindRuntimeAction,
			wantAction:  "checkpoint",
			wantEnv:     "prod",
			wantRuntime: "openclaw-prod",
		},
		{
			name:        "prod plugin install",
			args:        []string{"prod", "plugin", "install", "semantic-router"},
			wantKind:    KindRuntimePlugin,
			wantAction:  "install",
			wantSubject: "semantic-router",
			wantEnv:     "prod",
			wantRuntime: "openclaw-prod",
		},
		{
			name:        "prod plugin remove",
			args:        []string{"prod", "plugin", "remove", "semantic-router"},
			wantKind:    KindRuntimePlugin,
			wantAction:  "remove",
			wantSubject: "semantic-router",
			wantEnv:     "prod",
			wantRuntime: "openclaw-prod",
		},
		{
			name:        "prod plugin list",
			args:        []string{"prod", "plugin", "list"},
			wantKind:    KindRuntimePlugin,
			wantAction:  "list",
			wantEnv:     "prod",
			wantRuntime: "openclaw-prod",
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			result := Parse(testCase.args)
			if result.Route == nil {
				t.Fatal("Parse() route = nil")
			}
			if result.Route.Kind != testCase.wantKind || result.Route.Action != testCase.wantAction {
				t.Fatalf("Parse() route = %#v, want kind=%s action=%s", result.Route, testCase.wantKind, testCase.wantAction)
			}
			if result.Route.Subject != testCase.wantSubject {
				t.Fatalf("Parse() subject = %q, want %q", result.Route.Subject, testCase.wantSubject)
			}
			if result.Route.Environment != testCase.wantEnv || result.Route.Runtime != testCase.wantRuntime {
				t.Fatalf("Parse() route = %#v, want env=%s runtime=%s", result.Route, testCase.wantEnv, testCase.wantRuntime)
			}
		})
	}
}

func TestParseRuntimeSkillRollbackAlias(t *testing.T) {
	t.Parallel()

	result := Parse([]string{"dev", "skill", "rollback", "together"})
	if result.Route == nil {
		t.Fatal("Parse() route = nil")
	}
	if result.Route.Kind != KindRuntimeSkill || result.Route.Action != "remove" || result.Route.Subject != "together" {
		t.Fatalf("Parse() route = %#v, want dev runtime skill remove alias route", result.Route)
	}
}

func equalArgs(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
