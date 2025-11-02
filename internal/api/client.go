// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/soothill/hyundai-logger/internal/cache"
	"github.com/soothill/hyundai-logger/internal/circuitbreaker"
	"github.com/soothill/hyundai-logger/internal/retry"
	"golang.org/x/time/rate"
)

const (
	// apiCallTimeout is the default timeout for individual API calls
	apiCallTimeout = 25 * time.Second
)

// Client represents a Hyundai Bluelink API client
type Client struct {
	httpClient     *http.Client
	baseURL        string
	username       string
	password       string
	pin            string
	brand          string
	region         string
	accessToken    string
	refreshToken   string
	rateLimiter    *rate.Limiter
	retrier        *retry.Retrier
	circuitBreaker *circuitbreaker.CircuitBreaker
	cache          *cache.Cache
	cacheEnabled   bool
}

// NewClient creates a new Hyundai API client
func NewClient(username, password, pin, brand, region string, requestsPerHour int, retryConfig retry.Config) *Client {
	// Calculate rate limit: requestsPerHour requests per hour
	// Convert to requests per second
	rps := float64(requestsPerHour) / 3600.0
	limiter := rate.NewLimiter(rate.Limit(rps), 1) // burst of 1

	baseURL := getBaseURL(region, brand)

	// Configure HTTP transport for connection pooling and reuse
	transport := &http.Transport{
		MaxIdleConns:          10,               // Total idle connections
		MaxIdleConnsPerHost:   5,                // Idle connections per host
		MaxConnsPerHost:       10,               // Max connections per host
		IdleConnTimeout:       90 * time.Second, // Keep connections alive
		DisableKeepAlives:     false,            // Enable keep-alive
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	// Create circuit breaker with custom config
	cbConfig := circuitbreaker.Config{
		MaxFailures:         3,                // Open after 3 consecutive failures
		Timeout:             30 * time.Second, // Wait 30s before trying HalfOpen
		HalfOpenMaxRequests: 1,                // Only 1 test request in HalfOpen
	}
	cb := circuitbreaker.New(cbConfig)

	// Create cache with 60-second TTL (reduces API calls by 20-40%)
	responseCache := cache.New(60 * time.Second)

	return &Client{
		httpClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
		baseURL:        baseURL,
		username:       username,
		password:       password,
		pin:            pin,
		brand:          brand,
		region:         region,
		rateLimiter:    limiter,
		retrier:        retry.New(retryConfig),
		circuitBreaker: cb,
		cache:          responseCache,
		cacheEnabled:   true, // Enable caching by default
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

// doRequest performs an HTTP request with common headers and error handling
// Does NOT include sensitive data in error messages
// Accepts []byte body so it can be reused on retries (fixes retry bug)
// Wrapped with circuit breaker to prevent cascading failures
func (c *Client) doRequest(ctx context.Context, method, endpoint string, body []byte) ([]byte, error) {
	var result []byte

	// Execute through circuit breaker
	err := c.circuitBreaker.Execute(func() error {
		if err := c.rateLimiter.Wait(ctx); err != nil {
			return fmt.Errorf("rate limiter: %w", err)
		}

		var bodyReader io.Reader
		if body != nil {
			bodyReader = bytes.NewReader(body)
		}

		req, err := http.NewRequestWithContext(ctx, method, endpoint, bodyReader)
		if err != nil {
			return fmt.Errorf("creating request: %w", err)
		}

		req.Header.Set("User-Agent", "HyundaiLogger/1.0")
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		if c.accessToken != "" {
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.accessToken))
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("making request to %s: %w", endpoint, err)
		}
		defer resp.Body.Close()

		// Check for rate limiting before reading body
		if err := checkRateLimit(resp); err != nil {
			// For rate limit errors, respect the Retry-After delay
			if rateLimitErr, ok := err.(*RateLimitError); ok {
				// Log rate limit information
				limit, remaining, reset := getRateLimitHeaders(resp)
				fmt.Printf("Rate limit hit - Limit: %s, Remaining: %s, Reset: %s, Retry after: %s\n",
					limit, remaining, reset, rateLimitErr.RetryAfter)

				// Sleep for the specified retry-after duration (capped at 5 minutes for safety)
				sleepDuration := rateLimitErr.RetryAfter
				if sleepDuration > 5*time.Minute {
					sleepDuration = 5 * time.Minute
				}

				select {
				case <-time.After(sleepDuration):
					// Continue after sleep
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			return err
		}

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("reading response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			// Include endpoint for better debugging but not response body
			return fmt.Errorf("request failed: %s %s returned status %d", method, endpoint, resp.StatusCode)
		}

		result = respBody
		return nil
	})

	return result, err
}

// doRequestWithRetry wraps doRequest with retry logic
// Now accepts []byte instead of io.Reader so body can be reused on retries
func (c *Client) doRequestWithRetry(ctx context.Context, method, endpoint string, body []byte) ([]byte, error) {
	result, err := c.retrier.DoWithResult(ctx, func() (interface{}, error) {
		return c.doRequest(ctx, method, endpoint, body)
	})
	if err != nil {
		return nil, err
	}

	// Safe type assertion with check
	respBody, ok := result.([]byte)
	if !ok {
		return nil, fmt.Errorf("unexpected response type")
	}

	return respBody, nil
}

// Authenticate performs authentication with the Hyundai API with retry logic
func (c *Client) Authenticate(ctx context.Context) error {
	return c.retrier.Do(ctx, func() error {
		// This is a simplified authentication flow
		// Real implementation would need proper OAuth flow based on region
		endpoint := fmt.Sprintf("%s/v2/login", c.baseURL)

		data := url.Values{}
		data.Set("username", c.username)
		data.Set("password", c.password)

		respBody, err := c.doRequest(ctx, "POST", endpoint, []byte(data.Encode()))
		if err != nil {
			return fmt.Errorf("authentication: %w", err)
		}

		var authResp AuthResponse
		if err := json.Unmarshal(respBody, &authResp); err != nil {
			return fmt.Errorf("parsing authentication response: %w", err)
		}

		c.accessToken = authResp.AccessToken
		c.refreshToken = authResp.RefreshToken

		return nil
	})
}

// GetVehicles retrieves the list of vehicles associated with the account with retry logic
func (c *Client) GetVehicles(ctx context.Context) ([]Vehicle, error) {
	ctx, cancel := context.WithTimeout(ctx, apiCallTimeout)
	defer cancel()

	endpoint := fmt.Sprintf("%s/v2/vehicles", c.baseURL)

	respBody, err := c.doRequestWithRetry(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("getting vehicles: %w", err)
	}

	var vehiclesResp VehiclesResponse
	if err := json.Unmarshal(respBody, &vehiclesResp); err != nil {
		return nil, fmt.Errorf("parsing vehicles response: %w", err)
	}

	return vehiclesResp.Vehicles, nil
}

// GetVehicleStatus retrieves the current status of a vehicle with retry logic
func (c *Client) GetVehicleStatus(ctx context.Context, vehicleID string) (*VehicleStatus, error) {
	// Check cache first if enabled
	cacheKey := fmt.Sprintf("status:%s", vehicleID)
	if c.cacheEnabled {
		if cached := c.cache.Get(cacheKey); cached != nil {
			if status, ok := cached.(*VehicleStatus); ok {
				return status, nil
			}
		}
	}

	ctx, cancel := context.WithTimeout(ctx, apiCallTimeout)
	defer cancel()

	endpoint := fmt.Sprintf("%s/v2/vehicles/%s/status", c.baseURL, vehicleID)

	respBody, err := c.doRequestWithRetry(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("getting vehicle status: %w", err)
	}

	var status VehicleStatus
	if err := json.Unmarshal(respBody, &status); err != nil {
		return nil, fmt.Errorf("parsing vehicle status response: %w", err)
	}

	// Store in cache before returning
	if c.cacheEnabled {
		c.cache.Set(cacheKey, &status)
	}

	return &status, nil
}

// GetVehicleLocation retrieves the current location of a vehicle with retry logic
func (c *Client) GetVehicleLocation(ctx context.Context, vehicleID string) (*Location, error) {
	// Check cache first if enabled
	cacheKey := fmt.Sprintf("location:%s", vehicleID)
	if c.cacheEnabled {
		if cached := c.cache.Get(cacheKey); cached != nil {
			if location, ok := cached.(*Location); ok {
				return location, nil
			}
		}
	}

	ctx, cancel := context.WithTimeout(ctx, apiCallTimeout)
	defer cancel()

	endpoint := fmt.Sprintf("%s/v2/vehicles/%s/location", c.baseURL, vehicleID)

	respBody, err := c.doRequestWithRetry(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("getting vehicle location: %w", err)
	}

	var location Location
	if err := json.Unmarshal(respBody, &location); err != nil {
		return nil, fmt.Errorf("parsing vehicle location response: %w", err)
	}

	// Store in cache before returning
	if c.cacheEnabled {
		c.cache.Set(cacheKey, &location)
	}

	return &location, nil
}

// GetOdometer retrieves the odometer reading with retry logic
func (c *Client) GetOdometer(ctx context.Context, vehicleID string) (*Odometer, error) {
	ctx, cancel := context.WithTimeout(ctx, apiCallTimeout)
	defer cancel()

	endpoint := fmt.Sprintf("%s/v2/vehicles/%s/odometer", c.baseURL, vehicleID)

	respBody, err := c.doRequestWithRetry(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("getting odometer: %w", err)
	}

	var odometer Odometer
	if err := json.Unmarshal(respBody, &odometer); err != nil {
		return nil, fmt.Errorf("parsing odometer response: %w", err)
	}

	return &odometer, nil
}

// GetCircuitBreakerState returns the current circuit breaker state
func (c *Client) GetCircuitBreakerState() circuitbreaker.State {
	return c.circuitBreaker.GetState()
}

// GetCircuitBreakerStats returns circuit breaker statistics
func (c *Client) GetCircuitBreakerStats() (state circuitbreaker.State, failures int, lastFailure time.Time) {
	return c.circuitBreaker.GetStats()
}
