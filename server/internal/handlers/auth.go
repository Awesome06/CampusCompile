package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/microsoft"

	"campuscompile/api/internal/database"
	"campuscompile/api/internal/middleware"
	"campuscompile/api/internal/models"
	redisPkg "campuscompile/api/internal/redis"
)

var oauthConfig *oauth2.Config

// InitOAuthConfig is called once from main.go when the server boots
func InitOAuthConfig() {
	clientID := os.Getenv("AZURE_CLIENT_ID")
	clientSecret := os.Getenv("AZURE_CLIENT_SECRET")
	tenantID := os.Getenv("AZURE_TENANT_ID")

	if clientID == "" || clientSecret == "" || tenantID == "" {
		log.Fatal("FATAL STARTUP ERROR: Azure OAuth credentials (AZURE_CLIENT_ID, AZURE_CLIENT_SECRET, AZURE_TENANT_ID) are missing")
	}

	oauthConfig = &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  os.Getenv("REDIRECT_URL"),
		Scopes:       []string{"openid", "profile", "email", "User.Read"},
		Endpoint:     microsoft.AzureADEndpoint(tenantID),
	}
}

func generateStateOauthCookie(c *gin.Context) string {
	b := make([]byte, 32)
	rand.Read(b)
	state := base64.URLEncoding.EncodeToString(b)

	c.SetSameSite(http.SameSiteLaxMode)

	domain := os.Getenv("COOKIE_DOMAIN")

	isSecure := !strings.Contains(os.Getenv("BASE_URL"), "localhost")
	c.SetCookie("oauth_state", state, 600, "/", domain, isSecure, true)
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

	initialOnboarded := false
	var autoUsername *string

	if assignedRole == "professor" || assignedRole == "admin" {
		initialOnboarded = true // Auto-skip onboarding for faculty

		// Automatically Assign the first part of their email address before @ with . replaced by whitespace
		emailPrefix := strings.Split(emailLower, "@")[0]
		spacedPrefix := strings.ReplaceAll(emailPrefix, ".", " ")
		autoUsername = &spacedPrefix
	}

	// Define pointers to handle PostgreSQL NULL values safely
	var userID, finalRole string
	var isOnboarded bool
	var course, department, batch, section, studentGroup *string
	var graduationYear *int

	// Expanded RETURNING clause to fetch demographic data on login
	err = database.Pool.QueryRow(reqCtx, `
		INSERT INTO users (provider_id, email, real_name, username, role, is_onboarded)
		VALUES ($1, $2, $3, $4, CAST($5 AS user_role), $6)
		ON CONFLICT (email) 
		DO UPDATE 
			SET provider_id = EXCLUDED.provider_id, 
			real_name = EXCLUDED.real_name,
			username = COALESCE(users.username, EXCLUDED.username)
		RETURNING user_id, role::text, is_onboarded, course, department, graduation_year, batch, section, student_group;
	`, msUser.ID, emailLower, msUser.DisplayName, autoUsername, assignedRole, initialOnboarded).Scan(
		&userID, &finalRole, &isOnboarded,
		&course, &department, &graduationYear, &batch, &section, &studentGroup,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error during login"})
		return
	}

	sessionID := uuid.New().String()

	// 👇 Centralized JWT Generation
	tokenString, err := generateUserToken(
		userID, finalRole, isOnboarded, sessionID,
		course, department, batch, section, studentGroup, graduationYear,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate JWT"})
		return
	}

	redisKey := fmt.Sprintf("active_session:%s", userID)
	err = redisPkg.Client.Set(reqCtx, redisKey, sessionID, 72*time.Hour).Err()
	if err != nil {
		// Log the error internally and fail the login
		log.Printf("[CRITICAL] Failed to register session for User %s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize active session. Please try again."})
		return
	}

	baseURL := os.Getenv("BASE_URL")
	frontendRedirectURL := fmt.Sprintf("%s/oauth-success?token=%s&role=%s&onboarded=%t", baseURL, tokenString, finalRole, isOnboarded)
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

	// 👇 Centralized JWT Generation (Passing addresses of the struct fields)
	sessionID := uuid.New().String()

	tokenString, err := generateUserToken(
		userID, "student", true, sessionID,
		&req.Course, &req.Department, &req.Batch, &req.Section, &req.StudentGroup, &req.GraduationYear,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate new session"})
		return
	}

	err = redisPkg.Client.Set(c.Request.Context(), fmt.Sprintf("active_session:%s", userID), sessionID, 72*time.Hour).Err()
	if err != nil {
		log.Printf("[CRITICAL] Failed to register elevated session for User %s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Profile saved, but failed to initialize active session. Please log in again."})
		return
	}

	// Send ONE SINGLE JSON response containing everything
	c.JSON(http.StatusOK, gin.H{
		"message": "Profile forged successfully!",
		"token":   tokenString,
		"role":    "student",
	})
}

// generateUserToken centralizes JWT creation to prevent drift between login and onboarding
func generateUserToken(userID, role string, isOnboarded bool, sessionID string, course, dept, batch, section, group *string, gradYear *int) (string, error) {
	claims := jwt.MapClaims{
		"user_id":      userID,
		"session_id":   sessionID,
		"role":         role,
		"is_onboarded": isOnboarded,
		"exp":          time.Now().Add(time.Hour * 72).Unix(),
	}

	if isOnboarded {
		if course != nil {
			claims["course"] = *course
		}
		if dept != nil {
			claims["department"] = *dept
		}
		if gradYear != nil {
			claims["graduation_year"] = *gradYear
		}
		if batch != nil {
			claims["batch"] = *batch
		}
		if section != nil {
			claims["section"] = *section
		}
		if group != nil {
			claims["student_group"] = *group
		}
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(middleware.JwtSecret)
}
