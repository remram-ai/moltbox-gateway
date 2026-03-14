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

func TestParseRuntimeSkillRollback(t *testing.T) {
	t.Parallel()

	result := Parse([]string{"dev", "skill", "rollback", "together"})
	if result.Route == nil {
		t.Fatal("Parse() route = nil")
	}
	if result.Route.Kind != KindRuntimeSkill || result.Route.Action != "rollback" || result.Route.Subject != "together" {
		t.Fatalf("Parse() route = %#v, want dev runtime skill rollback route", result.Route)
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
			name:        "dev skill rollback",
			args:        []string{"dev", "skill", "rollback", "together"},
			wantKind:    KindRuntimeSkill,
			wantAction:  "rollback",
			wantSubject: "together",
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
			name:        "test skill deploy",
			args:        []string{"test", "skill", "deploy", "together"},
			wantKind:    KindRuntimeSkill,
			wantAction:  "deploy",
			wantSubject: "together",
			wantEnv:     "test",
			wantRuntime: "openclaw-test",
		},
		{
			name:        "test skill rollback",
			args:        []string{"test", "skill", "rollback", "together"},
			wantKind:    KindRuntimeSkill,
			wantAction:  "rollback",
			wantSubject: "together",
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
			name:        "prod skill deploy",
			args:        []string{"prod", "skill", "deploy", "together"},
			wantKind:    KindRuntimeSkill,
			wantAction:  "deploy",
			wantSubject: "together",
			wantEnv:     "prod",
			wantRuntime: "openclaw-prod",
		},
		{
			name:        "prod skill rollback",
			args:        []string{"prod", "skill", "rollback", "together"},
			wantKind:    KindRuntimeSkill,
			wantAction:  "rollback",
			wantSubject: "together",
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
