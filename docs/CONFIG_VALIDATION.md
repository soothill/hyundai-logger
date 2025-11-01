# Configuration Validation Tool

The Configuration Validation Tool is a standalone utility that verifies your Hyundai Logger configuration before deployment. It tests all connectivity, validates settings, and reports any issues found.

## Installation

The validation tool is built automatically when you build the project:

```bash
go build ./cmd/validate-config
```

This creates a `validate-config` binary (or `validate-config.exe` on Windows).

## Usage

### Basic Usage

```bash
./validate-config --config config.yaml
```

### Command-Line Options

```bash
./validate-config [OPTIONS]

Options:
  --config string    Path to configuration file (default "config.yaml")
  --verbose          Enable verbose output
  --no-color         Disable colored output
  --help             Show help message
```

### Examples

**Validate default config:**
```bash
./validate-config
```

**Validate specific config with verbose output:**
```bash
./validate-config --config production.yaml --verbose
```

**Validate in CI/CD pipeline (no colors):**
```bash
./validate-config --config config.yaml --no-color
```

## What Gets Validated

### 1. Configuration File Structure ✓
- File exists and is readable
- Valid YAML syntax
- Required fields present
- Field types correct
- Value ranges valid

### 2. Hyundai API Connection ✓
- Credentials valid
- Authentication successful
- Can fetch vehicle list
- API is reachable

### 3. InfluxDB Connection ✓
- Database URL reachable
- Authentication token valid
- Organization exists
- Bucket accessible
- Health check passes

### 4. Webhooks (if enabled) ✓
- Webhook URLs valid
- Test notifications sent
- Response received
- Each webhook tested individually

## Example Output

### Successful Validation

```
╔════════════════════════════════════════════════════════════════╗
║     Hyundai Logger Configuration Validation Tool              ║
╚════════════════════════════════════════════════════════════════╝

Configuration file: config.yaml

✓ Configuration File:          Successfully loaded
✓ Config Structure:             All required fields present and valid
✓ Hyundai API:                  Connected successfully, found 2 vehicle(s) (3.2s)
✓ InfluxDB:                     Connected successfully and healthy (1.1s)
✓ Webhook - Slack:              Test notification sent successfully (0.5s)
✓ Webhook - Discord:            Test notification sent successfully (0.6s)

╔════════════════════════════════════════════════════════════════╗
║                       Validation Summary                       ║
╚════════════════════════════════════════════════════════════════╝

✓ Configuration File:          Successfully loaded
✓ Config Structure:             All required fields present and valid
✓ Hyundai API:                  Connected successfully, found 2 vehicle(s) (3.2s)
✓ InfluxDB:                     Connected successfully and healthy (1.1s)
✓ Webhook - Slack:              Test notification sent successfully (0.5s)
✓ Webhook - Discord:            Test notification sent successfully (0.6s)

Total: 6  Passed: 6

✅ Configuration validation PASSED
Your configuration is ready to use!
```

### Failed Validation

```
╔════════════════════════════════════════════════════════════════╗
║     Hyundai Logger Configuration Validation Tool              ║
╚════════════════════════════════════════════════════════════════╝

Configuration file: config.yaml

✓ Configuration File:          Successfully loaded
✓ Config Structure:             All required fields present and valid
✗ Hyundai API - Authentication: Login failed: invalid credentials (2.5s)
✗ InfluxDB - Health:            Health check failed: unauthorized (1.0s)
✗ Webhook - Slack:              Test failed: 404 Not Found (0.3s)
⚠ Webhooks:                     Disabled in configuration

╔════════════════════════════════════════════════════════════════╗
║                       Validation Summary                       ║
╚════════════════════════════════════════════════════════════════╝

✓ Configuration File:          Successfully loaded
✓ Config Structure:             All required fields present and valid
✗ Hyundai API:                  Login failed: invalid credentials (2.5s)
✗ InfluxDB:                     Health check failed: unauthorized (1.0s)
⚠ Webhook - Slack:              Test failed: 404 Not Found (0.3s)

Total: 5  Passed: 2  Warnings: 1  Failed: 2

❌ Configuration validation FAILED
Please fix the errors above before running hyundai-logger.
```

## Exit Codes

- **0** - All validations passed
- **1** - One or more validations failed

Use exit codes in scripts:

```bash
if ./validate-config --config config.yaml; then
    echo "Config is valid, starting application..."
    ./hyundai-logger --config config.yaml
else
    echo "Config validation failed!"
    exit 1
fi
```

## Verbose Mode

Enable verbose mode to see detailed progress:

```bash
./validate-config --config config.yaml --verbose
```

Output:
```
╔════════════════════════════════════════════════════════════════╗
║     Hyundai Logger Configuration Validation Tool              ║
╚════════════════════════════════════════════════════════════════╝

Configuration file: config.yaml

✓ Configuration File:          Successfully loaded
✓ Config Structure:             All required fields present and valid

  Testing Hyundai API connection...
  ✓ Authentication successful
  Fetching vehicle list...
  ✓ Found 2 vehicle(s)
✓ Hyundai API:                  Connected successfully, found 2 vehicle(s) (3.2s)

  Testing InfluxDB connection...
  ✓ Client created
  Checking health...
  ✓ Health check passed
✓ InfluxDB:                     Connected successfully and healthy (1.1s)

  Testing Slack webhook...
✓ Webhook - Slack:              Test notification sent successfully (0.5s)

  Testing Discord webhook...
✓ Webhook - Discord:            Test notification sent successfully (0.6s)

[... summary ...]
```

