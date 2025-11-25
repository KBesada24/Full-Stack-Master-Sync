# Full Stack Master Sync - API Documentation

## Table of Contents
- [Overview](#overview)
- [Authentication](#authentication)
- [Base URL](#base-url)
- [Response Format](#response-format)
- [Error Handling](#error-handling)
- [Rate Limiting](#rate-limiting)
- [API Endpoints](#api-endpoints)

## Overview

The Full Stack Master Sync Backend API provides AI-enhanced code assistance, real-time synchronization, automated testing, and intelligent debugging capabilities for full-stack development.

**Version:** 1.0.0  
**Base URL:** `http://localhost:8080`

## Authentication

Currently, the API does not require authentication. Future versions will implement API key-based authentication.

## Base URL

Development: `http://localhost:8080`  
Production: Configure via `HOST` and `PORT` environment variables

## Response Format

All API responses follow a standard format:

### Success Response
```json
{
  "success": true,
  "message": "Operation completed successfully",
  "data": { ... },
  "timestamp": "2024-01-15T10:30:00Z",
  "trace_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

### Error Response
```json
{
  "success": false,
  "message": "Request failed",
  "error": {
    "code": "ERROR_CODE",
    "message": "Detailed error message",
    "details": {
      "field": "Additional context"
    }
  },
  "timestamp": "2024-01-15T10:30:00Z",
  "trace_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

## Error Handling

### Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `VALIDATION_ERROR` | 400 | Request validation failed |
| `BAD_REQUEST` | 400 | Malformed request |
| `UNAUTHORIZED` | 401 | Authentication required |
| `FORBIDDEN` | 403 | Insufficient permissions |
| `NOT_FOUND` | 404 | Resource not found |
| `INTERNAL_ERROR` | 500 | Internal server error |
| `SERVICE_UNAVAILABLE` | 503 | External service unavailable |
| `CIRCUIT_BREAKER_OPEN` | 503 | Circuit breaker activated |
| `RETRY_EXHAUSTED` | 503 | Retry attempts exhausted |

### Trace IDs

Every response includes a `trace_id` for debugging. Include this ID when reporting issues.

## Rate Limiting

- **Rate:** 100 requests per second
- **Burst:** 20 requests
- **Excluded paths:** `/health`, `/metrics`, `/ws`

When rate limited, you'll receive a `429 Too Many Requests` response.

---

## API Endpoints

### Health & Status

#### GET /health
Health check endpoint to verify server status.

**Response:**
```json
{
  "success": true,
  "message": "Health check passed",
  "data": {
    "status": "healthy",
    "version": "1.0.0",
    "environment": "development",
    "uptime": "2h30m15s",
    "checks": {
      "server": "ok",
      "config": "ok"
    }
  }
}
```

---

### AI Assistance API

#### POST /api/ai/suggestions
Get AI-powered code suggestions and improvements.

**Request Body:**
```json
{
  "code": "function add(a, b) { return a + b }",
  "language": "javascript",
  "context": "Simple addition function",
  "request_type": "suggestion"
}
```

**Parameters:**
- `code` (string, required): Code snippet to analyze
- `language` (string, required): Programming language
- `context` (string, optional): Additional context
- `request_type` (string, optional): Type of request (suggestion, debug, optimize)

**Response:**
```json
{
  "success": true,
  "message": "AI suggestions generated successfully",
  "data": {
    "suggestions": [
      {
        "type": "improvement",
        "description": "Add type checking for parameters",
        "code": "function add(a: number, b: number): number { return a + b }",
        "line_number": 1,
        "priority": "medium"
      }
    ],
    "analysis": "Function is simple but could benefit from type safety",
    "confidence": 0.85,
    "request_id": "req_123456"
  }
}
```

**Example cURL:**
```bash
curl -X POST http://localhost:8080/api/ai/suggestions \
  -H "Content-Type: application/json" \
  -d '{
    "code": "function add(a, b) { return a + b }",
    "language": "javascript",
    "context": "Simple addition function",
    "request_type": "suggestion"
  }'
```

#### POST /api/ai/analyze-logs
Analyze logs using AI to identify issues and suggest fixes.

**Request Body:**
```json
{
  "logs": [
    {
      "level": "error",
      "message": "Database connection failed",
      "timestamp": "2024-01-15T10:30:00Z"
    }
  ],
  "context": "Production environment"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Log analysis completed",
  "data": {
    "summary": "Database connectivity issues detected",
    "issues": [
      {
        "type": "connection_error",
        "count": 5,
        "severity": "critical",
        "solution": "Check database credentials and network connectivity"
      }
    ],
    "patterns": [
      {
        "pattern": "connection timeout",
        "frequency": 5,
        "description": "Repeated connection timeouts"
      }
    ],
    "suggestions": [
      "Verify database server is running",
      "Check firewall rules",
      "Validate connection string"
    ]
  }
}
```

#### GET /api/ai/status
Get AI service status and availability.

**Response:**
```json
{
  "success": true,
  "message": "AI service status",
  "data": {
    "available": true,
    "model": "gpt-4",
    "rate_limit_remaining": 95,
    "last_request": "2024-01-15T10:30:00Z"
  }
}
```

---

### Sync API

#### POST /api/sync/connect
Connect frontend and backend environments for synchronization.

**Request Body:**
```json
{
  "frontend_url": "http://localhost:3000",
  "backend_url": "http://localhost:8080",
  "environment": "development"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Environment connected successfully",
  "data": {
    "connection_id": "conn_123456",
    "status": "connected",
    "frontend_reachable": true,
    "backend_reachable": true
  }
}
```

#### GET /api/sync/status
Get current synchronization status.

**Response:**
```json
{
  "success": true,
  "message": "Sync status retrieved",
  "data": {
    "status": "active",
    "connected": true,
    "last_sync": "2024-01-15T10:30:00Z",
    "environments": {
      "development": "http://localhost:3000"
    },
    "health": {
      "frontend": true,
      "backend": true,
      "database": true
    }
  }
}
```

#### POST /api/sync/validate
Validate endpoint compatibility between environments.

**Request Body:**
```json
{
  "endpoint": "/api/users",
  "method": "GET",
  "expected_schema": { ... }
}
```

**Response:**
```json
{
  "success": true,
  "message": "Endpoint validation completed",
  "data": {
    "valid": true,
    "endpoint": "/api/users",
    "issues": [],
    "suggestions": []
  }
}
```

---

### Testing API

#### POST /api/testing/run
Trigger end-to-end test execution.

**Request Body:**
```json
{
  "framework": "cypress",
  "test_suite": "integration",
  "environment": "development",
  "config": {
    "browser": "chrome",
    "headless": true
  }
}
```

**Response:**
```json
{
  "success": true,
  "message": "Test run started",
  "data": {
    "run_id": "run_123456",
    "status": "running",
    "start_time": "2024-01-15T10:30:00Z",
    "framework": "cypress",
    "environment": "development"
  }
}
```

#### GET /api/testing/results/:runId
Get test execution results.

**Parameters:**
- `runId` (path parameter): Test run identifier

**Response:**
```json
{
  "success": true,
  "message": "Test results retrieved",
  "data": {
    "run_id": "run_123456",
    "status": "completed",
    "total_tests": 25,
    "passed_tests": 23,
    "failed_tests": 2,
    "duration": "45s",
    "results": [
      {
        "name": "User login test",
        "status": "passed",
        "duration": "2.5s"
      }
    ],
    "sync_issues": []
  }
}
```

#### POST /api/testing/validate-sync
Validate API-UI synchronization.

**Request Body:**
```json
{
  "api_endpoint": "/api/users",
  "ui_component": "UserList",
  "expected_behavior": "Display all users"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Sync validation completed",
  "data": {
    "synchronized": true,
    "issues": [],
    "recommendations": []
  }
}
```

---

### Logging API

#### POST /api/logs/submit
Submit log entries from frontend or other sources.

**Request Body:**
```json
{
  "logs": [
    {
      "level": "error",
      "source": "frontend",
      "message": "Failed to fetch user data",
      "context": {
        "user_id": "123",
        "endpoint": "/api/users/123"
      },
      "timestamp": "2024-01-15T10:30:00Z"
    }
  ]
}
```

**Response:**
```json
{
  "success": true,
  "message": "Logs submitted successfully",
  "data": {
    "received": 1,
    "stored": 1
  }
}
```

#### GET /api/logs/analyze
Analyze logs and detect patterns.

**Query Parameters:**
- `level` (optional): Filter by log level
- `source` (optional): Filter by source
- `from` (optional): Start timestamp
- `to` (optional): End timestamp

**Response:**
```json
{
  "success": true,
  "message": "Log analysis completed",
  "data": {
    "summary": "5 errors, 12 warnings in the last hour",
    "issues": [
      {
        "type": "api_error",
        "count": 5,
        "severity": "high",
        "description": "Multiple API failures detected"
      }
    ],
    "patterns": [
      {
        "pattern": "timeout",
        "frequency": 3
      }
    ],
    "suggestions": [
      "Investigate API timeout issues",
      "Check network connectivity"
    ]
  }
}
```

---

### Performance API

#### GET /api/performance/metrics
Get performance metrics for the application.

**Response:**
```json
{
  "success": true,
  "message": "Performance metrics retrieved",
  "data": {
    "requests_total": 1250,
    "requests_per_second": 25.5,
    "average_response_time": "45ms",
    "p95_response_time": "120ms",
    "p99_response_time": "250ms",
    "error_rate": 0.02
  }
}
```

#### GET /api/performance/memory
Get memory usage statistics.

**Response:**
```json
{
  "success": true,
  "message": "Memory statistics retrieved",
  "data": {
    "allocated": "45MB",
    "total_allocated": "1.2GB",
    "system_memory": "128MB",
    "gc_runs": 15,
    "goroutines": 42
  }
}
```

---

### WebSocket API

#### WS /ws
WebSocket connection for real-time updates.

**Connection:**
```javascript
const ws = new WebSocket('ws://localhost:8080/ws');

ws.onopen = () => {
  console.log('Connected to WebSocket');
};

ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  console.log('Received:', message);
};
```

**Message Format:**
```json
{
  "type": "sync_status_update",
  "data": {
    "status": "connected",
    "environment": "development"
  },
  "timestamp": "2024-01-15T10:30:00Z",
  "client_id": "client_123"
}
```

**Event Types:**
- `sync_status_update`: Sync status changes
- `test_progress`: Test execution updates
- `log_alert`: Critical log events
- `ai_suggestion_ready`: AI analysis completion

---

## Best Practices

### Error Handling
Always check the `success` field in responses and handle errors appropriately:

```javascript
const response = await fetch('/api/ai/suggestions', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify(requestData)
});

const result = await response.json();

if (!result.success) {
  console.error('Error:', result.error.message);
  console.error('Trace ID:', result.trace_id);
  return;
}

// Process successful response
console.log(result.data);
```

### Trace IDs
Include trace IDs in error reports for easier debugging:

```javascript
if (!result.success) {
  reportError({
    message: result.error.message,
    traceId: result.trace_id,
    endpoint: '/api/ai/suggestions'
  });
}
```

### WebSocket Reconnection
Implement automatic reconnection for WebSocket connections:

```javascript
let ws;
let reconnectAttempts = 0;
const maxReconnectAttempts = 5;

function connect() {
  ws = new WebSocket('ws://localhost:8080/ws');
  
  ws.onclose = () => {
    if (reconnectAttempts < maxReconnectAttempts) {
      reconnectAttempts++;
      setTimeout(connect, 1000 * reconnectAttempts);
    }
  };
  
  ws.onopen = () => {
    reconnectAttempts = 0;
  };
}

connect();
```

---

## Support

For issues or questions:
- Check the trace ID in error responses
- Review server logs for detailed error information
- Consult the troubleshooting guide in TROUBLESHOOTING.md
