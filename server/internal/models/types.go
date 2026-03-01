package models

import (
	"time"
)

// --- API REQUEST STRUCTS ---
type SubmitRequest struct {
	ProblemID  string `json:"problem_id" binding:"required"`
	Language   string `json:"language" binding:"required"`
	SourceCode string `json:"source_code" binding:"required"`
}

type RunRequest struct {
	Language    string `json:"language" binding:"required"`
	SourceCode  string `json:"source_code" binding:"required"`
	CustomInput string `json:"custom_input"`
}

// --- AUTH STRUCTS ---
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

// --- PROBLEM STRUCTS ---
type CreateProblemRequest struct {
	Title        string  `json:"title"`
	Description  string  `json:"description"`
	Difficulty   string  `json:"difficulty"`
	TimeLimit    float64 `json:"time_limit"`
	MemoryLimit  int     `json:"memory_limit"`
	SampleInput  string  `json:"sample_input"`
	SampleOutput string  `json:"sample_output"`
}

type Problem struct {
	ID           string  `json:"problem_id"`
	Title        string  `json:"title"`
	Slug         string  `json:"slug"`
	Description  string  `json:"description"`
	Difficulty   string  `json:"difficulty"`
	SampleInput  *string `json:"sample_input,omitempty"`
	SampleOutput *string `json:"sample_output,omitempty"`
}

// --- SUBMISSION STRUCTS ---
// OfficialSubmissionPayload updated to string to match UUID usage
type OfficialSubmissionPayload struct {
	SubmissionID string `json:"submission_id"`
}

// CustomRunPayload matches the "is_custom" logic in queue_listener.py
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
}
