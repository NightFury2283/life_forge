package gamification_repository

import (
	"context"
	"errors"
	"fmt"
	"github.com/NightFury2283/life_forge/internal/core/domain"
	core_errors "github.com/NightFury2283/life_forge/internal/core/errors"
	"github.com/jackc/pgx/v5"
)

func (r *GamificationRepository) GetHeroSummary(ctx context.Context, userID int) (domain.HeroSummary, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OperationTimeout())
	defer cancel()
	s := domain.HeroSummary{Activity: make([]domain.ActivityDay, 0)}
	if err := r.pool.QueryRow(ctx, `SELECT count(*),COALESCE(sum(xp),0) FROM lifeforge.completed_tasks WHERE user_id=$1`, userID).Scan(&s.Completed, &s.EarnedXP); err != nil {
		return s, err
	}
	rows, err := r.pool.Query(ctx, `SELECT to_char(completed_at AT TIME ZONE 'Europe/Moscow','YYYY-MM-DD'),count(*) FROM lifeforge.completed_tasks WHERE user_id=$1 AND completed_at>=CURRENT_TIMESTAMP-interval '35 days' GROUP BY 1 ORDER BY 1`, userID)
	if err != nil {
		return s, err
	}
	defer rows.Close()
	for rows.Next() {
		var day domain.ActivityDay
		if err := rows.Scan(&day.Date, &day.Count); err != nil {
			return s, err
		}
		s.Activity = append(s.Activity, day)
	}
	return s, rows.Err()
}
func (r *GamificationRepository) CompletedEvents(ctx context.Context, userID int) (map[string]bool, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OperationTimeout())
	defer cancel()
	rows, err := r.pool.Query(ctx, `SELECT calendar_id,event_id FROM lifeforge.completed_tasks WHERE user_id=$1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]bool)
	for rows.Next() {
		var c, e string
		if err := rows.Scan(&c, &e); err != nil {
			return nil, err
		}
		result[c+"/"+e] = true
	}
	return result, rows.Err()
}
func (r *GamificationRepository) CompleteEvent(ctx context.Context, userID int, c domain.Completion) (domain.UserProgress, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OperationTimeout())
	defer cancel()
	// The unique event insert and reward update form one atomic SQL statement.
	// Concurrent duplicate requests cannot earn a second reward.
	query := `WITH inserted AS (
 INSERT INTO lifeforge.completed_tasks(user_id,calendar_id,event_id,title,xp,coins)
 SELECT $1,$2,$3,$4,10,2 WHERE EXISTS(SELECT 1 FROM lifeforge.user_progress WHERE user_id=$1)
 ON CONFLICT DO NOTHING RETURNING user_id
 ) UPDATE lifeforge.user_progress p SET
 level=CASE WHEN xp+10>=xp_for_next_level THEN level+1 ELSE level END,
 xp=CASE WHEN xp+10>=xp_for_next_level THEN xp+10-xp_for_next_level ELSE xp+10 END,
 xp_for_next_level=CASE WHEN xp+10>=xp_for_next_level THEN floor(xp_for_next_level*1.5)::integer ELSE xp_for_next_level END,
 coins=coins+2,updated_at=CURRENT_TIMESTAMP
 FROM inserted WHERE p.user_id=inserted.user_id
 RETURNING p.user_id,p.level,p.xp,p.xp_for_next_level,p.coins`
	var p domain.UserProgress
	err := r.pool.QueryRow(ctx, query, userID, c.CalendarID, c.EventID, c.Title).Scan(&p.UserID, &p.Level, &p.XP, &p.XPForNextLevel, &p.Coins)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, core_errors.ErrConflict
	}
	if err != nil {
		return p, fmt.Errorf("complete event: %w", err)
	}
	return p, nil
}
