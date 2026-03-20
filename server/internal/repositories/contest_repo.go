package repositories

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"campuscompile/api/internal/models"
)

type ContestRepository interface {
	CreateContest(ctx context.Context, contest models.Contest, problems []map[string]interface{}) (string, error)
	UpdateContest(ctx context.Context, contestID string, contest models.Contest, problems []map[string]interface{}) error
	GetContestByID(ctx context.Context, contestID string) (models.Contest, error)
	RegisterUser(ctx context.Context, contestID, userID string) error
	CheckRegistration(ctx context.Context, contestID, userID string) (bool, error)
	GetUserProfiles(ctx context.Context, userIDs []string) (map[string]models.ContestProfile, error)
	GetContestProblems(ctx context.Context, contestID, userID string) ([]map[string]interface{}, error)
	GetPublicContests(ctx context.Context, demo *models.UserDemographics, limit, offset int, searchQuery string) ([]models.Contest, error)
	GetFacultyContests(ctx context.Context, authorID string, limit, offset int, searchQuery string) ([]models.Contest, error)
	GetAllContests(ctx context.Context, limit, offset int, searchQuery string) ([]models.Contest, error)
	DeleteContest(ctx context.Context, contestID string) error
	LogTelemetry(ctx context.Context, contestID, userID, eventType string, metadata []byte) error
	GetTelemetryAlerts(ctx context.Context, contestID string, userIDs []string) (map[string]models.TelemetryAlerts, error)
	GetMossAuditStatus(ctx context.Context, contestID string) (string, error)
	GetPendingMossAudits(ctx context.Context) ([]string, error)
	UpdateMossAuditStatus(ctx context.Context, contestID, status string) error
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

func (r *contestRepo) GetPublicContests(ctx context.Context, demo *models.UserDemographics, limit, offset int, searchQuery string) ([]models.Contest, error) {
	query := `
		SELECT contest_id, title, host_organization, start_time, end_time, access_rules, author_id, is_public, created_at
		FROM contests WHERE is_public = true
	`
	args := []interface{}{}
	argIdx := 1
	
	if demo != nil {
		if demo.Course != "" {
			query += ` AND (access_rules IS NULL OR access_rules->'allowed_courses' IS NULL OR jsonb_array_length(access_rules->'allowed_courses') = 0 OR access_rules->'allowed_courses' ? $` + fmt.Sprint(argIdx) + `)`
			args = append(args, demo.Course)
			argIdx++
		} else {
			query += ` AND (access_rules IS NULL OR access_rules->'allowed_courses' IS NULL OR jsonb_array_length(access_rules->'allowed_courses') = 0)`
		}
		if demo.Department != "" {
			query += ` AND (access_rules IS NULL OR access_rules->'allowed_departments' IS NULL OR jsonb_array_length(access_rules->'allowed_departments') = 0 OR access_rules->'allowed_departments' ? $` + fmt.Sprint(argIdx) + `)`
			args = append(args, demo.Department)
			argIdx++
		} else {
			query += ` AND (access_rules IS NULL OR access_rules->'allowed_departments' IS NULL OR jsonb_array_length(access_rules->'allowed_departments') = 0)`
		}
		if demo.Batch != "" {
			query += ` AND (access_rules IS NULL OR access_rules->'allowed_batches' IS NULL OR jsonb_array_length(access_rules->'allowed_batches') = 0 OR access_rules->'allowed_batches' ? $` + fmt.Sprint(argIdx) + `)`
			args = append(args, demo.Batch)
			argIdx++
		} else {
			query += ` AND (access_rules IS NULL OR access_rules->'allowed_batches' IS NULL OR jsonb_array_length(access_rules->'allowed_batches') = 0)`
		}
		if demo.Section != "" {
			query += ` AND (access_rules IS NULL OR access_rules->'allowed_sections' IS NULL OR jsonb_array_length(access_rules->'allowed_sections') = 0 OR access_rules->'allowed_sections' ? $` + fmt.Sprint(argIdx) + `)`
			args = append(args, demo.Section)
			argIdx++
		} else {
			query += ` AND (access_rules IS NULL OR access_rules->'allowed_sections' IS NULL OR jsonb_array_length(access_rules->'allowed_sections') = 0)`
		}
		if demo.StudentGroup != "" {
			query += ` AND (access_rules IS NULL OR access_rules->'allowed_student_groups' IS NULL OR jsonb_array_length(access_rules->'allowed_student_groups') = 0 OR access_rules->'allowed_student_groups' ? $` + fmt.Sprint(argIdx) + `)`
			args = append(args, demo.StudentGroup)
			argIdx++
		} else {
			query += ` AND (access_rules IS NULL OR access_rules->'allowed_student_groups' IS NULL OR jsonb_array_length(access_rules->'allowed_student_groups') = 0)`
		}
		if demo.GraduationYear != 0 {
			query += ` AND (access_rules IS NULL OR access_rules->'allowed_graduation_years' IS NULL OR jsonb_array_length(access_rules->'allowed_graduation_years') = 0 OR access_rules->'allowed_graduation_years' @> $` + fmt.Sprint(argIdx) + `::jsonb)`
			args = append(args, fmt.Sprintf("[%d]", demo.GraduationYear))
			argIdx++
		} else {
			query += ` AND (access_rules IS NULL OR access_rules->'allowed_graduation_years' IS NULL OR jsonb_array_length(access_rules->'allowed_graduation_years') = 0)`
		}
	}

	tsQuery := formatPrefixTSQuery(searchQuery)
	if tsQuery != "" {
		paramStr := `$` + fmt.Sprint(argIdx)
		query += ` AND fts @@ to_tsquery('english', ` + paramStr + `)`
		query += ` ORDER BY ts_rank(fts, to_tsquery('english', ` + paramStr + `)) DESC, start_time DESC`
		args = append(args, tsQuery)
		argIdx++
	} else {
		query += ` ORDER BY start_time DESC`
	}
	query += ` LIMIT $` + fmt.Sprint(argIdx) + ` OFFSET $` + fmt.Sprint(argIdx+1)
	args = append(args, limit, offset)

	return r.fetchContestsWithQuery(ctx, query, args...)
}

func (r *contestRepo) GetFacultyContests(ctx context.Context, authorID string, limit, offset int, searchQuery string) ([]models.Contest, error) {
	query := `
		SELECT contest_id, title, host_organization, start_time, end_time, access_rules, author_id, is_public, created_at
		FROM contests WHERE (is_public = true OR author_id = $1)
	`
	args := []interface{}{authorID}
	argIdx := 2
	
	tsQuery := formatPrefixTSQuery(searchQuery)
	if tsQuery != "" {
		paramStr := `$` + fmt.Sprint(argIdx)
		query += ` AND fts @@ to_tsquery('english', ` + paramStr + `)`
		query += ` ORDER BY ts_rank(fts, to_tsquery('english', ` + paramStr + `)) DESC, start_time DESC`
		args = append(args, tsQuery)
		argIdx++
	} else {
		query += ` ORDER BY start_time DESC`
	}
	query += ` LIMIT $` + fmt.Sprint(argIdx) + ` OFFSET $` + fmt.Sprint(argIdx+1)
	args = append(args, limit, offset)

	return r.fetchContestsWithQuery(ctx, query, args...)
}

func (r *contestRepo) GetAllContests(ctx context.Context, limit, offset int, searchQuery string) ([]models.Contest, error) {
	query := `
		SELECT contest_id, title, host_organization, start_time, end_time, access_rules, author_id, is_public, created_at
		FROM contests WHERE 1=1
	`
	args := []interface{}{}
	argIdx := 1
	
	tsQuery := formatPrefixTSQuery(searchQuery)
	if tsQuery != "" {
		paramStr := `$` + fmt.Sprint(argIdx)
		query += ` AND fts @@ to_tsquery('english', ` + paramStr + `)`
		query += ` ORDER BY ts_rank(fts, to_tsquery('english', ` + paramStr + `)) DESC, start_time DESC`
		args = append(args, tsQuery)
		argIdx++
	} else {
		query += ` ORDER BY start_time DESC`
	}
	query += ` LIMIT $` + fmt.Sprint(argIdx) + ` OFFSET $` + fmt.Sprint(argIdx+1)
	args = append(args, limit, offset)

	return r.fetchContestsWithQuery(ctx, query, args...)
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
		SELECT contest_id, title, host_organization, start_time, end_time, access_rules, author_id, is_public, created_at
		FROM contests WHERE contest_id = $1
	`, contestID).Scan(&c.ID, &c.Title, &c.HostOrganization, &c.StartTime, &c.EndTime, &rulesJSON, &c.AuthorID, &c.IsPublic, &c.CreatedAt)

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

func (r *contestRepo) GetUserProfiles(ctx context.Context, userIDs []string) (map[string]models.ContestProfile, error) {
	if len(userIDs) == 0 {
		return map[string]models.ContestProfile{}, nil
	}

	rows, err := r.db.Query(ctx, `
		SELECT user_id, COALESCE(username, 'Anonymous'), COALESCE(role, 'student') FROM users WHERE user_id = ANY($1::uuid[])
	`, userIDs)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	profiles := make(map[string]models.ContestProfile)
	for rows.Next() {
		var id, name, role string
		// 👇 STRICT CHECK: Catch scan failures immediately
		if err := rows.Scan(&id, &name, &role); err != nil {
			return nil, err
		}
		profiles[id] = models.ContestProfile{Username: name, Role: role}
	}

	// 👇 STRICT CHECK: Catch any connection drops that occurred during the loop
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return profiles, nil
}

func (r *contestRepo) GetContestProblems(ctx context.Context, contestID, userID string) ([]map[string]interface{}, error) {
	rows, err := r.db.Query(ctx, `
		SELECT p.problem_id, p.title, p.difficulty, cp.points_value,
		       COALESCE(
		           -- 👇 FIX: Cast status to ::text so it doesn't clash with the 'unsolved' string
		           (SELECT status::text FROM submissions 
		            WHERE user_id = $2 AND problem_id = p.problem_id AND contest_id = $1
		            ORDER BY 
		                CASE WHEN status = 'AC' THEN 1 ELSE 2 END, -- AC takes priority
		                submitted_at DESC 
		            LIMIT 1),
		           'unsolved'
		       ) as user_status
		FROM contest_problems cp
		JOIN problems p ON cp.problem_id = p.problem_id
		WHERE cp.contest_id = $1
		ORDER BY p.created_at ASC
	`, contestID, userID)

	if err != nil {
		// 👇 Added a helpful error print so we can see DB failures in the Docker logs
		fmt.Printf("[!] DB Error in GetContestProblems: %v\n", err)
		return nil, err
	}
	defer rows.Close()

	var problems []map[string]interface{}
	for rows.Next() {
		var id, title, difficulty, status string
		var points int
		if err := rows.Scan(&id, &title, &difficulty, &points, &status); err == nil {
			problems = append(problems, map[string]interface{}{
				"problem_id":   id,
				"title":        title,
				"difficulty":   difficulty,
				"points_value": points,
				"user_status":  status,
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
	// Because of our schema's ON DELETE CASCADE constraints, deleting the contest
	// will automatically destroy all linked submissions, telemetry, plagiarism reports,
	// registrations, and contest_problems mappings.
	// The problems themselves remain perfectly safe in the 'problems' table.
	_, err := r.db.Exec(ctx, `DELETE FROM contests WHERE contest_id = $1`, contestID)
	return err
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

	// This query counts both browser telemetry and instances where the user was flagged in a MOSS pair
	rows, err := r.db.Query(ctx, `
		WITH TelemetryCounts AS (
			SELECT user_id,
				COUNT(*) as total_telemetry,
				COUNT(CASE WHEN event_type = 'blur' THEN 1 END) as blur,
				COUNT(CASE WHEN event_type = 'paste_attempt' THEN 1 END) as paste_attempt,
				COUNT(CASE WHEN event_type = 'autotyper_suspected' THEN 1 END) as autotyper,
				COUNT(CASE WHEN event_type = 'visibility_spoof_suspected' THEN 1 END) as spoof,
				COUNT(CASE WHEN event_type = 'anomalous_routing' THEN 1 END) as routing_anomaly
			FROM contest_telemetry
			WHERE contest_id = $1 AND user_id = ANY($2::uuid[])
			GROUP BY user_id
		),
		PlagiarismCounts AS (
			SELECT u.user_id, COUNT(p.report_id) as plagiarism
			FROM unnest($2::uuid[]) AS u(user_id)
			LEFT JOIN plagiarism_reports p ON (p.user_1_id = u.user_id OR p.user_2_id = u.user_id) AND p.contest_id = $1
			GROUP BY u.user_id
		)
		SELECT 
			u.user_id, 
			COALESCE(t.total_telemetry, 0) + COALESCE(p.plagiarism, 0) as total,
			COALESCE(t.blur, 0),
			COALESCE(t.paste_attempt, 0),
			COALESCE(t.autotyper, 0),
			COALESCE(t.spoof, 0),
			COALESCE(p.plagiarism, 0),
			COALESCE(t.routing_anomaly, 0)
		FROM unnest($2::uuid[]) AS u(user_id)
		LEFT JOIN TelemetryCounts t ON u.user_id = t.user_id
		LEFT JOIN PlagiarismCounts p ON u.user_id = p.user_id
	`, contestID, userIDs)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	alertsMap := make(map[string]models.TelemetryAlerts)
	for rows.Next() {
		var userID string
		var alerts models.TelemetryAlerts

		// Fast-fail on schema mismatch instead of silently ignoring it
		if err := rows.Scan(&userID, &alerts.Total, &alerts.Blur, &alerts.PasteAttempt, &alerts.AutotyperSuspected, &alerts.VisibilitySpoofSuspected, &alerts.Plagiarism, &alerts.AnomalousRouting); err != nil {
			// Provide accurate, contextual debugging information
			return nil, fmt.Errorf("failed scanning telemetry alerts for user %s: %w", userID, err)
		}
		alertsMap[userID] = alerts
	}
	// Catch network drops that occurred during the loop
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("cursor error during telemetry iteration: %w", err)
	}

	return alertsMap, nil
}

func (r *contestRepo) GetMossAuditStatus(ctx context.Context, contestID string) (string, error) {
	var status string
	err := r.db.QueryRow(ctx, "SELECT moss_audit_status FROM contests WHERE contest_id = $1", contestID).Scan(&status)
	return status, err
}

func (r *contestRepo) GetPendingMossAudits(ctx context.Context) ([]string, error) {
	rows, err := r.db.Query(ctx, `
		SELECT contest_id FROM contests 
		WHERE end_time <= NOW() AND moss_audit_status = 'pending'
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *contestRepo) UpdateMossAuditStatus(ctx context.Context, contestID, status string) error {
	_, err := r.db.Exec(ctx, "UPDATE contests SET moss_audit_status = $1, updated_at = NOW() WHERE contest_id = $2", status, contestID)
	return err
}
