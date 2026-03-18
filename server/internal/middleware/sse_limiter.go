package middleware

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	redisPkg "campuscompile/api/internal/redis"
)

// The Increment Script (Sets a 12-hour rolling failsafe)
var incrementScript = redis.NewScript(`
	local current = tonumber(redis.call('GET', KEYS[1]) or '0')
	local max_conns = tonumber(ARGV[1])

	if current < max_conns then
		redis.call('INCR', KEYS[1])
		redis.call('EXPIRE', KEYS[1], 43200) 
		return 1
	else
		return 0
	end
`)

// The Decrement Script
var decrementScript = redis.NewScript(`
	local current = tonumber(redis.call('DECR', KEYS[1]) or '0')
	if current <= 0 then
		redis.call('DEL', KEYS[1])
	else
		redis.call('EXPIRE', KEYS[1], 43200)
	end
	return 1
`)

// RequireSSECap enforces a limit on concurrent streams per user, scoped by stream type.
func RequireSSECap(streamType string, maxConnections int) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.MustGet("user_id").(string)

		// Namespace the Redis key by stream type
		key := fmt.Sprintf("sse_count:%s:%s", streamType, userID)

		reqCtx := c.Request.Context()

		allowed, err := incrementScript.Run(reqCtx, redisPkg.Client, []string{key}, maxConnections).Int()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify connection limit"})
			c.Abort()
			return
		}

		// Unified 429 Response Contract
		if allowed == 0 {
			c.Header("Retry-After", "5") // Tell the browser/client to back off for 5 seconds
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":    fmt.Sprintf("Too many active %s streams. Please close other tabs.", streamType),
				"retry_in": 5, // Match the exact JSON contract used by the submission controller
			})
			c.Abort()
			return
		}

		// Background Cleanup and Heartbeat
		go func(ctx context.Context, connectionKey string) {
			// Ping Redis every 6 hours to keep the 12-hour TTL alive for active streams
			ticker := time.NewTicker(6 * time.Hour)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					// The client disconnected; safely release the slot with a bounded timeout
					cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()

					err := decrementScript.Run(cleanupCtx, redisPkg.Client, []string{connectionKey}).Err()
					if err != nil {
						log.Printf("[ERROR] Failed to decrement SSE connection count for %s: %v", connectionKey, err)
					}
					return // Exit the goroutine and free the memory

				case <-ticker.C:
					// Refresh the TTL to prevent mid-stream expiration
					redisPkg.Client.Expire(context.Background(), connectionKey, 12*time.Hour)
				}
			}
		}(reqCtx, key)

		c.Next()
	}
}
