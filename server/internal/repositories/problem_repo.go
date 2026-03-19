package repositories

import (
	"campuscompile/api/internal/models"
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ProblemRepository interface {
	CreateProblem(ctx context.Context, problemID, title, slug, description, difficulty string, timeLimit, memoryLimit int, authorID string, isPublic bool) error
	GetProblems(ctx context.Context, limit, offset int, searchQuery string) ([]map[string]interface{}, error)
	GetProblemByID(ctx context.Context, problemID string) (map[string]interface{}, []map[string]interface{}, error)
	GetProblemAuthor(ctx context.Context, problemID string) (string, error)
	UpdateProblem(ctx context.Context, problemID, title, description, difficulty string, timeLimit, memoryLimit int, isPublic bool) error
	DeleteProblem(ctx context.Context, problemID string) error
	GetFacultyProblems(ctx context.Context, authorID string, limit, offset int, searchQuery string) ([]map[string]interface{}, error)
	GetAllTestCases(ctx context.Context, problemID string) ([]map[string]interface{}, error)
	DeleteTestCases(ctx context.Context, problemID string) error
	InsertTestCasesBatch(ctx context.Context, problemID string, records []models.TestCaseUploadRecord) error
}

var nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9]`)

func formatPrefixTSQuery(query string) string {
	words := strings.Fields(query)
	var validWords []string
	for _, w := range words {
		cleanW := nonAlphanumericRegex.ReplaceAllString(w, "")
		if cleanW != "" {
			validWords = append(validWords, cleanW+":*")
		}
	}
	if len(validWords) > 0 {
		return strings.Join(validWords, " & ")
	}
	return ""
}

type problemRepo struct {
	db *pgxpool.Pool
}

func NewProblemRepository(db *pgxpool.Pool) ProblemRepository {
	return &problemRepo{db: db}
}

func (r *problemRepo) CreateProblem(ctx context.Context, problemID, title, slug, description, difficulty string, timeLimit, memoryLimit int, authorID string, isPublic bool) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO problems (problem_id, title, slug, description, difficulty, time_limit_ms, memory_limit_kb, author_id, is_public)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, problemID, title, slug, description, difficulty, timeLimit, memoryLimit, authorID, isPublic)
	return err
}

func (r *problemRepo) GetProblems(ctx context.Context, limit, offset int, searchQuery string) ([]map[string]interface{}, error) {
	query := `
		SELECT problem_id, title, slug, difficulty, time_limit_ms, memory_limit_kb 
		FROM problems WHERE is_public = true
	`
	args := []interface{}{}
	argIdx := 1

	tsQuery := formatPrefixTSQuery(searchQuery)
	if tsQuery != "" {
		paramStr := `$` + fmt.Sprint(argIdx)
		query += ` AND fts @@ to_tsquery('english', ` + paramStr + `)`
		query += ` ORDER BY ts_rank(fts, to_tsquery('english', ` + paramStr + `)) DESC, created_at DESC`
		args = append(args, tsQuery)
		argIdx++
	} else {
		query += ` ORDER BY created_at DESC`
	}

	query += ` LIMIT $` + fmt.Sprint(argIdx) + ` OFFSET $` + fmt.Sprint(argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var problems []map[string]interface{}
	for rows.Next() {
		var id, title, slug, difficulty string
		var timeLimit, memLimit int
		if err := rows.Scan(&id, &title, &slug, &difficulty, &timeLimit, &memLimit); err == nil {
			problems = append(problems, map[string]interface{}{
				"problem_id": id, "title": title, "slug": slug,
				"difficulty": difficulty, "time_limit_ms": timeLimit, "memory_limit_kb": memLimit,
			})
		}
	}
	return problems, nil
}

func (r *problemRepo) GetProblemByID(ctx context.Context, problemID string) (map[string]interface{}, []map[string]interface{}, error) {
	var title, description, difficulty string
	var authorID *string
	var timeLimit, memoryLimit int
	var isPublic bool

	err := r.db.QueryRow(ctx, `
		SELECT title, description, difficulty, time_limit_ms, memory_limit_kb, author_id, is_public 
		FROM problems WHERE problem_id = $1
	`, problemID).Scan(&title, &description, &difficulty, &timeLimit, &memoryLimit, &authorID, &isPublic)
	if err != nil {
		return nil, nil, err
	}

	var safeAuthorID string
	if authorID != nil {
		safeAuthorID = *authorID
	}

	problemMeta := map[string]interface{}{
		"problem_id": problemID, "title": title, "description": description,
		"difficulty": difficulty, "time_limit_ms": timeLimit, "memory_limit_kb": memoryLimit,
		"author_id": safeAuthorID, "is_public": isPublic,
	}

	rows, err := r.db.Query(ctx, `
		SELECT input_s3_key, expected_s3_key 
		FROM test_cases WHERE problem_id = $1 AND is_hidden = false
		ORDER BY input_s3_key ASC
	`, problemID)

	var samples []map[string]interface{}
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var inS3, outS3 string
			if err := rows.Scan(&inS3, &outS3); err == nil {
				samples = append(samples, map[string]interface{}{
					"input":           "", // Will be inflated by the Service layer
					"output":          "", // Will be inflated by the Service layer
					"input_s3_key":    inS3,
					"expected_s3_key": outS3,
				})
			}
		}
	}
	return problemMeta, samples, nil
}

func (r *problemRepo) GetProblemAuthor(ctx context.Context, problemID string) (string, error) {
	var authorID string
	err := r.db.QueryRow(ctx, "SELECT author_id FROM problems WHERE problem_id = $1", problemID).Scan(&authorID)
	return authorID, err
}

func (r *problemRepo) UpdateProblem(ctx context.Context, problemID, title, description, difficulty string, timeLimit, memoryLimit int, isPublic bool) error {
	_, err := r.db.Exec(ctx, `
		UPDATE problems SET title = $1, description = $2, difficulty = $3, time_limit_ms = $4, memory_limit_kb = $5, is_public = $6
		WHERE problem_id = $7
	`, title, description, difficulty, timeLimit, memoryLimit, isPublic, problemID)
	return err
}

func (r *problemRepo) DeleteProblem(ctx context.Context, problemID string) error {
	_, err := r.db.Exec(ctx, "DELETE FROM problems WHERE problem_id = $1", problemID)
	return err
}

func (r *problemRepo) GetFacultyProblems(ctx context.Context, authorID string, limit, offset int, searchQuery string) ([]map[string]interface{}, error) {
	query := `
		SELECT problem_id, title, difficulty, is_public, created_at 
		FROM problems WHERE (is_public = true OR author_id = $1)
	`
	args := []interface{}{authorID}
	argIdx := 2

	tsQuery := formatPrefixTSQuery(searchQuery)
	if tsQuery != "" {
		paramStr := `$` + fmt.Sprint(argIdx)
		query += ` AND fts @@ to_tsquery('english', ` + paramStr + `)`
		query += ` ORDER BY ts_rank(fts, to_tsquery('english', ` + paramStr + `)) DESC, created_at DESC`
		args = append(args, tsQuery)
		argIdx++
	} else {
		query += ` ORDER BY created_at DESC`
	}

	query += ` LIMIT $` + fmt.Sprint(argIdx) + ` OFFSET $` + fmt.Sprint(argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var problems []map[string]interface{}
	for rows.Next() {
		var id, title, difficulty string
		var isPublic bool
		var createdAt time.Time
		if err := rows.Scan(&id, &title, &difficulty, &isPublic, &createdAt); err == nil {
			problems = append(problems, map[string]interface{}{
				"problem_id": id, "title": title, "difficulty": difficulty,
				"is_public": isPublic, "created_at": createdAt,
			})
		}
	}
	return problems, nil
}

func (r *problemRepo) GetAllTestCases(ctx context.Context, problemID string) ([]map[string]interface{}, error) {
	rows, err := r.db.Query(ctx, "SELECT is_hidden, input_s3_key, expected_s3_key FROM test_cases WHERE problem_id = $1 ORDER BY input_s3_key ASC", problemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var testCases []map[string]interface{}
	for rows.Next() {
		var inS3, outS3 string
		var isHidden bool

		if err := rows.Scan(&isHidden, &inS3, &outS3); err == nil {
			testCases = append(testCases, map[string]interface{}{
				"is_hidden":       isHidden,
				"input_data":      "", // Will be inflated by the Service layer
				"expected_output": "", // Will be inflated by the Service layer
				"input_s3_key":    inS3,
				"expected_s3_key": outS3,
			})
		}
	}
	return testCases, nil
}

func (r *problemRepo) DeleteTestCases(ctx context.Context, problemID string) error {
	_, err := r.db.Exec(ctx, "DELETE FROM test_cases WHERE problem_id = $1", problemID)
	return err
}
func (r *problemRepo) InsertTestCasesBatch(ctx context.Context, problemID string, records []models.TestCaseUploadRecord) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, rec := range records {
		_, err = tx.Exec(ctx, `
			INSERT INTO test_cases (problem_id, is_hidden, input_s3_key, expected_s3_key) 
			VALUES ($1, $2, $3, $4)
		`, problemID, rec.IsHidden, rec.InputS3Key, rec.ExpectedS3Key)

		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}
