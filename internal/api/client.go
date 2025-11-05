package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/soothill/hyundai-logger/internal/auth"
)

// Client represents the Hyundai/Kia API client
type Client struct {
	Region       string
	Brand        string
	httpClient   *http.Client
	authClient   *auth.OAuth2Client
	stampManager *auth.StampManager
	baseURL      string
}

// NewClient creates a new API client
func NewClient(region, brand, username, password, pin, refreshToken string) (*Client, error) {
	authClient := auth.NewOAuth2Client(region, brand, username, password, pin)

	// Try to authenticate with refresh token first
	if refreshToken != "" {
		if err := authClient.AuthenticateWithRefreshToken(refreshToken); err != nil {
			return nil, fmt.Errorf("failed to authenticate with refresh token: %w", err)
		}
	} else {
		// Try to load existing tokens
		if err := authClient.AuthenticateWithRefreshToken(""); err != nil {
			return nil, fmt.Errorf("no valid authentication available. Please run the token fetcher script first")
		}
	}

	// Determine base URL based on region and brand
	baseURL := getBaseURL(region, brand)

	client := &Client{
		Region:     region,
		Brand:      brand,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		authClient: authClient,
		baseURL:    baseURL,
	}

	// Initialize stamp manager for EU regions
	if region == "EU" {
		client.stampManager = auth.NewStampManager(brand)
	}

	return client, nil
}

// getBaseURL returns the API base URL for the given region and brand
func getBaseURL(region, brand string) string {
	urls := map[string]map[string]string{
		"EU": {
			"hyundai": "https://prd.eu-ccapi.hyundai.com:8080",
			"kia":     "https://prd.eu-ccapi.kia.com:8080",
		},
		"US": {
			"hyundai": "https://api.telematics.hyundaiusa.com",
			"kia":     "https://api.owners.kia.com",
		},
		"CA": {
			"hyundai": "https://api.telematics.hyundaicanada.com",
			"kia":     "https://api.owners.kia.ca",
		},
	}

	if regionURLs, ok := urls[region]; ok {
		if url, ok := regionURLs[brand]; ok {
			return url
		}
	}

	// Default to EU Hyundai
	return "https://prd.eu-ccapi.hyundai.com:8080"
}

// doRequest performs an authenticated HTTP request
func (c *Client) doRequest(method, endpoint string, body interface{}) ([]byte, error) {
	return c.doRequestWithRetry(method, endpoint, body, 0)
}

// doRequestWithRetry performs an authenticated HTTP request with retry tracking
func (c *Client) doRequestWithRetry(method, endpoint string, body interface{}, retryCount int) ([]byte, error) {
	const maxRetries = 1 // Only retry once to prevent infinite recursion

	// Ensure we have a valid token
	if err := c.authClient.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	url := c.baseURL + endpoint
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}

	// Set common headers
	req.Header.Set("Authorization", "Bearer "+c.authClient.GetAccessToken())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", c.authClient.Config.UserAgent)

	// Add device ID if available
	if deviceID := c.authClient.GetDeviceID(); deviceID != "" {
		req.Header.Set("deviceId", deviceID)
		req.Header.Set("Device-Id", deviceID)
	}

	// Sign request with stamp for EU regions
	if c.stampManager != nil {
		if err := c.stampManager.SignRequest(req); err != nil {
			// Log warning but continue - stamp might not be required for all endpoints
			fmt.Printf("Warning: Failed to sign request with stamp: %v\n", err)
		}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Check for successful response
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return respBody, nil
	}

	// Handle token expiration
	if resp.StatusCode == 401 {
		// Check if we've already retried
		if retryCount >= maxRetries {
			return nil, fmt.Errorf("authentication failed after %d retries: %s", maxRetries, string(respBody))
		}

		// Try to refresh token
		if err := c.authClient.RefreshAccessToken(c.authClient.TokenStore.RefreshToken); err != nil {
			return nil, fmt.Errorf("token refresh failed: %w", err)
		}

		// Retry the request once with incremented counter
		return c.doRequestWithRetry(method, endpoint, body, retryCount+1)
	}

	return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(respBody))
}

// GetVehicles retrieves the list of vehicles
func (c *Client) GetVehicles() (*VehiclesResponse, error) {
	endpoint := "/api/v1/spa/vehicles"
	if c.Region == "US" || c.Region == "CA" {
		endpoint = "/v2/ac/v2/enrollment/details/" + c.authClient.Username
	}

	data, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var response VehiclesResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	// Store the first vehicle's ID if not already set
	if len(response.Vehicles) > 0 && c.authClient.TokenStore != nil {
		if c.authClient.TokenStore.VehicleID == "" {
			c.authClient.TokenStore.VehicleID = response.Vehicles[0].VehicleID
			_ = auth.SaveTokens(c.authClient.TokenStore) // Ignore error - token saving is best-effort
		}
	}

	return &response, nil
}

