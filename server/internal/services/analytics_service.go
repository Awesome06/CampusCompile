package services

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AnalyticsService interface {
	StartPlaylistAnalyticsDaemon(ctx context.Context)
}

type analyticsService struct {
	db *pgxpool.Pool
}

func NewAnalyticsService(db *pgxpool.Pool) AnalyticsService {
	return &analyticsService{db: db}
}

func (s *analyticsService) StartPlaylistAnalyticsDaemon(ctx context.Context) {
	// Aggregate hourly
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	// Initial aggregation on startup
	s.aggregatePlaylistAnalytics(ctx)

	for {
		select {
		case <-ctx.Done():
			slog.Info("Shutting down Playlist Analytics daemon gracefully", "component", "StartPlaylistAnalyticsDaemon")
			return
		case <-ticker.C:
			s.aggregatePlaylistAnalytics(ctx)
		}
	}
}

func (s *analyticsService) aggregatePlaylistAnalytics(ctx context.Context) {
	slog.Info("Starting hourly playlist analytics aggregation...")

	// 1. Calculate problem solve counts ONLY for students
	// 2. We use an UPSERT (ON CONFLICT) so we don't duplicate rows
	// The primary key for playlist_analytics_summary is (playlist_id, problem_id)
	query := `
		WITH PlaylistStudents AS (
		    SELECT DISTINCT pp.playlist_id, s.user_id 
			FROM submissions s 
			JOIN playlist_problems pp ON s.problem_id = pp.problem_id
			WHERE s.user_id IN (SELECT user_id FROM users WHERE role = 'student')
		),
		TotalStudents AS (
		    SELECT playlist_id, COUNT(DISTINCT user_id) as total_students
			FROM PlaylistStudents
			GROUP BY playlist_id
		),
		CompletedCounts AS (
			SELECT pp.playlist_id, s.problem_id, COUNT(DISTINCT s.user_id) as completed_count
			FROM submissions s
			JOIN playlist_problems pp ON s.problem_id = pp.problem_id
			JOIN PlaylistStudents ps ON s.user_id = ps.user_id AND ps.playlist_id = pp.playlist_id
			WHERE s.status = 'AC'
			GROUP BY pp.playlist_id, s.problem_id
		)
		INSERT INTO playlist_analytics_summary (playlist_id, problem_id, completed_count, total_students, last_updated)
		SELECT pp.playlist_id, pp.problem_id, COALESCE(cc.completed_count, 0), COALESCE(ts.total_students, 0), NOW()
		FROM playlist_problems pp
		LEFT JOIN CompletedCounts cc ON pp.playlist_id = cc.playlist_id AND pp.problem_id = cc.problem_id
		LEFT JOIN TotalStudents ts ON pp.playlist_id = ts.playlist_id
		ON CONFLICT (playlist_id, problem_id) 
		DO UPDATE SET 
			completed_count = EXCLUDED.completed_count,
			total_students = EXCLUDED.total_students,
			last_updated = NOW();
	`

	_, err := s.db.Exec(ctx, query)
	if err != nil {
		slog.Error("Failed to update playlist_analytics_summary", "error", err)
		return
	}

	slog.Info("Hourly playlist analytics aggregation completed successfully.")
}
