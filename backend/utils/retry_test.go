package utils

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetryExecutor_SuccessfulExecution(t *testing.T) {
	config := DefaultRetryConfig()
	executor := NewRetryExecutor(config, nil)

	ctx := context.Background()
	executed := false

	err := executor.Execute(ctx, func(ctx context.Context) error {
		executed = true
		return nil
	})

	assert.NoError(t, err)
	assert.True(t, executed)
}

func TestRetryExecutor_RetryOnFailure(t *testing.T) {
	config := &RetryConfig{
		MaxAttempts:       3,
		InitialDelay:      10 * time.Millisecond,
		MaxDelay:          100 * time.Millisecond,
		BackoffMultiplier: 2.0,
		Jitter:            false,
		RetryCondition: func(err error) bool {
			return err.Error() == "retryable error"
		},
	}
	executor := NewRetryExecutor(config, nil)

	ctx := context.Background()
	attemptCount := 0
	retryableError := errors.New("retryable error")

	err := executor.Execute(ctx, func(ctx context.Context) error {
		attemptCount++
		if attemptCount < 3 {
			return retryableError
		}
		return nil // Success on third attempt
	})

	assert.NoError(t, err)
	assert.Equal(t, 3, attemptCount)
}

func TestRetryExecutor_NonRetryableError(t *testing.T) {
	config := &RetryConfig{
		MaxAttempts:       3,
		InitialDelay:      10 * time.Millisecond,
		MaxDelay:          100 * time.Millisecond,
		BackoffMultiplier: 2.0,
		Jitter:            false,
		RetryCondition: func(err error) bool {
			return err.Error() == "retryable error"
		},
	}
	executor := NewRetryExecutor(config, nil)

	ctx := context.Background()
	attemptCount := 0
	nonRetryableError := errors.New("non-retryable error")

	err := executor.Execute(ctx, func(ctx context.Context) error {
		attemptCount++
		return nonRetryableError
	})

	assert.Error(t, err)
	assert.Equal(t, 1, attemptCount) // Should only attempt once

	var retryErr *RetryableError
	require.True(t, errors.As(err, &retryErr))
	assert.False(t, retryErr.Retryable)
	assert.Equal(t, 1, retryErr.Attempt)
}

func TestRetryExecutor_ExhaustAllAttempts(t *testing.T) {
	config := &RetryConfig{
		MaxAttempts:       3,
		InitialDelay:      10 * time.Millisecond,
		MaxDelay:          100 * time.Millisecond,
		BackoffMultiplier: 2.0,
		Jitter:            false,
		RetryCondition: func(err error) bool {
			return true // All errors are retryable
		},
	}
	executor := NewRetryExecutor(config, nil)

	ctx := context.Background()
	attemptCount := 0
	persistentError := errors.New("persistent error")

	err := executor.Execute(ctx, func(ctx context.Context) error {
		attemptCount++
		return persistentError
	})

	assert.Error(t, err)
	assert.Equal(t, 3, attemptCount) // Should attempt all 3 times
	assert.True(t, IsRetryableError(err))

	var retryErr *RetryableError
	require.True(t, errors.As(err, &retryErr))
	assert.True(t, retryErr.Retryable)
	assert.Equal(t, 3, retryErr.Attempt)
}

func TestRetryExecutor_ContextCancellation(t *testing.T) {
	config := &RetryConfig{
		MaxAttempts:       5,
		InitialDelay:      100 * time.Millisecond,
		MaxDelay:          1 * time.Second,
		BackoffMultiplier: 2.0,
		Jitter:            false,
		RetryCondition: func(err error) bool {
			return true
		},
	}
	executor := NewRetryExecutor(config, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	attemptCount := 0
	persistentError := errors.New("persistent error")

	err := executor.Execute(ctx, func(ctx context.Context) error {
		attemptCount++
		return persistentError
	})

	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
	// Should have made at least one attempt, but not all 5 due to timeout
	assert.GreaterOrEqual(t, attemptCount, 1)
	assert.Less(t, attemptCount, 5)
}

func TestRetryExecutor_CommonRetryableErrors(t *testing.T) {
	config := DefaultRetryConfig()
	executor := NewRetryExecutor(config, nil)

	testCases := []struct {
		name      string
		error     error
		retryable bool
	}{
		{"connection refused", errors.New("connection refused"), true},
		{"connection timeout", errors.New("connection timeout"), true},
		{"timeout", errors.New("timeout occurred"), true},
		{"rate limit", errors.New("rate limit exceeded"), true},
		{"service unavailable", errors.New("service unavailable"), true},
		{"network unreachable", errors.New("network is unreachable"), true},
		{"i/o timeout", errors.New("i/o timeout"), true},
		{"generic error", errors.New("some generic error"), false},
		{"validation error", errors.New("validation failed"), false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			attemptCount := 0

			err := executor.Execute(ctx, func(ctx context.Context) error {
				attemptCount++
				return tc.error
			})

			if tc.retryable {
				assert.Equal(t, config.MaxAttempts, attemptCount)
			} else {
				assert.Equal(t, 1, attemptCount)
			}
			assert.Error(t, err)
		})
	}
}

