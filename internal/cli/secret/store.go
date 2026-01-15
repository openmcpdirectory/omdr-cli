package secret

import (
	"fmt"

	"github.com/zalando/go-keyring"
)

const (
	// ServiceName is the service name used for keychain entries
	ServiceName = "omdr-cli"

	// DefaultUser is the default username/key for the token
	DefaultUser = "auth-token"
)

// Store securely stores a secret in the system keychain
func Store(service, user, password string) error {
	if service == "" {
		service = ServiceName
	}
	if user == "" {
		user = DefaultUser
	}

	if err := keyring.Set(service, user, password); err != nil {
		return fmt.Errorf("failed to store secret in keychain: %w", err)
	}
	return nil
}

// Get retrieves a secret from the system keychain
func Get(service, user string) (string, error) {
	if service == "" {
		service = ServiceName
	}
	if user == "" {
		user = DefaultUser
	}

	secret, err := keyring.Get(service, user)
	if err != nil {
		if err == keyring.ErrNotFound {
			return "", nil // Not found is not an error, just empty
		}
		return "", fmt.Errorf("failed to retrieve secret from keychain: %w", err)
	}
	return secret, nil
}

// Delete removes a secret from the system keychain
func Delete(service, user string) error {
	if service == "" {
		service = ServiceName
	}
	if user == "" {
		user = DefaultUser
	}

	if err := keyring.Delete(service, user); err != nil {
		// Ignore not found errors during delete
		if err == keyring.ErrNotFound {
			return nil
		}
		return fmt.Errorf("failed to delete secret from keychain: %w", err)
	}
	return nil
}
