package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// GuardClient handles communication with omdr-guard
type GuardClient struct {
	baseURL    string
	apiKey     string
	serverName string
	httpClient *http.Client
}

// NewGuardClient creates a new guard client
func NewGuardClient(baseURL, apiKey, serverName string) *GuardClient {
	return &GuardClient{
		baseURL:    baseURL,
		apiKey:     apiKey,
		serverName: serverName,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Forward forwards a JSON-RPC request to omdr-guard
func (c *GuardClient) Forward(ctx context.Context, req *JSONRPCRequest) (*JSONRPCResponse, error) {
	// Marshal request
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	// Build URL
	url := fmt.Sprintf("%s/v1/proxy/%s", c.baseURL, c.serverName)

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-OMDR-API-Key", c.apiKey)
	if req.ID != nil {
		httpReq.Header.Set("X-OMDR-Request-ID", fmt.Sprintf("%v", req.ID))
	}

	// Send request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	// Handle HTTP errors
	if resp.StatusCode != http.StatusOK {
		return c.handleHTTPError(req.ID, resp.StatusCode, respBody)
	}

	// Parse JSON-RPC response
	var jsonrpcResp JSONRPCResponse
	if err := json.Unmarshal(respBody, &jsonrpcResp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return &jsonrpcResp, nil
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
	url := fmt.Sprintf("%s/health", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("guard unhealthy: HTTP %d", resp.StatusCode)
	}

	return nil
}
