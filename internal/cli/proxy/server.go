package proxy

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/openmcpdirectory/omdr-cli/internal/cli/cache"
	clilogger "github.com/openmcpdirectory/omdr-cli/internal/cli/logger"
)

// Server implements an MCP protocol bridge over stdio
type Server struct {
	config Config
	client *GuardClient
	ctx    context.Context
	cancel context.CancelFunc
}

// Config holds proxy server configuration
type Config struct {
	ServerName string
	APIKey     string
	GuardURL   string
	AuthMode   string // "auth_only" or "full_proxy"
}

// NewServer creates a new MCP proxy server
func NewServer(config Config) *Server {
	ctx, cancel := context.WithCancel(context.Background())

	return &Server{
		config: config,
		client: NewGuardClient(config.GuardURL, config.APIKey, config.ServerName),
		ctx:    ctx,
		cancel: cancel,
	}
}

// ServeStdio starts the stdio JSON-RPC server
func (s *Server) ServeStdio() error {
	clilogger.Verbose("Starting MCP proxy for %s", s.config.ServerName)
	clilogger.Verbose("Guard URL: %s", s.config.GuardURL)
	clilogger.Verbose("Auth mode: %s", s.config.AuthMode)

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		clilogger.Verbose("Received shutdown signal")
		s.cancel()
	}()

	// For auth-only mode, authenticate once and get direct URL
	var directMCPURL string
	if s.config.AuthMode == "auth_only" {
		clilogger.Verbose("Auth-only mode: authenticating with Guard...")

		// Try to authenticate and get direct URL
		directURL, err := s.client.GetDirectURL(s.ctx)
		if err != nil {
			clilogger.Verbose("Auth-only authentication failed: %v, falling back to full proxy", err)
			// Fall back to full proxy mode
		} else {
			directMCPURL = directURL.ServerURL
			clilogger.Verbose("Auth-only mode: received direct URL: %s (expires at: %d)", directMCPURL, directURL.ExpiresAt)

			// Cache the direct URL for future use
			cacheData := &cache.DirectURLCache{
				ServerURL: directURL.ServerURL,
				AgentID:   directURL.AgentID,
				Tier:      directURL.Tier,
				RateLimits: cache.RateLimits{
					RPM: int(directURL.RateLimits.RPM),
					RPH: int(directURL.RateLimits.RPH),
				},
				ExpiresAt: directURL.ExpiresAt,
				Signature: directURL.Signature,
			}

			if err := cache.SaveCache(s.config.ServerName, cacheData); err != nil {
				clilogger.Verbose("Warning: Failed to cache direct URL: %v", err)
			} else {
				clilogger.Verbose("Direct URL cached successfully")
			}
		}
	}

	// Check guard health (skip if using direct URL)
	if directMCPURL == "" {
		if err := s.client.Health(s.ctx); err != nil {
			clilogger.Verbose("Warning: Guard health check failed: %v", err)
		}
	}

	// Start reading from stdin
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024) // 10MB max message size

	for scanner.Scan() {
		select {
		case <-s.ctx.Done():
			return nil
		default:
		}

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		clilogger.Verbose("Received request: %s", string(line))

		// Parse JSON-RPC request
		var req JSONRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			clilogger.Verbose("Parse error: %v", err)
			s.writeResponse(NewErrorResponse(nil, ParseError, "Invalid JSON"))
			continue
		}

		// Validate JSON-RPC version
		if req.JSONRPC != "2.0" {
			s.writeResponse(NewErrorResponse(req.ID, InvalidRequest, "Invalid JSON-RPC version"))
			continue
		}

		// Forward request based on mode
		var resp *JSONRPCResponse
		var err error

		if directMCPURL != "" {
			// Auth-only mode: forward directly to MCP server
			clilogger.Verbose("Forwarding directly to MCP server: %s", directMCPURL)
			resp, err = s.forwardDirect(s.ctx, &req, directMCPURL)
		} else {
			// Full proxy mode: forward through Guard
			clilogger.Verbose("Forwarding through Guard")
			resp, err = s.client.Forward(s.ctx, &req)
		}

		if err != nil {
			clilogger.Verbose("Forward error: %v", err)
			s.writeResponse(NewErrorResponse(req.ID, InternalError, fmt.Sprintf("Proxy error: %v", err)))
			continue
		}

		// Write response
		s.writeResponse(resp)
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		return fmt.Errorf("reading stdin: %w", err)
	}

	return nil
}

// writeResponse writes a JSON-RPC response to stdout
func (s *Server) writeResponse(resp *JSONRPCResponse) {
	data, err := json.Marshal(resp)
	if err != nil {
		clilogger.Verbose("Marshal error: %v", err)
		return
	}

	clilogger.Verbose("Sending response: %s", string(data))

	// Write to stdout with newline
	if _, err := os.Stdout.Write(append(data, '\n')); err != nil {
		clilogger.Verbose("Write error: %v", err)
	}
}

// forwardDirect sends a JSON-RPC request directly to the MCP server (auth-only mode)
func (s *Server) forwardDirect(ctx context.Context, req *JSONRPCRequest, mcpURL string) (*JSONRPCResponse, error) {
	// Marshal request
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", mcpURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	if req.ID != nil {
		httpReq.Header.Set("X-Request-ID", fmt.Sprintf("%v", req.ID))
	}

	// Send request
	httpClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.Do(httpReq)
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
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse JSON-RPC response
	var jsonrpcResp JSONRPCResponse
	if err := json.Unmarshal(respBody, &jsonrpcResp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return &jsonrpcResp, nil
}

// Close gracefully shuts down the server
func (s *Server) Close() error {
	s.cancel()
	return nil
}
