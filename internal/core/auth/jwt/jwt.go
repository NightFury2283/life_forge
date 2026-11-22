package core_auth_jwt

import (
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"strconv"
	"time"
)

type Claims struct {
	UserID int `json:"user_id"`
	jwt.RegisteredClaims
}

type TokenService struct {
	secretKey []byte
}

func NewTokenService(jwtConfig JWTConfig) *TokenService {
	return &TokenService{
		secretKey: []byte(jwtConfig.SecretKey),
	}
}

func (s *TokenService) GenerateToken(userID int) (string, error) {
	if userID <= 0 || len(s.secretKey) < 32 {
		return "", errors.New("positive user ID and JWT secret of at least 32 bytes are required")
	}
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "life-forge",
			Subject:   strconv.Itoa(userID),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secretKey)
}

func (s *TokenService) ParseToken(tokenString string) (int, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return s.secretKey, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(), jwt.WithIssuedAt(), jwt.WithIssuer("life-forge"))

	if err != nil {
		return 0, fmt.Errorf("Parse jwt token with claims: %w", err)
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		if claims.UserID <= 0 || claims.Subject != strconv.Itoa(claims.UserID) {
			return 0, errors.New("invalid user identity")
		}
		return claims.UserID, nil
	}

	return 0, fmt.Errorf("parse token claims to struct Claims or invalid token: %w", err)
}
