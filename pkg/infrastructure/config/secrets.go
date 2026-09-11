package config

import (
	"fmt"

	"github.com/glive/infrastructure/security"
)

// SecretStore provides an interface for storing secrets
type SecretStore interface {
	Get(key string) (string, error)
	Set(key string, value string) error
	Delete(key string) error
}

// ConfigSecretStore wraps the security credential store for configuration secrets
type ConfigSecretStore struct {
	store *security.SecureCredentialStore
}

// NewConfigSecretStore creates a new configuration secret store
func NewConfigSecretStore() (*ConfigSecretStore, error) {
	store, err := security.NewSecureCredentialStore("glive", "")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize secret store: %w", err)
	}

	return &ConfigSecretStore{
		store: store,
	}, nil
}

// Get retrieves a secret
func (s *ConfigSecretStore) Get(key string) (string, error) {
	return s.store.Get(key)
}

// Set stores a secret
func (s *ConfigSecretStore) Set(key string, value string) error {
	return s.store.Set(key, value)
}

// Delete removes a secret
func (s *ConfigSecretStore) Delete(key string) error {
	return s.store.Delete(key)
}

// SecretKeys defines the keys used for secrets
const (
	SecretKeyAPIKey = "api_key"
)
