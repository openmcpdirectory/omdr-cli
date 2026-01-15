package proxy

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

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

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		clilogger.Verbose("Received shutdown signal")
		s.cancel()
	}()

	// Check guard health
	if err := s.client.Health(s.ctx); err != nil {
		clilogger.Verbose("Warning: Guard health check failed: %v", err)
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

		// Forward to guard
		resp, err := s.client.Forward(s.ctx, &req)
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

// Close gracefully shuts down the server
func (s *Server) Close() error {
	s.cancel()
	return nil
}
