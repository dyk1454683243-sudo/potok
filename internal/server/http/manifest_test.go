package http

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mtiluk/potok/internal/server/blobstore"
	"github.com/mtiluk/potok/internal/server/store"
)

func newManifestTestHandler(t *testing.T, s *store.Store) *Handler {
	t.Helper()
	return &Handler{
		store:     s,
		blobStore: *blobstore.New(t.TempDir()),
	}
}

func putManifestBody(t *testing.T, expectedGeneration int64, ciphertext []byte) *bytes.Reader {
	t.Helper()
	body, err := json.Marshal(struct {
		ExpectedGeneration int64  `json:"expected_generation"`
		Ciphertext         string `json:"ciphertext"`
	}{expectedGeneration, base64.StdEncoding.EncodeToString(ciphertext)})
	if err != nil {
		t.Fatalf("marshal put-manifest body: %v", err)
	}
	return bytes.NewReader(body)
}

func TestPutManifestEndpoint(t *testing.T) {
	s := newTestStore(t)
	handler := newManifestTestHandler(t, s)

	user, err := s.CreateUser(context.Background(), "a@example.com", "hunter2")
	if err != nil {
		t.Fatalf("seed CreateUser: %v", err)
	}
	if _, err := s.CreateVault(context.Background(), user.ID, "notes"); err != nil {
		t.Fatalf("seed CreateVault: %v", err)
	}
	otherUser, err := s.CreateUser(context.Background(), "b@example.com", "hunter2")
	if err != nil {
		t.Fatalf("seed CreateUser(other): %v", err)
	}

	putRaw := func(apiKey, vaultName string, body *bytes.Reader) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPut, "/vaults/"+vaultName+"/manifest", body)
		req.Header.Set("Authorization", "Bearer "+apiKey)
		if vaultName != "" {
			req.SetPathValue("name", vaultName)
		}
		rec := httptest.NewRecorder()
		handler.APIKeyAuth(http.HandlerFunc(handler.PutManifest)).ServeHTTP(rec, req)
		return rec
	}

	t.Run("creates the first manifest at generation 1", func(t *testing.T) {
		rec := putRaw(user.APIKey, "notes", putManifestBody(t, 0, []byte("ciphertext-v1")))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected status code %d, got %d (body: %s)", http.StatusOK, rec.Code, rec.Body.String())
		}

		var resp struct {
			Generation int64 `json:"generation"`
		}
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if resp.Generation != 1 {
			t.Errorf("generation = %d, want 1", resp.Generation)
		}
	})

	t.Run("rejects a second first-write once a manifest exists", func(t *testing.T) {
		rec := putRaw(user.APIKey, "notes", putManifestBody(t, 0, []byte("ciphertext-conflict")))
		if rec.Code != http.StatusConflict {
			t.Errorf("expected status code %d, got %d (body: %s)", http.StatusConflict, rec.Code, rec.Body.String())
		}
	})

	t.Run("advances the generation when expected_generation matches", func(t *testing.T) {
		rec := putRaw(user.APIKey, "notes", putManifestBody(t, 1, []byte("ciphertext-v2")))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected status code %d, got %d (body: %s)", http.StatusOK, rec.Code, rec.Body.String())
		}

		var resp struct {
			Generation int64 `json:"generation"`
		}
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if resp.Generation != 2 {
			t.Errorf("generation = %d, want 2", resp.Generation)
		}
	})

	t.Run("rejects a stale expected_generation", func(t *testing.T) {
		rec := putRaw(user.APIKey, "notes", putManifestBody(t, 1, []byte("ciphertext-stale")))
		if rec.Code != http.StatusConflict {
			t.Errorf("expected status code %d, got %d (body: %s)", http.StatusConflict, rec.Code, rec.Body.String())
		}
	})

	t.Run("rejects a negative expected_generation", func(t *testing.T) {
		rec := putRaw(user.APIKey, "notes", putManifestBody(t, -1, []byte("x")))
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status code %d, got %d (body: %s)", http.StatusBadRequest, rec.Code, rec.Body.String())
		}
	})

	t.Run("rejects an empty ciphertext", func(t *testing.T) {
		rec := putRaw(user.APIKey, "notes", putManifestBody(t, 2, nil))
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status code %d, got %d (body: %s)", http.StatusBadRequest, rec.Code, rec.Body.String())
		}
	})

	t.Run("rejects malformed JSON", func(t *testing.T) {
		rec := putRaw(user.APIKey, "notes", bytes.NewReader([]byte("not json")))
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status code %d, got %d (body: %s)", http.StatusBadRequest, rec.Code, rec.Body.String())
		}
	})

	t.Run("rejects non-base64 ciphertext", func(t *testing.T) {
		raw := bytes.NewReader([]byte(`{"expected_generation":2,"ciphertext":"not-valid-base64!!"}`))
		rec := putRaw(user.APIKey, "notes", raw)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status code %d, got %d (body: %s)", http.StatusBadRequest, rec.Code, rec.Body.String())
		}
	})

	t.Run("missing vault name", func(t *testing.T) {
		rec := putRaw(user.APIKey, "", putManifestBody(t, 0, []byte("x")))
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status code %d, got %d (body: %s)", http.StatusBadRequest, rec.Code, rec.Body.String())
		}
	})

	t.Run("vault not found", func(t *testing.T) {
		rec := putRaw(user.APIKey, "does-not-exist", putManifestBody(t, 0, []byte("x")))
		if rec.Code != http.StatusNotFound {
			t.Errorf("expected status code %d, got %d (body: %s)", http.StatusNotFound, rec.Code, rec.Body.String())
		}
	})

	t.Run("cannot write to another user's vault", func(t *testing.T) {
		rec := putRaw(otherUser.APIKey, "notes", putManifestBody(t, 0, []byte("x")))
		if rec.Code != http.StatusNotFound {
			t.Errorf("expected status code %d, got %d (body: %s)", http.StatusNotFound, rec.Code, rec.Body.String())
		}
	})

	t.Run("unauthorized without middleware", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/vaults/notes/manifest", putManifestBody(t, 0, []byte("x")))
		req.SetPathValue("name", "notes")
		rec := httptest.NewRecorder()
		handler.PutManifest(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status code %d, got %d", http.StatusUnauthorized, rec.Code)
		}
	})
}

