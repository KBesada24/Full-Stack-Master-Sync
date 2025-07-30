package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	AllowCredentials bool
	ExposeHeaders    []string
	MaxAge           int
}

// DefaultCORSConfig returns default CORS configuration
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowOrigins: []string{"http://localhost:3000", "http://127.0.0.1:3000"},
		AllowMethods: []string{
			fiber.MethodGet,
			fiber.MethodPost,
			fiber.MethodPut,
			fiber.MethodPatch,
			fiber.MethodDelete,
			fiber.MethodOptions,
			fiber.MethodHead,
		},
		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Accept",
			"Authorization",
			"X-Requested-With",
			"X-Trace-ID",
			"X-Request-ID",
		},
		AllowCredentials: true,
		ExposeHeaders: []string{
			"X-Trace-ID",
			"X-Request-ID",
		},
		MaxAge: 86400, // 24 hours
	}
}

// NewCORS creates a new CORS middleware with custom configuration
func NewCORS(config CORSConfig) fiber.Handler {
	return cors.New(cors.Config{
		AllowOrigins:     strings.Join(config.AllowOrigins, ","),
		AllowMethods:     strings.Join(config.AllowMethods, ","),
		AllowHeaders:     strings.Join(config.AllowHeaders, ","),
		AllowCredentials: config.AllowCredentials,
		ExposeHeaders:    strings.Join(config.ExposeHeaders, ","),
		MaxAge:           config.MaxAge,
	})
}

// CORS creates a CORS middleware with default configuration
func CORS() fiber.Handler {
	return NewCORS(DefaultCORSConfig())
}

// CORSWithOrigins creates a CORS middleware with custom allowed origins
func CORSWithOrigins(origins []string) fiber.Handler {
	config := DefaultCORSConfig()
	config.AllowOrigins = origins
	return NewCORS(config)
}
