package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
)

var Prefix = "potok_"

// APIKeyHashPrefix marks a SHA-256 digest stored in users.api_key_hash.
// Generated keys are 256-bit CSPRNG values, so a fast cryptographic hash is
// enough to protect them at rest (same approach GitHub uses for PATs).
// User passwords stay on bcrypt.
const APIKeyHashPrefix = "sha256:"

const (
	keyBytes     = 32
	sha256HexLen = sha256.Size * 2
)

func GenerateAPIKey() (key string, err error) {
	raw := make([]byte, keyBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("auth: generate key: %w", err)
	}
	key = Prefix + base64.RawURLEncoding.EncodeToString(raw)
	return key, nil
}

func HashAPIKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return APIKeyHashPrefix + hex.EncodeToString(sum[:])
}

func IsHashedAPIKey(stored string) bool {
	digest, ok := strings.CutPrefix(stored, APIKeyHashPrefix)
	if !ok || len(digest) != sha256HexLen {
		return false
	}
	_, err := hex.DecodeString(digest)
	return err == nil
}
