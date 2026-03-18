package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"campuscompile/api/internal/database"
	redisPkg "campuscompile/api/internal/redis"
	"campuscompile/api/internal/repositories"
)

var ipHashSalt string

// InitSecurityConfig enforces fail-fast at boot if the environment is insecure.
// Call this from main.go alongside your database initialization.
func InitSecurityConfig() {
	ipHashSalt = os.Getenv("IP_HASH_SALT")
	if ipHashSalt == "" {
		log.Fatal("FATAL STARTUP ERROR: IP_HASH_SALT environment variable is missing. Refusing to start with insecure telemetry tracking.")
	}
}

// Helper 1: One-way cryptographic hash for Redis
func hashIP(ip string) string {
	hash := sha256.Sum256([]byte(ip + ipHashSalt))
	return hex.EncodeToString(hash[:])
}

// Helper 2: Robust network masking using Go's native IP parser
func maskIP(ipStr string) string {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return "***.***.***.***" // Fallback for completely malformed headers
	}

	// Safely extract and mask IPv4
	if ip4 := ip.To4(); ip4 != nil {
		return fmt.Sprintf("%d.%d.%d.***", ip4[0], ip4[1], ip4[2])
	}

	// Safely extract and mask IPv6
	// Standard privacy masking keeps the /48 routing prefix (the first 6 bytes)
	return fmt.Sprintf("%02x%02x:%02x%02x:%02x%02x::***", ip[0], ip[1], ip[2], ip[3], ip[4], ip[5])
}

// The IP Tracking Lua Script
// KEYS[1] = contest_ips:{contest_id}:{user_id}
// KEYS[2] = ip_alerted:{contest_id}:{user_id}
// ARGV[1] = Current Unix Timestamp
// ARGV[2] = The Client IP Address
// ARGV[3] = TTL in seconds (172800)
var ipTrackerScript = redis.NewScript(`
	redis.call('ZADD', KEYS[1], ARGV[1], ARGV[2])
	redis.call('EXPIRE', KEYS[1], ARGV[3])
	
	local unique_ips = redis.call('ZCARD', KEYS[1])
	
	if unique_ips >= 3 then
		local already_alerted = redis.call('GET', KEYS[2])
		if not already_alerted then
			redis.call('SET', KEYS[2], '1', 'EX', ARGV[3])
			return unique_ips 
		end
	end
	
	return 0
`)

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

		ttlSeconds := 172800
		hashedIP := hashIP(clientIP)

		result, err := ipTrackerScript.Run(
			c.Request.Context(),
			redisPkg.Client,
			[]string{ipKey, alertKey},
			now, hashedIP, ttlSeconds,
		).Int()

		if err != nil {
			log.Printf("[ERROR] Redis IP tracking script failed for User %s in Contest %s: %v", userID, contestID, err)
			c.Next()
			return
		}

		if result >= 3 {
			go func(uid, cid, ip string, totalIPs int) {
				bgCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				maskedIP := maskIP(ip)

				metadata := map[string]interface{}{
					"total_ips": totalIPs,
					"latest_ip": maskedIP,
				}
				metaBytes, err := json.Marshal(metadata)
				if err != nil {
					log.Printf("[ERROR] Failed to marshal IP anomaly metadata for User %s in Contest %s: %v\n", uid, cid, err)
					fallback := map[string]interface{}{
						"error": "failed to marshal IP anomaly metadata",
					}
					if fbBytes, fbErr := json.Marshal(fallback); fbErr == nil {
						metaBytes = fbBytes
					} else {
						log.Printf("[ERROR] Failed to marshal fallback IP anomaly metadata for User %s in Contest %s: %v\n", uid, cid, fbErr)
						metaBytes = nil
					}
				}

				repo := repositories.NewContestRepository(database.Pool)

				err = repo.LogTelemetry(bgCtx, cid, uid, "anomalous_routing", metaBytes)
				if err != nil {
					log.Printf("[ERROR] Failed to log IP anomaly to Postgres for %s: %v\n", uid, err)
					return
				}

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
