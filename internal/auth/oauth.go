package auth

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// OAuth2Config holds the OAuth configuration for different regions
type OAuth2Config struct {
	ClientID     string
	ClientSecret string
	AuthURL      string
	TokenURL     string
	RedirectURI  string
	UserAgent    string
}

// GetRegionConfig returns OAuth2 configuration based on region
func GetRegionConfig(region, brand string) *OAuth2Config {
	configs := map[string]map[string]*OAuth2Config{
		"EU": {
			"hyundai": &OAuth2Config{
				ClientID:     "6d477c38-3ca4-4cf3-9557-2a1929a94654",
				ClientSecret: "KUy49XxPzLpLuoK0xhBC77W6VXhmtQR9iQhmIFjjoY4IpxsV",
				AuthURL:      "https://prd.eu-ccapi.hyundai.com/api/v1/user/oauth2/authorize",
				TokenURL:     "https://prd.eu-ccapi.hyundai.com/api/v1/user/oauth2/token",
				RedirectURI:  "https://prd.eu-ccapi.hyundai.com/api/v1/user/oauth2/redirect",
				UserAgent:    "Mozilla/5.0 (Linux; Android 4.1.1; Galaxy Nexus Build/JRO03C) AppleWebKit/535.19 (KHTML, like Gecko) Chrome/18.0.1025.166 Mobile Safari/535.19_CCS_APP_AOS",
			},
			"kia": &OAuth2Config{
				ClientID:     "fdc85c00-0a2f-4c64-bcb4-2cfb1500730a",
				ClientSecret: "PC6SymPRRlW9TSxEJ7g5eetRJKPPSZaOXNBQS2QmB8PH0jLBZ8",
				AuthURL:      "https://prd.eu-ccapi.kia.com/api/v1/user/oauth2/authorize",
				TokenURL:     "https://prd.eu-ccapi.kia.com/api/v1/user/oauth2/token",
				RedirectURI:  "https://prd.eu-ccapi.kia.com/api/v1/user/oauth2/redirect",
				UserAgent:    "Mozilla/5.0 (Linux; Android 4.1.1; Galaxy Nexus Build/JRO03C) AppleWebKit/535.19 (KHTML, like Gecko) Chrome/18.0.1025.166 Mobile Safari/535.19",
			},
		},
		"US": {
			"hyundai": &OAuth2Config{
				ClientID:     "64621b96-0f0d-11ec-82a8-0242ac130003",
				ClientSecret: "LJbr0bHwOVKZOYP6P2ucLmVnBubhJM8TCLYFqnR1rsa0USIKzqpCPvbIQfUp2tc8aP3A8OpqD5oVDPXtPnLMpA==",
				AuthURL:      "https://api.telematics.hyundaiusa.com/oauth2/authorize",
				TokenURL:     "https://api.telematics.hyundaiusa.com/oauth2/token",
				RedirectURI:  "https://www.getpostman.com/oauth2/callback",
				UserAgent:    "okhttp/3.14.9",
			},
		},
		"CA": {
			"hyundai": &OAuth2Config{
				ClientID:     "64621b96-0f0d-11ec-82a8-0242ac130003",
				ClientSecret: "LJbr0bHwOVKZOYP6P2ucLmVnBubhJM8TCLYFqnR1rsa0USIKzqpCPvbIQfUp2tc8aP3A8OpqD5oVDPXtPnLMpA==",
				AuthURL:      "https://api.telematics.hyundaicanada.com/oauth2/authorize",
				TokenURL:     "https://api.telematics.hyundaicanada.com/oauth2/token",
				RedirectURI:  "https://www.getpostman.com/oauth2/callback",
				UserAgent:    "okhttp/3.14.9",
			},
		},
	}

	if regionConfig, ok := configs[region]; ok {
		if brandConfig, ok := regionConfig[strings.ToLower(brand)]; ok {
			return brandConfig
		}
	}

	// Default to EU Hyundai if not found
	return configs["EU"]["hyundai"]
}

// TokenStore manages token persistence
type TokenStore struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in"`
	ExpiresAt    time.Time `json:"expires_at"`
	DeviceID     string    `json:"device_id"`
	VehicleID    string    `json:"vehicle_id"`
}

// SaveTokens saves tokens to a file
func SaveTokens(store *TokenStore) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configDir := filepath.Join(homeDir, ".hyundai-logger")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return err
	}

	tokenFile := filepath.Join(configDir, "tokens.json")
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(tokenFile, data, 0600)
}

