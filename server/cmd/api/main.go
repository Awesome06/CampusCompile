package main

import (
	"fmt"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"campuscompile/api/internal/controllers"
	"campuscompile/api/internal/database"
	"campuscompile/api/internal/handlers"
	"campuscompile/api/internal/middleware"
	redisPkg "campuscompile/api/internal/redis"
	"campuscompile/api/internal/repositories"
	"campuscompile/api/internal/services"
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

	// --- Initialize Submissions Domain ---
	submissionRepo := repositories.NewSubmissionRepository(database.Pool)
	submissionService := services.NewSubmissionService(submissionRepo, redisPkg.Client)
	submissionController := controllers.NewSubmissionController(submissionService)

	// --- Initialize Problems Domain ---
	problemRepo := repositories.NewProblemRepository(database.Pool)
	problemService := services.NewProblemService(problemRepo)
	problemController := controllers.NewProblemController(problemService)

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
	router.GET("/api/problems", problemController.GetProblems) // Updated

	// --- PROTECTED ROUTES (Requires JWT) ---
	protected := router.Group("/api")
	protected.Use(middleware.RequireAuth)
	{
		protected.POST("/auth/onboard", handlers.CompleteOnboarding)

		// Public Problem Access & Execution (Students + Faculty)
		protected.GET("/problems/:id", problemController.GetProblemByID) // Updated

		// Submissions
		protected.POST("/submit", submissionController.SubmitCode)
		protected.POST("/run", submissionController.RunCode)                                 // Updated
		protected.GET("/run/:id", submissionController.GetRunStatus)                         // Updated
		protected.GET("/submissions/:id", submissionController.GetSubmissionStatus)          // Updated
		protected.GET("/submissions/history/:id", submissionController.GetSubmissionHistory) // Updated

		protected.GET("/submissions/stream/:id", submissionController.StreamSubmissionStatus)
		protected.GET("/run/stream/:id", submissionController.StreamRunStatus)

		// --- FACULTY & ADMIN ROUTES ---
		// Both Professors and Admins can create and edit problems
		faculty := protected.Group("")
		faculty.Use(middleware.RequireRole("professor", "admin"))
		{
			// Problem Management
			faculty.POST("/problems", problemController.CreateProblem)    // Updated
			faculty.PUT("/problems/:id", problemController.UpdateProblem) // Updated

			// Test Cases
			faculty.POST("/problems/:id/testcases/batch", problemController.AddTestCasesBatch)      // Updated
			faculty.PUT("/problems/:id/testcases/sync", problemController.SyncTestCasesBatch)       // Updated
			faculty.GET("/problems/:id/testcases/all", problemController.GetAllTestCasesForProblem) // Updated

			// Dashboard
			faculty.GET("/faculty/problems", problemController.GetFacultyProblems) // Updated
		}

		// --- ADMIN ONLY ROUTES ---
		adminGroup := protected.Group("")
		adminGroup.Use(middleware.RequireRole("admin"))
		{
			adminGroup.DELETE("/problems/:id", problemController.DeleteProblem) // Updated
		}
	}
	// 5. START SERVER
	fmt.Println("[*] API Server running on http://localhost:8080")
	router.Run(":8080")
}
