#!/usr/bin/env python3
"""
Hyundai/Kia OAuth Token Fetcher
This script helps obtain the refresh token needed for the new authentication system.
Requires: Python 3.6+, selenium, Chrome/Chromium browser
"""

import json
import sys
import time
import base64
import hashlib
import secrets
import urllib.parse
from pathlib import Path

try:
    from selenium import webdriver
    from selenium.webdriver.common.by import By
    from selenium.webdriver.support.ui import WebDriverWait
    from selenium.webdriver.support import expected_conditions as EC
    from selenium.webdriver.chrome.service import Service
    from selenium.webdriver.chrome.options import Options
except ImportError:
    print("Error: Selenium is required. Install with: pip3 install selenium")
    sys.exit(1)

# Configuration for different regions and brands
CONFIGS = {
    "EU": {
        "hyundai": {
            "client_id": "6d477c38-3ca4-4cf3-9557-2a1929a94654",
            "client_secret": "KUy49XxPzLpLuoK0xhBC77W6VXhmtQR9iQhmIFjjoY4IpxsV",
            "auth_url": "https://prd.eu-ccapi.hyundai.com/api/v1/user/oauth2/authorize",
            "token_url": "https://prd.eu-ccapi.hyundai.com/api/v1/user/oauth2/token",
            "redirect_uri": "https://prd.eu-ccapi.hyundai.com/api/v1/user/oauth2/redirect",
            "login_url": "https://prd.eu-ccapi.hyundai.com/api/v1/user/integrationinfo"
        },
        "kia": {
            "client_id": "fdc85c00-0a2f-4c64-bcb4-2cfb1500730a",
            "client_secret": "PC6SymPRRlW9TSxEJ7g5eetRJKPPSZaOXNBQS2QmB8PH0jLBZ8",
            "auth_url": "https://prd.eu-ccapi.kia.com/api/v1/user/oauth2/authorize",
            "token_url": "https://prd.eu-ccapi.kia.com/api/v1/user/oauth2/token",
            "redirect_uri": "https://prd.eu-ccapi.kia.com/api/v1/user/oauth2/redirect",
            "login_url": "https://prd.eu-ccapi.kia.com/api/v1/user/integrationinfo"
        }
    },
    "US": {
        "hyundai": {
            "client_id": "64621b96-0f0d-11ec-82a8-0242ac130003",
            "client_secret": "LJbr0bHwOVKZOYP6P2ucLmVnBubhJM8TCLYFqnR1rsa0USIKzqpCPvbIQfUp2tc8aP3A8OpqD5oVDPXtPnLMpA==",
            "auth_url": "https://api.telematics.hyundaiusa.com/oauth2/authorize",
            "token_url": "https://api.telematics.hyundaiusa.com/oauth2/token",
            "redirect_uri": "https://www.getpostman.com/oauth2/callback",
            "login_url": "https://api.telematics.hyundaiusa.com/v2/ac/oauth/authorize"
        }
    },
    "CA": {
        "hyundai": {
            "client_id": "64621b96-0f0d-11ec-82a8-0242ac130003",
            "client_secret": "LJbr0bHwOVKZOYP6P2ucLmVnBubhJM8TCLYFqnR1rsa0USIKzqpCPvbIQfUp2tc8aP3A8OpqD5oVDPXtPnLMpA==",
            "auth_url": "https://api.telematics.hyundaicanada.com/oauth2/authorize",
            "token_url": "https://api.telematics.hyundaicanada.com/oauth2/token",
            "redirect_uri": "https://www.getpostman.com/oauth2/callback",
            "login_url": "https://api.telematics.hyundaicanada.com/v2/ac/oauth/authorize"
        }
    }
}

def generate_code_verifier():
    """Generate a PKCE code verifier"""
    return base64.urlsafe_b64encode(secrets.token_bytes(32)).decode('utf-8').rstrip('=')

def generate_code_challenge(verifier):
    """Generate a PKCE code challenge from verifier"""
    digest = hashlib.sha256(verifier.encode('utf-8')).digest()
    return base64.urlsafe_b64encode(digest).decode('utf-8').rstrip('=')

