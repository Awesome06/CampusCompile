package repositories

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"

	"campuscompile/api/internal/models"
)

type ContestRepository interface {
	CreateContest(ctx context.Context, contest models.Contest) error
	GetContests(ctx context.Context) ([]models.Contest, error)
	GetContestByID(ctx context.Context, contestID string) (models.Contest, error)
	RegisterUser(ctx context.Context, contestID, userID string) error
	CheckRegistration(ctx context.Context, contestID, userID string) (bool, error)
	GetUsernames(ctx context.Context, userIDs []string) (map[string]string, error)
}

type contestRepo struct {
	db *pgxpool.Pool
}

func NewContestRepository(db *pgxpool.Pool) ContestRepository {
	return &contestRepo{db: db}
}

func (r *contestRepo) CreateContest(ctx context.Context, contest models.Contest) error {
	var rulesJSON []byte
	var err error

	if contest.AccessRules != nil {
		rulesJSON, err = json.Marshal(contest.AccessRules)
		if err != nil {
			return err
		}
	}

	_, err = r.db.Exec(ctx, `
		INSERT INTO contests (contest_id, title, host_organization, start_time, end_time, access_rules)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, contest.ID, contest.Title, contest.HostOrganization, contest.StartTime, contest.EndTime, rulesJSON)

	return err
}

func (r *contestRepo) GetContests(ctx context.Context) ([]models.Contest, error) {
	rows, err := r.db.Query(ctx, `
		SELECT contest_id, title, host_organization, start_time, end_time, created_at
		FROM contests ORDER BY start_time DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contests []models.Contest
	for rows.Next() {
		var c models.Contest
		if err := rows.Scan(&c.ID, &c.Title, &c.HostOrganization, &c.StartTime, &c.EndTime, &c.CreatedAt); err == nil {
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
