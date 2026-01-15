package runtime

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Masterminds/semver/v3"
)

// EngineRequirements represents the required versions of runtimes
type EngineRequirements map[string]string

// CheckEngineRequirements verifies that installed runtimes meet the specified requirements
func CheckEngineRequirements(requirements map[string]string) error {
	for engine, constraint := range requirements {
		if constraint == "" {
			continue
		}

		var err error
		switch strings.ToLower(engine) {
		case "node":
			err = checkNodeVersion(constraint)
		case "python":
			err = checkPythonVersion(constraint)
		case "docker":
			err = checkDockerVersion(constraint)
		// Add other engines as needed
		default:
			// For unknown engines, we log a warning but don't fail properly,
			// though typical behavior is usually strict or permissible.
			// Let's assume permissible for now to allow forward compatibility,
			// or maybe we should just ignore?
			// To match typical package manager behavior (npm/pip), we might warn only.
			// Currently implementation will only check what we instantiate.
			continue
		}

		if err != nil {
			return fmt.Errorf("engine requirement check failed for %s: %w", engine, err)
		}
	}
	return nil
}

func checkNodeVersion(constraint string) error {
	result := CheckNode()
	if !result.Available {
		// If engine is required but not installed, that's an error
		return fmt.Errorf("Node.js is required (%s) but not installed", constraint)
	}

	// Clean version string (e.g., "v18.16.0" -> "18.16.0")
	versionStr := strings.TrimPrefix(result.Version, "v")
	return validateVersion(versionStr, constraint)
}

func checkDockerVersion(constraint string) error {
	result := CheckDocker(false)
	if !result.Available {
		return fmt.Errorf("Docker is required (%s) but not installed", constraint)
	}

	// Docker version often looks like "24.0.5, build ced0996" or just "24.0.5"
	// extractVersionNumber handles basic X.Y.Z
	versionStr := extractVersionNumber(result.Version)
	if versionStr == "" {
		return fmt.Errorf("could not parse Docker version from '%s'", result.Version)
	}

	return validateVersion(versionStr, constraint)
}

func checkPythonVersion(constraint string) error {
	result := CheckPython()
	if !result.Available {
		return fmt.Errorf("Python is required (%s) but not installed", constraint)
	}

	// Python output often looks like "Python 3.11.4"
	// We need to extract just the version number
	versionStr := extractVersionNumber(result.Version)
	if versionStr == "" {
		return fmt.Errorf("could not parse Python version from '%s'", result.Version)
	}

	return validateVersion(versionStr, constraint)
}

func validateVersion(versionStr, constraintStr string) error {
	v, err := semver.NewVersion(versionStr)
	if err != nil {
		return fmt.Errorf("invalid installed version '%s': %w", versionStr, err)
	}

	c, err := semver.NewConstraint(constraintStr)
	if err != nil {
		return fmt.Errorf("invalid constraint '%s': %w", constraintStr, err)
	}

	if !c.Check(v) {
		return fmt.Errorf("installed version %s does not satisfy requirement %s", versionStr, constraintStr)
	}

	return nil
}

// extractVersionNumber extracts "X.Y.Z" from strings like "Python 3.11.4"
func extractVersionNumber(s string) string {
	re := regexp.MustCompile(`(\d+\.\d+(?:\.\d+)?)`)
	matches := re.FindStringSubmatch(s)
	if len(matches) > 1 {
		return matches[1]
	}
	return strings.TrimSpace(s) // Fallback if no pattern match, though unlikely for standard output
}
