// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package auth

import (
	"strings"
	"testing"
	"time"
)

func TestGenerateAPIKey(t *testing.T) {
	key, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("failed to generate API key: %v", err)
	}

	if key == "" {
		t.Error("expected non-empty API key")
	}

	// Key should be base64 encoded (32 bytes = 43-44 chars in base64)
	if len(key) < 40 {
		t.Errorf("API key too short: %d characters", len(key))
	}

	// Generate another key - should be different
	key2, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("failed to generate second API key: %v", err)
	}

	if key == key2 {
		t.Error("generated keys should be unique")
	}
}

func TestHashAPIKey(t *testing.T) {
	key := "test-api-key-12345"
	hash := HashAPIKey(key)

	// Hash should be 64 hex characters (SHA-256)
	if len(hash) != 64 {
		t.Errorf("expected hash length 64, got %d", len(hash))
	}

	// Same key should produce same hash
	hash2 := HashAPIKey(key)
	if hash != hash2 {
		t.Error("same key should produce same hash")
	}

	// Different key should produce different hash
	hash3 := HashAPIKey("different-key")
	if hash == hash3 {
		t.Error("different keys should produce different hashes")
	}
}

func TestCreateKey(t *testing.T) {
	manager := NewAPIKeyManager()

	scopes := []string{ScopeReadVehicles, ScopeReadData}
	apiKey, err := manager.CreateKey("test-key", "Test API Key", scopes, nil)

	if err != nil {
		t.Fatalf("failed to create API key: %v", err)
	}

	if apiKey.Name != "test-key" {
		t.Errorf("expected name 'test-key', got %s", apiKey.Name)
	}

	if apiKey.Description != "Test API Key" {
		t.Errorf("expected description 'Test API Key', got %s", apiKey.Description)
	}

	if len(apiKey.Scopes) != 2 {
		t.Errorf("expected 2 scopes, got %d", len(apiKey.Scopes))
	}

	if !apiKey.Enabled {
		t.Error("new key should be enabled")
	}

	if apiKey.Key == "" {
		t.Error("key should not be empty")
	}

	if apiKey.Hash == "" {
		t.Error("hash should not be empty")
	}

	if apiKey.ExpiresAt != nil {
		t.Error("key should not have expiration without expiresIn")
	}
}

func TestCreateKeyWithExpiration(t *testing.T) {
	manager := NewAPIKeyManager()

	expiresIn := 24 * time.Hour
	apiKey, err := manager.CreateKey("expiring-key", "Expiring Key", []string{ScopeAll}, &expiresIn)

	if err != nil {
		t.Fatalf("failed to create API key: %v", err)
	}

	if apiKey.ExpiresAt == nil {
		t.Fatal("key should have expiration")
	}

	expectedExpiry := time.Now().Add(expiresIn)
	diff := apiKey.ExpiresAt.Sub(expectedExpiry)
	if diff < 0 {
		diff = -diff
	}

	if diff > 1*time.Second {
		t.Errorf("expiration time off by %v", diff)
	}
}

func TestValidateKey(t *testing.T) {
	manager := NewAPIKeyManager()

	// Create a key
	apiKey, err := manager.CreateKey("valid-key", "Valid Key", []string{ScopeReadVehicles}, nil)
	if err != nil {
		t.Fatalf("failed to create API key: %v", err)
	}

	originalKey := apiKey.Key

	// Validate the key
	validated, err := manager.ValidateKey(originalKey)
	if err != nil {
		t.Fatalf("failed to validate key: %v", err)
	}

	if validated.Name != "valid-key" {
		t.Errorf("expected name 'valid-key', got %s", validated.Name)
	}

	if validated.LastUsedAt == nil {
		t.Error("last used time should be set after validation")
	}

	// Invalid key should fail
	_, err = manager.ValidateKey("invalid-key-12345")
	if err == nil {
		t.Error("expected error for invalid key")
	}
}

