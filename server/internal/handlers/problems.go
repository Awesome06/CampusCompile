package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"campuscompile/api/internal/database"
	"campuscompile/api/internal/models"
)

func GetProblems(c *gin.Context) {
	ctx := context.Background()
	rows, err := database.Pool.Query(ctx, "SELECT problem_id, title, slug, difficulty FROM problems ORDER BY created_at ASC")
	if err != nil {
		fmt.Printf("[!] Database query failed: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database query failed"})
		return
	}
	defer rows.Close()

	var problems []models.Problem
	for rows.Next() {
		var p models.Problem
		if err := rows.Scan(&p.ID, &p.Title, &p.Slug, &p.Difficulty); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse data"})
			return
		}
		problems = append(problems, p)
	}

	// Ensure an empty array is sent instead of null if there are no problems
	if problems == nil {
		problems = []models.Problem{}
	}
	c.JSON(http.StatusOK, problems)
}

func GetProblemByID(c *gin.Context) {
	id := c.Param("id")
	var p models.Problem
	ctx := context.Background()

	err := database.Pool.QueryRow(ctx,
		"SELECT problem_id, title, description, difficulty FROM problems WHERE problem_id = $1",
		id).Scan(&p.ID, &p.Title, &p.Description, &p.Difficulty)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Problem not found"})
		return
	}

	var sampleInput, sampleOutput string
	err = database.Pool.QueryRow(ctx,
		"SELECT input_data, expected_output FROM test_cases WHERE problem_id = $1 LIMIT 1",
		id).Scan(&sampleInput, &sampleOutput)

	if err == nil {
		p.SampleInput = &sampleInput
		p.SampleOutput = &sampleOutput
	}

	c.JSON(http.StatusOK, p)
}

func CreateProblem(c *gin.Context) {
	userRole, exists := c.Get("role")
	if !exists || (userRole != "professor" && userRole != "admin") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admins and professors can add problems."})
		return
	}

	var req models.CreateProblemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	slug := strings.ToLower(strings.ReplaceAll(req.Title, " ", "-"))
	ctx := context.Background()

	tx, err := database.Pool.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start database transaction"})
		return
	}
	defer tx.Rollback(ctx)

	var newProblemID string
	err = tx.QueryRow(ctx,
		`INSERT INTO problems (title, slug, description, difficulty) 
		 VALUES ($1, $2, $3, $4) RETURNING problem_id`,
		req.Title, slug, req.Description, req.Difficulty).Scan(&newProblemID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create problem"})
		return
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO test_cases (problem_id, input_data, expected_output, is_hidden) 
		 VALUES ($1, $2, $3, false)`,
		newProblemID, req.SampleInput, req.SampleOutput)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to attach sample test cases"})
		return
	}

	tx.Commit(ctx)
	c.JSON(http.StatusCreated, gin.H{"message": "Problem created successfully!", "problem_id": newProblemID})
}
