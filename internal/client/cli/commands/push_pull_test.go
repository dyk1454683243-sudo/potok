package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/mtiluk/potok/internal/client/crypto"
	"github.com/mtiluk/potok/internal/client/transport"
	"github.com/mtiluk/potok/internal/client/vault"
	"github.com/mtiluk/potok/internal/protocol"
	"github.com/mtiluk/potok/internal/server/blobstore"
	httpapi "github.com/mtiluk/potok/internal/server/http"
	"github.com/mtiluk/potok/internal/server/store"
)

func newTestServer(t *testing.T) *transport.Client {
	t.Helper()

	s, err := store.Open(context.Background(), ":memory:")
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	if err := s.Migrate(); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	user, err := s.CreateUser(context.Background(), "a@example.com", "hunter2")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	handler := httpapi.NewHandler(s, *blobstore.New(t.TempDir()))
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handler.Health)
	mux.HandleFunc("POST /register", handler.Register)
	mux.Handle("POST /vaults", handler.APIKeyAuth(http.HandlerFunc(handler.CreateVault)))
	mux.Handle("GET /vaults", handler.APIKeyAuth(http.HandlerFunc(handler.ListVaults)))
	mux.Handle("GET /vaults/{name}", handler.APIKeyAuth(http.HandlerFunc(handler.VaultByName)))
	mux.Handle("DELETE /vaults/{name}", handler.APIKeyAuth(http.HandlerFunc(handler.DeleteVault)))
	mux.Handle("PUT /vaults/{name}/blobs/{id}", handler.APIKeyAuth(http.HandlerFunc(handler.PutBlob)))
	mux.Handle("GET /vaults/{name}/blobs/{id}", handler.APIKeyAuth(http.HandlerFunc(handler.GetBlob)))
	mux.Handle("PUT /vaults/{name}/manifest", handler.APIKeyAuth(http.HandlerFunc(handler.PutManifest)))
	mux.Handle("GET /vaults/{name}/manifest", handler.APIKeyAuth(http.HandlerFunc(handler.GetManifest)))
	mux.Handle("GET /me", handler.APIKeyAuth(http.HandlerFunc(handler.Me)))

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	return transport.New(srv.URL, user.APIKey)
}

func writeVaultFile(t *testing.T, root, rel string, content []byte) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
}

func pushVault(t *testing.T, client *transport.Client, vaultName, vaultPath, passphrase string) {
	t.Helper()

	remote, err := client.GetOrCreateVault(vaultName)
	if err != nil {
		t.Fatalf("GetOrCreateVault: %v", err)
	}

	key := crypto.DeriveKey([]byte(passphrase), remote.KDFSalt)

	files, err := vault.ScanVault(vaultPath)
	if err != nil {
		t.Fatalf("ScanVault: %v", err)
	}

	manifest := protocol.Manifest{Files: make([]protocol.ManifestFile, 0, len(files))}
	for _, f := range files {
		plaintext, err := os.ReadFile(filepath.Join(vaultPath, f.Path))
		if err != nil {
			t.Fatalf("ReadFile(%s): %v", f.Path, err)
		}
		ciphertext, err := crypto.Encrypt(key, plaintext)
		if err != nil {
			t.Fatalf("Encrypt(%s): %v", f.Path, err)
		}
		blobID, err := client.UploadBlob(vaultName, ciphertext)
		if err != nil {
			t.Fatalf("UploadBlob(%s): %v", f.Path, err)
		}
		manifest.Files = append(manifest.Files, protocol.ManifestFile{
			Path: f.Path, BlobID: blobID, Size: f.Size, ModTime: f.ModTime,
		})
	}

	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	manifestCiphertext, err := crypto.Encrypt(key, manifestJSON)
	if err != nil {
		t.Fatalf("Encrypt(manifest): %v", err)
	}

	generation, err := client.CurrentManifestGeneration(vaultName)
	if err != nil {
		t.Fatalf("CurrentManifestGeneration: %v", err)
	}
	if err := client.PutManifest(vaultName, generation, manifestCiphertext); err != nil {
		t.Fatalf("PutManifest: %v", err)
	}
}

func pullVault(client *transport.Client, vaultName, destPath, passphrase string) error {
	remote, found, err := client.FetchVault(vaultName)
	if err != nil {
		return err
	}
	if !found {
		return errNotPushed
	}

	key := crypto.DeriveKey([]byte(passphrase), remote.KDFSalt)

	manifestCiphertext, found, err := client.FetchManifest(vaultName)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}

	manifestJSON, err := crypto.Decrypt(key, manifestCiphertext)
	if err != nil {
		return err
	}

	var manifest protocol.Manifest
	if err := json.Unmarshal(manifestJSON, &manifest); err != nil {
		return err
	}

	for _, f := range manifest.Files {
		ciphertext, err := client.DownloadBlob(vaultName, f.BlobID)
		if err != nil {
			return err
		}
		plaintext, err := crypto.Decrypt(key, ciphertext)
		if err != nil {
			return err
		}
		dest := filepath.Join(destPath, f.Path)
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(dest, plaintext, 0o644); err != nil {
			return err
		}
	}
	return nil
}

