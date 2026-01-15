package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/openmcpdirectory/omdr-cli/internal/cli/registry"
	"github.com/openmcpdirectory/omdr-cli/internal/cli/secret"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run <server-name>",
	Short: "Run an installed MCP server with secure secret injection",
	Long:  "Execute an MCP server, injecting secrets from the secure keychain as environment variables. This is typically called by your MCP client (VS Code, Claude, etc).",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serverName := args[0]

		// Load registry
		reg, err := registry.NewLocalRegistry()
		if err != nil {
			return fmt.Errorf("initializing registry: %w", err)
		}

		// Get server config
		config, err := reg.Get(serverName)
		if err != nil {
			return fmt.Errorf("loading server config: %w", err)
		}

		// Prepare command
		execCmd := exec.CommandContext(cmd.Context(), config.Command, config.Args...)
		execCmd.Stdin = os.Stdin
		execCmd.Stdout = os.Stdout
		execCmd.Stderr = os.Stderr

		// Prepare environment
		env := os.Environ()

		// Add static env vars from registry
		for k, v := range config.Env {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}

		// Inject secrets from keychain
		for _, secretKey := range config.Secrets {
			// Secret identifier in keychain: "omdr-env:<server>:<key>"
			keychainUser := fmt.Sprintf("omdr-env:%s:%s", serverName, secretKey)
			val, err := secret.Get(secret.ServiceName, keychainUser)
			if err != nil {
				// Warn but continue? Or fail? Fail safe.
				fmt.Fprintf(os.Stderr, "Warning: Failed to retrieve secret '%s': %v\n", secretKey, err)
				continue
			}
			if val != "" {
				env = append(env, fmt.Sprintf("%s=%s", secretKey, val))
			}
		}

		// Inject global OMDR_API_KEY if authenticated
		if token, err := secret.Get("", ""); err == nil && token != "" {
			env = append(env, fmt.Sprintf("OMDR_API_KEY=%s", token))
		}

		execCmd.Env = env

		// Handle signals
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			sig := <-sigChan
			if execCmd.Process != nil {
				execCmd.Process.Signal(sig)
			}
		}()

		if err := execCmd.Run(); err != nil {
			// Pass through exit code if possible
			if exitErr, ok := err.(*exec.ExitError); ok {
				return exitErr // Cobra will exit with this code? No, usually prints error.
				// os.Exit(exitErr.ExitCode()) // Can't do this inside RunE easily without printing usage.
				// Just returning err is fine for now.
			}
			return fmt.Errorf("server execution failed: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}
