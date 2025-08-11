package utils

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCircuitBreaker_InitialState(t *testing.T) {
	config := DefaultCircuitBreakerConfig("test")
	cb := NewCircuitBreaker(config, nil)

	assert.Equal(t, StateClosed, cb.GetState())
	stats := cb.GetStats()
	assert.Equal(t, "test", stats["name"])
	assert.Equal(t, "CLOSED", stats["state"])
	assert.Equal(t, 0, stats["failures"])
}

func TestCircuitBreaker_SuccessfulExecution(t *testing.T) {
	config := DefaultCircuitBreakerConfig("test")
	cb := NewCircuitBreaker(config, nil)

	ctx := context.Background()
	executed := false

	err := cb.Execute(ctx, func(ctx context.Context) error {
		executed = true
		return nil
	})

	assert.NoError(t, err)
	assert.True(t, executed)
	assert.Equal(t, StateClosed, cb.GetState())
}

func TestCircuitBreaker_FailureHandling(t *testing.T) {
	config := &CircuitBreakerConfig{
		MaxFailures:      2,
		Timeout:          100 * time.Millisecond,
		MaxRequests:      1,
		SuccessThreshold: 1,
		Name:             "test",
	}
	cb := NewCircuitBreaker(config, nil)

	ctx := context.Background()
	testError := errors.New("test error")

	// First failure - should remain closed
	err := cb.Execute(ctx, func(ctx context.Context) error {
		return testError
	})
	assert.Error(t, err)
	assert.Equal(t, StateClosed, cb.GetState())

	// Second failure - should open circuit
	err = cb.Execute(ctx, func(ctx context.Context) error {
		return testError
	})
	assert.Error(t, err)
	assert.Equal(t, StateOpen, cb.GetState())

	// Third attempt - should be rejected immediately
	executed := false
	err = cb.Execute(ctx, func(ctx context.Context) error {
		executed = true
		return nil
	})
	assert.Error(t, err)
	assert.False(t, executed)
	assert.True(t, IsCircuitBreakerError(err))
}

func TestCircuitBreaker_HalfOpenTransition(t *testing.T) {
	config := &CircuitBreakerConfig{
		MaxFailures:      1,
		Timeout:          50 * time.Millisecond,
		MaxRequests:      2,
		SuccessThreshold: 1,
		Name:             "test",
	}
	cb := NewCircuitBreaker(config, nil)

	ctx := context.Background()
	testError := errors.New("test error")

	// Trigger circuit open
	err := cb.Execute(ctx, func(ctx context.Context) error {
		return testError
	})
	assert.Error(t, err)
	assert.Equal(t, StateOpen, cb.GetState())

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	// Next request should transition to half-open
	executed := false
	err = cb.Execute(ctx, func(ctx context.Context) error {
		executed = true
		return nil
	})
	assert.NoError(t, err)
	assert.True(t, executed)
	assert.Equal(t, StateClosed, cb.GetState()) // Should close after successful request
}

func TestCircuitBreaker_HalfOpenFailure(t *testing.T) {
	config := &CircuitBreakerConfig{
		MaxFailures:      1,
		Timeout:          50 * time.Millisecond,
		MaxRequests:      2,
		SuccessThreshold: 1,
		Name:             "test",
	}
	cb := NewCircuitBreaker(config, nil)

	ctx := context.Background()
	testError := errors.New("test error")

	// Trigger circuit open
	err := cb.Execute(ctx, func(ctx context.Context) error {
		return testError
	})
	assert.Error(t, err)
	assert.Equal(t, StateOpen, cb.GetState())

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	// Next request should transition to half-open, then back to open on failure
	err = cb.Execute(ctx, func(ctx context.Context) error {
		return testError
	})
	assert.Error(t, err)
	assert.Equal(t, StateOpen, cb.GetState())
}

func TestCircuitBreaker_Reset(t *testing.T) {
	config := &CircuitBreakerConfig{
		MaxFailures:      1,
		Timeout:          100 * time.Millisecond,
		MaxRequests:      1,
		SuccessThreshold: 1,
		Name:             "test",
	}
	cb := NewCircuitBreaker(config, nil)

	ctx := context.Background()
	testError := errors.New("test error")

	// Trigger circuit open
	err := cb.Execute(ctx, func(ctx context.Context) error {
		return testError
	})
	assert.Error(t, err)
	assert.Equal(t, StateOpen, cb.GetState())

	// Reset circuit breaker
	cb.Reset()
	assert.Equal(t, StateClosed, cb.GetState())

	// Should work normally after reset
	executed := false
	err = cb.Execute(ctx, func(ctx context.Context) error {
		executed = true
		return nil
	})
	assert.NoError(t, err)
	assert.True(t, executed)
}

func TestCircuitBreakerManager(t *testing.T) {
	manager := NewCircuitBreakerManager(nil)

	// Test creating new circuit breaker
	cb1 := manager.GetOrCreate("test1", nil)
	assert.NotNil(t, cb1)
	assert.Equal(t, "test1", cb1.config.Name)

	// Test getting existing circuit breaker
	cb2 := manager.GetOrCreate("test1", nil)
	assert.Equal(t, cb1, cb2) // Should be the same instance

	// Test getting non-existent circuit breaker
	cb3, exists := manager.Get("nonexistent")
	assert.Nil(t, cb3)
	assert.False(t, exists)

	// Test getting all circuit breakers
	all := manager.GetAll()
	assert.Len(t, all, 1)
	assert.Contains(t, all, "test1")

	// Test stats
	stats := manager.GetStats()
	assert.Contains(t, stats, "test1")

	// Test reset
	success := manager.Reset("test1")
	assert.True(t, success)

	success = manager.Reset("nonexistent")
	assert.False(t, success)

	// Test reset all
	manager.GetOrCreate("test2", nil)
	manager.ResetAll()
	// Should not panic and should reset all breakers
}

func TestCircuitBreakerError(t *testing.T) {
	err := &CircuitBreakerError{
		State:   StateOpen,
		Message: "circuit breaker is open",
	}

	assert.Equal(t, "circuit breaker is open", err.Error())
	assert.True(t, IsCircuitBreakerError(err))

	// Test with wrapped error
	wrappedErr := errors.New("some other error")
	assert.False(t, IsCircuitBreakerError(wrappedErr))
}

func TestCircuitBreaker_ConcurrentAccess(t *testing.T) {
	config := DefaultCircuitBreakerConfig("concurrent_test")
	cb := NewCircuitBreaker(config, nil)

	ctx := context.Background()
	const numGoroutines = 10
	const numRequests = 100

	results := make(chan error, numGoroutines*numRequests)

	// Launch multiple goroutines making concurrent requests
	for i := 0; i < numGoroutines; i++ {
		go func() {
			for j := 0; j < numRequests; j++ {
				err := cb.Execute(ctx, func(ctx context.Context) error {
					time.Sleep(time.Microsecond) // Simulate some work
					return nil
				})
				results <- err
			}
		}()
	}

	// Collect results
	successCount := 0
	for i := 0; i < numGoroutines*numRequests; i++ {
		err := <-results
		if err == nil {
			successCount++
		}
	}

	// All requests should succeed since we're not introducing failures
	assert.Equal(t, numGoroutines*numRequests, successCount)
	assert.Equal(t, StateClosed, cb.GetState())
}

func BenchmarkCircuitBreaker_Execute(b *testing.B) {
	config := DefaultCircuitBreakerConfig("benchmark")
	cb := NewCircuitBreaker(config, nil)
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cb.Execute(ctx, func(ctx context.Context) error {
				return nil
			})
		}
	})
}