## Integration with CI/CD

### GitHub Actions

```yaml
name: Validate Configuration

on: [push, pull_request]

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Build validation tool
        run: go build ./cmd/validate-config

      - name: Create config from secrets
        run: |
          cat > config.yaml <<EOF
          hyundai:
            username: "${{ secrets.HYUNDAI_USERNAME }}"
            password: "${{ secrets.HYUNDAI_PASSWORD }}"
            pin: "${{ secrets.HYUNDAI_PIN }}"
            brand: "${{ secrets.HYUNDAI_BRAND }}"
            region: "${{ secrets.HYUNDAI_REGION }}"
          database:
            url: "${{ secrets.INFLUXDB_URL }}"
            token: "${{ secrets.INFLUXDB_TOKEN }}"
            organization: "${{ secrets.INFLUXDB_ORG }}"
            bucket: "${{ secrets.INFLUXDB_BUCKET }}"
          rate_limit:
            poll_interval_minutes: 5
            requests_per_hour: 50
          EOF

      - name: Validate configuration
        run: ./validate-config --config config.yaml --no-color
```

### GitLab CI

```yaml
validate-config:
  stage: test
  image: golang:1.21
  script:
    - go build ./cmd/validate-config
    - ./validate-config --config config.yaml --no-color
  only:
    - branches
```

### Docker

```dockerfile
FROM golang:1.21 AS builder
WORKDIR /app
COPY . .
RUN go build ./cmd/validate-config

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/validate-config /usr/local/bin/
COPY config.yaml /config.yaml
CMD ["validate-config", "--config", "/config.yaml"]
```

## Pre-Deployment Checklist

Use the validation tool as part of your deployment checklist:

```bash
#!/bin/bash
# deploy.sh

echo "Step 1: Validating configuration..."
if ! ./validate-config --config config.yaml; then
    echo "❌ Configuration validation failed!"
    exit 1
fi

echo "Step 2: Building application..."
go build ./cmd/hyundai-logger

echo "Step 3: Running tests..."
go test ./...

echo "Step 4: Deploying..."
# Your deployment commands here

echo "✅ Deployment complete!"
```

## Common Validation Errors

### Hyundai API Errors

**Authentication Failed**
```
✗ Hyundai API - Authentication: Login failed: invalid credentials
```

**Solution:**
- Verify username, password, and PIN are correct
- Check if account is locked
- Verify brand and region are correct

**No Vehicles Found**
```
⚠ Hyundai API: No vehicles found in account
```

**Solution:**
- Ensure vehicles are registered to your account
- Check if you're using the correct account
- Verify API region matches your account

### InfluxDB Errors

**Connection Failed**
```
✗ InfluxDB - Connection: Failed to create client: connection refused
```

**Solution:**
- Check if InfluxDB is running
- Verify URL is correct
- Check network connectivity

**Unauthorized**
```
✗ InfluxDB - Health: Health check failed: unauthorized
```

**Solution:**
- Verify authentication token is valid
- Check token has proper permissions
- Ensure organization name is correct

### Webhook Errors

**Invalid URL**
```
✗ Webhook - Slack: Test failed: no such host
```

**Solution:**
- Verify webhook URL is correct
- Check URL format (must include https://)
- Test URL in browser

**Timeout**
```
✗ Webhook - Discord: Test failed: context deadline exceeded
```

**Solution:**
- Check network connectivity
- Verify webhook endpoint is responsive
- Consider increasing timeout (if supported)

## Configuration Best Practices

1. **Validate Early** - Run validation before committing config changes
2. **Automate** - Include validation in CI/CD pipeline
3. **Version Control** - Keep validation tool up to date
4. **Document** - Note any expected warnings or exceptions
5. **Test Regularly** - Run validation periodically in production

## Troubleshooting

### Validation Tool Won't Start

**Error:** `./validate-config: command not found`

**Solution:**
```bash
# Build the tool first
go build ./cmd/validate-config

# Or build all tools
go build ./...
```

### Permission Denied

**Error:** `./validate-config: permission denied`

**Solution:**
```bash
chmod +x validate-config
```

### Cannot Find Config File

**Error:** `Configuration file not found: config.yaml`

**Solution:**
```bash
# Specify full path
./validate-config --config /path/to/config.yaml

# Or copy config to current directory
cp /path/to/config.yaml ./config.yaml
```

## Advanced Usage

### Scripting

Extract specific validation results:

```bash
#!/bin/bash

OUTPUT=$(./validate-config --config config.yaml --no-color 2>&1)

if echo "$OUTPUT" | grep -q "Hyundai API.*PASSED"; then
    echo "API check passed"
fi

if echo "$OUTPUT" | grep -q "InfluxDB.*PASSED"; then
    echo "Database check passed"
fi
```

### Custom Validation

Extend the tool with custom validation checks:

```go
// Add to cmd/validate-config/main.go

func validateCustom(cfg *config.Config) ValidationResult {
    // Your custom validation logic
    return ValidationResult{
        Component: "Custom Check",
        Status:    "pass",
        Message:   "Custom validation passed",
    }
}
```

## Support

For issues or feature requests:
- GitHub Issues: https://github.com/soothill/hyundai-logger/issues
- Documentation: https://github.com/soothill/hyundai-logger/docs

## Related Documentation

- [Configuration Guide](CONFIGURATION.md)
- [Webhook Setup](WEBHOOKS.md)
- [Deployment Guide](DEPLOYMENT.md)
