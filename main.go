package main

import (
	"cafe/database"
	"cafe/route"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func main() {
	database.InitDatabase()

	// Set Gin mode
	mode := os.Getenv("GIN_MODE")
	if mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		log.Println("Running in debug mode")
	}

	// Initialize router
	router := gin.Default()

	// Configure CORS
	allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
	origins := []string{"http://localhost:3000"}
	if allowedOrigins != "" {
		origins = append(origins, allowedOrigins)
	}
	corsConfig := cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
	router.Use(cors.New(corsConfig))
	log.Println("CORS configured")

	// Setup routes
	route.CafeRoutes(router)
	log.Println("Routes configured successfully")

	// Serve static files
	uploadDir := "./uploads"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		log.Fatalf("Failed to create uploads directory: %v", err)
	}
	router.Static("/uploads", uploadDir)

	frontendPath := "./frontend/build"
	if _, err := os.Stat(frontendPath); os.IsNotExist(err) {
		log.Println("Warning: Frontend build directory not found, static file serving may fail")
	}
	router.StaticFS("/static", http.Dir(filepath.Join(frontendPath, "static")))
	router.NoRoute(func(c *gin.Context) {
		c.File(filepath.Join(frontendPath, "index.html"))
	})
	log.Println("Static file serving configured")

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8083"
	}
	log.Printf("Starting server on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
