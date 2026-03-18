package controllers

import (
	"campuscompile/api/internal/models"
	"campuscompile/api/internal/services"
	"campuscompile/api/internal/utils"
	"log"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type SubmissionController struct {
	service services.SubmissionService
	rdb     *redis.Client
}

func NewSubmissionController(service services.SubmissionService, rdb *redis.Client) *SubmissionController {
	return &SubmissionController{service: service, rdb: rdb}
}

func (ctrl *SubmissionController) SubmitCode(c *gin.Context) {
	var req models.SubmitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[ERROR] Payload binding error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload. Please verify your submission format."})
		return
	}

	userID := c.MustGet("user_id").(string)

	// 1. Global Base Rate Limit (Protects against pure spam across all endpoints)
	allowedGlobal, remGlobal, err := utils.EnforceCooldown(c.Request.Context(), ctrl.rdb, userID, "submit:global", 3*time.Second)
	if err != nil {
		log.Printf("[ERROR] Redis global rate limiter failure: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify submission rate limit"})
		return
	}

	if !allowedGlobal {
		retrySeconds := int64(math.Max(1, math.Ceil(remGlobal.Seconds())))
		c.Header("Retry-After", strconv.FormatInt(retrySeconds, 10))
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error":    "You are submitting too fast.",
			"retry_in": retrySeconds,
		})
		return
	}

	// 2. Contest Specific Rate Limit (Stricter penalty for live arenas)
	if req.ContestID != nil && *req.ContestID != "" {
		contestAction := "submit:contest:" + *req.ContestID
		allowedContest, remContest, err := utils.EnforceCooldown(c.Request.Context(), ctrl.rdb, userID, contestAction, 10*time.Second)
		if err != nil {
			log.Printf("[ERROR] Redis contest rate limiter failure: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify contest rate limit"})
			return
		}

		if !allowedContest {
			retrySeconds := int64(math.Max(1, math.Ceil(remContest.Seconds())))
			c.Header("Retry-After", strconv.FormatInt(retrySeconds, 10))
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":    "Arena submission cooldown active.",
				"retry_in": retrySeconds,
			})
			return
		}
	}

	// Hand off to the Service layer
	submissionID, err := ctrl.service.ProcessSubmission(c.Request.Context(), req, userID)
	if err != nil {
		log.Printf("[ERROR] SUBMISSION CRASH: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process submission"})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message":       "Submission received. Judging in progress...",
		"submission_id": submissionID,
		"status":        "Pending",
	})
}

func (ctrl *SubmissionController) RunCode(c *gin.Context) {
	var req models.RunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[ERROR] Payload binding error in RunCode: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	runID, err := ctrl.service.ProcessRun(c.Request.Context(), req)
	if err != nil {
		log.Printf("[ERROR] ProcessRun failure: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to queue run"})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"run_id": runID, "status": "Pending"})
}

func (ctrl *SubmissionController) GetRunStatus(c *gin.Context) {
	runID := c.Param("id")

	result, err := ctrl.service.FetchRunStatus(c.Request.Context(), runID)
	if err != nil {
		log.Printf("[ERROR] FetchRunStatus failure for ID %s: %v", runID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Execution engine disconnected or malformed result"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (ctrl *SubmissionController) GetSubmissionStatus(c *gin.Context) {
	submissionID := c.Param("id")

	result, err := ctrl.service.FetchSubmissionStatus(c.Request.Context(), submissionID)
	if err != nil {
		log.Printf("[ERROR] FetchSubmissionStatus failure for ID %s: %v", submissionID, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Submission not found"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (ctrl *SubmissionController) GetSubmissionHistory(c *gin.Context) {
	problemID := c.Param("id")
	userID := c.MustGet("user_id").(string)

	contestIDQuery := c.Query("contest_id")
	var contestID *string
	if contestIDQuery != "" {
		contestID = &contestIDQuery
	}

	limitStr := c.DefaultQuery("limit", "10")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 100 {
		limit = 10
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	history, err := ctrl.service.FetchSubmissionHistory(c.Request.Context(), userID, problemID, contestID, limit, offset)
	if err != nil {
		log.Printf("[ERROR] FetchSubmissionHistory failure for User %s, Problem %s: %v", userID, problemID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch history"})
		return
	}

	c.JSON(http.StatusOK, history)
}

func (ctrl *SubmissionController) StreamSubmissionStatus(c *gin.Context) {
	submissionID := c.Param("id")

	ch, cleanup := ctrl.service.SubscribeToChannel(c.Request.Context(), "submission_updates:"+submissionID)
	defer cleanup()

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Flush()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case msg := <-ch:
			c.SSEvent("message", msg)
			c.Writer.Flush()
		}
	}
}

func (ctrl *SubmissionController) StreamRunStatus(c *gin.Context) {
	runID := c.Param("id")

	ch, cleanup := ctrl.service.SubscribeToChannel(c.Request.Context(), "run_updates:"+runID)
	defer cleanup()

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Flush()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case msg := <-ch:
			c.SSEvent("message", msg)
			c.Writer.Flush()
		}
	}
}
