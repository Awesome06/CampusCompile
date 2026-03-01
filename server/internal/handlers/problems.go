package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"campuscompile/api/internal/database"
	"campuscompile/api/internal/models"
)

// SampleTestCase defines the structure for the React frontend
type SampleTestCase struct {
	Input  string `json:"input"`
	Output string `json:"output"`
}

func GetProblems(c *gin.Context) {
	ctx := context.Background()
	// Added time_limit_ms and memory_limit_kb to support Arena preview cards
	rows, err := database.Pool.Query(ctx, "SELECT problem_id, title, slug, difficulty, time_limit_ms, memory_limit_kb FROM problems ORDER BY created_at ASC")
	if err != nil {
		fmt.Printf("[!] Database query failed: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database query failed"})
		return
	}
	defer rows.Close()

	var problems []gin.H
	for rows.Next() {
		var id, title, slug, difficulty string
		var timeLimit, memLimit int
		if err := rows.Scan(&id, &title, &slug, &difficulty, &timeLimit, &memLimit); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse data"})
			return
		}
		problems = append(problems, gin.H{
			"problem_id":      id,
			"title":           title,
			"slug":            slug,
			"difficulty":      difficulty,
			"time_limit_ms":   timeLimit,
			"memory_limit_kb": memLimit,
		})
	}

	if problems == nil {
		problems = []gin.H{}
	}
	c.JSON(http.StatusOK, problems)
}

func GetProblemByID(c *gin.Context) {
	problemID := c.Param("id")
	ctx := context.Background()

	// 1. Fetch the Problem Metadata
	var title, description, difficulty string
	var timeLimit, memoryLimit int

	err := database.Pool.QueryRow(ctx, `
		SELECT title, description, difficulty, time_limit_ms, memory_limit_kb 
		FROM problems WHERE problem_id = $1
	`, problemID).Scan(&title, &description, &difficulty, &timeLimit, &memoryLimit)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Problem not found in the Arena."})
		return
	}

	// 2. Fetch the File Paths for Public Samples ONLY
	rows, err := database.Pool.Query(ctx, `
		SELECT input_s3_key, expected_s3_key 
		FROM test_cases 
		WHERE problem_id = $1 AND is_hidden = false
	`, problemID)

	var samples []SampleTestCase
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var inPath, outPath string
			if err := rows.Scan(&inPath, &outPath); err == nil {
				// Read the physical files from the disk
				inData, _ := os.ReadFile(inPath)
				outData, _ := os.ReadFile(outPath)

				samples = append(samples, SampleTestCase{
					Input:  string(inData),
					Output: string(outData),
				})
			}
		}
	}

	// 3. Send the complete package to React
	c.JSON(http.StatusOK, gin.H{
		"problem_id":      problemID,
		"title":           title,
		"description":     description,
		"difficulty":      difficulty,
		"time_limit_ms":   timeLimit,
		"memory_limit_kb": memoryLimit,
		"samples":         samples,
	})
}

func CreateProblem(c *gin.Context) {
	var req models.CreateProblemRequest

	// 1. Parse the JSON from the React frontend
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	// 2. Generate a unique ID and a URL-friendly slug
	problemID := uuid.New().String()
	slug := strings.ToLower(strings.ReplaceAll(req.Title, " ", "-"))

	// 3. Insert the new problem into PostgreSQL
	// Adjust the column names if your database schema differs slightly
	_, err := database.Pool.Exec(c.Request.Context(), `
		INSERT INTO problems (problem_id, title, slug, description, difficulty, time_limit_ms, memory_limit_kb)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, problemID, req.Title, slug, req.Description, req.Difficulty, req.TimeLimit, req.MemoryLimit)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to forge problem in database"})
		return
	}

	// 4. Return success and the new ID back to the frontend
	c.JSON(http.StatusOK, gin.H{
		"message":    "Problem created successfully",
		"problem_id": problemID,
	})
}
