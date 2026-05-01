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
	"campuscompile/api/internal/middleware"
	redisPkg "campuscompile/api/internal/redis"
	"campuscompile/api/internal/repositories"
	"campuscompile/api/internal/routes"
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
	// Initialize security config only if IP_HASH_SALT is set, to avoid hard-failing
	// startup in environments (e.g., local/dev) where IP tracking is not configured.
	if ipHashSalt := os.Getenv("IP_HASH_SALT"); ipHashSalt == "" {
		log.Println("warning: IP_HASH_SALT is not set; IP-based security features are disabled")
	} else {
		middleware.InitSecurityConfig()
	}
	redisPkg.InitRedis(redisHost)
	handlers.InitOAuthConfig()
	storage.InitS3()

	// 3. SET UP GIN ROUTER
	router := gin.Default()
	router.SetTrustedProxies(nil)

	router.MaxMultipartMemory = 8 << 20 // 8 MiB

	// Define strict, type-safe limits
	const maxJSONSize int64 = 128 * 1024         // 128 KB for Source Code & JSON
	const maxUploadSize int64 = 15 * 1024 * 1024 // 15 MB for Test Case ZIPs

	// Instantiate our armors
	jsonArmor := middleware.PayloadArmor(maxJSONSize)
	uploadArmor := middleware.PayloadArmor(maxUploadSize)

	// --- Initialize Submissions Domain ---
	submissionRepo := repositories.NewSubmissionRepository(database.Pool)
	submissionService := services.NewSubmissionService(submissionRepo, redisPkg.Client)
	submissionController := controllers.NewSubmissionController(submissionService, redisPkg.Client)

	// --- Initialize Problems Domain ---
	problemRepo := repositories.NewProblemRepository(database.Pool)
	problemService := services.NewProblemService(problemRepo)
	problemController := controllers.NewProblemController(problemService)

	// --- Initialize Contests Domain (Phase 3) ---
	contestRepo := repositories.NewContestRepository(database.Pool)
	contestService := services.NewContestService(contestRepo, redisPkg.Client)
	contestController := controllers.NewContestController(contestService)

	// --- Initialize Playlist Domain (Phase 4) ---
	playlistRepo := repositories.NewPlaylistRepository(database.Pool)
	playlistService := services.NewPlaylistService(playlistRepo)
	playlistController := controllers.NewPlaylistController(playlistService)

	// Configure CORS for the React frontend dynamically
	allowedOrigin := os.Getenv("BASE_URL")
	if allowedOrigin == "" {
		allowedOrigin = "http://localhost:5173" // Safe local fallback
	}

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{allowedOrigin},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length", "Retry-After"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// 4. DEFINE ROUTES
	routes.Setup(
		router,
		problemController,
		submissionController,
		contestController,
		playlistController,
		jsonArmor,
		uploadArmor,
	)

	// 5. START SERVER WITH GRACEFUL SHUTDOWN

	// Create a context that listens for the interrupt signals (Ctrl+C or Docker stop)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Println("[*] API Server running on http://localhost:8080")

	// Pass the cancellable context to the daemons instead of context.Background()
	go contestService.StartAuditDaemon(ctx)
	go contestService.StartLeaderboardDaemon(ctx)
	go contestService.StartHeartbeatSweeperDaemon(ctx)

	analyticsService := services.NewAnalyticsService(database.Pool)
	go analyticsService.StartPlaylistAnalyticsDaemon(ctx)

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
