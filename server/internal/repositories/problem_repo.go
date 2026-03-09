package repositories

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"campuscompile/api/internal/models"
)

type ProblemRepository interface {
	CreateProblem(ctx context.Context, problemID, title, slug, description, difficulty string, timeLimit, memoryLimit int, authorID string, isPublic bool) error
	AddTestCasesInTx(ctx context.Context, testCases []models.TestCaseToInsert) error
	GetProblems(ctx context.Context) ([]map[string]interface{}, error)
	GetProblemByID(ctx context.Context, problemID string) (map[string]interface{}, []map[string]interface{}, error)
	GetProblemAuthor(ctx context.Context, problemID string) (string, error)
	UpdateProblem(ctx context.Context, problemID, title, description, difficulty string, timeLimit, memoryLimit int, isPublic bool) error
	DeleteProblem(ctx context.Context, problemID string) error
	GetFacultyProblems(ctx context.Context, authorID string) ([]map[string]interface{}, error)
	SyncTestCasesInTx(ctx context.Context, problemID string, testCases []models.TestCaseToInsert) error
	GetAllTestCases(ctx context.Context, problemID string) ([]map[string]interface{}, error)
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

func (r *problemRepo) AddTestCasesInTx(ctx context.Context, testCases []models.TestCaseToInsert) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, tc := range testCases {
		_, err := tx.Exec(ctx, `
			INSERT INTO test_cases (test_case_id, problem_id, input_data, expected_output, is_hidden)
			VALUES ($1, $2, $3, $4, $5)
		`, tc.ID, tc.ProblemID, tc.InputData, tc.ExpectedOutput, tc.IsHidden)

		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *problemRepo) GetProblems(ctx context.Context) ([]map[string]interface{}, error) {
	rows, err := r.db.Query(ctx, `
		SELECT problem_id, title, slug, difficulty, time_limit_ms, memory_limit_kb 
		FROM problems WHERE is_public = true ORDER BY created_at ASC
	`)
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
	var authorID *string // Null-safe
	var timeLimit, memoryLimit int
	var isPublic bool // Fetch the public status

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
		"author_id": safeAuthorID, "is_public": isPublic, // Added is_public
	}

	rows, err := r.db.Query(ctx, `
		SELECT input_data, expected_output FROM test_cases WHERE problem_id = $1 AND is_hidden = false
	`, problemID)

	var samples []map[string]interface{}
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var inData, outData string
			if err := rows.Scan(&inData, &outData); err == nil {
				samples = append(samples, map[string]interface{}{"input": inData, "output": outData})
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

func (r *problemRepo) GetFacultyProblems(ctx context.Context, authorID string) ([]map[string]interface{}, error) {
	rows, err := r.db.Query(ctx, `
		SELECT problem_id, title, difficulty, is_public, created_at 
		FROM problems WHERE author_id = $1 ORDER BY created_at DESC
	`, authorID)
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

func (r *problemRepo) SyncTestCasesInTx(ctx context.Context, problemID string, testCases []models.TestCaseToInsert) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, "DELETE FROM test_cases WHERE problem_id = $1", problemID)
	if err != nil {
		return err
	}

	for _, tc := range testCases {
		_, err := tx.Exec(ctx, `
			INSERT INTO test_cases (test_case_id, problem_id, input_data, expected_output, is_hidden)
			VALUES ($1, $2, $3, $4, $5)
		`, tc.ID, tc.ProblemID, tc.InputData, tc.ExpectedOutput, tc.IsHidden)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *problemRepo) GetAllTestCases(ctx context.Context, problemID string) ([]map[string]interface{}, error) {
	rows, err := r.db.Query(ctx, "SELECT input_data, expected_output, is_hidden FROM test_cases WHERE problem_id = $1", problemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var testCases []map[string]interface{}
	for rows.Next() {
		var inData, outData string
		var isHidden bool
		if err := rows.Scan(&inData, &outData, &isHidden); err == nil {
			testCases = append(testCases, map[string]interface{}{"input_data": inData, "expected_output": outData, "is_hidden": isHidden})
		}
	}
	return testCases, nil
}
