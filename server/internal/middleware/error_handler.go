package middleware

import (
	stdErrors "errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	appErrors "campuscompile/api/internal/errors"
)

// ErrorInterceptor acts as a global catcher for all AppErrors propagated by controllers.
// It logs the internal details securely and returns only the safe ClientMsg to the user.
func ErrorInterceptor() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err
			var appErr *appErrors.AppError
			
			if stdErrors.As(err, &appErr) {
				// Server-side logging (Strict isolation: only middleware logs)
				slog.Error("Intercepted structured application error",
					"component", "ErrorInterceptor",
					"method", c.Request.Method,
					"path", c.Request.URL.Path,
					"internal_error", appErr.Internal,
					"client_msg", appErr.ClientMsg,
				)
				
				// Safe client-facing response
				c.JSON(appErr.HTTPStatus, gin.H{"error": appErr.ClientMsg})
				return
			}

			// Fallback for non-AppErrors (e.g. raw standard library errors)
			slog.Error("Intercepted unhandled raw panic or error",
				"component", "ErrorInterceptor",
				"method", c.Request.Method,
				"path", c.Request.URL.Path,
				"error", err,
			)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "An unexpected system error occurred."})
		}
	}
}
