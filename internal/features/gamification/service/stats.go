package gamification_service

import (
	"context"
	"fmt"
	core_errors "github.com/NightFury2283/life_forge/internal/core/errors"

	"github.com/NightFury2283/life_forge/internal/core/domain"
)

func (s *GamificationService) CreateStat(
	ctx context.Context,
	stat domain.Stat,
) (domain.Stat, error) {
	if err := stat.Validate(); err != nil {
		return domain.Stat{}, fmt.Errorf("Validate stat domain: %w", err)
	}

	createdStat, err := s.gamificationRepository.CreateStat(ctx, stat)
	if err != nil {
		return domain.Stat{}, fmt.Errorf("create stat in repository: %w", err)
	}

	return createdStat, nil
}

func (s *GamificationService) ListStats(ctx context.Context, userID int) ([]domain.Stat, error) {
	stats, err := s.gamificationRepository.ListStats(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list stats: %w", err)
	}
	return stats, nil
}

func (s *GamificationService) DeleteStat(ctx context.Context, userID, statID int) error {
	if userID <= 0 || statID <= 0 {
		return core_errors.ErrInvalidArgument
	}
	if err := s.gamificationRepository.DeleteStat(ctx, userID, statID); err != nil {
		return fmt.Errorf("delete stat: %w", err)
	}
	return nil
}
