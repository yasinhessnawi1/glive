package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/crypto/pbkdf2"
)

const (
	// KeyDerivationIterations is the number of PBKDF2 iterations
	KeyDerivationIterations = 100000
	// SaltSize is the size of the salt in bytes
	SaltSize = 32
	// NonceSize is the size of the nonce for GCM
	NonceSize = 12
)

var (
	ErrCredentialNotFound = fmt.Errorf("credential not found")
	ErrInvalidKeyring     = fmt.Errorf("invalid keyring operation")
)

// SecureCredentialStore provides secure storage for credentials
// Uses OS keyring when available, falls back to encrypted file storage
type SecureCredentialStore struct {
	fallback    *EncryptedFileStore
	serviceName string
	mu          sync.RWMutex
}

// NewSecureCredentialStore creates a new secure credential store
// On Windows: Uses Windows Credential Manager
// On macOS: Uses Keychain
// On Linux: Uses Secret Service API
// Falls back to encrypted file storage if keyring unavailable
func NewSecureCredentialStore(serviceName string, configDir string) (*SecureCredentialStore, error) {
	if serviceName == "" {
		serviceName = "glive"
	}

	// Try to use OS keyring first
	// For now, we'll use encrypted file fallback
	// TODO: Integrate with github.com/99designs/keyring for cross-platform keyring support

	fallback, err := NewEncryptedFileStore(serviceName, configDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize credential store: %w", err)
	}

	return &SecureCredentialStore{
		fallback:    fallback,
		serviceName: serviceName,
	}, nil
}

// Set stores a credential securely
func (s *SecureCredentialStore) Set(key string, value string) error {
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}
	if value == "" {
		return fmt.Errorf("value cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// For now, use encrypted file storage
	// TODO: Try OS keyring first, fallback to encrypted file
	return s.fallback.Set(key, value)
}

// Get retrieves a credential securely
func (s *SecureCredentialStore) Get(key string) (string, error) {
	if key == "" {
		return "", fmt.Errorf("key cannot be empty")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	// For now, use encrypted file storage
	// TODO: Try OS keyring first, fallback to encrypted file
	return s.fallback.Get(key)
}

// Delete removes a credential
func (s *SecureCredentialStore) Delete(key string) error {
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return s.fallback.Delete(key)
}

// MaskCredential returns a masked version of a credential for display
func MaskCredential(value string) string {
	if len(value) <= 8 {
		return "****"
	}
	if len(value) <= 16 {
		return value[:2] + "..." + value[len(value)-2:]
	}
	return value[:4] + "..." + value[len(value)-4:]
}

// SanitizeForLogging removes credentials from strings
func SanitizeForLogging(s string, credentials ...string) string {
	result := s
	for _, cred := range credentials {
		if cred != "" && len(cred) > 4 {
			// Replace all occurrences
			result = strings.ReplaceAll(result, cred, "[REDACTED]")
			// Also replace masked versions
			masked := MaskCredential(cred)
			if masked != cred {
				result = strings.ReplaceAll(result, masked, "[REDACTED]")
			}
		}
	}
	return result
}

// EncryptedFileStore provides encrypted file-based credential storage
type EncryptedFileStore struct {
	storeDir string
	key      []byte
	mu       sync.RWMutex
}

// NewEncryptedFileStore creates a new encrypted file store
func NewEncryptedFileStore(serviceName string, configDir string) (*EncryptedFileStore, error) {
	if configDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		configDir = filepath.Join(homeDir, ".glive")
	}

	storeDir := filepath.Join(configDir, "credentials")
	if err := os.MkdirAll(storeDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create credential store directory: %w", err)
	}

	// Derive encryption key from machine-specific data
	key, err := deriveKey(serviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to derive encryption key: %w", err)
	}

	return &EncryptedFileStore{
		storeDir: storeDir,
		key:      key,
	}, nil
}

// Set stores an encrypted credential
func (e *EncryptedFileStore) Set(key string, value string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Encrypt the value
	encrypted, err := e.encrypt([]byte(value))
	if err != nil {
		return fmt.Errorf("failed to encrypt credential: %w", err)
	}

	// Store in file
	filePath := filepath.Join(e.storeDir, sanitizeKey(key)+".enc")
	if err := os.WriteFile(filePath, encrypted, 0600); err != nil {
		return fmt.Errorf("failed to write credential file: %w", err)
	}

	return nil
}

// Get retrieves and decrypts a credential
func (e *EncryptedFileStore) Get(key string) (string, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	filePath := filepath.Join(e.storeDir, sanitizeKey(key)+".enc")
	encrypted, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrCredentialNotFound
		}
		return "", fmt.Errorf("failed to read credential file: %w", err)
	}

	// Decrypt the value
	decrypted, err := e.decrypt(encrypted)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt credential: %w", err)
	}

	return string(decrypted), nil
}

// Delete removes a credential
func (e *EncryptedFileStore) Delete(key string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	filePath := filepath.Join(e.storeDir, sanitizeKey(key)+".enc")
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete credential file: %w", err)
	}

	return nil
}

// encrypt encrypts data using AES-256-GCM
func (e *EncryptedFileStore) encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, NonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// decrypt decrypts data using AES-256-GCM
func (e *EncryptedFileStore) decrypt(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < NonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := ciphertext[:NonceSize]
	ciphertext = ciphertext[NonceSize:]

	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// deriveKey derives an encryption key from machine-specific data
func deriveKey(serviceName string) ([]byte, error) {
	// Use machine-specific data for key derivation
	// In production, this should use a more secure method
	hostname, _ := os.Hostname()
	userHome, _ := os.UserHomeDir()

	salt := sha256.Sum256([]byte(serviceName + hostname + userHome))
	password := serviceName + hostname + userHome

	// Derive key using PBKDF2
	key := pbkdf2.Key([]byte(password), salt[:], KeyDerivationIterations, 32, sha256.New)
	return key, nil
}

// sanitizeKey sanitizes a key for use as a filename
func sanitizeKey(key string) string {
	// Replace invalid filename characters
	key = strings.ReplaceAll(key, "/", "_")
	key = strings.ReplaceAll(key, "\\", "_")
	key = strings.ReplaceAll(key, ":", "_")
	key = strings.ReplaceAll(key, "*", "_")
	key = strings.ReplaceAll(key, "?", "_")
	key = strings.ReplaceAll(key, "\"", "_")
	key = strings.ReplaceAll(key, "<", "_")
	key = strings.ReplaceAll(key, ">", "_")
	key = strings.ReplaceAll(key, "|", "_")
	return base64.URLEncoding.EncodeToString([]byte(key))
}

