package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// PayloadArmor creates a middleware that restricts the body size of specific requests.
// It fast-fails with a 413 if the Content-Length exceeds the maxBytes limit.
func PayloadArmor(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Content-Type Guard: Only armor JSON requests.
		// This allows multipart/form-data (like test case ZIP uploads) to pass through untouched.
		contentType := c.GetHeader("Content-Type")
		if !strings.HasPrefix(contentType, "application/json") {
			c.Next()
			return
		}

		// 2. Early Fast-Fail: Check declared Content-Length
		// If the client is honest and says the payload is too big, reject it before reading a single byte.
		if c.Request.ContentLength > maxBytes {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": "Payload too large. Maximum allowed JSON size exceeded.",
			})
			return
		}

		// 3. The Enforcer: Wrap the body for chunked/streaming requests
		// This protects against malicious clients that lie about their Content-Length.
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)

		c.Next()
	}
}
