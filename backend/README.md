# Full Stack Master Sync - Backend

A high-performance Go Fiber backend providing AI-enhanced code assistance, real-time synchronization, automated testing, and intelligent debugging capabilities.

## 🚀 Quick Start

### Prerequisites

- Go 1.21 or higher
- OpenAI API key (for AI features)

### Installation

1. **Clone and navigate to backend:**
   ```bash
   cd Full-Stack-Master-Sync/backend
   ```

2. **Install dependencies:**
   ```bash
   go mod download
   ```

3. **Configure environment:**
   ```bash
   cp .env.example .env
   ```
   
   Edit `.env` and set your `OPENAI_API_KEY`

4. **Run the server:**
   ```bash
   go run main.go
   ```

The server will start on `http://localhost:8080`

## 📚 Documentation

Comprehensive documentation is available in the `docs/` directory:

- **[API Documentation](docs/API.md)** - Complete API reference with examples
- **[Setup Guide](docs/SETUP.md)** - Detailed installation and configuration
- **[Development Guide](docs/DEVELOPMENT.md)** - Development workflow and best practices
- **[Troubleshooting Guide](docs/TROUBLESHOOTING.md)** - Common issues and solutions
- **[Performance Guide](docs/PERFORMANCE.md)** - Performance optimization tips

## 🎯 Features

### AI-Powered Assistance
- Code suggestions and improvements
- Intelligent log analysis
- Automated debugging recommendations

### Real-Time Synchronization
- WebSocket-based real-time updates
- Environment synchronization
- Live status monitoring

### Automated Testing
- Cypress and Playwright integration
- End-to-end test orchestration
- Sync validation testing

### Performance Monitoring
- Request/response metrics
- Memory usage tracking
- Connection pool monitoring
- Performance benchmarking

### Developer Experience
- Comprehensive error messages
- Debug endpoints (development mode)
- Feature toggles
- Structured logging

## 🔧 Configuration

### Environment Variables

Key configuration options:

```env
# Server
PORT=8080
ENVIRONMENT=development

# OpenAI (required for AI features)
OPENAI_API_KEY=your_key_here

# Logging
LOG_LEVEL=info          # debug, info, warn, error
LOG_FORMAT=json         # json, text

# Feature Toggles
ENABLE_AI_FEATURES=true
ENABLE_WEBSOCKET=true
ENABLE_PERFORMANCE_MONITORING=true
ENABLE_RATE_LIMITING=true
ENABLE_DEBUG_ENDPOINTS=false
```

See [Setup Guide](docs/SETUP.md) for complete configuration options.

## 🧪 Testing

### Run All Tests
```bash
go test ./...
```

### Run with Coverage
```bash
go test -cover ./...
```

### Run Integration Tests
```bash
go test ./integration_tests/...
```

### Run Benchmarks
```bash
go test -bench=. ./benchmarks
```

## 📡 API Endpoints

### Core Endpoints

- `GET /health` - Health check
- `GET /api` - API information
- `WS /ws` - WebSocket connection

### AI Assistance

- `POST /api/ai/suggestions` - Get code suggestions
- `POST /api/ai/analyze-logs` - Analyze logs with AI
- `GET /api/ai/status` - AI service status

### Synchronization

- `POST /api/sync/connect` - Connect environments
- `GET /api/sync/status` - Get sync status
- `POST /api/sync/validate` - Validate endpoints

### Testing

- `POST /api/testing/run` - Run tests
- `GET /api/testing/results/:runId` - Get test results
- `POST /api/testing/validate-sync` - Validate synchronization

### Logging

- `POST /api/logs/submit` - Submit logs
- `GET /api/logs/analyze` - Analyze logs
- `GET /api/logs/stats` - Log statistics

### Performance

- `GET /api/performance/metrics` - Performance metrics
- `GET /api/performance/memory` - Memory statistics
- `GET /api/performance/pools` - Connection pool stats

### Debug (Development Only)

