package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSaveAndLoadCache(t *testing.T) {
	// Create test cache
	cache := &DirectURLCache{
		ServerURL: "https://example.com/mcp",
		AgentID:   "test-agent",
		Tier:      "pro",
		RateLimits: RateLimits{
			RPM: 100,
			RPH: 1000,
		},
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
		Signature: "test-signature",
	}

	serverKey := "test/server"

	// Save cache
	err := SaveCache(serverKey, cache)
	if err != nil {
		t.Fatalf("SaveCache failed: %v", err)
	}

	// Load cache
	loaded, err := LoadCache(serverKey)
	if err != nil {
		t.Fatalf("LoadCache failed: %v", err)
	}

	if loaded == nil {
		t.Fatal("expected cache to be loaded, got nil")
	}

	// Verify fields
	if loaded.ServerURL != cache.ServerURL {
		t.Errorf("ServerURL mismatch: expected %s, got %s", cache.ServerURL, loaded.ServerURL)
	}

	if loaded.AgentID != cache.AgentID {
		t.Errorf("AgentID mismatch: expected %s, got %s", cache.AgentID, loaded.AgentID)
	}

	if loaded.Tier != cache.Tier {
		t.Errorf("Tier mismatch: expected %s, got %s", cache.Tier, loaded.Tier)
	}

	if loaded.Signature != cache.Signature {
		t.Errorf("Signature mismatch: expected %s, got %s", cache.Signature, loaded.Signature)
	}

	// Cleanup
	_ = DeleteCache(serverKey)
}

func TestLoadCache_NotExists(t *testing.T) {
	serverKey := "nonexistent/server"

	loaded, err := LoadCache(serverKey)
	if err != nil {
		t.Fatalf("LoadCache failed: %v", err)
	}

	if loaded != nil {
		t.Error("expected nil for non-existent cache")
	}
}

func TestIsExpired(t *testing.T) {
	// Test nil cache
	if !IsExpired(nil) {
		t.Error("nil cache should be expired")
	}

	// Test expired cache
	expiredCache := &DirectURLCache{
		ExpiresAt: time.Now().Add(-1 * time.Hour).Unix(),
	}
	if !IsExpired(expiredCache) {
		t.Error("cache with past expiration should be expired")
	}

	// Test valid cache
	validCache := &DirectURLCache{
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
	}
	if IsExpired(validCache) {
		t.Error("cache with future expiration should not be expired")
	}
}

func TestLoadCache_ExpiredAutoDelete(t *testing.T) {
	// Create expired cache
	cache := &DirectURLCache{
		ServerURL: "https://example.com/mcp",
		AgentID:   "test-agent",
		Tier:      "pro",
		RateLimits: RateLimits{
			RPM: 100,
			RPH: 1000,
		},
		ExpiresAt: time.Now().Add(-1 * time.Hour).Unix(), // Expired
		Signature: "test-signature",
	}

	serverKey := "test/expired-server"

	// Save cache
	err := SaveCache(serverKey, cache)
	if err != nil {
		t.Fatalf("SaveCache failed: %v", err)
	}

	// Load cache - should return nil and delete file
	loaded, err := LoadCache(serverKey)
	if err != nil {
		t.Fatalf("LoadCache failed: %v", err)
	}

	if loaded != nil {
		t.Error("expected nil for expired cache")
	}

	// Verify file was deleted
	cachePath, _ := getCachePath(serverKey)
	if _, err := os.Stat(cachePath); !os.IsNotExist(err) {
		t.Error("expired cache file should be deleted")
	}
}

func TestDeleteCache(t *testing.T) {
	// Create cache
	cache := &DirectURLCache{
		ServerURL: "https://example.com/mcp",
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
	}

	serverKey := "test/delete-server"

	// Save cache
	err := SaveCache(serverKey, cache)
	if err != nil {
		t.Fatalf("SaveCache failed: %v", err)
	}

	// Delete cache
	err = DeleteCache(serverKey)
	if err != nil {
		t.Fatalf("DeleteCache failed: %v", err)
	}

	// Verify deleted
	loaded, err := LoadCache(serverKey)
	if err != nil {
		t.Fatalf("LoadCache failed: %v", err)
	}

	if loaded != nil {
		t.Error("cache should be deleted")
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"namespace/server", "namespace_server"},
		{"test-server", "test-server"},
		{"test_server", "test_server"},
		{"test@server", "test_server"},
		{"test/server/v1", "test_server_v1"},
	}

	for _, tt := range tests {
		result := sanitizeFilename(tt.input)
		if result != tt.expected {
			t.Errorf("sanitizeFilename(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestGetCachePath(t *testing.T) {
	serverKey := "test/server"
	path, err := getCachePath(serverKey)
	if err != nil {
		t.Fatalf("getCachePath failed: %v", err)
	}

	// Verify path contains expected components
	if !contains(path, ".omdr") {
		t.Error("cache path should contain .omdr")
	}

	if !contains(path, "cache") {
		t.Error("cache path should contain cache")
	}

	if !contains(path, "direct_urls") {
		t.Error("cache path should contain direct_urls")
	}

	if !contains(path, "test_server.json") {
		t.Error("cache path should contain sanitized filename")
	}
}

func contains(s, substr string) bool {
	return filepath.Base(s) == substr || filepath.Dir(s) == substr || len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
