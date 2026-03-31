package controllers

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

// getSafeString extracts a string value natively from the Gin context
// effectively blocking internal nil sweeps from forcing 500 router panics.
func getSafeString(c *gin.Context, key string) (string, error) {
	val, exists := c.Get(key)
	if !exists {
		return "", fmt.Errorf("missing key %s", key)
	}
	str, ok := val.(string)
	if !ok || str == "" {
		return "", fmt.Errorf("missing or invalid key %s", key)
	}
	return str, nil
}
