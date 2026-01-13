package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
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

func (c *Client) doRequest(method, path string, body interface{}, result interface{}) error {
	return c.doRequestWithRetry(method, path, body, result, 3)
}

func (c *Client) doRequestWithRetry(method, path string, body interface{}, result interface{}, maxRetries int) error {
	var lastErr error
	backoff := 1 * time.Second

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(backoff)
			backoff *= 2
		}

		err := c.executeRequest(method, path, body, result)
		if err == nil {
			return nil
		}

		lastErr = err

		if !isRetryable(err) {
			return err
		}

		if apiErr, ok := err.(*APIError); ok && apiErr.Code == "RATE_LIMITED" {
			if apiErr.RetryAfter != "" {
				seconds, parseErr := strconv.Atoi(apiErr.RetryAfter)
				if parseErr == nil {
					time.Sleep(time.Duration(seconds) * time.Second)
					continue
				}
			}
		}
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}

func (c *Client) executeRequest(method, path string, body interface{}, result interface{}) error {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshaling request: %w", err)
		}
		reqBody = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequest(method, c.baseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		apiErr := &APIError{
			Code:       "RATE_LIMITED",
			Message:    "Rate limit exceeded",
			RetryAfter: resp.Header.Get("Retry-After"),
		}
		return apiErr
	}

	if resp.StatusCode >= 400 {
		var apiErr APIError
		if err := json.Unmarshal(respBody, &apiErr); err != nil {
			return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
		}
		return &apiErr
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("unmarshaling response: %w", err)
		}
	}

	return nil
}

func isRetryable(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.Code == "RATE_LIMITED" || apiErr.Code == "INTERNAL_ERROR"
	}

	errStr := err.Error()
	return bytes.Contains([]byte(errStr), []byte("timeout")) ||
		bytes.Contains([]byte(errStr), []byte("connection refused")) ||
		bytes.Contains([]byte(errStr), []byte("temporary failure"))
}

func (c *Client) Get(path string, result interface{}) error {
	return c.doRequest(http.MethodGet, path, nil, result)
}

func (c *Client) Post(path string, body interface{}, result interface{}) error {
	return c.doRequest(http.MethodPost, path, body, result)
}

func (c *Client) Put(path string, body interface{}, result interface{}) error {
	return c.doRequest(http.MethodPut, path, body, result)
}

func (c *Client) Delete(path string) error {
	return c.doRequest(http.MethodDelete, path, nil, nil)
}

func (c *Client) IsRateLimited(err error) (bool, time.Duration) {
	apiErr, ok := err.(*APIError)
	if !ok || apiErr.Code != "RATE_LIMITED" {
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

func (c *Client) PostMultipart(path string, contentType string, body io.Reader, result interface{}) error {
	req, err := http.NewRequest(http.MethodPost, c.baseURL+path, body)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", contentType)

	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
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
		return &apiErr
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("unmarshaling response: %w", err)
		}
	}

	return nil
}
