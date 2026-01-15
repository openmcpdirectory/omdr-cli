package runtime

import (
	"testing"
)

func TestExtractVersionNumber(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Python 3.11.4", "3.11.4"},
		{"Python 2.7.18", "2.7.18"},
		{"3.9.6", "3.9.6"},
		{"v18.16.0", "18.16.0"}, // Regex catches 18.16.0
		{"Foo Bar 1.2", "1.2"},
	}

	for _, tt := range tests {
		got := extractVersionNumber(tt.input)
		if got != tt.expected {
			t.Errorf("extractVersionNumber(%q) = %q; want %q", tt.input, got, tt.expected)
		}
	}
}

func TestValidateVersion(t *testing.T) {
	tests := []struct {
		version    string
		constraint string
		wantErr    bool
	}{
		{"1.0.0", ">=1.0.0", false},
		{"1.0.0", ">1.0.0", true},
		{"2.0.0", ">=1.0.0", false},
		{"0.9.0", ">=1.0.0", true},
		{"18.16.0", ">=16", false},
		{"14.0.0", ">=16", true},
		{"invalid", ">=1.0.0", true},
		{"1.0.0", "invalid", true},
	}

	for _, tt := range tests {
		err := validateVersion(tt.version, tt.constraint)
		if (err != nil) != tt.wantErr {
			t.Errorf("validateVersion(%q, %q) error = %v, wantErr %v", tt.version, tt.constraint, err, tt.wantErr)
		}
	}
}

// Mock exec for integration like tests?
// Since CheckNode/CheckPython use exec.Command, unit testing extracting implementation logic
// requires either mocking exec (complex in Go) or relying on extraction logic tests above.
// The integration part (actually running python/node) runs against the host system.
