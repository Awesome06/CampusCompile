package models

import (
	"time"
)

// ==========================================
// 1. API REQUEST STRUCTS
// ==========================================

type SubmitRequest struct {
	ProblemID  string  `json:"problem_id" binding:"required"`
	Language   string  `json:"language" binding:"required"`
	SourceCode string  `json:"source_code" binding:"required"`
	ContestID  *string `json:"contest_id,omitempty"` // Added for Phase 3 (Nullable)
}

type RunRequest struct {
	Language    string `json:"language" binding:"required"`
	SourceCode  string `json:"source_code" binding:"required"`
	CustomInput string `json:"custom_input"`
}

// ==========================================
// 2. AUTH & USER STRUCTS
// ==========================================

type MicrosoftGraphUser struct {
	ID                string `json:"id"`
	DisplayName       string `json:"displayName"`
	Mail              string `json:"mail"`
	UserPrincipalName string `json:"userPrincipalName"`
}

type OnboardRequest struct {
	Username       string `json:"username" binding:"required,max=50"`
	Course         string `json:"course" binding:"required"`
	Department     string `json:"department" binding:"required"`
	GraduationYear int    `json:"graduation_year" binding:"required"`
	Batch          string `json:"batch" binding:"required"`
	Section        string `json:"section" binding:"required"`
	StudentGroup   string `json:"student_group" binding:"required"`
}

type UserDemographics struct {
	Role           string
	UserID         string
	Course         string
	Department     string
	Batch          string
	Section        string
	StudentGroup   string
	GraduationYear int
}

type ContestProfile struct {
	Username string
	Role     string
}

// ==========================================
// 3. CONTEST STRUCTS (PHASE 3)
// ==========================================

type ContestAccessRules struct {
	AllowedCourses         []string `json:"allowed_courses,omitempty"`
	AllowedDepartments     []string `json:"allowed_departments,omitempty"`
	AllowedBatches         []string `json:"allowed_batches,omitempty"`
	AllowedSections        []string `json:"allowed_sections,omitempty"`
	AllowedStudentGroups   []string `json:"allowed_student_groups,omitempty"`
	AllowedGraduationYears []int    `json:"allowed_graduation_years,omitempty"`
}

type Contest struct {
	ID               string              `json:"contest_id"`
	Title            string              `json:"title"`
	HostOrganization string              `json:"host_organization,omitempty"`
	StartTime        time.Time           `json:"start_time"`
	EndTime          time.Time           `json:"end_time"`
	AccessRules      *ContestAccessRules `json:"access_rules,omitempty"`
	AuthorID         *string             `json:"author_id,omitempty"`
	IsPublic         bool                `json:"is_public"`
	CreatedAt        time.Time           `json:"created_at"`
	SolvedCount      int                 `json:"solved_count"`
	TotalCount       int                 `json:"total_count"`
}

type ContestRegistration struct {
	ContestID    string    `json:"contest_id"`
	UserID       string    `json:"user_id"`
	RegisteredAt time.Time `json:"registered_at"`
}

type ContestProblem struct {
	ContestID   string `json:"contest_id"`
	ProblemID   string `json:"problem_id"`
	PointsValue int    `json:"points_value"`
}

type CreateContestInput struct {
	Title            string                   `json:"title" binding:"required"`
	HostOrganization string                   `json:"host_organization"`
	StartTime        time.Time                `json:"start_time" binding:"required"`
	EndTime          time.Time                `json:"end_time" binding:"required"`
	IsPublic         bool                     `json:"is_public"`
	AccessRules      *ContestAccessRules      `json:"access_rules"`
	Problems         []map[string]interface{} `json:"problems"`
}

// ==========================================
// 4. PROBLEM & TEST CASE STRUCTS
// ==========================================

type CreateProblemRequest struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Difficulty  string   `json:"difficulty"`
	TimeLimit   int      `json:"time_limit"`
	MemoryLimit int      `json:"memory_limit"`
	IsPublic    bool     `json:"is_public"`
	Tags        []string `json:"tags,omitempty"`
}

