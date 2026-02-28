package middleware

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// In production, load this from os.Getenv("SESSION_SECRET") instead of hardcoding
var JwtSecret = []byte("super_secret_campus_key_change_me")

// RequireAuth is the bouncer that protects secure routes
func RequireAuth(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing Authorization header"})
		c.Abort()
		return
	}

	var tokenString string
	fmt.Sscanf(authHeader, "Bearer %s", &tokenString)
	if tokenString == "" {
		tokenString = authHeader
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
		c.Set("user_id", claims["user_id"])
		c.Set("role", claims["role"])
		c.Next()
	} else {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token payload"})
		c.Abort()
	}
}
