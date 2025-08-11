package utils

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRecoveryHandler_BasicRecovery(t *testing.T) {
	handler := NewRecoveryHandler(nil)

	cleanupCalled := false
	handler.RegisterCleanup(func() error {
		cleanupCalled = true
		return nil
	})

	// Test panic recovery
	func() {
		defer handler.Recover()
		panic("test panic")
	}()

	// Should not panic and cleanup should be called
	assert.True(t, cleanupCalled)

	stats := handler.GetStats()
	assert.Equal(t, 1, stats["panic_count"])
}

func TestRecoveryHandler_MultipleCleanups(t *testing.T) {
	handler := NewRecoveryHandler(nil)

	cleanup1Called := false
	cleanup2Called := false

	handler.RegisterCleanup(func() error {
		cleanup1Called = true
		return nil
	})

	handler.RegisterCleanup(func() error {
		cleanup2Called = true
		return nil
	})

	// Test panic recovery
	func() {
		defer handler.Recover()
		panic("test panic")
	}()

	assert.True(t, cleanup1Called)
	assert.True(t, cleanup2Called)
}

func TestRecoveryHandler_CleanupError(t *testing.T) {
	handler := NewRecoveryHandler(nil)

	cleanupError := errors.New("cleanup failed")
	handler.RegisterCleanup(func() error {
		return cleanupError
	})

	// Should not panic even if cleanup fails
	func() {
		defer handler.Recover()
		panic("test panic")
	}()

	stats := handler.GetStats()
	assert.Equal(t, 1, stats["panic_count"])
}

func TestRecoveryHandler_CleanupPanic(t *testing.T) {
	handler := NewRecoveryHandler(nil)

	handler.RegisterCleanup(func() error {
		panic("cleanup panic")
	})

	// Should not panic even if cleanup panics
	func() {
		defer handler.Recover()
		panic("test panic")
	}()

	stats := handler.GetStats()
	assert.Equal(t, 1, stats["panic_count"])
}

func TestRecoveryHandler_WithCallback(t *testing.T) {
	handler := NewRecoveryHandler(nil)

	var callbackValue interface{}

	func() {
		defer handler.RecoverWithCallback(func(r interface{}) {
			callbackValue = r
		})
		panic("test panic")
	}()

	assert.Equal(t, "test panic", callbackValue)
}

func TestRecoveryHandler_PanicStorm(t *testing.T) {
	handler := NewRecoveryHandler(nil)
	handler.maxPanics = 3
	handler.panicWindow = 100 * time.Millisecond

	// Trigger multiple panics quickly
	for i := 0; i < 5; i++ {
		func() {
			defer handler.Recover()
			panic("storm panic")
		}()
	}

	stats := handler.GetStats()
	assert.Equal(t, 5, stats["panic_count"])
	assert.True(t, stats["is_panic_storm"].(bool))
}

func TestRecoveryHandler_Reset(t *testing.T) {
	handler := NewRecoveryHandler(nil)

	// Trigger a panic
	func() {
		defer handler.Recover()
		panic("test panic")
	}()

	stats := handler.GetStats()
	assert.Equal(t, 1, stats["panic_count"])

	// Reset
	handler.Reset()

	stats = handler.GetStats()
	assert.Equal(t, 0, stats["panic_count"])
}

func TestGracefulShutdown_BasicShutdown(t *testing.T) {
	shutdown := NewGracefulShutdown(1*time.Second, nil)

	shutdown1Called := false
	shutdown2Called := false

	shutdown.RegisterShutdown(func(ctx context.Context) error {
		shutdown1Called = true
		return nil
	})

	shutdown.RegisterShutdown(func(ctx context.Context) error {
		shutdown2Called = true
		return nil
	})

	ctx := context.Background()
	err := shutdown.Shutdown(ctx)

	assert.NoError(t, err)
	assert.True(t, shutdown1Called)
	assert.True(t, shutdown2Called)
}

func TestGracefulShutdown_WithErrors(t *testing.T) {
	shutdown := NewGracefulShutdown(1*time.Second, nil)

	shutdownError := errors.New("shutdown failed")
	shutdown.RegisterShutdown(func(ctx context.Context) error {
		return shutdownError
	})

	shutdown.RegisterShutdown(func(ctx context.Context) error {
		return nil
	})

	ctx := context.Background()
	err := shutdown.Shutdown(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "shutdown completed with")
}

