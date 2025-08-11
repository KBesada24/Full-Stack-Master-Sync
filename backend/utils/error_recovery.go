package utils

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"
)

// RecoveryHandler handles panic recovery and error cleanup
type RecoveryHandler struct {
	logger        *Logger
	cleanupFuncs  []func() error
	mu            sync.RWMutex
	panicCount    int
	lastPanicTime time.Time
	maxPanics     int
	panicWindow   time.Duration
}

// NewRecoveryHandler creates a new recovery handler
func NewRecoveryHandler(logger *Logger) *RecoveryHandler {
	if logger == nil {
		logger = GetLogger()
	}

	return &RecoveryHandler{
		logger:       logger,
		cleanupFuncs: make([]func() error, 0),
		maxPanics:    10,
		panicWindow:  5 * time.Minute,
	}
}

// RegisterCleanup registers a cleanup function to be called during recovery
func (rh *RecoveryHandler) RegisterCleanup(cleanup func() error) {
	rh.mu.Lock()
	defer rh.mu.Unlock()
	rh.cleanupFuncs = append(rh.cleanupFuncs, cleanup)
}

// Recover handles panic recovery with cleanup
func (rh *RecoveryHandler) Recover() {
	if r := recover(); r != nil {
		rh.handlePanic(r)
	}
}

// RecoverWithCallback handles panic recovery with a custom callback
func (rh *RecoveryHandler) RecoverWithCallback(callback func(interface{})) {
	if r := recover(); r != nil {
		rh.handlePanic(r)
		if callback != nil {
			callback(r)
		}
	}
}

// handlePanic processes a panic and performs cleanup
func (rh *RecoveryHandler) handlePanic(panicValue interface{}) {
	rh.mu.Lock()
	defer rh.mu.Unlock()

	rh.panicCount++
	rh.lastPanicTime = time.Now()

	// Get stack trace
	stackTrace := make([]byte, 4096)
	stackSize := runtime.Stack(stackTrace, false)
	stackTrace = stackTrace[:stackSize]

	// Log the panic
	rh.logger.WithSource("recovery_handler").Error("Panic recovered", fmt.Errorf("panic: %v", panicValue), map[string]interface{}{
		"panic_value": panicValue,
		"stack_trace": string(stackTrace),
		"panic_count": rh.panicCount,
		"last_panic":  rh.lastPanicTime,
	})

	// Check if we're in a panic storm
	if rh.isPanicStorm() {
		rh.logger.WithSource("recovery_handler").Error("Panic storm detected", nil, map[string]interface{}{
			"panic_count":  rh.panicCount,
			"panic_window": rh.panicWindow,
			"max_panics":   rh.maxPanics,
		})
	}

	// Execute cleanup functions
	rh.executeCleanup()
}

// isPanicStorm checks if we're experiencing a panic storm
func (rh *RecoveryHandler) isPanicStorm() bool {
	return rh.panicCount >= rh.maxPanics && time.Since(rh.lastPanicTime) <= rh.panicWindow
}

// executeCleanup executes all registered cleanup functions
func (rh *RecoveryHandler) executeCleanup() {
	for i, cleanup := range rh.cleanupFuncs {
		func() {
			defer func() {
				if r := recover(); r != nil {
					rh.logger.WithSource("recovery_handler").Error("Cleanup function panicked", fmt.Errorf("cleanup panic: %v", r), map[string]interface{}{
						"cleanup_index": i,
					})
				}
			}()

			if err := cleanup(); err != nil {
				rh.logger.WithSource("recovery_handler").Error("Cleanup function failed", err, map[string]interface{}{
					"cleanup_index": i,
				})
			}
		}()
	}
}

// GetStats returns recovery statistics
func (rh *RecoveryHandler) GetStats() map[string]interface{} {
	rh.mu.RLock()
	defer rh.mu.RUnlock()

	return map[string]interface{}{
		"panic_count":     rh.panicCount,
		"last_panic_time": rh.lastPanicTime,
		"cleanup_funcs":   len(rh.cleanupFuncs),
		"max_panics":      rh.maxPanics,
		"panic_window":    rh.panicWindow,
		"is_panic_storm":  rh.isPanicStorm(),
	}
}

// Reset resets the panic statistics
func (rh *RecoveryHandler) Reset() {
	rh.mu.Lock()
	defer rh.mu.Unlock()

	rh.panicCount = 0
	rh.lastPanicTime = time.Time{}

	rh.logger.WithSource("recovery_handler").Info("Recovery handler statistics reset")
}

// GracefulShutdown handles graceful shutdown with cleanup
type GracefulShutdown struct {
	shutdownFuncs []func(context.Context) error
	timeout       time.Duration
	logger        *Logger
	mu            sync.RWMutex
}

// NewGracefulShutdown creates a new graceful shutdown handler
func NewGracefulShutdown(timeout time.Duration, logger *Logger) *GracefulShutdown {
	if logger == nil {
		logger = GetLogger()
	}
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &GracefulShutdown{
		shutdownFuncs: make([]func(context.Context) error, 0),
		timeout:       timeout,
		logger:        logger,
	}
}

// RegisterShutdown registers a shutdown function
func (gs *GracefulShutdown) RegisterShutdown(shutdownFunc func(context.Context) error) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	gs.shutdownFuncs = append(gs.shutdownFuncs, shutdownFunc)
}

