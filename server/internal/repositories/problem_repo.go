package repositories

import (
	"campuscompile/api/internal/models"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ProblemRepository interface {
	CreateProblem(ctx context.Context, problemID, title, slug, description, difficulty string, timeLimit, memoryLimit int, authorID string, isPublic bool, tags []string) error
	GetProblems(ctx context.Context, userID string, limit, offset int, searchQuery string) (map[string]interface{}, error)
	GetProblemByID(ctx context.Context, problemID string) (map[string]interface{}, []map[string]interface{}, error)
	GetProblemAuthor(ctx context.Context, problemID string) (string, error)
	UpdateProblem(ctx context.Context, problemID, title, description, difficulty string, timeLimit, memoryLimit int, isPublic bool, tags []string) error
	DeleteProblem(ctx context.Context, problemID string) error
	GetFacultyProblems(ctx context.Context, authorID string, limit, offset int, searchQuery string) ([]map[string]interface{}, error)
	GetAllTestCases(ctx context.Context, problemID string) ([]map[string]interface{}, error)
	DeleteTestCases(ctx context.Context, problemID string) error
	InsertTestCasesBatch(ctx context.Context, problemID string, records []models.TestCaseUploadRecord) error
}

type problemRepo struct {
	db *pgxpool.Pool
}

func NewProblemRepository(db *pgxpool.Pool) ProblemRepository {
	return &problemRepo{db: db}
}

func (r *problemRepo) CreateProblem(ctx context.Context, problemID, title, slug, description, difficulty string, timeLimit, memoryLimit int, authorID string, isPublic bool, tags []string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO problems (problem_id, title, slug, description, difficulty, time_limit_ms, memory_limit_kb, author_id, is_public)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, problemID, title, slug, description, difficulty, timeLimit, memoryLimit, authorID, isPublic)
	if err != nil {
		return err
	}

	tagSet := make(map[string]bool)
	for _, tag := range tags {
		tagStr := strings.TrimSpace(strings.ToLower(tag))
		if tagStr == "" || tagSet[tagStr] {
			continue
		}
		tagSet[tagStr] = true

		var tagID int
		err = tx.QueryRow(ctx, `
			INSERT INTO tags (name) VALUES ($1)
			ON CONFLICT (name) DO UPDATE SET name=EXCLUDED.name
			RETURNING tag_id
		`, tagStr).Scan(&tagID)
		if err != nil {
			return err
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO problem_tags (problem_id, tag_id) VALUES ($1, $2)
			ON CONFLICT DO NOTHING
		`, problemID, tagID)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *problemRepo) GetProblems(ctx context.Context, userID string, limit, offset int, searchQuery string) (map[string]interface{}, error) {
	// First, fetch the global counts for this user (only public problems)
	totalCount := 0
	solvedCount := 0

	countQuery := `
		SELECT 
			COUNT(DISTINCT p.problem_id) as total_count,
			COUNT(DISTINCT CASE WHEN s.status = 'AC' THEN p.problem_id END) as solved_count
		FROM problems p
		LEFT JOIN submissions s ON p.problem_id = s.problem_id AND s.user_id = $1
		WHERE p.is_public = true
	`
	err := r.db.QueryRow(ctx, countQuery, userID).Scan(&totalCount, &solvedCount)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch problem progression counts: %w", err)
	}

	// Now fetch the paginated problem list with the user's status
	query := `
		SELECT 
			p.problem_id, p.title, p.slug, p.difficulty, p.time_limit_ms, p.memory_limit_kb,
			COALESCE(
				(SELECT status::VARCHAR FROM submissions 
				 WHERE problem_id = p.problem_id AND user_id = $1 
				 ORDER BY CASE WHEN status = 'AC' THEN 1 ELSE 2 END, submitted_at DESC LIMIT 1),
				'Unattempted'
			) as user_status,
			COALESCE((
				SELECT ARRAY_AGG(t.name) 
				FROM problem_tags pt JOIN tags t ON pt.tag_id = t.tag_id 
				WHERE pt.problem_id = p.problem_id
			), ARRAY[]::VARCHAR[]) as tags
		FROM problems p WHERE p.is_public = true
	`
	args := []interface{}{userID}
	argIdx := 2

	tsQuery := formatPrefixTSQuery(searchQuery)
	if tsQuery != "" {
		paramStr := `$` + fmt.Sprint(argIdx)
		query += ` AND p.fts @@ to_tsquery('english', ` + paramStr + `)`
		query += ` ORDER BY ts_rank(p.fts, to_tsquery('english', ` + paramStr + `)) DESC, p.created_at DESC`
		args = append(args, tsQuery)
		argIdx++
	} else {
		query += ` ORDER BY p.created_at DESC`
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
		var id, title, slug, difficulty, userStatus string
		var timeLimit, memLimit int
		var tags []string
		if err := rows.Scan(&id, &title, &slug, &difficulty, &timeLimit, &memLimit, &userStatus, &tags); err == nil {
			// Normalize status to match standard
			if userStatus != "AC" && userStatus != "Unattempted" {
				userStatus = "Attempted" // Could be WA, TLE, etc.
			}
			problems = append(problems, map[string]interface{}{
				"problem_id": id, "title": title, "slug": slug,
				"difficulty": difficulty, "time_limit_ms": timeLimit, "memory_limit_kb": memLimit,
				"user_status": userStatus, "tags": tags,
			})
		}
	}
	
	if problems == nil {
		problems = []map[string]interface{}{}
	}

	return map[string]interface{}{
		"problems":     problems,
		"solved_count": solvedCount,
		"total_count":  totalCount,
	}, nil
}

func (r *problemRepo) GetProblemByID(ctx context.Context, problemID string) (map[string]interface{}, []map[string]interface{}, error) {
	var title, description, difficulty string
	var authorID *string
	var timeLimit, memoryLimit int
	var isPublic bool
	var tags []string

	err := r.db.QueryRow(ctx, `
		SELECT title, description, difficulty, time_limit_ms, memory_limit_kb, author_id, is_public,
			COALESCE((
				SELECT ARRAY_AGG(t.name) 
				FROM problem_tags pt JOIN tags t ON pt.tag_id = t.tag_id 
				WHERE pt.problem_id = p.problem_id
			), ARRAY[]::VARCHAR[]) as tags
		FROM problems p WHERE problem_id = $1
	`, problemID).Scan(&title, &description, &difficulty, &timeLimit, &memoryLimit, &authorID, &isPublic, &tags)
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
		"author_id": safeAuthorID, "is_public": isPublic, "tags": tags,
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

func (r *problemRepo) UpdateProblem(ctx context.Context, problemID, title, description, difficulty string, timeLimit, memoryLimit int, isPublic bool, tags []string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		UPDATE problems SET title = $1, description = $2, difficulty = $3, time_limit_ms = $4, memory_limit_kb = $5, is_public = $6
		WHERE problem_id = $7
	`, title, description, difficulty, timeLimit, memoryLimit, isPublic, problemID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, "DELETE FROM problem_tags WHERE problem_id = $1", problemID)
	if err != nil {
		return err
	}

	tagSet := make(map[string]bool)
	for _, tag := range tags {
		tagStr := strings.TrimSpace(strings.ToLower(tag))
		if tagStr == "" || tagSet[tagStr] {
			continue
		}
		tagSet[tagStr] = true

		var tagID int
		err = tx.QueryRow(ctx, `
			INSERT INTO tags (name) VALUES ($1)
			ON CONFLICT (name) DO UPDATE SET name=EXCLUDED.name
			RETURNING tag_id
		`, tagStr).Scan(&tagID)
		if err != nil {
			return err
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO problem_tags (problem_id, tag_id) VALUES ($1, $2)
			ON CONFLICT DO NOTHING
		`, problemID, tagID)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *problemRepo) DeleteProblem(ctx context.Context, problemID string) error {
	_, err := r.db.Exec(ctx, "DELETE FROM problems WHERE problem_id = $1", problemID)
	return err
}

func (r *problemRepo) GetFacultyProblems(ctx context.Context, authorID string, limit, offset int, searchQuery string) ([]map[string]interface{}, error) {
	query := `
		SELECT problem_id, title, difficulty, is_public, created_at,
			COALESCE((
				SELECT ARRAY_AGG(t.name) 
				FROM problem_tags pt JOIN tags t ON pt.tag_id = t.tag_id 
				WHERE pt.problem_id = p.problem_id
			), ARRAY[]::VARCHAR[]) as tags
		FROM problems p WHERE (is_public = true OR author_id = $1)
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
		var tags []string
		if err := rows.Scan(&id, &title, &difficulty, &isPublic, &createdAt, &tags); err == nil {
			problems = append(problems, map[string]interface{}{
				"problem_id": id, "title": title, "difficulty": difficulty,
				"is_public": isPublic, "created_at": createdAt, "tags": tags,
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
