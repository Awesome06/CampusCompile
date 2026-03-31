package services

import (
	"context"
	"fmt"
	appErrors "campuscompile/api/internal/errors"

	"campuscompile/api/internal/models"
	"campuscompile/api/internal/repositories"
)

type PlaylistService interface {
	CreatePlaylist(ctx context.Context, req models.CreatePlaylistRequest, authorID string) (string, error)
	GetPlaylists(ctx context.Context, userID, viewMode string, limit, offset int) ([]models.PlaylistResponse, error)
	GetPlaylistByID(ctx context.Context, playlistID string) (models.PlaylistResponse, error)
	GetPlaylistProblems(ctx context.Context, playlistID, userID string) ([]models.PlaylistProblemResponse, error)
	GetPlaylistAnalytics(ctx context.Context, playlistID string) ([]models.PlaylistAnalyticsItem, error)
	ModifyPlaylist(ctx context.Context, playlistID, userID, userRole string, req models.CreatePlaylistRequest) error
}

type playlistService struct {
	repo repositories.PlaylistRepository
}

func NewPlaylistService(repo repositories.PlaylistRepository) PlaylistService {
	return &playlistService{repo: repo}
}

func (s *playlistService) CreatePlaylist(ctx context.Context, req models.CreatePlaylistRequest, authorID string) (string, error) {
	return s.repo.CreatePlaylist(ctx, req, authorID)
}

func (s *playlistService) GetPlaylists(ctx context.Context, userID, viewMode string, limit, offset int) ([]models.PlaylistResponse, error) {
	return s.repo.GetPlaylists(ctx, userID, viewMode, limit, offset)
}

func (s *playlistService) GetPlaylistByID(ctx context.Context, playlistID string) (models.PlaylistResponse, error) {
	return s.repo.GetPlaylistByID(ctx, playlistID)
}

func (s *playlistService) GetPlaylistProblems(ctx context.Context, playlistID, userID string) ([]models.PlaylistProblemResponse, error) {
	return s.repo.GetPlaylistProblems(ctx, playlistID, userID)
}

func (s *playlistService) GetPlaylistAnalytics(ctx context.Context, playlistID string) ([]models.PlaylistAnalyticsItem, error) {
	return s.repo.GetPlaylistAnalytics(ctx, playlistID)
}

func (s *playlistService) ModifyPlaylist(ctx context.Context, playlistID, userID, userRole string, req models.CreatePlaylistRequest) error {
	playlist, err := s.repo.GetPlaylistByID(ctx, playlistID)
	if err != nil {
		return err
	}
	
	isAuthor := playlist.AuthorID != nil && *playlist.AuthorID == userID
	if !isAuthor && userRole != "admin" {
		return fmt.Errorf("%w: you do not have permission to modify this playlist", appErrors.ErrUnauthorized)
	}

	return s.repo.UpdatePlaylist(ctx, playlistID, req)
}
