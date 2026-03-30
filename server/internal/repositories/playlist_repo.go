package repositories

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"campuscompile/api/internal/models"
)

type PlaylistRepository interface {
	CreatePlaylist(ctx context.Context, req models.CreatePlaylistRequest, authorID string) (string, error)
	GetPlaylists(ctx context.Context, userID, viewMode string, limit, offset int) ([]models.PlaylistResponse, error)
	GetPlaylistByID(ctx context.Context, playlistID string) (models.PlaylistResponse, error)
	GetPlaylistProblems(ctx context.Context, playlistID, userID string) ([]models.PlaylistProblemResponse, error)
	GetPlaylistAnalytics(ctx context.Context, playlistID string) ([]models.PlaylistAnalyticsItem, error)
	UpdatePlaylist(ctx context.Context, playlistID string, req models.CreatePlaylistRequest) error
}

type playlistRepo struct {
	db *pgxpool.Pool
}

func NewPlaylistRepository(db *pgxpool.Pool) PlaylistRepository {
	return &playlistRepo{db: db}
}

func (r *playlistRepo) CreatePlaylist(ctx context.Context, req models.CreatePlaylistRequest, authorID string) (string, error) {
	if req.IsPublic {
		var problemIDs []string
		for _, p := range req.Problems {
			if id, ok := p["problem_id"].(string); ok && id != "" {
				problemIDs = append(problemIDs, id)
			}
		}

		if len(problemIDs) > 0 {
			var privateCount int
			err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM problems WHERE problem_id = ANY($1::uuid[]) AND is_public = false`, problemIDs).Scan(&privateCount)
			if err != nil {
				return "", fmt.Errorf("failed to validate problem visibility: %w", err)
			}
			if privateCount > 0 {
				return "", fmt.Errorf("cannot publish playlist: it contains %d private/draft problems", privateCount)
			}
		}
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	var playlistID string
	err = tx.QueryRow(ctx, `
		INSERT INTO playlists (title, description, author_id, is_public, overall_difficulty)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING playlist_id
	`, req.Title, req.Description, authorID, req.IsPublic, req.OverallDifficulty).Scan(&playlistID)

	if err != nil {
		return "", err
	}

	tagSet := make(map[string]bool)
	for _, tag := range req.Tags {
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
			return "", err
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO playlist_tags (playlist_id, tag_id) VALUES ($1, $2)
			ON CONFLICT DO NOTHING
		`, playlistID, tagID)
		if err != nil {
			return "", err
		}
	}

	for i, p := range req.Problems {
		problemID, ok := p["problem_id"].(string)
		if !ok || problemID == "" {
			return "", fmt.Errorf("invalid or missing problem_id attached to sequence index %d", i)
		}
		
		var customDiff *string
		if cd, ok := p["custom_difficulty"].(string); ok && cd != "" && cd != "No Change" {
			cdStr := strings.TrimSpace(cd)
			// Postgres format casing matches UI
			if strings.EqualFold(cdStr, "easy") { 
				cdStr = "Easy" 
			} else if strings.EqualFold(cdStr, "medium") { 
				cdStr = "Medium" 
			} else if strings.EqualFold(cdStr, "hard") { 
				cdStr = "Hard" 
			} else {
				return "", fmt.Errorf("invalid custom_difficulty literal '%s' attached to sequence index %d", cd, i)
			}
			customDiff = &cdStr
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO playlist_problems (playlist_id, problem_id, order_index, custom_difficulty)
			VALUES ($1, $2, $3, $4)
		`, playlistID, problemID, i+1, customDiff)

		if err != nil {
			return "", err
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return "", err
	}

	return playlistID, nil
}

func (r *playlistRepo) GetPlaylists(ctx context.Context, userID, viewMode string, limit, offset int) ([]models.PlaylistResponse, error) {
	query := `
		SELECT p.playlist_id, p.title, p.description, p.author_id, p.is_public, p.overall_difficulty, 
		       COALESCE((
		           SELECT ARRAY_AGG(t.name) 
		           FROM playlist_tags pt JOIN tags t ON pt.tag_id = t.tag_id 
                   WHERE pt.playlist_id = p.playlist_id
		       ), ARRAY[]::VARCHAR[]) as tags,
		       p.created_at, p.updated_at,
		       COALESCE(u.real_name, u.username, 'Anonymous') as author_name,
		       (SELECT COUNT(*) FROM playlist_problems pp WHERE pp.playlist_id = p.playlist_id) as total_count,
		       COALESCE((
		           SELECT COUNT(DISTINCT s.problem_id) 
		           FROM submissions s
		           JOIN playlist_problems pp2 ON s.problem_id = pp2.problem_id
		           WHERE pp2.playlist_id = p.playlist_id AND s.user_id = $1 AND s.status = 'AC'
		       ), 0) as solved_count
		FROM playlists p
		LEFT JOIN users u ON p.author_id = u.user_id
		WHERE 1=1
	`
	
	args := []interface{}{userID}
	argIdx := 2

	if viewMode == "faculty" {
		query += ` AND p.author_id = $` + fmt.Sprint(argIdx)
		args = append(args, userID)
		argIdx++
	} else {
		// Public
		query += ` AND p.is_public = true`
	}

	query += ` ORDER BY p.created_at DESC LIMIT $` + fmt.Sprint(argIdx) + ` OFFSET $` + fmt.Sprint(argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var playlists []models.PlaylistResponse
	for rows.Next() {
		var p models.PlaylistResponse
		var authorID *string
		if err := rows.Scan(&p.ID, &p.Title, &p.Description, &authorID, &p.IsPublic, &p.OverallDifficulty, &p.Tags, &p.CreatedAt, &p.UpdatedAt, &p.AuthorName, &p.TotalCount, &p.SolvedCount); err != nil {
			return nil, fmt.Errorf("failed to scan playlist row: %w", err)
		}
		p.AuthorID = authorID
		playlists = append(playlists, p)
	}
	if playlists == nil {
		playlists = []models.PlaylistResponse{}
	}
	return playlists, nil
}

func (r *playlistRepo) GetPlaylistByID(ctx context.Context, playlistID string) (models.PlaylistResponse, error) {
	var p models.PlaylistResponse
	var authorID *string
	err := r.db.QueryRow(ctx, `
		SELECT p.playlist_id, p.title, p.description, p.author_id, p.is_public, p.overall_difficulty, 
		       COALESCE((
		           SELECT ARRAY_AGG(t.name) 
		           FROM playlist_tags pt JOIN tags t ON pt.tag_id = t.tag_id 
                   WHERE pt.playlist_id = p.playlist_id
		       ), ARRAY[]::VARCHAR[]) as tags,
		       p.created_at, p.updated_at,
		       COALESCE(u.real_name, u.username, 'Anonymous') as author_name,
		       (SELECT COUNT(*) FROM playlist_problems pp WHERE pp.playlist_id = p.playlist_id) as total_count
		FROM playlists p
		LEFT JOIN users u ON p.author_id = u.user_id
		WHERE p.playlist_id = $1
	`, playlistID).Scan(&p.ID, &p.Title, &p.Description, &authorID, &p.IsPublic, &p.OverallDifficulty, &p.Tags, &p.CreatedAt, &p.UpdatedAt, &p.AuthorName, &p.TotalCount)
	p.AuthorID = authorID
	return p, err
}

func (r *playlistRepo) GetPlaylistProblems(ctx context.Context, playlistID, userID string) ([]models.PlaylistProblemResponse, error) {
	rows, err := r.db.Query(ctx, `
		SELECT p.problem_id, p.title, p.slug, p.difficulty, pp.order_index, pp.custom_difficulty,
		       COALESCE(
		           (SELECT status::VARCHAR FROM submissions 
		            WHERE user_id = $2 AND problem_id = p.problem_id 
		            ORDER BY CASE WHEN status = 'AC' THEN 1 ELSE 2 END, submitted_at DESC LIMIT 1),
		           'Unattempted'
		       ) as status
		FROM playlist_problems pp
		JOIN problems p ON pp.problem_id = p.problem_id
		WHERE pp.playlist_id = $1
		ORDER BY pp.order_index ASC
	`, playlistID, userID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var problems []models.PlaylistProblemResponse
	for rows.Next() {
		var pr models.PlaylistProblemResponse
		var customDiff *string
		if err := rows.Scan(&pr.ID, &pr.Title, &pr.Slug, &pr.Difficulty, &pr.OrderIndex, &customDiff, &pr.Status); err != nil {
			return nil, fmt.Errorf("failed to scan playlist problem row: %w", err)
		}
		pr.CustomDifficulty = customDiff
		if pr.Status != "AC" && pr.Status != "Unattempted" {
			pr.Status = "Attempted/WA"
		} else if pr.Status == "AC" {
			pr.Status = "Accepted"
		}
		problems = append(problems, pr)
	}
	if problems == nil {
		problems = []models.PlaylistProblemResponse{}
	}
	return problems, nil
}

func (r *playlistRepo) GetPlaylistAnalytics(ctx context.Context, playlistID string) ([]models.PlaylistAnalyticsItem, error) {
	rows, err := r.db.Query(ctx, `
		WITH PlaylistStudents AS (
		    SELECT DISTINCT s.user_id 
			FROM submissions s 
			JOIN playlist_problems pp ON s.problem_id = pp.problem_id 
			WHERE pp.playlist_id = $1
		),
		TotalStudents AS (
		    SELECT COUNT(*) as t FROM PlaylistStudents
		)
		SELECT 
		    pp.problem_id, 
		    pp.order_index, 
		    (SELECT COUNT(DISTINCT s.user_id) FROM submissions s 
			 JOIN PlaylistStudents ps ON s.user_id = ps.user_id 
			 WHERE s.problem_id = pp.problem_id AND s.status = 'AC') as completed_count,
		    (SELECT t FROM TotalStudents) as total_students
		FROM playlist_problems pp
		WHERE pp.playlist_id = $1
		ORDER BY pp.order_index ASC
	`, playlistID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var analytics []models.PlaylistAnalyticsItem
	for rows.Next() {
		var a models.PlaylistAnalyticsItem
		if err := rows.Scan(&a.ProblemID, &a.OrderIndex, &a.CompletedCount, &a.TotalStudents); err != nil {
			return nil, fmt.Errorf("failed to scan playlist analytics row: %w", err)
		}
		analytics = append(analytics, a)
	}
	if analytics == nil {
		analytics = []models.PlaylistAnalyticsItem{}
	}
	return analytics, nil
}

func (r *playlistRepo) UpdatePlaylist(ctx context.Context, playlistID string, req models.CreatePlaylistRequest) error {
	if req.IsPublic {
		var problemIDs []string
		for _, p := range req.Problems {
			if id, ok := p["problem_id"].(string); ok && id != "" {
				problemIDs = append(problemIDs, id)
			}
		}

		if len(problemIDs) > 0 {
			var privateCount int
			err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM problems WHERE problem_id = ANY($1::uuid[]) AND is_public = false`, problemIDs).Scan(&privateCount)
			if err != nil {
				return fmt.Errorf("failed to validate problem visibility: %w", err)
			}
			if privateCount > 0 {
				return fmt.Errorf("cannot publish playlist: it contains %d private/draft problems", privateCount)
			}
		}
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		UPDATE playlists 
		SET title = $1, description = $2, is_public = $3, overall_difficulty = $4, updated_at = NOW()
		WHERE playlist_id = $5
	`, req.Title, req.Description, req.IsPublic, req.OverallDifficulty, playlistID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, "DELETE FROM playlist_tags WHERE playlist_id = $1", playlistID)
	if err != nil {
		return err
	}

	tagSet := make(map[string]bool)
	for _, tag := range req.Tags {
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
			INSERT INTO playlist_tags (playlist_id, tag_id) VALUES ($1, $2)
			ON CONFLICT DO NOTHING
		`, playlistID, tagID)
		if err != nil {
			return err
		}
	}

	_, err = tx.Exec(ctx, "DELETE FROM playlist_problems WHERE playlist_id = $1", playlistID)
	if err != nil {
		return err
	}

	for i, p := range req.Problems {
		problemID, ok := p["problem_id"].(string)
		if !ok || problemID == "" {
			return fmt.Errorf("invalid or missing problem_id attached to sequence index %d", i)
		}
		
		var customDiff *string
		if cd, ok := p["custom_difficulty"].(string); ok && cd != "" && cd != "No Change" {
			cdStr := strings.TrimSpace(cd)
			if strings.EqualFold(cdStr, "easy") { 
				cdStr = "Easy" 
			} else if strings.EqualFold(cdStr, "medium") { 
				cdStr = "Medium" 
			} else if strings.EqualFold(cdStr, "hard") { 
				cdStr = "Hard" 
			} else {
				return fmt.Errorf("invalid custom_difficulty literal '%s' attached to sequence index %d", cd, i)
			}
			customDiff = &cdStr
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO playlist_problems (playlist_id, problem_id, order_index, custom_difficulty)
			VALUES ($1, $2, $3, $4)
		`, playlistID, problemID, i+1, customDiff)

		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}
