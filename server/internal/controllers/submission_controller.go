package controllers

import (
	"campuscompile/api/internal/models"
	"campuscompile/api/internal/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

type SubmissionController struct {
	service services.SubmissionService
}

func NewSubmissionController(service services.SubmissionService) *SubmissionController {
	return &SubmissionController{service: service}
}

func (ctrl *SubmissionController) SubmitCode(c *gin.Context) {
	var req models.SubmitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	userID := c.MustGet("user_id").(string)

	// Hand off to the Service layer
	submissionID, err := ctrl.service.ProcessSubmission(c.Request.Context(), req, userID)
	if err != nil {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	runID, err := ctrl.service.ProcessRun(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to queue run"})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"run_id": runID, "status": "Pending"})
}

func (ctrl *SubmissionController) GetRunStatus(c *gin.Context) {
	runID := c.Param("id")

	result, err := ctrl.service.FetchRunStatus(c.Request.Context(), runID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Execution engine disconnected or malformed result"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (ctrl *SubmissionController) GetSubmissionStatus(c *gin.Context) {
	submissionID := c.Param("id")

	result, err := ctrl.service.FetchSubmissionStatus(c.Request.Context(), submissionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Submission not found"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (ctrl *SubmissionController) GetSubmissionHistory(c *gin.Context) {
	problemID := c.Param("id")
	userID := c.MustGet("user_id").(string)

	history, err := ctrl.service.FetchSubmissionHistory(c.Request.Context(), userID, problemID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch history"})
		return
	}

	c.JSON(http.StatusOK, history)
}
