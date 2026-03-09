package repositories

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"

	"campuscompile/api/internal/models"
)

type ContestRepository interface {
	CreateContest(ctx context.Context, contest models.Contest, problems []map[string]interface{}) (string, error)
	UpdateContest(ctx context.Context, contestID string, contest models.Contest, problems []map[string]interface{}) error
	GetContestByID(ctx context.Context, contestID string) (models.Contest, error)
	RegisterUser(ctx context.Context, contestID, userID string) error
	CheckRegistration(ctx context.Context, contestID, userID string) (bool, error)
	GetUsernames(ctx context.Context, userIDs []string) (map[string]string, error)
	GetContestProblems(ctx context.Context, contestID string) ([]map[string]interface{}, error)
	GetPublicContests(ctx context.Context) ([]models.Contest, error)
	GetFacultyContests(ctx context.Context, authorID string) ([]models.Contest, error)
	GetAllContests(ctx context.Context) ([]models.Contest, error)
	DeleteContest(ctx context.Context, contestID string) error
	LogTelemetry(ctx context.Context, contestID, userID, eventType string, metadata []byte) error
	GetTelemetryAlerts(ctx context.Context, contestID string, userIDs []string) (map[string]models.TelemetryAlerts, error)
}

type contestRepo struct {
	db *pgxpool.Pool
}

func NewContestRepository(db *pgxpool.Pool) ContestRepository {
	return &contestRepo{db: db}
}

func (r *contestRepo) CreateContest(ctx context.Context, contest models.Contest, problems []map[string]interface{}) (string, error) {
	// 1. Begin Database Transaction
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	// Defer a rollback in case anything panics or fails before the commit
	defer tx.Rollback(ctx)

	// Safely marshal the demographics JSONB rules
	var rulesJSON []byte
	if contest.AccessRules != nil {
		rulesJSON, _ = json.Marshal(contest.AccessRules)
	}

	// 2. Insert the Contest
	var contestID string
	err = tx.QueryRow(ctx, `
		INSERT INTO contests (title, host_organization, start_time, end_time, access_rules, author_id, is_public)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING contest_id
	`, contest.Title, contest.HostOrganization, contest.StartTime, contest.EndTime, rulesJSON, contest.AuthorID, contest.IsPublic).Scan(&contestID)

	if err != nil {
		return "", err
	}

	// 3. Map the Selected Problems
	for _, p := range problems {
		_, err = tx.Exec(ctx, `
			INSERT INTO contest_problems (contest_id, problem_id, points_value)
			VALUES ($1, $2, $3)
		`, contestID, p["problem_id"], p["points_value"])

		if err != nil {
			return "", err // This triggers the defer tx.Rollback()
		}
	}

	// 4. Commit the Transaction
	if err = tx.Commit(ctx); err != nil {
		return "", err
	}

	return contestID, nil
}

func (r *contestRepo) GetPublicContests(ctx context.Context) ([]models.Contest, error) {
	return r.fetchContestsWithQuery(ctx, `
		SELECT contest_id, title, host_organization, start_time, end_time, access_rules, author_id, is_public, created_at
		FROM contests WHERE is_public = true ORDER BY start_time DESC
	`)
}

func (r *contestRepo) GetFacultyContests(ctx context.Context, authorID string) ([]models.Contest, error) {
	return r.fetchContestsWithQuery(ctx, `
		SELECT contest_id, title, host_organization, start_time, end_time, access_rules, author_id, is_public, created_at
		FROM contests WHERE is_public = true OR author_id = $1 ORDER BY start_time DESC
	`, authorID)
}

func (r *contestRepo) GetAllContests(ctx context.Context) ([]models.Contest, error) {
	return r.fetchContestsWithQuery(ctx, `
		SELECT contest_id, title, host_organization, start_time, end_time, access_rules, author_id, is_public, created_at
		FROM contests ORDER BY start_time DESC
	`)
}

func (r *contestRepo) fetchContestsWithQuery(ctx context.Context, query string, args ...interface{}) ([]models.Contest, error) {
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contests []models.Contest
	for rows.Next() {
		var c models.Contest
		var rulesJSON []byte

		if err := rows.Scan(&c.ID, &c.Title, &c.HostOrganization, &c.StartTime, &c.EndTime, &rulesJSON, &c.AuthorID, &c.IsPublic, &c.CreatedAt); err == nil {
			if rulesJSON != nil {
				var rules models.ContestAccessRules
				if err := json.Unmarshal(rulesJSON, &rules); err == nil {
					c.AccessRules = &rules
				}
			}
			contests = append(contests, c)
		}
	}
	if contests == nil {
		contests = []models.Contest{}
	}
	return contests, nil
}

func (r *contestRepo) GetContestByID(ctx context.Context, contestID string) (models.Contest, error) {
	var c models.Contest
	var rulesJSON []byte

	err := r.db.QueryRow(ctx, `
		SELECT contest_id, title, host_organization, start_time, end_time, access_rules, created_at
		FROM contests WHERE contest_id = $1
	`, contestID).Scan(&c.ID, &c.Title, &c.HostOrganization, &c.StartTime, &c.EndTime, &rulesJSON, &c.CreatedAt)

	if err != nil {
		return c, err
	}

	if rulesJSON != nil {
		var rules models.ContestAccessRules
		if err := json.Unmarshal(rulesJSON, &rules); err == nil {
			c.AccessRules = &rules
		}
	}

	return c, nil
}

