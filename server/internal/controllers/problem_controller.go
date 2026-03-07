package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"campuscompile/api/internal/models"
	"campuscompile/api/internal/services"
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

func (ctrl *ProblemController) AddTestCasesBatch(c *gin.Context) {
	problemID := c.Param("id")

	var req models.BatchTestCasesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid test cases payload format"})
		return
	}

	if len(req.TestCases) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one test case is required"})
		return
	}

	err := ctrl.service.AddTestCases(c.Request.Context(), problemID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save test cases to database"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Test cases published successfully"})
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

	err := ctrl.service.ModifyProblem(c.Request.Context(), c.Param("id"), c.MustGet("user_id").(string), req)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Update failed or unauthorized"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Problem updated successfully"})
}

func (ctrl *ProblemController) DeleteProblem(c *gin.Context) {
	if err := ctrl.service.RemoveProblem(c.Request.Context(), c.Param("id")); err != nil {
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

func (ctrl *ProblemController) SyncTestCasesBatch(c *gin.Context) {
	var req models.BatchTestCasesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid test cases payload"})
		return
	}

	if err := ctrl.service.SyncTestCases(c.Request.Context(), c.Param("id"), req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sync test cases"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Test cases synced successfully"})
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
