package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	appErrors "campuscompile/api/internal/errors"
	"campuscompile/api/internal/models"
	"campuscompile/api/internal/services"
)

type PlaylistController struct {
	service services.PlaylistService
}

func NewPlaylistController(service services.PlaylistService) *PlaylistController {
	return &PlaylistController{service: service}
}

func (ctrl *PlaylistController) CreatePlaylist(c *gin.Context) {
	var req models.CreatePlaylistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(appErrors.NewAppError(http.StatusBadRequest, err, "Invalid payload format for playlist"))
		return
	}

	authorID := c.MustGet("user_id").(string)

	playlistID, err := ctrl.service.CreatePlaylist(c.Request.Context(), req, authorID)
	if err != nil {
		c.Error(appErrors.NewAppError(http.StatusInternalServerError, err, "Failed to create playlist"))
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":     "Playlist created successfully",
		"playlist_id": playlistID,
	})
}

func (ctrl *PlaylistController) GetPlaylists(c *gin.Context) {
	limit, offset := parsePaginationArgs(c, 25)
	viewMode := c.Query("view_mode") // "public" or "faculty"

	var userID string
	if uid, exists := c.Get("user_id"); exists {
		if strUid, ok := uid.(string); ok {
			userID = strUid
		}
	}

	playlists, err := ctrl.service.GetPlaylists(c.Request.Context(), userID, viewMode, limit, offset)
	if err != nil {
		c.Error(appErrors.NewAppError(http.StatusInternalServerError, err, "Failed to fetch playlists"))
		return
	}

	c.JSON(http.StatusOK, playlists)
}

func (ctrl *PlaylistController) GetPlaylistByID(c *gin.Context) {
	playlistID := c.Param("id")

	playlist, err := ctrl.service.GetPlaylistByID(c.Request.Context(), playlistID)
	if err != nil {
		c.Error(appErrors.NewAppError(http.StatusNotFound, err, "Playlist not found"))
		return
	}

	c.JSON(http.StatusOK, playlist)
}

func (ctrl *PlaylistController) GetPlaylistProblems(c *gin.Context) {
	playlistID := c.Param("id")
	userID := c.MustGet("user_id").(string)

	problems, err := ctrl.service.GetPlaylistProblems(c.Request.Context(), playlistID, userID)
	if err != nil {
		c.Error(appErrors.NewAppError(http.StatusInternalServerError, err, "Failed to fetch playlist problems"))
		return
	}

	c.JSON(http.StatusOK, problems)
}

func (ctrl *PlaylistController) GetPlaylistAnalytics(c *gin.Context) {
	playlistID := c.Param("id")

	// Protected by RoleMiddleware("professor", "admin") in router
	analytics, err := ctrl.service.GetPlaylistAnalytics(c.Request.Context(), playlistID)
	if err != nil {
		c.Error(appErrors.NewAppError(http.StatusInternalServerError, err, "Failed to load analytics"))
		return
	}

	c.JSON(http.StatusOK, analytics)
}
