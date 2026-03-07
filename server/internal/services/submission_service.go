package services

import (
	"campuscompile/api/internal/models"
	"campuscompile/api/internal/repositories"
	"context"
	"encoding/json"

	"github.com/google/uuid"
	redisClient "github.com/redis/go-redis/v9"
)

type SubmissionService interface {
	ProcessSubmission(ctx context.Context, req models.SubmitRequest, userID string) (string, error)
	ProcessRun(ctx context.Context, req models.RunRequest) (string, error)
	FetchRunStatus(ctx context.Context, runID string) (map[string]interface{}, error)
	FetchSubmissionStatus(ctx context.Context, submissionID string) (map[string]interface{}, error)
	FetchSubmissionHistory(ctx context.Context, userID, problemID string) ([]models.SubmissionHistoryEntry, error)
}

type submissionService struct {
	repo  repositories.SubmissionRepository
	redis *redisClient.Client
}

func NewSubmissionService(repo repositories.SubmissionRepository, redis *redisClient.Client) SubmissionService {
	return &submissionService{repo: repo, redis: redis}
}

func (s *submissionService) ProcessSubmission(ctx context.Context, req models.SubmitRequest, userID string) (string, error) {
	submissionID := uuid.New().String()

	// 1. Save to DB via Repo
	err := s.repo.CreateSubmission(ctx, submissionID, userID, req.ProblemID, req.Language, req.SourceCode)
	if err != nil {
		return "", err
	}

	// 2. Queue in Redis
	payload := models.OfficialSubmissionPayload{
		SubmissionID: submissionID,
	}
	jsonPayload, _ := json.Marshal(payload)

	err = s.redis.LPush(ctx, "submission_queue", jsonPayload).Err()
	if err != nil {
		return "", err // Note: Consider compensating transactions here later!
	}

	return submissionID, nil
}

func (s *submissionService) ProcessRun(ctx context.Context, req models.RunRequest) (string, error) {
	runID := uuid.New().String()

	// 1. Maintain the precise "is_custom" structure for the Python worker
	payload := models.CustomRunPayload{
		IsCustom:    true,
		RunID:       runID,
		Language:    req.Language,
		SourceCode:  req.SourceCode,
		CustomInput: req.CustomInput,
	}

	jsonPayload, _ := json.Marshal(payload)

	// 2. Queue in Redis
	err := s.redis.LPush(ctx, "submission_queue", jsonPayload).Err()
	if err != nil {
		return "", err
	}

	return runID, nil
}

func (s *submissionService) FetchRunStatus(ctx context.Context, runID string) (map[string]interface{}, error) {
	val, err := s.redis.Get(ctx, "run_result:"+runID).Result()

	if err == redisClient.Nil {
		return map[string]interface{}{"status": "Pending"}, nil
	} else if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *submissionService) FetchSubmissionStatus(ctx context.Context, submissionID string) (map[string]interface{}, error) {
	status, language, message, code, err := s.repo.GetSubmissionStatus(ctx, submissionID)
	if err != nil {
		return nil, err
	}

	// Construct the exact map the frontend expects
	return map[string]interface{}{
		"submission_id": submissionID,
		"language":      language,
		"status":        status,
		"message":       message,
		"source_code":   code,
	}, nil
}

func (s *submissionService) FetchSubmissionHistory(ctx context.Context, userID, problemID string) ([]models.SubmissionHistoryEntry, error) {
	return s.repo.GetSubmissionHistory(ctx, userID, problemID)
}
