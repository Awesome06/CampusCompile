package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"

	"campuscompile/api/internal/models"
	"campuscompile/api/internal/repositories"
	"campuscompile/api/internal/storage"
)

type ProblemService interface {
	ForgeProblem(ctx context.Context, req models.CreateProblemRequest, authorID string) (string, error)
	FetchProblems(ctx context.Context) ([]map[string]interface{}, error)
	FetchProblemByID(ctx context.Context, problemID string) (map[string]interface{}, error)
	ModifyProblem(ctx context.Context, problemID, userID, userRole string, req models.CreateProblemRequest) error
	RemoveProblem(ctx context.Context, problemID, userID, userRole string) error
	FetchFacultyProblems(ctx context.Context, authorID string) ([]map[string]interface{}, error)
	FetchAllTestCases(ctx context.Context, problemID string) ([]map[string]interface{}, error)
	ClearTestCases(ctx context.Context, problemID, userID, userRole string) error
}

type problemService struct {
	repo repositories.ProblemRepository
}

func NewProblemService(repo repositories.ProblemRepository) ProblemService {
	return &problemService{repo: repo}
}

func (s *problemService) ForgeProblem(ctx context.Context, req models.CreateProblemRequest, authorID string) (string, error) {
	problemID := uuid.New().String()
	slug := strings.ToLower(strings.ReplaceAll(req.Title, " ", "-"))

	err := s.repo.CreateProblem(
		ctx, problemID, req.Title, slug, req.Description, req.Difficulty,
		req.TimeLimit, req.MemoryLimit, authorID, req.IsPublic,
	)

	if err != nil {
		return "", err
	}
	return problemID, nil
}

func (s *problemService) FetchProblems(ctx context.Context) ([]map[string]interface{}, error) {
	return s.repo.GetProblems(ctx)
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
		return errors.New("unauthorized: only the original author can edit this problem")
	}

	return s.repo.UpdateProblem(ctx, problemID, req.Title, req.Description, req.Difficulty, req.TimeLimit, req.MemoryLimit, req.IsPublic)
}

func (s *problemService) RemoveProblem(ctx context.Context, problemID, userID, userRole string) error {
	// 👇 NEW: Trigger the garbage collector AND check for failure
	err := s.ClearTestCases(ctx, problemID, userID, userRole)
	if err != nil {
		return fmt.Errorf("cannot delete problem: %w", err)
	}

	return s.repo.DeleteProblem(ctx, problemID)
}

func (s *problemService) FetchFacultyProblems(ctx context.Context, authorID string) ([]map[string]interface{}, error) {
	return s.repo.GetFacultyProblems(ctx, authorID)
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
		return errors.New("unauthorized: only the original author or an admin can clear test cases")
	}

	// 2. Fetch the old test cases
	oldTestCases, err := s.repo.GetAllTestCases(ctx, problemID)
	if err != nil {
		return fmt.Errorf("failed to retrieve test cases for cleanup: %w", err)
	}

	// 3. Delete from MinIO/S3 and catch errors
	var s3Errors []error
	for _, tc := range oldTestCases {
		if inKey, ok := tc["input_s3_key"].(string); ok && inKey != "" {
			_, err := storage.S3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
				Bucket: aws.String(storage.BucketName),
				Key:    aws.String(inKey),
			})
			if err != nil {
				log.Printf("[!] S3 Deletion Error (Input): %v", err)
				s3Errors = append(s3Errors, err)
			}
		}

		if outKey, ok := tc["expected_s3_key"].(string); ok && outKey != "" {
			_, err := storage.S3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
				Bucket: aws.String(storage.BucketName),
				Key:    aws.String(outKey),
			})
			if err != nil {
				log.Printf("[!] S3 Deletion Error (Output): %v", err)
				s3Errors = append(s3Errors, err)
			}
		}
	}

	// 4. Abort DB deletion if S3 cleanup failed to prevent orphaned files
	if len(s3Errors) > 0 {
		return fmt.Errorf("failed to clean up %d S3 objects, aborting database deletion to prevent state mismatch", len(s3Errors))
	}

	// 5. Safe to wipe the rows from PostgreSQL
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
