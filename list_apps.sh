#!/bin/bash

# Script to list all DigitalOcean apps
# Usage: ./list_apps.sh

if [ -z "$DIGITALOCEAN_API_TOKEN" ]; then
    echo "Error: DIGITALOCEAN_API_TOKEN environment variable is not set"
    echo ""
    echo "Please set it with:"
    echo "  export DIGITALOCEAN_API_TOKEN=your_token_here"
    echo ""
    echo "You can get your API token from: https://cloud.digitalocean.com/account/api/tokens"
    exit 1
fi

echo "Fetching your DigitalOcean apps..."
echo ""

# Fetch apps from DigitalOcean API
response=$(curl -s -X GET \
    -H "Authorization: Bearer $DIGITALOCEAN_API_TOKEN" \
    -H "Content-Type: application/json" \
    "https://api.digitalocean.com/v2/apps?per_page=200")

# Check if request was successful
http_code=$(curl -s -o /dev/null -w "%{http_code}" -X GET \
    -H "Authorization: Bearer $DIGITALOCEAN_API_TOKEN" \
    -H "Content-Type: application/json" \
    "https://api.digitalocean.com/v2/apps?per_page=200")

if [ "$http_code" != "200" ]; then
    echo "Error: Failed to fetch apps (HTTP $http_code)"
    echo "Response: $response"
    exit 1
fi

# Check if we have Python for pretty printing
if command -v python3 &> /dev/null; then
    echo "$response" | python3 -m json.tool
else
    echo "$response"
fi
