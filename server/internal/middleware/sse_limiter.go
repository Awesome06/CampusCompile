package middleware

import (
	"context"
	"fmt"
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
		redis.call('EXPIRE', KEYS[1], 300) 
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
		redis.call('EXPIRE', KEYS[1], 300)
	end
	return 1
`)

// RequireSSECap enforces a strict limit on concurrent active connections per user.
func RequireSSECap(maxConnections int) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.MustGet("user_id").(string)
		key := fmt.Sprintf("sse_count:%s", userID)

		// Execute the Lua check-and-increment script
		allowed, err := incrementScript.Run(c.Request.Context(), redisPkg.Client, []string{key}, maxConnections).Int()
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

		// Background Cleanup Goroutine
		// This waits silently until the client closes the browser tab or loses internet
		go func() {
			<-c.Request.Context().Done()

			// We MUST use context.Background() here because c.Request.Context()
			// is officially dead/canceled at this point.
			err := decrementScript.Run(context.Background(), redisPkg.Client, []string{key}).Err()
			if err != nil {
				// In a production environment, you might want to log this failure,
				// though the 5-minute rolling TTL acts as an automatic failsafe.
			}
		}()

		c.Next()
	}
}
