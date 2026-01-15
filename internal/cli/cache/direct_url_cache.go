package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// DirectURLCache represents a cached direct connection URL for auth-only mode
type DirectURLCache struct {
	ServerURL  string     `json:"server_url"`
	AgentID    string     `json:"agent_id"`
	Tier       string     `json:"tier"`
	RateLimits RateLimits `json:"rate_limits"`
	ExpiresAt  int64      `json:"expires_at"`
	Signature  string     `json:"signature"`
	CachedAt   int64      `json:"cached_at"`
}

// RateLimits represents rate limiting configuration
type RateLimits struct {
	RPM int `json:"rpm"` // Requests per minute
	RPH int `json:"rph"` // Requests per hour
}

// LoadCache loads a cached direct URL for the given server key
// Returns nil if cache doesn't exist or is expired
func LoadCache(serverKey string) (*DirectURLCache, error) {
	cachePath, err := getCachePath(serverKey)
	if err != nil {
		return nil, err
	}

	// Check if cache file exists
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		return nil, nil // No cache exists
	}

	// Read cache file
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, fmt.Errorf("reading cache file: %w", err)
	}

	// Parse JSON
	var cache DirectURLCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, fmt.Errorf("parsing cache: %w", err)
	}

	// Check if expired
	if IsExpired(&cache) {
		// Delete expired cache
		_ = os.Remove(cachePath)
		return nil, nil
	}

	return &cache, nil
}

// SaveCache saves a direct URL cache for the given server key
func SaveCache(serverKey string, cache *DirectURLCache) error {
	cachePath, err := getCachePath(serverKey)
	if err != nil {
		return err
	}

	// Ensure cache directory exists
	cacheDir := filepath.Dir(cachePath)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("creating cache directory: %w", err)
	}

	// Set cached timestamp
	cache.CachedAt = time.Now().Unix()

	// Marshal to JSON
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling cache: %w", err)
	}

	// Write to file
	if err := os.WriteFile(cachePath, data, 0600); err != nil {
		return fmt.Errorf("writing cache file: %w", err)
	}

	return nil
}

// IsExpired checks if a cached direct URL has expired
func IsExpired(cache *DirectURLCache) bool {
	if cache == nil {
		return true
	}
	return time.Now().Unix() > cache.ExpiresAt
}

// DeleteCache removes the cached direct URL for the given server key
func DeleteCache(serverKey string) error {
	cachePath, err := getCachePath(serverKey)
	if err != nil {
		return err
	}

	if err := os.Remove(cachePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("deleting cache: %w", err)
	}

	return nil
}

// getCachePath returns the file path for a server's cache
func getCachePath(serverKey string) (string, error) {
	// Get user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}

	// Create cache directory path: ~/.omdr/cache/direct_urls/
	cacheDir := filepath.Join(homeDir, ".omdr", "cache", "direct_urls")

	// Sanitize server key for use as filename
	filename := sanitizeFilename(serverKey) + ".json"

	return filepath.Join(cacheDir, filename), nil
}

// sanitizeFilename converts a server key to a safe filename
// Replaces slashes and special characters
func sanitizeFilename(serverKey string) string {
	// Replace / with _
	safe := ""
	for _, ch := range serverKey {
		if ch == '/' || ch == '\\' {
			safe += "_"
		} else if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '_' {
			safe += string(ch)
		} else {
			safe += "_"
		}
	}
	return safe
}