var errNotPushed = errors.New("vault has not been pushed")

func TestPushThenPullRoundTrip(t *testing.T) {
	client := newTestServer(t)

	source := t.TempDir()
	writeVaultFile(t, source, "note.md", []byte("hello, potok"))
	writeVaultFile(t, source, "daily/2026-09-16.md", []byte("today's entry"))
	writeVaultFile(t, source, "attachments/photo.png", []byte("binary-ish content"))

	pushVault(t, client, "notes", source, "hunter2")

	dest := t.TempDir()
	if err := pullVault(client, "notes", dest, "hunter2"); err != nil {
		t.Fatalf("pullVault: %v", err)
	}

	for _, rel := range []string{"note.md", filepath.Join("daily", "2026-09-16.md"), filepath.Join("attachments", "photo.png")} {
		want, err := os.ReadFile(filepath.Join(source, rel))
		if err != nil {
			t.Fatalf("read source %s: %v", rel, err)
		}
		got, err := os.ReadFile(filepath.Join(dest, rel))
		if err != nil {
			t.Fatalf("read pulled %s: %v", rel, err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%s: pulled content = %q, want %q", rel, got, want)
		}
	}
}

func TestPushThenPullWrongPassphraseFails(t *testing.T) {
	client := newTestServer(t)

	source := t.TempDir()
	writeVaultFile(t, source, "note.md", []byte("hello, potok"))

	pushVault(t, client, "notes", source, "hunter2")

	dest := t.TempDir()
	err := pullVault(client, "notes", dest, "wrong-passphrase")
	if err == nil {
		t.Fatal("expected pull with the wrong passphrase to fail, got nil error")
	}
}

func TestPushThenPullCaseInsensitiveVaultName(t *testing.T) {
	client := newTestServer(t)

	source := t.TempDir()
	writeVaultFile(t, source, "note.md", []byte("hello, potok"))

	pushVault(t, client, "Notes", source, "hunter2")

	created, found, err := client.FetchVault("NOTES")
	if err != nil {
		t.Fatalf("FetchVault(NOTES): %v", err)
	}
	if !found {
		t.Fatal("FetchVault(NOTES) not found")
	}
	if created.Name != "Notes" {
		t.Errorf("FetchVault(NOTES).Name = %q, want stored casing %q", created.Name, "Notes")
	}

	again, err := client.GetOrCreateVault("notes")
	if err != nil {
		t.Fatalf("GetOrCreateVault(notes) after Notes exists: %v", err)
	}
	if again.ID != created.ID {
		t.Errorf("GetOrCreateVault(notes) created a new vault; ID = %q, want %q", again.ID, created.ID)
	}
	if again.Name != "Notes" {
		t.Errorf("GetOrCreateVault(notes).Name = %q, want original %q", again.Name, "Notes")
	}

	dest := t.TempDir()
	if err := pullVault(client, "NOTES", dest, "hunter2"); err != nil {
		t.Fatalf("pullVault(NOTES): %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dest, "note.md"))
	if err != nil {
		t.Fatalf("read pulled note.md: %v", err)
	}
	if string(content) != "hello, potok" {
		t.Errorf("pulled content = %q, want %q", content, "hello, potok")
	}
}

func TestPushCreatesRemoteVaultOnFirstPush(t *testing.T) {
	client := newTestServer(t)

	if _, found, err := client.FetchVault("notes"); err != nil {
		t.Fatalf("FetchVault (before push): %v", err)
	} else if found {
		t.Fatal("vault already exists before any push")
	}

	source := t.TempDir()
	writeVaultFile(t, source, "note.md", []byte("x"))
	pushVault(t, client, "notes", source, "hunter2")

	if _, found, err := client.FetchVault("notes"); err != nil {
		t.Fatalf("FetchVault (after push): %v", err)
	} else if !found {
		t.Fatal("vault was not created by push")
	}
}

func TestPushTwiceAdvancesGeneration(t *testing.T) {
	client := newTestServer(t)

	source := t.TempDir()
	writeVaultFile(t, source, "note.md", []byte("v1"))
	pushVault(t, client, "notes", source, "hunter2")

	gen1, err := client.CurrentManifestGeneration("notes")
	if err != nil {
		t.Fatalf("CurrentManifestGeneration: %v", err)
	}

	writeVaultFile(t, source, "note.md", []byte("v2"))
	pushVault(t, client, "notes", source, "hunter2")

	gen2, err := client.CurrentManifestGeneration("notes")
	if err != nil {
		t.Fatalf("CurrentManifestGeneration: %v", err)
	}

	if gen2 <= gen1 {
		t.Errorf("generation did not advance: gen1=%d gen2=%d", gen1, gen2)
	}

	dest := t.TempDir()
	if err := pullVault(client, "notes", dest, "hunter2"); err != nil {
		t.Fatalf("pullVault: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dest, "note.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "v2" {
		t.Errorf("pulled content = %q, want %q (the second push's content)", got, "v2")
	}
}