def get_refresh_token(region, brand):
    """Main function to obtain refresh token through browser authentication"""
    
    if region not in CONFIGS:
        print(f"Error: Region '{region}' not supported. Supported regions: {list(CONFIGS.keys())}")
        sys.exit(1)
    
    if brand not in CONFIGS[region]:
        print(f"Error: Brand '{brand}' not supported in region '{region}'. Supported brands: {list(CONFIGS[region].keys())}")
        sys.exit(1)
    
    config = CONFIGS[region][brand]
    
    # Generate PKCE parameters
    code_verifier = generate_code_verifier()
    code_challenge = generate_code_challenge(code_verifier)
    state = secrets.token_urlsafe(32)
    
    # Build authorization URL
    auth_params = {
        'response_type': 'code',
        'client_id': config['client_id'],
        'redirect_uri': config['redirect_uri'],
        'state': state,
        'code_challenge': code_challenge,
        'code_challenge_method': 'S256'
    }
    
    auth_url = config['auth_url'] + '?' + urllib.parse.urlencode(auth_params)
    
    print("\n" + "="*60)
    print(f"Hyundai/Kia OAuth Token Fetcher")
    print(f"Region: {region} | Brand: {brand.capitalize()}")
    print("="*60)
    
    # Setup Chrome options
    chrome_options = Options()
    chrome_options.add_argument('--disable-blink-features=AutomationControlled')
    chrome_options.add_experimental_option("excludeSwitches", ["enable-automation"])
    chrome_options.add_experimental_option('useAutomationExtension', False)
    
    # Initialize Chrome driver
    print("\n1. Opening Chrome browser...")
    try:
        driver = webdriver.Chrome(options=chrome_options)
    except Exception as e:
        print(f"Error: Failed to start Chrome. Make sure Chrome/Chromium is installed.")
        print(f"Error details: {e}")
        sys.exit(1)
    
    try:
        # Navigate to authorization URL
        print("2. Navigating to Hyundai/Kia login page...")
        driver.get(auth_url)
        
        print("\n" + "-"*60)
        print("IMPORTANT: Please complete the following steps:")
        print("1. Enter your Hyundai/Kia account credentials")
        print("2. Complete any reCAPTCHA if shown")
        print("3. Accept any consent screens")
        print("4. Wait for the redirect (you'll see an error page)")
        print("-"*60)
        
        print("\nWaiting for login completion...")
        print("(The script will continue automatically after successful login)")
        
        # Wait for redirect to callback URL
        wait = WebDriverWait(driver, 300)  # 5 minute timeout
        wait.until(lambda d: config['redirect_uri'] in d.current_url)
        
        # Get the authorization code from URL
        current_url = driver.current_url
        parsed_url = urllib.parse.urlparse(current_url)
        params = urllib.parse.parse_qs(parsed_url.query)
        
        if 'code' not in params:
            print("Error: Authorization code not found in redirect URL")
            sys.exit(1)
        
        auth_code = params['code'][0]
        print(f"\n3. Authorization successful! Got auth code: {auth_code[:10]}...")
        
    finally:
        driver.quit()
    
    # Exchange authorization code for tokens
    print("\n4. Exchanging authorization code for tokens...")
    
    import requests
    
    token_data = {
        'grant_type': 'authorization_code',
        'code': auth_code,
        'redirect_uri': config['redirect_uri'],
        'client_id': config['client_id'],
        'code_verifier': code_verifier
    }
    
    if config.get('client_secret'):
        token_data['client_secret'] = config['client_secret']
    
    try:
        response = requests.post(
            config['token_url'],
            data=token_data,
            headers={
                'Content-Type': 'application/x-www-form-urlencoded',
                'User-Agent': 'Mozilla/5.0 (Linux; Android 4.1.1; Galaxy Nexus Build/JRO03C) AppleWebKit/535.19 (KHTML, like Gecko) Chrome/18.0.1025.166 Mobile Safari/535.19'
            }
        )
        
        if response.status_code != 200:
            print(f"Error: Token exchange failed with status {response.status_code}")
            print(f"Response: {response.text}")
            sys.exit(1)
        
        token_response = response.json()
        
    except Exception as e:
        print(f"Error: Failed to exchange code for tokens: {e}")
        sys.exit(1)
    
    # Save tokens
    refresh_token = token_response.get('refresh_token')
    access_token = token_response.get('access_token')
    
    if not refresh_token:
        print("Error: No refresh token in response")
        sys.exit(1)
    
    # Save to file
    config_dir = Path.home() / '.hyundai-logger'
    config_dir.mkdir(exist_ok=True)
    
    tokens_file = config_dir / 'tokens.json'
    tokens_data = {
        'access_token': access_token,
        'refresh_token': refresh_token,
        'token_type': token_response.get('token_type', 'Bearer'),
        'expires_in': token_response.get('expires_in', 3600),
        'expires_at': (time.time() + token_response.get('expires_in', 3600)),
        'region': region,
        'brand': brand
    }
    
    with open(tokens_file, 'w') as f:
        json.dump(tokens_data, f, indent=2)
    
    print("\n" + "="*60)
    print("SUCCESS! Tokens obtained and saved!")
    print("="*60)
    print(f"\nRefresh Token: {refresh_token[:20]}...")
    print(f"Access Token: {access_token[:20]}...")
    print(f"\nTokens saved to: {tokens_file}")
    print("\nYou can now use the hyundai-logger with these tokens.")
    print("\nTo use in config.yaml, add:")
    print(f"  refresh_token: \"{refresh_token}\"")
    print("\n" + "="*60)

if __name__ == "__main__":
    if len(sys.argv) != 3:
        print("Usage: python3 get_refresh_token.py <REGION> <BRAND>")
        print("Regions: EU, US, CA")
        print("Brands: hyundai, kia")
        print("\nExample: python3 get_refresh_token.py EU hyundai")
        sys.exit(1)
    
    region = sys.argv[1].upper()
    brand = sys.argv[2].lower()
    
    # Check for required packages
    try:
        import requests
    except ImportError:
        print("Error: requests package is required. Install with: pip3 install requests")
        sys.exit(1)
    
    get_refresh_token(region, brand)
