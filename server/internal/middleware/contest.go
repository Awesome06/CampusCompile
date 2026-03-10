package middleware

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"

	"campuscompile/api/internal/database"
	"campuscompile/api/internal/models"
)

// RequireContestClearance evaluates the user's demographic tags against the contest's access rules
func RequireContestClearance() gin.HandlerFunc {
	return func(c *gin.Context) {
		contestID := c.Param("id")
		if contestID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Contest ID is required"})
			c.Abort()
			return
		}

		// 1. Let Admins and Professors bypass demographic restrictions completely
		role, _ := c.Get("role")
		if role == "admin" || role == "professor" {
			c.Next()
			return
		}

		// 2. Fetch the Contest's Access Rules from the DB
		var rawRules []byte
		err := database.Pool.QueryRow(c.Request.Context(),
			"SELECT access_rules FROM contests WHERE contest_id = $1", contestID).Scan(&rawRules)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Contest not found"})
			c.Abort()
			return
		}

		// 3. If no rules exist (NULL), it's a global/public contest. Allow access.
		if rawRules == nil {
			c.Next()
			return
		}

		// Parse the JSONB rules
		var rules models.ContestAccessRules
		if err := json.Unmarshal(rawRules, &rules); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse contest security rules"})
			c.Abort()
			return
		}

		// 4. The Evaluation Engine
		// If an array has items, the user's trait MUST be in that array. If it's empty, ANY trait is allowed.
		if !containsStr(rules.AllowedCourses, getString(c, "course")) {
			denyAccess(c, "course")
			return
		}
		if !containsStr(rules.AllowedDepartments, getString(c, "department")) {
			denyAccess(c, "department")
			return
		}
		if !containsStr(rules.AllowedBatches, getString(c, "batch")) {
			denyAccess(c, "batch")
			return
		}
		if !containsStr(rules.AllowedSections, getString(c, "section")) {
			denyAccess(c, "section")
			return
		}
		if !containsStr(rules.AllowedStudentGroups, getString(c, "student_group")) {
			denyAccess(c, "student_group")
			return
		}
		if !containsInt(rules.AllowedGraduationYears, getInt(c, "graduation_year")) {
			denyAccess(c, "graduation year")
			return
		}

		// User passed all checks!
		c.Next()
	}
}

// --- HELPER FUNCTIONS ---

func denyAccess(c *gin.Context, restrictionType string) {
	c.JSON(http.StatusForbidden, gin.H{
		"error": "Clearance Denied: Your " + restrictionType + " does not meet the requirements for this contest.",
	})
	c.Abort()
}

func containsStr(slice []string, val string) bool {
	if len(slice) == 0 {
		return true
	}
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

func containsInt(slice []int, val int) bool {
	if len(slice) == 0 {
		return true
	}
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

func getString(c *gin.Context, key string) string {
	val, _ := c.Get(key)
	str, _ := val.(string)
	return str
}

func getInt(c *gin.Context, key string) int {
	val, _ := c.Get(key)
	num, _ := val.(int)
	return num
}
