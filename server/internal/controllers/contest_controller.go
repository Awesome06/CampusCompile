package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

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
	userRole := c.MustGet("role").(string)
	channelSuffix := "student"
	if userRole == "admin" || userRole == "professor" {
		channelSuffix = "faculty"
	}

	channelName := fmt.Sprintf("contest:leaderboard_updates:%s:%s", channelSuffix, contestID)
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
	demo := models.UserDemographics{
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

	auditStatus, leaderboard, err := ctrl.service.FetchEnrichedLeaderboard(c.Request.Context(), contestID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load leaderboard"})
		return
	}

	userRole := c.MustGet("role").(string)
	if userRole == "student" {
		for _, entry := range leaderboard {
			delete(entry, "alerts")
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"audit_status": auditStatus,
		"leaderboard":  leaderboard,
	})
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

func (ctrl *ContestController) UpdateContest(c *gin.Context) {
	contestID := c.Param("id")
	userID := c.MustGet("user_id").(string)
	userRole := c.MustGet("role").(string)

	// 1. Fetch the existing contest to check ownership and time
	existingContest, err := ctrl.service.FetchContestByID(c.Request.Context(), contestID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Contest not found"})
		return
	}

	// 2. Security Check: Ownership Check
	if userRole != "admin" {
		if existingContest.AuthorID == nil || *existingContest.AuthorID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access Denied: You do not own this contest"})
			return
		}
	}

	// 3. 🔒 NEW: TIME LOCK CHECK 🔒
	if time.Now().After(existingContest.StartTime) && userRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Time Lock Active: You cannot modify an arena that has already started."})
		return
	}

	// 4. Parse the payload
	var input models.CreateContestInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload: " + err.Error()})
		return
	}

	contest := models.Contest{
		Title:            input.Title,
		HostOrganization: input.HostOrganization,
		StartTime:        input.StartTime,
		EndTime:          input.EndTime,
		IsPublic:         input.IsPublic,
		AccessRules:      input.AccessRules,
	}

	// 5. Execute the update
	err = ctrl.service.UpdateContest(c.Request.Context(), contestID, contest, input.Problems)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update contest"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Arena configurations updated successfully"})
}

func (ctrl *ContestController) DeleteContest(c *gin.Context) {
	contestID := c.Param("id")
	userRole := c.MustGet("role").(string)

	// Fetch the contest to check its timing
	existingContest, err := ctrl.service.FetchContestByID(c.Request.Context(), contestID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Contest not found"})
		return
	}

	// 🔒 NEW: TIME LOCK CHECK 🔒
	if time.Now().After(existingContest.StartTime) && userRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Time Lock Active: You cannot delete a contest that has already started."})
		return
	}

	err = ctrl.service.DeleteContest(c.Request.Context(), contestID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete contest"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Arena permanently destroyed"})
}

func (ctrl *ContestController) LogTelemetry(c *gin.Context) {
	contestID := c.Param("id")
	userID := c.MustGet("user_id").(string)

	var payload models.TelemetryPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload format"})
		return
	}

	// Fire and forget; don't block the student's browser waiting for a DB write
	go ctrl.service.LogTelemetry(context.Background(), contestID, userID, payload)

	c.JSON(http.StatusOK, gin.H{"status": "logged"})
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
