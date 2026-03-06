package main

import (
	"fmt"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	// Import your newly created internal packages
	"campuscompile/api/internal/database"
	"campuscompile/api/internal/handlers"
	"campuscompile/api/internal/middleware"
	redisPkg "campuscompile/api/internal/redis"
	"campuscompile/api/internal/storage"
)

func main() {
	// 1. DOCKER NETWORK CONFIGURATION
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}

	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" {
		redisHost = "localhost"
	}

	// 2. INITIALIZE SERVICES
	database.InitDB(dbHost)
	redisPkg.InitRedis(redisHost)
	handlers.InitOAuthConfig()
	storage.InitS3()

	// 3. SET UP GIN ROUTER
	router := gin.Default()
	router.SetTrustedProxies(nil)

	// Configure CORS for the React frontend
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// 4. DEFINE ROUTES

	// --- PUBLIC ROUTES ---
	authGroup := router.Group("/api/auth")
	{
		authGroup.GET("/login", handlers.HandleAzureLogin)
		authGroup.GET("/callback", handlers.HandleAzureCallback)
	}
	router.GET("/api/problems", handlers.GetProblems)

	// --- PROTECTED ROUTES (Requires JWT) ---
	protected := router.Group("/api")
	protected.Use(middleware.RequireAuth)
	{
		protected.POST("/auth/onboard", handlers.CompleteOnboarding)

		// Public Problem Access & Execution (Students + Faculty)
		protected.GET("/problems/:id", handlers.GetProblemByID)
		protected.POST("/submit", handlers.SubmitCode)
		protected.POST("/run", handlers.RunCode)
		protected.GET("/run/:id", handlers.GetRunStatus)
		protected.GET("/submissions/:id", handlers.GetSubmissionStatus)
		protected.GET("/submissions/history/:id", handlers.GetSubmissionHistory)

		// --- FACULTY ONLY ROUTES ---
		// This group chains RequireAuth -> RequireRole
		faculty := protected.Group("")
		faculty.Use(middleware.RequireRole("professor", "admin"))
		{
			// Manual problem creation
			faculty.POST("/problems", handlers.CreateProblem)
			router.POST("/problems/:id/testcases/batch", handlers.AddTestCasesBatch)
			// Stream massive test cases directly to MinIO (S3)
			faculty.POST("/problems/testcases", handlers.UploadTestCase)
		}
	}
	// 5. START SERVER
	fmt.Println("[*] API Server running on http://localhost:8080")
	router.Run(":8080")
}