func TestValidateKeyDisabled(t *testing.T) {
	manager := NewAPIKeyManager()

	apiKey, _ := manager.CreateKey("disabled-key", "Disabled Key", []string{ScopeAll}, nil)
	originalKey := apiKey.Key

	// Revoke the key
	err := manager.RevokeKey(originalKey)
	if err != nil {
		t.Fatalf("failed to revoke key: %v", err)
	}

	// Should fail validation
	_, err = manager.ValidateKey(originalKey)
	if err == nil {
		t.Error("expected error for disabled key")
	}

	if !strings.Contains(err.Error(), "disabled") {
		t.Errorf("expected 'disabled' in error, got: %v", err)
	}
}

func TestValidateKeyExpired(t *testing.T) {
	manager := NewAPIKeyManager()

	// Create key that expires in 1 millisecond
	expiresIn := 1 * time.Millisecond
	apiKey, _ := manager.CreateKey("expiring-key", "Expiring Key", []string{ScopeAll}, &expiresIn)
	originalKey := apiKey.Key

	// Wait for expiration
	time.Sleep(10 * time.Millisecond)

	// Should fail validation
	_, err := manager.ValidateKey(originalKey)
	if err == nil {
		t.Error("expected error for expired key")
	}

	if !strings.Contains(err.Error(), "expired") {
		t.Errorf("expected 'expired' in error, got: %v", err)
	}
}

func TestRevokeKey(t *testing.T) {
	manager := NewAPIKeyManager()

	apiKey, _ := manager.CreateKey("revoke-key", "To Revoke", []string{ScopeAll}, nil)
	originalKey := apiKey.Key

	// Revoke
	err := manager.RevokeKey(originalKey)
	if err != nil {
		t.Fatalf("failed to revoke key: %v", err)
	}

	// Should be disabled
	_, err = manager.ValidateKey(originalKey)
	if err == nil {
		t.Error("revoked key should not validate")
	}

	// Revoking non-existent key should fail
	err = manager.RevokeKey("non-existent-key")
	if err == nil {
		t.Error("expected error revoking non-existent key")
	}
}

func TestDeleteKey(t *testing.T) {
	manager := NewAPIKeyManager()

	apiKey, _ := manager.CreateKey("delete-key", "To Delete", []string{ScopeAll}, nil)
	originalKey := apiKey.Key

	// Delete
	err := manager.DeleteKey(originalKey)
	if err != nil {
		t.Fatalf("failed to delete key: %v", err)
	}

	// Should not exist
	_, err = manager.ValidateKey(originalKey)
	if err == nil {
		t.Error("deleted key should not validate")
	}

	// Deleting again should fail
	err = manager.DeleteKey(originalKey)
	if err == nil {
		t.Error("expected error deleting non-existent key")
	}
}

func TestListKeys(t *testing.T) {
	manager := NewAPIKeyManager()

	// Create multiple keys
	manager.CreateKey("key1", "First Key", []string{ScopeReadVehicles}, nil)
	manager.CreateKey("key2", "Second Key", []string{ScopeReadData}, nil)
	manager.CreateKey("key3", "Third Key", []string{ScopeAll}, nil)

	keys := manager.ListKeys()

	if len(keys) != 3 {
		t.Errorf("expected 3 keys, got %d", len(keys))
	}

	// Verify keys are redacted
	for _, key := range keys {
		if key.Key != "***" {
			t.Errorf("key should be redacted, got: %s", key.Key)
		}

		if key.Name == "" {
			t.Error("name should not be empty")
		}
	}
}

func TestGetKeyByName(t *testing.T) {
	manager := NewAPIKeyManager()

	manager.CreateKey("findme", "Find Me", []string{ScopeAll}, nil)

	key, err := manager.GetKeyByName("findme")
	if err != nil {
		t.Fatalf("failed to find key: %v", err)
	}

	if key.Name != "findme" {
		t.Errorf("expected name 'findme', got %s", key.Name)
	}

	if key.Description != "Find Me" {
		t.Errorf("expected description 'Find Me', got %s", key.Description)
	}

	// Key should be redacted
	if key.Key != "***" {
		t.Error("key should be redacted in GetKeyByName")
	}

	// Non-existent key
	_, err = manager.GetKeyByName("nonexistent")
	if err == nil {
		t.Error("expected error for non-existent key")
	}
}

