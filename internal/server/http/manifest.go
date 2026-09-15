package http

import (
	"encoding/base64"
	"encoding/json"
	"net/http"

	"github.com/mtiluk/potok/internal/server/store"
)

func (h *Handler) PutManifest(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	vaultName := r.PathValue("name")
	if vaultName == "" {
		http.Error(w, "vault name is required", http.StatusBadRequest)
		return
	}

	vault, err := h.store.VaultByName(r.Context(), user.ID, vaultName)
	if err != nil {
		if err == store.ErrVaultNotFound {
			http.Error(w, "vault not found", http.StatusNotFound)
			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var body struct {
		ExpectedGeneration int64  `json:"expected_generation"`
		Ciphertext         string `json:"ciphertext"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "malformed request", http.StatusBadRequest)
		return
	}

	if body.ExpectedGeneration < 0 {
		http.Error(w, "expected_generation must not be negative", http.StatusBadRequest)
		return
	}

	if body.Ciphertext == "" {
		http.Error(w, "ciphertext is required", http.StatusBadRequest)
		return
	}

	ciphertext, err := base64.StdEncoding.DecodeString(body.Ciphertext)
	if err != nil {
		http.Error(w, "ciphertext must be base64-encoded", http.StatusBadRequest)
		return
	}

	manifest, err := h.store.PutManifest(r.Context(), vault.ID, body.ExpectedGeneration, ciphertext)
	if err != nil {
		if err == store.ErrManifestConflict {
			http.Error(w, "manifest generation conflict", http.StatusConflict)
			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(struct {
		Generation int64 `json:"generation"`
	}{manifest.Generation})
}

func (h *Handler) GetManifest(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	vaultName := r.PathValue("name")
	if vaultName == "" {
		http.Error(w, "vault name is required", http.StatusBadRequest)
		return
	}

	vault, err := h.store.VaultByName(r.Context(), user.ID, vaultName)
	if err != nil {
		if err == store.ErrVaultNotFound {
			http.Error(w, "vault not found", http.StatusNotFound)
			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	manifest, err := h.store.GetManifest(r.Context(), vault.ID)
	if err != nil {
		if err == store.ErrManifestNotFound {
			http.Error(w, "manifest not found", http.StatusNotFound)
			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		Generation int64  `json:"generation"`
		Ciphertext string `json:"ciphertext"`
	}{manifest.Generation, base64.StdEncoding.EncodeToString(manifest.Ciphertext)})
}
