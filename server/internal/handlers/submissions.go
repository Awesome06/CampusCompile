package handlers

import (
	"context"
	"encoding/json"
	"fmt"
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
	ctx := context.Background()

	_, err := database.Pool.Exec(ctx,
		`INSERT INTO submissions (submission_id, user_id, problem_id, language, source_code, status) 
		 VALUES ($1, $2, $3, $4, $5, 'Pending')`,
		submissionID, userID, req.ProblemID, req.Language, req.SourceCode)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save submission"})
		return
	}

	redisMsg := fmt.Sprintf(`{"submission_id": "%s"}`, submissionID)
	err = redisPkg.Client.LPush(ctx, "submission_queue", redisMsg).Err()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to queue submission"})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"message": "Submission received", "submission_id": submissionID, "status": "Pending"})
}

func GetSubmissionStatus(c *gin.Context) {
	submissionID := c.Param("id")
	var status, language string
	var errorLogs *string
	ctx := context.Background()

	err := database.Pool.QueryRow(ctx, "SELECT status, language, error_logs FROM submissions WHERE submission_id = $1", submissionID).Scan(&status, &language, &errorLogs)
	if err != nil {
		if err.Error() == "no rows in result set" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Submission not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch status"})
		return
	}

	message := ""
	if errorLogs != nil {
		message = *errorLogs
	}

	c.JSON(http.StatusOK, gin.H{
		"submission_id": submissionID,
		"language":      language,
		"status":        status,
		"message":       message,
	})
}

func RunCode(c *gin.Context) {
	var req models.RunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	runID := uuid.New().String()
	payload := map[string]interface{}{
		"is_custom":    true,
		"run_id":       runID,
		"language":     req.Language,
		"source_code":  req.SourceCode,
		"custom_input": req.CustomInput,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create execution payload"})
		return
	}

	ctx := context.Background()
	err = redisPkg.Client.LPush(ctx, "submission_queue", jsonPayload).Err()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to queue run"})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"run_id": runID, "status": "Pending"})
}

func GetRunStatus(c *gin.Context) {
	runID := c.Param("id")
	ctx := context.Background()

	val, err := redisPkg.Client.Get(ctx, "run_result:"+runID).Result()

	if err == redis.Nil {
		c.JSON(http.StatusOK, gin.H{"status": "Pending"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Redis error"})
		return
	}

	var result map[string]interface{}
	json.Unmarshal([]byte(val), &result)
	c.JSON(http.StatusOK, result)
}

func GetSubmissionHistory(c *gin.Context) {
	problemID := c.Param("id")
	userID := c.MustGet("user_id").(string)
	ctx := context.Background()

	rows, err := database.Pool.Query(ctx,
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
