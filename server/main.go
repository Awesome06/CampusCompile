package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

var (
	dbPool *pgxpool.Pool
	rdb    *redis.Client
	ctx    = context.Background()
	// In production, NEVER hardcode this. It should be an environment variable.
	jwtSecret = []byte("super_secret_campus_key_change_me")
)

// --- STRUCTS ---
type CreateProblemRequest struct {
	Title        string  `json:"title"`
	Description  string  `json:"description"`
	Difficulty   string  `json:"difficulty"`
	TimeLimit    float64 `json:"time_limit"`
	MemoryLimit  int     `json:"memory_limit"`
	SampleInput  string  `json:"sample_input"`
	SampleOutput string  `json:"sample_output"`
}
type Problem struct {
	ID           string  `json:"problem_id"`
	Title        string  `json:"title"`
	Slug         string  `json:"slug"`
	Description  string  `json:"description"`
	Difficulty   string  `json:"difficulty"`
	SampleInput  *string `json:"sample_input,omitempty"`  // 👇 NEW
	SampleOutput *string `json:"sample_output,omitempty"` // 👇 NEW
}

type SubmitRequest struct {
	ProblemID  string `json:"problem_id"`
	Language   string `json:"language"`
	SourceCode string `json:"source_code"`
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=1"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type RunRequest struct {
	Language    string `json:"language"`
	SourceCode  string `json:"source_code"`
	CustomInput string `json:"custom_input"`
}

type SubmissionHistoryEntry struct {
	ID          string    `json:"submission_id"`
	Language    string    `json:"language"`
	Status      string    `json:"status"`
	SubmittedAt time.Time `json:"submitted_at"`
}

// --- MAIN ---
func main() {
	// --- DOCKER NETWORK CONFIGURATION ---
	// Grab the hosts from Docker, or default to localhost if running manually
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}

	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" {
		redisHost = "localhost"
	}

	// 1. Connect to PostgreSQL
	dbURL := fmt.Sprintf("postgres://campus_app:app@%s:5432/CampusCompile_db?sslmode=disable", dbHost)

	fmt.Println("[*] Attempting to connect to PostgreSQL...")
	var err error
	for i := 1; i <= 5; i++ {
		dbPool, err = pgxpool.New(ctx, dbURL)

		// If the pool was created, try to actually Ping the database
		if err == nil {
			err = dbPool.Ping(ctx)
		}

		if err == nil {
			fmt.Println("[*] Connected to PostgreSQL successfully!")
			break
		}

		fmt.Printf("[!] Database not ready (Attempt %d/5). Waiting 2 seconds...\n", i)
		time.Sleep(2 * time.Second)

		if i == 5 {
			log.Fatalf("Fatal: Could not connect to database after 5 attempts: %v\n", err)
		}
	}

	// 2. Connect to Redis
	rdb = redis.NewClient(&redis.Options{Addr: redisHost + ":6379"})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Unable to connect to Redis: %v\n", err)
	}
	fmt.Println("[*] Connected to Redis successfully!")

	// 3. Set up the Gin Router
	router := gin.Default()
	router.SetTrustedProxies(nil)

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"}, // Your React app's URL
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// 4. Define endpoints
	// --- PUBLIC ROUTES (No token needed) ---
	// --- PUBLIC ROUTES (Anyone can browse the list) ---
	router.POST("/api/auth/register", registerUser)
	router.POST("/api/auth/login", loginUser)
	router.GET("/api/problems", getProblems)

	// --- PROTECTED ROUTES (Bouncer checks token first) ---
	protected := router.Group("/api")
	protected.Use(requireAuth) // Attach the middleware
	{
		protected.POST("/submit", submitCode)
		protected.POST("/run", runCode)
		protected.GET("/run/:id", getRunStatus)
		protected.GET("/problems/:id", getProblemByID)
		protected.GET("/submissions/:id", getSubmissionStatus)
		protected.GET("/submissions/history/:id", getSubmissionHistory)
		protected.POST("/problems", createProblem)
	}

	// 5. Start the server
	fmt.Println("[*] API Server running on http://localhost:8080")
	router.Run(":8080")
}

