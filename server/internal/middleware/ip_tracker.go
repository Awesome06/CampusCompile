package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	redisPkg "campuscompile/api/internal/redis"
	// Import your repository/telemetry package here
)

// The IP Tracking Lua Script
// KEYS[1] = contest_ips:{contest_id}:{user_id} (The ZSET holding unique IPs)
// KEYS[2] = ip_alerted:{contest_id}:{user_id} (The circuit breaker flag)
// ARGV[1] = Current Unix Timestamp
// ARGV[2] = The Client IP Address
// ARGV[3] = TTL in seconds (e.g., 43200 for 12 hours to cover any contest length)
var ipTrackerScript = redis.NewScript(`
	-- 1. Add the IP to the Sorted Set. If it already exists, just updates the timestamp score.
	redis.call('ZADD', KEYS[1], ARGV[1], ARGV[2])
	redis.call('EXPIRE', KEYS[1], ARGV[3])
	
	-- 2. Check the total number of unique IPs recorded
	local unique_ips = redis.call('ZCARD', KEYS[1])
	
	-- 3. If the threshold is breached, check the circuit breaker
	if unique_ips >= 3 then
		local already_alerted = redis.call('GET', KEYS[2])
		if not already_alerted then
			-- Flip the circuit breaker so we only alert the DB once per contest
			redis.call('SET', KEYS[2], '1', 'EX', ARGV[3])
			return unique_ips -- Return the count to trigger the Go DB alert
		end
	end
	
	return 0 -- No alert needed
`)

// TrackContestIP monitors for rapid geographical/network shifting during a live contest.
func TrackContestIP() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.MustGet("user_id").(string)
		contestID := c.Param("id")

		// Gin automatically resolves X-Forwarded-For headers if trusted proxies are set up
		clientIP := c.ClientIP()

		if contestID == "" {
			c.Next()
			return
		}

		ipKey := fmt.Sprintf("contest_ips:%s:%s", contestID, userID)
		alertKey := fmt.Sprintf("ip_alerted:%s:%s", contestID, userID)
		now := time.Now().Unix()
		ttlSeconds := 43200 // 12 hours of memory

		// Execute the Lua script
		result, err := ipTrackerScript.Run(
			c.Request.Context(),
			redisPkg.Client,
			[]string{ipKey, alertKey},
			now, clientIP, ttlSeconds,
		).Int()

		if err != nil {
			// Log the Redis error internally, but do not block the student's request
			c.Next()
			return
		}

		// If result > 0, the threshold was breached for the very first time
		if result >= 3 {
			// Fire a background goroutine to log the anomaly so we don't block the HTTP response
			go func(uid, cid, ip string, totalIPs int) {
				// Use a fresh background context, as the request context will die soon
				ctx := context.Background()

				// TODO: Replace with your actual database repository call
				// e.g., db.RecordTelemetryEvent(ctx, cid, uid, "anomalous_routing", ...)
				fmt.Printf("[SECURITY ALERT] User %s used %d distinct IPs in Contest %s. Latest IP: %s\n",
					uid, totalIPs, cid, ip)
			}(userID, contestID, clientIP, result)
		}

		c.Next()
	}
}
