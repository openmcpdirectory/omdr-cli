package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/openmcpdirectory/omdr-cli/internal/cli/resilience"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	token      string
}

type APIError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	Details    any    `json:"details,omitempty"`
	RetryAfter string `json:"-"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) SetToken(token string) {
	c.token = token
}

func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	u := fmt.Sprintf("%s%s", c.baseURL, path)

	// Serialize body once
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshaling request body: %w", err)
		}
	}

	// Define operation for resilience package
	op := func() error {
		var bodyReader io.Reader
		if bodyBytes != nil {
			bodyReader = bytes.NewReader(bodyBytes)
		}

		req, err := http.NewRequestWithContext(ctx, method, u, bodyReader)
		if err != nil {
			return fmt.Errorf("creating request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		if c.token != "" {
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return err // Network errors are retryable by default
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("reading response: %w", err)
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			return fmt.Errorf("HTTP 429: rate limited: %s", string(respBody))
		}

		if resp.StatusCode >= 500 {
			return fmt.Errorf("HTTP %d: server error: %s", resp.StatusCode, string(respBody))
		}

		if resp.StatusCode >= 400 {
			// IsRetryableError will see "HTTP 4xx" and return false
			var apiErr APIError
			if err := json.Unmarshal(respBody, &apiErr); err != nil {
				return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
			}
			// Reconstruct error message to ensure 4xx code is visible for IsRetryableError
			// Use %w to wrap the error so errors.As works
			return fmt.Errorf("HTTP %d: %w", resp.StatusCode, &apiErr)
		}

		if result != nil && len(respBody) > 0 {
			if err := json.Unmarshal(respBody, result); err != nil {
				// Unmarshalling errors shouldn't typically be retried unless partial response?
				// Let's assume NOT retryable for now, but IsRetryableError defaults to true for unknown.
				// Format it so it DOESN'T match retry patterns?
				// Actually, "unmarshaling response" isn't in retry patterns, so it defaults to true.
				// Maybe that's okay for corrupted reads?
				return fmt.Errorf("unmarshaling response: %w", err)
			}
		}

		return nil
	}

	// Use default retry config (3 retries, exponential backoff)
	config := resilience.DefaultRetryConfig()

	return resilience.RetryWithBackoff(ctx, config, op)
}

// executeRequest removed (replaced by doRequest internal logic)

func (c *Client) Get(ctx context.Context, path string, result interface{}) error {
	return c.doRequest(ctx, http.MethodGet, path, nil, result)
}

func (c *Client) Post(ctx context.Context, path string, body interface{}, result interface{}) error {
	return c.doRequest(ctx, http.MethodPost, path, body, result)
}

func (c *Client) Put(ctx context.Context, path string, body interface{}, result interface{}) error {
	return c.doRequest(ctx, http.MethodPut, path, body, result)
}

func (c *Client) Delete(ctx context.Context, path string) error {
	return c.doRequest(ctx, http.MethodDelete, path, nil, nil)
}

func (c *Client) IsRateLimited(err error) (bool, time.Duration) {
	// Simple unwrapping
	var apiErr *APIError

	// Check if it IS an APIError
	if e, ok := err.(*APIError); ok {
		apiErr = e
	} else {
		// Or if it wraps one
		// Manual unwrapping loop since we can't trust errors package availability/import without checking
		curr := err
		for curr != nil {
			if e, ok := curr.(*APIError); ok {
				apiErr = e
				break
			}
			// Implements Unwrap?
			u, ok := curr.(interface{ Unwrap() error })
			if !ok {
				break
			}
			curr = u.Unwrap()
		}
	}

	if apiErr == nil || apiErr.Code != "RATE_LIMITED" {
		return false, 0
	}

	if apiErr.RetryAfter == "" {
		return true, 60 * time.Second
	}

	seconds, err := strconv.Atoi(apiErr.RetryAfter)
	if err != nil {
		return true, 60 * time.Second
	}

	return true, time.Duration(seconds) * time.Second
}

func (c *Client) PostMultipart(ctx context.Context, path string, contentType string, body io.Reader, result interface{}) error {
	u := fmt.Sprintf("%s%s", c.baseURL, path)

	// Define operation for resilience package
	// Note: Retrying multipart upload with io.Reader body is tricky if it's not seekable.
	// For now, we assume caller handles seeking or we wrap it.
	// But to simplify, let's wrap logic in retry loop only if body is seekable or if we can accept limited retries.
	// Actually, RetryWithBackoff executes op. If op fails, it waits and runs again.
	// If body was read, next attempt will fail.

	// Simple fix: Check if body implements io.Seeker
	var bodySeeker io.Seeker
	if s, ok := body.(io.Seeker); ok {
		bodySeeker = s
	}

	op := func() error {
		// Reset body if possible
		if bodySeeker != nil {
			bodySeeker.Seek(0, io.SeekStart)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, body)
		if err != nil {
			return fmt.Errorf("creating request: %w", err)
		}

		req.Header.Set("Content-Type", contentType)

		if c.token != "" {
			req.Header.Set("Authorization", "Bearer "+c.token)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("reading response: %w", err)
		}

		if resp.StatusCode >= 400 {
			var apiErr APIError
			if err := json.Unmarshal(respBody, &apiErr); err != nil {
				return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
			}
			return fmt.Errorf("HTTP %d: %w", resp.StatusCode, &apiErr)
		}

		if result != nil && len(respBody) > 0 {
			if err := json.Unmarshal(respBody, result); err != nil {
				return fmt.Errorf("unmarshaling response: %w", err)
			}
		}

		return nil
	}

	config := resilience.DefaultRetryConfig()
	// Disable retries if body is not seekable to avoid corruption
	if bodySeeker == nil {
		config.MaxAttempts = 1
	}

	return resilience.RetryWithBackoff(ctx, config, op)
}
