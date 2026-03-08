package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/microsoft"

	"campuscompile/api/internal/database"
	"campuscompile/api/internal/middleware"
	"campuscompile/api/internal/models"
)

var oauthConfig *oauth2.Config

// InitOAuthConfig is called once from main.go when the server boots
func InitOAuthConfig() {
	oauthConfig = &oauth2.Config{
		ClientID:     os.Getenv("AZURE_CLIENT_ID"),
		ClientSecret: os.Getenv("AZURE_CLIENT_SECRET"),
		RedirectURL:  "http://localhost:8080/api/auth/callback",
		Scopes:       []string{"openid", "profile", "email", "User.Read"},
		Endpoint:     microsoft.AzureADEndpoint(os.Getenv("AZURE_TENANT_ID")),
	}
}

func generateStateOauthCookie(c *gin.Context) string {
	b := make([]byte, 32)
	rand.Read(b)
	state := base64.URLEncoding.EncodeToString(b)
	c.SetCookie("oauth_state", state, int(10*time.Minute.Seconds()), "/", "localhost", false, true)
	return state
}

func HandleAzureLogin(c *gin.Context) {
	oauthState := generateStateOauthCookie(c)
	url := oauthConfig.AuthCodeURL(
		oauthState,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("prompt", "select_account"),
	)
	c.Redirect(http.StatusTemporaryRedirect, url)
}

func HandleAzureCallback(c *gin.Context) {
	reqCtx := c.Request.Context()

	// CSRF Protection
	returnedState := c.Query("state")
	cookieState, err := c.Cookie("oauth_state")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "OAuth state cookie missing."})
		return
	}
	if returnedState != cookieState {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid OAuth state. Potential CSRF attack detected."})
		return
	}

	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Authorization code missing"})
		return
	}

	token, err := oauthConfig.Exchange(reqCtx, code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to exchange token"})
		return
	}

	client := oauthConfig.Client(reqCtx, token)
	resp, err := client.Get("https://graph.microsoft.com/v1.0/me")
	if err != nil || resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user profile"})
		return
	}
	defer resp.Body.Close()

	var msUser models.MicrosoftGraphUser
	if err := json.NewDecoder(resp.Body).Decode(&msUser); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse Microsoft response"})
		return
	}

	email := msUser.Mail
	if email == "" {
		email = msUser.UserPrincipalName
	}
	emailLower := strings.ToLower(email)

	// Automated Role Detection
	assignedRole := "student"
	hasNumbers, _ := regexp.MatchString(`[0-9]`, emailLower)
	if !hasNumbers {
		assignedRole = "professor"
	}

	// Admin Set Up
	if emailLower == "mrigank.bhatnagar@bennett.edu.in" {
		assignedRole = "admin"
	}

	initialOnboarded := false
	if assignedRole == "professor" || assignedRole == "admin" {
		initialOnboarded = true // Auto-skip onboarding for faculty
	}

	// 👇 NEW: Define pointers to handle PostgreSQL NULL values safely
	var userID, finalRole string
	var isOnboarded bool
	var course, department, batch, section, studentGroup *string
	var graduationYear *int

	// 👇 NEW: Expanded RETURNING clause to fetch demographic data on login
	err = database.Pool.QueryRow(reqCtx, `
		INSERT INTO users (provider_id, email, real_name, role, is_onboarded)
		VALUES ($1, $2, $3, CAST($4 AS user_role), $5)
		ON CONFLICT (email) 
		DO UPDATE 
			SET provider_id = EXCLUDED.provider_id, 
			real_name = EXCLUDED.real_name
		RETURNING user_id, role::text, is_onboarded, course, department, graduation_year, batch, section, student_group;
	`, msUser.ID, emailLower, msUser.DisplayName, assignedRole, initialOnboarded).Scan(
		&userID, &finalRole, &isOnboarded,
		&course, &department, &graduationYear, &batch, &section, &studentGroup,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error during login"})
		return
	}

	// 👇 NEW: Dynamically build the JWT claims
	claims := jwt.MapClaims{
		"user_id":      userID,
		"role":         finalRole,
		"is_onboarded": isOnboarded,
		"exp":          time.Now().Add(time.Hour * 72).Unix(),
	}

	// Only attach demographic claims if they are NOT null (Admins/Professors will bypass this)
	if course != nil {
		claims["course"] = *course
	}
	if department != nil {
		claims["department"] = *department
	}
	if graduationYear != nil {
		claims["graduation_year"] = *graduationYear
	}
	if batch != nil {
		claims["batch"] = *batch
	}
	if section != nil {
		claims["section"] = *section
	}
	if studentGroup != nil {
		claims["student_group"] = *studentGroup
	}

	ccToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := ccToken.SignedString(middleware.JwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate JWT"})
		return
	}

	frontendRedirectURL := fmt.Sprintf("http://localhost:5173/oauth-success?token=%s&role=%s&onboarded=%t", tokenString, finalRole, isOnboarded)
	c.Redirect(http.StatusTemporaryRedirect, frontendRedirectURL)
}

func CompleteOnboarding(c *gin.Context) {
	userID := c.MustGet("user_id").(string)

	var req models.OnboardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data."})
		return
	}

	// Update the database
	_, err := database.Pool.Exec(c.Request.Context(), `
        UPDATE users 
        SET username = $1, course = $2, department = $3, graduation_year = $4, batch = $5, section = $6, student_group = $7, is_onboarded = true
        WHERE user_id = $8
    `, req.Username, req.Course, req.Department, req.GraduationYear, req.Batch, req.Section, req.StudentGroup, userID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database update failed"})
		return
	}

	// 👇 1. Generate the NEW token immediately
	// 👇 1. Generate the NEW token immediately, now packed with demographic claims
	newToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":         userID,
		"role":            "student",
		"is_onboarded":    true,
		"course":          req.Course,
		"department":      req.Department,
		"graduation_year": req.GraduationYear,
		"batch":           req.Batch,
		"section":         req.Section,
		"student_group":   req.StudentGroup,
		"exp":             time.Now().Add(time.Hour * 72).Unix(),
	})

	tokenString, err := newToken.SignedString(middleware.JwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate new session"})
		return
	}

	// 👇 2. Send ONE SINGLE JSON response containing everything
	c.JSON(http.StatusOK, gin.H{
		"message": "Profile forged successfully!",
		"token":   tokenString,
		"role":    "student",
	})
}
