#!/bin/bash
# Update Docker Hub repository description and overview
# Requires: DOCKERHUB_USERNAME and DOCKERHUB_TOKEN environment variables

set -e

REPO_NAME="${DOCKER_REPO:-soothill/hyundai-logger}"
DOCKERHUB_REPO=$(echo "$REPO_NAME" | cut -d'/' -f2)
DOCKERHUB_NAMESPACE=$(echo "$REPO_NAME" | cut -d'/' -f1)

# Check for required environment variables
if [ -z "$DOCKERHUB_USERNAME" ] || [ -z "$DOCKERHUB_TOKEN" ]; then
    echo "Error: DOCKERHUB_USERNAME and DOCKERHUB_TOKEN environment variables are required"
    echo ""
    echo "To create a Docker Hub access token:"
    echo "  1. Go to https://hub.docker.com/settings/security"
    echo "  2. Click 'New Access Token'"
    echo "  3. Give it a description (e.g., 'CI/CD')"
    echo "  4. Copy the token"
    echo ""
    echo "Then set:"
    echo "  export DOCKERHUB_USERNAME=your-username"
    echo "  export DOCKERHUB_TOKEN=your-token"
    exit 1
fi

echo "Updating Docker Hub repository: $REPO_NAME"
echo ""

# Read documentation files
if [ -f "DOCKER_HUB.md" ]; then
    FULL_DESCRIPTION=$(cat DOCKER_HUB.md)
    echo "Using DOCKER_HUB.md for full description"
elif [ -f "README.md" ]; then
    FULL_DESCRIPTION=$(cat README.md)
    echo "Using README.md for full description"
else
    echo "Error: No DOCKER_HUB.md or README.md found"
    exit 1
fi

# Short description (first line of README or DOCKER_HUB.md)
SHORT_DESCRIPTION="Hyundai Bluelink vehicle data logger with InfluxDB integration - Multi-arch support (amd64/arm64/armv7)"

# Get authentication token
echo "Authenticating with Docker Hub..."
TOKEN=$(curl -s -H "Content-Type: application/json" \
    -X POST \
    -d "{\"username\": \"${DOCKERHUB_USERNAME}\", \"password\": \"${DOCKERHUB_TOKEN}\"}" \
    https://hub.docker.com/v2/users/login/ | jq -r .token)

if [ -z "$TOKEN" ] || [ "$TOKEN" = "null" ]; then
    echo "Error: Failed to authenticate with Docker Hub"
    exit 1
fi

echo "✓ Authenticated successfully"
echo ""

# Update repository description
echo "Updating repository overview..."
RESPONSE=$(curl -s -w "\n%{http_code}" \
    -H "Authorization: JWT ${TOKEN}" \
    -H "Content-Type: application/json" \
    -X PATCH \
    -d "{\"full_description\": $(echo "$FULL_DESCRIPTION" | jq -Rs .), \"description\": \"$SHORT_DESCRIPTION\"}" \
    "https://hub.docker.com/v2/repositories/${DOCKERHUB_NAMESPACE}/${DOCKERHUB_REPO}/")

HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
RESPONSE_BODY=$(echo "$RESPONSE" | sed '$d')

if [ "$HTTP_CODE" = "200" ]; then
    echo "✓ Docker Hub overview updated successfully"
    echo ""
    echo "View at: https://hub.docker.com/r/${REPO_NAME}"
else
    echo "Error: Failed to update Docker Hub (HTTP $HTTP_CODE)"
    echo "Response: $RESPONSE_BODY"
    exit 1
fi
