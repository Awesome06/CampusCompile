package controllers

import (
	"campuscompile/api/internal/errors"
	"campuscompile/api/internal/models"
	"campuscompile/api/internal/services"
	"campuscompile/api/internal/utils"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
	// 1. SAFE PARSING FIRST
	// Protected against memory/CPU exhaustion by the 128KB PayloadArmor middleware.
	var req models.SubmitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// No internal logging needed here as the middleware catches it if we wrap it
		c.Error(errors.NewAppError(http.StatusBadRequest, err, "Invalid request payload. Please verify your submission format."))
		return
	}

	// 2. IDENTITY & CONTEXT DETERMINATION
	userID := c.MustGet("user_id").(string)
	cooldownDuration := 3 * time.Second

	if req.ContestID != nil && *req.ContestID != "" {
		if err := uuid.Validate(*req.ContestID); err != nil {
			c.Error(errors.NewAppError(http.StatusBadRequest, err, "Invalid contest identifier format."))
			return
		}

		// Payload is valid and context is an Arena. Escalate the penalty duration.
		cooldownDuration = 10 * time.Second
	}

	// 3. SINGLE, ACCURATE RATE LIMIT ENFORCEMENT
	allowed, remaining, err := utils.EnforceCooldown(c.Request.Context(), ctrl.rdb, userID, "submit", cooldownDuration)
	if err != nil {
		c.Error(errors.NewAppError(http.StatusInternalServerError, err, "Failed to verify submission rate limit"))
		return
	}

	if !allowed {
		retrySeconds := int64(math.Max(1, math.Ceil(remaining.Seconds())))

		// Set HTTP 429 Header (Requires CORS ExposeHeaders config)
		c.Header("Retry-After", strconv.FormatInt(retrySeconds, 10))

		errorMsg := "You are submitting too fast."
		if cooldownDuration == 10*time.Second {
			errorMsg = "Arena submission cooldown active."
		}

		c.JSON(http.StatusTooManyRequests, gin.H{
			"error":    errorMsg,
			"retry_in": retrySeconds,
		})
		return
	}

	// 4. Hand off to the Service layer
	submissionID, err := ctrl.service.ProcessSubmission(c.Request.Context(), req, userID)
	if err != nil {
		c.Error(errors.NewAppError(http.StatusInternalServerError, err, "Failed to process submission"))
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
		c.Error(errors.NewAppError(http.StatusBadRequest, err, "Invalid request payload"))
		return
	}

	runID, err := ctrl.service.ProcessRun(c.Request.Context(), req)
	if err != nil {
		c.Error(errors.NewAppError(http.StatusInternalServerError, err, "Failed to queue run"))
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"run_id": runID, "status": "Pending"})
}

func (ctrl *SubmissionController) GetRunStatus(c *gin.Context) {
	runID := c.Param("id")

	result, err := ctrl.service.FetchRunStatus(c.Request.Context(), runID)
	if err != nil {
		c.Error(errors.NewAppError(http.StatusInternalServerError, err, "Execution engine disconnected or malformed result"))
		return
	}

	c.JSON(http.StatusOK, result)
}

func (ctrl *SubmissionController) GetSubmissionStatus(c *gin.Context) {
	submissionID := c.Param("id")

	result, err := ctrl.service.FetchSubmissionStatus(c.Request.Context(), submissionID)
	if err != nil {
		c.Error(errors.NewAppError(http.StatusNotFound, err, "Submission not found"))
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

	limit, offset := parsePaginationArgs(c, 10)

	history, err := ctrl.service.FetchSubmissionHistory(c.Request.Context(), userID, problemID, contestID, limit, offset)
	if err != nil {
		c.Error(errors.NewAppError(http.StatusInternalServerError, err, "Failed to fetch history"))
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
		case msg, ok := <-ch:
			if !ok {
				return
			}
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
		case msg, ok := <-ch:
			if !ok {
				return
			}
			c.SSEvent("message", msg)
			c.Writer.Flush()
		}
	}
}
