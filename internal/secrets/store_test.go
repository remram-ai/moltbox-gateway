package secrets

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestStoreSetGetListDeleteByScope(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewStore(root)

	if err := store.Set("dev", "TOGETHER_API_KEY", "secret-dev-value"); err != nil {
		t.Fatalf("Set(dev) error = %v", err)
	}
	if err := store.Set("service", "POSTGRES_PASSWORD", "secret-service-value"); err != nil {
		t.Fatalf("Set(service) error = %v", err)
	}

	got, err := store.Get("dev", "TOGETHER_API_KEY")
	if err != nil {
		t.Fatalf("Get(dev) error = %v", err)
	}
	if got != "secret-dev-value" {
		t.Fatalf("Get(dev) = %q, want secret-dev-value", got)
	}

	devSecrets, err := store.List("dev")
	if err != nil {
		t.Fatalf("List(dev) error = %v", err)
	}
	if len(devSecrets) != 1 || devSecrets[0] != "TOGETHER_API_KEY" {
		t.Fatalf("List(dev) = %#v, want TOGETHER_API_KEY", devSecrets)
	}

	serviceSecrets, err := store.List("service")
	if err != nil {
		t.Fatalf("List(service) error = %v", err)
	}
	if len(serviceSecrets) != 1 || serviceSecrets[0] != "POSTGRES_PASSWORD" {
		t.Fatalf("List(service) = %#v, want POSTGRES_PASSWORD", serviceSecrets)
	}

	deleted, err := store.Delete("dev", "TOGETHER_API_KEY")
	if err != nil {
		t.Fatalf("Delete(dev) error = %v", err)
	}
	if !deleted {
		t.Fatal("expected secret to be deleted")
	}

	if _, err := store.Get("dev", "TOGETHER_API_KEY"); err != ErrSecretNotFound {
		t.Fatalf("Get after delete error = %v, want ErrSecretNotFound", err)
	}
}

func TestStoreDoesNotPersistPlaintext(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewStore(root)
	if err := store.Set("prod", "TOGETHER_API_KEY", "plain-text-should-not-appear"); err != nil {
		t.Fatalf("Set(prod) error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, "prod", "TOGETHER_API_KEY.json"))
	if err != nil {
		t.Fatalf("read encrypted record: %v", err)
	}
	if strings.Contains(string(data), "plain-text-should-not-appear") {
		t.Fatalf("encrypted record leaked plaintext: %s", data)
	}
}

func TestStoreResolveSkipsMissingSecrets(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewStore(root)
	if err := store.Set("test", "TOGETHER_API_KEY", "test-value"); err != nil {
		t.Fatalf("Set(test) error = %v", err)
	}

	resolved, err := store.Resolve("test", []string{"TOGETHER_API_KEY", "MISSING"})
	if err != nil {
		t.Fatalf("Resolve(test) error = %v", err)
	}
	if resolved["TOGETHER_API_KEY"] != "test-value" {
		t.Fatalf("Resolve(test) value = %#v, want test-value", resolved)
	}
	if _, ok := resolved["MISSING"]; ok {
		t.Fatalf("Resolve(test) unexpectedly returned missing secret: %#v", resolved)
	}
}

func TestStoreProtectsKeyAndSecretFiles(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewStore(root)
	if err := store.Set("dev", "TOGETHER_API_KEY", "secret-dev-value"); err != nil {
		t.Fatalf("Set(dev) error = %v", err)
	}

	keyData, err := os.ReadFile(filepath.Join(root, "master.key"))
	if err != nil {
		t.Fatalf("read master key: %v", err)
	}
	if len(keyData) != 32 {
		t.Fatalf("master key length = %d, want 32", len(keyData))
	}

	if runtime.GOOS != "windows" {
		keyInfo, err := os.Stat(filepath.Join(root, "master.key"))
		if err != nil {
			t.Fatalf("stat master key: %v", err)
		}
		if got := keyInfo.Mode().Perm(); got != 0o600 {
			t.Fatalf("master key mode = %#o, want 0600", got)
		}

		secretInfo, err := os.Stat(filepath.Join(root, "dev", "TOGETHER_API_KEY.json"))
		if err != nil {
			t.Fatalf("stat secret file: %v", err)
		}
		if got := secretInfo.Mode().Perm(); got != 0o600 {
			t.Fatalf("secret file mode = %#o, want 0600", got)
		}
	}
}

func TestStoreRotatesNoncePerWrite(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewStore(root)
	if err := store.Set("dev", "TOGETHER_API_KEY", "same-value"); err != nil {
		t.Fatalf("first Set(dev) error = %v", err)
	}

	var first record
	firstData, err := os.ReadFile(filepath.Join(root, "dev", "TOGETHER_API_KEY.json"))
	if err != nil {
		t.Fatalf("read first encrypted record: %v", err)
	}
	if err := json.Unmarshal(firstData, &first); err != nil {
		t.Fatalf("decode first encrypted record: %v", err)
	}

	if err := store.Set("dev", "TOGETHER_API_KEY", "same-value"); err != nil {
		t.Fatalf("second Set(dev) error = %v", err)
	}

	var second record
	secondData, err := os.ReadFile(filepath.Join(root, "dev", "TOGETHER_API_KEY.json"))
	if err != nil {
		t.Fatalf("read second encrypted record: %v", err)
	}
	if err := json.Unmarshal(secondData, &second); err != nil {
		t.Fatalf("decode second encrypted record: %v", err)
	}

	if first.Nonce == second.Nonce {
		t.Fatalf("nonce reused across writes: %q", first.Nonce)
	}
	if first.Ciphertext == second.Ciphertext {
		t.Fatalf("ciphertext reused across writes: %q", first.Ciphertext)
	}
}
