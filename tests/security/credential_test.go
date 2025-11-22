package security

import (
	"strings"
	"testing"

	"github.com/glive/infrastructure/security"
)

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestCredentialMasking(t *testing.T) {
	tests := []struct {
		name     string
		cred     string
		expected string
	}{
		{"short credential", "abc", "****"},
		{"medium credential", "sk-1234567890abcdef", "sk-1...cdef"},
		{"long credential", "sk_test_FAKE1234567890abcdefghijklmnopqrstuvwxyz", "sk_t...wxyz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			masked := security.MaskCredential(tt.cred)
			if masked == tt.cred {
				t.Errorf("credential was not masked")
			}
			if len(masked) < 4 {
				t.Errorf("masked credential too short: %q", masked)
			}
		})
	}
}

func TestCredentialSanitization(t *testing.T) {
	credential := "sk-1234567890abcdef"
	logMessage := "API key is sk-1234567890abcdef"

	sanitized := security.SanitizeForLogging(logMessage, credential)

	if contains(sanitized, credential) {
		t.Error("credential should be removed from log message")
	}

	if !contains(sanitized, "[REDACTED]") {
		t.Error("log message should contain [REDACTED]")
	}
}

func TestCredentialStore(t *testing.T) {
	// Note: This test requires OS keyring or encrypted file store
	// May skip on systems without keyring support
	tempDir := t.TempDir()

	store, err := security.NewSecureCredentialStore("glive-test", tempDir)
	if err != nil {
		t.Skipf("credential store not available: %v", err)
	}

	key := "test_key"
	value := "test_value_12345"

	// Test Set
	err = store.Set(key, value)
	if err != nil {
		t.Fatalf("failed to set credential: %v", err)
	}

	// Test Get
	retrieved, err := store.Get(key)
	if err != nil {
		t.Fatalf("failed to get credential: %v", err)
	}

	if retrieved != value {
		t.Errorf("expected credential %q, got %q", value, retrieved)
	}

	// Test Delete
	err = store.Delete(key)
	if err != nil {
		t.Fatalf("failed to delete credential: %v", err)
	}

	// Verify deleted
	_, err = store.Get(key)
	if err == nil {
		t.Error("expected error when getting deleted credential")
	}
}
