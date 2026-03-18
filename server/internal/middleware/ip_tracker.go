package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"campuscompile/api/internal/database"
	redisPkg "campuscompile/api/internal/redis"
	"campuscompile/api/internal/repositories"
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
		clientIP := c.ClientIP()

		if contestID == "" {
			c.Next()
			return
		}

		ipKey := fmt.Sprintf("contest_ips:%s:%s", contestID, userID)
		alertKey := fmt.Sprintf("ip_alerted:%s:%s", contestID, userID)
		now := time.Now().Unix()

		// Expanded to 48 hours to safely cover multi-day hackathons without needing a DB lookup
		ttlSeconds := 172800

		result, err := ipTrackerScript.Run(
			c.Request.Context(),
			redisPkg.Client,
			[]string{ipKey, alertKey},
			now, clientIP, ttlSeconds,
		).Int()

		if err != nil {
			log.Printf("[ERROR] Redis IP tracking script failed for User %s in Contest %s: %v", userID, contestID, err)
			c.Next() // Still fail-open so the student can submit code
			return
		}

		if result >= 3 {
			go func(uid, cid, ip string, totalIPs int) {
				// Enforce a strict 10-second timeout for the entire background operation
				bgCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				metadata := map[string]interface{}{
					"total_ips": totalIPs,
					"latest_ip": ip,
				}
				metaBytes, _ := json.Marshal(metadata)

				repo := repositories.NewContestRepository(database.Pool)

				// Pass the bounded context to Postgres
				err := repo.LogTelemetry(bgCtx, cid, uid, "anomalous_routing", metaBytes)
				if err != nil {
					log.Printf("[ERROR] Failed to log IP anomaly to Postgres for %s: %v\n", uid, err)
					return
				}

				// Pass the bounded context to Redis
				err = redisPkg.Client.SAdd(bgCtx, "dirty_contests", cid).Err()
				if err != nil {
					log.Printf("[CRITICAL] Failed to flag contest %s as dirty after IP anomaly: %v\n", cid, err)
				} else {
					log.Printf("[SECURITY] Logged anomalous_routing for User %s. IPs: %d\n", uid, totalIPs)
				}
			}(userID, contestID, clientIP, result)
		}

		c.Next()
	}
}
