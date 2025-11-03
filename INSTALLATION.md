# Hyundai Logger Update - Installation Guide

## Quick Install

1. Extract the files:
   ```bash
   ./extract.sh
   ```

2. Install dependencies:
   ```bash
   # Python dependencies for token fetcher
   pip3 install selenium requests
   
   # Go dependencies
   go mod download
   ```

3. Get your refresh token:
   ```bash
   python3 scripts/get_refresh_token.py <REGION> <BRAND>
   # Example: python3 scripts/get_refresh_token.py EU hyundai
   ```

4. Configure:
   - Edit `config.yaml`
   - Add your refresh token
   - Set your PIN, region, and brand

5. Build and run:
   ```bash
   go build -o hyundai-logger cmd/hyundai-logger/main.go
   ./hyundai-logger -config config.yaml
   ```

## Manual Installation

If the extraction script doesn't work, you can manually copy the files from:
- `src/` directory - contains all the updated files
- `all-files-combined.txt` - contains all source code in a single file

## File Structure

```
hyundai-logger-updated/
├── cmd/hyundai-logger/main.go      # Main application
├── internal/
│   ├── api/                        # API client with OAuth2
│   ├── auth/                       # Authentication modules
│   ├── config/                     # Configuration
│   └── database/                   # Database client
├── scripts/
│   └── get_refresh_token.py        # Token fetcher
├── config.yaml                     # Configuration template
├── setup.sh                        # Setup assistant
├── Makefile                        # Build automation
└── README.md                       # Documentation
```

## Important Notes

- **Chrome Required**: The token fetcher needs Chrome/Chromium for reCAPTCHA
- **Battery Warning**: Don't exceed 24 requests/hour or you'll drain your 12V battery
- **Token Security**: Keep your refresh token secure, it provides full account access
