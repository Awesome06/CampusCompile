package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"campuscompile/api/internal/database"
	"campuscompile/api/internal/models"
)

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

	// 2. Fetch the Public Samples directly from the DB text columns
	rows, err := database.Pool.Query(ctx, `
		SELECT input_data, expected_output 
		FROM test_cases 
		WHERE problem_id = $1 AND is_hidden = false
	`, problemID)

	var samples []SampleTestCase
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var inData, outData string
			if err := rows.Scan(&inData, &outData); err == nil {
				samples = append(samples, SampleTestCase{
					Input:  inData,
					Output: outData,
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

	// 1. Parse the JSON from the React frontend (Title, Description, etc.)
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	// 🛡️ 2. ZERO-TRUST: Retrieve the verified UUID from the middleware context
	authorID, exists := c.Get("user_id") // Matches the key set in jwt.go
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User identity not found in request context"})
		return
	}

	// 3. Generate a unique ID and a URL-friendly slug
	problemID := uuid.New().String()
	slug := strings.ToLower(strings.ReplaceAll(req.Title, " ", "-"))

	// 4. Insert the new problem into PostgreSQL, using the secure authorID
	_, err := database.Pool.Exec(c.Request.Context(), `
		INSERT INTO problems (problem_id, title, slug, description, difficulty, time_limit_ms, memory_limit_kb, author_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, problemID, req.Title, slug, req.Description, req.Difficulty, req.TimeLimit, req.MemoryLimit, authorID)

	if err != nil {
		fmt.Printf("[!] Failed to forge problem: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to forge problem in database"})
		return
	}

	// 5. Return success and the new ID back to the frontend
	c.JSON(http.StatusOK, gin.H{
		"message":    "Problem created successfully",
		"problem_id": problemID,
	})
}

func AddTestCasesBatch(c *gin.Context) {
	problemID := c.Param("id")

	var req models.BatchTestCasesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid test cases payload format"})
		return
	}

	if len(req.TestCases) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one test case is required"})
		return
	}

	ctx := context.Background()

	// Begin a database transaction
	tx, err := database.Pool.Begin(ctx)
	if err != nil {
		fmt.Printf("[!] Failed to start transaction: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	// Automatically rollback if the transaction isn't committed
	defer tx.Rollback(ctx)

	// Iterate and insert each test case
	for _, tc := range req.TestCases {
		tcID := uuid.New().String()

		_, err := tx.Exec(ctx, `
			INSERT INTO test_cases (test_case_id, problem_id, input_data, expected_output, is_hidden)
			VALUES ($1, $2, $3, $4, $5)
		`, tcID, problemID, tc.Input, tc.ExpectedOutput, tc.IsHidden)

		if err != nil {
			fmt.Printf("[!] Failed to insert test case %s: %v\n", tcID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save test cases to database"})
			return
		}
	}

	// Commit the transaction if all inserts succeed
	if err := tx.Commit(ctx); err != nil {
		fmt.Printf("[!] Failed to commit test case transaction: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to finalize test cases"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Test cases published successfully",
	})
}
