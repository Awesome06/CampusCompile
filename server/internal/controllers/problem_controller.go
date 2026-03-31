package controllers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv" // <-- Added
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	appErrors "campuscompile/api/internal/errors"
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

func getSafeString(c *gin.Context, key string) (string, error) {
	val, exists := c.Get(key)
	if !exists {
		return "", fmt.Errorf("missing key %s", key)
	}
	str, ok := val.(string)
	if !ok || str == "" {
		return "", fmt.Errorf("missing or invalid key %s", key)
	}
	return str, nil
}

func (ctrl *ProblemController) CreateProblem(c *gin.Context) {
	var req models.CreateProblemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(appErrors.NewAppError(http.StatusBadRequest, err, "Invalid request payload"))
		return
	}

	authorID, err := getSafeString(c, "user_id")
	if err != nil {
		c.Error(appErrors.NewAppError(http.StatusUnauthorized, err, "User identity not found or malformed in request context"))
		return
	}

	problemID, err := ctrl.service.ForgeProblem(c.Request.Context(), req, authorID)
	if err != nil {
		c.Error(appErrors.NewAppError(http.StatusInternalServerError, err, "Failed to forge problem in database"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Problem created successfully",
		"problem_id": problemID,
	})
}

func (ctrl *ProblemController) GetProblems(c *gin.Context) {
	limit, offset := parsePaginationArgs(c, 25)
	searchQuery := c.Query("search")

	userID, err := getSafeString(c, "user_id")
	if err != nil {
		c.Error(appErrors.NewAppError(http.StatusUnauthorized, err, "Malformed authentication token context"))
		return
	}

	result, err := ctrl.service.FetchProblems(c.Request.Context(), userID, limit, offset, searchQuery)
	if err != nil {
		c.Error(appErrors.NewAppError(http.StatusInternalServerError, err, "Database query failed"))
		return
	}
	if result == nil {
		result = map[string]interface{}{"problems": []map[string]interface{}{}, "solved_count": 0, "total_count": 0}
	}
	c.JSON(http.StatusOK, result)
}

func (ctrl *ProblemController) GetProblemByID(c *gin.Context) {
	problem, err := ctrl.service.FetchProblemByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.Error(appErrors.NewAppError(http.StatusNotFound, err, "Problem not found in the Arena."))
		return
	}
	c.JSON(http.StatusOK, problem)
}

