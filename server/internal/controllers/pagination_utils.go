package controllers

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

func parsePaginationArgs(c *gin.Context, defaultLimit int) (int, int) {
	limitStr := c.DefaultQuery("limit", strconv.Itoa(defaultLimit))
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = defaultLimit
	}

	if limit < 1 {
		limit = defaultLimit
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
