# Code Optimization and Improvement Recommendations

## Overview
This document outlines optimization opportunities and improvements identified in the Hyundai Logger codebase.

## High Priority Issues

### 1. HTTP Request Code Duplication (internal/api/client.go)
**Issue:** Lines 90-318 repeat the same HTTP request pattern across multiple methods.

**Current State:**
```go
// Repeated in Authenticate, GetVehicles, GetVehicleStatus, GetVehicleLocation
req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.accessToken))
req.Header.Set("User-Agent", "HyundaiLogger/1.0")
resp, err := c.httpClient.Do(req)
defer resp.Body.Close()
body, err := io.ReadAll(resp.Body)
// ... error handling
```

**Recommendation:** Extract into helper methods:
```go
// doRequest performs an HTTP request with common headers and error handling
func (c *Client) doRequest(ctx context.Context, method, endpoint string, body io.Reader) ([]byte, error) {
    if err := c.rateLimiter.Wait(ctx); err != nil {
        return nil, fmt.Errorf("rate limiter: %w", err)
    }

    req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
    if err != nil {
        return nil, fmt.Errorf("creating request: %w", err)
    }

    req.Header.Set("User-Agent", "HyundaiLogger/1.0")
    if c.accessToken != "" {
        req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.accessToken))
    }

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("making request: %w", err)
    }
    defer resp.Body.Close()

    respBody, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("reading response: %w", err)
    }

    if resp.StatusCode != http.StatusOK {
        // Don't include full body in error (may contain sensitive data)
        return nil, fmt.Errorf("request failed with status %d", resp.StatusCode)
    }

    return respBody, nil
}

// doRequestWithRetry wraps doRequest with retry logic
func (c *Client) doRequestWithRetry(ctx context.Context, method, endpoint string, body io.Reader) ([]byte, error) {
    result, err := c.retrier.DoWithResult(ctx, func() (interface{}, error) {
        return c.doRequest(ctx, method, endpoint, body)
    })
    if err != nil {
        return nil, err
    }
    return result.([]byte), nil
}
```

**Impact:** Reduces ~200 lines of duplicated code, improves maintainability.

---

### 2. GetOdometer Lacks Retry Logic (internal/api/client.go:282)
**Issue:** GetOdometer doesn't use retry logic like other methods.

**Current:**
```go
func (c *Client) GetOdometer(ctx context.Context, vehicleID string) (*Odometer, error) {
    if err := c.rateLimiter.Wait(ctx); err != nil {
        return nil, fmt.Errorf("rate limiter: %w", err)
    }
    // ... direct HTTP call without retry
}
```

**Recommendation:** Wrap in retry logic like other methods:
```go
func (c *Client) GetOdometer(ctx context.Context, vehicleID string) (*Odometer, error) {
    result, err := c.retrier.DoWithResult(ctx, func() (interface{}, error) {
        // ... existing implementation
    })
    if err != nil {
        return nil, err
    }
    return result.(*Odometer), nil
}
```

**Impact:** Improved reliability for odometer fetching.

---

### 3. Inefficient WriteAPI Creation (internal/database/database.go)
**Issue:** Creating new WriteAPIBlocking for each database write operation.

**Lines Affected:** 92, 115, 231, 258

**Current:**
```go
func (db *DB) InsertVehicleStatus(ctx context.Context, status *api.VehicleStatus) error {
    writeAPI := db.client.WriteAPIBlocking(db.org, db.bucket) // Created each time
    // ...
}
```

**Recommendation:** Create WriteAPI once and reuse:
```go
type DB struct {
    client   influxdb2.Client
    writeAPI api.WriteAPIBlocking
    org      string
    bucket   string
}

func New(ctx context.Context, url, token, org, bucket string) (*DB, error) {
    client := influxdb2.NewClient(url, token)
    // ... health check ...

    return &DB{
        client:   client,
        writeAPI: client.WriteAPIBlocking(org, bucket),
        org:      org,
        bucket:   bucket,
    }, nil
}

func (db *DB) InsertVehicleStatus(ctx context.Context, status *api.VehicleStatus) error {
    // Use db.writeAPI directly
    return db.writeAPI.WritePoint(ctx, points...)
}
```

**Impact:** Reduces object allocation overhead, improves performance.

---

### 4. SQL/Flux Injection Vulnerability (internal/database/database.go:281)
**Issue:** Direct string interpolation in Flux query allows injection.

**Current:**
```go
query := fmt.Sprintf(`
    from(bucket: "%s")
    |> filter(fn: (r) => r["vin"] == "%s")
