package main

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
	"github.com/golang-jwt/jwt/v5" // ✅ FIXED: Matches the v5 import in main.go
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/microsoft"
)

var oauthConfig *oauth2.Config

// Struct to catch the data Microsoft sends back
type MicrosoftGraphUser struct {
	ID                string `json:"id"`
	DisplayName       string `json:"displayName"`
	Mail              string `json:"mail"`
	UserPrincipalName string `json:"userPrincipalName"`
}

func initOAuthConfig() {
	oauthConfig = &oauth2.Config{
		ClientID:     os.Getenv("AZURE_CLIENT_ID"),
		ClientSecret: os.Getenv("AZURE_CLIENT_SECRET"),
		// In production, change this to your actual domain (e.g., https://campuscompile.bennett.edu.in/api/auth/callback)
		RedirectURL: "http://localhost:8080/api/auth/callback",
		Scopes:      []string{"openid", "profile", "email", "User.Read"},
		Endpoint:    microsoft.AzureADEndpoint(os.Getenv("AZURE_TENANT_ID")),
	}
}

func generateStateOauthCookie(c *gin.Context) string {
	b := make([]byte, 32)
	rand.Read(b)
	state := base64.URLEncoding.EncodeToString(b)

	// Set cookie: name, value, maxAge (seconds), path, domain, secure, httpOnly
	// Note: Set 'secure' to true in production when running on HTTPS
	c.SetCookie("oauth_state", state, int(10*time.Minute.Seconds()), "/", "localhost", false, true)

	return state
}

// 1. Redirects the React frontend to the Microsoft Login Screen
func handleAzureLogin(c *gin.Context) {
	// In production, use a secure random string for the state parameter to prevent CSRF
	oauthState := generateStateOauthCookie(c)

	url := oauthConfig.AuthCodeURL(
		oauthState, // 👈 Now using the dynamic, randomized state
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("prompt", "select_account"),
	)
	c.Redirect(http.StatusTemporaryRedirect, url)
}

// 2. Catches the user coming back from Microsoft
func handleAzureCallback(c *gin.Context) {
	reqCtx := c.Request.Context() // ✅ FIXED: Use the request context for better memory management

	// 1. Grab the state Microsoft sent back in the URL
	returnedState := c.Query("state")

	// 2. Grab the original state we planted in the browser cookie
	cookieState, err := c.Cookie("oauth_state")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "OAuth state cookie missing. Please ensure cookies are enabled and try again."})
		return
	}

	// 3. If they don't match exactly, kill the request immediately
	if returnedState != cookieState {
		fmt.Printf("[SECURITY] CSRF attack thwarted! Expected state: %s, got: %s\n", cookieState, returnedState)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid OAuth state. Potential CSRF attack detected."})
		return
	}

	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Authorization code missing"})
		return
	}

	// Exchange the code for an Access Token
	token, err := oauthConfig.Exchange(reqCtx, code)
	if err != nil {
		fmt.Printf("[ERROR] Token Exchange: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to exchange token"})
		return
	}

	// Fetch the user's profile from Microsoft Graph API
	client := oauthConfig.Client(reqCtx, token)
	resp, err := client.Get("https://graph.microsoft.com/v1.0/me")
	if err != nil || resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user profile from Microsoft"})
		return
	}
	defer resp.Body.Close()

	var msUser MicrosoftGraphUser
	if err := json.NewDecoder(resp.Body).Decode(&msUser); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse Microsoft response"})
		return
	}

	// Ensure we have an email (sometimes 'mail' is null, so fallback to UserPrincipalName)
	email := msUser.Mail
	if email == "" {
		email = msUser.UserPrincipalName
	}

	emailLower := strings.ToLower(email)

	// 👇 1. AUTOMATED ROLE DETECTION (Regex Engine)
	assignedRole := "student" // Fallback default

	// Bennett student emails contain their enrollment numbers (e.g., e24cseu0343@bennett.edu.in)
	// Faculty emails are strictly alphabetical (e.g., firstname.lastname@bennett.edu.in)
	hasNumbers, _ := regexp.MatchString(`[0-9]`, emailLower)

	if !hasNumbers {
		// No numbers detected -> Grant Professor privileges
		assignedRole = "professor"
	}

	// 👇 2. Database UPSERT Logic (The "Calculate Once" Magic)
	var userID, finalRole string
	var isOnboarded bool

	// Upsert query: Creates the user with 'assignedRole' if they don't exist.
	// If they DO exist, it updates provider_id/real_name but LEAVES 'role' ALONE.
	err = dbPool.QueryRow(reqCtx, `
		INSERT INTO users (provider_id, email, real_name, role, is_onboarded)
		VALUES ($1, $2, $3, $4, false)
		ON CONFLICT (email) 
		DO UPDATE SET provider_id = EXCLUDED.provider_id, real_name = EXCLUDED.real_name
		RETURNING user_id, role::text, is_onboarded;
	`, msUser.ID, emailLower, msUser.DisplayName, assignedRole).Scan(&userID, &finalRole, &isOnboarded)

	if err != nil {
		fmt.Printf("[ERROR] DB Upsert failed: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error during login"})
		return
	}

	// 👇 3. Generate CampusCompile JWT
	// Note: We use 'finalRole' returned from the DB, NOT the 'assignedRole' calculation
	ccToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":      userID,
		"role":         finalRole,
		"is_onboarded": isOnboarded,
		"exp":          time.Now().Add(time.Hour * 72).Unix(),
	})

	tokenString, err := ccToken.SignedString(jwtSecret)
	if err != nil {
		fmt.Printf("[ERROR] JWT Signing failed: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate JWT"})
		return
	}

	// Redirect back to the React app with the data attached to the URL
	// Note: Replaced 'role' with 'finalRole' here as well
	frontendRedirectURL := fmt.Sprintf("http://localhost:5173/oauth-success?token=%s&role=%s&onboarded=%t", tokenString, finalRole, isOnboarded)
	c.Redirect(http.StatusTemporaryRedirect, frontendRedirectURL)
}

// 5. Completes the SSO Onboarding Process
func completeOnboarding(c *gin.Context) {
	// Securely grab the user_id that the requireAuth middleware attached to the context
	userID := c.MustGet("user_id").(string)

	var req OnboardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data. Please check all fields."})
		return
	}

	// Update the user's row and flip is_onboarded to true
	_, err := dbPool.Exec(c.Request.Context(), `
		UPDATE users 
		SET username = $1, 
		    course = $2, 
		    department = $3, 
		    course_year = $4, 
		    batch = $5, 
		    section = $6, 
		    student_group = $7, 
		    is_onboarded = true
		WHERE user_id = $8
	`, req.Username, req.Course, req.Department, req.CourseYear, req.Batch, req.Section, req.StudentGroup, userID)

	if err != nil {
		// PostgreSQL throws a specific error if someone tries to claim an existing username
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") || strings.Contains(err.Error(), "users_username_key") {
			c.JSON(http.StatusConflict, gin.H{"error": "That Arena Username is already taken. Please choose another."})
			return
		}

		fmt.Printf("[ERROR] Failed to save onboarding data for user %s: %v\n", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save profile data. Please try again."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Profile forged successfully!"})
}
