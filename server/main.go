package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

var (
	dbPool *pgxpool.Pool
	rdb    *redis.Client
	ctx    = context.Background()
)

type Problem struct {
	ID         string `json:"problem_id"`
	Title      string `json:"title"`
	Slug       string `json:"slug"`
	Difficulty string `json:"difficulty"`
}

// Struct to map the incoming JSON payload from the frontend
type SubmitRequest struct {
	UserID     string `json:"user_id"`
	ProblemID  string `json:"problem_id"`
	Language   string `json:"language"`
	SourceCode string `json:"source_code"`
}

func main() {
	// 1. Connect to PostgreSQL (Make sure your actual password is here)
	dbURL := "postgres://campus_app:app@localhost:5432/CampusCompile_db?sslmode=disable"
	var err error
	dbPool, err = pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("Unable to create database pool: %v\n", err)
	}
	defer dbPool.Close()
	fmt.Println("[*] Connected to PostgreSQL successfully!")

	// 2. Connect to Redis
	rdb = redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Unable to connect to Redis: %v\n", err)
	}
	fmt.Println("[*] Connected to Redis successfully!")

	// 3. Set up the Gin Router
	router := gin.Default()

	// FIX: Silence the proxy warning
	router.SetTrustedProxies(nil)

	// 4. Define endpoints
	router.GET("/api/problems", getProblems)
	router.POST("/api/submit", submitCode)
	router.GET("/api/submissions/:id", getSubmissionStatus) // <-- ADD THIS LINE

	// 5. Start the server
	fmt.Println("[*] API Server running on http://localhost:8080")
	router.Run(":8080")
}

func getProblems(c *gin.Context) {
	rows, err := dbPool.Query(ctx, "SELECT problem_id, title, slug, difficulty FROM problems")
	if err != nil {
		fmt.Printf("[!] Database query failed: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database query failed"})
		return
	}
	defer rows.Close()

	var problems []Problem
	for rows.Next() {
		var p Problem
		if err := rows.Scan(&p.ID, &p.Title, &p.Slug, &p.Difficulty); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse data"})
			return
		}
		problems = append(problems, p)
	}
	c.JSON(http.StatusOK, problems)
}

// Handler function for POST /api/submit
func submitCode(c *gin.Context) {
	var req SubmitRequest

	// 1. Validate the incoming JSON
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	// 2. Generate a new UUID for this submission
	submissionID := uuid.New().String()

	// 3. Save the submission to PostgreSQL with status 'Pending'
	_, err := dbPool.Exec(ctx,
		`INSERT INTO submissions (submission_id, user_id, problem_id, language, source_code, status) 
		 VALUES ($1, $2, $3, $4, $5, 'Pending')`,
		submissionID, req.UserID, req.ProblemID, req.Language, req.SourceCode)

	if err != nil {
		fmt.Printf("[!] DB Insert Error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save submission"})
		return
	}

	// 4. Push the submission ID into the Redis Queue
	redisMsg := fmt.Sprintf(`{"submission_id": "%s"}`, submissionID)
	err = rdb.LPush(ctx, "submission_queue", redisMsg).Err()
	if err != nil {
		fmt.Printf("[!] Redis Push Error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to queue submission"})
		return
	}

	// 5. Return success to the user instantly
	c.JSON(http.StatusAccepted, gin.H{
		"message":       "Submission received",
		"submission_id": submissionID,
		"status":        "Pending",
	})
}

// Handler function for GET /api/submissions/:id
func getSubmissionStatus(c *gin.Context) {
	// 1. Grab the ID from the URL parameter
	submissionID := c.Param("id")

	// 2. Query PostgreSQL for just the status and language
	var status, language string
	err := dbPool.QueryRow(ctx,
		"SELECT status, language FROM submissions WHERE submission_id = $1",
		submissionID,
	).Scan(&status, &language)

	// 3. Handle errors (like if the ID doesn't exist)
	if err != nil {
		if err.Error() == "no rows in result set" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Submission not found"})
			return
		}
		fmt.Printf("[!] Database query failed: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch status"})
		return
	}

	// 4. Return the status to the frontend
	c.JSON(http.StatusOK, gin.H{
		"submission_id": submissionID,
		"language":      language,
		"status":        status,
	})
}
