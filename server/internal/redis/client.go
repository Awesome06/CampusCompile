package redis

import (
	"context"
	"fmt"
	"log"

	redisClient "github.com/redis/go-redis/v9"
)

// Client is the globally accessible Redis client
var Client *redisClient.Client

// InitRedis initializes the Redis connection
func InitRedis(host string) {
	Client = redisClient.NewClient(&redisClient.Options{
		Addr: host + ":6379",
	})

	ctx := context.Background()
	if err := Client.Ping(ctx).Err(); err != nil {
		log.Fatalf("Unable to connect to Redis: %v\n", err)
	}

	fmt.Println("[*] Connected to Redis successfully!")

	// 👇 NEW: Boot up the multiplexer
	InitHub(Client)
}