// --- MIDDLEWARE (The Bouncer) ---
func requireAuth(c *gin.Context) {
	// 1. Grab the token from the "Authorization" header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing Authorization header"})
		c.Abort() // Stop the request right here
		return
	}

	// Expecting header format: "Bearer <token>"
	var tokenString string
	fmt.Sscanf(authHeader, "Bearer %s", &tokenString)
	if tokenString == "" {
		tokenString = authHeader // fallback just in case
	}

	// 2. Verify the token's cryptographic signature using our secret key
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
		c.Abort()
		return
	}

	// 3. Extract the user_id from the token and attach it to the request context
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		c.Set("user_id", claims["user_id"])
		c.Set("role", claims["role"])
		c.Next() // Allow the request to proceed to /submit!
	} else {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token payload"})
		c.Abort()
	}
}

// --- AUTH HANDLERS ---

func registerUser(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data"})
		return
	}

	// 1. Hash the password using bcrypt
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// 2. Generate a UUID for the new user
	userID := uuid.New().String()

	// 3. Insert into the database. (Role defaults to 'student', rating to 1200)
	_, err = dbPool.Exec(ctx,
		`INSERT INTO users (user_id, username, email, password_hash, role, campus_rating) 
		 VALUES ($1, $2, $3, $4, 'student', 1200)`,
		userID, req.Username, req.Email, string(hashedPassword))

	if err != nil {
		fmt.Printf("[!] DB Insert Error: %v\n", err)
		c.JSON(http.StatusConflict, gin.H{"error": "Username or Email already exists"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "User registered successfully!", "user_id": userID})
}

func loginUser(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data"})
		return
	}

	fmt.Printf("[DEBUG] Attempting login for: %s\n", req.Email)

	// 1. Fetch user data - we cast role::text to handle the ENUM
	var userID, storedHash, role string
	err := dbPool.QueryRow(ctx,
		"SELECT user_id, password_hash, role::text FROM users WHERE email = $1",
		req.Email).Scan(&userID, &storedHash, &role)

	if err != nil {
		fmt.Printf("[ERROR] Database query failed for %s: %v\n", req.Email, err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password. Are you sure you are a Knight?"})
		return
	}

	fmt.Printf("[DEBUG] User found. Role: %s. Comparing passwords...\n", role)

	// 2. Compare the provided password with the stored hash
	err = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(req.Password))
	if err != nil {
		fmt.Printf("[ERROR] Password mismatch for %s: %v\n", req.Email, err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password. Are you sure you are a Knight?"})
		return
	}

	// 3. Generate JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"role":    role,
		"exp":     time.Now().Add(time.Hour * 72).Unix(),
	})

	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		fmt.Printf("[ERROR] JWT generation failed: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	fmt.Printf("[SUCCESS] Login successful for %s\n", req.Email)

	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"token":   tokenString,
		"role":    role,
	})
}

// --- EXISTING HANDLERS ---
func getProblems(c *gin.Context) {
	rows, err := dbPool.Query(ctx, "SELECT problem_id, title, slug, difficulty FROM problems ORDER BY created_at ASC")
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

func submitCode(c *gin.Context) {
	var req SubmitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	// 👇 SECURE: Grab the authenticated user_id extracted by the middleware
	// This ensures a student can ONLY submit code under their own account
	userID := c.MustGet("user_id").(string)

	submissionID := uuid.New().String()

	// 👇 SECURE: We pass the verified userID variable here instead of req.UserID
	_, err := dbPool.Exec(ctx,
		`INSERT INTO submissions (submission_id, user_id, problem_id, language, source_code, status) 
		 VALUES ($1, $2, $3, $4, $5, 'Pending')`,
		submissionID, userID, req.ProblemID, req.Language, req.SourceCode)

	if err != nil {
		fmt.Printf("[!] DB Insert Error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save submission"})
		return
	}

	redisMsg := fmt.Sprintf(`{"submission_id": "%s"}`, submissionID)
	err = rdb.LPush(ctx, "submission_queue", redisMsg).Err()
	if err != nil {
		fmt.Printf("[!] Redis Push Error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to queue submission"})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"message": "Submission received", "submission_id": submissionID, "status": "Pending"})
}

func getSubmissionStatus(c *gin.Context) {
	submissionID := c.Param("id")
	var status, language string

	var errorLogs *string

	err := dbPool.QueryRow(ctx, "SELECT status, language, error_logs FROM submissions WHERE submission_id = $1", submissionID).Scan(&status, &language, &errorLogs)
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
		"message":       message, // Sends the compiler errors to the React UI!
	})
}

