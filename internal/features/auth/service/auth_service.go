package auth_service

import (
	"context"
	"fmt"

	"github.com/NightFury2283/life_forge/internal/core/domain"
	"golang.org/x/oauth2"
)

type GoogleAuthProvider interface {
	GetAuthURL(state string) string
	ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error)
	GetGoogleUserInfo(ctx context.Context, tok *oauth2.Token) (*domain.GoogleUserInfo, error)
}

type UserRepository interface {
	GetOrCreateUserByGoogleID(ctx context.Context, googleID, email, name string) (domain.User, error)
}

type TokenService interface {
	GenerateToken(userID int) (string, error)
}

type AuthService struct {
	googleProvider GoogleAuthProvider
	userRepo       UserRepository
	tokenService   TokenService
}

func NewAuthService(
	googleProvider GoogleAuthProvider,
	userRepo UserRepository,
	tokenService TokenService,
) *AuthService {
	return &AuthService{
		googleProvider: googleProvider,
		userRepo:       userRepo,
		tokenService:   tokenService,
	}
}

func (s *AuthService) GetGoogleAuthURL(state string) string {
	return s.googleProvider.GetAuthURL(state)
}

func (s *AuthService) LoginViaGoogle(ctx context.Context, code string) (string, error) {
	if code == "" {
		return "", fmt.Errorf("google authorization code is required")
	}
	token, err := s.googleProvider.ExchangeCode(ctx, code)
	if err != nil {
		return "", fmt.Errorf("exchange google code: %w", err)
	}

	userInfo, err := s.googleProvider.GetGoogleUserInfo(ctx, token)
	if err != nil {
		return "", fmt.Errorf("get google user info: %w", err)
	}

	user, err := s.userRepo.GetOrCreateUserByGoogleID(ctx, userInfo.ID, userInfo.Email, userInfo.Name)
	if err != nil {
		return "", fmt.Errorf("get or create user: %w", err)
	}

	jwtToken, err := s.tokenService.GenerateToken(user.ID)
	if err != nil {
		return "", fmt.Errorf("generate jwt: %w", err)
	}

	return jwtToken, nil
}
