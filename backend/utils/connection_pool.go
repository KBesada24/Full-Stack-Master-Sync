package utils

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"sync"
	"time"
)

// ConnectionPoolConfig holds configuration for HTTP connection pooling
type ConnectionPoolConfig struct {
	MaxIdleConns          int           // Maximum number of idle connections
	MaxIdleConnsPerHost   int           // Maximum number of idle connections per host
	MaxConnsPerHost       int           // Maximum number of connections per host
	IdleConnTimeout       time.Duration // How long an idle connection is kept alive
	TLSHandshakeTimeout   time.Duration // TLS handshake timeout
	ResponseHeaderTimeout time.Duration // Response header timeout
	ExpectContinueTimeout time.Duration // Expect continue timeout
	DialTimeout           time.Duration // Connection dial timeout
	KeepAlive             time.Duration // Keep alive duration
	DisableCompression    bool          // Disable compression
	DisableKeepAlives     bool          // Disable keep alives
}

// DefaultConnectionPoolConfig returns default connection pool configuration
func DefaultConnectionPoolConfig() *ConnectionPoolConfig {
	return &ConnectionPoolConfig{
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		MaxConnsPerHost:       100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DialTimeout:           30 * time.Second,
		KeepAlive:             30 * time.Second,
		DisableCompression:    false,
		DisableKeepAlives:     false,
	}
}

// ConnectionPool manages HTTP client connections with pooling
type ConnectionPool struct {
	client *http.Client
	config *ConnectionPoolConfig
	stats  *ConnectionPoolStats
	mu     sync.RWMutex
}

// ConnectionPoolStats holds statistics about connection pool usage
type ConnectionPoolStats struct {
	ActiveConnections int64     `json:"active_connections"`
	IdleConnections   int64     `json:"idle_connections"`
	TotalRequests     int64     `json:"total_requests"`
	FailedRequests    int64     `json:"failed_requests"`
	AverageLatency    float64   `json:"average_latency_ms"`
	TotalLatency      int64     `json:"total_latency_ms"`
	LastUsed          time.Time `json:"last_used"`
	CreatedAt         time.Time `json:"created_at"`
}

// Global connection pools for different services
var (
	defaultPool *ConnectionPool
	poolsMap    = make(map[string]*ConnectionPool)
	poolsMutex  sync.RWMutex
)

// NewConnectionPool creates a new connection pool with the given configuration
func NewConnectionPool(config *ConnectionPoolConfig) *ConnectionPool {
	if config == nil {
		config = DefaultConnectionPoolConfig()
	}

	// Create custom dialer
	dialer := &net.Dialer{
		Timeout:   config.DialTimeout,
		KeepAlive: config.KeepAlive,
	}

	// Create custom transport with connection pooling
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		MaxIdleConns:          config.MaxIdleConns,
		MaxIdleConnsPerHost:   config.MaxIdleConnsPerHost,
		MaxConnsPerHost:       config.MaxConnsPerHost,
		IdleConnTimeout:       config.IdleConnTimeout,
		TLSHandshakeTimeout:   config.TLSHandshakeTimeout,
		ResponseHeaderTimeout: config.ResponseHeaderTimeout,
		ExpectContinueTimeout: config.ExpectContinueTimeout,
		DisableCompression:    config.DisableCompression,
		DisableKeepAlives:     config.DisableKeepAlives,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
		},
	}

	// Create HTTP client with custom transport
	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second, // Overall request timeout
	}

	return &ConnectionPool{
		client: client,
		config: config,
		stats: &ConnectionPoolStats{
			CreatedAt: time.Now(),
			LastUsed:  time.Now(),
		},
	}
}

// GetDefaultConnectionPool returns the default connection pool
func GetDefaultConnectionPool() *ConnectionPool {
	if defaultPool == nil {
		defaultPool = NewConnectionPool(DefaultConnectionPoolConfig())
	}
	return defaultPool
}

// GetConnectionPool returns a named connection pool, creating it if it doesn't exist
func GetConnectionPool(name string, config *ConnectionPoolConfig) *ConnectionPool {
	poolsMutex.RLock()
	pool, exists := poolsMap[name]
	poolsMutex.RUnlock()

	if exists {
		return pool
	}

	poolsMutex.Lock()
	defer poolsMutex.Unlock()

	// Double-check after acquiring write lock
	if pool, exists = poolsMap[name]; exists {
		return pool
	}

	// Create new pool
	pool = NewConnectionPool(config)
	poolsMap[name] = pool
	return pool
}

