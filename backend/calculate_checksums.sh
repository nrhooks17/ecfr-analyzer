#!/bin/bash

# Calculate Agency Checksums Script
# Calls the checksum calculation API endpoint

set -e  # Exit on any error

echo "[INFO] Starting agency checksum calculation via API..."

# Make POST request to calculate checksums endpoint
response=$(curl -s -X POST http://localhost:8082/api/v1/calculate-checksums)

if [ $? -eq 0 ]; then
    echo "[SUCCESS] API call completed"
    echo "[INFO] Response: $response"
    
    # Check if the response indicates success
    if echo "$response" | grep -q '"success":true'; then
        echo "[SUCCESS] All agency checksums calculated successfully!"
    else
        echo "[WARNING] Some agencies may have had errors - check response above"
    fi
else
    echo "[ERROR] Failed to call checksum calculation API"
    exit 1
fi