func (ctrl *ProblemController) UpdateProblem(c *gin.Context) {
	var req models.CreateProblemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(appErrors.NewAppError(http.StatusBadRequest, err, "Invalid payload"))
		return
	}

	userRole, err := getSafeString(c, "role")
	if err != nil {
		c.Error(appErrors.NewAppError(http.StatusUnauthorized, err, "Malformed authentication token context: missing role"))
		return
	}
	userID, err := getSafeString(c, "user_id")
	if err != nil {
		c.Error(appErrors.NewAppError(http.StatusUnauthorized, err, "Malformed authentication token context: missing user_id"))
		return
	}

	err = ctrl.service.ModifyProblem(c.Request.Context(), c.Param("id"), userID, userRole, req)
	if err != nil {
		c.Error(appErrors.NewAppError(http.StatusForbidden, err, "Update failed or unauthorized"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Problem updated successfully"})
}

func (ctrl *ProblemController) DeleteProblem(c *gin.Context) {
	problemID := c.Param("id")
	userRole, err := getSafeString(c, "role")
	if err != nil {
		c.Error(appErrors.NewAppError(http.StatusUnauthorized, err, "Malformed authentication token context"))
		return
	}
	userID, err := getSafeString(c, "user_id")
	if err != nil {
		c.Error(appErrors.NewAppError(http.StatusUnauthorized, err, "Malformed authentication token context"))
		return
	}

	// Fetch problem to verify ownership and publish status
	problemMeta, err := ctrl.service.FetchProblemByID(c.Request.Context(), problemID)
	if err != nil {
		c.Error(appErrors.NewAppError(http.StatusNotFound, err, "Problem not found"))
		return
	}

	if userRole == "professor" {
		authorID, ok := problemMeta["author_id"].(string)
		if !ok || authorID != userID {
			c.Error(appErrors.NewAppError(http.StatusForbidden, nil, "Access Denied: You do not own this problem"))
			return
		}

		isPublic, ok := problemMeta["is_public"].(bool)
		if ok && isPublic {
			c.Error(appErrors.NewAppError(http.StatusForbidden, nil, "Access Denied: Professors cannot delete published problems. Please contact an Administrator."))
			return
		}
	}

	if err := ctrl.service.RemoveProblem(c.Request.Context(), problemID, userID, userRole); err != nil {
		c.Error(appErrors.NewAppError(http.StatusInternalServerError, err, "Failed to delete problem"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Problem completely erased."})
}

func (ctrl *ProblemController) GetFacultyProblems(c *gin.Context) {
	limit, offset := parsePaginationArgs(c, 25)
	searchQuery := c.Query("search")

	userID, err := getSafeString(c, "user_id")
	if err != nil {
		c.Error(appErrors.NewAppError(http.StatusUnauthorized, err, "Malformed authentication token context"))
		return
	}

	problems, err := ctrl.service.FetchFacultyProblems(c.Request.Context(), userID, limit, offset, searchQuery)
	if err != nil {
		c.Error(appErrors.NewAppError(http.StatusInternalServerError, err, "Failed to fetch your problems"))
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
		c.Error(appErrors.NewAppError(http.StatusInternalServerError, err, "Failed to fetch test cases"))
		return
	}
	if testCases == nil {
		testCases = []map[string]interface{}{}
	}
	c.JSON(http.StatusOK, gin.H{"test_cases": testCases})
}

func (ctrl *ProblemController) ClearTestCases(c *gin.Context) {
	problemID := c.Param("id")
	userID, err := getSafeString(c, "user_id")
	if err != nil {
		c.Error(appErrors.NewAppError(http.StatusUnauthorized, err, "Malformed authentication token context"))
		return
	}
	userRole, err := getSafeString(c, "role")
	if err != nil {
		c.Error(appErrors.NewAppError(http.StatusUnauthorized, err, "Malformed authentication token context"))
		return
	}

	err = ctrl.service.ClearTestCases(c.Request.Context(), problemID, userID, userRole)
	if err != nil {
		// 👇 NEW: Safely check for the Sentinel Error
		if errors.Is(err, services.ErrUnauthorizedAction) {
			c.Error(appErrors.NewAppError(http.StatusForbidden, err, "You lack permissions to clear these test cases."))
		} else {
			c.Error(appErrors.NewAppError(http.StatusInternalServerError, err, "Failed to clear test cases due to an internal system error."))
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Test cases cleared successfully"})
}

func (ctrl *ProblemController) UploadTestCasesBatch(c *gin.Context) {
	problemID := c.Param("id")
	userID, err := getSafeString(c, "user_id")
	if err != nil {
		c.Error(appErrors.NewAppError(http.StatusUnauthorized, err, "Malformed authentication token context"))
		return
	}

	var userRole string
	if roleVal, exists := c.Get("role"); exists {
		if r, ok := roleVal.(string); ok {
			userRole = r
		}
	}

	authorID, err := ctrl.service.GetProblemAuthor(c.Request.Context(), problemID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.Error(appErrors.NewAppError(http.StatusNotFound, err, "Problem not found"))
		} else {
			// Operational database issue (connectivity, timeout, etc.)
			c.Error(appErrors.NewAppError(http.StatusInternalServerError, err, "Failed to verify problem ownership"))
		}
		return
	}

	if userRole != "admin" && userID != authorID {
		c.Error(appErrors.NewAppError(http.StatusForbidden, nil, "unauthorized: only the original author or an admin can upload test cases"))
		return
	}

	// 2. Parse Multipart Form
	form, err := c.MultipartForm()
	if err != nil {
		c.Error(appErrors.NewAppError(http.StatusBadRequest, err, "Invalid form data"))
		return
	}

	inputFiles := form.File["input_files"]
	expectedFiles := form.File["expected_files"]
	isHiddenVals := form.Value["is_hidden"]

	// 3. Strict Payload Validation
	if len(inputFiles) == 0 {
		c.Error(appErrors.NewAppError(http.StatusBadRequest, nil, "No test cases provided in payload"))
		return
	}

	if len(inputFiles) != len(expectedFiles) || len(inputFiles) != len(isHiddenVals) {
		c.Error(appErrors.NewAppError(http.StatusBadRequest, nil, "Mismatched file arrays in payload"))
		return
	}

	var uploadedKeys []string
	var records []models.TestCaseUploadRecord

	// 4. Process Files and Upload to S3
	for i := 0; i < len(inputFiles); i++ {
		// Process Input File
		inFile, err := inputFiles[i].Open()
		if err != nil {
			ctrl.cleanupS3Keys(c.Request.Context(), uploadedKeys)
			c.Error(appErrors.NewAppError(http.StatusInternalServerError, err, "Failed to open input file"))
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
			c.Error(appErrors.NewAppError(http.StatusInternalServerError, err, "S3 input upload failed"))
			return
		}
		uploadedKeys = append(uploadedKeys, inKey)

		// Process Expected Output File
		outFile, err := expectedFiles[i].Open()
		if err != nil {
			ctrl.cleanupS3Keys(c.Request.Context(), uploadedKeys)
			c.Error(appErrors.NewAppError(http.StatusInternalServerError, err, "Failed to open expected file"))
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
			c.Error(appErrors.NewAppError(http.StatusInternalServerError, err, "S3 expected output upload failed"))
			return
		}
		uploadedKeys = append(uploadedKeys, outKey)

		// Parse Boolean strictly
		isHidden, err := strconv.ParseBool(isHiddenVals[i])
		if err != nil {
			ctrl.cleanupS3Keys(c.Request.Context(), uploadedKeys)
			c.Error(appErrors.NewAppError(http.StatusBadRequest, err, "Invalid boolean value for is_hidden flag"))
			return
		}

		// Map to our DTO
		records = append(records, models.TestCaseUploadRecord{
			InputS3Key:    inKey,
			ExpectedS3Key: outKey,
			IsHidden:      isHidden,
		})
	}

	// 5. Delegate Database Insertion to Service Layer
	err = ctrl.service.SaveTestCasesBatch(c.Request.Context(), problemID, records)
	if err != nil {
		ctrl.cleanupS3Keys(c.Request.Context(), uploadedKeys)
		c.Error(appErrors.NewAppError(http.StatusInternalServerError, err, "Failed to persist test cases to database"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Atomic batch upload successful"})
}

// Helper to clean up orphaned S3 objects if the transaction fails mid-flight
func (ctrl *ProblemController) cleanupS3Keys(ctx context.Context, keys []string) {
	for _, key := range keys {
		_, err := storage.S3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(storage.BucketName),
			Key:    aws.String(key),
		})
		if err != nil {
			slog.Error("Failed to clean up orphaned S3 object",
				"component", "ProblemController.cleanupS3Keys",
				"s3_key", key,
				"error", err,
			)
		}
	}
}
