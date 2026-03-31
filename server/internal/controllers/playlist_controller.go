package controllers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

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

	authorID, err := getSafeString(c, "user_id")
	if err != nil {
		c.Error(appErrors.NewAppError(http.StatusUnauthorized, err, "Malformed authentication token context"))
		return
	}

	playlistID, err := ctrl.service.CreatePlaylist(c.Request.Context(), req, authorID)
	if err != nil {
		if errors.Is(err, appErrors.ErrValidationFailed) {
			c.Error(appErrors.NewAppError(http.StatusBadRequest, err, err.Error()))
		} else {
			c.Error(appErrors.NewAppError(http.StatusInternalServerError, err, "Failed to create playlist"))
		}
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

	userID, err := getSafeString(c, "user_id")
	if err != nil {
		c.Error(appErrors.NewAppError(http.StatusUnauthorized, err, "Malformed authentication token context"))
		return
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
	if err := uuid.Validate(playlistID); err != nil {
		c.Error(appErrors.NewAppError(http.StatusBadRequest, err, "Malformed UUID format for playlist ID"))
		return
	}

	playlist, err := ctrl.service.GetPlaylistByID(c.Request.Context(), playlistID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.Error(appErrors.NewAppError(http.StatusNotFound, err, "Playlist not found"))
		} else {
			c.Error(appErrors.NewAppError(http.StatusInternalServerError, err, "Failed to fetch playlist"))
		}
		return
	}

	if !playlist.IsPublic {
		reqUserID, err := getSafeString(c, "user_id")
		if err != nil {
			c.Error(appErrors.NewAppError(http.StatusUnauthorized, err, "Malformed authentication token context"))
			return
		}
		reqUserRole, err := getSafeString(c, "role")
		if err != nil {
			c.Error(appErrors.NewAppError(http.StatusUnauthorized, err, "Malformed authentication token context"))
			return
		}

		isAuthor := playlist.AuthorID != nil && *playlist.AuthorID == reqUserID
		hasPermission := reqUserRole == "admin"

		if !isAuthor && !hasPermission {
			c.Error(appErrors.NewAppError(http.StatusForbidden, nil, "You do not have permission to view this private playlist"))
			return
		}
	}

	c.JSON(http.StatusOK, playlist)
}

func (ctrl *PlaylistController) GetPlaylistProblems(c *gin.Context) {
	playlistID := c.Param("id")
	if err := uuid.Validate(playlistID); err != nil {
		c.Error(appErrors.NewAppError(http.StatusBadRequest, err, "Malformed UUID format for playlist ID"))
		return
	}

	userID, err := getSafeString(c, "user_id")
	if err != nil {
		c.Error(appErrors.NewAppError(http.StatusUnauthorized, err, "Malformed authentication token context"))
		return
	}

	// Authenticate Visibility before fetching problems
	playlist, err := ctrl.service.GetPlaylistByID(c.Request.Context(), playlistID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.Error(appErrors.NewAppError(http.StatusNotFound, err, "Playlist not found"))
		} else {
			c.Error(appErrors.NewAppError(http.StatusInternalServerError, err, "Failed to authenticate playlist visibility"))
		}
		return
	}

	if !playlist.IsPublic {
		reqUserRole, err := getSafeString(c, "role")
		if err != nil {
			c.Error(appErrors.NewAppError(http.StatusUnauthorized, err, "Malformed authentication token context: missing role"))
			return
		}

		isAuthor := playlist.AuthorID != nil && *playlist.AuthorID == userID
		hasPermission := reqUserRole == "admin"

		if !isAuthor && !hasPermission {
			c.Error(appErrors.NewAppError(http.StatusForbidden, nil, "You do not have permission to view problems in this private playlist"))
			return
		}
	}

	problems, err := ctrl.service.GetPlaylistProblems(c.Request.Context(), playlistID, userID)
	if err != nil {
		c.Error(appErrors.NewAppError(http.StatusInternalServerError, err, "Failed to fetch playlist problems"))
		return
	}

	c.JSON(http.StatusOK, problems)
}

func (ctrl *PlaylistController) GetPlaylistAnalytics(c *gin.Context) {
	playlistID := c.Param("id")
	if err := uuid.Validate(playlistID); err != nil {
		c.Error(appErrors.NewAppError(http.StatusBadRequest, err, "Malformed UUID format for playlist ID"))
		return
	}
	
	playlist, err := ctrl.service.GetPlaylistByID(c.Request.Context(), playlistID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.Error(appErrors.NewAppError(http.StatusNotFound, err, "Playlist not found"))
		} else {
			c.Error(appErrors.NewAppError(http.StatusInternalServerError, err, "Failed to authenticate playlist visibility"))
		}
		return
	}

	userID, err := getSafeString(c, "user_id")
	if err != nil {
		c.Error(appErrors.NewAppError(http.StatusUnauthorized, err, "Malformed authentication token context: missing user_id"))
		return
	}
	userRole, err := getSafeString(c, "role")
	if err != nil {
		c.Error(appErrors.NewAppError(http.StatusUnauthorized, err, "Malformed authentication token context: missing role"))
		return
	}

	isAuthor := playlist.AuthorID != nil && *playlist.AuthorID == userID
	if !isAuthor && userRole != "admin" {
		c.Error(appErrors.NewAppError(http.StatusForbidden, nil, "You do not have permission to view analytics for this playlist"))
		return
	}

	analytics, err := ctrl.service.GetPlaylistAnalytics(c.Request.Context(), playlistID)
	if err != nil {
		c.Error(appErrors.NewAppError(http.StatusInternalServerError, err, "Failed to fetch playlist analytics"))
		return
	}

	c.JSON(http.StatusOK, analytics)
}

func (ctrl *PlaylistController) UpdatePlaylist(c *gin.Context) {
	var req models.CreatePlaylistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(appErrors.NewAppError(http.StatusBadRequest, err, "Invalid request payload"))
		return
	}

	playlistID := c.Param("id")
	if err := uuid.Validate(playlistID); err != nil {
		c.Error(appErrors.NewAppError(http.StatusBadRequest, err, "Malformed UUID format for playlist ID"))
		return
	}

	userID, err := getSafeString(c, "user_id")
	if err != nil {
		c.Error(appErrors.NewAppError(http.StatusUnauthorized, err, "Malformed authentication token context: missing user_id"))
		return
	}
	userRole, err := getSafeString(c, "role")
	if err != nil {
		c.Error(appErrors.NewAppError(http.StatusUnauthorized, err, "Malformed authentication token context: missing role"))
		return
	}

	err = ctrl.service.ModifyPlaylist(c.Request.Context(), playlistID, userID, userRole, req)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.Error(appErrors.NewAppError(http.StatusNotFound, err, "Playlist not found"))
		} else if errors.Is(err, appErrors.ErrUnauthorized) {
			c.Error(appErrors.NewAppError(http.StatusForbidden, err, err.Error()))
		} else if errors.Is(err, appErrors.ErrValidationFailed) {
			c.Error(appErrors.NewAppError(http.StatusBadRequest, err, err.Error()))
		} else {
			c.Error(appErrors.NewAppError(http.StatusInternalServerError, err, "Failed to update playlist"))
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Playlist updated successfully"})
}
