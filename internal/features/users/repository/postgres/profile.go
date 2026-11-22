package users_postgres_repository

import (
	"context"
	"errors"
	"fmt"
	core_errors "github.com/NightFury2283/life_forge/internal/core/errors"
	"github.com/jackc/pgx/v5"

	"github.com/NightFury2283/life_forge/internal/core/domain"
)

func (r *UsersRepository) GetProfile(ctx context.Context, userID int) (domain.UserProgress, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OperationTimeout())
	defer cancel()

	query := `
	SELECT level, xp, xp_for_next_level, coins 
	FROM lifeforge.user_progress
	WHERE user_id = $1;
	`

	row := r.pool.QueryRow(ctx, query, userID)

	var userProfile UserProgressModel
	err := row.Scan(
		&userProfile.Level,
		&userProfile.XP,
		&userProfile.XPForNextLevel,
		&userProfile.Coins,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.UserProgress{}, core_errors.ErrNotFound
		}
		return domain.UserProgress{}, fmt.Errorf("User profile row scan: %w", err)
	}

	userDomainProfile := domain.UserProgress{
		UserID:         userID,
		Level:          userProfile.Level,
		XP:             userProfile.XP,
		XPForNextLevel: userProfile.XPForNextLevel,
		Coins:          userProfile.Coins,
	}

	return userDomainProfile, nil
}
