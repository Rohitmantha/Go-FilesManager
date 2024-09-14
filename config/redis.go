package config

import (
	"context"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
)

var (
	RedisClient *redis.Client
	RedisCtx    = context.Background()
)

// ConnectRedis initializes the Redis connection
func ConnectRedis() {
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379", // Redis instance address
		Password: "",               // Set password if required
		DB:       0,                // Use default DB
	})

	// Ping the Redis server to test connection
	_, err := RedisClient.Ping(RedisCtx).Result()
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
