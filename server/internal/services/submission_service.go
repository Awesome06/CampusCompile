package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"campuscompile/api/internal/models"
	"campuscompile/api/internal/repositories"
	"campuscompile/api/internal/storage"

	myredis "campuscompile/api/internal/redis"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	redisClient "github.com/redis/go-redis/v9"
)

type SubmissionService interface {
	ProcessSubmission(ctx context.Context, req models.SubmitRequest, userID string) (string, error)
	ProcessRun(ctx context.Context, req models.RunRequest) (string, error)
	FetchRunStatus(ctx context.Context, runID string) (map[string]interface{}, error)
	FetchSubmissionStatus(ctx context.Context, submissionID string) (map[string]interface{}, error)
	FetchSubmissionHistory(ctx context.Context, userID, problemID string, contestID *string, limit, offset int) ([]models.SubmissionHistoryEntry, error)
	SubscribeToChannel(ctx context.Context, channel string) (<-chan string, func())
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

	// 1. Generate S3 Key
	s3Key := fmt.Sprintf("submissions/%s/source.%s", submissionID, req.Language)

	// 👇 CHANGED: Use strings.NewReader instead of bytes.NewBufferString
	codeReader := strings.NewReader(req.SourceCode)

	_, err := storage.S3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(storage.BucketName),
		Key:         aws.String(s3Key),
		Body:        codeReader, // 👈 Pass the seekable reader here
		ContentType: aws.String("text/plain"),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload source code to S3: %w", err)
	}

	// 3. Save ONLY the S3 key to the database (and the Contest ID!)
	err = s.repo.CreateSubmission(ctx, submissionID, userID, req.ProblemID, req.Language, s3Key, req.ContestID)
	if err != nil {
		return "", err
	}

	// 4. Queue in Redis
	payload := models.OfficialSubmissionPayload{
		SubmissionID: submissionID,
	}
	jsonPayload, _ := json.Marshal(payload)

	err = s.redis.LPush(ctx, "submission_queue", jsonPayload).Err()
	if err != nil {
		return "", err
	}

	return submissionID, nil
}

func (s *submissionService) ProcessRun(ctx context.Context, req models.RunRequest) (string, error) {
	runID := uuid.New().String()

	// 1. For Custom Runs, we still pass the raw source code in the payload
	// because we do not save custom runs to the database or S3.
	payload := models.CustomRunPayload{
		IsCustom:    true,
		RunID:       runID,
		Language:    req.Language,
		SourceCode:  req.SourceCode,
		CustomInput: req.CustomInput,
	}

	jsonPayload, _ := json.Marshal(payload)

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
	status, language, message, s3Key, err := s.repo.GetSubmissionStatus(ctx, submissionID)
	if err != nil {
		return nil, err
	}

	// Fetch code from MinIO using the retrieved key
	var sourceCode string
	if s3Key != "" {
		result, err := storage.S3Client.GetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(storage.BucketName),
			Key:    aws.String(s3Key),
		})
		if err == nil {
			defer result.Body.Close()
			bodyBytes, err := io.ReadAll(result.Body)
			if err == nil {
				sourceCode = string(bodyBytes)
			}
		}
	}

	return map[string]interface{}{
		"submission_id": submissionID,
		"language":      language,
		"status":        status,
		"message":       message,
		"source_code":   sourceCode,
	}, nil
}

func (s *submissionService) FetchSubmissionHistory(ctx context.Context, userID, problemID string, contestID *string, limit, offset int) ([]models.SubmissionHistoryEntry, error) {
	return s.repo.GetSubmissionHistory(ctx, userID, problemID, contestID, limit, offset)
}

func (s *submissionService) SubscribeToChannel(ctx context.Context, channel string) (<-chan string, func()) {
	ch := myredis.GlobalHub.Subscribe(channel)
	cleanup := func() { myredis.GlobalHub.Unsubscribe(channel, ch) }
	return ch, cleanup
}
