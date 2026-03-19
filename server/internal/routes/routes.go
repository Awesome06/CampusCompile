package routes

import (
	"campuscompile/api/internal/controllers"
	"campuscompile/api/internal/handlers"
	"campuscompile/api/internal/middleware"

	"github.com/gin-gonic/gin"
)

// Setup configures all API routes for the backend Application
func Setup(
	router *gin.Engine,
	problemController *controllers.ProblemController,
	submissionController *controllers.SubmissionController,
	contestController *controllers.ContestController,
	jsonArmor gin.HandlerFunc,
	uploadArmor gin.HandlerFunc,
) {
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
	{
		protected.POST("/auth/onboard", jsonArmor, handlers.CompleteOnboarding)
		protected.GET("/problems/:id", problemController.GetProblemByID)

		// Submissions
		protected.POST("/submit", jsonArmor, submissionController.SubmitCode)
		protected.POST("/run", jsonArmor, submissionController.RunCode)
		protected.GET("/run/:id", submissionController.GetRunStatus)
		protected.GET("/submissions/:id", submissionController.GetSubmissionStatus)
		protected.GET("/submissions/history/:id", submissionController.GetSubmissionHistory)

		protected.GET("/submissions/stream/:id", middleware.RequireSSECap("submission", 3), submissionController.StreamSubmissionStatus)
		protected.GET("/run/stream/:id", middleware.RequireSSECap("run", 3), submissionController.StreamRunStatus)

		// --- CONTEST ROUTES (PHASE 3) ---
		// ANY logged-in user can view the list of upcoming/past contests
		protected.GET("/contests", contestController.GetContests)

		// The Bouncer: You must pass demographic clearance to enter the arena or view the leaderboard
		arena := protected.Group("/contests/:id")
		arena.Use(middleware.RequireContestClearance())
		{
			// 1. The Tracker
			ipTracker := middleware.TrackContestIP()

			// 2. High-Value Endpoints (Tracked)
			arena.GET("", ipTracker, contestController.GetContestDetails)
			arena.POST("/register", ipTracker, contestController.RegisterForContest)
			arena.GET("/problems", ipTracker, contestController.GetContestProblems)

			// 3. High-Frequency Endpoints (Untracked)
			arena.GET("/leaderboard", contestController.GetLeaderboard)
			arena.GET("/leaderboard/stream", middleware.RequireSSECap("leaderboard", 2), contestController.StreamLeaderboard)
			arena.POST("/telemetry", jsonArmor, contestController.LogTelemetry)
			arena.POST("/telemetry/batch", jsonArmor, contestController.LogTelemetryBatch)
		}

		// --- FACULTY & ADMIN ROUTES ---
		faculty := protected.Group("")
		faculty.Use(middleware.RequireRole("professor", "admin"))
		{
			faculty.GET("/problems/:id/testcases/all", problemController.GetAllTestCasesForProblem)
			faculty.DELETE("/problems/:id/testcases", problemController.ClearTestCases)
			faculty.GET("/faculty/problems", problemController.GetFacultyProblems)

			faculty.POST("/problems", jsonArmor, problemController.CreateProblem)
			faculty.PUT("/problems/:id", jsonArmor, problemController.UpdateProblem)

			// Apply jsonArmor to Contest creation and modification
			faculty.POST("/contests", jsonArmor, contestController.CreateContest)
			faculty.PUT("/contests/:id", jsonArmor, contestController.UpdateContest)

			// Test case uploads remain under the heavier uploadArmor
			faculty.POST("/problems/:id/testcases/batch", uploadArmor, problemController.UploadTestCasesBatch)

			faculty.DELETE("/problems/:id", problemController.DeleteProblem)
			faculty.DELETE("/contests/:id", contestController.DeleteContest)
		}

		// --- ADMIN ONLY ROUTES ---
		adminGroup := protected.Group("")
		adminGroup.Use(middleware.RequireRole("admin"))
		{
		}
	}
}
