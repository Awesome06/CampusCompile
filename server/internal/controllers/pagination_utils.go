package controllers

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

func parsePaginationArgs(c *gin.Context) (int, int) {
	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 10
	}

	if limit < 1 {
		limit = 10
	} else if limit > 100 {
		limit = 100
	}

	offsetStr := c.DefaultQuery("offset", "0")
	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	return limit, offset
}
