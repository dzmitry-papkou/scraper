# Fundamentals of AI

A Hacker News scraper with statistical analysis capabilities built in Go.

## Quick Start

```bash
# Start database
./env/db.sh --startdb

# Run CLI
go run cmd/cli/main.go
```

## Features

- **Smart Scraping**: Multiple scraping modes (latest, new posts only, full archive)
- **Statistical Analysis**: Correlation analysis, t-tests, trend analysis
- **Auto-scheduling**: Configurable automatic scraping intervals
- **Data Export**: CSV export with full post history
- **PostgreSQL Storage**: Persistent data with post history tracking
