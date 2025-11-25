# Troubleshooting Guide

## Table of Contents
- [Server Issues](#server-issues)
- [API Issues](#api-issues)
- [WebSocket Issues](#websocket-issues)
- [AI Service Issues](#ai-service-issues)
- [Performance Issues](#performance-issues)
- [Testing Issues](#testing-issues)
- [Configuration Issues](#configuration-issues)

## Server Issues

### Server Won't Start

#### Port Already in Use
**Symptoms:**
```
Server failed to start: listen tcp :8080: bind: address already in use
```

**Diagnosis:**
```bash
# Check what's using the port
lsof -i :8080  # macOS/Linux
netstat -ano | findstr :8080  # Windows
```

**Solutions:**
1. Kill the process using the port
2. Use a different port: `PORT=8081 go run main.go`
3. Check for zombie processes

#### Configuration Validation Failed
**Symptoms:**
```
Configuration validation failed: [PORT is required]
```

**Solutions:**
1. Check `.env` file exists and is properly formatted
2. Verify all required environment variables are set
3. Run with `LOG_LEVEL=debug` for detailed validation errors
4. Use `/debug/config` endpoint (development mode) to inspect configuration

### Server Crashes

#### Panic Recovery
**Symptoms:**
- Server crashes with panic message
- Stack trace in logs

**Diagnosis:**
1. Check error recovery stats: `GET /error-recovery/stats`
2. Review logs for panic details
3. Note the trace ID from the error

**Solutions:**
1. The error recovery service should automatically recover from panics
2. Check if circuit breakers are open: Look for `CIRCUIT_BREAKER_OPEN` errors
3. Review recent code changes
4. Enable detailed error logging: `ENABLE_DETAILED_ERRORS=true`

#### Memory Issues
**Symptoms:**
- Server becomes unresponsive
- High memory usage
- Out of memory errors

**Diagnosis:**
```bash
# Check memory stats
curl http://localhost:8080/api/performance/memory

# Monitor goroutines
curl http://localhost:8080/api/performance/system
```

**Solutions:**
1. Trigger garbage collection: `POST /api/performance/gc`
2. Check for goroutine leaks
3. Review connection pool stats: `GET /api/performance/pools`
4. Restart the server
5. Increase available memory

## API Issues

### 400 Bad Request

#### Validation Errors
**Symptoms:**
```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Validation failed",
    "details": {
      "field_name": "field is required"
    }
  }
}
```

**Solutions:**
1. Check request body matches expected schema
2. Verify all required fields are present
3. Ensure correct data types
4. Review API documentation for endpoint requirements

#### Malformed JSON
**Symptoms:**
```json
{
  "success": false,
  "error": {
    "code": "BAD_REQUEST",
    "message": "Invalid JSON"
  }
}
```

**Solutions:**
1. Validate JSON syntax
2. Check for trailing commas
3. Ensure proper escaping of special characters
4. Use a JSON validator tool

### 429 Too Many Requests

**Symptoms:**
```json
{
  "success": false,
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "Too many requests"
  }
}
```

**Solutions:**
1. Implement request throttling in client
2. Add delays between requests
3. Check rate limit configuration
4. Consider increasing rate limits for your use case
5. Use batch endpoints where available

### 500 Internal Server Error

**Symptoms:**
```json
{
  "success": false,
  "error": {
    "code": "INTERNAL_ERROR",
    "message": "Internal server error",
    "details": {
      "trace_id": "550e8400-e29b-41d4-a716-446655440000"
    }
  }
}
```

**Diagnosis:**
1. Note the `trace_id` from the response
2. Search server logs for the trace ID
3. Check error recovery stats: `GET /error-recovery/stats`

**Solutions:**
1. Report the issue with the trace ID
2. Check server logs for detailed error information
3. Verify external service availability (OpenAI, etc.)
4. Check circuit breaker status

### 503 Service Unavailable

#### Circuit Breaker Open
**Symptoms:**
```json
{
  "success": false,
  "error": {
    "code": "CIRCUIT_BREAKER_OPEN",
    "message": "Service temporarily unavailable",
    "details": {
      "circuit_breaker": "open",
      "retry_after": "60s"
    }
  }
}
```

**Diagnosis:**
- Circuit breaker has opened due to repeated failures
- External service (OpenAI) is experiencing issues

**Solutions:**
1. Wait for the retry_after duration
2. Check external service status
3. Review error recovery stats
4. Circuit breaker will automatically close after cooldown period

#### Retry Exhausted
**Symptoms:**
```json
{
  "success": false,
  "error": {
    "code": "RETRY_EXHAUSTED",
    "message": "Service temporarily unavailable after retry attempts"
  }
}
```

**Solutions:**
1. Check network connectivity
2. Verify external service availability
3. Review retry configuration
4. Wait before retrying

## WebSocket Issues

### Connection Fails

**Symptoms:**
- WebSocket connection cannot be established
- Connection immediately closes

**Diagnosis:**
```bash
# Check WebSocket stats
curl http://localhost:8080/ws/stats
```

**Solutions:**
1. Verify WebSocket endpoint: `ws://localhost:8080/ws`
2. Check CORS configuration
3. Ensure WebSocket is enabled: `ENABLE_WEBSOCKET=true`
4. Review firewall rules
5. Check browser console for detailed errors

### Connection Drops

**Symptoms:**
- WebSocket connection drops frequently
- Reconnection attempts fail

**Diagnosis:**
1. Check WebSocket stats for connection patterns
2. Review server logs for disconnect reasons
3. Monitor network stability

**Solutions:**
1. Implement automatic reconnection with exponential backoff
2. Check network stability
3. Verify server isn't restarting
4. Review connection timeout settings
5. Implement heartbeat/ping-pong mechanism

### Messages Not Received

**Symptoms:**
- WebSocket connected but no messages received
- Messages delayed or lost

**Diagnosis:**
```javascript
// Check WebSocket state
console.log(ws.readyState);
// 0: CONNECTING, 1: OPEN, 2: CLOSING, 3: CLOSED
```

**Solutions:**
1. Verify WebSocket is in OPEN state
2. Check message format matches expected schema
3. Review server logs for broadcast errors
4. Ensure client is subscribed to correct message types
5. Check for message filtering on client side

## AI Service Issues

### AI Service Unavailable

**Symptoms:**
```json
{
  "success": false,
  "error": {
    "code": "SERVICE_UNAVAILABLE",
    "message": "AI service temporarily unavailable"
  }
}
```

**Diagnosis:**
```bash
# Check AI service status
curl http://localhost:8080/api/ai/status
```

**Solutions:**
1. Verify `OPENAI_API_KEY` is set correctly
2. Check OpenAI API status: https://status.openai.com/
3. Verify API key is valid and has credits
4. Check rate limits on OpenAI account
5. Review circuit breaker status

### Invalid API Key

**Symptoms:**
- AI requests fail with authentication errors
- 401 Unauthorized from OpenAI

**Solutions:**
1. Verify API key in `.env` file
2. Check for extra spaces or newlines in API key
3. Generate new API key at https://platform.openai.com/api-keys
4. Ensure API key has proper permissions

### Rate Limit Exceeded (OpenAI)

**Symptoms:**
- AI requests fail with rate limit errors
- 429 responses from OpenAI

**Solutions:**
1. Implement request queuing
2. Add delays between AI requests
3. Upgrade OpenAI plan for higher limits
4. Use caching for repeated requests
5. Implement fallback responses

## Performance Issues

### Slow Response Times

**Symptoms:**
- API requests take longer than expected
- Timeouts occur

**Diagnosis:**
```bash
# Check performance metrics
curl http://localhost:8080/api/performance/metrics

# Check endpoint-specific metrics
curl http://localhost:8080/api/performance/endpoint?path=/api/ai/suggestions

# Check top slow endpoints
curl http://localhost:8080/api/performance/top?metric=avg_duration&limit=10
```

**Solutions:**
1. Review endpoint-specific metrics
2. Check external service response times
3. Monitor database query performance (if applicable)
4. Review connection pool utilization
5. Consider caching frequently accessed data
6. Check for N+1 query problems

### High Memory Usage

**Symptoms:**
- Memory usage continuously increases
- Server becomes sluggish

**Diagnosis:**
```bash
# Check memory stats
curl http://localhost:8080/api/performance/memory

# Check system info
curl http://localhost:8080/api/performance/system
```

**Solutions:**
1. Trigger garbage collection: `POST /api/performance/gc`
2. Check for memory leaks in recent code changes
3. Review goroutine count for leaks
4. Monitor connection pool for unclosed connections
5. Restart server if memory leak is confirmed
6. Use Go profiling tools: `go tool pprof`

### Connection Pool Exhaustion

**Symptoms:**
- Requests fail with connection errors
- "No available connections" errors

**Diagnosis:**
```bash
# Check connection pool stats
curl http://localhost:8080/api/performance/pools
```

**Solutions:**
1. Increase connection pool size
2. Check for connection leaks (connections not being returned)
3. Review connection timeout settings
4. Ensure connections are properly closed after use
5. Monitor active vs idle connections

## Testing Issues

### Tests Fail to Run

**Symptoms:**
- Test execution fails immediately
- Framework not found errors

**Solutions:**
1. Verify test framework is installed (Cypress/Playwright)
2. Check test configuration in `.env`
3. Ensure test files exist in expected locations
4. Review test service status: `GET /api/testing/status`

### Test Results Not Available

**Symptoms:**
- Test run starts but results never appear
- `GET /api/testing/results/:runId` returns not found

**Diagnosis:**
```bash
# Check active test runs
curl http://localhost:8080/api/testing/active

# Check test run history
curl http://localhost:8080/api/testing/history
```

**Solutions:**
1. Verify test run ID is correct
2. Check if test run is still in progress
3. Review test service logs for errors
4. Ensure test framework is properly configured

### Sync Validation Fails

**Symptoms:**
- Sync validation reports issues
- API-UI synchronization problems detected

**Solutions:**
1. Review sync validation results for specific issues
2. Check API endpoint responses match expected schema
3. Verify UI components are using correct API endpoints
4. Update API or UI to resolve synchronization issues
5. Use `/api/sync/validate` to test specific endpoints

## Configuration Issues

### Environment Variables Not Loaded

**Symptoms:**
- Configuration uses default values
- Features don't work as expected

**Solutions:**
1. Verify `.env` file exists in backend directory
2. Check file permissions on `.env`
3. Ensure no syntax errors in `.env` file
4. Restart server after changing `.env`
5. Use absolute paths if relative paths fail

### Feature Toggles Not Working

**Symptoms:**
- Features enabled/disabled incorrectly
- Configuration changes don't take effect

**Solutions:**
1. Verify feature toggle environment variables are set correctly
2. Check boolean values are "true" or "false" (lowercase)
3. Restart server after changing feature toggles
4. Use `/debug/config` to verify current configuration (development mode)

### Invalid Configuration Values

**Symptoms:**
```
Configuration validation failed: [LOG_LEVEL must be one of: debug, info, warn, error]
```

**Solutions:**
1. Review validation error messages
2. Check allowed values in SETUP.md
3. Fix invalid values in `.env`
4. Use default values if unsure
5. Consult configuration documentation

## General Debugging Tips

### Enable Debug Logging

```bash
LOG_LEVEL=debug go run main.go
```

### Use Trace IDs

Every API response includes a `trace_id`. Use it to track requests through logs:

```bash
# Search logs for specific trace ID
grep "550e8400-e29b-41d4-a716-446655440000" logs/*.log
```

### Check Health Endpoints

```bash
# Overall health
curl http://localhost:8080/health

# AI service health
curl http://localhost:8080/api/ai/health

# Testing service health
curl http://localhost:8080/api/testing/health

# Logging service health
curl http://localhost:8080/api/logs/health

# Performance monitoring health
curl http://localhost:8080/api/performance/health
```

### Monitor Error Recovery

```bash
# Check error recovery statistics
curl http://localhost:8080/error-recovery/stats
```

### Review Logs

Logs include structured information for debugging:

```json
{
  "level": "error",
  "timestamp": "2024-01-15T10:30:00Z",
  "message": "Request failed",
  "trace_id": "550e8400-e29b-41d4-a716-446655440000",
  "source": "ai_service",
  "error": "OpenAI API timeout",
  "context": {
    "method": "POST",
    "path": "/api/ai/suggestions",
    "duration": "30s"
  }
}
```

## Getting Help

If you're still experiencing issues:

1. **Collect Information:**
   - Trace ID from error response
   - Relevant log entries
   - Configuration (with sensitive values removed)
   - Steps to reproduce

2. **Check Documentation:**
   - API.md for endpoint details
   - SETUP.md for configuration
   - DEVELOPMENT.md for development guidelines

3. **Report Issue:**
   - Include trace ID
   - Provide error messages
   - Describe expected vs actual behavior
   - Include relevant logs
