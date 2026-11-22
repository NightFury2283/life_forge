package gamification_repository

import core_postgres_pool "github.com/NightFury2283/life_forge/internal/core/repository/postgres/pool"

type GamificationRepository struct {
	pool core_postgres_pool.Pool
}

func NewGamificationRepository(
	pool core_postgres_pool.Pool,
) *GamificationRepository {
	return &GamificationRepository{
		pool: pool,
	}
}
