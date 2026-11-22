package gamification_repository

import (
	"context"
	"errors"
	"fmt"
	core_errors "github.com/NightFury2283/life_forge/internal/core/errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/NightFury2283/life_forge/internal/core/domain"
)

func (r *GamificationRepository) CreateStat(
	ctx context.Context,
	stat domain.Stat,
) (domain.Stat, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OperationTimeout())
	defer cancel()

	query := `
	INSERT INTO lifeforge.user_stats (user_id, title, description, value)
	VALUES ($1, $2, $3, COALESCE($4, 0))
	RETURNING id, user_id, title, description, value;
	`

	row := r.pool.QueryRow(ctx, query, stat.UserID, stat.Title, stat.Description, stat.Value)

	var statModel StatModel

	err := row.Scan(
		&statModel.ID,
		&statModel.UserID,
		&statModel.Title,
		&statModel.Description,
		&statModel.Value,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.Stat{}, core_errors.ErrConflict
		}
		return domain.Stat{}, fmt.Errorf("row scan: %w", err)
	}

	statDomain := domain.NewStat(
		statModel.ID,
		statModel.UserID,
		statModel.Title,
		statModel.Description,
		statModel.Value,
	)

	return statDomain, nil
}

func (r *GamificationRepository) AdjustStat(ctx context.Context, userID, statID, delta int) (domain.Stat, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OperationTimeout())
	defer cancel()
	var stat domain.Stat
	err := r.pool.QueryRow(ctx, `UPDATE lifeforge.user_stats SET value=GREATEST(0,LEAST(2147483647,value::bigint+$3)) WHERE id=$1 AND user_id=$2 RETURNING id,user_id,title,description,value`, statID, userID, delta).Scan(&stat.ID, &stat.UserID, &stat.Title, &stat.Description, &stat.Value)
	if errors.Is(err, pgx.ErrNoRows) {
		return stat, core_errors.ErrNotFound
	}
	return stat, err
}

func (r *GamificationRepository) ListStats(ctx context.Context, userID int) ([]domain.Stat, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OperationTimeout())
	defer cancel()
	rows, err := r.pool.Query(ctx, `SELECT id, user_id, title, description, value
		FROM lifeforge.user_stats WHERE user_id = $1 ORDER BY id`, userID)
	if err != nil {
		return nil, fmt.Errorf("query stats: %w", err)
	}
	defer rows.Close()
	stats := make([]domain.Stat, 0)
	for rows.Next() {
		var stat domain.Stat
		if err := rows.Scan(&stat.ID, &stat.UserID, &stat.Title, &stat.Description, &stat.Value); err != nil {
			return nil, fmt.Errorf("scan stat: %w", err)
		}
		stats = append(stats, stat)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate stats: %w", err)
	}
	return stats, nil
}

func (r *GamificationRepository) DeleteStat(ctx context.Context, userID, statID int) error {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OperationTimeout())
	defer cancel()
	tag, err := r.pool.Exec(ctx, `DELETE FROM lifeforge.user_stats WHERE id = $1 AND user_id = $2`, statID, userID)
	if err != nil {
		return fmt.Errorf("delete stat: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return core_errors.ErrNotFound
	}
	return nil
}
