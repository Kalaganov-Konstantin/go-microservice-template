package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
)

type Config interface {
	DSN() string
	GetMaxOpenConns() int
	GetMaxIdleConns() int
	GetConnMaxLifetime() time.Duration
	GetConnMaxIdleTime() time.Duration
}

type DB struct {
	*sql.DB
	config Config
}

func New(cfg Config) (*DB, error) {
	db, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(cfg.GetMaxOpenConns())
	db.SetMaxIdleConns(cfg.GetMaxIdleConns())
	db.SetConnMaxLifetime(cfg.GetConnMaxLifetime())
	db.SetConnMaxIdleTime(cfg.GetConnMaxIdleTime())

	return &DB{
		DB:     db,
		config: cfg,
	}, nil
}

func (db *DB) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return db.DB.PingContext(ctx)
}

func (db *DB) Close() error {
	return db.DB.Close()
}
