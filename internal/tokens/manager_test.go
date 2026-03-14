package tokens

import (
	"testing"

	"github.com/remram-ai/moltbox-gateway/pkg/cli"
)

func TestManagerLifecycleAndValidation(t *testing.T) {
	t.Parallel()

	manager := NewManager(t.TempDir())
	route := &cli.Route{Resource: "gateway", Kind: cli.KindGatewayToken, Subject: "test-agent"}

	listBefore, err := manager.List(route)
	if err != nil {
		t.Fatalf("List() before create error = %v", err)
	}
	if len(listBefore.Tokens) != 0 {
		t.Fatalf("tokens before create = %v, want empty", listBefore.Tokens)
	}

	created, err := manager.Create(route)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Token == "" {
		t.Fatal("Create() returned empty token")
	}

	valid, err := manager.ValidateBearerToken("Bearer " + created.Token)
	if err != nil {
		t.Fatalf("ValidateBearerToken() error = %v", err)
	}
	if !valid.Authorized || valid.Name != "test-agent" {
		t.Fatalf("ValidateBearerToken() = %#v, want authorized test-agent", valid)
	}

	listAfter, err := manager.List(route)
	if err != nil {
		t.Fatalf("List() after create error = %v", err)
	}
	if len(listAfter.Tokens) != 1 || listAfter.Tokens[0].Name != "test-agent" {
		t.Fatalf("tokens after create = %v, want singleton %q", listAfter.Tokens, "test-agent")
	}

	rotated, err := manager.Rotate(route)
	if err != nil {
		t.Fatalf("Rotate() error = %v", err)
	}
	if rotated.Token == "" || rotated.Token == created.Token {
		t.Fatalf("Rotate() token = %q, want non-empty value different from create token %q", rotated.Token, created.Token)
	}

	valid, err = manager.ValidateBearerToken("Bearer " + rotated.Token)
	if err != nil {
		t.Fatalf("ValidateBearerToken(rotated) error = %v", err)
	}
	if !valid.Authorized || valid.Name != "test-agent" {
		t.Fatalf("ValidateBearerToken(rotated) = %#v, want authorized test-agent", valid)
	}

	deleted, err := manager.Delete(route)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted.Deleted {
		t.Fatal("Delete() = false, want true")
	}

	valid, err = manager.ValidateBearerToken("Bearer " + rotated.Token)
	if err != nil {
		t.Fatalf("ValidateBearerToken(after delete) error = %v", err)
	}
	if valid.Authorized {
		t.Fatalf("ValidateBearerToken(after delete) = %#v, want unauthorized", valid)
	}
}

func TestValidateBearerTokenIgnoresNonMCPSecrets(t *testing.T) {
	t.Parallel()

	manager := NewManager(t.TempDir())
	if err := manager.store.Set(secretScope, "POSTGRES_PASSWORD", "not-a-token"); err != nil {
		t.Fatalf("store.Set(non-token) error = %v", err)
	}

	route := &cli.Route{Resource: "gateway", Kind: cli.KindGatewayToken, Subject: "alpha"}
	created, err := manager.Create(route)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	valid, err := manager.ValidateBearerToken("Bearer " + created.Token)
	if err != nil {
		t.Fatalf("ValidateBearerToken() error = %v", err)
	}
	if !valid.Authorized || valid.Name != "alpha" {
		t.Fatalf("ValidateBearerToken() = %#v, want authorized alpha", valid)
	}

	listed, err := manager.List(route)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(listed.Tokens) != 1 || listed.Tokens[0].Name != "alpha" {
		t.Fatalf("List() = %#v, want only MCP token alpha", listed.Tokens)
	}
}
