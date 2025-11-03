#!/bin/bash

# Manual Token Helper Script
# Use this if you encounter captcha during automatic authentication
#
# This script helps you manually obtain authentication tokens using your browser

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
TOKEN_FILE="$PROJECT_ROOT/.auth_tokens"

echo "═══════════════════════════════════════════════════════"
echo "  Hyundai Logger - Manual Token Helper"
echo "═══════════════════════════════════════════════════════"
echo ""
echo "If automatic authentication fails due to captcha, use this"
echo "script to manually extract tokens from your browser."
echo ""
echo "💡 TIP: For automated token extraction, try the Python script:"
echo "   https://gist.github.com/RustyDust/e2a7be978affd85fb5ef5a345f31f67a"
echo ""

# Function to get user input with default
get_input() {
    local prompt="$1"
    local default="$2"
    local value

    if [ -n "$default" ]; then
        read -p "$prompt [$default]: " value
        echo "${value:-$default}"
    else
        read -p "$prompt: " value
        echo "$value"
    fi
}

# Get region and brand
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Step 1: Configuration"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

BRAND=$(get_input "Brand (hyundai/kia/genesis)" "hyundai")
REGION=$(get_input "Region (US/CA/EU/KR/AU/JP/IN/BR)" "EU")

# Determine the login URL based on brand and region
case "$REGION" in
    "EU")
        case "$BRAND" in
            "hyundai")
                LOGIN_URL="https://prd.eu-ccapi.hyundai.com:8080/api/v1/user/oauth2/authorize"
                ;;
            "kia")
                LOGIN_URL="https://prd.eu-ccapi.kia.com:8080/api/v1/user/oauth2/authorize"
                ;;
            *)
                echo "❌ Unsupported brand for EU region"
                exit 1
                ;;
        esac
        ;;
    "US"|"CA")
        case "$BRAND" in
            "hyundai")
                LOGIN_URL="https://api.telematics.hyundaiusa.com/v2/login"
                ;;
            "kia")
                LOGIN_URL="https://api.owners.kia.com/apigw/v1/oauth2/authorize"
                ;;
            *)
                echo "❌ Unsupported brand for US/CA region"
                exit 1
                ;;
        esac
        ;;
    *)
        echo "❌ Unsupported region: $REGION"
        exit 1
        ;;
esac

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Step 2: Manual Browser Login"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Instructions:"
echo ""
echo "1. Open your browser and go to the ${BRAND^} mobile website:"
case "$BRAND" in
    "hyundai")
        echo "   https://www.mybluelink.ca/ (or your region's site)"
        ;;
    "kia")
        echo "   https://www.kia.ca/owners (or your region's site)"
        ;;
esac
echo ""
echo "2. Open Browser Developer Tools:"
echo "   - Chrome/Edge: F12 or Ctrl+Shift+I"
echo "   - Firefox: F12 or Ctrl+Shift+I"
echo "   - Safari: Cmd+Option+I"
echo ""
echo "3. Go to the 'Network' tab in Developer Tools"
echo ""
echo "4. Login to your account (solve any captchas that appear)"
echo ""
echo "5. After successful login, look for a network request to:"
echo "   - 'login' or 'oauth' or 'token'"
echo ""
echo "6. Click on the request and find the Response containing:"
echo "   - access_token"
echo "   - refresh_token"
echo ""
echo "7. Copy these tokens and paste them below"
echo ""
echo "Press Enter when you're ready to continue..."
read -r

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Step 3: Enter Tokens"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

echo "Paste your access_token:"
read -r ACCESS_TOKEN

echo ""
echo "Paste your refresh_token:"
read -r REFRESH_TOKEN

echo ""
echo "Optional - Paste your device_id (EU only, leave empty to auto-generate):"
echo "(Look in browser Network tab for 'ccsp-device-id' header)"
read -r DEVICE_ID

# Validate tokens are not empty
if [ -z "$ACCESS_TOKEN" ] || [ -z "$REFRESH_TOKEN" ]; then
    echo ""
    echo "❌ Error: Both tokens are required"
    exit 1
fi

# Validate tokens by testing with API
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Step 4: Validating Tokens"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Testing tokens with API..."

# Temporarily save tokens to file for validation
cat > "$TOKEN_FILE.tmp" <<EOF
ACCESS_TOKEN="$ACCESS_TOKEN"
REFRESH_TOKEN="$REFRESH_TOKEN"
EOF

# Export tokens for validation
export ACCESS_TOKEN
export REFRESH_TOKEN

# Test tokens by trying to fetch vehicles
VALIDATION_OUTPUT=$(cd "$PROJECT_ROOT" && go run cmd/preflight/main.go 2>&1 | grep -A 3 "Testing API authentication")
VALIDATION_RESULT=$?

# Check if validation succeeded
if echo "$VALIDATION_OUTPUT" | grep -q "Authentication successful (using existing tokens)"; then
    echo "✓ Token validation successful!"
    echo "  Tokens are valid and working"
    rm -f "$TOKEN_FILE.tmp"
elif echo "$VALIDATION_OUTPUT" | grep -q "existing tokens invalid"; then
    echo "❌ Token validation failed"
    echo "  The tokens appear to be invalid or expired"
    echo ""
    echo "Possible issues:"
    echo "  - Tokens may have been copied incorrectly"
    echo "  - Tokens may have expired (access tokens expire in 1-2 hours)"
    echo "  - Wrong region or brand selected"
    echo ""
    rm -f "$TOKEN_FILE.tmp"
    read -p "Continue anyway and save tokens? (y/N): " CONTINUE
    if [[ ! "$CONTINUE" =~ ^[Yy]$ ]]; then
        echo "❌ Aborted. Please try again with fresh tokens."
        exit 1
    fi
    echo "⚠️  Saving tokens despite validation failure..."
else
    echo "⚠️  Could not validate tokens (preflight check unavailable)"
    echo "  Tokens will be saved without validation"
    rm -f "$TOKEN_FILE.tmp"
fi

# Save tokens to file
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Step 5: Saving Tokens"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

cat > "$TOKEN_FILE" <<EOF
# Authentication tokens (generated manually)
# Generated: $(date)
# Brand: $BRAND
# Region: $REGION

ACCESS_TOKEN="$ACCESS_TOKEN"
REFRESH_TOKEN="$REFRESH_TOKEN"
EOF

# Add device ID if provided
if [ -n "$DEVICE_ID" ]; then
    echo "DEVICE_ID=\"$DEVICE_ID\"" >> "$TOKEN_FILE"
fi

chmod 600 "$TOKEN_FILE"

echo "✓ Tokens saved to: $TOKEN_FILE"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Step 6: Next Steps"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "✓ Tokens are automatically loaded from .auth_tokens"
echo ""
echo "You can now run:"
echo "  make preflight  # Test connectivity"
echo "  make run        # Start logging"
echo ""
echo "Optional - Set environment variables manually:"
echo ""
echo "export ACCESS_TOKEN=\"$ACCESS_TOKEN\""
echo "export REFRESH_TOKEN=\"$REFRESH_TOKEN\""
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✓ Done!"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Note: These tokens may expire. If they do, run this script again."
echo ""
