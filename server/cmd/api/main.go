package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"campuscompile/api/internal/controllers"
	"campuscompile/api/internal/database"
	"campuscompile/api/internal/handlers"
	"campuscompile/api/internal/middleware" // Make sure this is imported
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

	// --- Initialize Contests Domain (Phase 3) ---
	contestRepo := repositories.NewContestRepository(database.Pool)
	contestService := services.NewContestService(contestRepo, redisPkg.Client)
	contestController := controllers.NewContestController(contestService)

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
	router.GET("/api/problems", problemController.GetProblems)

	// --- PROTECTED ROUTES (Requires JWT) ---
	protected := router.Group("/api")
	protected.Use(middleware.RequireAuth)
	protected.Use(middleware.PayloadArmor())
	{
		protected.POST("/auth/onboard", handlers.CompleteOnboarding)
		protected.GET("/problems/:id", problemController.GetProblemByID)

		// Submissions
		protected.POST("/submit", submissionController.SubmitCode)
		protected.POST("/run", submissionController.RunCode)
		protected.GET("/run/:id", submissionController.GetRunStatus)
		protected.GET("/submissions/:id", submissionController.GetSubmissionStatus)
		protected.GET("/submissions/history/:id", submissionController.GetSubmissionHistory)

		protected.GET("/submissions/stream/:id", submissionController.StreamSubmissionStatus)
		protected.GET("/run/stream/:id", submissionController.StreamRunStatus)

		// 👇 --- CONTEST ROUTES (PHASE 3) ---
		// ANY logged-in user can view the list of upcoming/past contests
		protected.GET("/contests", contestController.GetContests)

		// The Bouncer: You must pass demographic clearance to enter the arena or view the leaderboard
		arena := protected.Group("/contests/:id")
		arena.Use(middleware.RequireContestClearance())
		{
			arena.GET("", contestController.GetContestDetails)
			arena.POST("/register", contestController.RegisterForContest)
			arena.GET("/leaderboard", contestController.GetLeaderboard)
			arena.GET("/leaderboard/stream", contestController.StreamLeaderboard)
			arena.GET("/problems", contestController.GetContestProblems)
			arena.POST("/telemetry", contestController.LogTelemetry)
			arena.POST("/telemetry/batch", contestController.LogTelemetryBatch)
		}

		// --- FACULTY & ADMIN ROUTES ---
		faculty := protected.Group("")
		faculty.Use(middleware.RequireRole("professor", "admin"))
		{
			faculty.POST("/problems", problemController.CreateProblem)
			faculty.PUT("/problems/:id", problemController.UpdateProblem)
			faculty.GET("/problems/:id/testcases/all", problemController.GetAllTestCasesForProblem)
			faculty.DELETE("/problems/:id/testcases", problemController.ClearTestCases)
			faculty.POST("/problems/:id/testcases/batch", problemController.UploadTestCasesBatch)
			faculty.GET("/faculty/problems", problemController.GetFacultyProblems)

			// Faculty Contest Management
			faculty.POST("/contests", contestController.CreateContest)
			faculty.PUT("/contests/:id", contestController.UpdateContest)

			faculty.DELETE("/problems/:id", problemController.DeleteProblem)
			faculty.DELETE("/contests/:id", contestController.DeleteContest)
		}

		// --- ADMIN ONLY ROUTES ---
		adminGroup := protected.Group("")
		adminGroup.Use(middleware.RequireRole("admin"))
		{

		}
	}
	// 5. START SERVER WITH GRACEFUL SHUTDOWN

	// Create a context that listens for the interrupt signals (Ctrl+C or Docker stop)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Println("[*] API Server running on http://localhost:8080")

	// Pass the cancellable context to the daemons instead of context.Background()
	go contestService.StartAuditDaemon(ctx)
	go contestService.StartLeaderboardDaemon(ctx)

	// Configure the HTTP server manually instead of using router.Run()
	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	// Run the server in a non-blocking goroutine
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("[!] Listen error: %s\n", err)
		}
	}()

	// Block main thread until the interrupt signal is received
	<-ctx.Done()

	// Restore default behavior on the interrupt signal
	stop()
	fmt.Println("\n[*] Shutting down gracefully. Pressing Ctrl+C again will force exit.")

	// Give the server and active HTTP connections 5 seconds to finish their work
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatal("[!] Server forced to shutdown: ", err)
	}

	fmt.Println("[*] API Server cleanly exited")
}
