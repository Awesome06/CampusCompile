package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"campuscompile/api/internal/database"
	"campuscompile/api/internal/models"
	redisPkg "campuscompile/api/internal/redis"
)

func SubmitCode(c *gin.Context) {
	var req models.SubmitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	userID := c.MustGet("user_id").(string)
	submissionID := uuid.New().String()

	// 1. Log submission as Pending in PostgreSQL
	_, err := database.Pool.Exec(c.Request.Context(),
		`INSERT INTO submissions (submission_id, user_id, problem_id, language, source_code, status) 
		 VALUES ($1, $2, $3, $4, $5, 'Pending')`,
		submissionID, userID, req.ProblemID, req.Language, req.SourceCode)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to log submission"})
		return
	}

	// 2. Queue for the Judging Worker using the standardized struct
	payload := models.OfficialSubmissionPayload{
		SubmissionID: submissionID,
	}

	jsonPayload, _ := json.Marshal(payload)
	err = redisPkg.Client.LPush(c.Request.Context(), "submission_queue", jsonPayload).Err()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to queue execution"})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message":       "Submission received. Judging in progress...",
		"submission_id": submissionID,
		"status":        "Pending",
	})
}

func RunCode(c *gin.Context) {
	var req models.RunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	runID := uuid.New().String()

	// Use CustomRunPayload to ensure "is_custom": true is sent to the worker
	payload := models.CustomRunPayload{
		IsCustom:    true,
		RunID:       runID,
		Language:    req.Language,
		SourceCode:  req.SourceCode,
		CustomInput: req.CustomInput,
	}

	jsonPayload, _ := json.Marshal(payload)
	err := redisPkg.Client.LPush(c.Request.Context(), "submission_queue", jsonPayload).Err()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to queue run"})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"run_id": runID, "status": "Pending"})
}

func GetRunStatus(c *gin.Context) {
	runID := c.Param("id")
	// Worker saves result to "run_result:{run_id}"
	val, err := redisPkg.Client.Get(c.Request.Context(), "run_result:"+runID).Result()

	if err == redis.Nil {
		c.JSON(http.StatusOK, gin.H{"status": "Pending"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Execution engine disconnected"})
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Malformed result in cache"})
		return
	}
	c.JSON(http.StatusOK, result)
}

// GetSubmissionStatus and GetSubmissionHistory remain unchanged as they correctly interface with DB
func GetSubmissionStatus(c *gin.Context) {
	submissionID := c.Param("id")
	var status, language string
	var errorLogs, sourceCode *string // Using consistent camelCase for Go variables

	// Query the database for the submission details including the source code
	err := database.Pool.QueryRow(c.Request.Context(), `
		SELECT status, language, error_logs, source_code 
		FROM submissions WHERE submission_id = $1
	`, submissionID).Scan(&status, &language, &errorLogs, &sourceCode)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Submission not found"})
		return
	}

	// Handle pointer-to-string conversions safely
	message := ""
	if errorLogs != nil {
		message = *errorLogs
	}

	code := ""
	if sourceCode != nil {
		code = *sourceCode
	}

	// Return the standardized JSON payload for the Arena frontend
	c.JSON(http.StatusOK, gin.H{
		"submission_id": submissionID,
		"language":      language,
		"status":        status,
		"message":       message,
		"source_code":   code,
	})
}

func GetSubmissionHistory(c *gin.Context) {
	problemID := c.Param("id")
	userID := c.MustGet("user_id").(string)

	rows, err := database.Pool.Query(c.Request.Context(),
		`SELECT submission_id, language, status, submitted_at 
		 FROM submissions 
		 WHERE user_id = $1 AND problem_id = $2 
		 ORDER BY submitted_at DESC`,
		userID, problemID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch history"})
		return
	}
	defer rows.Close()

	var history []models.SubmissionHistoryEntry
	for rows.Next() {
		var entry models.SubmissionHistoryEntry
		if err := rows.Scan(&entry.ID, &entry.Language, &entry.Status, &entry.SubmittedAt); err != nil {
			continue
		}
		history = append(history, entry)
	}

	if history == nil {
		history = []models.SubmissionHistoryEntry{}
	}

	c.JSON(http.StatusOK, history)
}
