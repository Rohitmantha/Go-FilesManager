package handlers

import (
	"auth-app/config"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

// Redis configuration
var (
	redisClient *redis.Client
	redisCtx    = context.Background()
	cacheTTL    = 5 * time.Minute // Time-to-live for cached metadata
)

// Get Files Metadata (GET /files)
// Get Files Metadata (GET /files)
func GetFiles(c *gin.Context) {
	// Get the user_id from the context
	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Check if userID is float64 (commonly from JWT), then convert to int64
	var userIDInt int64
	switch v := userID.(type) {
	case int64:
		userIDInt = v
	case float64:
		userIDInt = int64(v)
	default:
		log.Printf("Unexpected type for user_id: %T, value: %v\n", userID, userID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse user ID"})
		return
	}

	// Try fetching from Redis cache first
	cacheKey := "files_metadata:" + strconv.FormatInt(userIDInt, 10)
	metadataJSON, err := config.RedisClient.Get(redisCtx, cacheKey).Result()

	var metadata []map[string]interface{}
	if err == redis.Nil {
		// If not cached, fetch from DB and cache it
		metadataJSON, err = fetchFilesMetadataFromDB(userIDInt)
		if err != nil {
			log.Printf("Database error: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch file metadata"})
			return
		}

		// Cache the metadata with a TTL
		err = config.RedisClient.Set(redisCtx, cacheKey, metadataJSON, cacheTTL).Err()
		if err != nil {
			log.Printf("Redis cache error: %v\n", err)
		}
	} else if err != nil {
		// Handle other Redis errors
		log.Printf("Redis error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Redis error"})
		return
	}

	// Unmarshal the metadata JSON to a slice of maps
	if err := json.Unmarshal([]byte(metadataJSON), &metadata); err != nil {
		log.Printf("Error unmarshaling metadata JSON: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse metadata"})
		return
	}

	// Return the metadata in the desired format
	c.JSON(http.StatusOK, gin.H{"files": metadata})
}

// Generate Shareable Link (GET /share/:file_id)

func ShareFile(c *gin.Context) {
	fileID := c.Param("file_id")
	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Convert userID to int64
	var userIDInt int64
	switch v := userID.(type) {
	case int64:
		userIDInt = v
	case float64:
		userIDInt = int64(v)
	default:
		log.Printf("Unexpected type for user_id: %T, value: %v\n", userID, userID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse user ID"})
		return
	}

	cacheKey := "file_share_link:" + fileID

	// Check if the shareable link is already cached
	shareLink, err := config.RedisClient.Get(redisCtx, cacheKey).Result()
	if err == redis.Nil {
		// If not cached, fetch from DB to get the public URL
		s3URL, err := fetchFileURLFromDB(fileID, userIDInt)
		if err != nil {
			log.Printf("Could not fetch file URL: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not fetch file URL"})
			return
		}

		// Cache the public URL for 24 hours
		err = config.RedisClient.Set(redisCtx, cacheKey, s3URL, 24*time.Hour).Err()
		if err != nil {
			log.Printf("Redis cache error: %v\n", err)
		}

		// Return the public URL
		c.JSON(http.StatusOK, gin.H{"share_link": s3URL})
	} else if err != nil {
		// Handle Redis error
		log.Printf("Redis error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Redis error"})
		return
	} else {
		// Return the cached public URL
		c.JSON(http.StatusOK, gin.H{"share_link": shareLink})
	}
}

// Helper: Fetch file metadata from MySQL
func fetchFilesMetadataFromDB(userID int64) (string, error) {
	query := "SELECT id, file_name, file_size, upload_date, s3_url FROM files WHERE user_id = ?"
	rows, err := config.DB.Query(query, userID)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var metadata []map[string]interface{}
	for rows.Next() {
		var fileID int64
		var fileName, s3URL string
		var fileSize int64
		var uploadDateStr string // Use string for scanning

		if err := rows.Scan(&fileID, &fileName, &fileSize, &uploadDateStr, &s3URL); err != nil {
			return "", err
		}

		// Convert uploadDateStr to time.Time
		uploadDate, err := time.Parse("2006-01-02 15:04:05", uploadDateStr) // Adjust the layout as needed
		if err != nil {
			return "", err
		}

		metadata = append(metadata, map[string]interface{}{
			"file_id":     fileID,
			"file_name":   fileName,
			"file_size":   fileSize,
			"upload_date": uploadDate.Format(time.RFC3339), // Ensure correct format for JSON
			"s3_url":      s3URL,
		})
	}

	// Marshal the metadata to JSON
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return "", err
	}
	return string(metadataJSON), nil
}

// Helper: Fetch file URL from MySQL
func fetchFileURLFromDB(fileID string, userID int64) (string, error) {
	query := "SELECT s3_url FROM files WHERE id = ? AND user_id = ?"
	var s3URL string
	err := config.DB.QueryRow(query, fileID, userID).Scan(&s3URL)
	if err != nil {
		return "", err
	}
	return s3URL, nil
}

// Search Files (GET /files/search?name=&date=)
func SearchFiles(c *gin.Context) {
	// Get the user_id from the context
	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Convert userID to int64
	var userIDInt int64
	switch v := userID.(type) {
	case int64:
		userIDInt = v
	case float64:
		userIDInt = int64(v)
	default:
		log.Printf("Unexpected type for user_id: %T, value: %v\n", userID, userID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse user ID"})
		return
	}

	// Get search parameters
	name := c.Query("name")
	date := c.Query("date")

	// Build the query
	query := "SELECT id, file_name, file_size, upload_date, s3_url FROM files WHERE user_id = ?"
	var args []interface{}
	args = append(args, userIDInt)

	if name != "" {
		query += " AND file_name LIKE ?"
		args = append(args, "%"+name+"%")
	}
	if date != "" {
		query += " AND DATE(upload_date) = ?"
		args = append(args, date)
	}

	// Execute the query
	rows, err := config.DB.Query(query, args...)
	if err != nil {
		log.Printf("Database error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search files"})
		return
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var id, fileSize int64
		var fileName, s3URL string
		var uploadDateStr string // Use string for scanning

		if err := rows.Scan(&id, &fileName, &fileSize, &uploadDateStr, &s3URL); err != nil {
			log.Printf("Row scan error: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse file data"})
			return
		}

		// Convert uploadDateStr to time.Time
		uploadDate, err := time.Parse("2006-01-02 15:04:05", uploadDateStr) // Adjust the layout as needed
		if err != nil {
			log.Printf("Date parsing error: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse upload date"})
			return
		}

		results = append(results, map[string]interface{}{
			"id":          id,
			"file_name":   fileName,
			"file_size":   fileSize,
			"upload_date": uploadDate.Format(time.RFC3339),
			"s3_url":      s3URL,
		})
	}

	c.JSON(http.StatusOK, gin.H{"files": results})
}

// Helper: Generate a short link (mock implementation)
func generateShortLink(s3URL string) string {
	return "https://short.url/" + s3URL[len(s3URL)-8:] // Mock implementation
}
