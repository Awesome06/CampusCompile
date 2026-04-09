package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	appErrors "campuscompile/api/internal/errors"
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
	if err := uuid.Validate(contestID); err != nil {
		c.Error(appErrors.NewAppError(http.StatusBadRequest, err, "Malformed UUID format for contest ID"))
		return
	}
	userRole, err := getSafeString(c, "role")
	if err != nil {
		c.Error(appErrors.NewAppError(http.StatusUnauthorized, err, "Malformed authentication token context"))
		return
	}

	// 1. The Bouncer - Fetch the ENRICHED leaderboard immediately BEFORE allocating resources
	auditStatus, initialLeaderboard, err := ctrl.service.FetchEnrichedLeaderboard(c.Request.Context(), contestID)

	// If the contest is fully graded/audited, reject the persistent stream to save server resources.
	if err == nil && (auditStatus == "completed" || auditStatus == "failed") {
		c.Error(appErrors.NewAppError(http.StatusGone, nil, "Contest is finalized. Persistent streaming is disabled to conserve resources."))
		return // Terminates the request immediately!
	}

	// 2. Subscribe to the contest's specific Redis broadcast channel
	channelSuffix := "student"
	if userRole == "admin" || userRole == "professor" {
		channelSuffix = "faculty"
	}

	channelName := fmt.Sprintf("contest:leaderboard_updates:%s:%s", channelSuffix, contestID)
	ch, cleanup := ctrl.service.SubscribeToChannel(c.Request.Context(), channelName)
	defer cleanup()

	// 3. Set the necessary headers to keep the HTTP connection alive for SSE
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache, no-transform")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.Flush()

	// 4. Send Initial Payload
	if err == nil {
		if userRole == "student" {
			for _, entry := range initialLeaderboard {
				delete(entry, "alerts") // Hide MOSS plagiarism alerts from students
			}
		}
		initialData, _ := json.Marshal(map[string]interface{}{
			"audit_status": auditStatus,
			"leaderboard":  initialLeaderboard,
		})
		c.SSEvent("message", string(initialData))
		c.Writer.Flush()
	}

	// 5. Listen for live updates from the Python Worker / Go Engine
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case msg := <-ch:
			c.SSEvent("message", msg)
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

	limit, offset := parsePaginationArgs(c, 10)
	searchQuery := c.Query("search")
	viewMode := c.Query("view_mode")

	// 3. Fetch cleanly filtered contests
	contests, err := ctrl.service.FetchContests(c.Request.Context(), demo, limit, offset, searchQuery, viewMode)
	if err != nil {
		c.Error(appErrors.NewAppError(http.StatusInternalServerError, err, "Failed to fetch contests"))
		return
	}

	c.JSON(http.StatusOK, contests)
}

