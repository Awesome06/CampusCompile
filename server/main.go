package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
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
type Problem struct {
	ID          string `json:"problem_id"`
	Title       string `json:"title"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Difficulty  string `json:"difficulty"`
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

	// 1. Fetch the user's ID and hashed password from the database
	var userID, storedHash string
	err := dbPool.QueryRow(ctx, "SELECT user_id, password_hash FROM users WHERE email = $1", req.Email).Scan(&userID, &storedHash)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	// 2. Compare the provided password with the stored hash
	err = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(req.Password))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	// 3. Password is correct! Generate a JWT.
	// We store the user_id inside the token payload
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 72).Unix(), // Token expires in 3 days
	})

	// Sign the token with our secret key
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// 4. Return the token to the user
	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"token":   tokenString,
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

	err := dbPool.QueryRow(ctx,
		"SELECT problem_id, title, description, difficulty FROM problems WHERE problem_id = $1",
		id).Scan(&p.ID, &p.Title, &p.Description, &p.Difficulty)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Problem not found"})
		return
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
