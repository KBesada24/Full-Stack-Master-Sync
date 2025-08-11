package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/KBesada24/Full-Stack-Master-Sync.git/config"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/handlers"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/middleware"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/services"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/utils"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	websocket2 "github.com/gofiber/websocket/v2"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Load configuration
	cfg := config.Load()

	// Validate configuration
	if errors := cfg.Validate(); len(errors) > 0 {
		log.Fatal("Configuration validation failed:", errors)
	}

	// Initialize logger
	logger := utils.GetLogger()
	logger.Info("Starting Full Stack Master Sync Backend", map[string]interface{}{
		"version":     "1.0.0",
		"environment": cfg.Environment,
		"port":        cfg.Port,
	})

	// Initialize WebSocket hub
	websocket.InitializeHub()

	// Create Fiber app with configuration
	app := createFiberApp(cfg, logger)

	// Setup middleware
	setupMiddleware(app, cfg, logger)

	// Setup routes
	setupRoutes(app, cfg, logger)

	// Start server with graceful shutdown
	startServerWithGracefulShutdown(app, cfg, logger)
}

// createFiberApp creates and configures the Fiber application
func createFiberApp(cfg *config.Config, logger *utils.Logger) *fiber.App {
	return fiber.New(fiber.Config{
		AppName:      "Full Stack Master Sync Backend v1.0.0",
		ServerHeader: "Full-Stack-Master-Sync",
		ErrorHandler: createErrorHandler(logger),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
		BodyLimit:    10 * 1024 * 1024, // 10MB
		JSONEncoder:  utils.JSONMarshal,
		JSONDecoder:  utils.JSONUnmarshal,
	})
}

// setupMiddleware configures all middleware for the application
func setupMiddleware(app *fiber.App, cfg *config.Config, logger *utils.Logger) {
	// Recovery middleware (should be first)
	app.Use(recover.New(recover.Config{
		EnableStackTrace: cfg.IsDevelopment(),
	}))

	// Correlation ID middleware
	app.Use(middleware.CorrelationID())

	// CORS middleware
	corsOrigins := []string{cfg.FrontendURL}
	if cfg.IsDevelopment() {
		corsOrigins = append(corsOrigins, "http://localhost:3000", "http://127.0.0.1:3000")
	}
	app.Use(middleware.CORSWithOrigins(corsOrigins))

	// Request validation middleware
	app.Use(middleware.RequestValidation())

	// Request logging middleware
	app.Use(middleware.RequestLogging())

	// Structured logging middleware
	app.Use(middleware.StructuredLogging(logger))

	// Access log middleware
	app.Use(middleware.AccessLog(logger))

	// Error logging middleware
	app.Use(middleware.ErrorLogging(logger))
}

// setupRoutes configures all routes for the application
func setupRoutes(app *fiber.App, cfg *config.Config, logger *utils.Logger) {
	// Health check endpoint
	app.Get("/health", healthCheckHandler(cfg))

	// WebSocket endpoint
	app.Use("/ws", websocket.WebSocketUpgrade)
	app.Get("/ws", websocket2.New(websocket.WebSocketHandler))

	// WebSocket stats endpoint
	app.Get("/ws/stats", func(c *fiber.Ctx) error {
		stats := websocket.GetWebSocketStats()
		return utils.SuccessResponse(c, "WebSocket statistics", stats)
	})

	// API version group
	api := app.Group("/api")

	// Initialize services
	aiService := services.NewAIService(cfg)
	syncService := services.NewSyncService()
	testService := services.NewTestService(cfg, websocket.GetHub())
	logService := services.NewLogService(aiService, websocket.GetHub())

	// Initialize handlers
	aiHandler := handlers.NewAIHandler(aiService)
	syncHandler := handlers.NewSyncHandler(syncService)
	testingHandler := handlers.NewTestingHandler(testService)
	loggingHandler := handlers.NewLoggingHandler(logService)

	// Setup AI routes
	setupAIRoutes(api, aiHandler)

	// Setup Sync routes
	setupSyncRoutes(api, syncHandler)

	// Setup Testing routes
	setupTestingRoutes(api, testingHandler)

	// Setup Logging routes
	setupLoggingRoutes(api, loggingHandler)

	// Add API routes here (will be implemented in future tasks)
	// For now, just add a basic API info endpoint
	api.Get("/", func(c *fiber.Ctx) error {
		return utils.SuccessResponse(c, "Full Stack Master Sync API", fiber.Map{
			"version":     "1.0.0",
			"environment": cfg.Environment,
			"endpoints": []string{
				"GET /health - Health check",
				"GET /api - API information",
				"GET /ws - WebSocket connection",
				"GET /ws/stats - WebSocket statistics",
				"POST /api/ai/suggestions - Get AI code suggestions",
				"POST /api/ai/analyze-logs - Analyze logs with AI",
				"GET /api/ai/status - Get AI service status",
				"GET /api/ai/health - AI service health check",
				"POST /api/sync/connect - Connect to sync environment",
				"GET /api/sync/status - Get sync status",
				"POST /api/sync/validate - Validate endpoint compatibility",
				"GET /api/sync/environments - Get all environments",
				"DELETE /api/sync/environments/:name - Remove environment",
				"POST /api/testing/run - Trigger test execution",
				"GET /api/testing/results/:runId - Get test results",
				"POST /api/testing/validate-sync - Validate API-UI synchronization",
				"GET /api/testing/active - Get active test runs",
				"GET /api/testing/history - Get test run history",
				"DELETE /api/testing/runs/:runId - Cancel test run",
				"GET /api/testing/status - Get testing service status",
				"GET /api/testing/health - Testing service health check",
				"POST /api/logs/submit - Submit log entries",
				"GET /api/logs/analyze - Analyze logs and detect patterns",
				"GET /api/logs/stats - Get log statistics",
				"DELETE /api/logs/clear - Clear all logs",
				"GET /api/logs/status - Get logging service status",
				"GET /api/logs/health - Logging service health check",
			},
		})
	})

	logger.Info("Routes configured successfully", map[string]interface{}{
		"health_endpoint":    "/health",
		"api_base":           "/api",
		"websocket_endpoint": "/ws",
	})
}

