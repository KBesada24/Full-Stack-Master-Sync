package benchmarks

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/KBesada24/Full-Stack-Master-Sync.git/config"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/handlers"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/middleware"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/models"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/services"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/utils"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
)

// setupBenchmarkApp creates a Fiber app configured for benchmarking
func setupBenchmarkApp(b *testing.B) *fiber.App {
	// Load test configuration
	cfg := &config.Config{
		Environment:  "test",
		Port:         "8080",
		LogLevel:     "error", // Reduce logging for benchmarks
		LogFormat:    "json",
		OpenAIAPIKey: "test-key",
		FrontendURL:  "http://localhost:3000",
	}

	// Initialize logger with minimal logging
	utils.InitLogger(cfg.LogLevel, cfg.LogFormat)
	logger := utils.GetLogger()

	// Initialize WebSocket hub
	websocket.InitializeHub()

	// Create Fiber app
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	// Add minimal middleware for benchmarks
	app.Use(middleware.PerformanceMonitoring())

	// Initialize services
	wsHub := websocket.GetHub()
	aiService := services.NewAIService(cfg, wsHub, logger)
	syncService := services.NewSyncService(wsHub)
	testService := services.NewTestService(cfg, wsHub)
	logService := services.NewLogService(aiService, wsHub)

	// Initialize handlers
	aiHandler := handlers.NewAIHandler(aiService)
	syncHandler := handlers.NewSyncHandler(syncService)
	testingHandler := handlers.NewTestingHandler(testService)
	loggingHandler := handlers.NewLoggingHandler(logService)

	// Setup routes
	api := app.Group("/api")

	// AI routes
	ai := api.Group("/ai")
	ai.Post("/suggestions", aiHandler.GetCodeSuggestions)
	ai.Post("/analyze-logs", aiHandler.AnalyzeLogs)
	ai.Get("/status", aiHandler.GetAIStatus)

	// Sync routes
	sync := api.Group("/sync")
	sync.Post("/connect", syncHandler.ConnectEnvironment)
	sync.Get("/status", syncHandler.GetSyncStatus)
	sync.Post("/validate", syncHandler.ValidateEndpoint)

	// Testing routes
	testing := api.Group("/testing")
	testing.Post("/run", testingHandler.RunTests)
	testing.Get("/results/:runId", testingHandler.GetTestResults)
	testing.Post("/validate-sync", testingHandler.ValidateSync)

	// Logging routes
	logs := api.Group("/logs")
	logs.Post("/submit", loggingHandler.SubmitLogs)
	logs.Get("/analyze", loggingHandler.AnalyzeLogs)

	return app
}

// BenchmarkHealthEndpoint benchmarks the health check endpoint
func BenchmarkHealthEndpoint(b *testing.B) {
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":    "healthy",
			"timestamp": "2024-01-01T00:00:00Z",
		})
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/health", nil)
			resp, err := app.Test(req)
			if err != nil {
				b.Fatal(err)
			}
			resp.Body.Close()
		}
	})
}

// BenchmarkAICodeSuggestions benchmarks the AI code suggestions endpoint
func BenchmarkAICodeSuggestions(b *testing.B) {
	app := setupBenchmarkApp(b)

	requestBody := models.AIRequest{
		Code:        "function hello() { console.log('hello'); }",
		Language:    "javascript",
		Context:     "React component",
		RequestType: "suggestion",
	}

	bodyBytes, err := json.Marshal(requestBody)
	require.NoError(b, err)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("POST", "/api/ai/suggestions", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			if err != nil {
				b.Fatal(err)
			}
			resp.Body.Close()
		}
	})
}

// BenchmarkSyncConnect benchmarks the sync connect endpoint
func BenchmarkSyncConnect(b *testing.B) {
	app := setupBenchmarkApp(b)

	requestBody := models.SyncConnectionRequest{
		FrontendURL: "http://localhost:3000",
		BackendURL:  "http://localhost:8080",
		Environment: "development",
	}

	bodyBytes, err := json.Marshal(requestBody)
	require.NoError(b, err)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("POST", "/api/sync/connect", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			if err != nil {
				b.Fatal(err)
			}
			resp.Body.Close()
		}
	})
}

// BenchmarkSyncStatus benchmarks the sync status endpoint
func BenchmarkSyncStatus(b *testing.B) {
	app := setupBenchmarkApp(b)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/api/sync/status", nil)

			resp, err := app.Test(req)
			if err != nil {
				b.Fatal(err)
			}
			resp.Body.Close()
		}
	})
}

