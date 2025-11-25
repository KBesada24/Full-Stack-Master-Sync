package config

import (
	"os"
	"strconv"
	"strings"
)

// Config holds all configuration for the application
type Config struct {
	// Server Configuration
	Port        string
	Host        string
	Environment string

	// OpenAI Configuration
	OpenAIAPIKey string

	// CORS Configuration
	FrontendURL string

	// WebSocket Configuration
	WSEndpoint string

	// Logging Configuration
	LogLevel  string
	LogFormat string

	// Testing Configuration
	CypressBaseURL    string
	PlaywrightBaseURL string

	// Feature Toggles
	EnableAIFeatures            bool
	EnableWebSocket             bool
	EnablePerformanceMonitoring bool
	EnableRateLimiting          bool
	EnableCircuitBreaker        bool
	EnableDetailedErrors        bool
	EnableDebugEndpoints        bool
}

// Load loads configuration from environment variables with defaults
func Load() *Config {
	return &Config{
		// Server Configuration
		Port:        getEnv("PORT", "8080"),
		Host:        getEnv("HOST", "localhost"),
		Environment: getEnv("ENVIRONMENT", "development"),

		// OpenAI Configuration
		OpenAIAPIKey: getEnv("OPENAI_API_KEY", ""),

		// CORS Configuration
		FrontendURL: getEnv("FRONTEND_URL", "http://localhost:3000"),

		// WebSocket Configuration
		WSEndpoint: getEnv("WS_ENDPOINT", "/ws"),

		// Logging Configuration
		LogLevel:  strings.ToLower(getEnv("LOG_LEVEL", "info")),
		LogFormat: strings.ToLower(getEnv("LOG_FORMAT", "json")),

		// Testing Configuration
		CypressBaseURL:    getEnv("CYPRESS_BASE_URL", "http://localhost:3000"),
		PlaywrightBaseURL: getEnv("PLAYWRIGHT_BASE_URL", "http://localhost:3000"),

		// Feature Toggles (default to enabled)
		EnableAIFeatures:            getEnvAsBool("ENABLE_AI_FEATURES", true),
		EnableWebSocket:             getEnvAsBool("ENABLE_WEBSOCKET", true),
		EnablePerformanceMonitoring: getEnvAsBool("ENABLE_PERFORMANCE_MONITORING", true),
		EnableRateLimiting:          getEnvAsBool("ENABLE_RATE_LIMITING", true),
		EnableCircuitBreaker:        getEnvAsBool("ENABLE_CIRCUIT_BREAKER", true),
		EnableDetailedErrors:        getEnvAsBool("ENABLE_DETAILED_ERRORS", false),
		EnableDebugEndpoints:        getEnvAsBool("ENABLE_DEBUG_ENDPOINTS", false),
	}
}

// getEnv gets an environment variable with a fallback default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt gets an environment variable as integer with a fallback default value
func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvAsBool gets an environment variable as boolean with a fallback default value
func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// IsDevelopment returns true if the environment is development
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

// IsProduction returns true if the environment is production
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

// GetServerAddress returns the full server address
func (c *Config) GetServerAddress() string {
	return c.Host + ":" + c.Port
}

// Validate validates the configuration and returns any errors
func (c *Config) Validate() []string {
	var errors []string

	// Validate required fields
	if c.Port == "" {
		errors = append(errors, "PORT is required")
	}

	if c.Host == "" {
		errors = append(errors, "HOST is required")
	}

	// Validate log level
	validLogLevels := []string{"debug", "info", "warn", "error"}
	if !contains(validLogLevels, c.LogLevel) {
		errors = append(errors, "LOG_LEVEL must be one of: debug, info, warn, error")
	}

	// Validate log format
	validLogFormats := []string{"json", "text"}
	if !contains(validLogFormats, c.LogFormat) {
		errors = append(errors, "LOG_FORMAT must be one of: json, text")
	}

	// Validate environment
	validEnvironments := []string{"development", "staging", "production"}
	if !contains(validEnvironments, c.Environment) {
		errors = append(errors, "ENVIRONMENT must be one of: development, staging, production")
	}

	return errors
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
