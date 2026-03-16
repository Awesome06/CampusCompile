package controllers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"

	"campuscompile/api/internal/database"
	"campuscompile/api/internal/models"
	"campuscompile/api/internal/services"
	"campuscompile/api/internal/storage"
)

type ProblemController struct {
	service services.ProblemService
}

func NewProblemController(service services.ProblemService) *ProblemController {
	return &ProblemController{service: service}
}

func (ctrl *ProblemController) CreateProblem(c *gin.Context) {
	var req models.CreateProblemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	authorID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User identity not found in request context"})
		return
	}

	problemID, err := ctrl.service.ForgeProblem(c.Request.Context(), req, authorID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to forge problem in database"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Problem created successfully",
		"problem_id": problemID,
	})
}

func (ctrl *ProblemController) GetProblems(c *gin.Context) {
	problems, err := ctrl.service.FetchProblems(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database query failed"})
		return
	}
	if problems == nil {
		problems = []map[string]interface{}{}
	}
	c.JSON(http.StatusOK, problems)
}

func (ctrl *ProblemController) GetProblemByID(c *gin.Context) {
	problem, err := ctrl.service.FetchProblemByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Problem not found in the Arena."})
		return
	}
	c.JSON(http.StatusOK, problem)
}

func (ctrl *ProblemController) UpdateProblem(c *gin.Context) {
	var req models.CreateProblemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload"})
		return
	}

	userRole := c.MustGet("role").(string)
	userID := c.MustGet("user_id").(string)

	err := ctrl.service.ModifyProblem(c.Request.Context(), c.Param("id"), userID, userRole, req)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Update failed or unauthorized"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Problem updated successfully"})
}

func (ctrl *ProblemController) DeleteProblem(c *gin.Context) {
	problemID := c.Param("id")
	userRole := c.MustGet("role").(string)
	userID := c.MustGet("user_id").(string)

	// Fetch problem to verify ownership and publish status
	problemMeta, err := ctrl.service.FetchProblemByID(c.Request.Context(), problemID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Problem not found"})
		return
	}

	if userRole == "professor" {
		authorID, ok := problemMeta["author_id"].(string)
		if !ok || authorID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access Denied: You do not own this problem"})
			return
		}

		isPublic, ok := problemMeta["is_public"].(bool)
		if ok && isPublic {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access Denied: Professors cannot delete published problems. Please contact an Administrator."})
			return
		}
	}

	if err := ctrl.service.RemoveProblem(c.Request.Context(), problemID, userID, userRole); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete problem"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Problem completely erased."})
}

