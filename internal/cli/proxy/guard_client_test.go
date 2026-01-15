package proxy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGuardClient_GetDirectURL(t *testing.T) {
	// Create mock Guard server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.URL.Path != "/v1/auth/direct/test/server" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		if r.Method != "POST" {
			t.Errorf("unexpected method: %s", r.Method)
		}

		apiKey := r.Header.Get("X-OMDR-API-Key")
		if apiKey != "test-api-key" {
			t.Errorf("unexpected API key: %s", apiKey)
		}

		// Return mock direct URL response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"server_url": "https://mcp.example.com",
			"agent_id": "agent-123",
			"tier": "pro",
			"rate_limits": {
				"rpm": 100,
				"rph": 1000
			},
			"expires_at": 1737123456,
			"signature": "test-signature"
		}`))
	}))
	defer server.Close()

	// Create client
	client := NewGuardClient(server.URL, "test-api-key", "test/server")

	// Test GetDirectURL
	ctx := context.Background()
	directURL, err := client.GetDirectURL(ctx)
	if err != nil {
		t.Fatalf("GetDirectURL failed: %v", err)
	}

	// Verify response
	if directURL.ServerURL != "https://mcp.example.com" {
		t.Errorf("unexpected server URL: %s", directURL.ServerURL)
	}

	if directURL.AgentID != "agent-123" {
		t.Errorf("unexpected agent ID: %s", directURL.AgentID)
	}

	if directURL.Tier != "pro" {
		t.Errorf("unexpected tier: %s", directURL.Tier)
	}

	if directURL.RateLimits.RPM != 100 {
		t.Errorf("unexpected RPM: %d", directURL.RateLimits.RPM)
	}

	if directURL.RateLimits.RPH != 1000 {
		t.Errorf("unexpected RPH: %d", directURL.RateLimits.RPH)
	}

	if directURL.ExpiresAt != 1737123456 {
		t.Errorf("unexpected expires_at: %d", directURL.ExpiresAt)
	}

	if directURL.Signature != "test-signature" {
		t.Errorf("unexpected signature: %s", directURL.Signature)
	}
}

func TestGuardClient_GetDirectURL_AuthenticationFailed(t *testing.T) {
	// Create mock Guard server that returns 401
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Invalid API key"))
	}))
	defer server.Close()

	// Create client
	client := NewGuardClient(server.URL, "invalid-key", "test/server")

	// Test GetDirectURL
	ctx := context.Background()
	_, err := client.GetDirectURL(ctx)
	if err == nil {
		t.Error("expected error for authentication failure")
	}

	if err.Error() != "authentication failed: HTTP 401: Invalid API key" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestGuardClient_GetDirectURL_Timeout(t *testing.T) {
	// Create mock Guard server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create client with short timeout
	client := NewGuardClient(server.URL, "test-key", "test/server")
	client.httpClient.Timeout = 100 * time.Millisecond

	// Test GetDirectURL with timeout
	ctx := context.Background()
	_, err := client.GetDirectURL(ctx)
	if err == nil {
		t.Error("expected timeout error")
	}
}