- `GET /debug/config` - View configuration
- `GET /debug/routes` - List all routes
- `GET /debug/env` - Environment variables
- `GET /debug/features` - Feature toggle status

See [API Documentation](docs/API.md) for complete endpoint reference.

## 🔍 Development Mode

Enable development features:

```bash
ENVIRONMENT=development \
LOG_LEVEL=debug \
ENABLE_DEBUG_ENDPOINTS=true \
ENABLE_DETAILED_ERRORS=true \
go run main.go
```

Development mode enables:
- Detailed error messages with stack traces
- Debug endpoints
- Verbose logging
- Additional CORS origins

## 🐛 Debugging

### View Configuration
```bash
curl http://localhost:8080/debug/config
```

### Check Health
```bash
curl http://localhost:8080/health
```

### Monitor Performance
```bash
curl http://localhost:8080/api/performance/metrics
```

### View Logs
Enable debug logging:
```bash
LOG_LEVEL=debug go run main.go
```

## 🚦 Feature Toggles

Control features via environment variables:

| Feature | Variable | Default | Description |
|---------|----------|---------|-------------|
| AI Features | `ENABLE_AI_FEATURES` | `true` | Code suggestions and log analysis |
| WebSocket | `ENABLE_WEBSOCKET` | `true` | Real-time connections |
| Performance Monitoring | `ENABLE_PERFORMANCE_MONITORING` | `true` | Metrics collection |
| Rate Limiting | `ENABLE_RATE_LIMITING` | `true` | API rate limiting |
| Circuit Breaker | `ENABLE_CIRCUIT_BREAKER` | `true` | External service protection |
| Detailed Errors | `ENABLE_DETAILED_ERRORS` | `false` | Stack traces in errors |
| Debug Endpoints | `ENABLE_DEBUG_ENDPOINTS` | `false` | Debug API endpoints |

## 📊 Performance

The backend is optimized for high performance:

- **Fiber Framework**: Fast HTTP routing
- **Connection Pooling**: Efficient resource usage
- **Circuit Breaker**: Resilient external calls
- **Rate Limiting**: Protection against abuse
- **Memory Monitoring**: Automatic GC optimization

See [Performance Guide](docs/PERFORMANCE.md) for optimization tips.

## 🔒 Error Handling

Robust error handling with:

- Automatic panic recovery
- Circuit breaker for external services
- Retry mechanisms with exponential backoff
- Detailed error logging with trace IDs
- Graceful degradation

## 🤝 Contributing

1. Follow the [Development Guide](docs/DEVELOPMENT.md)
2. Write tests for new features
3. Update documentation
4. Run linters: `golangci-lint run`
5. Ensure tests pass: `go test ./...`

## 📝 Project Structure

```
backend/
├── main.go              # Application entry point
├── config/              # Configuration management
├── handlers/            # HTTP request handlers
├── services/            # Business logic
├── middleware/          # HTTP middleware
├── models/              # Data models
├── websocket/           # WebSocket infrastructure
├── utils/               # Utility functions
├── integration_tests/   # Integration tests
├── benchmarks/          # Performance benchmarks
└── docs/                # Documentation
```

## 🆘 Troubleshooting

Common issues and solutions:

### Port Already in Use
```bash
PORT=8081 go run main.go
```

### OpenAI API Key Not Set
Ensure `OPENAI_API_KEY` is set in `.env`

### Configuration Validation Failed
Check `.env` for missing or invalid values

See [Troubleshooting Guide](docs/TROUBLESHOOTING.md) for more solutions.

## 📖 Additional Resources

- [Go Documentation](https://golang.org/doc/)
- [Fiber Documentation](https://docs.gofiber.io/)
- [OpenAI API Reference](https://platform.openai.com/docs/api-reference)

## 📄 License

[Your License Here]

## 🙏 Acknowledgments

Built with:
- [Fiber](https://gofiber.io/) - Web framework
- [OpenAI](https://openai.com/) - AI capabilities
- [WebSocket](https://github.com/gofiber/websocket) - Real-time communication