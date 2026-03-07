package repositories

import (
	"campuscompile/api/internal/models"
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type SubmissionRepository interface {
	CreateSubmission(ctx context.Context, submissionID, userID, problemID, language, sourceCode string) error
	GetSubmissionStatus(ctx context.Context, submissionID string) (status, language, message, sourceCode string, err error)
	GetSubmissionHistory(ctx context.Context, userID, problemID string) ([]models.SubmissionHistoryEntry, error)
}

type submissionRepo struct {
	db *pgxpool.Pool
}

func NewSubmissionRepository(db *pgxpool.Pool) SubmissionRepository {
	return &submissionRepo{db: db}
}

func (r *submissionRepo) CreateSubmission(ctx context.Context, submissionID, userID, problemID, language, sourceCode string) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO submissions (submission_id, user_id, problem_id, language, source_code, status) 
		 VALUES ($1, $2, $3, $4, $5, 'Pending')`,
		submissionID, userID, problemID, language, sourceCode)
	return err
}

func (r *submissionRepo) GetSubmissionStatus(ctx context.Context, submissionID string) (string, string, string, string, error) {
	var status, language string
	var errorLogs, sourceCode *string

	err := r.db.QueryRow(ctx, `
		SELECT status, language, error_logs, source_code 
		FROM submissions WHERE submission_id = $1
	`, submissionID).Scan(&status, &language, &errorLogs, &sourceCode)

	if err != nil {
		return "", "", "", "", err
	}

	// Safely handle NULL pointers from the database
	message := ""
	if errorLogs != nil {
		message = *errorLogs
	}

	code := ""
	if sourceCode != nil {
		code = *sourceCode
	}

	return status, language, message, code, nil
}

func (r *submissionRepo) GetSubmissionHistory(ctx context.Context, userID, problemID string) ([]models.SubmissionHistoryEntry, error) {
	rows, err := r.db.Query(ctx, `
		SELECT submission_id, language, status, submitted_at 
		FROM submissions 
		WHERE user_id = $1 AND problem_id = $2 
		ORDER BY submitted_at DESC
	`, userID, problemID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []models.SubmissionHistoryEntry
	for rows.Next() {
		var entry models.SubmissionHistoryEntry
		if err := rows.Scan(&entry.ID, &entry.Language, &entry.Status, &entry.SubmittedAt); err != nil {
			continue
		}
		history = append(history, entry)
	}

	if history == nil {
		history = []models.SubmissionHistoryEntry{}
	}

	return history, nil
}