func (r *contestRepo) RegisterUser(ctx context.Context, contestID, userID string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO contest_registrations (contest_id, user_id)
		VALUES ($1, $2)
		ON CONFLICT (contest_id, user_id) DO NOTHING
	`, contestID, userID)
	return err
}

func (r *contestRepo) CheckRegistration(ctx context.Context, contestID, userID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM contest_registrations WHERE contest_id = $1 AND user_id = $2)
	`, contestID, userID).Scan(&exists)
	return exists, err
}

func (r *contestRepo) GetUsernames(ctx context.Context, userIDs []string) (map[string]string, error) {
	if len(userIDs) == 0 {
		return map[string]string{}, nil
	}

	// Bulk fetch using Postgres ANY() array operator
	rows, err := r.db.Query(ctx, `
		SELECT user_id, username FROM users WHERE user_id = ANY($1)
	`, userIDs)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	usernames := make(map[string]string)
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err == nil {
			usernames[id] = name
		}
	}
	return usernames, nil
}

func (r *contestRepo) GetContestProblems(ctx context.Context, contestID string) ([]map[string]interface{}, error) {
	rows, err := r.db.Query(ctx, `
		SELECT p.problem_id, p.title, p.difficulty, cp.points_value 
		FROM contest_problems cp
		JOIN problems p ON cp.problem_id = p.problem_id
		WHERE cp.contest_id = $1
		ORDER BY p.created_at ASC
	`, contestID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var problems []map[string]interface{}
	for rows.Next() {
		var id, title, difficulty string
		var points int
		if err := rows.Scan(&id, &title, &difficulty, &points); err == nil {
			problems = append(problems, map[string]interface{}{
				"problem_id":   id,
				"title":        title,
				"difficulty":   difficulty,
				"points_value": points, // Useful if you ever want custom weighting
			})
		}
	}

	if problems == nil {
		problems = []map[string]interface{}{}
	}

	return problems, nil
}

func (r *contestRepo) UpdateContest(ctx context.Context, contestID string, contest models.Contest, problems []map[string]interface{}) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var rulesJSON []byte
	if contest.AccessRules != nil {
		rulesJSON, _ = json.Marshal(contest.AccessRules)
	}

	// 1. Update the main contest record
	_, err = tx.Exec(ctx, `
		UPDATE contests 
		SET title = $1, host_organization = $2, start_time = $3, end_time = $4, access_rules = $5, is_public = $6
		WHERE contest_id = $7
	`, contest.Title, contest.HostOrganization, contest.StartTime, contest.EndTime, rulesJSON, contest.IsPublic, contestID)

	if err != nil {
		return err
	}

	// 2. Clear out the old problem mappings
	_, err = tx.Exec(ctx, `DELETE FROM contest_problems WHERE contest_id = $1`, contestID)
	if err != nil {
		return err
	}

	// 3. Insert the newly selected problem mappings
	for _, p := range problems {
		_, err = tx.Exec(ctx, `
			INSERT INTO contest_problems (contest_id, problem_id, points_value)
			VALUES ($1, $2, $3)
		`, contestID, p["problem_id"], p["points_value"])

		if err != nil {
			return err
		}
	}

	// 4. Commit the transaction
	return tx.Commit(ctx)
}

func (r *contestRepo) DeleteContest(ctx context.Context, contestID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// 1. Preserve student work: Unlink submissions from this contest
	_, err = tx.Exec(ctx, `UPDATE submissions SET contest_id = NULL WHERE contest_id = $1`, contestID)
	if err != nil {
		return err
	}

	// 2. Destroy the problem mappings
	_, err = tx.Exec(ctx, `DELETE FROM contest_problems WHERE contest_id = $1`, contestID)
	if err != nil {
		return err
	}

	// 3. Destroy the actual contest record
	_, err = tx.Exec(ctx, `DELETE FROM contests WHERE contest_id = $1`, contestID)
	if err != nil {
		return err
	}

	// Commit the transaction
	return tx.Commit(ctx)
}

func (r *contestRepo) LogTelemetry(ctx context.Context, contestID, userID, eventType string, metadata []byte) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO contest_telemetry (contest_id, user_id, event_type, metadata)
		VALUES ($1, $2, $3, $4)
	`, contestID, userID, eventType, metadata)
	return err
}

func (r *contestRepo) GetTelemetryAlerts(ctx context.Context, contestID string, userIDs []string) (map[string]models.TelemetryAlerts, error) {
	if len(userIDs) == 0 {
		return map[string]models.TelemetryAlerts{}, nil
	}

	rows, err := r.db.Query(ctx, `
		SELECT user_id,
			COUNT(*) as total,
			COUNT(CASE WHEN event_type = 'blur' THEN 1 END) as blur,
			COUNT(CASE WHEN event_type = 'paste_attempt' THEN 1 END) as paste_attempt,
			COUNT(CASE WHEN event_type = 'autotyper_suspected' THEN 1 END) as autotyper,
			COUNT(CASE WHEN event_type = 'visibility_spoof_suspected' THEN 1 END) as spoof
		FROM contest_telemetry
		WHERE contest_id = $1 AND user_id = ANY($2)
		GROUP BY user_id
	`, contestID, userIDs)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	alertsMap := make(map[string]models.TelemetryAlerts)
	for rows.Next() {
		var userID string
		var alerts models.TelemetryAlerts
		if err := rows.Scan(&userID, &alerts.Total, &alerts.Blur, &alerts.PasteAttempt, &alerts.AutotyperSuspected, &alerts.VisibilitySpoofSuspected); err == nil {
			alertsMap[userID] = alerts
		}
	}

	return alertsMap, nil
}
