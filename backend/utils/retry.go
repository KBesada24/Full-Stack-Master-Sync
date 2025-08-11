package utils

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"time"
)

// RetryConfig holds configuration for retry mechanisms
type RetryConfig struct {
	// MaxAttempts is the maximum number of retry attempts
	MaxAttempts int
	// InitialDelay is the initial delay between retries
	InitialDelay time.Duration
	// MaxDelay is the maximum delay between retries
	MaxDelay time.Duration
	// BackoffMultiplier is the multiplier for exponential backoff
	BackoffMultiplier float64
	// Jitter adds randomness to delay to avoid thundering herd
	Jitter bool
	// RetryableErrors is a list of error types that should trigger retries
	RetryableErrors []error
	// RetryCondition is a custom function to determine if an error is retryable
	RetryCondition func(error) bool
}

// DefaultRetryConfig returns a default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts:       3,
		InitialDelay:      100 * time.Millisecond,
		MaxDelay:          30 * time.Second,
		BackoffMultiplier: 2.0,
		Jitter:            true,
		RetryableErrors:   []error{},
		RetryCondition:    nil,
	}
}

// RetryableError represents an error that can be retried
type RetryableError struct {
	Err       error
	Retryable bool
	Attempt   int
}

// Error implements the error interface
func (e *RetryableError) Error() string {
	return fmt.Sprintf("attempt %d: %v", e.Attempt, e.Err)
}

// Unwrap returns the underlying error
func (e *RetryableError) Unwrap() error {
	return e.Err
}

// IsRetryableError checks if an error is retryable
func IsRetryableError(err error) bool {
	var retryableErr *RetryableError
	if errors.As(err, &retryableErr) {
		return retryableErr.Retryable
	}
	return false
}

// RetryExecutor handles retry logic
type RetryExecutor struct {
	config *RetryConfig
	logger *Logger
}

// NewRetryExecutor creates a new retry executor
func NewRetryExecutor(config *RetryConfig, logger *Logger) *RetryExecutor {
	if config == nil {
		config = DefaultRetryConfig()
	}
	if logger == nil {
		logger = GetLogger()
	}

	return &RetryExecutor{
		config: config,
		logger: logger,
	}
}

// Execute executes a function with retry logic
func (re *RetryExecutor) Execute(ctx context.Context, operation func(context.Context) error) error {
	var lastErr error

	for attempt := 1; attempt <= re.config.MaxAttempts; attempt++ {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Execute the operation
		err := operation(ctx)
		if err == nil {
			// Success
			if attempt > 1 {
				re.logger.WithSource("retry_executor").Info("Operation succeeded after retry", map[string]interface{}{
					"attempt":        attempt,
					"total_attempts": re.config.MaxAttempts,
				})
			}
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !re.isRetryable(err) {
			re.logger.WithSource("retry_executor").Debug("Error is not retryable", map[string]interface{}{
				"error":   err.Error(),
				"attempt": attempt,
			})
			return &RetryableError{
				Err:       err,
				Retryable: false,
				Attempt:   attempt,
			}
		}

		// Don't sleep after the last attempt
		if attempt == re.config.MaxAttempts {
			break
		}

		// Calculate delay for next attempt
		delay := re.calculateDelay(attempt)

		re.logger.WithSource("retry_executor").Warn("Operation failed, retrying", map[string]interface{}{
			"error":        err.Error(),
			"attempt":      attempt,
			"max_attempts": re.config.MaxAttempts,
			"retry_delay":  delay,
		})

		// Wait before next attempt
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}

	// All attempts failed
	re.logger.WithSource("retry_executor").Error("All retry attempts failed", lastErr, map[string]interface{}{
		"max_attempts": re.config.MaxAttempts,
	})

	return &RetryableError{
		Err:       lastErr,
		Retryable: true,
		Attempt:   re.config.MaxAttempts,
	}
}

// isRetryable determines if an error should trigger a retry
func (re *RetryExecutor) isRetryable(err error) bool {
	// Check custom retry condition first
	if re.config.RetryCondition != nil {
		return re.config.RetryCondition(err)
	}

	// Check against configured retryable errors
	for _, retryableErr := range re.config.RetryableErrors {
		if errors.Is(err, retryableErr) {
			return true
		}
	}

	// Check for common retryable error patterns
	return re.isCommonRetryableError(err)
}

// isCommonRetryableError checks for common retryable error patterns
func (re *RetryExecutor) isCommonRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Network-related errors that are typically retryable
	retryablePatterns := []string{
		"connection refused",
		"connection reset",
		"connection timeout",
		"timeout",
		"temporary failure",
		"service unavailable",
		"too many requests",
		"rate limit",
		"network is unreachable",
		"no route to host",
		"i/o timeout",
	}

	for _, pattern := range retryablePatterns {
		if contains(errStr, pattern) {
			return true
		}
	}

	// Check for circuit breaker errors
	if IsCircuitBreakerError(err) {
		return false // Don't retry circuit breaker errors
	}

	return false
}

// calculateDelay calculates the delay for the next retry attempt
func (re *RetryExecutor) calculateDelay(attempt int) time.Duration {
	// Calculate exponential backoff delay
	delay := float64(re.config.InitialDelay) * math.Pow(re.config.BackoffMultiplier, float64(attempt-1))

	// Apply maximum delay limit
	if delay > float64(re.config.MaxDelay) {
		delay = float64(re.config.MaxDelay)
	}

	// Add jitter if enabled
	if re.config.Jitter {
		jitter := rand.Float64() * 0.1 * delay // Up to 10% jitter
		delay += jitter
	}

	return time.Duration(delay)
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					containsSubstring(s, substr)))
}

// containsSubstring performs case-insensitive substring search
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// RetryWithCircuitBreaker combines retry logic with circuit breaker
func RetryWithCircuitBreaker(
	ctx context.Context,
	retryConfig *RetryConfig,
	circuitBreaker *CircuitBreaker,
	operation func(context.Context) error,
	logger *Logger,
) error {
	if logger == nil {
		logger = GetLogger()
	}

	retryExecutor := NewRetryExecutor(retryConfig, logger)

	return retryExecutor.Execute(ctx, func(ctx context.Context) error {
		return circuitBreaker.Execute(ctx, operation)
	})
}

// ExponentialBackoff calculates exponential backoff delay
func ExponentialBackoff(attempt int, initialDelay, maxDelay time.Duration, multiplier float64, jitter bool) time.Duration {
	delay := float64(initialDelay) * math.Pow(multiplier, float64(attempt-1))

	if delay > float64(maxDelay) {
		delay = float64(maxDelay)
	}

	if jitter {
		jitterAmount := rand.Float64() * 0.1 * delay
		delay += jitterAmount
	}

	return time.Duration(delay)
}

// LinearBackoff calculates linear backoff delay
func LinearBackoff(attempt int, initialDelay, maxDelay time.Duration, jitter bool) time.Duration {
	delay := float64(initialDelay) * float64(attempt)

	if delay > float64(maxDelay) {
		delay = float64(maxDelay)
	}

	if jitter {
		jitterAmount := rand.Float64() * 0.1 * delay
		delay += jitterAmount
	}

	return time.Duration(delay)
}

// FixedBackoff returns a fixed delay
func FixedBackoff(delay time.Duration, jitter bool) time.Duration {
	if jitter {
		jitterAmount := rand.Float64() * 0.1 * float64(delay)
		delay += time.Duration(jitterAmount)
	}
	return delay
}
