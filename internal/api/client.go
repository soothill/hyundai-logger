// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/soothill/hyundai-logger/internal/retry"
	"golang.org/x/time/rate"
)

// Client represents a Hyundai Bluelink API client
type Client struct {
	httpClient   *http.Client
	baseURL      string
	username     string
	password     string
	pin          string
	brand        string
	region       string
	accessToken  string
	refreshToken string
	vehicleID    string
	rateLimiter  *rate.Limiter
	retrier      *retry.Retrier
}

// NewClient creates a new Hyundai API client
func NewClient(username, password, pin, brand, region string, requestsPerHour int, retryConfig retry.Config) *Client {
	// Calculate rate limit: requestsPerHour requests per hour
	// Convert to requests per second
	rps := float64(requestsPerHour) / 3600.0
	limiter := rate.NewLimiter(rate.Limit(rps), 1) // burst of 1

	baseURL := getBaseURL(region, brand)

	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL:     baseURL,
		username:    username,
		password:    password,
		pin:         pin,
		brand:       brand,
		region:      region,
		rateLimiter: limiter,
		retrier:     retry.New(retryConfig),
	}
}

// getBaseURL returns the appropriate base URL for the region and brand
func getBaseURL(region, brand string) string {
	// Based on reverse engineered endpoints
	urls := map[string]map[string]string{
		"US": {
			"hyundai": "https://api.telematics.hyundaiusa.com",
			"kia":     "https://api.owners.kia.com",
		},
		"CA": {
			"hyundai": "https://api.telematics.hyundaiusa.com",
			"kia":     "https://api.owners.kia.com",
		},
		"EU": {
			"hyundai": "https://prd.eu-ccapi.hyundai.com:8080",
			"kia":     "https://prd.eu-ccapi.kia.com:8080",
		},
	}

	if regionURLs, ok := urls[region]; ok {
		if baseURL, ok := regionURLs[strings.ToLower(brand)]; ok {
			return baseURL
		}
	}

	// Default to US Hyundai
	return "https://api.telematics.hyundaiusa.com"
}

// Authenticate performs authentication with the Hyundai API with retry logic
func (c *Client) Authenticate(ctx context.Context) error {
	return c.retrier.Do(ctx, func() error {
		if err := c.rateLimiter.Wait(ctx); err != nil {
			return fmt.Errorf("rate limiter: %w", err)
		}

		// This is a simplified authentication flow
		// Real implementation would need proper OAuth flow based on region
		endpoint := fmt.Sprintf("%s/v2/login", c.baseURL)

		data := url.Values{}
		data.Set("username", c.username)
		data.Set("password", c.password)

		req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(data.Encode()))
		if err != nil {
			return fmt.Errorf("creating request: %w", err)
		}

		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("User-Agent", "HyundaiLogger/1.0")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("making request: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("reading response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("authentication failed (status %d): %s", resp.StatusCode, string(body))
		}

		var authResp AuthResponse
		if err := json.Unmarshal(body, &authResp); err != nil {
			return fmt.Errorf("parsing response: %w", err)
		}

		c.accessToken = authResp.AccessToken
		c.refreshToken = authResp.RefreshToken

		return nil
	})
}

// GetVehicles retrieves the list of vehicles associated with the account with retry logic
func (c *Client) GetVehicles(ctx context.Context) ([]Vehicle, error) {
	result, err := c.retrier.DoWithResult(ctx, func() (interface{}, error) {
		if err := c.rateLimiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limiter: %w", err)
		}

		endpoint := fmt.Sprintf("%s/v2/vehicles", c.baseURL)

		req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}

		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.accessToken))
		req.Header.Set("User-Agent", "HyundaiLogger/1.0")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("making request: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("reading response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("get vehicles failed (status %d): %s", resp.StatusCode, string(body))
		}

		var vehiclesResp VehiclesResponse
		if err := json.Unmarshal(body, &vehiclesResp); err != nil {
			return nil, fmt.Errorf("parsing response: %w", err)
		}

		return vehiclesResp.Vehicles, nil
	})

	if err != nil {
		return nil, err
	}

	return result.([]Vehicle), nil
}

// GetVehicleStatus retrieves the current status of a vehicle with retry logic
func (c *Client) GetVehicleStatus(ctx context.Context, vehicleID string) (*VehicleStatus, error) {
	result, err := c.retrier.DoWithResult(ctx, func() (interface{}, error) {
		if err := c.rateLimiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limiter: %w", err)
		}

		endpoint := fmt.Sprintf("%s/v2/vehicles/%s/status", c.baseURL, vehicleID)

		req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}

		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.accessToken))
		req.Header.Set("User-Agent", "HyundaiLogger/1.0")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("making request: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("reading response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("get vehicle status failed (status %d): %s", resp.StatusCode, string(body))
		}

		var status VehicleStatus
		if err := json.Unmarshal(body, &status); err != nil {
			return nil, fmt.Errorf("parsing response: %w", err)
		}

		return &status, nil
	})

	if err != nil {
		return nil, err
	}

	return result.(*VehicleStatus), nil
}

// GetVehicleLocation retrieves the current location of a vehicle with retry logic
func (c *Client) GetVehicleLocation(ctx context.Context, vehicleID string) (*Location, error) {
	result, err := c.retrier.DoWithResult(ctx, func() (interface{}, error) {
		if err := c.rateLimiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limiter: %w", err)
		}

		endpoint := fmt.Sprintf("%s/v2/vehicles/%s/location", c.baseURL, vehicleID)

		req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}

		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.accessToken))
		req.Header.Set("User-Agent", "HyundaiLogger/1.0")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("making request: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("reading response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("get vehicle location failed (status %d): %s", resp.StatusCode, string(body))
		}

		var location Location
		if err := json.Unmarshal(body, &location); err != nil {
			return nil, fmt.Errorf("parsing response: %w", err)
		}

		return &location, nil
	})

	if err != nil {
		return nil, err
	}

	return result.(*Location), nil
}

// GetOdometer retrieves the odometer reading
func (c *Client) GetOdometer(ctx context.Context, vehicleID string) (*Odometer, error) {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}

	endpoint := fmt.Sprintf("%s/v2/vehicles/%s/odometer", c.baseURL, vehicleID)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.accessToken))
	req.Header.Set("User-Agent", "HyundaiLogger/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get odometer failed (status %d): %s", resp.StatusCode, string(body))
	}

	var odometer Odometer
	if err := json.Unmarshal(body, &odometer); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return &odometer, nil
}