func (ctrl *ProblemController) GetFacultyProblems(c *gin.Context) {
	problems, err := ctrl.service.FetchFacultyProblems(c.Request.Context(), c.MustGet("user_id").(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch your problems"})
		return
	}
	if problems == nil {
		problems = []map[string]interface{}{}
	}
	c.JSON(http.StatusOK, problems)
}

func (ctrl *ProblemController) GetAllTestCasesForProblem(c *gin.Context) {
	testCases, err := ctrl.service.FetchAllTestCases(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch test cases"})
		return
	}
	if testCases == nil {
		testCases = []map[string]interface{}{}
	}
	c.JSON(http.StatusOK, gin.H{"test_cases": testCases})
}

func (ctrl *ProblemController) ClearTestCases(c *gin.Context) {
	problemID := c.Param("id")
	userID := c.MustGet("user_id").(string)
	userRole := c.MustGet("role").(string)

	err := ctrl.service.ClearTestCases(c.Request.Context(), problemID, userID, userRole)
	if err != nil {
		// 👇 NEW: Safely check for the Sentinel Error
		if errors.Is(err, services.ErrUnauthorizedAction) {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Test cases cleared successfully"})
}

func (ctrl *ProblemController) UploadTestCasesBatch(c *gin.Context) {
	problemID := c.Param("id")
	userID := c.MustGet("user_id").(string)
	userRole := c.MustGet("role").(string)

	// 👇 FIX (Issue 5): Authorization check to prevent cross-professor overwrites
	var authorID string
	err := database.Pool.QueryRow(c.Request.Context(), "SELECT author_id FROM problems WHERE problem_id = $1", problemID).Scan(&authorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify problem ownership"})
		return
	}
	if userRole != "admin" && userID != authorID {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized: only the original author or an admin can upload test cases"})
		return
	}

	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid form data"})
		return
	}

	inputFiles := form.File["input_files"]
	expectedFiles := form.File["expected_files"]
	isHiddenVals := form.Value["is_hidden"]

	if len(inputFiles) != len(expectedFiles) || len(inputFiles) != len(isHiddenVals) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mismatched file arrays in payload"})
		return
	}

	var uploadedKeys []string
	type tcRecord struct {
		inKey    string
		outKey   string
		isHidden bool
	}
	var records []tcRecord

	for i := 0; i < len(inputFiles); i++ {
		// 👇 FIX (Issue 6): Handle input file Open() error to prevent nil pointer panic
		inFile, err := inputFiles[i].Open()
		if err != nil {
			ctrl.cleanupS3Keys(c.Request.Context(), uploadedKeys)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open input file"})
			return
		}

		inKey := fmt.Sprintf("problems/%s/tc_%d_in.txt", problemID, time.Now().UnixNano())
		_, err = storage.S3Client.PutObject(c.Request.Context(), &s3.PutObjectInput{
			Bucket: aws.String(storage.BucketName),
			Key:    aws.String(inKey),
			Body:   inFile,
		})
		inFile.Close() // Safely close after successful open

		if err != nil {
			ctrl.cleanupS3Keys(c.Request.Context(), uploadedKeys)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "S3 input upload failed"})
			return
		}
		uploadedKeys = append(uploadedKeys, inKey)

		// 👇 FIX (Issue 1): Handle expected file Open() error to prevent nil pointer panic
		outFile, err := expectedFiles[i].Open()
		if err != nil {
			ctrl.cleanupS3Keys(c.Request.Context(), uploadedKeys)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open expected file"})
			return
		}

		outKey := fmt.Sprintf("problems/%s/tc_%d_out.txt", problemID, time.Now().UnixNano())
		_, err = storage.S3Client.PutObject(c.Request.Context(), &s3.PutObjectInput{
			Bucket: aws.String(storage.BucketName),
			Key:    aws.String(outKey),
			Body:   outFile,
		})
		outFile.Close() // Safely close after successful open

		if err != nil {
			ctrl.cleanupS3Keys(c.Request.Context(), uploadedKeys)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "S3 expected output upload failed"})
			return
		}
		uploadedKeys = append(uploadedKeys, outKey)

		isHidden := isHiddenVals[i] == "true"
		records = append(records, tcRecord{inKey, outKey, isHidden})
	}

	tx, err := database.Pool.Begin(c.Request.Context())
	if err != nil {
		ctrl.cleanupS3Keys(c.Request.Context(), uploadedKeys)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start database transaction"})
		return
	}
	defer tx.Rollback(c.Request.Context())

	for _, rec := range records {
		_, err = tx.Exec(c.Request.Context(), `
			INSERT INTO test_cases (problem_id, is_hidden, input_s3_key, expected_s3_key) 
			VALUES ($1, $2, $3, $4)
		`, problemID, rec.isHidden, rec.inKey, rec.outKey)

		if err != nil {
			ctrl.cleanupS3Keys(c.Request.Context(), uploadedKeys)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database insert failed"})
			return
		}
	}

	if err := tx.Commit(c.Request.Context()); err != nil {
		ctrl.cleanupS3Keys(c.Request.Context(), uploadedKeys)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Atomic batch upload successful"})
}

// Relocated cleanup function
func (ctrl *ProblemController) cleanupS3Keys(ctx context.Context, keys []string) {
	for _, key := range keys {
		storage.S3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(storage.BucketName),
			Key:    aws.String(key),
		})
	}
}