// LoadTokens loads tokens from file
func LoadTokens() (*TokenStore, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	tokenFile := filepath.Join(homeDir, ".hyundai-logger", "tokens.json")
	data, err := os.ReadFile(tokenFile)
	if err != nil {
		return nil, err
	}

	var store TokenStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, err
	}

	return &store, nil
}

// OAuth2Client handles the OAuth2 authentication flow
type OAuth2Client struct {
	Config     *OAuth2Config
	TokenStore *TokenStore
	Region     string
	Brand      string
	Username   string
	Password   string
	PIN        string
	httpClient *http.Client
}

// NewOAuth2Client creates a new OAuth2 client
func NewOAuth2Client(region, brand, username, password, pin string) *OAuth2Client {
	return &OAuth2Client{
		Config:     GetRegionConfig(region, brand),
		Region:     region,
		Brand:      brand,
		Username:   username,
		Password:   password,
		PIN:        pin,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// AuthenticateWithRefreshToken uses a stored or provided refresh token
func (c *OAuth2Client) AuthenticateWithRefreshToken(refreshToken string) error {
	if refreshToken == "" {
		// Try to load from file
		store, err := LoadTokens()
		if err == nil && store.RefreshToken != "" {
			refreshToken = store.RefreshToken
			c.TokenStore = store
		} else {
			return fmt.Errorf("no refresh token available")
		}
	}

	// Use refresh token to get new access token
	return c.RefreshAccessToken(refreshToken)
}

// RefreshAccessToken refreshes the access token using refresh token
func (c *OAuth2Client) RefreshAccessToken(refreshToken string) error {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)
	data.Set("client_id", c.Config.ClientID)

	if c.Config.ClientSecret != "" {
		data.Set("client_secret", c.Config.ClientSecret)
	}

	req, err := http.NewRequest("POST", c.Config.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", c.Config.UserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token refresh failed: %s", string(body))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return err
	}

	// Update token store
	if c.TokenStore == nil {
		c.TokenStore = &TokenStore{}
	}

	c.TokenStore.AccessToken = tokenResp.AccessToken
	if tokenResp.RefreshToken != "" {
		c.TokenStore.RefreshToken = tokenResp.RefreshToken
	}
	c.TokenStore.TokenType = tokenResp.TokenType
	c.TokenStore.ExpiresIn = tokenResp.ExpiresIn
	c.TokenStore.ExpiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	// Generate device ID if not present
	if c.TokenStore.DeviceID == "" {
		c.TokenStore.DeviceID = c.GenerateDeviceID()
	}

	// Save tokens
	return SaveTokens(c.TokenStore)
}

// IsTokenValid checks if the current token is still valid
func (c *OAuth2Client) IsTokenValid() bool {
	if c.TokenStore == nil {
		return false
	}
	return time.Now().Before(c.TokenStore.ExpiresAt.Add(-5 * time.Minute)) // 5 min buffer
}

// EnsureValidToken ensures we have a valid access token
func (c *OAuth2Client) EnsureValidToken() error {
	if !c.IsTokenValid() {
		if c.TokenStore != nil && c.TokenStore.RefreshToken != "" {
			return c.RefreshAccessToken(c.TokenStore.RefreshToken)
		}
		return fmt.Errorf("no valid token and no refresh token available")
	}
	return nil
}

// GetAccessToken returns the current access token
func (c *OAuth2Client) GetAccessToken() string {
	if c.TokenStore != nil {
		return c.TokenStore.AccessToken
	}
	return ""
}

// GetDeviceID returns the stored device ID
func (c *OAuth2Client) GetDeviceID() string {
	if c.TokenStore != nil {
		return c.TokenStore.DeviceID
	}
	return ""
}

// SetDeviceID sets and saves the device ID
func (c *OAuth2Client) SetDeviceID(deviceID string) error {
	if c.TokenStore == nil {
		c.TokenStore = &TokenStore{}
	}
	c.TokenStore.DeviceID = deviceID
	return SaveTokens(c.TokenStore)
}

// GenerateDeviceID generates a new device ID if needed
func (c *OAuth2Client) GenerateDeviceID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback to a deterministic device ID if random fails
		return fmt.Sprintf("fallback-device-id-%d", time.Now().Unix())
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
