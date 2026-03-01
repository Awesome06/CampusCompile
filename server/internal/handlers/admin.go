package handlers

import (
	"fmt"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"campuscompile/api/internal/database"
	"campuscompile/api/internal/storage"
)

// UploadTestCase streams massive test cases directly to MinIO (S3)
// Completely replaces the old Polygon fetch-and-save architecture.
func UploadTestCase(c *gin.Context) {
	problemID := c.PostForm("problem_id")
	if problemID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "problem_id is required"})
		return
	}

	// 1. Grab the files from the multipart form request
	inputFile, err := c.FormFile("input_file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "input_file is required"})
		return
	}

	expectedFile, err := c.FormFile("expected_file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "expected_file is required"})
		return
	}

	// 2. Open the file streams (DO NOT read into memory!)
	inputSrc, _ := inputFile.Open()
	defer inputSrc.Close()

	expectedSrc, _ := expectedFile.Open()
	defer expectedSrc.Close()

	// 3. Generate unique S3 Keys
	tcUUID := uuid.New().String()
	inputS3Key := fmt.Sprintf("problems/%s/tc_%s_input.txt", problemID, tcUUID)
	expectedS3Key := fmt.Sprintf("problems/%s/tc_%s_expected.txt", problemID, tcUUID)

	// 4. Stream directly to MinIO
	ctx := c.Request.Context()

	_, err = storage.S3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(storage.BucketName),
		Key:    aws.String(inputS3Key),
		Body:   inputSrc,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload input to S3"})
		return
	}

	_, err = storage.S3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(storage.BucketName),
		Key:    aws.String(expectedS3Key),
		Body:   expectedSrc,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload expected output to S3"})
		return
	}

	// 5. Save ONLY the S3 keys to PostgreSQL
	_, err = database.Pool.Exec(ctx, `
		INSERT INTO test_cases (problem_id, input_data, expected_output, input_s3_key, expected_s3_key) 
		VALUES ($1, NULL, NULL, $2, $3)
	`, problemID, inputS3Key, expectedS3Key)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to link test case in database"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Test case securely streamed to S3 and linked to problem!",
		"input_key": inputS3Key,
	})
}
