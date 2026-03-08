package services

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	redisClient "github.com/redis/go-redis/v9"

	"campuscompile/api/internal/models"
	"campuscompile/api/internal/repositories"
)

type UserDemographics struct {
	Role           string
	UserID         string
	Course         string
	Department     string
	Batch          string
	Section        string
	StudentGroup   string
	GraduationYear int
}

type ContestService interface {
	ForgeContest(ctx context.Context, req models.Contest) (string, error)
	FetchContests(ctx context.Context, user UserDemographics) ([]models.Contest, error)
	FetchContestByID(ctx context.Context, contestID string) (models.Contest, error)
	EnrollUser(ctx context.Context, contestID, userID string) error
	IsUserEnrolled(ctx context.Context, contestID, userID string) (bool, error)
	SubscribeToChannel(ctx context.Context, channel string) (<-chan *redisClient.Message, func())
	FetchCurrentLeaderboard(ctx context.Context, contestID string) ([]redisClient.Z, error)
	StartLeaderboardTicker(ctx context.Context, contestID string)
	FetchEnrichedLeaderboard(ctx context.Context, contestID string) ([]map[string]interface{}, error)
	FetchContestProblems(ctx context.Context, contestID string) ([]map[string]interface{}, error)
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

func (s *contestService) FetchContests(ctx context.Context, user UserDemographics) ([]models.Contest, error) {
	var rawContests []models.Contest
	var err error

	// 1. Fetch based on Role Visibility
	if user.Role == "admin" {
		rawContests, err = s.repo.GetAllContests(ctx)
	} else if user.Role == "professor" {
		rawContests, err = s.repo.GetFacultyContests(ctx, user.UserID)
	} else {
		rawContests, err = s.repo.GetPublicContests(ctx)
	}

	if err != nil {
		return nil, err
	}

	// 2. The Smart Filter: Students only see what they are allowed to enter
	if user.Role == "student" {
		var filtered []models.Contest
		for _, c := range rawContests {
			if c.AccessRules == nil {
				filtered = append(filtered, c) // Global contest, no restrictions
				continue
			}
			if isEligible(c.AccessRules, user) {
				filtered = append(filtered, c)
			}
		}
		return filtered, nil
	}

	// Admins and Professors see everything fetched
	return rawContests, nil
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

func (s *contestService) StartLeaderboardTicker(ctx context.Context, contestID string) {
	// Wake up every 2 seconds
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	dirtyKey := fmt.Sprintf("contest:%s:is_dirty", contestID)
	updatesChannel := fmt.Sprintf("contest:leaderboard_updates:%s", contestID)

	for {
		select {
		case <-ctx.Done():
			return // Kills the background routine if the server shuts down
		case <-ticker.C:
			// 1. Ask Redis if the Python worker flagged an update
			isDirty, err := s.redis.Get(ctx, dirtyKey).Bool()
			if err != nil || !isDirty {
				continue // Go back to sleep, nothing changed
			}

			// 2. Acknowledge and reset the flag immediately
			s.redis.Set(ctx, dirtyKey, "false", 0)

			// 3. Fetch the leaderboard, enrich it, and broadcast!
			leaderboard, err := s.FetchEnrichedLeaderboard(ctx, contestID)
			if err == nil {
				payload, _ := json.Marshal(leaderboard)
				s.redis.Publish(ctx, updatesChannel, payload)
			}
		}
	}
}

func (s *contestService) FetchEnrichedLeaderboard(ctx context.Context, contestID string) ([]map[string]interface{}, error) {
	leaderboardKey := fmt.Sprintf("contest:leaderboard:%s", contestID)

	// Fetch the ranked list from Redis
	zset, err := s.redis.ZRevRangeWithScores(ctx, leaderboardKey, 0, -1).Result()
	if err != nil {
		return nil, err
	}

	if len(zset) == 0 {
		return []map[string]interface{}{}, nil
	}

	// Extract IDs for bulk DB query
	var userIDs []string
	for _, z := range zset {
		userIDs = append(userIDs, z.Member.(string))
	}

	usernamesMap, _ := s.repo.GetUsernames(ctx, userIDs)

	var enriched []map[string]interface{}
	for i, z := range zset {
		userID := z.Member.(string)
		score := z.Score

		// Reverse-engineer the ICPC composite float
		var solves float64
		if score > 0 {
			solves = math.Ceil(score)
		} else {
			solves = 0 // Handles edge cases or students with 0 points
		}

		penalty := math.Round((solves - score) * 100000.0)

		enriched = append(enriched, map[string]interface{}{
			"rank":     i + 1,
			"user_id":  userID,
			"username": usernamesMap[userID], // Joined from Postgres
			"solves":   int(solves),
			"penalty":  int(penalty),
		})
	}
	return enriched, nil
}

func (s *contestService) FetchContestProblems(ctx context.Context, contestID string) ([]map[string]interface{}, error) {
	return s.repo.GetContestProblems(ctx, contestID)
}

func isEligible(rules *models.ContestAccessRules, user UserDemographics) bool {
	if !containsStr(rules.AllowedCourses, user.Course) {
		return false
	}
	if !containsStr(rules.AllowedDepartments, user.Department) {
		return false
	}
	if !containsStr(rules.AllowedBatches, user.Batch) {
		return false
	}
	if !containsStr(rules.AllowedSections, user.Section) {
		return false
	}
	if !containsStr(rules.AllowedStudentGroups, user.StudentGroup) {
		return false
	}
	if !containsInt(rules.AllowedGraduationYears, user.GraduationYear) {
		return false
	}
	return true
}

func containsStr(slice []string, val string) bool {
	if len(slice) == 0 {
		return true
	} // Empty array means ANY is allowed
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

func containsInt(slice []int, val int) bool {
	if len(slice) == 0 {
		return true
	}
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}
