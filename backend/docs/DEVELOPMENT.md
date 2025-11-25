# Development Guide

## Table of Contents
- [Development Environment](#development-environment)
- [Project Structure](#project-structure)
- [Coding Standards](#coding-standards)
- [Development Workflow](#development-workflow)
- [Testing Guidelines](#testing-guidelines)
- [Debugging](#debugging)
- [Performance Optimization](#performance-optimization)
- [Contributing](#contributing)

## Development Environment

### Setup

1. **Install Go 1.21+**
   ```bash
   go version  # Verify installation
   ```

2. **Install Development Tools**
   ```bash
   # Air for hot reload
   go install github.com/cosmtrek/air@latest
   
   # golangci-lint for linting
   go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
   
   # godoc for documentation
   go install golang.org/x/tools/cmd/godoc@latest
   ```

3. **Configure IDE**
   - Install Go extension for VS Code or GoLand
   - Enable format on save
   - Configure linter integration

### Development Mode

Run in development mode for enhanced debugging:

```bash
ENVIRONMENT=development LOG_LEVEL=debug go run main.go
```

Development mode enables:
- Detailed error messages with stack traces
- Additional CORS origins for local development
- Debug endpoints
- Verbose logging

### Hot Reload

Use Air for automatic reloading during development:

```bash
# Create .air.toml configuration
air init

# Run with hot reload
air
```

## Project Structure

```
backend/
├── main.go                 # Application entry point
├── config/                 # Configuration management
│   └── config.go
├── handlers/               # HTTP request handlers
│   ├── ai.go
│   ├── sync.go
│   ├── testing.go
│   └── logging.go
├── services/               # Business logic layer
│   ├── ai_service.go
│   ├── sync_service.go
│   ├── test_service.go
│   └── log_service.go
├── middleware/             # HTTP middleware
│   ├── cors.go
│   ├── validation.go
│   ├── logging.go
│   └── auth.go
├── models/                 # Data models
│   ├── ai.go
│   ├── sync.go
│   ├── testing.go
│   └── common.go
├── websocket/              # WebSocket infrastructure
│   ├── hub.go
│   ├── client.go
│   └── handlers.go
├── utils/                  # Utility functions
│   ├── response.go
│   ├── validation.go
│   ├── logger.go
│   ├── circuit_breaker.go
│   ├── retry.go
│   └── error_recovery.go
├── integration_tests/      # Integration tests
├── benchmarks/             # Performance benchmarks
└── docs/                   # Documentation
```

### Layer Responsibilities

**Handlers:**
- Parse HTTP requests
- Validate input
- Call service layer
- Format responses
- Handle HTTP-specific concerns

**Services:**
- Implement business logic
- Coordinate between components
- Handle external API calls
- Manage state and data

**Middleware:**
- Cross-cutting concerns
- Request/response processing
- Authentication/authorization
- Logging and monitoring

**Models:**
- Data structures
- Validation rules
- JSON serialization
- Type definitions

**Utils:**
- Reusable helper functions
- Common utilities
- Error handling
- Logging infrastructure

## Coding Standards

### Go Style Guide

Follow the official Go style guide and best practices:

1. **Formatting:**
   ```bash
   # Format code
   go fmt ./...
   
   # Or use gofmt directly
   gofmt -w .
   ```

2. **Naming Conventions:**
   - Use camelCase for variables and functions
   - Use PascalCase for exported types and functions
   - Use descriptive names
   - Avoid abbreviations unless widely known

3. **Error Handling:**
   ```go
   // Good: Check errors immediately
   result, err := someFunction()
   if err != nil {
       return fmt.Errorf("failed to do something: %w", err)
   }
   
   // Bad: Ignoring errors
   result, _ := someFunction()
   ```

4. **Comments:**
   ```go
   // Good: Document exported functions
   // GetUserByID retrieves a user by their unique identifier.
   // Returns an error if the user is not found.
   func GetUserByID(id string) (*User, error) {
       // ...
   }
   ```

### Code Organization

1. **Package Structure:**
   - One package per directory
   - Keep packages focused and cohesive
   - Avoid circular dependencies

2. **File Organization:**
   - Group related functions together
   - Keep files under 500 lines
   - Use separate files for tests

3. **Function Size:**
   - Keep functions small and focused
   - Extract complex logic into helper functions
   - Aim for single responsibility

### Linting

Run linters before committing:

```bash
# Run golangci-lint
golangci-lint run

# Run with auto-fix
golangci-lint run --fix
```

## Development Workflow

### Feature Development

1. **Create Feature Branch:**
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Implement Feature:**
   - Write code following standards
   - Add tests for new functionality
   - Update documentation

3. **Test Locally:**
   ```bash
   # Run tests
   go test ./...
   
   # Run with coverage
   go test -cover ./...
   
   # Run specific tests
   go test -v ./handlers -run TestAIHandler
   ```

4. **Commit Changes:**
   ```bash
   git add .
   git commit -m "feat: add new feature"
   ```

5. **Push and Create PR:**
   ```bash
   git push origin feature/your-feature-name
   ```

### Commit Messages

Follow conventional commits format:

```
<type>(<scope>): <subject>

<body>

<footer>
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes
- `refactor`: Code refactoring
- `test`: Test changes
- `chore`: Build/tooling changes

Examples:
```
feat(ai): add code suggestion caching

Implement caching for AI code suggestions to reduce API calls
and improve response times.

Closes #123
```

## Testing Guidelines

### Unit Tests

Write unit tests for all new code:

```go
func TestAIService_GetSuggestions(t *testing.T) {
    // Arrange
    service := NewAIService(mockConfig, mockHub, mockLogger)
    request := &AIRequest{
        Code:     "function test() {}",
        Language: "javascript",
    }
    
    // Act
    response, err := service.GetSuggestions(request)
    
    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, response)
    assert.Greater(t, len(response.Suggestions), 0)
}
```

### Table-Driven Tests

Use table-driven tests for multiple scenarios:

```go
func TestValidateEmail(t *testing.T) {
    tests := []struct {
        name    string
        email   string
        wantErr bool
    }{
        {"valid email", "user@example.com", false},
        {"invalid email", "invalid", true},
        {"empty email", "", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateEmail(tt.email)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidateEmail() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### Integration Tests

Write integration tests for complete workflows:

```go
func TestAIWorkflow(t *testing.T) {
    // Setup test server
    app := setupTestApp()
    
    // Test request
    req := httptest.NewRequest("POST", "/api/ai/suggestions", body)
    resp, err := app.Test(req)
    
    // Assertions
    assert.NoError(t, err)
    assert.Equal(t, 200, resp.StatusCode)
}
```

### Test Coverage

Maintain high test coverage:

```bash
# Generate coverage report
go test -coverprofile=coverage.out ./...

# View coverage in browser
go tool cover -html=coverage.out

# Check coverage percentage
go test -cover ./...
```

Target: 80%+ code coverage

### Mocking

Use interfaces for mocking external dependencies:

```go
// Define interface
type AIServiceInterface interface {
    GetSuggestions(req *AIRequest) (*AIResponse, error)
}

// Mock implementation
type MockAIService struct {
    mock.Mock
}

func (m *MockAIService) GetSuggestions(req *AIRequest) (*AIResponse, error) {
    args := m.Called(req)
    return args.Get(0).(*AIResponse), args.Error(1)
}
```

## Debugging

### Debug Logging

Enable debug logging for detailed output:

```bash
LOG_LEVEL=debug go run main.go
```

### Debug Endpoints

Available in development mode:

```bash
# View current configuration
curl http://localhost:8080/debug/config

# List all routes
curl http://localhost:8080/debug/routes

# View environment variables (sensitive values masked)
curl http://localhost:8080/debug/env
```

### Trace IDs

Use trace IDs to track requests:

```go
traceID := utils.GetTraceID(c)
logger.WithTraceID(traceID).Info("Processing request", map[string]interface{}{
    "method": c.Method(),
    "path":   c.Path(),
})
```

### Delve Debugger

Use Delve for interactive debugging:

```bash
# Install Delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Run with debugger
dlv debug main.go

# Set breakpoint
(dlv) break main.main
(dlv) continue
```

### Profiling

Profile application performance:

```bash
# CPU profiling
go test -cpuprofile=cpu.prof -bench=.
go tool pprof cpu.prof

# Memory profiling
go test -memprofile=mem.prof -bench=.
go tool pprof mem.prof

# Web interface
go tool pprof -http=:8081 cpu.prof
```

## Performance Optimization

### Benchmarking

Write benchmarks for performance-critical code:

```go
func BenchmarkAIService_GetSuggestions(b *testing.B) {
    service := NewAIService(config, hub, logger)
    request := &AIRequest{
        Code:     "function test() {}",
        Language: "javascript",
    }
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = service.GetSuggestions(request)
    }
}
```

Run benchmarks:

```bash
# Run all benchmarks
go test -bench=. ./...

# Run specific benchmark
go test -bench=BenchmarkAIService ./services

# With memory allocation stats
go test -bench=. -benchmem ./...
```

### Performance Monitoring

Monitor performance in production:

```bash
# Check performance metrics
curl http://localhost:8080/api/performance/metrics

# Check memory usage
curl http://localhost:8080/api/performance/memory

# Check connection pools
curl http://localhost:8080/api/performance/pools
```

### Optimization Tips

1. **Use Connection Pooling:**
   - Reuse HTTP connections
   - Configure appropriate pool sizes
   - Monitor pool utilization

2. **Implement Caching:**
   - Cache frequently accessed data
   - Use appropriate TTLs
   - Implement cache invalidation

3. **Optimize Database Queries:**
   - Use indexes appropriately
   - Avoid N+1 queries
   - Use batch operations

4. **Reduce Allocations:**
   - Reuse buffers
   - Use sync.Pool for temporary objects
   - Avoid unnecessary string concatenation

5. **Use Goroutines Wisely:**
   - Don't create unlimited goroutines
   - Use worker pools for bounded concurrency
   - Always clean up goroutines

## Contributing

### Before Submitting

1. **Run Tests:**
   ```bash
   go test ./...
   ```

2. **Run Linters:**
   ```bash
   golangci-lint run
   ```

3. **Format Code:**
   ```bash
   go fmt ./...
   ```

4. **Update Documentation:**
   - Update API.md for API changes
   - Update README.md for setup changes
   - Add inline code comments

5. **Test Locally:**
   - Test all affected endpoints
   - Verify error handling
   - Check performance impact

### Pull Request Checklist

- [ ] Tests added/updated
- [ ] Documentation updated
- [ ] Code formatted and linted
- [ ] No breaking changes (or documented)
- [ ] Performance impact considered
- [ ] Error handling implemented
- [ ] Logging added for debugging

### Code Review

When reviewing code:

1. **Functionality:**
   - Does it work as intended?
   - Are edge cases handled?
   - Is error handling appropriate?

2. **Code Quality:**
   - Is it readable and maintainable?
   - Does it follow coding standards?
   - Are there any code smells?

3. **Testing:**
   - Are tests comprehensive?
   - Do tests cover edge cases?
   - Is test coverage adequate?

4. **Performance:**
   - Are there performance concerns?
   - Could it be optimized?
   - Does it scale well?

5. **Security:**
   - Are inputs validated?
   - Are there security vulnerabilities?
   - Is sensitive data protected?

## Best Practices

### Error Handling

```go
// Wrap errors with context
if err != nil {
    return fmt.Errorf("failed to process request: %w", err)
}

// Use custom error types for specific errors
type ValidationError struct {
    Field   string
    Message string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation error on field %s: %s", e.Field, e.Message)
}
```

### Logging

```go
// Use structured logging
logger.Info("Request processed", map[string]interface{}{
    "method":   "POST",
    "path":     "/api/ai/suggestions",
    "duration": duration,
    "status":   200,
})

// Include trace IDs
logger.WithTraceID(traceID).Error("Request failed", err, map[string]interface{}{
    "method": "POST",
    "path":   "/api/ai/suggestions",
})
```

### Context Usage

```go
// Pass context through call chain
func ProcessRequest(ctx context.Context, req *Request) error {
    // Check for cancellation
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }
    
    // Pass context to downstream calls
    return service.DoSomething(ctx, req)
}
```

### Resource Cleanup

```go
// Always defer cleanup
file, err := os.Open("file.txt")
if err != nil {
    return err
}
defer file.Close()

// Use context with timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
```

## Additional Resources

- [Effective Go](https://golang.org/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Go Proverbs](https://go-proverbs.github.io/)
- [API Documentation](API.md)
- [Setup Guide](SETUP.md)
- [Troubleshooting Guide](TROUBLESHOOTING.md)
