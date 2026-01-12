package mcpspec

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

var (
	// semverRegex matches semantic versioning format (MAJOR.MINOR.PATCH with optional pre-release and build metadata)
	semverRegex = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`)

	validate *validator.Validate
)

func init() {
	validate = validator.New()

	// Register custom semver validator
	validate.RegisterValidation("semver", validateSemver)
}

// ValidationError represents a structured validation error with field path
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Value   any    `json:"value,omitempty"`
}

// ValidationErrors is a collection of validation errors
type ValidationErrors []ValidationError

func (v ValidationErrors) Error() string {
	if len(v) == 0 {
		return "validation failed"
	}

	var msgs []string
	for _, err := range v {
		msgs = append(msgs, fmt.Sprintf("%s: %s", err.Field, err.Message))
	}
	return strings.Join(msgs, "; ")
}

// ValidateManifest validates an MCP manifest against the schema
func ValidateManifest(manifest *MCPManifest) error {
	if manifest == nil {
		return ValidationErrors{{
			Field:   "manifest",
			Message: "manifest cannot be nil",
		}}
	}

	err := validate.Struct(manifest)
	if err == nil {
		return nil
	}

	// Convert validator errors to structured validation errors
	validationErrs, ok := err.(validator.ValidationErrors)
	if !ok {
		return fmt.Errorf("validation failed: %w", err)
	}

	var errors ValidationErrors
	for _, fieldErr := range validationErrs {
		errors = append(errors, ValidationError{
			Field:   getFieldPath(fieldErr),
			Message: getErrorMessage(fieldErr),
			Value:   fieldErr.Value(),
		})
	}

	return errors
}

// ValidateManifestJSON validates a manifest from JSON bytes
func ValidateManifestJSON(data []byte) (*MCPManifest, error) {
	if len(data) == 0 {
		return nil, ValidationErrors{{
			Field:   "manifest",
			Message: "manifest data is empty",
		}}
	}

	var manifest MCPManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, ValidationErrors{{
			Field:   "manifest",
			Message: fmt.Sprintf("invalid JSON: %s", err.Error()),
		}}
	}

	if err := ValidateManifest(&manifest); err != nil {
		return nil, err
	}

	return &manifest, nil
}

// validateSemver is a custom validator for semantic versioning
func validateSemver(fl validator.FieldLevel) bool {
	version := fl.Field().String()
	return semverRegex.MatchString(version)
}

// getFieldPath extracts the JSON field path from a validator field error
func getFieldPath(fe validator.FieldError) string {
	// Convert struct field name to JSON field name (lowercase first letter)
	field := fe.Field()
	if len(field) > 0 {
		field = strings.ToLower(field[:1]) + field[1:]
	}

	// Handle nested fields
	namespace := fe.Namespace()
	if idx := strings.Index(namespace, "."); idx != -1 {
		// Remove the struct name prefix
		path := namespace[idx+1:]
		// Convert to JSON-style path
		parts := strings.Split(path, ".")
		for i, part := range parts {
			if len(part) > 0 {
				parts[i] = strings.ToLower(part[:1]) + part[1:]
			}
		}
		return strings.Join(parts, ".")
	}

	return field
}

// getErrorMessage generates a human-readable error message
func getErrorMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "field is required"
	case "semver":
		return "must be valid semantic version (e.g., 1.0.0)"
	case "min":
		return fmt.Sprintf("must be at least %s", fe.Param())
	case "max":
		return fmt.Sprintf("must be at most %s", fe.Param())
	case "oneof":
		return fmt.Sprintf("must be one of: %s", fe.Param())
	default:
		return fmt.Sprintf("validation failed on '%s'", fe.Tag())
	}
}
