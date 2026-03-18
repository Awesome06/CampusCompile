package middleware

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	redisPkg "campuscompile/api/internal/redis"
)

// 1. The Increment Script (The Bouncer)
// Atomically checks if current < max. If true, increments and sets a 5-min rolling TTL.
var incrementScript = redis.NewScript(`
	local current = tonumber(redis.call('GET', KEYS[1]) or '0')
	local max_conns = tonumber(ARGV[1])

	if current < max_conns then
		redis.call('INCR', KEYS[1])
		redis.call('EXPIRE', KEYS[1], 43200) -- 12-hour rolling failsafe for server crashes
		return 1
	else
		return 0
	end
`)

// 2. The Decrement Script (The Janitor)
// Safely decrements the connection count. If it hits 0, deletes the key to save memory.
var decrementScript = redis.NewScript(`
	local current = tonumber(redis.call('DECR', KEYS[1]) or '0')
	if current <= 0 then
		redis.call('DEL', KEYS[1])
	else
		redis.call('EXPIRE', KEYS[1], 43200) -- Maintain the failsafe for remaining connections
	end
	return 1
`)

// RequireSSECap enforces a strict limit on concurrent active connections per user.
func RequireSSECap(maxConnections int) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.MustGet("user_id").(string)
		key := fmt.Sprintf("sse_count:%s", userID)

		// Capture the exact context and key outside the goroutine
		reqCtx := c.Request.Context()

		allowed, err := incrementScript.Run(reqCtx, redisPkg.Client, []string{key}, maxConnections).Int()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify connection limit"})
			c.Abort()
			return
		}

		if allowed == 0 {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Too many active live streams. Please close other Arena tabs."})
			c.Abort()
			return
		}

		// Safely pass the captured context and key into the goroutine
		go func(ctx context.Context, connectionKey string) {
			<-ctx.Done() // Wait for this specific request's lifecycle to end

			// Always use context.Background() for the cleanup call since the request ctx is now dead
			err := decrementScript.Run(context.Background(), redisPkg.Client, []string{connectionKey}).Err()
			if err != nil {
				log.Printf("[ERROR] Failed to decrement SSE connection count for %s: %v", connectionKey, err)
			}
		}(reqCtx, key)

		c.Next()
	}
}
