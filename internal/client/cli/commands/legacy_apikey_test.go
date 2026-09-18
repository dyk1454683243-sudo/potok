package commands

import (
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/mtiluk/potok/internal/client/config"
	"github.com/mtiluk/potok/internal/client/secrets"
	"github.com/zalando/go-keyring"
)

func TestMigrateLegacyAPIKeyMovesPlaintextIntoKeyring(t *testing.T) {
	t.Setenv("POTOK_CONFIG_DIR", t.TempDir())
	keyring.MockInit()

	path, err := config.Path()
	if err != nil {
		t.Fatalf("Path() = %v", err)
	}
	if err := os.WriteFile(path, []byte(`{
  "server_url": "https://potok.example.com",
  "api_key": "potok_legacy",
  "vaults": []
}
`), 0o600); err != nil {
		t.Fatalf("WriteFile() = %v", err)
	}

	if err := MigrateLegacyAPIKey(); err != nil {
		t.Fatalf("MigrateLegacyAPIKey() = %v", err)
	}

	got, err := secrets.Get(secrets.APIKey)
	if err != nil {
		t.Fatalf("secrets.Get() = %v", err)
	}
	if got != "potok_legacy" {
		t.Errorf("keyring API key = %q, want potok_legacy", got)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() = %v", err)
	}
	if containsAPIKey(t, data) {
		t.Errorf("config.json still has api_key:\n%s", data)
	}
}

func TestMigrateLegacyAPIKeyDoesNotOverwriteExistingKeyring(t *testing.T) {
	t.Setenv("POTOK_CONFIG_DIR", t.TempDir())
	keyring.MockInit()

	if err := secrets.Set(secrets.APIKey, "already-in-keyring"); err != nil {
		t.Fatalf("secrets.Set() = %v", err)
	}

	path, err := config.Path()
	if err != nil {
		t.Fatalf("Path() = %v", err)
	}
	if err := os.WriteFile(path, []byte(`{
  "server_url": "https://potok.example.com",
  "api_key": "should-not-win"
}
`), 0o600); err != nil {
		t.Fatalf("WriteFile() = %v", err)
	}

	if err := MigrateLegacyAPIKey(); err != nil {
		t.Fatalf("MigrateLegacyAPIKey() = %v", err)
	}

	got, err := secrets.Get(secrets.APIKey)
	if err != nil {
		t.Fatalf("secrets.Get() = %v", err)
	}
	if got != "already-in-keyring" {
		t.Errorf("keyring API key = %q, want the existing value", got)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() = %v", err)
	}
	if containsAPIKey(t, data) {
		t.Errorf("config.json still has api_key:\n%s", data)
	}
}

func TestMigrateLegacyAPIKeyNoopsWhenMissing(t *testing.T) {
	t.Setenv("POTOK_CONFIG_DIR", t.TempDir())
	keyring.MockInit()

	if err := MigrateLegacyAPIKey(); err != nil {
		t.Fatalf("MigrateLegacyAPIKey() on missing config = %v", err)
	}
	if _, err := secrets.Get(secrets.APIKey); !errors.Is(err, secrets.ErrNotFound) {
		t.Errorf("secrets.Get() = %v, want ErrNotFound", err)
	}
}

func containsAPIKey(t *testing.T, data []byte) bool {
	t.Helper()
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal rewritten config: %v", err)
	}
	_, ok := raw["api_key"]
	return ok
}
