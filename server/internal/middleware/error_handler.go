package middleware

import (
	"log"

	"github.com/gin-gonic/gin"

	"campuscompile/api/internal/errors"
)

// ErrorInterceptor acts as a global catcher for all AppErrors propagated by controllers.
// It logs the internal details securely and returns only the safe ClientMsg to the user.
func ErrorInterceptor() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err
			if appErr, ok := err.(*errors.AppError); ok {
				// Server-side logging (Strict isolation: only middleware logs)
				log.Printf("[ERROR] %v | Internal: %v", appErr.ClientMsg, appErr.Internal)
				
				// Safe client-facing response
				c.JSON(appErr.HTTPStatus, gin.H{"error": appErr.ClientMsg})
				return
			}

			// Fallback for non-AppErrors (e.g. raw standard library errors)
			log.Printf("[UNHANDLED ERROR] %v", err)
			c.JSON(500, gin.H{"error": "An unexpected system error occurred."})
		}
	}
}