`, db.bucket, vehicleID)
```

**Recommendation:** Use parameterized queries or proper escaping:
```go
import "github.com/influxdata/influxdb-client-go/v2/api/query"

// Use query parameters (if supported) or validate/escape input
func (db *DB) GetLatestStatus(ctx context.Context, vehicleID string) (*api.VehicleStatus, error) {
    // Validate vehicleID doesn't contain special characters
    if !isValidVehicleID(vehicleID) {
        return nil, fmt.Errorf("invalid vehicle ID format")
    }

    queryAPI := db.client.QueryAPI(db.org)
    // Still use fmt.Sprintf but with validated input
    query := fmt.Sprintf(`...`, escapeFluxString(db.bucket), escapeFluxString(vehicleID))
    // ...
}

func escapeFluxString(s string) string {
    // Escape special characters for Flux
    return strings.ReplaceAll(strings.ReplaceAll(s, `"`, `\"`), `\`, `\\`)
}

func isValidVehicleID(id string) bool {
    // Only allow alphanumeric and hyphens
    matched, _ := regexp.MatchString(`^[a-zA-Z0-9-]+$`, id)
    return matched
}
```

**Impact:** Prevents potential injection attacks.

---

### 5. Sensitive Data in Error Messages (internal/api/client.go)
**Issue:** Full response bodies included in error messages (may contain tokens, credentials, etc.).

**Lines:** 125, 169, 216, 263, 309

**Current:**
```go
if resp.StatusCode != http.StatusOK {
    return fmt.Errorf("authentication failed (status %d): %s", resp.StatusCode, string(body))
}
```

**Recommendation:** Only include status code, not body:
```go
if resp.StatusCode != http.StatusOK {
    // Log full error internally if needed, but don't expose in error message
    c.logger.Debug("Request failed with body: %s", string(body))
    return fmt.Errorf("request failed with status %d", resp.StatusCode)
}
```

**Impact:** Prevents accidental leakage of sensitive data in logs.

---

## Medium Priority Issues

### 6. Code Duplication in Retry Logic (internal/retry/retry.go)
**Issue:** `Do` and `DoWithResult` have 95% identical code.

**Recommendation:** Refactor to reduce duplication:
```go
func (r *Retrier) Do(ctx context.Context, operation func() error) error {
    _, err := r.DoWithResult(ctx, func() (interface{}, error) {
        return nil, operation()
    })
    return err
}
```

**Impact:** Reduces maintenance burden, ensures consistent behavior.

---

### 7. Redundant Context Checks (internal/retry/retry.go)
**Issue:** Context is checked at lines 54 and 63 (and 94/103).

**Current:**
```go
if ctx.Err() != nil {
    return fmt.Errorf("context canceled after %d attempts: %w", attempt, ctx.Err())
}
// ... delay calculation ...
select {
case <-ctx.Done():  // Checked again here
    return fmt.Errorf("context canceled after %d attempts: %w", attempt, ctx.Err())
case <-time.After(delay):
}
```

**Recommendation:** Remove first check, keep select statement:
```go
// Calculate backoff delay
delay := r.calculateDelay(attempt)

