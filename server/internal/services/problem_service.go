package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"

	appErrors "campuscompile/api/internal/errors"
	"campuscompile/api/internal/models"
	"campuscompile/api/internal/repositories"
	"campuscompile/api/internal/storage"
)

type ProblemService interface {
	ForgeProblem(ctx context.Context, req models.CreateProblemRequest, authorID string) (string, error)
	FetchProblems(ctx context.Context, userID string, limit, offset int, searchQuery string) (map[string]interface{}, error)
	FetchProblemByID(ctx context.Context, problemID string) (map[string]interface{}, error)
	ModifyProblem(ctx context.Context, problemID, userID, userRole string, req models.CreateProblemRequest) error
	RemoveProblem(ctx context.Context, problemID, userID, userRole string) error
	FetchFacultyProblems(ctx context.Context, authorID string, limit, offset int, searchQuery string) (map[string]interface{}, error)
	FetchAllTestCases(ctx context.Context, problemID string) ([]map[string]interface{}, error)
	ClearTestCases(ctx context.Context, problemID, userID, userRole string) error
	SaveTestCasesBatch(ctx context.Context, problemID string, records []models.TestCaseUploadRecord) error
	GetProblemAuthor(ctx context.Context, problemID string) (string, error)
}

var ErrUnauthorizedAction = errors.New("unauthorized: only the original author or an admin can perform this action")

type problemService struct {
	repo repositories.ProblemRepository
}

func NewProblemService(repo repositories.ProblemRepository) ProblemService {
	return &problemService{repo: repo}
}

func (s *problemService) SaveTestCasesBatch(ctx context.Context, problemID string, records []models.TestCaseUploadRecord) error {
	return s.repo.InsertTestCasesBatch(ctx, problemID, records)
}

func (s *problemService) GetProblemAuthor(ctx context.Context, problemID string) (string, error) {
	return s.repo.GetProblemAuthor(ctx, problemID)
}

func (s *problemService) ForgeProblem(ctx context.Context, req models.CreateProblemRequest, authorID string) (string, error) {
	problemID := uuid.New().String()
	slug := strings.ToLower(strings.ReplaceAll(req.Title, " ", "-"))

	err := s.repo.CreateProblem(
		ctx, problemID, req.Title, slug, req.Description, req.Difficulty,
		req.TimeLimit, req.MemoryLimit, authorID, req.IsPublic, req.Tags,
	)

	if err != nil {
		return "", err
	}
	return problemID, nil
}

func (s *problemService) FetchProblems(ctx context.Context, userID string, limit, offset int, searchQuery string) (map[string]interface{}, error) {
	return s.repo.GetProblems(ctx, userID, limit, offset, searchQuery)
}

func (s *problemService) FetchProblemByID(ctx context.Context, problemID string) (map[string]interface{}, error) {
	meta, samples, err := s.repo.GetProblemByID(ctx, problemID)
	if err != nil {
		return nil, err
	}

	// Inflate public samples from S3 so they appear in the Arena description
	for _, sample := range samples {
		if key, ok := sample["input_s3_key"].(string); ok && sample["input"] == "" {
			sample["input"] = fetchS3Text(ctx, key)
		}
		if key, ok := sample["expected_s3_key"].(string); ok && sample["output"] == "" {
			sample["output"] = fetchS3Text(ctx, key)
		}
	}

	meta["samples"] = samples
	return meta, nil
}

func (s *problemService) ModifyProblem(ctx context.Context, problemID, userID, userRole string, req models.CreateProblemRequest) error {
	authorID, err := s.repo.GetProblemAuthor(ctx, problemID)
	if err != nil {
		return err
	}

	// 👇 FIX: Allow admins to bypass the author check
	if userRole != "admin" && userID != authorID {
		return fmt.Errorf("%w: only the original author or an admin can modify this problem", appErrors.ErrUnauthorized)
	}

	return s.repo.UpdateProblem(ctx, problemID, req.Title, req.Description, req.Difficulty, req.TimeLimit, req.MemoryLimit, req.IsPublic, req.Tags)
}

func (s *problemService) RemoveProblem(ctx context.Context, problemID, userID, userRole string) error {
	// 👇 NEW: Trigger the garbage collector AND check for failure
	err := s.ClearTestCases(ctx, problemID, userID, userRole)
	if err != nil {
		return fmt.Errorf("cannot delete problem: %w", err)
	}

	return s.repo.DeleteProblem(ctx, problemID)
}

func (s *problemService) FetchFacultyProblems(ctx context.Context, authorID string, limit, offset int, searchQuery string) (map[string]interface{}, error) {
	return s.repo.GetFacultyProblems(ctx, authorID, limit, offset, searchQuery)
}

func (s *problemService) FetchAllTestCases(ctx context.Context, problemID string) ([]map[string]interface{}, error) {
	testCases, err := s.repo.GetAllTestCases(ctx, problemID)
	if err != nil {
		return nil, err
	}

	// Inflate the S3 files back into raw text for the React frontend
	for _, tc := range testCases {
		if key, ok := tc["input_s3_key"].(string); ok && tc["input_data"] == "" {
			tc["input_data"] = fetchS3Text(ctx, key)
		}
		if key, ok := tc["expected_s3_key"].(string); ok && tc["expected_output"] == "" {
			tc["expected_output"] = fetchS3Text(ctx, key)
		}
	}
	return testCases, nil
}

func (s *problemService) ClearTestCases(ctx context.Context, problemID, userID, userRole string) error {
	// 1. Verify Ownership / Role
	authorID, err := s.repo.GetProblemAuthor(ctx, problemID)
	if err != nil {
		return err
	}
	if userRole != "admin" && userID != authorID {
		return ErrUnauthorizedAction // 👈 Return the Sentinel Error
	}

	// 2. Fetch the old test cases
	oldTestCases, err := s.repo.GetAllTestCases(ctx, problemID)
	if err != nil {
		return fmt.Errorf("failed to retrieve test cases for cleanup: %w", err)
	}

	// 3. Delete from MinIO/S3 and log errors, but don't stop execution
	for _, tc := range oldTestCases {
		if inKey, ok := tc["input_s3_key"].(string); ok && inKey != "" {
			_, err := storage.S3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
				Bucket: aws.String(storage.BucketName),
				Key:    aws.String(inKey),
			})
			if err != nil {
				slog.Error("Failed to delete orphaned S3 input object",
					"component", "ProblemService.ClearTestCases",
					"problem_id", problemID,
					"s3_key", inKey,
					"error", err,
				)
			}
		}

		if outKey, ok := tc["expected_s3_key"].(string); ok && outKey != "" {
			_, err := storage.S3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
				Bucket: aws.String(storage.BucketName),
				Key:    aws.String(outKey),
			})
			if err != nil {
				slog.Error("Failed to delete orphaned S3 expected output object",
					"component", "ProblemService.ClearTestCases",
					"problem_id", problemID,
					"s3_key", outKey,
					"error", err,
				)
			}
		}
	}

	// 👇 FIX (Issue 4): ALWAYS delete the DB rows even if S3 fails.
	// This prevents the DB from pointing to S3 objects that might no longer exist or are in a partial state.
	return s.repo.DeleteTestCases(ctx, problemID)
}

// Helper function to dynamically pull the text from S3
func fetchS3Text(ctx context.Context, s3Key string) string {
	if s3Key == "" {
		return ""
	}
	result, err := storage.S3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(storage.BucketName),
		Key:    aws.String(s3Key),
	})
	if err == nil {
		defer result.Body.Close()
		bytes, _ := io.ReadAll(result.Body)
		return string(bytes)
	}
	return ""
}
