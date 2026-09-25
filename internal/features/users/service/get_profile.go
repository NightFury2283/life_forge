package users_service

import (
	"context"
	"fmt"

	"github.com/NightFury2283/life_forge/internal/core/domain"
)

func (s *UsersService) GetProfile(ctx context.Context, userID int) (*domain.UserProgress, error) {
	// 1. repo.Get user info
	userProfile, err := s.usersRepository.GetProfile(ctx, userID)

	if err != nil {
		return &domain.UserProgress{}, fmt.Errorf("Get profile from repo: %w", err)
	}

	// 2. send user info
	return userProfile, nil
}