// Wait before retrying (context check happens in select)
select {
case <-ctx.Done():
    return fmt.Errorf("context canceled after %d attempts: %w", attempt, ctx.Err())
case <-time.After(delay):
    // Continue to next attempt
}
```

**Impact:** Minor performance improvement, cleaner code.

---

### 8. Unsafe Type Assertions (internal/api/client.go)
**Issue:** Type assertions without safety checks could panic.

**Lines:** 184, 231, 278

**Current:**
```go
return result.([]Vehicle), nil
```

**Recommendation:** Use safe type assertion:
```go
vehicles, ok := result.([]Vehicle)
if !ok {
    return nil, fmt.Errorf("unexpected result type")
}
return vehicles, nil
```

**Impact:** Prevents potential panics, more robust error handling.

---

### 9. HTTP Client Configuration (internal/api/client.go:48)
**Issue:** HTTP client uses default settings with hardcoded timeout.

**Current:**
```go
httpClient: &http.Client{
    Timeout: 30 * time.Second,
},
```

**Recommendation:** Configure connection pooling and make timeout configurable:
```go
func NewClient(username, password, pin, brand, region string, requestsPerHour int, retryConfig retry.Config, httpTimeout time.Duration) *Client {
    if httpTimeout == 0 {
        httpTimeout = 30 * time.Second
    }

    transport := &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
        IdleConnTimeout:     90 * time.Second,
        DisableKeepAlives:   false,
    }

    return &Client{
        httpClient: &http.Client{
            Timeout:   httpTimeout,
            Transport: transport,
        },
        // ...
    }
}
```

**Impact:** Better resource management, configurable timeouts.

---

### 10. Missing Graceful Shutdown for InfluxDB (internal/database/database.go:49)
**Issue:** WriteAPI should be flushed before closing client.

**Current:**
```go
func (db *DB) Close() {
    db.client.Close()
}
```

**Recommendation:** Flush any pending writes:
```go
func (db *DB) Close() error {
    // If using async WriteAPI, flush it
    if db.writeAPI != nil {
        db.writeAPI.Flush(context.Background())
    }
    db.client.Close()
    return nil
}
```

**Impact:** Ensures all data is written before shutdown.

---

### 11. Incomplete GetLatestStatus Implementation (internal/database/database.go:278)
**Issue:** Only returns timestamp, doesn't parse actual status fields.

**Current:**
```go
// Parse result into VehicleStatus (simplified)
status := &api.VehicleStatus{}
if result.Record().Time().Unix() > 0 {
    status.Timestamp = result.Record().Time()
}
return status, nil
```

**Recommendation:** Either implement fully or remove if not used:
```go
// Option 1: Implement fully
func (db *DB) GetLatestStatus(ctx context.Context, vehicleID string) (*api.VehicleStatus, error) {
    // Query all measurements and reconstruct VehicleStatus
    // This requires multiple queries or a complex Flux query with pivot
}

// Option 2: If not used, remove the method
```

**Impact:** Either provides complete functionality or removes dead code.

---

## Low Priority / Nice to Have

### 12. Add Metrics/Instrumentation
**Recommendation:** Add metrics for monitoring:
- API call success/failure rates
- Retry counts
- Response times
- Database write latencies
- Active polling intervals

**Implementation:**
```go
// Add prometheus metrics
type Metrics struct {
    apiCalls        *prometheus.CounterVec
    retryAttempts   prometheus.Counter
    responseTimes   prometheus.Histogram
    dbWriteDuration prometheus.Histogram
}
```

---

### 13. Add Jitter to Retry Backoff
**Issue:** All retry attempts use exact exponential backoff.

**Recommendation:** Add jitter to prevent thundering herd:
```go
func (r *Retrier) calculateDelay(attempt int) time.Duration {
    delayMs := float64(r.config.InitialDelayMs) * math.Pow(r.config.BackoffMultiplier, float64(attempt-1))

    if delayMs > float64(r.config.MaxDelayMs) {
        delayMs = float64(r.config.MaxDelayMs)
    }

    // Add jitter: ±25% randomness
    jitter := delayMs * 0.25 * (rand.Float64()*2 - 1)
    delayMs += jitter

    return time.Duration(delayMs) * time.Millisecond
}
```

---

### 14. Add Request ID Tracing
**Recommendation:** Add request IDs for distributed tracing:
```go
func (c *Client) doRequest(ctx context.Context, method, endpoint string, body io.Reader) ([]byte, error) {
    requestID := uuid.New().String()
    req.Header.Set("X-Request-ID", requestID)

    // Log request ID for correlation
    log.Printf("Request %s: %s %s", requestID, method, endpoint)
    // ...
}
```

---

### 15. Add Circuit Breaker Pattern
**Recommendation:** Prevent cascading failures with circuit breaker:
```go
import "github.com/sony/gobreaker"

type Client struct {
    // ...
    circuitBreaker *gobreaker.CircuitBreaker
}

func (c *Client) doRequest(...) {
    result, err := c.circuitBreaker.Execute(func() (interface{}, error) {
        // ... actual HTTP request ...
    })
}
```

---

## Summary Statistics

- **High Priority Issues:** 5 (security, reliability, performance)
- **Medium Priority Issues:** 6 (code quality, maintainability)
- **Low Priority Issues:** 3 (observability, resilience)

**Estimated Impact:**
- Code reduction: ~200-300 lines
- Performance improvement: 10-15% (from WriteAPI reuse)
- Security: Fixes injection vulnerability and sensitive data leakage
- Reliability: Adds missing retry logic to GetOdometer

## Recommended Implementation Order

1. Fix SQL injection vulnerability (#4) - Security critical
2. Remove sensitive data from errors (#5) - Security important
3. Extract HTTP request helper (#1) - Largest code improvement
4. Add retry to GetOdometer (#2) - Consistency
5. Reuse WriteAPI (#3) - Performance gain
6. Implement other medium/low priority items as time permits
