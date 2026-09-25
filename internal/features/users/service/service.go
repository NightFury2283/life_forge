package users_service

import (
	"context"

	"github.com/NightFury2283/life_forge/internal/core/domain"
)

type UsersService struct {
	usersRepository UsersRepository
}

type UsersRepository interface {
	GetProfile(
		ctx context.Context,
		userID int,
	) (*domain.UserProgress, error)
}

func NewUsersService(usersRepository UsersRepository) *UsersService {
	return &UsersService{
		usersRepository: usersRepository,
	}
}
