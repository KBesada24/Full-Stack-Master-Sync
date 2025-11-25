# Full Stack Master Sync - Setup Guide

## Table of Contents
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Configuration](#configuration)
- [Running the Application](#running-the-application)
- [Development Mode](#development-mode)
- [Testing](#testing)
- [Troubleshooting](#troubleshooting)

## Prerequisites

### Required Software
- **Go**: Version 1.21 or higher
  - Download from: https://golang.org/dl/
  - Verify installation: `go version`

- **Git**: For version control
  - Download from: https://git-scm.com/
  - Verify installation: `git --version`

### Optional Software
- **Docker**: For containerized deployment (future)
- **Make**: For build automation (optional)

### External Services
- **OpenAI API Key**: Required for AI features
  - Sign up at: https://platform.openai.com/
  - Create API key at: https://platform.openai.com/api-keys

## Installation

### 1. Clone the Repository
```bash
git clone <repository-url>
cd Full-Stack-Master-Sync/backend
```

### 2. Install Dependencies
```bash
go mod download
```

### 3. Verify Installation
```bash
go mod verify
```

## Configuration

### Environment Variables

Create a `.env` file in the backend directory:

```bash
cp .env.example .env
```

### Required Configuration

Edit `.env` with your settings:

```env
# Server Configuration
PORT=8080
HOST=localhost
ENVIRONMENT=development

# OpenAI Configuration
OPENAI_API_KEY=your_openai_api_key_here

# CORS Configuration
FRONTEND_URL=http://localhost:3000

# WebSocket Configuration
WS_ENDPOINT=/ws

# Logging Configuration
LOG_LEVEL=info
LOG_FORMAT=json

# Testing Configuration
CYPRESS_BASE_URL=http://localhost:3000
PLAYWRIGHT_BASE_URL=http://localhost:3000
```

### Configuration Options

#### Server Configuration
- `PORT`: Server port (default: 8080)
- `HOST`: Server host (default: localhost)
- `ENVIRONMENT`: Environment mode (development, staging, production)

#### Logging Configuration
- `LOG_LEVEL`: Logging level (debug, info, warn, error)
- `LOG_FORMAT`: Log format (json, text)

#### Feature Toggles
- `ENABLE_AI_FEATURES`: Enable/disable AI features (default: true)
- `ENABLE_WEBSOCKET`: Enable/disable WebSocket (default: true)
- `ENABLE_PERFORMANCE_MONITORING`: Enable/disable performance monitoring (default: true)
- `ENABLE_RATE_LIMITING`: Enable/disable rate limiting (default: true)

### Configuration Validation

The application validates configuration on startup. If validation fails, you'll see detailed error messages:

```
Configuration validation failed: [
  "OPENAI_API_KEY is required for AI features",
  "LOG_LEVEL must be one of: debug, info, warn, error"
]
```

## Running the Application

### Development Mode

Start the server in development mode:

```bash
go run main.go
```

You should see:
```
🚀 Server starting on port 8080
📊 Health check available at: http://localhost:8080/health
🔗 API base URL: http://localhost:8080/api
📈 Error recovery stats: http://localhost:8080/error-recovery/stats
```

### Production Mode

Build and run the production binary:

```bash
# Build the binary
go build -o server main.go

# Run the binary
./server
```

### With Custom Configuration

Override environment variables:

```bash
PORT=9000 ENVIRONMENT=production go run main.go
```

## Development Mode

### Features

Development mode (`ENVIRONMENT=development`) enables:

1. **Detailed Error Messages**: Stack traces and detailed error information
2. **CORS Relaxation**: Additional allowed origins for local development
3. **Enhanced Logging**: More verbose logging output
4. **Debug Endpoints**: Additional debugging endpoints

### Debug Endpoints

Available in development mode:

- `GET /debug/config`: View current configuration
- `GET /debug/routes`: List all registered routes
- `GET /debug/env`: View environment variables (sensitive values masked)

### Hot Reload

For automatic reloading during development, use `air`:

```bash
# Install air
go install github.com/cosmtrek/air@latest

# Run with hot reload
air
```

Create `.air.toml` for configuration:
```toml
root = "."
tmp_dir = "tmp"

[build]
  cmd = "go build -o ./tmp/main ."
  bin = "tmp/main"
  include_ext = ["go"]
  exclude_dir = ["tmp", "vendor"]
```

## Testing

### Run All Tests

```bash
go test ./...
```

### Run Tests with Coverage

```bash
go test -cover ./...
```

### Run Tests with Detailed Output

```bash
go test -v ./...
```

### Run Specific Package Tests

```bash
go test ./handlers
go test ./services
go test ./middleware
```

### Run Integration Tests

```bash
go test ./integration_tests/...
```

### Run Benchmarks

```bash
go test -bench=. ./benchmarks
```

### Generate Coverage Report

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

## Troubleshooting

### Common Issues

#### 1. Port Already in Use

**Error:**
```
Server failed to start: listen tcp :8080: bind: address already in use
```

**Solution:**
```bash
# Find process using port 8080
lsof -i :8080  # macOS/Linux
netstat -ano | findstr :8080  # Windows

# Kill the process or use a different port
PORT=8081 go run main.go
```

#### 2. OpenAI API Key Not Set

**Error:**
```
AI service unavailable: OpenAI API key not configured
```

**Solution:**
- Ensure `OPENAI_API_KEY` is set in `.env`
- Verify the API key is valid at https://platform.openai.com/api-keys

#### 3. Module Download Failures

**Error:**
```
go: module lookup failed
```

**Solution:**
```bash
# Clear module cache
go clean -modcache

# Re-download modules
go mod download
```

#### 4. Configuration Validation Errors

**Error:**
```
Configuration validation failed: [...]
```

**Solution:**
- Review error messages for specific issues
- Check `.env` file for missing or invalid values
- Refer to Configuration Options section above

### Debugging Tips

#### Enable Debug Logging

```bash
LOG_LEVEL=debug go run main.go
```

#### View Trace IDs

All API responses include a `trace_id`. Use this for debugging:

```bash
# Search logs for specific trace ID
grep "550e8400-e29b-41d4-a716-446655440000" logs/*.log
```

#### Check Health Status

```bash
curl http://localhost:8080/health
```

#### Monitor Performance

```bash
curl http://localhost:8080/api/performance/metrics
```

### Getting Help

1. **Check Logs**: Review application logs for detailed error information
2. **Trace IDs**: Include trace IDs when reporting issues
3. **Configuration**: Verify configuration with `/debug/config` (development mode)
4. **Documentation**: Consult API.md for endpoint documentation

## Next Steps

After setup:

1. **Verify Installation**: Visit http://localhost:8080/health
2. **Test API**: Try example requests from API.md
3. **Connect Frontend**: Configure frontend to use backend URL
4. **Enable Features**: Configure feature toggles as needed
5. **Review Logs**: Monitor logs for any issues

## Additional Resources

- [API Documentation](API.md)
- [Troubleshooting Guide](TROUBLESHOOTING.md)
- [Development Guide](DEVELOPMENT.md)
- [Performance Guide](PERFORMANCE.md)
