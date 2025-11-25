# Performance Monitoring and Optimization

This document describes the performance monitoring and optimization features implemented in the Full Stack Master Sync backend.

## Features

### 1. Request/Response Time Monitoring

The backend automatically tracks request and response times for all endpoints:

- **Global Metrics**: Total requests, average response time, min/max response times
- **Endpoint-Specific Metrics**: Per-endpoint performance tracking
- **Real-time Monitoring**: Live performance data collection

#### Accessing Performance Metrics

```bash
# Get overall performance metrics
GET /api/performance/metrics

# Get endpoint-specific metrics
GET /api/performance/endpoint?method=GET&path=/api/sync/status

# Get top endpoints by various metrics
GET /api/performance/top?sort_by=request_count&limit=10
```

### 2. Memory Usage Monitoring

Comprehensive memory usage tracking and optimization:

- **Runtime Memory Stats**: Heap, stack, and GC statistics
- **Memory Profiling**: Detailed memory allocation tracking
- **Garbage Collection**: Manual GC triggering and monitoring

#### Memory Monitoring Endpoints

```bash
# Get detailed memory statistics
GET /api/performance/memory

# Get system information including memory
GET /api/performance/system

# Trigger garbage collection
POST /api/performance/gc
```

### 3. Connection Pooling

Optimized HTTP client connection pooling for external API calls:

- **OpenAI API Pool**: Dedicated connection pool for AI service calls
- **Testing Framework Pool**: Optimized for testing framework interactions
- **Custom Pools**: Support for service-specific connection pools

#### Connection Pool Configuration

```go
// OpenAI optimized pool
pool := utils.OpenAIConnectionPool()

// Custom pool configuration
config := &utils.ConnectionPoolConfig{
    MaxIdleConns:          100,
    MaxIdleConnsPerHost:   10,
    MaxConnsPerHost:       100,
    IdleConnTimeout:       90 * time.Second,
    TLSHandshakeTimeout:   10 * time.Second,
    ResponseHeaderTimeout: 30 * time.Second,
}
pool := utils.NewConnectionPool(config)
```

#### Connection Pool Monitoring

```bash
# Get connection pool statistics
GET /api/performance/pools
```

### 4. Rate Limiting

Configurable rate limiting to prevent abuse and ensure fair usage:

- **Per-IP Rate Limiting**: Default rate limiting by client IP
- **Custom Key Generation**: Support for user-based or custom rate limiting
- **Configurable Limits**: Adjustable requests per second and burst size
- **Path Exclusions**: Skip rate limiting for specific paths

#### Rate Limiting Configuration

```go
rateLimitConfig := middleware.RateLimitConfig{
    RequestsPerSecond: 100, // 100 requests per second
    BurstSize:         20,  // Allow bursts of up to 20 requests
    SkipPaths:         []string{"/health", "/metrics"},
    KeyGenerator: func(c *fiber.Ctx) string {
        return c.Get("X-User-ID", c.IP()) // Rate limit by user ID or IP
    },
}
```

### 5. Performance Benchmarks

Comprehensive benchmarking suite for critical endpoints:

- **Endpoint Benchmarks**: Performance testing for all API endpoints
- **Middleware Benchmarks**: Performance impact of middleware layers
- **Concurrent Load Testing**: Multi-user performance testing

#### Running Benchmarks

```bash
# Run all benchmarks
go test ./benchmarks -bench=. -run=^$

# Run specific benchmark
go test ./benchmarks -bench=BenchmarkHealthEndpoint -run=^$

# Run performance test script
go run scripts/performance_test.go
```

## Performance Optimization Features

### 1. Middleware Optimization

- **Performance Monitoring Middleware**: Minimal overhead request tracking
- **Memory Monitoring Middleware**: Efficient memory usage tracking
- **Connection Pooling Middleware**: Optimized connection management

### 2. Circuit Breaker Pattern

Implemented for external API calls to prevent cascading failures:

- **OpenAI API Protection**: Circuit breaker for AI service calls
- **Configurable Thresholds**: Customizable failure thresholds and timeouts
- **Automatic Recovery**: Self-healing when services recover

### 3. Retry Mechanisms

Intelligent retry logic for transient failures:

- **Exponential Backoff**: Configurable backoff strategies
- **Jitter Support**: Random jitter to prevent thundering herd
- **Conditional Retries**: Retry only on specific error types

### 4. Resource Optimization

- **Memory Pool Management**: Efficient memory allocation and reuse
- **Goroutine Management**: Controlled concurrency to prevent resource exhaustion
- **Connection Reuse**: HTTP connection pooling and keep-alive optimization

