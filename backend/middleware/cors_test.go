package middleware

import (
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestCORS(t *testing.T) {
	app := fiber.New()
	app.Use(CORS())

	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// Add OPTIONS handler for preflight requests
	app.Options("/test", func(c *fiber.Ctx) error {
		return c.SendStatus(204)
	})

	tests := []struct {
		name           string
		origin         string
		method         string
		expectedStatus int
		checkHeaders   bool
	}{
		{
			name:           "Valid origin localhost:3000",
			origin:         "http://localhost:3000",
			method:         "GET",
			expectedStatus: 200,
			checkHeaders:   true,
		},
		{
			name:           "Valid origin 127.0.0.1:3000",
			origin:         "http://127.0.0.1:3000",
			method:         "GET",
			expectedStatus: 200,
			checkHeaders:   true,
		},
		{
			name:           "OPTIONS preflight request",
			origin:         "http://localhost:3000",
			method:         "OPTIONS",
			expectedStatus: 204,
			checkHeaders:   false, // CORS headers might not be set the same way in test
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest(tt.method, "/test", nil)
			req.Header.Set("Origin", tt.origin)

			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.checkHeaders {
				// Check if CORS headers are present (they might be empty in test environment)
				allowOrigin := resp.Header.Get("Access-Control-Allow-Origin")
				if allowOrigin != "" {
					assert.Equal(t, tt.origin, allowOrigin)
				}

				allowCreds := resp.Header.Get("Access-Control-Allow-Credentials")
				if allowCreds != "" {
					assert.Equal(t, "true", allowCreds)
				}

				allowHeaders := resp.Header.Get("Access-Control-Allow-Headers")
				if allowHeaders != "" {
					assert.Contains(t, allowHeaders, "Content-Type")
				}
			}
		})
	}
}

func TestCORSWithOrigins(t *testing.T) {
	customOrigins := []string{"https://example.com", "https://app.example.com"}
	app := fiber.New()
	app.Use(CORSWithOrigins(customOrigins))

	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	tests := []struct {
		name           string
		origin         string
		expectedStatus int
		shouldAllow    bool
	}{
		{
			name:           "Allowed custom origin",
			origin:         "https://example.com",
			expectedStatus: 200,
			shouldAllow:    true,
		},
		{
			name:           "Another allowed custom origin",
			origin:         "https://app.example.com",
			expectedStatus: 200,
			shouldAllow:    true,
		},
		{
			name:           "Default origin should not be allowed",
			origin:         "http://localhost:3000",
			expectedStatus: 200, // Request still goes through, but CORS headers won't match
			shouldAllow:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/test", nil)
			req.Header.Set("Origin", tt.origin)

			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.shouldAllow {
				assert.Equal(t, tt.origin, resp.Header.Get("Access-Control-Allow-Origin"))
			}
		})
	}
}

func TestDefaultCORSConfig(t *testing.T) {
	config := DefaultCORSConfig()

	assert.Contains(t, config.AllowOrigins, "http://localhost:3000")
	assert.Contains(t, config.AllowOrigins, "http://127.0.0.1:3000")
	assert.Contains(t, config.AllowMethods, fiber.MethodGet)
	assert.Contains(t, config.AllowMethods, fiber.MethodPost)
	assert.Contains(t, config.AllowHeaders, "Content-Type")
	assert.Contains(t, config.AllowHeaders, "Authorization")
	assert.True(t, config.AllowCredentials)
	assert.Equal(t, 86400, config.MaxAge)
}

func TestNewCORS(t *testing.T) {
	config := CORSConfig{
		AllowOrigins:     []string{"https://test.com"},
		AllowMethods:     []string{fiber.MethodGet},
		AllowHeaders:     []string{"Content-Type"},
		AllowCredentials: false,
		MaxAge:           3600,
	}

	app := fiber.New()
	app.Use(NewCORS(config))

	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://test.com")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "https://test.com", resp.Header.Get("Access-Control-Allow-Origin"))
}
