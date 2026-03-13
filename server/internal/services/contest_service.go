package services

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	redisClient "github.com/redis/go-redis/v9"

	"campuscompile/api/internal/database"
	"campuscompile/api/internal/models"
	"campuscompile/api/internal/repositories"
)

type ContestService interface {
	CreateContest(ctx context.Context, contest models.Contest, problems []map[string]interface{}) (string, error)
	FetchContests(ctx context.Context, user models.UserDemographics) ([]models.Contest, error)
	FetchContestByID(ctx context.Context, contestID string) (models.Contest, error)
	EnrollUser(ctx context.Context, contestID, userID string) error
	IsUserEnrolled(ctx context.Context, contestID, userID string) (bool, error)
	SubscribeToChannel(ctx context.Context, channel string) (<-chan *redisClient.Message, func())
	FetchCurrentLeaderboard(ctx context.Context, contestID string) ([]redisClient.Z, error)
	FetchEnrichedLeaderboard(ctx context.Context, contestID string) (string, []map[string]interface{}, error)
	FetchContestProblems(ctx context.Context, contestID, userID string) ([]map[string]interface{}, error)
	StartLeaderboardDaemon(ctx context.Context)
	UpdateContest(ctx context.Context, contestID string, contest models.Contest, problems []map[string]interface{}) error
	DeleteContest(ctx context.Context, contestID string) error
	LogTelemetry(ctx context.Context, contestID, userID string, payload models.TelemetryPayload) error
	StartAuditDaemon(ctx context.Context)
}

type contestService struct {
	repo  repositories.ContestRepository
	redis *redisClient.Client
}

func NewContestService(repo repositories.ContestRepository, redis *redisClient.Client) ContestService {
	return &contestService{repo: repo, redis: redis}
}

func (s *contestService) UpdateContest(ctx context.Context, contestID string, contest models.Contest, problems []map[string]interface{}) error {
	return s.repo.UpdateContest(ctx, contestID, contest, problems)
}

func (s *contestService) CreateContest(ctx context.Context, contest models.Contest, problems []map[string]interface{}) (string, error) {
	return s.repo.CreateContest(ctx, contest, problems)
}

func (s *contestService) FetchContests(ctx context.Context, user models.UserDemographics) ([]models.Contest, error) {
	var rawContests []models.Contest
	var err error

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

	if user.Role == "student" {
		var filtered []models.Contest
		for _, c := range rawContests {
			if c.AccessRules == nil {
				filtered = append(filtered, c)
				continue
			}
			if isEligible(c.AccessRules, user) {
				filtered = append(filtered, c)
			}
		}
		return filtered, nil
	}

	return rawContests, nil
}

func (s *contestService) FetchContestByID(ctx context.Context, contestID string) (models.Contest, error) {
	return s.repo.GetContestByID(ctx, contestID)
}

func (s *contestService) EnrollUser(ctx context.Context, contestID, userID string) error {
	err := s.repo.RegisterUser(ctx, contestID, userID)
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}

	leaderboardKey := fmt.Sprintf("contest:leaderboard:%s", contestID)

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
	return s.redis.ZRevRangeWithScores(ctx, leaderboardKey, 0, -1).Result()
}

func (s *contestService) StartLeaderboardDaemon(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second) // Check for updates every 10 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// 1. Fetch all contest IDs currently flagged as dirty using SMembers (O(N) where N is dirty count)
			dirtyContests, err := s.redis.SMembers(ctx, "dirty_contests").Result()
			if err != nil || len(dirtyContests) == 0 {
				continue
			}

			for _, contestID := range dirtyContests {
				// 2. Remove it from the set immediately to prevent duplicate broadcasts (SREM)
				s.redis.SRem(ctx, "dirty_contests", contestID)

				// 3. Fetch enriched data and broadcast
				auditStatus, leaderboardFull, err := s.FetchEnrichedLeaderboard(ctx, contestID)
				if err == nil {
					// Broadcast FULL intelligence to Faculty
					payloadFull, _ := json.Marshal(map[string]interface{}{
						"audit_status": auditStatus,
						"leaderboard":  leaderboardFull,
					})
					s.redis.Publish(ctx, fmt.Sprintf("contest:leaderboard_updates:faculty:%s", contestID), payloadFull)

					// Strip alerts and broadcast CLEAN intelligence to Students
					var leaderboardStripped []map[string]interface{}
					for _, entry := range leaderboardFull {
						strippedEntry := make(map[string]interface{})
						for k, v := range entry {
							if k != "alerts" {
								strippedEntry[k] = v
							}
						}
						leaderboardStripped = append(leaderboardStripped, strippedEntry)
					}

					payloadStripped, _ := json.Marshal(map[string]interface{}{
						"audit_status": auditStatus,
						"leaderboard":  leaderboardStripped,
					})
					s.redis.Publish(ctx, fmt.Sprintf("contest:leaderboard_updates:student:%s", contestID), payloadStripped)
				}
			}
		}
	}
}

