# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Paddy Cap is a Go web application that aggregates orders from multiple e-commerce platforms (WooCommerce and Orderspace) and provides a unified interface for viewing and managing them. The application fetches orders from both systems concurrently and displays them in a web interface with HTML templates.

## Development Commands

### Build and Run
- `go run cmd/main.go` - Run the application
- `go build -o bin/paddy-cap cmd/main.go` - Build the application
- `go mod tidy` - Clean up dependencies
- `go mod download` - Download dependencies

### Code Quality
- `go fmt ./...` - Format code
- `go vet ./...` - Static analysis for potential issues
- `go test ./...` - Run tests (no tests currently exist)

### Database
- `docker-compose up -d` - Start PostgreSQL database
- `sqlc generate` - Generate Go code from SQL queries (when db/queries/*.sql files exist)

### Environment Setup
- Copy `.env.example` to `.env` and configure environment variables
- `export $(cat .env | xargs)` - Export .env variables to environment

## Architecture

### Core Components

1. **Main Application** (`cmd/main.go`)
   - Entry point that initializes services and starts HTTP server
   - Handles environment configuration and dependency injection

2. **Server Layer** (`server/`)
   - `server.go` - HTTP server setup with middleware chain
   - `routes.go` - Route handlers for orders and health endpoints
   - `renderer.go` - HTML template rendering system with custom functions
   - `config.go` - Server configuration

3. **Service Layer** (`service/`)
   - `order/service.go` - Order aggregation service that unifies data from multiple sources
   - `orderspace/` - Orderspace API client and data models
   - `woocommerce/` - WooCommerce API client and data models

4. **Middleware** (`middleware/middleware.go`)
   - Request logging, CORS, and request ID handling

5. **Templates** (`views/`)
   - Layout-based template system with partials
   - Organized into `layout/`, `page/`, and `partials/` directories

### Key Patterns

- **Service-oriented architecture** with clear separation of concerns
- **Concurrent data fetching** using goroutines for improved performance
- **Unified data models** that normalize data from different APIs
- **Template inheritance** with base layouts and reusable partials
- **Environment-driven configuration** with graceful fallbacks

### Data Flow

1. Routes in `server/routes.go` handle HTTP requests
2. Order service fetches data concurrently from WooCommerce and Orderspace APIs
3. Data is transformed into unified `Order` structs
4. Results are sorted by date and rendered using HTML templates or returned as JSON

### API Endpoints

- `GET /` - Home page
- `GET /healthz` - Health check
- `GET /orders` - List all orders (supports JSON via Content-Type header)
- `GET /orders/{origin}/{id}` - Get specific order details (origin: "woocommerce" or "orderspace")

## Environment Variables Required

- `ORDERSPACE_BASE_URL`, `ORDERSPACE_CLIENT_ID`, `ORDERSPACE_CLIENT_SECRET`
- `WOO_BASE_URL`, `WOO_CONSUMER_KEY`, `WOO_CONSUMER_SECRET`
- `DB_CONNECTION_STRING` (optional, for future database integration)
- `HOST` (default: localhost), `PORT` (default: 8080)

## Database Integration

The project is configured for PostgreSQL using sqlc for type-safe SQL generation:
- `sqlc.yaml` - Configuration for SQL code generation
- `docker-compose.yaml` - PostgreSQL setup
- Database queries should be placed in `db/queries/` when implemented
- Migrations should go in `db/migrations/`

## Template System

Templates use Go's html/template with custom functions:
- `even` - Check if number is even
- `subtract` - Subtract two floats
- `subtractFloat` - Complex subtraction from strings
- `divideFloat` - Divide string by integer
- `title` - Title case conversion