func TestHasScope(t *testing.T) {
	tests := []struct {
		name         string
		keyScopes    []string
		checkScope   string
		expectedHas  bool
	}{
		{"exact match", []string{ScopeReadVehicles}, ScopeReadVehicles, true},
		{"no match", []string{ScopeReadVehicles}, ScopeWriteVehicles, false},
		{"wildcard", []string{ScopeAll}, ScopeReadVehicles, true},
		{"multiple scopes match", []string{ScopeReadVehicles, ScopeReadData}, ScopeReadData, true},
		{"multiple scopes no match", []string{ScopeReadVehicles, ScopeReadData}, ScopeAdmin, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiKey := &APIKey{
				Scopes: tt.keyScopes,
			}

			hasScope := apiKey.HasScope(tt.checkScope)
			if hasScope != tt.expectedHas {
				t.Errorf("expected HasScope=%v, got %v", tt.expectedHas, hasScope)
			}
		})
	}
}

func TestIsExpired(t *testing.T) {
	// Not expired - no expiration set
	apiKey := &APIKey{}
	if apiKey.IsExpired() {
		t.Error("key with no expiration should not be expired")
	}

	// Not expired - future expiration
	future := time.Now().Add(1 * time.Hour)
	apiKey = &APIKey{ExpiresAt: &future}
	if apiKey.IsExpired() {
		t.Error("key with future expiration should not be expired")
	}

	// Expired - past expiration
	past := time.Now().Add(-1 * time.Hour)
	apiKey = &APIKey{ExpiresAt: &past}
	if !apiKey.IsExpired() {
		t.Error("key with past expiration should be expired")
	}
}

func TestTimeUntilExpiry(t *testing.T) {
	// No expiration
	apiKey := &APIKey{}
	duration := apiKey.TimeUntilExpiry()
	if duration != nil {
		t.Error("key with no expiration should return nil duration")
	}

	// Future expiration
	future := time.Now().Add(2 * time.Hour)
	apiKey = &APIKey{ExpiresAt: &future}
	duration = apiKey.TimeUntilExpiry()

	if duration == nil {
		t.Fatal("expected duration, got nil")
	}

	// Should be approximately 2 hours
	if *duration < 1*time.Hour || *duration > 3*time.Hour {
		t.Errorf("expected duration around 2h, got %v", *duration)
	}
}

func TestScopeConstants(t *testing.T) {
	// Verify scope constants are defined
	scopes := []string{
		ScopeReadVehicles,
		ScopeWriteVehicles,
		ScopeReadData,
		ScopeExportData,
		ScopeAdmin,
		ScopeAll,
	}

	for _, scope := range scopes {
		if scope == "" {
			t.Error("scope constant should not be empty")
		}
	}

	// Verify wildcard scope
	if ScopeAll != "*" {
		t.Errorf("expected ScopeAll to be '*', got %s", ScopeAll)
	}
}

func TestConcurrentOperations(t *testing.T) {
	manager := NewAPIKeyManager()

	// Create a key
	apiKey, _ := manager.CreateKey("concurrent-key", "Concurrent Test", []string{ScopeAll}, nil)
	originalKey := apiKey.Key

	// Concurrent validations
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			_, err := manager.ValidateKey(originalKey)
			if err != nil {
				t.Errorf("concurrent validation failed: %v", err)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify last used time is set
	validated, _ := manager.ValidateKey(originalKey)
	if validated.LastUsedAt == nil {
		t.Error("last used time should be set")
	}
}

// Benchmark tests
func BenchmarkGenerateAPIKey(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = GenerateAPIKey()
	}
}

func BenchmarkHashAPIKey(b *testing.B) {
	key := "test-api-key-benchmark"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = HashAPIKey(key)
	}
}

func BenchmarkValidateKey(b *testing.B) {
	manager := NewAPIKeyManager()
	apiKey, _ := manager.CreateKey("benchmark-key", "Benchmark", []string{ScopeAll}, nil)
	key := apiKey.Key

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = manager.ValidateKey(key)
	}
}
