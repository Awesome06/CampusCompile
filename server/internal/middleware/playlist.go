package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"campuscompile/api/internal/database"
	"campuscompile/api/internal/redis"
)

// RequirePlaylistClearance ensures only registered users can view private playlists
func RequirePlaylistClearance() gin.HandlerFunc {
	return func(c *gin.Context) {
		playlistID := c.Param("id")
		if playlistID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Playlist ID is required"})
			c.Abort()
			return
		}

		// 1. Let Admins and Professors bypass restrictions
		role, _ := c.Get("role")
		if role == "admin" || role == "professor" {
			c.Next()
			return
		}

		userID := getString(c, "user_id")
		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
			c.Abort()
			return
		}

		// 2. Check the aggressive caching layer in Redis first
		cacheKey := fmt.Sprintf("playlist:auth:%s", playlistID)
		isMember, err := redis.Client.SIsMember(c.Request.Context(), cacheKey, userID).Result()
		if err == nil && isMember {
			c.Next()
			return
		}

		// 3. Cache Miss: pull the playlist visibility natively
		var isPublic bool
		err = database.Pool.QueryRow(c.Request.Context(), "SELECT is_public FROM playlists WHERE playlist_id = $1", playlistID).Scan(&isPublic)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Playlist not found"})
			c.Abort()
			return
		}

		if isPublic {
			c.Next()
			return
		}

		// 4. Playlist is private, check registration table
		var hasRegistration bool
		err = database.Pool.QueryRow(c.Request.Context(), 
			"SELECT EXISTS(SELECT 1 FROM playlist_registrations WHERE playlist_id = $1 AND user_id = $2)", 
			playlistID, userID).Scan(&hasRegistration)
			
		if err != nil || !hasRegistration {
			c.JSON(http.StatusForbidden, gin.H{"error": "You do not have access to this private playlist"})
			c.Abort()
			return
		}

		// 5. User is valid! Populate the Redis cache for 10 minutes to reduce DB load
		pipe := redis.Client.Pipeline()
		pipe.SAdd(c.Request.Context(), cacheKey, userID)
		// Optionally check TTL if you don't want to overwrite, but pipeline with Expire is fine for rolling expiration
		pipe.Expire(c.Request.Context(), cacheKey, 10*time.Minute)
		_, _ = pipe.Exec(c.Request.Context())

		c.Next()
	}
}
