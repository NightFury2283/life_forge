package core_auth_jwt

import (
	"fmt"
	"github.com/kelseyhightower/envconfig"
)

type JWTConfig struct {
	SecretKey string `envconfig:"JWT_SECRET_KEY" required:"true"`
}

func NewConfig() (JWTConfig, error) {
	var jwtConfig JWTConfig

	if err := envconfig.Process("", &jwtConfig); err != nil {
		return JWTConfig{}, fmt.Errorf("Process jwt env config: %w", err)
	}

	return jwtConfig, nil
}

func NewConfigMust() JWTConfig {
	jwtConfig, err := NewConfig()
	if err != nil {
		err = fmt.Errorf("get jwt config: %w", err)
		panic(err)
	}

	return jwtConfig
}