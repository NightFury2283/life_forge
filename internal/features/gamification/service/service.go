package gamification_service

import (
	"context"

	"github.com/NightFury2283/life_forge/internal/core/domain"
)

type GamificationService struct {
	gamificationRepository GamificationRepository
}

type GamificationRepository interface {
	ListStats(ctx context.Context, userID int) ([]domain.Stat, error)
	DeleteStat(ctx context.Context, userID, statID int) error
	CreateStat(
		ctx context.Context,
		stat domain.Stat,
	) (domain.Stat, error)
}

func NewGamificationService(repository GamificationRepository) *GamificationService {
	return &GamificationService{
		gamificationRepository: repository,
	}
}
