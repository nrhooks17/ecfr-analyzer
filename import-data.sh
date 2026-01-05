#!/bin/bash

# eCFR Data Import Script
echo "Starting eCFR data import..."

BASE_URL="http://localhost:8082/api/v1"

echo "1. Importing agencies (with CFR references)..."
curl -X POST "$BASE_URL/import/agencies"
echo

echo "2. Importing titles (with content - this may take a while)..."
curl -X POST "$BASE_URL/import/titles"
echo

echo "3. Creating historical snapshots..."
curl -X POST "$BASE_URL/import/historical-snapshots"
echo

echo "Data import completed!"
echo "You can check the status at: $BASE_URL/status"