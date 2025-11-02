# Captcha Workaround Guide

If you encounter captcha challenges when authenticating with the Hyundai/Kia API, this guide will help you bypass them.

## Why Captchas Appear

The Hyundai/Kia API servers use bot detection mechanisms that may trigger captchas when:
- Too many requests are made in a short period
- Suspicious User-Agent strings are detected
- Missing or incorrect headers
- New IP addresses or locations

## Recent Updates

We've updated the HTTP client to better mimic official mobile apps:

### ✅ Updated Features

1. **Realistic User-Agent Strings**
   - Now uses `okhttp/3.12.1` (official mobile app library)
   - For EU Hyundai: Uses Android WebView User-Agent

2. **Region-Specific Headers**
   - **EU Region:**
     - `ccsp-service-id`: Service identifier
     - `ccsp-application-id`: Application identifier
     - `Stamp`: Timestamp-based authentication stamp
     - `clientId`: ANDROID

   - **US/CA Region:**
     - `clientId`: ANDROID
     - Appropriate Host headers

3. **Standard HTTP Headers**
   - `Accept`: application/json
   - `Accept-Language`: en-US,en;q=0.9
   - `Accept-Encoding`: gzip, deflate, br
   - `Content-Type`: application/json

## Manual Authentication (If Captcha Still Appears)

If you still encounter captchas, use the manual authentication helper:

### Quick Start

```bash
make manual-auth
```

### Step-by-Step Process

1. **Run the Helper Script**
   ```bash
   make manual-auth
   ```

2. **Open Browser Developer Tools**
   - Chrome/Edge: Press `F12` or `Ctrl+Shift+I`
   - Firefox: Press `F12` or `Ctrl+Shift+I`
   - Safari: Press `Cmd+Option+I`

3. **Navigate to Network Tab**
   - Click on the "Network" tab in Developer Tools
   - Check "Preserve log" option

4. **Login Through Official Website**

   **Hyundai:**
   - US: https://www.mybluelink.ca/
   - EU: https://www.bluelink.hyundai.com/

   **Kia:**
   - US: https://www.kia.ca/owners
   - EU: https://www.kia.com/eu/owners/

5. **Solve the Captcha**
   - Complete any captcha challenges that appear
   - Login with your credentials

6. **Capture the Tokens**
   - After successful login, look in the Network tab for requests to:
     - `/login`
     - `/oauth`
     - `/token`
     - `/authorize`

   - Click on the request
   - Find the Response tab
   - Look for JSON containing:
     ```json
     {
       "access_token": "eyJ...",
       "refresh_token": "eyJ...",
       "expires_in": 3600
     }
     ```

7. **Enter Tokens in the Script**
   - Copy the `access_token` and `refresh_token`
   - Paste them when prompted by the script

8. **Tokens are Saved**
   - Tokens are saved to `.auth_tokens` (gitignored)
   - Add them to your `.env` file or export as environment variables

### Using Manual Tokens

After running `make manual-auth`, add the tokens to your environment:

**Option 1: Environment Variables**
```bash
export ACCESS_TOKEN="your_access_token_here"
export REFRESH_TOKEN="your_refresh_token_here"
```

**Option 2: .env File**
```bash
echo 'ACCESS_TOKEN="your_access_token_here"' >> .env
echo 'REFRESH_TOKEN="your_refresh_token_here"' >> .env
```

## Additional Tips

### 1. Use a VPN
Some users report success by using a VPN to appear in the same region as their account:
- EU accounts → Use EU VPN server
- US accounts → Use US VPN server

### 2. Wait Between Requests
If you've been rate-limited:
```yaml
rate_limit:
  requests_per_hour: 30  # Lower this value
  poll_interval_minutes: 10  # Increase this value
```

### 3. Browser Extensions
Consider using these browser extensions while capturing tokens:
- **ModHeader** - Add custom headers
- **EditThisCookie** - Manage cookies
- **JSON Formatter** - Better JSON viewing

### 4. Mobile App Method
Another option is to use a mobile proxy like:
- **Charles Proxy** (macOS/Windows)
- **mitmproxy** (Linux)
- **HTTP Toolkit** (Cross-platform)

These can intercept traffic from the official mobile app to capture tokens.

## Token Expiration

Authentication tokens typically expire after:
- Access Token: 1-2 hours
- Refresh Token: 30-90 days

When tokens expire:
1. The logger will attempt to use the refresh token automatically
2. If that fails, run `make manual-auth` again

## Troubleshooting

### Tokens Not Working
- Ensure you copied the complete token (they're usually very long)
- Check that tokens aren't expired
- Verify you're using the correct region in your config

### Still Getting Captchas
- Clear your browser cookies and cache
- Try a different browser
- Use private/incognito mode
- Try at a different time (APIs may have rate limits by time of day)

### Network Errors
- Check if your IP is blocked (try different network)
- Verify firewall isn't blocking connections
- Run `make preflight` to test connectivity

## Security Notes

⚠️ **Important Security Considerations:**

1. **Never Share Tokens**
   - Tokens provide full access to your vehicle
   - Never commit tokens to git repositories
   - `.auth_tokens` is automatically gitignored

2. **Token Storage**
   - Tokens are stored in plain text in `.auth_tokens`
   - File permissions are set to 600 (owner read/write only)
   - Consider encrypting tokens at rest for production use

3. **Rotate Tokens Regularly**
   - Generate new tokens periodically
   - Revoke old tokens if compromised

## Need Help?

If you're still having issues:
1. Check the GitHub issues: https://github.com/soothill/hyundai-logger/issues
2. Run `make preflight` to test connectivity
3. Enable debug logging in your config:
   ```yaml
   logging:
     level: "debug"
   ```

## References

- [Hyundai Bluelink](https://www.bluelink.hyundai.com/)
- [Kia Connect](https://www.kia.com/owners)
- [Genesis Connected Services](https://www.genesis.com/)
