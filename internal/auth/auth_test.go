package auth

import (
	"path/filepath"
	"testing"
	"time"
)

func TestSaveLoadTokens_UsesProvidedPath(t *testing.T) {
	tokensPath := filepath.Join(t.TempDir(), "custom", "tokens.json")
	original := TokenResponse{
		AccessToken:  "access",
		RefreshToken: "refresh",
		ExpiresAt:    time.Date(2026, 2, 26, 12, 0, 0, 0, time.UTC),
	}

	if err := SaveTokens(tokensPath, original); err == nil {
		t.Fatalf("expected SaveTokens to fail when parent dir is missing")
	}
}

func TestSaveLoadTokens_RoundTrip(t *testing.T) {
	tokensPath := filepath.Join(t.TempDir(), "tokens.json")
	original := TokenResponse{
		AccessToken:  "access",
		RefreshToken: "refresh",
		ExpiresAt:    time.Date(2026, 2, 26, 12, 0, 0, 0, time.UTC),
	}

	if err := SaveTokens(tokensPath, original); err != nil {
		t.Fatalf("SaveTokens: %v", err)
	}

	loaded, err := LoadTokens(tokensPath)
	if err != nil {
		t.Fatalf("LoadTokens: %v", err)
	}

	if loaded.AccessToken != original.AccessToken || loaded.RefreshToken != original.RefreshToken || !loaded.ExpiresAt.Equal(original.ExpiresAt) {
		t.Fatalf("loaded tokens mismatch: got %+v want %+v", loaded, original)
	}
}
