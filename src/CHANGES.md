# Hyundai Logger - Authentication Update Summary

## Overview
This update addresses the recent changes in Hyundai/Kia's authentication system that broke the original logger. The new system uses OAuth2 with reCAPTCHA validation and requires device ID management.

## Key Changes

### 1. New Authentication System (`internal/auth/`)

#### OAuth2 Handler (`oauth.go`)
- Implements full OAuth2 flow with PKCE support
- Manages refresh tokens and access tokens
- Automatic token refresh when expired
- Device ID generation and persistence
- Region-specific client configurations for EU, US, and CA
- Token storage in `~/.hyundai-logger/tokens.json`

#### Stamp Manager (`stamp.go`)
- EU-specific cryptographic signatures for API requests
- Fetches stamps from community-maintained services
- 7-day stamp validity with automatic renewal
- Fallback to alternative stamp sources

### 2. Updated API Client (`internal/api/client.go`)
- Uses OAuth2 authentication instead of username/password
- Automatic token refresh on 401 errors
- Device ID header injection for all requests
- Region-specific endpoint handling
- Stamp signing for EU requests
- Support for all vehicle control commands (lock/unlock, climate, charging)

### 3. Refresh Token Fetcher (`scripts/get_refresh_token.py`)
- Python script using Selenium for browser automation
- Handles reCAPTCHA challenges automatically
- Supports all regions (EU, US, CA) and brands (Hyundai, Kia)
- Saves tokens automatically to `~/.hyundai-logger/tokens.json`
- PKCE flow implementation for secure token exchange

### 4. Configuration Updates (`internal/config/config.go`)
- New `refresh_token` field for OAuth2
- Device ID storage and management
- Backward compatibility with environment variables
- Enhanced validation for new authentication fields

### 5. Main Application (`cmd/hyundai-logger/main.go`)
- Updated to use new authentication system
- Falls back to stored tokens if available
- Enhanced error handling for authentication failures
- Verbose mode for debugging

## Migration Guide

### For Users
1. **Get Refresh Token**: Run `python3 scripts/get_refresh_token.py <REGION> <BRAND>`
2. **Update Config**: Add `refresh_token` to your `config.yaml`
3. **Remove Old Auth**: Username/password no longer needed (keep for reference)
4. **Run Updated Version**: Use the new binary

### For Developers
1. **Replace Auth Module**: Old session-based auth is replaced with OAuth2
2. **Update API Calls**: All API calls now use bearer token authentication
3. **Add Device ID**: Device ID is required for most API endpoints
4. **Handle Stamps**: EU region requires stamp signatures

## Technical Details

### OAuth2 Flow
1. User initiates auth through browser
2. Completes login with reCAPTCHA
3. Authorization code captured from redirect
4. Code exchanged for refresh/access tokens
5. Tokens stored locally with encryption
6. Access token auto-refreshes using refresh token

### Device ID Management
- Auto-generated UUID if not present
- Stored with tokens for persistence
- Required header for API requests
- Region-specific handling

### Stamp Mechanism (EU Only)
- Cryptographic signatures for API security
- Community-maintained stamp service
- 7-day validity period
- Automatic renewal

## Files Changed/Added

### New Files
- `internal/auth/oauth.go` - OAuth2 implementation
- `internal/auth/stamp.go` - EU stamp signatures
- `scripts/get_refresh_token.py` - Browser-based token fetcher
- `setup.sh` - Interactive setup assistant

### Modified Files
- `internal/api/client.go` - OAuth2 authentication, device ID support
- `internal/config/config.go` - Refresh token configuration
- `cmd/hyundai-logger/main.go` - New auth flow integration
- `config.yaml` - Added refresh_token field

### Supporting Files
- `Makefile` - Build automation with token fetch targets
- `README.md` - Complete documentation update
- `go.mod` - Dependencies remain mostly the same

## Security Improvements
1. **No Password Storage**: Only refresh token stored
2. **Token Encryption**: Tokens stored with 0600 permissions
3. **Automatic Refresh**: Access tokens refresh automatically
4. **PKCE Protection**: Authorization flow uses PKCE for security

## Known Issues & Limitations
1. **Chrome Required**: Token fetcher needs Chrome for reCAPTCHA
2. **Manual Login**: Initial setup requires manual browser login
3. **Token Expiry**: Refresh tokens may expire after extended periods
4. **API Changes**: Using reverse-engineered APIs that may change

## Testing Recommendations
1. Test token fetcher with your region/brand combination
2. Verify device ID persistence across restarts
3. Test token refresh after access token expiry (1 hour)
4. Monitor for rate limiting issues
5. Verify all vehicle data points are captured

## Rollback Plan
If issues occur:
1. Keep backup of old code
2. Tokens can be regenerated anytime
3. Database schema remains compatible
4. Config changes are backward compatible

## Future Enhancements
1. Web UI for token management
2. Automatic token refresh scheduling
3. Multiple vehicle support improvements
4. Enhanced error recovery
5. Token encryption at rest
