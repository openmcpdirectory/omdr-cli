package runtime

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// CheckResult represents the result of a runtime check
type CheckResult struct {
	Available bool
	Version   string
	Error     error
}

// CheckDocker verifies Docker is installed and optionally checks if daemon is running
func CheckDocker(checkDaemon bool) CheckResult {
	cmd := exec.Command("docker", "--version")
	output, err := cmd.Output()

	if err != nil {
		return CheckResult{
			Available: false,
			Error:     fmt.Errorf("docker not found"),
		}
	}

	version := strings.TrimSpace(string(output))

	if !checkDaemon {
		return CheckResult{
			Available: true,
			Version:   version,
		}
	}

	// Check if Docker daemon is running
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pingCmd := exec.CommandContext(ctx, "docker", "info")
	if err := pingCmd.Run(); err != nil {
		return CheckResult{
			Available: true,
			Version:   version,
			Error:     fmt.Errorf("daemon not running"),
		}
	}

	return CheckResult{
		Available: true,
		Version:   version,
	}
}

// CheckNode verifies Node.js is installed
func CheckNode() CheckResult {
	cmd := exec.Command("node", "--version")
	output, err := cmd.Output()

	if err != nil {
		return CheckResult{
			Available: false,
			Error:     fmt.Errorf("node not found"),
		}
	}

	version := strings.TrimSpace(string(output))
	return CheckResult{
		Available: true,
		Version:   version,
	}
}

// CheckPython verifies Python is installed
func CheckPython() CheckResult {
	// Try python3 first, then python
	cmd := exec.Command("python3", "--version")
	output, err := cmd.Output()

	if err != nil {
		cmd = exec.Command("python", "--version")
		output, err = cmd.Output()
		if err != nil {
			return CheckResult{
				Available: false,
				Error:     fmt.Errorf("python not found"),
			}
		}
	}

	version := strings.TrimSpace(string(output))
	return CheckResult{
		Available: true,
		Version:   version,
	}
}
