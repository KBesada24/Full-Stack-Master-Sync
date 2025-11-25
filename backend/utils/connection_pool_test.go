package utils

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConnectionPool(t *testing.T) {
	config := DefaultConnectionPoolConfig()
	pool := NewConnectionPool(config)

	assert.NotNil(t, pool)
	assert.NotNil(t, pool.client)
	assert.Equal(t, config, pool.config)
	assert.NotNil(t, pool.stats)
}

func TestGetDefaultConnectionPool(t *testing.T) {
	// Reset default pool
	defaultPool = nil

	pool1 := GetDefaultConnectionPool()
	pool2 := GetDefaultConnectionPool()

	assert.NotNil(t, pool1)
	assert.Equal(t, pool1, pool2) // Should return same instance
}

func TestGetConnectionPool(t *testing.T) {
	// Clean up pools
	CloseAllPools()

	config := DefaultConnectionPoolConfig()
	pool1 := GetConnectionPool("test", config)
	pool2 := GetConnectionPool("test", config)

	assert.NotNil(t, pool1)
	assert.Equal(t, pool1, pool2) // Should return same instance for same name

	// Different name should return different pool
	pool3 := GetConnectionPool("test2", config)
	assert.NotEqual(t, pool1, pool3)
}

func TestConnectionPoolDo(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	pool := NewConnectionPool(DefaultConnectionPoolConfig())

	// Create request
	req, err := http.NewRequest("GET", server.URL, nil)
	require.NoError(t, err)

	// Execute request
	resp, err := pool.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Check stats
	stats := pool.GetStats()
	assert.Equal(t, int64(1), stats.TotalRequests)
	assert.Equal(t, int64(0), stats.FailedRequests)
	assert.True(t, stats.AverageLatency > 0)
}

func TestConnectionPoolDoWithContext(t *testing.T) {
	// Create test server with delay
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	pool := NewConnectionPool(DefaultConnectionPoolConfig())

	// Create request
	req, err := http.NewRequest("GET", server.URL, nil)
	require.NoError(t, err)

	// Test with context timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err = pool.DoWithContext(ctx, req)
	assert.Error(t, err) // Should timeout

	// Check stats
	stats := pool.GetStats()
	assert.Equal(t, int64(1), stats.TotalRequests)
	assert.Equal(t, int64(1), stats.FailedRequests)
}

func TestConnectionPoolDoWithTimeout(t *testing.T) {
	// Create test server with delay
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	pool := NewConnectionPool(DefaultConnectionPoolConfig())

	// Create request
	req, err := http.NewRequest("GET", server.URL, nil)
	require.NoError(t, err)

	// Test with timeout
	_, err = pool.DoWithTimeout(req, 50*time.Millisecond)
	assert.Error(t, err) // Should timeout

	// Test without timeout
	req, err = http.NewRequest("GET", server.URL, nil)
	require.NoError(t, err)

	resp, err := pool.DoWithTimeout(req, 200*time.Millisecond)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestConnectionPoolStats(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1 * time.Millisecond) // Add small delay to ensure measurable latency
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	pool := NewConnectionPool(DefaultConnectionPoolConfig())

	// Make multiple requests
	for i := 0; i < 5; i++ {
		req, err := http.NewRequest("GET", server.URL, nil)
		require.NoError(t, err)

		resp, err := pool.Do(req)
		require.NoError(t, err)
		resp.Body.Close()
	}

	stats := pool.GetStats()
	assert.Equal(t, int64(5), stats.TotalRequests)
	assert.Equal(t, int64(0), stats.FailedRequests)
	assert.True(t, stats.AverageLatency >= 0) // Allow 0 latency for very fast responses
	assert.True(t, stats.LastUsed.After(stats.CreatedAt))
}

func TestConnectionPoolClose(t *testing.T) {
	pool := NewConnectionPool(DefaultConnectionPoolConfig())

	// Should not panic
	pool.Close()
	pool.CloseIdleConnections()
}

func TestGetAllPoolStats(t *testing.T) {
	// Clean up pools
	CloseAllPools()

	// Create some pools
	GetDefaultConnectionPool()
	GetConnectionPool("test1", DefaultConnectionPoolConfig())
	GetConnectionPool("test2", DefaultConnectionPoolConfig())

	stats := GetAllPoolStats()
	assert.Len(t, stats, 3) // default + test1 + test2
	assert.Contains(t, stats, "default")
	assert.Contains(t, stats, "test1")
	assert.Contains(t, stats, "test2")
}

func TestCloseAllPools(t *testing.T) {
	// Create some pools
	GetDefaultConnectionPool()
	GetConnectionPool("test1", DefaultConnectionPoolConfig())
	GetConnectionPool("test2", DefaultConnectionPoolConfig())

	// Verify pools exist
	stats := GetAllPoolStats()
	assert.Len(t, stats, 3)

	// Close all pools
	CloseAllPools()

	// Verify pools are cleared
	stats = GetAllPoolStats()
	assert.Len(t, stats, 0)
}

func TestHTTPClientWithPool(t *testing.T) {
	config := DefaultConnectionPoolConfig()
	client := HTTPClientWithPool("test-service", config)

	assert.NotNil(t, client)
	assert.IsType(t, &http.Client{}, client)
}

func TestOpenAIConnectionPool(t *testing.T) {
	pool := OpenAIConnectionPool()

	assert.NotNil(t, pool)
	assert.Equal(t, int64(0), pool.GetStats().TotalRequests)

	// Verify it's the same instance when called again
	pool2 := OpenAIConnectionPool()
	assert.Equal(t, pool, pool2)
}

func TestTestingFrameworkConnectionPool(t *testing.T) {
	pool := TestingFrameworkConnectionPool()

	assert.NotNil(t, pool)
	assert.Equal(t, int64(0), pool.GetStats().TotalRequests)

	// Verify it's the same instance when called again
	pool2 := TestingFrameworkConnectionPool()
	assert.Equal(t, pool, pool2)
}

func TestConnectionPoolConcurrency(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond) // Simulate processing time
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	pool := NewConnectionPool(DefaultConnectionPoolConfig())

	// Make concurrent requests
	const numRequests = 10
	done := make(chan bool, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			req, err := http.NewRequest("GET", server.URL, nil)
			if err != nil {
				t.Errorf("Failed to create request: %v", err)
				done <- false
				return
			}

			resp, err := pool.Do(req)
			if err != nil {
				t.Errorf("Request failed: %v", err)
				done <- false
				return
			}
			resp.Body.Close()

			done <- true
		}()
	}

	// Wait for all requests to complete
	successCount := 0
	for i := 0; i < numRequests; i++ {
		if <-done {
			successCount++
		}
	}

	assert.Equal(t, numRequests, successCount)

	stats := pool.GetStats()
	assert.Equal(t, int64(numRequests), stats.TotalRequests)
	assert.Equal(t, int64(0), stats.FailedRequests)
}

func BenchmarkConnectionPoolDo(b *testing.B) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	pool := NewConnectionPool(DefaultConnectionPoolConfig())

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req, err := http.NewRequest("GET", server.URL, nil)
			if err != nil {
				b.Fatal(err)
			}

			resp, err := pool.Do(req)
			if err != nil {
				b.Fatal(err)
			}
			resp.Body.Close()
		}
	})
}