## Monitoring Endpoints

### Performance Metrics

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/performance/metrics` | GET | Get overall performance metrics |
| `/api/performance/memory` | GET | Get detailed memory statistics |
| `/api/performance/pools` | GET | Get connection pool statistics |
| `/api/performance/system` | GET | Get comprehensive system information |
| `/api/performance/endpoint` | GET | Get endpoint-specific metrics |
| `/api/performance/top` | GET | Get top endpoints by metrics |
| `/api/performance/health` | GET | Performance monitoring health check |

### Management Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/performance/reset` | POST | Reset all performance metrics |
| `/api/performance/gc` | POST | Trigger garbage collection |

## Performance Metrics

### Global Metrics

```json
{
  "request_count": 1000,
  "total_response_time_ms": 50000,
  "average_response_time_ms": 50.0,
  "min_response_time_ms": 10,
  "max_response_time_ms": 500,
  "active_connections": 5,
  "memory_usage": {
    "alloc_mb": 25.6,
    "sys_mb": 45.2,
    "heap_alloc_mb": 20.1,
    "num_gc": 15
  },
  "last_updated": "2024-01-01T12:00:00Z"
}
```

### Endpoint Metrics

```json
{
  "path": "/api/sync/status",
  "method": "GET",
  "request_count": 100,
  "average_response_time_ms": 25.5,
  "min_response_time_ms": 10,
  "max_response_time_ms": 100,
  "error_count": 2,
  "error_rate": 2.0,
  "last_accessed": "2024-01-01T12:00:00Z"
}
```

### Connection Pool Stats

```json
{
  "openai": {
    "active_connections": 3,
    "idle_connections": 7,
    "total_requests": 500,
    "failed_requests": 5,
    "average_latency_ms": 150.5,
    "last_used": "2024-01-01T12:00:00Z"
  }
}
```

## Best Practices

### 1. Monitoring

- **Regular Metrics Review**: Monitor performance metrics regularly
- **Alert Thresholds**: Set up alerts for performance degradation
- **Trend Analysis**: Track performance trends over time

### 2. Optimization

- **Connection Pool Tuning**: Adjust pool sizes based on usage patterns
- **Rate Limit Adjustment**: Fine-tune rate limits based on capacity
- **Memory Management**: Monitor memory usage and trigger GC when needed

### 3. Testing

- **Load Testing**: Regular load testing to identify bottlenecks
- **Benchmark Comparison**: Compare benchmarks across releases
- **Performance Regression Testing**: Automated performance testing in CI/CD

## Configuration

### Environment Variables

```bash
# Rate limiting
RATE_LIMIT_REQUESTS_PER_SECOND=100
RATE_LIMIT_BURST_SIZE=20

# Connection pooling
CONNECTION_POOL_MAX_IDLE=100
CONNECTION_POOL_MAX_IDLE_PER_HOST=10
CONNECTION_POOL_TIMEOUT=30s

# Performance monitoring
PERFORMANCE_MONITORING_ENABLED=true
MEMORY_MONITORING_ENABLED=true
```

### Application Configuration

```go
// Performance middleware configuration
app.Use(middleware.PerformanceMonitoring())
app.Use(middleware.MemoryMonitoring())
app.Use(middleware.ConnectionPooling())

// Rate limiting configuration
rateLimitConfig := middleware.RateLimitConfig{
    RequestsPerSecond: 100,
    BurstSize:         20,
    SkipPaths:         []string{"/health", "/metrics"},
}
app.Use(middleware.RateLimiting(rateLimitConfig))
```

## Troubleshooting

### High Memory Usage

1. Check memory statistics: `GET /api/performance/memory`
2. Trigger garbage collection: `POST /api/performance/gc`
3. Review memory allocation patterns
4. Consider adjusting connection pool sizes

### High Response Times

1. Check endpoint metrics: `GET /api/performance/endpoint`
2. Review top slow endpoints: `GET /api/performance/top?sort_by=average_response_time`
3. Check connection pool utilization
4. Review external service performance

### Rate Limiting Issues

1. Check rate limit configuration
2. Review client request patterns
3. Consider adjusting rate limits or implementing user-based limiting
4. Monitor rate limit violations in logs

## Future Enhancements

- **Distributed Tracing**: OpenTelemetry integration for distributed tracing
- **Metrics Export**: Prometheus metrics export
- **Performance Dashboards**: Grafana dashboard integration
- **Automated Scaling**: Auto-scaling based on performance metrics
- **Performance Alerts**: Automated alerting for performance issues