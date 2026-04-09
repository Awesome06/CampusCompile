package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"time"

	myredis "campuscompile/api/internal/redis"

	redisClient "github.com/redis/go-redis/v9"

	"campuscompile/api/internal/models"
	"campuscompile/api/internal/repositories"
)

type ContestService interface {
	CreateContest(ctx context.Context, contest models.Contest, problems []map[string]interface{}) (string, error)
	FetchContests(ctx context.Context, user models.UserDemographics, limit, offset int, searchQuery, viewMode string) ([]models.Contest, error)
	FetchContestByID(ctx context.Context, contestID string) (models.Contest, error)
	EnrollUser(ctx context.Context, contestID, userID string) error
	IsUserEnrolled(ctx context.Context, contestID, userID string) (bool, error)
	IsUserDisqualified(ctx context.Context, contestID, userID string) (bool, error)
	ProcessHeartbeat(ctx context.Context, contestID, userID string) error
	SubscribeToChannel(ctx context.Context, channel string) (<-chan string, func())
	FetchCurrentLeaderboard(ctx context.Context, contestID string) ([]redisClient.Z, error)
	FetchEnrichedLeaderboard(ctx context.Context, contestID string) (string, []map[string]interface{}, error)
	SnapshotLeaderboard(ctx context.Context, contestID string, data map[string]interface{}) error
	GetFinalizedLeaderboard(ctx context.Context, contestID string) (map[string]interface{}, error)
	FetchContestProblems(ctx context.Context, contestID, userID string) ([]map[string]interface{}, error)
	StartLeaderboardDaemon(ctx context.Context)
	UpdateContest(ctx context.Context, contestID string, contest models.Contest, problems []map[string]interface{}) error
	DeleteContest(ctx context.Context, contestID string) error
	LogTelemetry(ctx context.Context, contestID, userID string, payload models.TelemetryPayload) error
	StartAuditDaemon(ctx context.Context)
	StartHeartbeatSweeperDaemon(ctx context.Context)
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

func (s *contestService) FetchContests(ctx context.Context, user models.UserDemographics, limit, offset int, searchQuery, viewMode string) ([]models.Contest, error) {
	var rawContests []models.Contest
	var err error

	switch user.Role {
	case "admin":
		rawContests, err = s.repo.GetAllContests(ctx, limit, offset, searchQuery, viewMode)
	case "professor":
		rawContests, err = s.repo.GetFacultyContests(ctx, user.UserID, limit, offset, searchQuery, viewMode)
	default:
		rawContests, err = s.repo.GetPublicContests(ctx, &user, limit, offset, searchQuery, viewMode)
	}

	if err != nil {
		return nil, err
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

func (s *contestService) IsUserDisqualified(ctx context.Context, contestID, userID string) (bool, error) {
	return s.repo.IsUserDisqualified(ctx, contestID, userID)
}

func (s *contestService) ProcessHeartbeat(ctx context.Context, contestID, userID string) error {
	heartbeatKey := fmt.Sprintf("contest:heartbeats:%s", contestID)
	// Track user heartbeat timestamp as score in a Sorted Set.
	return s.redis.ZAdd(ctx, heartbeatKey, redisClient.Z{
		Score:  float64(time.Now().Unix()),
		Member: userID,
	}).Err()
}

func (s *contestService) SubscribeToChannel(ctx context.Context, channel string) (<-chan string, func()) {
	ch := myredis.GlobalHub.Subscribe(channel)
	cleanup := func() { myredis.GlobalHub.Unsubscribe(channel, ch) }
	return ch, cleanup
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
			slog.Info("Shutting down leaderboard daemon gracefully", "component", "StartLeaderboardDaemon")
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

	// 👇 FIXED: Routed through the repository interface
	auditStatus, err := s.repo.GetMossAuditStatus(ctx, contestID)
	if err != nil {
		auditStatus = "pending" // Safe fallback
	}

	if len(zset) == 0 {
		return auditStatus, []map[string]interface{}{}, nil
	}

	var userIDs []string
	for _, z := range zset {
		userIDs = append(userIDs, z.Member.(string))
	}

	profilesMap, err := s.repo.GetUserProfiles(ctx, userIDs)
	if err != nil {
		return auditStatus, nil, fmt.Errorf("failed to fetch user profiles for leaderboard: %w", err)
	}

	alertsMap, err := s.repo.GetTelemetryAlerts(ctx, contestID, userIDs)
	if err != nil {
		return auditStatus, nil, fmt.Errorf("failed to fetch telemetry alerts for leaderboard: %w", err)
	}

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

func (s *contestService) SnapshotLeaderboard(ctx context.Context, contestID string, data map[string]interface{}) error {
	return s.repo.SaveFinalizedLeaderboard(ctx, contestID, data)
}

func (s *contestService) GetFinalizedLeaderboard(ctx context.Context, contestID string) (map[string]interface{}, error) {
	return s.repo.GetFinalizedLeaderboard(ctx, contestID)
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
			slog.Info("Shutting down MOSS audit daemon gracefully", "component", "StartAuditDaemon")
			return
		case <-ticker.C:
			// 1. SWEEP FOR PENDING AND STUCK JOBS
			// Ensure GetPendingMossAudits utilizes the timeout SQL logic mentioned above
			pendingIDs, err := s.repo.GetPendingMossAudits(ctx)
			if err != nil || len(pendingIDs) == 0 {
				continue
			}

			for _, id := range pendingIDs {
				payload, err := json.Marshal(map[string]interface{}{
					"job_type":   "moss_audit",
					"contest_id": id,
					"queued_at":  time.Now().Unix(), // Helpful for worker metrics
				})
				if err != nil {
					slog.Error("Failed to marshal MOSS audit payload",
						"component", "StartAuditDaemon",
						"contest_id", id,
						"error", err,
					)
					continue
				}

				// 2. ATOMIC-LIKE DB CLAIM
				// Claim the job first to prevent other daemon instances from grabbing it
				if err := s.repo.UpdateMossAuditStatus(ctx, id, "in_progress"); err != nil {
					slog.Error("Failed to claim DB status for MOSS audit daemon",
						"component", "StartAuditDaemon",
						"contest_id", id,
						"error", err,
					)
					continue
				}

				// 3. PUSH TO REDIS
				if err := s.redis.LPush(ctx, "submission_queue", payload).Err(); err != nil {
					slog.Error("Failed to enqueue MOSS audit to Redis. Attempting to rollback state.",
						"component", "StartAuditDaemon",
						"contest_id", id,
						"error", err,
					)

					// 4. EXPLICIT REVERT HANDLING
					revertErr := s.repo.UpdateMossAuditStatus(ctx, id, "pending")
					if revertErr != nil {
						slog.Error("CRITICAL EXPLICIT REVERT FAILURE: DB out of sync with Redis",
							"component", "StartAuditDaemon",
							"contest_id", id,
							"error", revertErr,
						)
					}
					continue
				}

				// 5. CACHE / UI UPDATES
				cacheErr := s.redis.SAdd(ctx, "dirty_contests", id).Err()
				if cacheErr != nil {
					slog.Warn("Failed to set contest as dirty inside audit daemon cache",
						"component", "StartAuditDaemon",
						"contest_id", id,
						"error", cacheErr,
					)
				}
			}
		}
	}
}

func (s *contestService) StartHeartbeatSweeperDaemon(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second) // Check every 15 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Shutting down heartbeat sweeper gracefully", "component", "StartHeartbeatSweeperDaemon")
			return
		case <-ticker.C:
			// Fetch active contests (contests currently ongoing)
			// For simplicity, we can fetch all keys matching contest:heartbeats:*
			// This could be optimized by tracking active contests in a Redis set
			iter := s.redis.Scan(ctx, 0, "contest:heartbeats:*", 0).Iterator()
			
			cutoffTime := float64(time.Now().Unix() - 30) // 30 seconds grace period

			for iter.Next(ctx) {
				key := iter.Val()
				// Extract contestID from key (contest:heartbeats:contestID)
				contestID := key[len("contest:heartbeats:"):]
				
				// Find users whose heartbeat score is older than cutoffTime
				expiredUsers, err := s.redis.ZRangeByScore(ctx, key, &redisClient.ZRangeBy{
					Min: "-inf",
					Max: fmt.Sprintf("%f", cutoffTime),
				}).Result()

				if err == nil && len(expiredUsers) > 0 {
					for _, userID := range expiredUsers {
						// Found a disconnected user. 
						// 1. Remove from heartbeats
						s.redis.ZRem(ctx, key, userID)
						
						// 2. Disqualify them via repo
						err = s.repo.DisqualifyUser(ctx, contestID, userID)
						if err != nil {
							slog.Error("Failed to disqualify user", "contest_id", contestID, "user_id", userID, "error", err)
						} else {
							slog.Info("Disqualified user due to missed heartbeat", "contest_id", contestID, "user_id", userID)
							// Optional: Log telemetry about disqualification
						}
					}
				}
			}
		}
	}
}