// BenchmarkTestRun benchmarks the test run endpoint
func BenchmarkTestRun(b *testing.B) {
	app := setupBenchmarkApp(b)

	requestBody := models.TestRunRequest{
		Framework:   "cypress",
		TestSuite:   "integration",
		Environment: "development",
		Config: map[string]string{
			"baseUrl": "http://localhost:3000",
		},
	}

	bodyBytes, err := json.Marshal(requestBody)
	require.NoError(b, err)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("POST", "/api/testing/run", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			if err != nil {
				b.Fatal(err)
			}
			resp.Body.Close()
		}
	})
}

// BenchmarkLogSubmit benchmarks the log submit endpoint
func BenchmarkLogSubmit(b *testing.B) {
	app := setupBenchmarkApp(b)

	requestBody := []models.LogEntry{
		{
			Level:   "info",
			Source:  "frontend",
			Message: "User action performed",
			Context: map[string]interface{}{
				"action": "button_click",
				"page":   "dashboard",
			},
		},
	}

	bodyBytes, err := json.Marshal(requestBody)
	require.NoError(b, err)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("POST", "/api/logs/submit", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			if err != nil {
				b.Fatal(err)
			}
			resp.Body.Close()
		}
	})
}

// BenchmarkPerformanceMiddleware benchmarks the performance monitoring middleware
func BenchmarkPerformanceMiddleware(b *testing.B) {
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	app.Use(middleware.PerformanceMonitoring())
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/test", nil)
			resp, err := app.Test(req)
			if err != nil {
				b.Fatal(err)
			}
			resp.Body.Close()
		}
	})
}

// BenchmarkRateLimitingMiddleware benchmarks the rate limiting middleware
func BenchmarkRateLimitingMiddleware(b *testing.B) {
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	rateLimitConfig := middleware.RateLimitConfig{
		RequestsPerSecond: 1000, // High limit for benchmarking
		BurstSize:         100,
	}

	app.Use(middleware.RateLimiting(rateLimitConfig))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/test", nil)
			resp, err := app.Test(req)
			if err != nil {
				b.Fatal(err)
			}
			resp.Body.Close()
		}
	})
}

// BenchmarkMemoryMonitoring benchmarks the memory monitoring middleware
func BenchmarkMemoryMonitoring(b *testing.B) {
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	app.Use(middleware.MemoryMonitoring())
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/test", nil)
			resp, err := app.Test(req)
			if err != nil {
				b.Fatal(err)
			}
			resp.Body.Close()
		}
	})
}

// BenchmarkConnectionPooling benchmarks the connection pooling middleware
func BenchmarkConnectionPooling(b *testing.B) {
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	app.Use(middleware.ConnectionPooling())
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/test", nil)
			resp, err := app.Test(req)
			if err != nil {
				b.Fatal(err)
			}
			resp.Body.Close()
		}
	})
}

// BenchmarkFullMiddlewareStack benchmarks the complete middleware stack
func BenchmarkFullMiddlewareStack(b *testing.B) {
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	// Add all performance-related middleware
	app.Use(middleware.PerformanceMonitoring())
	app.Use(middleware.MemoryMonitoring())
	app.Use(middleware.ConnectionPooling())

	rateLimitConfig := middleware.RateLimitConfig{
		RequestsPerSecond: 1000,
		BurstSize:         100,
	}
	app.Use(middleware.RateLimiting(rateLimitConfig))

	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/test", nil)
			resp, err := app.Test(req)
			if err != nil {
				b.Fatal(err)
			}
			resp.Body.Close()
		}
	})
}

// BenchmarkJSONSerialization benchmarks JSON serialization performance
func BenchmarkJSONSerialization(b *testing.B) {
	data := models.SyncStatusResponse{
		Status:    "connected",
		Connected: true,
		Environments: map[string]string{
			"development": "http://localhost:3000",
			"staging":     "http://staging.example.com",
			"production":  "http://production.example.com",
		},
		Health: models.HealthStatus{
			Frontend: true,
			Backend:  true,
			Database: true,
			Message:  "All systems operational",
		},
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := json.Marshal(data)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkConcurrentRequests benchmarks concurrent request handling
func BenchmarkConcurrentRequests(b *testing.B) {
	app := setupBenchmarkApp(b)

	b.ResetTimer()
	b.SetParallelism(100) // High concurrency
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/api/sync/status", nil)
			resp, err := app.Test(req)
			if err != nil {
				b.Fatal(err)
			}
			resp.Body.Close()
		}
	})
}
