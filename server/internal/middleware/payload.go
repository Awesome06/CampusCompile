package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// PayloadArmor enforces a strict byte limit on incoming request bodies.
// It fast-fails with a 413 if the Content-Length header is honest,
// and enforces an http.MaxBytesReader to catch malicious streaming payloads.
func PayloadArmor(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Early Fast-Fail: Check declared Content-Length
		if c.Request.ContentLength > maxBytes {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": "Payload too large. Maximum allowed size exceeded.",
			})
			return
		}

		// 2. The Enforcer: Wrap the body to catch lying clients
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)

		c.Next()
	}
}
