package handlers

import (
	"archive/zip"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"campuscompile/api/internal/database"
	"campuscompile/api/internal/services"
)

func ImportPolygonPackage(c *gin.Context) {
	// 1. Get the authenticated user ID (the Author)
	authorID := c.MustGet("user_id").(string)

	// 2. Catch the Multipart File named "package"
	file, err := c.FormFile("package")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No package file uploaded."})
		return
	}

	// 3. Create a temporary secure workspace for this extraction
	tempDir, err := os.MkdirTemp("", "polygon-import-*")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize extraction workspace."})
		return
	}
	// CRITICAL: Ensure we nuke this temp folder when the function ends
	defer os.RemoveAll(tempDir)

	// 4. Save the ZIP to the temp directory
	zipPath := filepath.Join(tempDir, file.Filename)
	if err := c.SaveUploadedFile(file, zipPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save package to disk."})
		return
	}

	// 5. Send it to the Modality C Parser Engine
	polygonData, statementText, err := services.ProcessPolygonZip(zipPath)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Invalid Polygon package or problem.xml missing."})
		return
	}

	// 6. Extract the basic metadata needed for the database
	title := "Untitled Polygon Problem"
	if len(polygonData.Names) > 0 {
		title = polygonData.Names[0].Value
	}

	slug := strings.ToLower(strings.ReplaceAll(title, " ", "-")) + "-" + uuid.New().String()[:8]

	timeLimitMs := polygonData.Judging.Testset.TimeLimit
	if timeLimitMs == 0 {
		timeLimitMs = 2000 // Fallback to 2.0s
	}

	memoryLimitKb := polygonData.Judging.Testset.MemoryLimit / 1024
	if memoryLimitKb == 0 {
		memoryLimitKb = 262144 // Fallback to 256MB
	}

	// 7. Insert the core problem into PostgreSQL
	var newProblemID string
	err = database.Pool.QueryRow(c.Request.Context(), `
		INSERT INTO problems (title, slug, description, difficulty, time_limit_ms, memory_limit_kb, author_id, has_checker)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING problem_id
	`,
		title,
		slug,
		statementText,
		"Medium",
		timeLimitMs,
		memoryLimitKb,
		authorID,
		true,
	).Scan(&newProblemID)

	if err != nil {
		fmt.Println("DB Error:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit problem to database."})
		return
	}

	// 8. Re-open the zip for the streaming phase
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read zip archive for tests."})
		return
	}
	defer r.Close()

	// 9. Stream to disk and get the file paths
	testCases, err := services.ExtractAndSaveTestCases(&r.Reader, polygonData, newProblemID)
	if err != nil || len(testCases) == 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Failed to extract test cases to disk."})
		return
	}

	// 10. Start a Database Transaction for inserting the paths
	tx, err := database.Pool.Begin(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start database transaction."})
		return
	}
	defer tx.Rollback(c.Request.Context())

	// 11. Bulk Insert the File Paths
	for _, tc := range testCases {
		_, err := tx.Exec(c.Request.Context(), `
			INSERT INTO test_cases (problem_id, test_index, is_hidden, input_file_path, output_file_path)
			VALUES ($1, $2, $3, $4, $5)
		`, newProblemID, tc.TestIndex, tc.IsHidden, tc.InputFilePath, tc.OutputFilePath)

		if err != nil {
			fmt.Println("DB Error on Test Case:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit test case metadata."})
			return
		}
	}

	// 12. Commit transaction
	if err := tx.Commit(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to finalize problem creation."})
		return
	}

	// 13. Final Success Response
	c.JSON(http.StatusOK, gin.H{
		"message":    fmt.Sprintf("Problem forged! Saved %d test cases to disk.", len(testCases)),
		"problem_id": newProblemID,
	})
}
