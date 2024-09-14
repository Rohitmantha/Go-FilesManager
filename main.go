package main

import (
	"auth-app/config"
	"auth-app/handlers"
	"auth-app/middleware"

	"github.com/gin-gonic/gin"
)

func main() {
	// Connect to the database
	config.ConnectDB()

	// Connect to Redis for caching
	config.ConnectRedis()

	r := gin.Default()

	// Public routes
	r.POST("/register", handlers.Register)
	r.POST("/login", handlers.Login)

	// Protected routes
	protected := r.Group("/protected")
	protected.Use(middleware.AuthRequired)
	{
		protected.POST("/upload", handlers.UploadFile)       // Phase 2: File Upload
		protected.GET("/files", handlers.GetFiles)           // Phase 3: Retrieve all files
		protected.GET("/share/:file_id", handlers.ShareFile) // Phase 3: Share file with public URL
		protected.GET("/files/search", handlers.SearchFiles) // Phase 4: Search files
		protected.GET("/profile", func(c *gin.Context) {
			userID := c.MustGet("user_id")
			c.JSON(200, gin.H{"user_id": userID, "message": "Welcome to your profile"})
		})
	}

	// Start the server
	r.Run(":8080")
}