// Shutdown performs graceful shutdown
func (gs *GracefulShutdown) Shutdown(ctx context.Context) error {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	// Create context with timeout
	shutdownCtx, cancel := context.WithTimeout(ctx, gs.timeout)
	defer cancel()

	gs.logger.WithSource("graceful_shutdown").Info("Starting graceful shutdown", map[string]interface{}{
		"shutdown_funcs": len(gs.shutdownFuncs),
		"timeout":        gs.timeout,
	})

	var errors []error

	// Execute shutdown functions in reverse order (LIFO)
	for i := len(gs.shutdownFuncs) - 1; i >= 0; i-- {
		shutdownFunc := gs.shutdownFuncs[i]

		func() {
			defer func() {
				if r := recover(); r != nil {
					err := fmt.Errorf("shutdown function panicked: %v", r)
					errors = append(errors, err)
					gs.logger.WithSource("graceful_shutdown").Error("Shutdown function panicked", err, map[string]interface{}{
						"function_index": i,
					})
				}
			}()

			if err := shutdownFunc(shutdownCtx); err != nil {
				errors = append(errors, err)
				gs.logger.WithSource("graceful_shutdown").Error("Shutdown function failed", err, map[string]interface{}{
					"function_index": i,
				})
			}
		}()

		// Check if context is cancelled
		select {
		case <-shutdownCtx.Done():
			gs.logger.WithSource("graceful_shutdown").Warn("Shutdown timeout reached", map[string]interface{}{
				"remaining_functions": i,
			})
			return shutdownCtx.Err()
		default:
		}
	}

	if len(errors) > 0 {
		gs.logger.WithSource("graceful_shutdown").Error("Graceful shutdown completed with errors", nil, map[string]interface{}{
			"error_count": len(errors),
		})
		return fmt.Errorf("shutdown completed with %d errors", len(errors))
	}

	gs.logger.WithSource("graceful_shutdown").Info("Graceful shutdown completed successfully")
	return nil
}

// ErrorRecoveryService provides comprehensive error recovery capabilities
type ErrorRecoveryService struct {
	recoveryHandler  *RecoveryHandler
	gracefulShutdown *GracefulShutdown
	circuitBreakers  *CircuitBreakerManager
	logger           *Logger
	healthChecks     map[string]func(context.Context) error
	healthCheckMu    sync.RWMutex
}

// NewErrorRecoveryService creates a new error recovery service
func NewErrorRecoveryService(logger *Logger) *ErrorRecoveryService {
	if logger == nil {
		logger = GetLogger()
	}

	return &ErrorRecoveryService{
		recoveryHandler:  NewRecoveryHandler(logger),
		gracefulShutdown: NewGracefulShutdown(30*time.Second, logger),
		circuitBreakers:  NewCircuitBreakerManager(logger),
		logger:           logger,
		healthChecks:     make(map[string]func(context.Context) error),
	}
}

// RegisterCleanup registers a cleanup function
func (ers *ErrorRecoveryService) RegisterCleanup(cleanup func() error) {
	ers.recoveryHandler.RegisterCleanup(cleanup)
}

// RegisterShutdown registers a shutdown function
func (ers *ErrorRecoveryService) RegisterShutdown(shutdownFunc func(context.Context) error) {
	ers.gracefulShutdown.RegisterShutdown(shutdownFunc)
}

// RegisterHealthCheck registers a health check function
func (ers *ErrorRecoveryService) RegisterHealthCheck(name string, healthCheck func(context.Context) error) {
	ers.healthCheckMu.Lock()
	defer ers.healthCheckMu.Unlock()
	ers.healthChecks[name] = healthCheck
}

// GetCircuitBreaker gets or creates a circuit breaker
func (ers *ErrorRecoveryService) GetCircuitBreaker(name string, config *CircuitBreakerConfig) *CircuitBreaker {
	return ers.circuitBreakers.GetOrCreate(name, config)
}

// Recover handles panic recovery
func (ers *ErrorRecoveryService) Recover() {
	ers.recoveryHandler.Recover()
}

// RecoverWithCallback handles panic recovery with callback
func (ers *ErrorRecoveryService) RecoverWithCallback(callback func(interface{})) {
	ers.recoveryHandler.RecoverWithCallback(callback)
}

// Shutdown performs graceful shutdown
func (ers *ErrorRecoveryService) Shutdown(ctx context.Context) error {
	return ers.gracefulShutdown.Shutdown(ctx)
}

// PerformHealthChecks performs all registered health checks
func (ers *ErrorRecoveryService) PerformHealthChecks(ctx context.Context) map[string]error {
	ers.healthCheckMu.RLock()
	defer ers.healthCheckMu.RUnlock()

	results := make(map[string]error)

	for name, healthCheck := range ers.healthChecks {
		func() {
			defer func() {
				if r := recover(); r != nil {
					results[name] = fmt.Errorf("health check panicked: %v", r)
				}
			}()

			results[name] = healthCheck(ctx)
		}()
	}

	return results
}

// GetStats returns comprehensive error recovery statistics
func (ers *ErrorRecoveryService) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"recovery_handler": ers.recoveryHandler.GetStats(),
		"circuit_breakers": ers.circuitBreakers.GetStats(),
		"health_checks":    len(ers.healthChecks),
	}
}