type Problem struct {
	ID           string   `json:"problem_id"`
	Title        string   `json:"title"`
	Slug         string   `json:"slug"`
	Description  string   `json:"description"`
	Difficulty   string   `json:"difficulty"`
	SampleInput  *string  `json:"sample_input,omitempty"`
	SampleOutput *string  `json:"sample_output,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	SampleTestCases []map[string]interface{} `json:"sample_test_cases,omitempty"`
}

type TestCaseUploadRecord struct {
	InputS3Key    string
	ExpectedS3Key string
	IsHidden      bool
	IsSample      bool
}

// ==========================================
// 5. PLAYLIST STRUCTS (PHASE 4)
// ==========================================

type Playlist struct {
	ID                string    `json:"playlist_id"`
	Title             string    `json:"title"`
	Description       string    `json:"description,omitempty"`
	AuthorID          *string   `json:"author_id,omitempty"`
	IsPublic          bool      `json:"is_public"`
	OverallDifficulty string    `json:"overall_difficulty"`
	Tags              []string  `json:"tags,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type PlaylistProblem struct {
	PlaylistID       string  `json:"playlist_id"`
	ProblemID        string  `json:"problem_id"`
	OrderIndex       int     `json:"order_index"`
	CustomDifficulty *string `json:"custom_difficulty,omitempty"`
}

type PlaylistProblemInput struct {
	ProblemID        string  `json:"problem_id" binding:"required"`
	CustomDifficulty *string `json:"custom_difficulty,omitempty"`
}

type CreatePlaylistRequest struct {
	Title             string                 `json:"title" binding:"required"`
	Description       string                 `json:"description"`
	IsPublic          bool                   `json:"is_public"`
	OverallDifficulty string                 `json:"overall_difficulty" binding:"required"`
	Tags              []string               `json:"tags"`
	Problems          []PlaylistProblemInput `json:"problems" binding:"required,dive"`
}

type PlaylistResponse struct {
	Playlist
	AuthorName  string `json:"author_name"`
	SolvedCount int    `json:"solved_count"`
	TotalCount  int    `json:"total_count"`
}

type PlaylistProblemResponse struct {
	ID               string  `json:"problem_id"`
	Title            string  `json:"title"`
	Slug             string  `json:"slug"`
	Difficulty       string  `json:"difficulty"`
	OrderIndex       int     `json:"order_index"`
	CustomDifficulty *string `json:"custom_difficulty,omitempty"`
	Status           string  `json:"status"` // "AC", "Attempted", "Unattempted"
}

type PlaylistAnalyticsItem struct {
	ProblemID      string `json:"problem_id"`
	OrderIndex     int    `json:"order_index"`
	CompletedCount int    `json:"completed_count"`
	TotalStudents  int    `json:"total_students"`
}

// ==========================================
// 6. SUBMISSION & EXECUTION STRUCTS
// ==========================================

type OfficialSubmissionPayload struct {
	SubmissionID string  `json:"submission_id"`
	ContestID    *string `json:"contest_id,omitempty"`
}

type CustomRunPayload struct {
	IsCustom    bool   `json:"is_custom"`
	RunID       string `json:"run_id"`
	Language    string `json:"language"`
	SourceCode  string `json:"source_code"`
	CustomInput string `json:"custom_input"`
}

type SubmissionHistoryEntry struct {
	ID          string    `json:"submission_id"`
	Language    string    `json:"language"`
	Status      string    `json:"status"`
	SubmittedAt time.Time `json:"submitted_at"`
	ContestID   *string   `json:"contest_id,omitempty"`
}

// ==========================================
// 7. ANTI-CHEAT & TELEMETRY
// ==========================================

type TelemetryPayload struct {
	EventType string                 `json:"event_type" binding:"required"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type TelemetryAlerts struct {
	Total                    int `json:"total"`
	Blur                     int `json:"blur"`
	PasteAttempt             int `json:"paste_attempt"`
	AutotyperSuspected       int `json:"autotyper_suspected"`
	VisibilitySpoofSuspected int `json:"visibility_spoof_suspected"`
	Plagiarism               int `json:"plagiarism"`
	AnomalousRouting         int `json:"anomalous_routing"`
}

type BatchTelemetryPayload struct {
	Events []TelemetryPayload `json:"events" binding:"required"`
}
