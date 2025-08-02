package database

import (
	"context"
	"microservice/internal/platform/database/postgres"
	"microservice/internal/platform/logger"
	"sync"

	"microservice/internal/config"
)

type Lifecycle struct {
	cfg    *config.DatabaseConfig
	logger logger.Logger
	db     *postgres.DB
	mu     sync.Mutex
}

func NewDatabaseLifecycle(cfg *config.DatabaseConfig, log logger.Logger) *Lifecycle {
	return &Lifecycle{
		cfg:    cfg,
		logger: log,
	}
}

func (d *Lifecycle) Start(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Close existing connection if any
	if d.db != nil {
		d.logger.Warn("Database connection already exists, closing existing connection")
		if err := d.db.Close(); err != nil {
			d.logger.Error("Failed to close existing database connection", logger.Error(err))
		}
		d.db = nil
	}

	d.logger.Info("Starting database connection")

	db, err := postgres.New(&d.cfg.Postgres)
	if err != nil {
		d.logger.Error("Failed to create PostgreSQL connection", logger.Error(err))
		return err
	}

	if err := db.Ping(ctx); err != nil {
		d.logger.Error("Failed to ping PostgreSQL", logger.Error(err))
		if closeErr := db.Close(); closeErr != nil {
			d.logger.Error("Failed to close database after ping failure", logger.Error(closeErr))
		}
		return err
	}

	d.db = db
	d.logger.Info("Successfully connected to PostgreSQL database")
	return nil
}

func (d *Lifecycle) Stop(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return nil
	}

	d.logger.Info("Closing database connection")

	done := make(chan error, 1)
	go func() {
		done <- d.db.Close()
	}()

	select {
	case err := <-done:
		d.db = nil
		if err != nil {
			d.logger.Error("Error closing database connection", logger.Error(err))
			return err
		}
		d.logger.Info("Database connection closed successfully")
		return nil
	case <-ctx.Done():
		d.logger.Warn("Database shutdown timeout, forcing close")
		d.db = nil
		return ctx.Err()
	}
}

func (d *Lifecycle) Connection() *postgres.DB {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.db
}