// GetContestDetails fetches specific metadata and checks if the current user is registered
func (ctrl *ContestController) GetContestDetails(c *gin.Context) {
	contestID := c.Param("id")
	if err := uuid.Validate(contestID); err != nil {
		c.Error(appErrors.NewAppError(http.StatusBadRequest, err, "Malformed UUID format for contest ID"))
		return
	}
	userID, err := getSafeString(c, "user_id")
	if err != nil {
		c.Error(appErrors.NewAppError(http.StatusUnauthorized, err, "Malformed authentication token context"))
		return
	}
	userRole, err := getSafeString(c, "role")
	if err != nil {
		c.Error(appErrors.NewAppError(http.StatusUnauthorized, err, "Malformed authentication token context"))
		return
	}

	contest, err := ctrl.service.FetchContestByID(c.Request.Context(), contestID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Contest not found"})
		return
	}

	// 🔒 API GUARD: PREVENT EARLY ACCESS 🔒
	if time.Now().Before(contest.StartTime) && userRole != "admin" {
		if contest.AuthorID == nil || *contest.AuthorID != userID {
			c.Error(appErrors.NewAppError(http.StatusForbidden, nil, "The Arena is locked. Please wait until the start time."))
			return
		}
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
	if err := uuid.Validate(contestID); err != nil {
		c.Error(appErrors.NewAppError(http.StatusBadRequest, err, "Malformed UUID format for contest ID"))
		return
	}
	userID, err := getSafeString(c, "user_id")
	if err != nil {
		c.Error(appErrors.NewAppError(http.StatusUnauthorized, err, "Malformed authentication token context"))
		return
	}

	err = ctrl.service.EnrollUser(c.Request.Context(), contestID, userID)
	if err != nil {
		c.Error(appErrors.NewAppError(http.StatusInternalServerError, err, "Failed to register for contest"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Successfully entered the arena!"})
}

func (ctrl *ContestController) ProcessHeartbeat(c *gin.Context) {
	contestID := c.Param("id")
	userID, err := getSafeString(c, "user_id")
	if err != nil {
		c.Error(appErrors.NewAppError(http.StatusUnauthorized, err, "Malformed authentication token context"))
		return
	}

	err = ctrl.service.ProcessHeartbeat(c.Request.Context(), contestID, userID)
	if err != nil {
		c.Error(appErrors.NewAppError(http.StatusInternalServerError, err, "Failed to process heartbeat"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "alive"})
}

// GetLeaderboard provides the initial static snapshot of the leaderboard before the SSE stream takes over
func (ctrl *ContestController) GetLeaderboard(c *gin.Context) {
	contestID := c.Param("id")
	if err := uuid.Validate(contestID); err != nil {
		c.Error(appErrors.NewAppError(http.StatusBadRequest, err, "Malformed UUID format for contest ID"))
		return
	}
	userRole, err := getSafeString(c, "role")
	if err != nil {
		c.Error(appErrors.NewAppError(http.StatusUnauthorized, err, "Malformed authentication token context"))
		return
	}

	// 1. Check if the contest is over
	contest, err := ctrl.service.FetchContestByID(c.Request.Context(), contestID)
	if err == nil && time.Now().After(contest.EndTime) {
		// Try to serve the static snapshot
		snapshot, err := ctrl.service.GetFinalizedLeaderboard(c.Request.Context(), contestID)
		if err == nil && snapshot != nil {
			// Security: Strip out MOSS plagiarism alerts if the user is a student
			if userRole == "student" {
				if leaderboard, ok := snapshot["leaderboard"].([]interface{}); ok {
					for _, entryIntf := range leaderboard {
						if entry, ok := entryIntf.(map[string]interface{}); ok {
							delete(entry, "alerts")
						}
					}
				}
			}
			c.JSON(http.StatusOK, snapshot)
			return
		}
	}

	// 2. Fetch the live state
	auditStatus, leaderboard, err := ctrl.service.FetchEnrichedLeaderboard(c.Request.Context(), contestID)
	if err != nil {
		c.Error(appErrors.NewAppError(http.StatusInternalServerError, err, "Failed to load leaderboard"))
		return
	}

	// 3. Snapshot it passively if the contest is over and no snapshot existed yet
	if time.Now().After(contest.EndTime) {
		dataToSnapshot := map[string]interface{}{
			"audit_status": auditStatus,
			"leaderboard":  leaderboard,
		}
		// Goroutine to not block the response
		go func(cid string, snap map[string]interface{}) {
			_ = ctrl.service.SnapshotLeaderboard(context.Background(), cid, snap)
		}(contestID, dataToSnapshot)
	}

	// 4. Security: Strip out MOSS plagiarism alerts if the user is a student
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
	if err := uuid.Validate(contestID); err != nil {
		c.Error(appErrors.NewAppError(http.StatusBadRequest, err, "Malformed UUID format for contest ID"))
		return
	}
	userRole, err := getSafeString(c, "role")
	if err != nil {
		c.Error(appErrors.NewAppError(http.StatusUnauthorized, err, "Malformed authentication token context"))
		return
	}
	userID, err := getSafeString(c, "user_id")
	if err != nil {
		c.Error(appErrors.NewAppError(http.StatusUnauthorized, err, "Malformed authentication token context"))
		return
	}

	// 🔒 API GUARD: PREVENT PROBLEM LEAKS 🔒
	contest, err := ctrl.service.FetchContestByID(c.Request.Context(), contestID)
	if err == nil && time.Now().Before(contest.StartTime) && userRole != "admin" {
		if contest.AuthorID == nil || *contest.AuthorID != userID {
			c.Error(appErrors.NewAppError(http.StatusForbidden, nil, "Classified: Problems cannot be viewed before the contest begins."))
			return
		}
	}

	if userRole == "student" {
		disqualified, err := ctrl.service.IsUserDisqualified(c.Request.Context(), contestID, userID)
		if err == nil && disqualified {
			c.Error(appErrors.NewAppError(http.StatusForbidden, nil, "You have been disqualified from this contest due to session abandonment or abnormal activity."))
			return
		}
	}

	// 👇 Pass the userID down to the SQL query
	problems, err := ctrl.service.FetchContestProblems(c.Request.Context(), contestID, userID)
	if err != nil {
		c.Error(appErrors.NewAppError(http.StatusInternalServerError, err, "Failed to fetch contest problems"))
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
	userID, err := getSafeString(c, "user_id")
	if err != nil {
		c.Error(appErrors.NewAppError(http.StatusUnauthorized, err, "Malformed authentication token context"))
		return
	}

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
		c.Error(appErrors.NewAppError(http.StatusInternalServerError, err, "Failed to forge contest in database"))
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":    "Contest successfully forged",
		"contest_id": contestID,
	})
}

func (ctrl *ContestController) UpdateContest(c *gin.Context) {
	contestID := c.Param("id")
	if err := uuid.Validate(contestID); err != nil {
		c.Error(appErrors.NewAppError(http.StatusBadRequest, err, "Malformed UUID format for contest ID"))
		return
	}
	userID, err := getSafeString(c, "user_id")
	if err != nil {
		c.Error(appErrors.NewAppError(http.StatusUnauthorized, err, "Malformed authentication token context"))
		return
	}
	userRole, err := getSafeString(c, "role")
	if err != nil {
		c.Error(appErrors.NewAppError(http.StatusUnauthorized, err, "Malformed authentication token context"))
		return
	}

	// 1. Fetch the existing contest to check ownership and time
	existingContest, err := ctrl.service.FetchContestByID(c.Request.Context(), contestID)
	if err != nil {
		c.Error(appErrors.NewAppError(http.StatusNotFound, err, "Contest not found"))
		return
	}

	// 2. Security Check: Ownership Check
	if userRole != "admin" {
		if existingContest.AuthorID == nil || *existingContest.AuthorID != userID {
			c.Error(appErrors.NewAppError(http.StatusForbidden, nil, "Access Denied: You do not own this contest"))
			return
		}
	}

	// 3. 🔒 NEW: TIME LOCK CHECK 🔒
	if time.Now().After(existingContest.StartTime) && userRole != "admin" {
		c.Error(appErrors.NewAppError(http.StatusForbidden, nil, "Time Lock Active: You cannot modify an arena that has already started."))
		return
	}

	// 4. Parse the payload
	var input models.CreateContestInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(appErrors.NewAppError(http.StatusBadRequest, err, "Invalid request payload format structure."))
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
		c.Error(appErrors.NewAppError(http.StatusInternalServerError, err, "Failed to update contest"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Arena configurations updated successfully"})
}

func (ctrl *ContestController) DeleteContest(c *gin.Context) {
	contestID := c.Param("id")
	if err := uuid.Validate(contestID); err != nil {
		c.Error(appErrors.NewAppError(http.StatusBadRequest, err, "Malformed UUID format for contest ID"))
		return
	}
	userRole, err := getSafeString(c, "role")
	if err != nil {
		c.Error(appErrors.NewAppError(http.StatusUnauthorized, err, "Malformed authentication token context"))
		return
	}
	userID, err := getSafeString(c, "user_id")
	if err != nil {
		c.Error(appErrors.NewAppError(http.StatusUnauthorized, err, "Malformed authentication token context"))
		return
	}

	// Fetch the contest to check its timing and ownership
	existingContest, err := ctrl.service.FetchContestByID(c.Request.Context(), contestID)
	if err != nil {
		c.Error(appErrors.NewAppError(http.StatusNotFound, err, "Contest not found"))
		return
	}

	// 🔒 PROFESSOR DELETION RULES 🔒
	if userRole == "professor" {
		if existingContest.AuthorID == nil || *existingContest.AuthorID != userID {
			c.Error(appErrors.NewAppError(http.StatusForbidden, nil, "Access Denied: You do not own this contest."))
			return
		}
		if existingContest.IsPublic {
			c.Error(appErrors.NewAppError(http.StatusForbidden, nil, "Access Denied: Professors cannot delete published contests. Please contact an Administrator."))
			return
		}
	}

	// 🔒 TIME LOCK CHECK (Kept from Phase 3) 🔒
	if time.Now().After(existingContest.StartTime) && userRole != "admin" {
		c.Error(appErrors.NewAppError(http.StatusForbidden, nil, "Time Lock Active: You cannot delete a contest that has already started."))
		return
	}

	err = ctrl.service.DeleteContest(c.Request.Context(), contestID)
	if err != nil {
		c.Error(appErrors.NewAppError(http.StatusInternalServerError, err, "Failed to delete contest"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Arena permanently destroyed"})
}

func (ctrl *ContestController) LogTelemetry(c *gin.Context) {
	contestID := c.Param("id")
	if err := uuid.Validate(contestID); err != nil {
		c.Error(appErrors.NewAppError(http.StatusBadRequest, err, "Malformed UUID format for contest ID"))
		return
	}
	userID, err := getSafeString(c, "user_id")
	if err != nil {
		c.Error(appErrors.NewAppError(http.StatusUnauthorized, err, "Malformed authentication token context"))
		return
	}
	userRole, err := getSafeString(c, "role")
	if err != nil {
		c.Error(appErrors.NewAppError(http.StatusUnauthorized, err, "Malformed authentication token context"))
		return
	}

	var payload models.TelemetryPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.Error(appErrors.NewAppError(http.StatusBadRequest, err, "Invalid payload format"))
		return
	}

	// ISOLATE TELEMETRY
	// Drop the payload silently for faculty so they don't pollute the anti-cheat logs
	if userRole == "admin" || userRole == "professor" {
		c.JSON(http.StatusOK, gin.H{"status": "ignored_for_faculty"})
		return
	}

	// Fire and forget for students, with error logging to catch DB drops
	go func() {
		err := ctrl.service.LogTelemetry(context.Background(), contestID, userID, payload)
		if err != nil {
			slog.Error("Telemetry drop detected",
				"component", "LogTelemetry",
				"contest_id", contestID,
				"user_id", userID,
				"error", err,
			)
		}
	}()

	c.JSON(http.StatusOK, gin.H{"status": "logged"})
}

func (ctrl *ContestController) LogTelemetryBatch(c *gin.Context) {
	contestID := c.Param("id")
	if err := uuid.Validate(contestID); err != nil {
		c.Error(appErrors.NewAppError(http.StatusBadRequest, err, "Malformed UUID format for contest ID"))
		return
	}
	userID, err := getSafeString(c, "user_id")
	if err != nil {
		c.Error(appErrors.NewAppError(http.StatusUnauthorized, err, "Malformed authentication token context"))
		return
	}
	userRole, err := getSafeString(c, "role")
	if err != nil {
		c.Error(appErrors.NewAppError(http.StatusUnauthorized, err, "Malformed authentication token context"))
		return
	}

	var payload models.BatchTelemetryPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.Error(appErrors.NewAppError(http.StatusBadRequest, err, "Invalid batch payload format"))
		return
	}

	// 2. ISOLATE TELEMETRY: Drop the batch silently for faculty
	if userRole == "admin" || userRole == "professor" {
		c.JSON(http.StatusOK, gin.H{"status": "ignored_for_faculty", "count": 0})
		return
	}

	// 3. Fire and forget for students
	go func() {
		for _, event := range payload.Events {
			err := ctrl.service.LogTelemetry(context.Background(), contestID, userID, event)
			if err != nil {
				slog.Error("Batch telemetry drop detected",
					"component", "ContestController.LogTelemetryBatch",
					"contest_id", contestID,
					"user_id", userID,
					"error", err,
				)
			}
		}
	}()

	c.JSON(http.StatusOK, gin.H{"status": "batch_logged", "count": len(payload.Events)})
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
