package sshwrap

import (
	"reflect"
	"testing"
)

func TestResolveTestOperatorPreservesQuotedArgs(t *testing.T) {
	t.Parallel()

	args, deny, err := Resolve(ModeTestOperator, `moltbox test openclaw agent --message "Say hello in one sentence." --json`)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if deny != "" {
		t.Fatalf("Resolve() deny = %q, want empty", deny)
	}

	want := []string{"test", "openclaw", "agent", "--message", "Say hello in one sentence.", "--json"}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("Resolve() args = %#v, want %#v", args, want)
	}
}

func TestResolveTestOperatorAllowsAbsoluteCLIPath(t *testing.T) {
	t.Parallel()

	args, deny, err := Resolve(ModeTestOperator, `/usr/local/bin/moltbox test openclaw health --json`)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if deny != "" {
		t.Fatalf("Resolve() deny = %q, want empty", deny)
	}

	want := []string{"test", "openclaw", "health", "--json"}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("Resolve() args = %#v, want %#v", args, want)
	}
}

func TestResolveRejectsShellOperators(t *testing.T) {
	t.Parallel()

	_, _, err := Resolve(ModeTestOperator, `moltbox test openclaw health --json; whoami`)
	if err == nil {
		t.Fatal("Resolve() error = nil, want unsupported shell operator")
	}
}

func TestResolveProdOperatorBlocksProdMutation(t *testing.T) {
	t.Parallel()

	_, deny, err := Resolve(ModeProdOperator, `moltbox prod openclaw plugins install browser`)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if deny == "" {
		t.Fatal("Resolve() deny = empty, want prod mutation denied")
	}
}

func TestResolveProdOperatorAllowsMutationHelpAndDryRun(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
	}{
		{
			name: "config set help",
			raw:  `moltbox prod openclaw config set --help`,
		},
		{
			name: "config set dry run",
			raw:  `moltbox prod openclaw config set logging.level \"info\" --dry-run`,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, deny, err := Resolve(ModeProdOperator, test.raw)
			if err != nil {
				t.Fatalf("Resolve() error = %v", err)
			}
			if deny != "" {
				t.Fatalf("Resolve() deny = %q, want empty", deny)
			}
		})
	}
}

func TestResolveBootstrapPolicy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		raw      string
		wantDeny string
	}{
		{
			name: "bootstrap gateway allowed",
			raw:  `moltbox bootstrap gateway`,
		},
		{
			name: "service status allowed",
			raw:  `moltbox service status test`,
		},
		{
			name: "test health allowed",
			raw:  `moltbox test openclaw health --json`,
		},
		{
			name:     "service deploy denied",
			raw:      `moltbox service deploy test`,
			wantDeny: "service access is limited to list, status, and logs",
		},
		{
			name:     "secret access denied",
			raw:      `moltbox secret list test`,
			wantDeny: "secret access is not permitted for bootstrap sessions",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, deny, err := Resolve(ModeBootstrap, test.raw)
			if err != nil {
				t.Fatalf("Resolve() error = %v", err)
			}
			if deny != test.wantDeny {
				t.Fatalf("Resolve() deny = %q, want %q", deny, test.wantDeny)
			}
		})
	}
}
