// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// APIKey represents an API key with metadata
type APIKey struct {
	Key         string
	Hash        string
	Name        string
	Description string
	Scopes      []string
	CreatedAt   time.Time
	ExpiresAt   *time.Time
	LastUsedAt  *time.Time
	Enabled     bool
}

// APIKeyManager manages API keys for authentication
type APIKeyManager struct {
	mu   sync.RWMutex
	keys map[string]*APIKey // Map of hash -> APIKey
}

// NewAPIKeyManager creates a new API key manager
func NewAPIKeyManager() *APIKeyManager {
	return &APIKeyManager{
		keys: make(map[string]*APIKey),
	}
}

// GenerateAPIKey generates a new cryptographically secure API key
func GenerateAPIKey() (string, error) {
	// Generate 32 random bytes
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generating random bytes: %w", err)
	}

	// Encode to base64
	key := base64.URLEncoding.EncodeToString(bytes)
	return key, nil
}

// HashAPIKey creates a SHA-256 hash of an API key
func HashAPIKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

// CreateKey creates a new API key
func (m *APIKeyManager) CreateKey(name, description string, scopes []string, expiresIn *time.Duration) (*APIKey, error) {
	// Generate key
	key, err := GenerateAPIKey()
	if err != nil {
		return nil, err
	}

	// Hash key
	hash := HashAPIKey(key)

	// Create API key object
	apiKey := &APIKey{
		Key:         key, // Returned only once
		Hash:        hash,
		Name:        name,
		Description: description,
		Scopes:      scopes,
		CreatedAt:   time.Now(),
		Enabled:     true,
	}

	// Set expiration if provided
	if expiresIn != nil {
		expiresAt := time.Now().Add(*expiresIn)
		apiKey.ExpiresAt = &expiresAt
	}

	// Store key
	m.mu.Lock()
	m.keys[hash] = apiKey
	m.mu.Unlock()

	return apiKey, nil
}

// ValidateKey validates an API key and returns the associated key info
func (m *APIKeyManager) ValidateKey(key string) (*APIKey, error) {
	hash := HashAPIKey(key)

	m.mu.RLock()
	apiKey, exists := m.keys[hash]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("invalid API key")
	}

	// Check if enabled
	if !apiKey.Enabled {
		return nil, fmt.Errorf("API key disabled")
	}

	// Check expiration
	if apiKey.ExpiresAt != nil && time.Now().After(*apiKey.ExpiresAt) {
		return nil, fmt.Errorf("API key expired")
	}

	// Update last used time
	m.mu.Lock()
	now := time.Now()
	apiKey.LastUsedAt = &now
	m.mu.Unlock()

	return apiKey, nil
}

// RevokeKey revokes an API key (disables it)
func (m *APIKeyManager) RevokeKey(key string) error {
	hash := HashAPIKey(key)

	m.mu.Lock()
	defer m.mu.Unlock()

	apiKey, exists := m.keys[hash]
	if !exists {
		return fmt.Errorf("API key not found")
	}

	apiKey.Enabled = false
	return nil
}

// DeleteKey permanently deletes an API key
func (m *APIKeyManager) DeleteKey(key string) error {
	hash := HashAPIKey(key)

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.keys[hash]; !exists {
		return fmt.Errorf("API key not found")
	}

	delete(m.keys, hash)
	return nil
}

// ListKeys returns all API keys (without the actual key values)
func (m *APIKeyManager) ListKeys() []*APIKey {
	m.mu.RLock()
	defer m.mu.RUnlock()

	keys := make([]*APIKey, 0, len(m.keys))
	for _, apiKey := range m.keys {
		// Create a copy without the actual key
		keyCopy := *apiKey
		keyCopy.Key = "***" // Redacted
		keys = append(keys, &keyCopy)
	}

	return keys
}

// GetKeyByName retrieves a key by its name
func (m *APIKeyManager) GetKeyByName(name string) (*APIKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, apiKey := range m.keys {
		if apiKey.Name == name {
			// Return a copy without the actual key
			keyCopy := *apiKey
			keyCopy.Key = "***"
			return &keyCopy, nil
		}
	}

	return nil, fmt.Errorf("API key not found: %s", name)
}

// HasScope checks if an API key has a specific scope
func (k *APIKey) HasScope(scope string) bool {
	for _, s := range k.Scopes {
		if s == scope || s == "*" {
			return true
		}
	}
	return false
}

// IsExpired checks if the API key is expired
func (k *APIKey) IsExpired() bool {
	if k.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*k.ExpiresAt)
}

// TimeUntilExpiry returns the duration until expiry
func (k *APIKey) TimeUntilExpiry() *time.Duration {
	if k.ExpiresAt == nil {
		return nil
	}

	duration := time.Until(*k.ExpiresAt)
	return &duration
}

// Scopes for API keys
const (
	ScopeReadVehicles  = "vehicles:read"
	ScopeWriteVehicles = "vehicles:write"
	ScopeReadData      = "data:read"
	ScopeExportData    = "data:export"
	ScopeAdmin         = "admin"
	ScopeAll           = "*"
)
