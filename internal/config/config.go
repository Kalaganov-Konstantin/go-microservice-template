package config

import (
	"microservice/internal/platform/logger"
	"strings"

	"github.com/kelseyhightower/envconfig"
)

const (
	EnvDevelopment = "development"
	EnvStaging     = "staging"
	EnvProduction  = "production"
	EnvTest        = "test"
)

type BaseConfig struct {
	Environment string       `envconfig:"ENV" default:"development" validate:"oneof=development staging production test"`
	Logger      LoggerConfig `envconfig:"LOGGER"`
}

type LoggerConfig struct {
	Level  logger.Level  `envconfig:"LEVEL" default:"info"`
	Format logger.Format `envconfig:"FORMAT" default:"json"`
}

func LoadBase() (*BaseConfig, error) {
	var cfg BaseConfig
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *BaseConfig) IsDevelopment() bool {
	return strings.ToLower(c.Environment) == EnvDevelopment
}

func (c *BaseConfig) IsProduction() bool {
	return strings.ToLower(c.Environment) == EnvProduction
}

func (c *BaseConfig) IsStaging() bool {
	return strings.ToLower(c.Environment) == EnvStaging
}

func (c *BaseConfig) IsTest() bool {
	return strings.ToLower(c.Environment) == EnvTest
}
