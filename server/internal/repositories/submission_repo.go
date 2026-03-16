package repositories

import (
	"campuscompile/api/internal/models"
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SubmissionRepository interface {
	CreateSubmission(ctx context.Context, submissionID, userID, problemID, language, s3Key string, contestID *string) error
	GetSubmissionStatus(ctx context.Context, submissionID string) (status, language, message, s3Key string, err error)
	GetSubmissionHistory(ctx context.Context, userID, problemID string, contestID *string, limit, offset int) ([]models.SubmissionHistoryEntry, error)
}

type submissionRepo struct {
	db *pgxpool.Pool
}

func NewSubmissionRepository(db *pgxpool.Pool) SubmissionRepository {
	return &submissionRepo{db: db}
}

func (r *submissionRepo) CreateSubmission(ctx context.Context, submissionID, userID, problemID, language, s3Key string, contestID *string) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO submissions (submission_id, user_id, problem_id, contest_id, language, source_code_s3_key, status) 
		 VALUES ($1, $2, $3, $4, $5, $6, 'Pending')`,
		submissionID, userID, problemID, contestID, language, s3Key)
	return err
}

func (r *submissionRepo) GetSubmissionStatus(ctx context.Context, submissionID string) (string, string, string, string, error) {
	var status, language string
	var errorLogs, s3Key *string

	err := r.db.QueryRow(ctx, `
		SELECT status, language, error_logs, source_code_s3_key 
		FROM submissions WHERE submission_id = $1
	`, submissionID).Scan(&status, &language, &errorLogs, &s3Key)

	if err != nil {
		return "", "", "", "", err
	}

	message := ""
	if errorLogs != nil {
		message = *errorLogs
	}

	key := ""
	if s3Key != nil {
		key = *s3Key
	}

	return status, language, message, key, nil
}

func (r *submissionRepo) GetSubmissionHistory(ctx context.Context, userID, problemID string, contestID *string, limit, offset int) ([]models.SubmissionHistoryEntry, error) {
	var rows pgx.Rows
	var err error

	if contestID != nil && *contestID != "" {
		// 🔒 STRICT CONTEST MODE
		rows, err = r.db.Query(ctx, `
			SELECT submission_id, language, status, submitted_at, contest_id
			FROM submissions 
			WHERE user_id = $1 AND problem_id = $2 AND contest_id = $3
			ORDER BY submitted_at DESC
			LIMIT $4 OFFSET $5
		`, userID, problemID, *contestID, limit, offset)
	} else {
		// 🔓 PRACTICE MODE
		rows, err = r.db.Query(ctx, `
			SELECT s.submission_id, s.language, s.status, s.submitted_at, s.contest_id
			FROM submissions s
			LEFT JOIN contests c ON s.contest_id = c.contest_id
			WHERE s.user_id = $1 AND s.problem_id = $2 
			AND (s.contest_id IS NULL OR c.end_time <= NOW())
			ORDER BY s.submitted_at DESC
			LIMIT $3 OFFSET $4
		`, userID, problemID, limit, offset)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []models.SubmissionHistoryEntry
	for rows.Next() {
		var entry models.SubmissionHistoryEntry
		if err := rows.Scan(&entry.ID, &entry.Language, &entry.Status, &entry.SubmittedAt, &entry.ContestID); err != nil {
			continue
		}
		history = append(history, entry)
	}

	if history == nil {
		history = []models.SubmissionHistoryEntry{}
	}

	return history, nil
}