// Do executes an HTTP request using the connection pool
func (cp *ConnectionPool) Do(req *http.Request) (*http.Response, error) {
	startTime := time.Now()

	// Update stats - increment active connections and total requests
	cp.mu.Lock()
	cp.stats.TotalRequests++
	cp.stats.ActiveConnections++
	cp.stats.LastUsed = time.Now()
	cp.mu.Unlock()

	// Execute request
	resp, err := cp.client.Do(req)

	// Calculate latency
	latency := time.Since(startTime)
	latencyMs := latency.Milliseconds()

	// Update stats with result
	cp.mu.Lock()
	cp.stats.ActiveConnections-- // Request completed
	cp.stats.TotalLatency += latencyMs

	if cp.stats.TotalRequests > 0 {
		cp.stats.AverageLatency = float64(cp.stats.TotalLatency) / float64(cp.stats.TotalRequests)
	}

	if err != nil {
		cp.stats.FailedRequests++
	}
	cp.mu.Unlock()

	return resp, err
}

// DoWithContext executes an HTTP request with context using the connection pool
func (cp *ConnectionPool) DoWithContext(ctx context.Context, req *http.Request) (*http.Response, error) {
	// Set context on request
	req = req.WithContext(ctx)
	return cp.Do(req)
}

// DoWithTimeout executes an HTTP request with timeout using the connection pool
func (cp *ConnectionPool) DoWithTimeout(req *http.Request, timeout time.Duration) (*http.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return cp.DoWithContext(ctx, req)
}

// GetClient returns the underlying HTTP client
func (cp *ConnectionPool) GetClient() *http.Client {
	return cp.client
}

// GetStats returns connection pool statistics
func (cp *ConnectionPool) GetStats() *ConnectionPoolStats {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	// Create a copy to avoid race conditions
	stats := &ConnectionPoolStats{
		ActiveConnections: cp.stats.ActiveConnections,
		IdleConnections:   cp.stats.IdleConnections,
		TotalRequests:     cp.stats.TotalRequests,
		FailedRequests:    cp.stats.FailedRequests,
		AverageLatency:    cp.stats.AverageLatency,
		TotalLatency:      cp.stats.TotalLatency,
		LastUsed:          cp.stats.LastUsed,
		CreatedAt:         cp.stats.CreatedAt,
	}

	return stats
}

// Close closes the connection pool and cleans up resources
func (cp *ConnectionPool) Close() {
	if transport, ok := cp.client.Transport.(*http.Transport); ok {
		transport.CloseIdleConnections()
	}
}

// CloseIdleConnections closes idle connections in the pool
func (cp *ConnectionPool) CloseIdleConnections() {
	if transport, ok := cp.client.Transport.(*http.Transport); ok {
		transport.CloseIdleConnections()
	}
}

// GetAllPoolStats returns statistics for all connection pools
func GetAllPoolStats() map[string]*ConnectionPoolStats {
	poolsMutex.RLock()
	defer poolsMutex.RUnlock()

	stats := make(map[string]*ConnectionPoolStats)

	// Add default pool stats
	if defaultPool != nil {
		stats["default"] = defaultPool.GetStats()
	}

	// Add named pool stats
	for name, pool := range poolsMap {
		stats[name] = pool.GetStats()
	}

	return stats
}

// CloseAllPools closes all connection pools
func CloseAllPools() {
	poolsMutex.Lock()
	defer poolsMutex.Unlock()

	// Close default pool
	if defaultPool != nil {
		defaultPool.Close()
	}

	// Close named pools
	for _, pool := range poolsMap {
		pool.Close()
	}

	// Clear pools map
	poolsMap = make(map[string]*ConnectionPool)
	defaultPool = nil
}

// Helper functions

// HTTPClientWithPool creates an HTTP client with connection pooling for a specific service
func HTTPClientWithPool(serviceName string, config *ConnectionPoolConfig) *http.Client {
	pool := GetConnectionPool(serviceName, config)
	return pool.GetClient()
}

// OpenAIConnectionPool returns a connection pool optimized for OpenAI API calls
func OpenAIConnectionPool() *ConnectionPool {
	config := &ConnectionPoolConfig{
		MaxIdleConns:          20,
		MaxIdleConnsPerHost:   5,
		MaxConnsPerHost:       20,
		IdleConnTimeout:       60 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 60 * time.Second, // OpenAI can be slow
		ExpectContinueTimeout: 1 * time.Second,
		DialTimeout:           10 * time.Second,
		KeepAlive:             30 * time.Second,
		DisableCompression:    false,
		DisableKeepAlives:     false,
	}

	return GetConnectionPool("openai", config)
}

// TestingFrameworkConnectionPool returns a connection pool optimized for testing framework API calls
func TestingFrameworkConnectionPool() *ConnectionPool {
	config := &ConnectionPoolConfig{
		MaxIdleConns:          10,
		MaxIdleConnsPerHost:   3,
		MaxConnsPerHost:       10,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DialTimeout:           10 * time.Second,
		KeepAlive:             15 * time.Second,
		DisableCompression:    false,
		DisableKeepAlives:     false,
	}

	return GetConnectionPool("testing", config)
}
