package handlers

import (
	"os"
	"runtime"
	"strings"

	"github.com/KBesada24/Full-Stack-Master-Sync.git/config"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/utils"
	"github.com/gofiber/fiber/v2"
)

// DebugHandler handles debug-related endpoints
type DebugHandler struct {
	config *config.Config
}

// NewDebugHandler creates a new debug handler
func NewDebugHandler(cfg *config.Config) *DebugHandler {
	return &DebugHandler{
		config: cfg,
	}
}

// GetConfig returns the current configuration (with sensitive values masked)
func (h *DebugHandler) GetConfig(c *fiber.Ctx) error {
	// Mask sensitive values
	maskedConfig := fiber.Map{
		"server": fiber.Map{
			"port":        h.config.Port,
			"host":        h.config.Host,
			"environment": h.config.Environment,
		},
		"openai": fiber.Map{
			"api_key_set": h.config.OpenAIAPIKey != "",
			"api_key":     maskSensitiveValue(h.config.OpenAIAPIKey),
		},
		"cors": fiber.Map{
			"frontend_url": h.config.FrontendURL,
		},
		"websocket": fiber.Map{
			"endpoint": h.config.WSEndpoint,
		},
		"logging": fiber.Map{
			"level":  h.config.LogLevel,
			"format": h.config.LogFormat,
		},
		"testing": fiber.Map{
			"cypress_base_url":    h.config.CypressBaseURL,
			"playwright_base_url": h.config.PlaywrightBaseURL,
		},
		"feature_toggles": fiber.Map{
			"ai_features":            h.config.EnableAIFeatures,
			"websocket":              h.config.EnableWebSocket,
			"performance_monitoring": h.config.EnablePerformanceMonitoring,
			"rate_limiting":          h.config.EnableRateLimiting,
			"circuit_breaker":        h.config.EnableCircuitBreaker,
			"detailed_errors":        h.config.EnableDetailedErrors,
			"debug_endpoints":        h.config.EnableDebugEndpoints,
		},
	}

	return utils.SuccessResponse(c, "Configuration retrieved", maskedConfig)
}

// GetRoutes returns all registered routes
func (h *DebugHandler) GetRoutes(c *fiber.Ctx) error {
	app := c.App()
	routes := []fiber.Map{}

	// Iterate through all routes
	for _, route := range app.GetRoutes() {
		routes = append(routes, fiber.Map{
			"method": route.Method,
			"path":   route.Path,
			"name":   route.Name,
		})
	}

	return utils.SuccessResponse(c, "Routes retrieved", fiber.Map{
		"total":  len(routes),
		"routes": routes,
	})
}

// GetEnvironment returns environment variables (with sensitive values masked)
func (h *DebugHandler) GetEnvironment(c *fiber.Ctx) error {
	env := os.Environ()
	maskedEnv := make(map[string]string)

	// Sensitive keys that should be masked
	sensitiveKeys := []string{
		"API_KEY",
		"SECRET",
		"PASSWORD",
		"TOKEN",
		"PRIVATE",
		"CREDENTIAL",
	}

	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		value := parts[1]

		// Check if key contains sensitive information
		isSensitive := false
		for _, sensitiveKey := range sensitiveKeys {
			if strings.Contains(strings.ToUpper(key), sensitiveKey) {
				isSensitive = true
				break
			}
		}

		if isSensitive {
			maskedEnv[key] = maskSensitiveValue(value)
		} else {
			maskedEnv[key] = value
		}
	}

	return utils.SuccessResponse(c, "Environment variables retrieved", fiber.Map{
		"total":       len(maskedEnv),
		"environment": maskedEnv,
	})
}

// GetSystemInfo returns system information
func (h *DebugHandler) GetSystemInfo(c *fiber.Ctx) error {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	systemInfo := fiber.Map{
		"go_version":    runtime.Version(),
		"os":            runtime.GOOS,
		"arch":          runtime.GOARCH,
		"num_cpu":       runtime.NumCPU(),
		"num_goroutine": runtime.NumGoroutine(),
		"memory": fiber.Map{
			"alloc_mb":       bToMb(m.Alloc),
			"total_alloc_mb": bToMb(m.TotalAlloc),
			"sys_mb":         bToMb(m.Sys),
			"num_gc":         m.NumGC,
		},
	}

	return utils.SuccessResponse(c, "System information retrieved", systemInfo)
}

// GetFeatureToggles returns the current state of all feature toggles
func (h *DebugHandler) GetFeatureToggles(c *fiber.Ctx) error {
	toggles := fiber.Map{
		"ai_features": fiber.Map{
			"enabled":     h.config.EnableAIFeatures,
			"description": "Enable AI-powered code suggestions and log analysis",
		},
		"websocket": fiber.Map{
			"enabled":     h.config.EnableWebSocket,
			"description": "Enable real-time WebSocket connections",
		},
		"performance_monitoring": fiber.Map{
			"enabled":     h.config.EnablePerformanceMonitoring,
			"description": "Enable performance metrics collection and monitoring",
		},
		"rate_limiting": fiber.Map{
			"enabled":     h.config.EnableRateLimiting,
			"description": "Enable API rate limiting",
		},
		"circuit_breaker": fiber.Map{
			"enabled":     h.config.EnableCircuitBreaker,
			"description": "Enable circuit breaker for external service calls",
		},
		"detailed_errors": fiber.Map{
			"enabled":     h.config.EnableDetailedErrors,
			"description": "Enable detailed error messages with stack traces",
		},
		"debug_endpoints": fiber.Map{
			"enabled":     h.config.EnableDebugEndpoints,
			"description": "Enable debug endpoints for development",
		},
	}

	return utils.SuccessResponse(c, "Feature toggles retrieved", toggles)
}

// GetHealthChecks returns detailed health check information
func (h *DebugHandler) GetHealthChecks(c *fiber.Ctx) error {
	checks := fiber.Map{
		"server": fiber.Map{
			"status":  "healthy",
			"message": "Server is running",
		},
		"configuration": fiber.Map{
			"status":  "healthy",
			"message": "Configuration is valid",
		},
		"features": fiber.Map{
			"ai_features":            h.config.EnableAIFeatures,
			"websocket":              h.config.EnableWebSocket,
			"performance_monitoring": h.config.EnablePerformanceMonitoring,
		},
	}

	// Check if OpenAI API key is configured
	if h.config.EnableAIFeatures {
		if h.config.OpenAIAPIKey != "" {
			checks["openai"] = fiber.Map{
				"status":  "configured",
				"message": "OpenAI API key is set",
			}
		} else {
			checks["openai"] = fiber.Map{
				"status":  "warning",
				"message": "OpenAI API key is not configured",
			}
		}
	}

	return utils.SuccessResponse(c, "Health checks retrieved", checks)
}

// maskSensitiveValue masks sensitive values for display
func maskSensitiveValue(value string) string {
	if value == "" {
		return ""
	}

	if len(value) <= 8 {
		return "****"
	}

	// Show first 4 and last 4 characters
	return value[:4] + "****" + value[len(value)-4:]
}

// bToMb converts bytes to megabytes
func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}