func (s *contestService) FetchEnrichedLeaderboard(ctx context.Context, contestID string) (string, []map[string]interface{}, error) {
	leaderboardKey := fmt.Sprintf("contest:leaderboard:%s", contestID)

	zset, err := s.redis.ZRevRangeWithScores(ctx, leaderboardKey, 0, -1).Result()
	if err != nil {
		return "", nil, err
	}

	var auditStatus string
	err = database.Pool.QueryRow(ctx, "SELECT moss_audit_status FROM contests WHERE contest_id = $1", contestID).Scan(&auditStatus)
	if err != nil {
		auditStatus = "pending"
	}

	if len(zset) == 0 {
		return auditStatus, []map[string]interface{}{}, nil
	}

	var userIDs []string
	for _, z := range zset {
		userIDs = append(userIDs, z.Member.(string))
	}

	// 👇 Use the newly upgraded repository method
	profilesMap, _ := s.repo.GetUserProfiles(ctx, userIDs)
	alertsMap, _ := s.repo.GetTelemetryAlerts(ctx, contestID, userIDs)

	var enriched []map[string]interface{}
	currentRank := 1 // <-- Dynamic rank counter

	for _, z := range zset {
		userID := z.Member.(string)
		profile, exists := profilesMap[userID]

		// 👇 TASK 2.3: ISOLATE LEADERBOARD
		// Silently skip missing users or elevated faculty members
		if !exists || profile.Role == "admin" || profile.Role == "professor" {
			continue
		}

		score := z.Score
		var solves float64
		if score > 0 {
			solves = math.Ceil(score)
		} else {
			solves = 0
		}

		penalty := math.Round((solves - score) * 100000.0)
		userAlerts := alertsMap[userID]

		enriched = append(enriched, map[string]interface{}{
			"rank":     currentRank, // <-- Use the dynamic rank so there are no skipped numbers
			"user_id":  userID,
			"username": profile.Username,
			"solves":   int(solves),
			"penalty":  int(penalty),
			"alerts":   userAlerts,
		})

		currentRank++ // Increment only for valid students
	}
	return auditStatus, enriched, nil
}

func (s *contestService) FetchContestProblems(ctx context.Context, contestID, userID string) ([]map[string]interface{}, error) {
	return s.repo.GetContestProblems(ctx, contestID, userID)
}

func isEligible(rules *models.ContestAccessRules, user models.UserDemographics) bool {
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

func (s *contestService) DeleteContest(ctx context.Context, contestID string) error {
	err := s.repo.DeleteContest(ctx, contestID)

	if err == nil {
		// Nuke all Redis tracking keys associated with this contest
		s.redis.Del(ctx, fmt.Sprintf("contest:leaderboard:%s", contestID))
		s.redis.SRem(ctx, "dirty_contests", contestID) // <-- CHANGED
	}

	return err
}

func (s *contestService) LogTelemetry(ctx context.Context, contestID, userID string, payload models.TelemetryPayload) error {
	metadataBytes, _ := json.Marshal(payload.Metadata)
	return s.repo.LogTelemetry(ctx, contestID, userID, payload.EventType, metadataBytes)
}

func (s *contestService) StartAuditDaemon(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pendingIDs, err := s.repo.GetPendingAuditContests(ctx)
			if err != nil {
				continue
			}

			for _, id := range pendingIDs {
				database.Pool.Exec(ctx, "UPDATE contests SET moss_audit_status = 'in_progress' WHERE contest_id = $1", id)

				payload, _ := json.Marshal(map[string]interface{}{
					"job_type":   "moss_audit",
					"contest_id": id,
				})
				s.redis.LPush(ctx, "submission_queue", payload)
				s.redis.SAdd(ctx, "dirty_contests", id)
			}
		}
	}
}

func containsStr(slice []string, val string) bool {
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
