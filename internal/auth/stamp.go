package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Stamp represents a cryptographic stamp for API requests (EU region)
type Stamp struct {
	AppID     string    `json:"appId"`
	Stamp     string    `json:"stamp"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Frequency int       `json:"frequency"`
}

// StampManager handles stamp generation and caching for EU regions
type StampManager struct {
	stamps     map[string]*Stamp
	httpClient *http.Client
	brand      string
	appID      string
}

// NewStampManager creates a new stamp manager
func NewStampManager(brand string) *StampManager {
	// App IDs for different brands
	appIDs := map[string]string{
		"hyundai": "99cfff84-f4e2-4be8-a5ed-e5b755eb6581",
		"kia":     "693a33fa-c117-43f2-ae3b-61a02d24f417",
	}

	appID := appIDs[brand]
	if appID == "" {
		appID = appIDs["hyundai"] // default
	}

	return &StampManager{
		stamps:     make(map[string]*Stamp),
		httpClient: &http.Client{Timeout: 10 * time.Second},
		brand:      brand,
		appID:      appID,
	}
}

// GetStamp retrieves a valid stamp for API requests
func (sm *StampManager) GetStamp() (*Stamp, error) {
	// Check if we have a valid cached stamp
	if stamp, ok := sm.stamps[sm.appID]; ok {
		if time.Now().Before(stamp.ExpiresAt) {
			return stamp, nil
		}
	}

	// Fetch new stamp from remote source
	return sm.fetchRemoteStamp()
}

// fetchRemoteStamp fetches a stamp from the remote stamp service
func (sm *StampManager) fetchRemoteStamp() (*Stamp, error) {
	// Using the community-maintained stamp service
	stampURL := fmt.Sprintf("https://raw.githubusercontent.com/neoPix/bluelinky-stamps/master/%s-%s.v2.json",
		sm.brand, sm.appID)

	resp, err := sm.httpClient.Get(stampURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch stamp: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Try alternative stamp source
		alternativeURL := fmt.Sprintf("https://raw.githubusercontent.com/Hacksore/bluelinky-stamps/master/%s.json",
			sm.brand)

		resp, err = sm.httpClient.Get(alternativeURL)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch stamp from alternative source: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("stamp service returned status %d", resp.StatusCode)
		}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var stampData []struct {
		AppID     string `json:"appId"`
		Stamp     string `json:"stamp"`
		Generated string `json:"generated"`
		Frequency int    `json:"frequency"`
	}

	if err := json.Unmarshal(body, &stampData); err != nil {
		// Try parsing as single stamp
		var singleStamp struct {
			AppID     string `json:"appId"`
			Stamp     string `json:"stamp"`
			Generated string `json:"generated"`
			Frequency int    `json:"frequency"`
		}

		if err := json.Unmarshal(body, &singleStamp); err != nil {
			return nil, fmt.Errorf("failed to parse stamp data: %w", err)
		}

		stampData = append(stampData, singleStamp)
	}

	// Find the most recent valid stamp
	var latestStamp *Stamp
	for _, s := range stampData {
		if s.AppID != sm.appID && s.AppID != "" {
			continue
		}

		generatedTime, err := time.Parse("2006-01-02 15:04:05", s.Generated)
		if err != nil {
			// Try alternative format
			generatedTime, err = time.Parse(time.RFC3339, s.Generated)
			if err != nil {
				continue
			}
		}

		stamp := &Stamp{
			AppID:     s.AppID,
			Stamp:     s.Stamp,
			CreatedAt: generatedTime,
			ExpiresAt: generatedTime.Add(7 * 24 * time.Hour), // Stamps valid for 7 days
			Frequency: s.Frequency,
		}

		if latestStamp == nil || stamp.CreatedAt.After(latestStamp.CreatedAt) {
			latestStamp = stamp
		}
	}

	if latestStamp == nil {
		return nil, fmt.Errorf("no valid stamp found for app ID %s", sm.appID)
	}

	// Cache the stamp
	sm.stamps[sm.appID] = latestStamp

	return latestStamp, nil
}

// GenerateLocalStamp attempts to generate a stamp locally (experimental)
func (sm *StampManager) GenerateLocalStamp(data string) string {
	// This is a simplified version - the actual algorithm is more complex
	// and involves specific cryptographic operations that are reverse-engineered
	secret := fmt.Sprintf("%s:%s:%d", sm.brand, sm.appID, time.Now().Unix())
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// SignRequest adds stamp headers to a request (for EU region)
func (sm *StampManager) SignRequest(req *http.Request) error {
	stamp, err := sm.GetStamp()
	if err != nil {
		// For non-EU regions or if stamp fails, just continue without stamp
		// as stamps are only required for EU
		return nil
	}

	// Add stamp headers
	req.Header.Set("Stamp", stamp.Stamp)
	req.Header.Set("AppId", stamp.AppID)

	// Generate a unique request ID
	req.Header.Set("RequestId", generateRequestID())

	return nil
}

// generateRequestID creates a unique request ID
func generateRequestID() string {
	return fmt.Sprintf("%d-%d", time.Now().Unix(), time.Now().Nanosecond())
}
