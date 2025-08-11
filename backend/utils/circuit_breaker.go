package utils

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState int

const (
	// StateClosed - circuit breaker is closed, requests are allowed
	StateClosed CircuitBreakerState = iota
	// StateOpen - circuit breaker is open, requests are rejected
	StateOpen
	// StateHalfOpen - circuit breaker is half-open, limited requests are allowed
	StateHalfOpen
)

// String returns the string representation of the circuit breaker state
func (s CircuitBreakerState) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// CircuitBreakerConfig holds configuration for a circuit breaker
type CircuitBreakerConfig struct {
	// MaxFailures is the maximum number of failures before opening the circuit
	MaxFailures int
	// Timeout is the duration to wait before transitioning from open to half-open
	Timeout time.Duration
	// MaxRequests is the maximum number of requests allowed in half-open state
	MaxRequests int
	// SuccessThreshold is the number of consecutive successes needed to close the circuit
	SuccessThreshold int
	// Name is the identifier for this circuit breaker
	Name string
}

// DefaultCircuitBreakerConfig returns a default configuration
func DefaultCircuitBreakerConfig(name string) *CircuitBreakerConfig {
	return &CircuitBreakerConfig{
		MaxFailures:      5,
		Timeout:          30 * time.Second,
		MaxRequests:      3,
		SuccessThreshold: 2,
		Name:             name,
	}
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	config           *CircuitBreakerConfig
	state            CircuitBreakerState
	failures         int
	successes        int
	requests         int
	lastFailureTime  time.Time
	lastSuccessTime  time.Time
	stateChangedTime time.Time
	mu               sync.RWMutex
	logger           *Logger
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config *CircuitBreakerConfig, logger *Logger) *CircuitBreaker {
	if config == nil {
		config = DefaultCircuitBreakerConfig("default")
	}
	if logger == nil {
		logger = GetLogger()
	}

	return &CircuitBreaker{
		config:           config,
		state:            StateClosed,
		stateChangedTime: time.Now(),
		logger:           logger,
	}
}

// Execute executes a function with circuit breaker protection
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func(context.Context) error) error {
	// Check if request is allowed
	if !cb.allowRequest() {
		return &CircuitBreakerError{
			State:   cb.GetState(),
			Message: fmt.Sprintf("circuit breaker %s is %s", cb.config.Name, cb.GetState()),
		}
	}

	// Execute the function
	err := fn(ctx)

	// Record the result
	if err != nil {
		cb.recordFailure()
		return err
	}

	cb.recordSuccess()
	return nil
}

// allowRequest determines if a request should be allowed
func (cb *CircuitBreaker) allowRequest() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		// Check if timeout has passed
		if time.Since(cb.stateChangedTime) >= cb.config.Timeout {
			cb.setState(StateHalfOpen)
			cb.requests = 0
			return true
		}
		return false
	case StateHalfOpen:
		// Allow limited requests in half-open state
		return cb.requests < cb.config.MaxRequests
	default:
		return false
	}
}

// recordSuccess records a successful request
func (cb *CircuitBreaker) recordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.lastSuccessTime = time.Now()

	switch cb.state {
	case StateClosed:
		// Reset failure count on success
		cb.failures = 0
	case StateHalfOpen:
		cb.successes++
		cb.requests++

		// Close circuit if enough successes
		if cb.successes >= cb.config.SuccessThreshold {
			cb.setState(StateClosed)
			cb.failures = 0
			cb.successes = 0
			cb.requests = 0
		}
	}

	cb.logger.WithSource("circuit_breaker").Debug("Request succeeded", map[string]interface{}{
		"circuit_breaker": cb.config.Name,
		"state":           cb.state.String(),
		"failures":        cb.failures,
		"successes":       cb.successes,
	})
}

// recordFailure records a failed request
func (cb *CircuitBreaker) recordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailureTime = time.Now()

	switch cb.state {
	case StateClosed:
		// Open circuit if max failures reached
		if cb.failures >= cb.config.MaxFailures {
			cb.setState(StateOpen)
		}
	case StateHalfOpen:
		cb.requests++
		// Go back to open state on any failure in half-open
		cb.setState(StateOpen)
		cb.successes = 0
		cb.requests = 0
	}

	cb.logger.WithSource("circuit_breaker").Warn("Request failed", map[string]interface{}{
		"circuit_breaker": cb.config.Name,
		"state":           cb.state.String(),
		"failures":        cb.failures,
		"successes":       cb.successes,
	})
}

