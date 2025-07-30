package main

import (
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName: "Full Stack Master Sync Backend",
	})

	// Basic health check endpoint
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"message": "Full Stack Master Sync Backend is running",
			"version": "1.0.0",
		})
	})

	// Start server
	port := "8080"
	fmt.Printf("🚀 Server starting on port %s\n", port)
	log.Fatal(app.Listen(":" + port))
}
