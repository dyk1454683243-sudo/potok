package auth

import (
	"strings"
	"testing"
)

func TestGenerateAPIKey(t *testing.T) {
	valid, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey() error: %v", err)
	}

	if !strings.HasPrefix(valid, Prefix) {
		t.Errorf("GenerateAPIKey() = %v, want %v", valid, Prefix+valid)
	}
	if IsHashedAPIKey(valid) {
		t.Errorf("GenerateAPIKey() = %q, looks like a stored hash", valid)
	}
}

func TestHashAPIKey(t *testing.T) {
	key, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey() error: %v", err)
	}

	got := HashAPIKey(key)
	if got == key {
		t.Fatal("HashAPIKey() returned the plaintext key")
	}
	if !strings.HasPrefix(got, APIKeyHashPrefix) {
		t.Errorf("HashAPIKey() = %q, want prefix %q", got, APIKeyHashPrefix)
	}
	if !IsHashedAPIKey(got) {
		t.Errorf("IsHashedAPIKey(%q) = false, want true", got)
	}
	if HashAPIKey(key) != got {
		t.Error("HashAPIKey() is not deterministic")
	}

	other, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey() error: %v", err)
	}
	if HashAPIKey(other) == got {
		t.Error("HashAPIKey() collided for two distinct keys")
	}
}

func TestIsHashedAPIKey(t *testing.T) {
	valid := HashAPIKey("potok_test")
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"hashed", valid, true},
		{"plaintext generated", Prefix + "abc", false},
		{"empty", "", false},
		{"prefix only", APIKeyHashPrefix, false},
		{"wrong length", APIKeyHashPrefix + "abcd", false},
		{"non hex", APIKeyHashPrefix + strings.Repeat("g", 64), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsHashedAPIKey(tt.in); got != tt.want {
				t.Errorf("IsHashedAPIKey(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
