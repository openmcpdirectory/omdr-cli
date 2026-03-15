package mcpspec

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

var (
	semverRegex = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`)

	validate *validator.Validate
)

func init() {
	validate = validator.New()
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

// ValidateManifest validates an MCP manifest against the schema, including
// OMDR extension cross-field rules.
func ValidateManifest(manifest *MCPManifest) error {
	if manifest == nil {
		return ValidationErrors{{
			Field:   "manifest",
			Message: "manifest cannot be nil",
		}}
	}

	err := validate.Struct(manifest)

	var errors ValidationErrors

	if err != nil {
		validationErrs, ok := err.(validator.ValidationErrors)
		if !ok {
			return fmt.Errorf("validation failed: %w", err)
		}
		for _, fieldErr := range validationErrs {
			errors = append(errors, ValidationError{
				Field:   getFieldPath(fieldErr),
				Message: getErrorMessage(fieldErr),
				Value:   fieldErr.Value(),
			})
		}
	}

	if manifest.OMDR != nil {
		errors = append(errors, validateOMDRExtension(manifest.OMDR)...)
	}

	if len(errors) > 0 {
		return errors
	}
	return nil
}

func validateOMDRExtension(ext *OMDRExtension) ValidationErrors {
	var errs ValidationErrors

	if ext.Deployment == DeployHosted && ext.Hosting == nil {
		errs = append(errs, ValidationError{
			Field:   "omdr.hosting",
			Message: "hosting config is required when deployment is \"hosted\"",
		})
	}

	if ext.Deployment == DeploySelfHosted {
		if ext.Hosting == nil || ext.Hosting.EndpointURL == "" {
			errs = append(errs, ValidationError{
				Field:   "omdr.hosting.endpoint_url",
				Message: "endpoint_url is required when deployment is \"self_hosted\"",
			})
		}
	}

	if ext.Hosting != nil && ext.Deployment == DeployHosted && ext.Hosting.ArtifactType == "" {
		errs = append(errs, ValidationError{
			Field:   "omdr.hosting.artifact_type",
			Message: "artifact_type is required for hosted deployment",
		})
	}

	if ext.Hosting != nil && ext.Hosting.ArtifactType == ArtifactDocker {
		if ext.Hosting.Dockerfile == "" && ext.Hosting.GitHubURL == "" {
			errs = append(errs, ValidationError{
				Field:   "omdr.hosting.dockerfile",
				Message: "dockerfile or github_url is required for docker artifact",
			})
		}
	}

	if ext.Pricing != nil {
		switch ext.Pricing.Model {
		case PricingPerCall:
			if ext.Pricing.PerCallCents <= 0 {
				errs = append(errs, ValidationError{
					Field:   "omdr.pricing.per_call_cents",
					Message: "per_call_cents must be > 0 when model is \"per_call\"",
				})
			}
		case PricingSubscription:
			if ext.Pricing.MonthlyCents <= 0 {
				errs = append(errs, ValidationError{
					Field:   "omdr.pricing.monthly_cents",
					Message: "monthly_cents must be > 0 when model is \"subscription\"",
				})
			}
		}
	}

	if ext.Scaling != nil && ext.Scaling.MaxInstances > 0 && ext.Scaling.MinInstances > ext.Scaling.MaxInstances {
		errs = append(errs, ValidationError{
			Field:   "omdr.scaling.min_instances",
			Message: "min_instances cannot exceed max_instances",
		})
	}

	return errs
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

func validateSemver(fl validator.FieldLevel) bool {
	version := fl.Field().String()
	return semverRegex.MatchString(version)
}

func getFieldPath(fe validator.FieldError) string {
	field := fe.Field()
	if len(field) > 0 {
		field = strings.ToLower(field[:1]) + field[1:]
	}

	namespace := fe.Namespace()
	if idx := strings.Index(namespace, "."); idx != -1 {
		path := namespace[idx+1:]
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
