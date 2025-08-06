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
	"github.com/KBesada24/Full-Stack-Master-Sync.git/middleware"
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
			},
		})
	})

	logger.Info("Routes configured successfully", map[string]interface{}{
		"health_endpoint":    "/health",
		"api_base":           "/api",
		"websocket_endpoint": "/ws",
	})
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
