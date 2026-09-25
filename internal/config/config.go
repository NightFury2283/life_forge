package config

import (
	"fmt"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	PostgresDSN     string `envconfig:"POSTGRES_DSN" required:"true"`
	GigaChatAuthKey string `envconfig:"GIGACHAT_AUTH_KEY" required:"true"`
}

func NewConfig() (*Config, error) {
	var config *Config
	if err := envconfig.Process("", config); err != nil {
		return nil, fmt.Errorf("process envconfig: %w", err)
	}

	return config, nil
}

func NewConfigMust() *Config {
	config, err := NewConfig()
	if err != nil {
		err = fmt.Errorf("get config from NewConfig: %w", err)
		panic(err)
	}

	return config
}
