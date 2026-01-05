# eCFR Analyzer

A web application for analyzing and visualizing the electronic Code of Federal Regulations (eCFR).

## Setup

1. **Clone and start the application:**
   ```bash
   git clone <repository-url>
   cd ecfr-analyzer
   docker compose up --build
   ```

2. **Import CFR data:**
   ```bash
   ./import_data.sh
   ```

3. **Generate checksums:**
   ```bash
   cd backend
   ./calculate_checksums.sh
   ```

## Usage

- **Frontend**: http://localhost:3002
- **Backend API**: http://localhost:8082

## Features

- Dashboard with agency statistics and visualizations
- Historical trend analysis
- Data integrity verification with checksums
- Agency-specific breakdowns
- Export functionality

## Architecture

- **Frontend**: React with TypeScript
- **Backend**: Go with PostgreSQL
- **Deployment**: Docker Compose

## Development

The application automatically refreshes data every hour from the official eCFR API.
