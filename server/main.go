package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
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

// --- MAIN ---
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
	router.POST("/api/auth/register", registerUser)
	router.POST("/api/auth/login", loginUser)
	router.GET("/api/problems", getProblems)
	router.GET("/api/problems/:id", getProblemByID)
	router.GET("/api/submissions/:id", getSubmissionStatus)

	// --- PROTECTED ROUTES (Bouncer checks token first) ---
	protected := router.Group("/api")
	protected.Use(requireAuth) // Attach the middleware
	{
		protected.POST("/submit", submitCode)
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
	err := dbPool.QueryRow(ctx, "SELECT status, language FROM submissions WHERE submission_id = $1", submissionID).Scan(&status, &language)
	if err != nil {
		if err.Error() == "no rows in result set" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Submission not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch status"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"submission_id": submissionID, "language": language, "status": status})
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
