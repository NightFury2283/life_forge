package users_postgres_repository

import (
	"context"
	"fmt"

	"github.com/NightFury2283/life_forge/internal/core/domain"
)

func (repo *UsersRepository) GetOrCreateUserByGoogleID(
	ctx context.Context,
	googleID, email, name string,
) (*domain.User, error) {
	ctx, cancel := context.WithTimeout(ctx, repo.pool.OperationTimeout())
	defer cancel()

	var user domain.User

	query := `
		INSERT INTO lifeforge.users (google_id, email, name)
		VALUES ($1, $2, $3)
		ON CONFLICT (google_id) DO UPDATE
		SET email = COALESCE(EXCLUDED.email, lifeforge.users.email),
    		name  = COALESCE(EXCLUDED.name,  lifeforge.users.name)
		RETURNING id, google_id, email, name, created_at`

	err := repo.pool.QueryRow(ctx, query, googleID, email, name).Scan(
		&user.ID, &user.GoogleID, &user.Email, &user.Name, &user.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("upsert user: %w", err)
	}

	//TODO: возможно создавать прогресс, если новый пользователь

	return &user, nil
}
