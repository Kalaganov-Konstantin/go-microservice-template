package config

import (
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type DatabaseConfig struct {
	BaseConfig
	Postgres PostgresConfig `envconfig:"POSTGRES"`
}

type PostgresConfig struct {
	Host            string        `envconfig:"HOST" default:"localhost"`
	Port            int           `envconfig:"PORT" default:"5432"`
	User            string        `envconfig:"USER" default:"postgres"`
	Password        string        `envconfig:"PASSWORD" default:""`
	Database        string        `envconfig:"DB" default:"microservice"`
	SSLMode         string        `envconfig:"SSL_MODE" default:"disable"`
	MaxOpenConns    int           `envconfig:"MAX_OPEN_CONNS" default:"25"`
	MaxIdleConns    int           `envconfig:"MAX_IDLE_CONNS" default:"5"`
	ConnMaxLifetime time.Duration `envconfig:"CONN_MAX_LIFETIME" default:"5m"`
	ConnMaxIdleTime time.Duration `envconfig:"CONN_MAX_IDLE_TIME" default:"5m"`
}

func (c *PostgresConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Database, c.SSLMode)
}

func (c *PostgresConfig) GetMaxOpenConns() int {
	return c.MaxOpenConns
}

func (c *PostgresConfig) GetMaxIdleConns() int {
	return c.MaxIdleConns
}

func (c *PostgresConfig) GetConnMaxLifetime() time.Duration {
	return c.ConnMaxLifetime
}

func (c *PostgresConfig) GetConnMaxIdleTime() time.Duration {
	return c.ConnMaxIdleTime
}

func LoadDatabase() (*DatabaseConfig, error) {
	var cfg DatabaseConfig
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