func TestGetManifestEndpoint(t *testing.T) {
	s := newTestStore(t)
	handler := newManifestTestHandler(t, s)

	user, err := s.CreateUser(context.Background(), "a@example.com", "hunter2")
	if err != nil {
		t.Fatalf("seed CreateUser: %v", err)
	}
	if _, err := s.CreateVault(context.Background(), user.ID, "notes"); err != nil {
		t.Fatalf("seed CreateVault: %v", err)
	}
	otherUser, err := s.CreateUser(context.Background(), "b@example.com", "hunter2")
	if err != nil {
		t.Fatalf("seed CreateUser(other): %v", err)
	}

	content := []byte("hello, manifest")

	seedReq := httptest.NewRequest(http.MethodPut, "/vaults/notes/manifest", putManifestBody(t, 0, content))
	seedReq.Header.Set("Authorization", "Bearer "+user.APIKey)
	seedReq.SetPathValue("name", "notes")
	seedRec := httptest.NewRecorder()
	handler.APIKeyAuth(http.HandlerFunc(handler.PutManifest)).ServeHTTP(seedRec, seedReq)
	if seedRec.Code != http.StatusOK {
		t.Fatalf("seed PutManifest failed: %d (body: %s)", seedRec.Code, seedRec.Body.String())
	}

	get := func(apiKey, vaultName string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/vaults/"+vaultName+"/manifest", nil)
		req.Header.Set("Authorization", "Bearer "+apiKey)
		if vaultName != "" {
			req.SetPathValue("name", vaultName)
		}
		rec := httptest.NewRecorder()
		handler.APIKeyAuth(http.HandlerFunc(handler.GetManifest)).ServeHTTP(rec, req)
		return rec
	}

	t.Run("returns the stored generation and ciphertext", func(t *testing.T) {
		rec := get(user.APIKey, "notes")
		if rec.Code != http.StatusOK {
			t.Fatalf("expected status code %d, got %d (body: %s)", http.StatusOK, rec.Code, rec.Body.String())
		}

		var resp struct {
			Generation int64  `json:"generation"`
			Ciphertext string `json:"ciphertext"`
		}
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if resp.Generation != 1 {
			t.Errorf("generation = %d, want 1", resp.Generation)
		}

		got, err := base64.StdEncoding.DecodeString(resp.Ciphertext)
		if err != nil {
			t.Fatalf("decode ciphertext: %v", err)
		}
		if !bytes.Equal(got, content) {
			t.Errorf("ciphertext = %q, want %q", got, content)
		}
	})

	t.Run("manifest not found for a vault with no manifest pushed yet", func(t *testing.T) {
		if _, err := s.CreateVault(context.Background(), user.ID, "empty-vault"); err != nil {
			t.Fatalf("seed CreateVault(empty-vault): %v", err)
		}
		rec := get(user.APIKey, "empty-vault")
		if rec.Code != http.StatusNotFound {
			t.Errorf("expected status code %d, got %d (body: %s)", http.StatusNotFound, rec.Code, rec.Body.String())
		}
	})

	t.Run("vault not found", func(t *testing.T) {
		rec := get(user.APIKey, "does-not-exist")
		if rec.Code != http.StatusNotFound {
			t.Errorf("expected status code %d, got %d (body: %s)", http.StatusNotFound, rec.Code, rec.Body.String())
		}
	})

	t.Run("missing vault name", func(t *testing.T) {
		rec := get(user.APIKey, "")
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status code %d, got %d (body: %s)", http.StatusBadRequest, rec.Code, rec.Body.String())
		}
	})

	t.Run("cannot read another user's vault manifest", func(t *testing.T) {
		rec := get(otherUser.APIKey, "notes")
		if rec.Code != http.StatusNotFound {
			t.Errorf("expected status code %d, got %d (body: %s)", http.StatusNotFound, rec.Code, rec.Body.String())
		}
	})

	t.Run("unauthorized without middleware", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/vaults/notes/manifest", nil)
		req.SetPathValue("name", "notes")
		rec := httptest.NewRecorder()
		handler.GetManifest(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status code %d, got %d", http.StatusUnauthorized, rec.Code)
		}
	})
}