func TestRetryExecutor_CircuitBreakerIntegration(t *testing.T) {
	// Create circuit breaker that opens after 1 failure
	cbConfig := &CircuitBreakerConfig{
		MaxFailures:      1,
		Timeout:          100 * time.Millisecond,
		MaxRequests:      1,
		SuccessThreshold: 1,
		Name:             "test",
	}
	cb := NewCircuitBreaker(cbConfig, nil)

	retryConfig := &RetryConfig{
		MaxAttempts:       3,
		InitialDelay:      10 * time.Millisecond,
		MaxDelay:          100 * time.Millisecond,
		BackoffMultiplier: 2.0,
		Jitter:            false,
	}

	ctx := context.Background()
	testError := errors.New("test error")
	attemptCount := 0

	err := RetryWithCircuitBreaker(ctx, retryConfig, cb, func(ctx context.Context) error {
		attemptCount++
		return testError
	}, nil)

	assert.Error(t, err)
	// Circuit breaker should open after first failure, preventing retries
	assert.Equal(t, 1, attemptCount)
	assert.Equal(t, StateOpen, cb.GetState())
}

func TestExponentialBackoff(t *testing.T) {
	initialDelay := 100 * time.Millisecond
	maxDelay := 1 * time.Second
	multiplier := 2.0

	testCases := []struct {
		attempt  int
		expected time.Duration
	}{
		{1, 100 * time.Millisecond},
		{2, 200 * time.Millisecond},
		{3, 400 * time.Millisecond},
		{4, 800 * time.Millisecond},
		{5, 1 * time.Second}, // Capped at maxDelay
		{6, 1 * time.Second}, // Still capped
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("attempt_%d", tc.attempt), func(t *testing.T) {
			delay := ExponentialBackoff(tc.attempt, initialDelay, maxDelay, multiplier, false)
			assert.Equal(t, tc.expected, delay)
		})
	}
}

func TestLinearBackoff(t *testing.T) {
	initialDelay := 100 * time.Millisecond
	maxDelay := 500 * time.Millisecond

	testCases := []struct {
		attempt  int
		expected time.Duration
	}{
		{1, 100 * time.Millisecond},
		{2, 200 * time.Millisecond},
		{3, 300 * time.Millisecond},
		{4, 400 * time.Millisecond},
		{5, 500 * time.Millisecond}, // Capped at maxDelay
		{6, 500 * time.Millisecond}, // Still capped
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("attempt_%d", tc.attempt), func(t *testing.T) {
			delay := LinearBackoff(tc.attempt, initialDelay, maxDelay, false)
			assert.Equal(t, tc.expected, delay)
		})
	}
}

func TestFixedBackoff(t *testing.T) {
	delay := 200 * time.Millisecond
	result := FixedBackoff(delay, false)
	assert.Equal(t, delay, result)
}

func TestJitterBackoff(t *testing.T) {
	delay := 1 * time.Second

	// Test multiple times to ensure jitter is working
	results := make([]time.Duration, 10)
	for i := 0; i < 10; i++ {
		results[i] = FixedBackoff(delay, true)
	}

	// All results should be different (with very high probability)
	// and within the expected range (original delay + up to 10% jitter)
	minExpected := delay
	maxExpected := delay + time.Duration(float64(delay)*0.1)

	allSame := true
	for i := 1; i < len(results); i++ {
		if results[i] != results[0] {
			allSame = false
			break
		}
	}
	assert.False(t, allSame, "Jitter should produce different results")

	for i, result := range results {
		assert.GreaterOrEqual(t, result, minExpected, "Result %d should be >= min expected", i)
		assert.LessOrEqual(t, result, maxExpected, "Result %d should be <= max expected", i)
	}
}

func BenchmarkRetryExecutor_Execute(b *testing.B) {
	config := DefaultRetryConfig()
	executor := NewRetryExecutor(config, nil)
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			executor.Execute(ctx, func(ctx context.Context) error {
				return nil
			})
		}
	})
}

func BenchmarkExponentialBackoff(b *testing.B) {
	initialDelay := 100 * time.Millisecond
	maxDelay := 1 * time.Second
	multiplier := 2.0

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ExponentialBackoff(5, initialDelay, maxDelay, multiplier, true)
	}
}