// GetVehicleStatus retrieves the status of a specific vehicle
func (c *Client) GetVehicleStatus(vehicleID string) (*VehicleStatus, error) {
	endpoint := fmt.Sprintf("/api/v2/spa/vehicles/%s/status", vehicleID)
	if c.Region == "US" || c.Region == "CA" {
		endpoint = fmt.Sprintf("/v2/ac/v2/rcs/rvs/vehicleStatus/%s", vehicleID)
	}

	// Include device ID in request body for some regions
	requestBody := map[string]interface{}{}
	if deviceID := c.authClient.GetDeviceID(); deviceID != "" {
		requestBody["deviceId"] = deviceID
	}

	data, err := c.doRequest("GET", endpoint, requestBody)
	if err != nil {
		return nil, err
	}

	var status VehicleStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return nil, err
	}

	return &status, nil
}

// RefreshVehicleStatus forces a refresh of vehicle status from the car
func (c *Client) RefreshVehicleStatus(vehicleID string) error {
	endpoint := fmt.Sprintf("/api/v2/spa/vehicles/%s/status/refresh", vehicleID)
	if c.Region == "US" || c.Region == "CA" {
		endpoint = fmt.Sprintf("/v2/ac/v2/rcs/rvs/vehicleStatus/%s/refresh", vehicleID)
	}

	requestBody := map[string]interface{}{}
	if deviceID := c.authClient.GetDeviceID(); deviceID != "" {
		requestBody["deviceId"] = deviceID
	}

	_, err := c.doRequest("POST", endpoint, requestBody)
	return err
}

// GetLocation retrieves the vehicle's current location
func (c *Client) GetLocation(vehicleID string) (*Location, error) {
	endpoint := fmt.Sprintf("/api/v2/spa/vehicles/%s/location", vehicleID)
	if c.Region == "US" || c.Region == "CA" {
		endpoint = fmt.Sprintf("/v2/ac/v2/rcs/rvs/location/%s", vehicleID)
	}

	data, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var location Location
	if err := json.Unmarshal(data, &location); err != nil {
		return nil, err
	}

	return &location, nil
}

// StartClimate starts the vehicle's climate control
func (c *Client) StartClimate(vehicleID string, targetTemp float64) error {
	endpoint := fmt.Sprintf("/api/v1/spa/vehicles/%s/control/climate", vehicleID)

	requestBody := map[string]interface{}{
		"action":      "start",
		"hvacType":    1,
		"temperature": targetTemp,
	}

	if deviceID := c.authClient.GetDeviceID(); deviceID != "" {
		requestBody["deviceId"] = deviceID
	}

	_, err := c.doRequest("POST", endpoint, requestBody)
	return err
}

// StopClimate stops the vehicle's climate control
func (c *Client) StopClimate(vehicleID string) error {
	endpoint := fmt.Sprintf("/api/v1/spa/vehicles/%s/control/climate", vehicleID)

	requestBody := map[string]interface{}{
		"action": "stop",
	}

	if deviceID := c.authClient.GetDeviceID(); deviceID != "" {
		requestBody["deviceId"] = deviceID
	}

	_, err := c.doRequest("POST", endpoint, requestBody)
	return err
}

// Lock locks the vehicle
func (c *Client) Lock(vehicleID string) error {
	endpoint := fmt.Sprintf("/api/v1/spa/vehicles/%s/control/door", vehicleID)

	requestBody := map[string]interface{}{
		"action": "close",
		"pin":    c.authClient.PIN,
	}

	if deviceID := c.authClient.GetDeviceID(); deviceID != "" {
		requestBody["deviceId"] = deviceID
	}

	_, err := c.doRequest("POST", endpoint, requestBody)
	return err
}

// Unlock unlocks the vehicle
func (c *Client) Unlock(vehicleID string) error {
	endpoint := fmt.Sprintf("/api/v1/spa/vehicles/%s/control/door", vehicleID)

	requestBody := map[string]interface{}{
		"action": "open",
		"pin":    c.authClient.PIN,
	}

	if deviceID := c.authClient.GetDeviceID(); deviceID != "" {
		requestBody["deviceId"] = deviceID
	}

	_, err := c.doRequest("POST", endpoint, requestBody)
	return err
}

// StartCharge starts EV charging
func (c *Client) StartCharge(vehicleID string) error {
	endpoint := fmt.Sprintf("/api/v1/spa/vehicles/%s/control/charge", vehicleID)

	requestBody := map[string]interface{}{
		"action": "start",
	}

	if deviceID := c.authClient.GetDeviceID(); deviceID != "" {
		requestBody["deviceId"] = deviceID
	}

	_, err := c.doRequest("POST", endpoint, requestBody)
	return err
}

// StopCharge stops EV charging
func (c *Client) StopCharge(vehicleID string) error {
	endpoint := fmt.Sprintf("/api/v1/spa/vehicles/%s/control/charge", vehicleID)

	requestBody := map[string]interface{}{
		"action": "stop",
	}

	if deviceID := c.authClient.GetDeviceID(); deviceID != "" {
		requestBody["deviceId"] = deviceID
	}

	_, err := c.doRequest("POST", endpoint, requestBody)
	return err
}