// setState changes the circuit breaker state
func (cb *CircuitBreaker) setState(newState CircuitBreakerState) {
	if cb.state != newState {
		oldState := cb.state
		cb.state = newState
		cb.stateChangedTime = time.Now()

		cb.logger.WithSource("circuit_breaker").Info("Circuit breaker state changed", map[string]interface{}{
			"circuit_breaker": cb.config.Name,
			"old_state":       oldState.String(),
			"new_state":       newState.String(),
			"failures":        cb.failures,
		})
	}
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// GetStats returns statistics about the circuit breaker
func (cb *CircuitBreaker) GetStats() map[string]interface{} {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return map[string]interface{}{
		"name":               cb.config.Name,
		"state":              cb.state.String(),
		"failures":           cb.failures,
		"successes":          cb.successes,
		"requests":           cb.requests,
		"last_failure_time":  cb.lastFailureTime,
		"last_success_time":  cb.lastSuccessTime,
		"state_changed_time": cb.stateChangedTime,
		"max_failures":       cb.config.MaxFailures,
		"timeout":            cb.config.Timeout,
		"max_requests":       cb.config.MaxRequests,
		"success_threshold":  cb.config.SuccessThreshold,
	}
}

// Reset resets the circuit breaker to its initial state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = StateClosed
	cb.failures = 0
	cb.successes = 0
	cb.requests = 0
	cb.stateChangedTime = time.Now()

	cb.logger.WithSource("circuit_breaker").Info("Circuit breaker reset", map[string]interface{}{
		"circuit_breaker": cb.config.Name,
	})
}

// CircuitBreakerError represents an error from a circuit breaker
type CircuitBreakerError struct {
	State   CircuitBreakerState
	Message string
}

// Error implements the error interface
func (e *CircuitBreakerError) Error() string {
	return e.Message
}

// IsCircuitBreakerError checks if an error is a circuit breaker error
func IsCircuitBreakerError(err error) bool {
	var cbErr *CircuitBreakerError
	return errors.As(err, &cbErr)
}

// CircuitBreakerManager manages multiple circuit breakers
type CircuitBreakerManager struct {
	breakers map[string]*CircuitBreaker
	mu       sync.RWMutex
	logger   *Logger
}

// NewCircuitBreakerManager creates a new circuit breaker manager
func NewCircuitBreakerManager(logger *Logger) *CircuitBreakerManager {
	if logger == nil {
		logger = GetLogger()
	}

	return &CircuitBreakerManager{
		breakers: make(map[string]*CircuitBreaker),
		logger:   logger,
	}
}

// GetOrCreate gets an existing circuit breaker or creates a new one
func (cbm *CircuitBreakerManager) GetOrCreate(name string, config *CircuitBreakerConfig) *CircuitBreaker {
	cbm.mu.Lock()
	defer cbm.mu.Unlock()

	if cb, exists := cbm.breakers[name]; exists {
		return cb
	}

	if config == nil {
		config = DefaultCircuitBreakerConfig(name)
	}

	cb := NewCircuitBreaker(config, cbm.logger)
	cbm.breakers[name] = cb

	cbm.logger.WithSource("circuit_breaker_manager").Info("Circuit breaker created", map[string]interface{}{
		"name": name,
	})

	return cb
}

// Get gets an existing circuit breaker
func (cbm *CircuitBreakerManager) Get(name string) (*CircuitBreaker, bool) {
	cbm.mu.RLock()
	defer cbm.mu.RUnlock()

	cb, exists := cbm.breakers[name]
	return cb, exists
}

// GetAll returns all circuit breakers
func (cbm *CircuitBreakerManager) GetAll() map[string]*CircuitBreaker {
	cbm.mu.RLock()
	defer cbm.mu.RUnlock()

	result := make(map[string]*CircuitBreaker)
	for name, cb := range cbm.breakers {
		result[name] = cb
	}
	return result
}

// GetStats returns statistics for all circuit breakers
func (cbm *CircuitBreakerManager) GetStats() map[string]interface{} {
	cbm.mu.RLock()
	defer cbm.mu.RUnlock()

	stats := make(map[string]interface{})
	for name, cb := range cbm.breakers {
		stats[name] = cb.GetStats()
	}
	return stats
}

// Reset resets a specific circuit breaker
func (cbm *CircuitBreakerManager) Reset(name string) bool {
	cbm.mu.RLock()
	defer cbm.mu.RUnlock()

	if cb, exists := cbm.breakers[name]; exists {
		cb.Reset()
		return true
	}
	return false
}

// ResetAll resets all circuit breakers
func (cbm *CircuitBreakerManager) ResetAll() {
	cbm.mu.RLock()
	defer cbm.mu.RUnlock()

	for _, cb := range cbm.breakers {
		cb.Reset()
	}

	cbm.logger.WithSource("circuit_breaker_manager").Info("All circuit breakers reset")
}