func getProblemByID(c *gin.Context) {
	id := c.Param("id")
	var p Problem

	// 1. Fetch the problem details
	err := dbPool.QueryRow(ctx,
		"SELECT problem_id, title, description, difficulty FROM problems WHERE problem_id = $1",
		id).Scan(&p.ID, &p.Title, &p.Description, &p.Difficulty)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Problem not found"})
		return
	}

	// 2. Fetch the FIRST test case to use as a sample
	var sampleInput, sampleOutput string

	// We use LIMIT 1 to just grab the first one available
	err = dbPool.QueryRow(ctx,
		"SELECT input_data, expected_output FROM test_cases WHERE problem_id = $1 LIMIT 1",
		id).Scan(&sampleInput, &sampleOutput)

	// If we successfully found a test case, attach it to the struct
	if err == nil {
		p.SampleInput = &sampleInput
		p.SampleOutput = &sampleOutput
	}

	c.JSON(http.StatusOK, p)
}

func runCode(c *gin.Context) {
	var req RunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	runID := uuid.New().String()

	// Create a custom payload with the 'is_custom' flag
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

	// Drop it into the exact same queue the official submissions use
	err = rdb.LPush(ctx, "submission_queue", jsonPayload).Err()
	if err != nil {
		fmt.Printf("[!] Redis Push Error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to queue run"})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"run_id": runID, "status": "Pending"})
}

func getRunStatus(c *gin.Context) {
	runID := c.Param("id")

	// Check Redis for the result
	val, err := rdb.Get(ctx, "run_result:"+runID).Result()

	if err == redis.Nil {
		// Key doesn't exist yet, worker is still running
		c.JSON(http.StatusOK, gin.H{"status": "Pending"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Redis error"})
		return
	}

	// The worker finished! Parse the JSON string from Redis and send it to React
	var result map[string]interface{}
	json.Unmarshal([]byte(val), &result)
	c.JSON(http.StatusOK, result)
}

func getSubmissionHistory(c *gin.Context) {
	problemID := c.Param("id")
	userID := c.MustGet("user_id").(string) // Grab the authenticated user's ID

	// Fetch history ordered by newest first
	rows, err := dbPool.Query(ctx,
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

	var history []SubmissionHistoryEntry
	for rows.Next() {
		var entry SubmissionHistoryEntry
		if err := rows.Scan(&entry.ID, &entry.Language, &entry.Status, &entry.SubmittedAt); err != nil {
			continue // Skip broken rows
		}
		history = append(history, entry)
	}

	// If they have no history, return an empty array instead of null
	if history == nil {
		history = []SubmissionHistoryEntry{}
	}

	c.JSON(http.StatusOK, history)
}

func createProblem(c *gin.Context) {
	// 1. STRICT LOWERCASE RBAC CHECK (Relies on DB consistency)
	userRole, exists := c.Get("role")
	if !exists || (userRole != "professor" && userRole != "admin") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admins and professors can add problems."})
		return
	}

	// 2. PARSE THE INCOMING REACT FORM DATA
	var req CreateProblemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	slug := strings.ToLower(strings.ReplaceAll(req.Title, " ", "-"))

	// 3. DATABASE TRANSACTIONS
	// We need this to insert into BOTH problems and test_cases tables safely.
	tx, err := dbPool.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start database transaction"})
		return
	}
	defer tx.Rollback(ctx) // Safely rolls back if tx.Commit() isn't reached

	// Insert into problems table
	var newProblemID string
	err = tx.QueryRow(ctx,
		`INSERT INTO problems (title, slug, description, difficulty) 
		 VALUES ($1, $2, $3, $4) RETURNING problem_id`,
		req.Title, slug, req.Description, req.Difficulty).Scan(&newProblemID)

	if err != nil {
		fmt.Printf("[!] DB Insert Error (Problems): %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create problem"})
		return
	}

	// Insert the sample test case into the test_cases table
	// is_hidden is set to false so it shows up in the Arena UI
	_, err = tx.Exec(ctx,
		`INSERT INTO test_cases (problem_id, input_data, expected_output, is_hidden) 
		 VALUES ($1, $2, $3, false)`,
		newProblemID, req.SampleInput, req.SampleOutput)

	if err != nil {
		fmt.Printf("[!] DB Insert Error (Test Cases): %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to attach sample test cases"})
		return
	}

	// Commit the transaction since both queries succeeded
	err = tx.Commit(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to finalize problem creation"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":    "Problem created successfully!",
		"problem_id": newProblemID,
	})
}
