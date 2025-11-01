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

# Short description (Docker Hub has 100 byte limit)
SHORT_DESCRIPTION="Hyundai Bluelink vehicle logger with InfluxDB - Multi-arch (amd64/arm64/armv7)"

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

# Verify repository exists and user has access
echo "Checking repository access..."
REPO_CHECK=$(curl -s -w "\n%{http_code}" \
    -H "Authorization: JWT ${TOKEN}" \
    "https://hub.docker.com/v2/repositories/${DOCKERHUB_NAMESPACE}/${DOCKERHUB_REPO}/")

REPO_CHECK_CODE=$(echo "$REPO_CHECK" | tail -n1)
REPO_CHECK_BODY=$(echo "$REPO_CHECK" | sed '$d')

if [ "$REPO_CHECK_CODE" = "404" ]; then
    echo ""
    echo "❌ Error: Repository not found"
    echo ""
    echo "The repository '${REPO_NAME}' does not exist on Docker Hub."
    echo ""
    echo "Please either:"
    echo "  1. Create the repository first by pushing an image:"
    echo "     make docker-build-multiarch"
    echo "     make docker-push"
    echo ""
    echo "  2. Or create it manually at:"
    echo "     https://hub.docker.com/repository/create"
    echo ""
    exit 1
elif [ "$REPO_CHECK_CODE" = "403" ]; then
    echo ""
    echo "❌ Error: Access forbidden"
    echo ""
    echo "Your token does not have permission to access '${REPO_NAME}'."
    echo ""
    echo "Common issues:"
    echo "  1. The repository namespace (${DOCKERHUB_NAMESPACE}) must match your username (${DOCKERHUB_USERNAME})"
    echo "  2. The access token must have 'Read & Write' permissions"
    echo "  3. If using an organization, ensure your user has write access"
    echo ""
    echo "To fix:"
    echo "  1. Verify DOCKER_REPO matches your Docker Hub namespace:"
    echo "     export DOCKER_REPO=${DOCKERHUB_USERNAME}/hyundai-logger"
    echo ""
    echo "  2. Create a new token with correct permissions:"
    echo "     https://hub.docker.com/settings/security"
    echo "     - Select 'Read & Write' or 'Read, Write & Delete'"
    echo ""
    exit 1
elif [ "$REPO_CHECK_CODE" != "200" ]; then
    echo "Warning: Unexpected response (HTTP $REPO_CHECK_CODE)"
    echo "Response: $REPO_CHECK_BODY"
    echo ""
    echo "Attempting to update anyway..."
    echo ""
fi

if [ "$REPO_CHECK_CODE" = "200" ]; then
    echo "✓ Repository access verified"
    echo ""
fi

# Update repository description
# Note: Docker Hub API requires updating description and full_description separately
echo "Updating short description..."
SHORT_RESPONSE=$(curl -s -w "\n%{http_code}" \
    -H "Authorization: JWT ${TOKEN}" \
    -H "Content-Type: application/json" \
    -X PATCH \
    -d "{\"description\": \"$SHORT_DESCRIPTION\"}" \
    "https://hub.docker.com/v2/repositories/${DOCKERHUB_NAMESPACE}/${DOCKERHUB_REPO}/")

SHORT_HTTP_CODE=$(echo "$SHORT_RESPONSE" | tail -n1)
SHORT_RESPONSE_BODY=$(echo "$SHORT_RESPONSE" | sed '$d')

if [ "$SHORT_HTTP_CODE" = "200" ]; then
    echo "✓ Short description updated"
else
    echo "⚠️  Warning: Failed to update short description (HTTP $SHORT_HTTP_CODE)"
    echo "Response: $SHORT_RESPONSE_BODY"
fi

echo ""
echo "Updating full description (overview)..."
RESPONSE=$(curl -s -w "\n%{http_code}" \
    -H "Authorization: JWT ${TOKEN}" \
    -H "Content-Type: application/json" \
    -X PATCH \
    -d "{\"full_description\": $(echo "$FULL_DESCRIPTION" | jq -Rs .)}" \
    "https://hub.docker.com/v2/repositories/${DOCKERHUB_NAMESPACE}/${DOCKERHUB_REPO}/")

HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
RESPONSE_BODY=$(echo "$RESPONSE" | sed '$d')

if [ "$HTTP_CODE" = "200" ]; then
    echo "✓ Full description updated successfully"
    echo ""
    echo "View at: https://hub.docker.com/r/${REPO_NAME}"
else
    echo ""
    echo "❌ Error: Failed to update full description (HTTP $HTTP_CODE)"
    echo ""
    echo "Response: $RESPONSE_BODY"
    echo ""
    if [ "$HTTP_CODE" = "403" ]; then
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo "Permission Denied - Token Issues"
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo ""
        echo "Your token can READ the repository but cannot WRITE to it."
        echo ""
        echo "Common causes:"
        echo "  1. Token created with 'Public Repo Read Only' permissions"
        echo "  2. Token missing 'Read, Write, Delete' scope"
        echo "  3. Using a password instead of a Personal Access Token"
        echo ""
        echo "Solution - Create a new token with correct permissions:"
        echo "  1. Go to: https://hub.docker.com/settings/security"
        echo "  2. Click 'New Access Token'"
        echo "  3. Description: 'Repository Updates' or similar"
        echo "  4. Access permissions: Select 'Read, Write, Delete'"
        echo "     (Or at minimum 'Read & Write')"
        echo "  5. Copy the token immediately (you won't see it again)"
        echo ""
        echo "Then update your environment:"
        echo "  export DOCKERHUB_TOKEN=dckr_pat_XXXXXXXXXXXXXXXXX"
        echo ""
        echo "Alternative - Manual update via Docker Hub UI:"
        echo "  1. Go to: https://hub.docker.com/repository/docker/${REPO_NAME}/general"
        echo "  2. Copy content from DOCKER_HUB.md"
        echo "  3. Paste into 'Repository overview' section"
        echo ""
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo ""
    fi
    exit 1
fi
