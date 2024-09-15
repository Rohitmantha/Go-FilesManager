package config

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/joho/godotenv"
)

var (
	RedisClient *redis.Client
	RedisCtx    = context.Background()
)

// ConnectRedis initializes the Redis connection
func ConnectRedis() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file")
	}

	// Get environment variables
	redisPassword := os.Getenv("REDIS_PWD")  
	redisHost := os.Getenv("REDIS_HOST")     
	redisPort := os.Getenv("REDIS_PORT")
	redisDB := os.Getenv("REDIS_DB")         
	
	// Parse the Redis DB index as an integer
	dbIndex, err := strconv.Atoi(redisDB)
	if err != nil {
		log.Fatalf("Invalid Redis DB index: %v", err)
	}

	// Build the Redis address
	dsn := fmt.Sprintf("%s:%s", redisHost, redisPort)

	// Initialize Redis client
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     dsn,         // Redis instance address
		Password: redisPassword, // Redis password, if any
		DB:       dbIndex,     // Redis DB index
	})

	// Ping the Redis server to test the connection
	_, err = RedisClient.Ping(RedisCtx).Result()
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
}

// SetRedisTTL sets the time-to-live for Redis cached data
func SetRedisTTL(key string, value string, ttl time.Duration) error {
	return RedisClient.Set(RedisCtx, key, value, ttl).Err()
}

// GetRedisValue retrieves data from Redis by key
func GetRedisValue(key string) (string, error) {
	return RedisClient.Get(RedisCtx, key).Result()
}
