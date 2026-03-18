package middleware

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	redisPkg "campuscompile/api/internal/redis"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

var JwtSecret = []byte("super_secret_campus_key_change_me")

const defaultRedisAuthTimeout = 1 * time.Second

var redisAuthTimeout = loadRedisAuthTimeout()

func loadRedisAuthTimeout() time.Duration {
	envVal, ok := os.LookupEnv("REDIS_AUTH_TIMEOUT")
	if !ok || strings.TrimSpace(envVal) == "" {
		return defaultRedisAuthTimeout
	}
	d, err := time.ParseDuration(envVal)
	if err != nil || d <= 0 {
		fmt.Printf("invalid REDIS_AUTH_TIMEOUT %q, using default %s\n", envVal, defaultRedisAuthTimeout)
		return defaultRedisAuthTimeout
	}
	return d
}

func RequireAuth(c *gin.Context) {
	var tokenString string
	authHeader := c.GetHeader("Authorization")

	if authHeader == "" {
		// STRICT RESTRICTION: Only allow URL tokens for SSE streaming endpoints
		if strings.Contains(c.Request.URL.Path, "/stream") {
			tokenString = c.Query("token")
		}
	} else {
		tokenString = strings.TrimPrefix(authHeader, "Bearer ")
	}

	if tokenString == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing Authorization token"})
		c.Abort()
		return
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return JwtSecret, nil
	})

	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
		c.Abort()
		return
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		// 1. SAFELY CAST CRITICAL CLAIMS TO STRINGS
		if userIDRaw, exists := claims["user_id"]; exists {
			c.Set("user_id", fmt.Sprintf("%v", userIDRaw))
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token payload: missing user_id"})
			c.Abort()
			return
		}

		// 1. Extract the Session ID from the JWT
		sessionIDRaw, exists := claims["session_id"]
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token payload: missing session_id"})
			c.Abort()
			return
		}
		sessionID := fmt.Sprintf("%v", sessionIDRaw)
		userID := fmt.Sprintf("%v", claims["user_id"])

		// 2. Query Redis for the single source of truth
		redisKey := fmt.Sprintf("active_session:%s", userID)

		// 🚨 CIRCUIT BREAKER: Enforce a strict 200ms timeout
		// If Redis doesn't answer instantly, assume it's dead and fail closed to protect the Go scheduler
		redisCtx, cancel := context.WithTimeout(c.Request.Context(), redisAuthTimeout)
		defer cancel()

		activeSession, err := redisPkg.Client.Get(redisCtx, redisKey).Result()

		// 3. The Guillotine Logic
		if err == redis.Nil {
			// Expected failure: Key doesn't exist (User logged out or TTL expired)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Session expired. Please log in again."})
			c.Abort()
			return
		} else if err != nil {
			// Unexpected failure: Redis is unreachable or timed out (Fail closed gracefully)
			c.Header("Retry-After", "5")
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Authentication layer temporarily unavailable. Please retry."})
			c.Abort()
			return
		} else if activeSession != sessionID {
			// Cryptographically valid, but legally dead
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Session superseded by a login on another device."})
			c.Abort()
			return
		}

		if roleRaw, exists := claims["role"]; exists {
			c.Set("role", fmt.Sprintf("%v", roleRaw))
		} else {
			// Fallback for legacy tokens minted before role claims were strictly enforced
			c.Set("role", "student")
		}

		// Extract demographic context if the user is onboarded
		if isOnboarded, _ := claims["is_onboarded"].(bool); isOnboarded {
			if course, ok := claims["course"].(string); ok {
				c.Set("course", course)
			}
			if dept, ok := claims["department"].(string); ok {
				c.Set("department", dept)
			}
			if batch, ok := claims["batch"].(string); ok {
				c.Set("batch", batch)
			}
			if section, ok := claims["section"].(string); ok {
				c.Set("section", section)
			}
			if group, ok := claims["student_group"].(string); ok {
				c.Set("student_group", group)
			}
			// JWT unmarshals numbers as float64; safely cast them back
			if gradYear, ok := claims["graduation_year"].(float64); ok {
				c.Set("graduation_year", int(gradYear))
			}
		}

		c.Next()
	} else {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token payload"})
		c.Abort()
	}
}

// RequireRole restricts access to specific user roles (e.g., "admin", "professor")
func RequireRole(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract the role from the Gin context (set by RequireAuth)
		userRole, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized: No role found in session."})
			c.Abort()
			return
		}

		roleStr, ok := userRole.(string)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized: Invalid role format."})
			c.Abort()
			return
		}

		// Check if the user's role matches any of the allowed roles
		isAllowed := false
		for _, role := range allowedRoles {
			if roleStr == role {
				isAllowed = true
				break
			}
		}

		if !isAllowed {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access Denied: You do not have clearance for this action."})
			c.Abort()
			return
		}

		c.Next() // Role is valid, proceed to the handler
	}
}
