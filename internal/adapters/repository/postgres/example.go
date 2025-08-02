package postgres

import (
	"context"
	"database/sql"
	"errors"

	"microservice/internal/adapters/database"
	"microservice/internal/core/domain/example"

	"github.com/lib/pq"
)

type Repository struct {
	db *database.Lifecycle
}

func NewRepository(db *database.Lifecycle) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetByID(ctx context.Context, id string) (*example.Entity, error) {
	query := `SELECT id, email, name FROM examples WHERE id = $1`

	var entity example.Entity
	err := r.db.Connection().QueryRowContext(ctx, query, id).Scan(
		&entity.ID,
		&entity.Email,
		&entity.Name,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, example.ErrEntityNotFound
		}
		return nil, err
	}

	return &entity, nil
}

func (r *Repository) Save(ctx context.Context, entity *example.Entity) error {
	query := `INSERT INTO examples (id, email, name) VALUES ($1, $2, $3)`

	_, err := r.db.Connection().ExecContext(ctx, query, entity.ID, entity.Email, entity.Name)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return &example.AlreadyExistsError{ID: entity.ID}
		}
		return err
	}

	return nil
}

func (r *Repository) CreateTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS examples (
			id VARCHAR(255) PRIMARY KEY,
			email VARCHAR(255) NOT NULL,
			name VARCHAR(255) NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`

	_, err := r.db.Connection().ExecContext(ctx, query)
	return err
}