// setupAIRoutes configures AI-related routes
func setupAIRoutes(api fiber.Router, aiHandler *handlers.AIHandler) {
	// AI routes group
	ai := api.Group("/ai")

	// AI assistance endpoints
	ai.Post("/suggestions", aiHandler.GetCodeSuggestions)
	ai.Post("/analyze-logs", aiHandler.AnalyzeLogs)
	ai.Get("/status", aiHandler.GetAIStatus)
	ai.Get("/health", aiHandler.HealthCheck)
}

// setupSyncRoutes configures sync-related routes
func setupSyncRoutes(api fiber.Router, syncHandler *handlers.SyncHandler) {
	// Sync routes group
	sync := api.Group("/sync")

	// Environment sync endpoints
	sync.Post("/connect", syncHandler.ConnectEnvironment)
	sync.Get("/status", syncHandler.GetSyncStatus)
	sync.Post("/validate", syncHandler.ValidateEndpoint)
	sync.Get("/environments", syncHandler.GetEnvironments)
	sync.Delete("/environments/:name", syncHandler.RemoveEnvironment)
}

// setupTestingRoutes configures testing-related routes
func setupTestingRoutes(api fiber.Router, testingHandler *handlers.TestingHandler) {
	// Testing routes group
	testing := api.Group("/testing")

	// Core testing endpoints
	testing.Post("/run", testingHandler.RunTests)
	testing.Get("/results/:runId", testingHandler.GetTestResults)
	testing.Post("/validate-sync", testingHandler.ValidateSync)

	// Additional testing endpoints
	testing.Get("/active", testingHandler.GetActiveRuns)
	testing.Get("/history", testingHandler.GetRunHistory)
	testing.Delete("/runs/:runId", testingHandler.CancelTestRun)
	testing.Get("/status", testingHandler.GetTestingStatus)
	testing.Get("/health", testingHandler.HealthCheck)
}

// setupLoggingRoutes configures logging-related routes
func setupLoggingRoutes(api fiber.Router, loggingHandler *handlers.LoggingHandler) {
	// Logging routes group
	logs := api.Group("/logs")

	// Core logging endpoints
	logs.Post("/submit", loggingHandler.SubmitLogs)
	logs.Get("/analyze", loggingHandler.AnalyzeLogs)
	logs.Get("/stats", loggingHandler.GetLogStats)
	logs.Delete("/clear", loggingHandler.ClearLogs)
	logs.Get("/status", loggingHandler.GetLoggingStatus)
	logs.Get("/health", loggingHandler.HealthCheck)
}

// healthCheckHandler creates the health check endpoint handler
func healthCheckHandler(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Perform basic health checks
		health := fiber.Map{
			"status":      "healthy",
			"message":     "Full Stack Master Sync Backend is running",
			"version":     "1.0.0",
			"environment": cfg.Environment,
			"timestamp":   time.Now().UTC(),
			"uptime":      time.Since(startTime).String(),
			"checks": fiber.Map{
				"server": "ok",
				"config": "ok",
			},
		}

		// Add additional health checks here in future tasks
		// For now, basic server health is sufficient

		return utils.SuccessResponse(c, "Health check passed", health)
	}
}

// createErrorHandler creates a custom error handler for Fiber
func createErrorHandler(logger *utils.Logger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		// Default to 500 server error
		code := fiber.StatusInternalServerError
		message := "Internal Server Error"

		// Check if it's a Fiber error
		if e, ok := err.(*fiber.Error); ok {
			code = e.Code
			message = e.Message
		}

		// Log the error
		traceID := utils.GetTraceID(c)
		logger.WithTraceID(traceID).WithSource("error_handler").Error(
			"Request error", err, map[string]interface{}{
				"method":     c.Method(),
				"path":       c.Path(),
				"status":     code,
				"ip":         c.IP(),
				"user_agent": c.Get("User-Agent"),
			})

		// Return error response
		return utils.ErrorResponse(c, code, "REQUEST_ERROR", message, nil)
	}
}

// startServerWithGracefulShutdown starts the server with graceful shutdown handling
func startServerWithGracefulShutdown(app *fiber.App, cfg *config.Config, logger *utils.Logger) {
	// Channel to listen for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		address := ":" + cfg.Port
		logger.Info("Server starting", map[string]interface{}{
			"address":     address,
			"environment": cfg.Environment,
		})

		fmt.Printf("🚀 Server starting on port %s\n", cfg.Port)
		fmt.Printf("📊 Health check available at: http://localhost:%s/health\n", cfg.Port)
		fmt.Printf("🔗 API base URL: http://localhost:%s/api\n", cfg.Port)

		if err := app.Listen(address); err != nil {
			logger.Error("Server failed to start", err, map[string]interface{}{
				"address": address,
			})
			log.Fatal("Server failed to start:", err)
		}
	}()

	// Wait for interrupt signal
	<-quit
	logger.Info("Shutdown signal received, starting graceful shutdown...")

	// Create a context with timeout for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown server
	if err := app.ShutdownWithContext(ctx); err != nil {
		logger.Error("Server forced to shutdown", err, nil)
		log.Fatal("Server forced to shutdown:", err)
	}

	logger.Info("Server shutdown completed successfully")
	fmt.Println("✅ Server shutdown completed")
}

// Global variable to track server start time for uptime calculation
var startTime = time.Now()