func TestGracefulShutdown_WithPanic(t *testing.T) {
	shutdown := NewGracefulShutdown(1*time.Second, nil)

	shutdown.RegisterShutdown(func(ctx context.Context) error {
		panic("shutdown panic")
	})

	normalShutdownCalled := false
	shutdown.RegisterShutdown(func(ctx context.Context) error {
		normalShutdownCalled = true
		return nil
	})

	ctx := context.Background()
	err := shutdown.Shutdown(ctx)

	assert.Error(t, err)
	assert.True(t, normalShutdownCalled) // Should continue with other shutdowns
}

func TestGracefulShutdown_Timeout(t *testing.T) {
	shutdown := NewGracefulShutdown(100*time.Millisecond, nil)

	shutdown.RegisterShutdown(func(ctx context.Context) error {
		time.Sleep(200 * time.Millisecond) // Longer than timeout
		return nil
	})

	ctx := context.Background()
	err := shutdown.Shutdown(ctx)

	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
}

func TestErrorRecoveryService_Integration(t *testing.T) {
	service := NewErrorRecoveryService(nil)

	// Register cleanup
	cleanupCalled := false
	service.RegisterCleanup(func() error {
		cleanupCalled = true
		return nil
	})

	// Register shutdown
	shutdownCalled := false
	service.RegisterShutdown(func(ctx context.Context) error {
		shutdownCalled = true
		return nil
	})

	// Register health check
	service.RegisterHealthCheck("test", func(ctx context.Context) error {
		return nil
	})

	// Test recovery
	func() {
		defer service.Recover()
		panic("test panic")
	}()

	assert.True(t, cleanupCalled)

	// Test health checks
	ctx := context.Background()
	healthResults := service.PerformHealthChecks(ctx)
	assert.Contains(t, healthResults, "test")
	assert.NoError(t, healthResults["test"])

	// Test shutdown
	err := service.Shutdown(ctx)
	assert.NoError(t, err)
	assert.True(t, shutdownCalled)

	// Test circuit breaker
	cb := service.GetCircuitBreaker("test", nil)
	assert.NotNil(t, cb)
	assert.Equal(t, "test", cb.config.Name)

	// Test stats
	stats := service.GetStats()
	assert.Contains(t, stats, "recovery_handler")
	assert.Contains(t, stats, "circuit_breakers")
	assert.Contains(t, stats, "health_checks")
}

func TestErrorRecoveryService_HealthCheckPanic(t *testing.T) {
	service := NewErrorRecoveryService(nil)

	service.RegisterHealthCheck("panic_check", func(ctx context.Context) error {
		panic("health check panic")
	})

	service.RegisterHealthCheck("normal_check", func(ctx context.Context) error {
		return nil
	})

	ctx := context.Background()
	healthResults := service.PerformHealthChecks(ctx)

	assert.Contains(t, healthResults, "panic_check")
	assert.Contains(t, healthResults, "normal_check")
	assert.Error(t, healthResults["panic_check"])
	assert.NoError(t, healthResults["normal_check"])
	assert.Contains(t, healthResults["panic_check"].Error(), "health check panicked")
}

func TestErrorRecoveryService_HealthCheckError(t *testing.T) {
	service := NewErrorRecoveryService(nil)

	healthError := errors.New("health check failed")
	service.RegisterHealthCheck("failing_check", func(ctx context.Context) error {
		return healthError
	})

	ctx := context.Background()
	healthResults := service.PerformHealthChecks(ctx)

	assert.Contains(t, healthResults, "failing_check")
	assert.Equal(t, healthError, healthResults["failing_check"])
}

func TestErrorRecoveryService_ConcurrentAccess(t *testing.T) {
	service := NewErrorRecoveryService(nil)

	const numGoroutines = 10
	const numOperations = 100

	// Test concurrent circuit breaker access
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()

			for j := 0; j < numOperations; j++ {
				cbName := fmt.Sprintf("cb_%d", id%3) // Use 3 different circuit breakers
				cb := service.GetCircuitBreaker(cbName, nil)

				ctx := context.Background()
				cb.Execute(ctx, func(ctx context.Context) error {
					return nil
				})
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify circuit breakers were created
	stats := service.GetStats()
	cbStats := stats["circuit_breakers"].(map[string]interface{})
	assert.GreaterOrEqual(t, len(cbStats), 3)
}

func BenchmarkRecoveryHandler_Recover(b *testing.B) {
	handler := NewRecoveryHandler(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		func() {
			defer handler.Recover()
			// No panic - just measure overhead
		}()
	}
}

func BenchmarkErrorRecoveryService_GetCircuitBreaker(b *testing.B) {
	service := NewErrorRecoveryService(nil)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			cbName := fmt.Sprintf("cb_%d", i%10)
			service.GetCircuitBreaker(cbName, nil)
			i++
		}
	})
}
