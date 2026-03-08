package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"campuscompile/api/internal/models"
	"campuscompile/api/internal/services"
)

type ContestController struct {
	service services.ContestService
}

func NewContestController(service services.ContestService) *ContestController {
	return &ContestController{service: service}
}

// StreamLeaderboard is the SSE endpoint that keeps the React UI perfectly in sync
func (ctrl *ContestController) StreamLeaderboard(c *gin.Context) {
	contestID := c.Param("id")

	// 1. Subscribe to the contest's specific Redis broadcast channel
	channelName := fmt.Sprintf("contest:leaderboard_updates:%s", contestID)
	ch, cleanup := ctrl.service.SubscribeToChannel(c.Request.Context(), channelName)
	defer cleanup()

	// 2. Set the necessary headers to keep the HTTP connection alive for SSE
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Flush()

	// 3. Immediately fetch and send the current leaderboard state so the UI isn't blank
	initialLeaderboard, err := ctrl.service.FetchCurrentLeaderboard(c.Request.Context(), contestID)
	if err == nil {
		initialData, _ := json.Marshal(initialLeaderboard)
		c.SSEvent("message", string(initialData))
		c.Writer.Flush()
	}

	// 4. Enter the infinite loop: listen for live updates from the Python Worker / Go Engine
	for {
		select {
		case <-c.Request.Context().Done():
			// The student closed the tab or navigated away; kill the connection cleanly
			return
		case msg := <-ch:
			// A student just got an 'AC' and the leaderboard shifted! Push the new data.
			c.SSEvent("message", msg.Payload)
			c.Writer.Flush()
		}
	}
}

// GetContests fetches the high-level list of all contests
func (ctrl *ContestController) GetContests(c *gin.Context) {
	// 1. Extract the raw JWT claims
	userRole, _ := c.Get("role")
	userID, _ := c.Get("user_id")

	// 2. Safely extract demographics (will be empty for Admins/Professors, which is fine)
	demo := services.UserDemographics{
		Role:           userRole.(string),
		UserID:         userID.(string),
		Course:         getString(c, "course"),
		Department:     getString(c, "department"),
		Batch:          getString(c, "batch"),
		Section:        getString(c, "section"),
		StudentGroup:   getString(c, "student_group"),
		GraduationYear: getInt(c, "graduation_year"),
	}

	// 3. Fetch cleanly filtered contests
	contests, err := ctrl.service.FetchContests(c.Request.Context(), demo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch contests"})
		return
	}

	c.JSON(http.StatusOK, contests)
}

// GetContestDetails fetches specific metadata and checks if the current user is registered
func (ctrl *ContestController) GetContestDetails(c *gin.Context) {
	contestID := c.Param("id")
	userID := c.MustGet("user_id").(string)

	contest, err := ctrl.service.FetchContestByID(c.Request.Context(), contestID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Contest not found"})
		return
	}

	// Check if this specific user has clicked "Join Arena" yet
	isRegistered, _ := ctrl.service.IsUserEnrolled(c.Request.Context(), contestID, userID)

	c.JSON(http.StatusOK, gin.H{
		"contest":       contest,
		"is_registered": isRegistered,
	})
}

// RegisterForContest handles the user opting into the arena and initializing their Redis score
func (ctrl *ContestController) RegisterForContest(c *gin.Context) {
	contestID := c.Param("id")
	userID := c.MustGet("user_id").(string)

	err := ctrl.service.EnrollUser(c.Request.Context(), contestID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register for contest"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Successfully entered the arena!"})
}

// GetLeaderboard provides the initial static snapshot of the leaderboard before the SSE stream takes over
func (ctrl *ContestController) GetLeaderboard(c *gin.Context) {
	contestID := c.Param("id")

	leaderboard, err := ctrl.service.FetchEnrichedLeaderboard(c.Request.Context(), contestID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load leaderboard"})
		return
	}

	c.JSON(http.StatusOK, leaderboard)
}

func (ctrl *ContestController) GetContestProblems(c *gin.Context) {
	contestID := c.Param("id")

	problems, err := ctrl.service.FetchContestProblems(c.Request.Context(), contestID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch contest problems"})
		return
	}

	c.JSON(http.StatusOK, problems)
}

func (ctrl *ContestController) CreateContest(c *gin.Context) {
	var input models.CreateContestInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload: " + err.Error()})
		return
	}

	// Securely grab the professor/admin's ID from the JWT
	userID := c.MustGet("user_id").(string)

	contest := models.Contest{
		Title:            input.Title,
		HostOrganization: input.HostOrganization,
		StartTime:        input.StartTime,
		EndTime:          input.EndTime,
		IsPublic:         input.IsPublic,
		AccessRules:      input.AccessRules,
		AuthorID:         &userID,
	}

	contestID, err := ctrl.service.CreateContest(c.Request.Context(), contest, input.Problems)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to forge contest in database"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":    "Contest successfully forged",
		"contest_id": contestID,
	})
}

func getString(c *gin.Context, key string) string {
	val, exists := c.Get(key)
	if !exists {
		return ""
	}
	str, _ := val.(string)
	return str
}

func getInt(c *gin.Context, key string) int {
	val, exists := c.Get(key)
	if !exists {
		return 0
	}
	num, _ := val.(int)
	return num
}
