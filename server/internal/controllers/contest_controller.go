package controllers

import (
	"encoding/json"
	"fmt"

	"github.com/gin-gonic/gin"

	"campuscompile/api/internal/services"
)

type ContestController struct {
	service services.ContestService
}

func NewContestController(service services.ContestService) *ContestController {
	return &ContestController{service: service}
}

// StreamLeaderboard is the SSE endpoint that keeps the React UI perfectly in sync
func (ctrl *ContestController) StreamLeaderboard(c *gin.Context) {
	contestID := c.Param("id")

	// 1. Subscribe to the contest's specific Redis broadcast channel
	channelName := fmt.Sprintf("contest:leaderboard_updates:%s", contestID)
	ch, cleanup := ctrl.service.SubscribeToChannel(c.Request.Context(), channelName)
	defer cleanup()

	// 2. Set the necessary headers to keep the HTTP connection alive for SSE
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Flush()

	// 3. Immediately fetch and send the current leaderboard state so the UI isn't blank
	initialLeaderboard, err := ctrl.service.FetchCurrentLeaderboard(c.Request.Context(), contestID)
	if err == nil {
		initialData, _ := json.Marshal(initialLeaderboard)
		c.SSEvent("message", string(initialData))
		c.Writer.Flush()
	}

	// 4. Enter the infinite loop: listen for live updates from the Python Worker / Go Engine
	for {
		select {
		case <-c.Request.Context().Done():
			// The student closed the tab or navigated away; kill the connection cleanly
			return
		case msg := <-ch:
			// A student just got an 'AC' and the leaderboard shifted! Push the new data.
			c.SSEvent("message", msg.Payload)
			c.Writer.Flush()
		}
	}
}
