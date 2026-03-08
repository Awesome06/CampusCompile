package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	redisClient "github.com/redis/go-redis/v9"

	"campuscompile/api/internal/models"
	"campuscompile/api/internal/repositories"
)

type ContestService interface {
	ForgeContest(ctx context.Context, req models.Contest) (string, error)
	FetchContests(ctx context.Context) ([]models.Contest, error)
	FetchContestByID(ctx context.Context, contestID string) (models.Contest, error)
	EnrollUser(ctx context.Context, contestID, userID string) error
	IsUserEnrolled(ctx context.Context, contestID, userID string) (bool, error)
	SubscribeToChannel(ctx context.Context, channel string) (<-chan *redisClient.Message, func())
	FetchCurrentLeaderboard(ctx context.Context, contestID string) ([]redisClient.Z, error)
}

type contestService struct {
	repo  repositories.ContestRepository
	redis *redisClient.Client
}

func NewContestService(repo repositories.ContestRepository, redis *redisClient.Client) ContestService {
	return &contestService{repo: repo, redis: redis}
}

func (s *contestService) ForgeContest(ctx context.Context, req models.Contest) (string, error) {
	req.ID = uuid.New().String()

	// Normalize slug formatting for the title
	req.Title = strings.TrimSpace(req.Title)

	err := s.repo.CreateContest(ctx, req)
	if err != nil {
		return "", err
	}
	return req.ID, nil
}

func (s *contestService) FetchContests(ctx context.Context) ([]models.Contest, error) {
	return s.repo.GetContests(ctx)
}

func (s *contestService) FetchContestByID(ctx context.Context, contestID string) (models.Contest, error) {
	return s.repo.GetContestByID(ctx, contestID)
}

func (s *contestService) EnrollUser(ctx context.Context, contestID, userID string) error {
	// 1. Persist the registration in PostgreSQL
	err := s.repo.RegisterUser(ctx, contestID, userID)
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}

	// 2. Initialize the user in the Redis Leaderboard (ZSET)
	// Key format: "contest:leaderboard:{contestID}"
	leaderboardKey := fmt.Sprintf("contest:leaderboard:%s", contestID)

	// Add the user to the set with a starting score of 0 (0 solves, 0 penalty)
	// NX ensures we only add them if they aren't already in the ZSET
	err = s.redis.ZAddNX(ctx, leaderboardKey, redisClient.Z{
		Score:  0,
		Member: userID,
	}).Err()

	if err != nil {
		return fmt.Errorf("redis error initializing leaderboard: %w", err)
	}

	return nil
}

func (s *contestService) IsUserEnrolled(ctx context.Context, contestID, userID string) (bool, error) {
	return s.repo.CheckRegistration(ctx, contestID, userID)
}

func (s *contestService) SubscribeToChannel(ctx context.Context, channel string) (<-chan *redisClient.Message, func()) {
	pubsub := s.redis.Subscribe(ctx, channel)
	return pubsub.Channel(), func() { pubsub.Close() }
}

func (s *contestService) FetchCurrentLeaderboard(ctx context.Context, contestID string) ([]redisClient.Z, error) {
	leaderboardKey := fmt.Sprintf("contest:leaderboard:%s", contestID)

	// ZRevRangeWithScores fetches the sorted set from highest score to lowest
	// 0, -1 means "fetch everyone from rank 1 to the very last person"
	return s.redis.ZRevRangeWithScores(ctx, leaderboardKey, 0, -1).Result()
}
