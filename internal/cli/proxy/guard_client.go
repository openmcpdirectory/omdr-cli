package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/openmcpdirectory/omdr-cli/internal/cli/resilience"
)

// GuardClient handles communication with omdr-guard
type GuardClient struct {
	baseURL    string
	apiKey     string
	serverName string
	httpClient *http.Client
	cb         *resilience.CircuitBreaker
}

// NewGuardClient creates a new guard client
func NewGuardClient(baseURL, apiKey, serverName string) *GuardClient {
	return &GuardClient{
		baseURL:    baseURL,
		apiKey:     apiKey,
		serverName: serverName,
		httpClient: &http.Client{
			// Short timeout for guard requests to fail fast
			Timeout: 5 * time.Second,
		},
		// Initialize circuit breaker: 5 failures to open, 30s reset timeout
		cb: resilience.NewCircuitBreaker(5, 30*time.Second),
	}
}

// Forward forwards a JSON-RPC request to omdr-guard
func (c *GuardClient) Forward(ctx context.Context, req *JSONRPCRequest) (*JSONRPCResponse, error) {
	var resp *JSONRPCResponse

	err := c.cb.Call(func() error {
		// Marshal request
		body, err := json.Marshal(req)
		if err != nil {
			return resilience.PermissionError(fmt.Errorf("marshaling request: %w", err))
		}

		// Build URL
		url := fmt.Sprintf("%s/v1/proxy/%s", c.baseURL, c.serverName)

		// Create HTTP request
		httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
		if err != nil {
			return resilience.PermissionError(fmt.Errorf("creating request: %w", err))
		}

		// Set headers
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("X-OMDR-API-Key", c.apiKey)
		if req.ID != nil {
			httpReq.Header.Set("X-OMDR-Request-ID", fmt.Sprintf("%v", req.ID))
		}

		// Send request
		httpResp, err := c.httpClient.Do(httpReq)
		if err != nil {
			return fmt.Errorf("sending request: %w", err)
		}
		defer httpResp.Body.Close()

		// Read response body
		respBody, err := io.ReadAll(httpResp.Body)
		if err != nil {
			return fmt.Errorf("reading response: %w", err)
		}

		// Handle HTTP errors
		if httpResp.StatusCode != http.StatusOK {
			// If it's a 5xx error, it's a failure. 4xx is permission/client error.
			if httpResp.StatusCode >= 500 {
				// We still want to preserve the error response if possible, but for circuit breaker,
				// we return error to trip it if it's a server failure.
				// However, Forward is expected to return *JSONRPCResponse even on error sometimes?
				// No, the original code called handleHTTPError which returns *JSONRPCResponse.
				// If we return error here, c.cb.Call returns error.

				// Let's modify handleHTTPError to NOT return *JSONRPCResponse, but error?
				// Or we return nil error but set resp?
				// The requirement is to PROTECT the proxy from Guard failures.
				// So if Guard is 500ing, we want to trip CB.
				return fmt.Errorf("guard error %d: %s", httpResp.StatusCode, string(respBody))
			}

			// For 4xx, we might want to return the JSON-RPC error response normally.
			// But cb.Call returns error.
			// So we need to set resp and return nil, OR return a specific error type that ignores CB?
			// resilience.PermissionError ignores CB.

			jsonRpcResp, _ := c.handleHTTPError(req.ID, httpResp.StatusCode, respBody)
			resp = jsonRpcResp
			return nil
		}

		// Parse JSON-RPC response
		var jsonrpcResp JSONRPCResponse
		if err := json.Unmarshal(respBody, &jsonrpcResp); err != nil {
			return resilience.PermissionError(fmt.Errorf("parsing response: %w", err))
		}

		resp = &jsonrpcResp
		return nil
	})

	if err != nil {
		return nil, err
	}

	return resp, nil
}

// handleHTTPError converts HTTP errors to JSON-RPC errors
func (c *GuardClient) handleHTTPError(id interface{}, statusCode int, body []byte) (*JSONRPCResponse, error) {
	var message string

	switch statusCode {
	case http.StatusUnauthorized:
		message = "Authentication required. Run 'omdr auth login'"
	case http.StatusPaymentRequired:
		message = "Insufficient credits. Run 'omdr credits buy <amount>'"
	case http.StatusForbidden:
		message = "Access denied to this server"
	case http.StatusNotFound:
		message = "Server not found or not hosted"
	case http.StatusTooManyRequests:
		message = "Rate limit exceeded. Please try again later"
	case http.StatusServiceUnavailable:
		message = "Server temporarily unavailable"
	default:
		message = fmt.Sprintf("HTTP %d: %s", statusCode, string(body))
	}

	return NewErrorResponse(id, InternalError, message), nil
}

// Health checks if the guard is reachable
func (c *GuardClient) Health(ctx context.Context) error {
	return c.cb.Call(func() error {
		url := fmt.Sprintf("%s/health", c.baseURL)

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return resilience.PermissionError(err)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			if resp.StatusCode >= 500 {
				return fmt.Errorf("guard unhealthy: HTTP %d", resp.StatusCode)
			}
			return resilience.PermissionError(fmt.Errorf("guard unhealthy: HTTP %d", resp.StatusCode))
		}

		return nil
	})
}

// DirectURLResponse represents the response from auth-only authentication
type DirectURLResponse struct {
	ServerURL  string `json:"server_url"`
	AgentID    string `json:"agent_id"`
	Tier       string `json:"tier"`
	RateLimits struct {
		RPM int `json:"rpm"`
		RPH int `json:"rph"`
	} `json:"rate_limits"`
	ExpiresAt int64  `json:"expires_at"`
	Signature string `json:"signature"`
}

// GetDirectURL authenticates and retrieves a direct connection URL for auth-only mode
func (c *GuardClient) GetDirectURL(ctx context.Context) (*DirectURLResponse, error) {
	var result *DirectURLResponse

	err := c.cb.Call(func() error {
		url := fmt.Sprintf("%s/v1/auth/direct/%s", c.baseURL, c.serverName)

		req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
		if err != nil {
			return resilience.PermissionError(fmt.Errorf("creating request: %w", err))
		}

		req.Header.Set("X-OMDR-API-Key", c.apiKey)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("sending request: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("reading response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			if resp.StatusCode >= 500 {
				return fmt.Errorf("authentication failed: HTTP %d: %s", resp.StatusCode, string(body))
			}
			return resilience.PermissionError(fmt.Errorf("authentication failed: HTTP %d: %s", resp.StatusCode, string(body)))
		}

		var directURL DirectURLResponse
		if err := json.Unmarshal(body, &directURL); err != nil {
			return resilience.PermissionError(fmt.Errorf("parsing response: %w", err))
		}

		result = &directURL
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}
