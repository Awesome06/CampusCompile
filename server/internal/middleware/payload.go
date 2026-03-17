package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// MaxPayloadSize sets the absolute ceiling for incoming JSON requests.
// 128KB is extremely generous for competitive programming source code.
const MaxPayloadSize = 128 * 1024

// PayloadArmor wraps the request body in an http.MaxBytesReader.
// If the payload exceeds the limit, it immediately cuts off the connection
// and prevents memory exhaustion.
func PayloadArmor() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Wrap the native request body
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxPayloadSize)

		c.Next()
	}
}
